package db

// 한국어 읽기 안내
// 작업의 직접 source 관계·회사 범위·순서 있는 LLM 체인과 현재 프로필 커서를 읽고 갱신한다.
// 목록 hydrate는 작업별 3회 추가 조회 대신 세 종류의 일괄 쿼리로 N+1 문제를 피한다. 단일 작업의 LLM 커서와 체인은 한 SQL 스냅샷에서 읽는다.
// 체인 변경은 revision을 증가시키며 오래된 모델 호출이 보고한 실패가 새 체인을 잘못 넘기지 않게 한다. task 행을 먼저 잠그는 순서가 중요하다.
// 특정 blocked_reason으로 막힌 intent를 다시 여는 보조 함수도 있다. 모든 blocked 상태를 무조건 재시도하는 기능은 아니다.

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const IntentBlockedLLMQuota = "llm_quota_exhausted"

// TaskLLMProfile is one ordered entry in a task's explicit failover chain.
type TaskLLMProfile struct {
	ProfileID   int64      `json:"profile_id"`
	Position    int        `json:"position"`
	Status      string     `json:"status"`
	LastError   string     `json:"last_error,omitempty"`
	ExhaustedAt *time.Time `json:"exhausted_at,omitempty"`
}

// TaskSource identifies one directly related task and its exploration.
type TaskSource struct {
	TaskID        int64
	ExplorationID int64
	Description   string
	Goal          string
	Status        string
}

// TaskLLMTransition reports the shared task-level result of marking one profile
// quota-exhausted. NextProfileID is nil when the explicit chain is exhausted.
type TaskLLMTransition struct {
	PreviousProfileID int64
	NextProfileID     *int64
	ChainExhausted    bool
	Advanced          bool
	// Stale means the chain was replaced after the failing call selected its
	// provider. The caller may retry a pre-stream request against the new chain,
	// but must not report or persist a transition for this result.
	Stale bool
}

func (d *DB) hydrateTaskContext(t *Task) error {
	if t == nil {
		return nil
	}
	// A nil cursor on a non-empty chain is a persisted end-of-chain marker. Do not
	// repair it from an earlier ready entry: the user may have manually selected a
	// profile in the middle and legitimately exhausted every candidate after it.
	legacyProfileID, activeProfileID, revision, chain, err := d.taskLLMContext(t.ID)
	if err != nil {
		return err
	}
	sources, err := d.TaskSourceIDs(t.ID)
	if err != nil {
		return err
	}
	t.SourceTaskIDs = sources
	companies, err := d.TaskCompanyIDs(t.ID)
	if err != nil {
		return err
	}
	t.CompanyIDs = companies
	applyTaskLLMContext(t, legacyProfileID, activeProfileID, revision, chain)
	return nil
}

func applyTaskLLMContext(
	t *Task,
	legacyProfileID, activeProfileID *int64,
	revision int64,
	chain []TaskLLMProfile,
) {
	t.LLMProfileID = legacyProfileID
	t.ActiveLLMProfileID = activeProfileID
	t.LLMChainRevision = revision
	t.LLMProfileIDs = make([]int64, 0, len(chain))
	t.LLMFailoverState = "default"
	t.LLMFailoverReason = ""
	var latest *TaskLLMProfile
	activeReady := false
	for i := range chain {
		entry := chain[i]
		t.LLMProfileIDs = append(t.LLMProfileIDs, entry.ProfileID)
		if t.ActiveLLMProfileID != nil && entry.ProfileID == *t.ActiveLLMProfileID && entry.Status == "ready" {
			activeReady = true
		}
		if entry.ExhaustedAt != nil && (latest == nil || entry.ExhaustedAt.After(*latest.ExhaustedAt)) {
			copy := entry
			latest = &copy
		}
	}
	if len(chain) > 0 {
		if activeReady {
			t.LLMFailoverState = "ready"
		} else {
			t.LLMFailoverState = "chain_exhausted"
		}
	}
	if latest != nil {
		t.LLMFailoverReason = latest.LastError
	}
}

type taskBatchContext struct {
	legacyProfileID *int64
	activeProfileID *int64
	revision        int64
	chain           []TaskLLMProfile
	sourceIDs       []int64
	companyIDs      []int64
}

// hydrateTasksContext loads every task's LLM chain, source tasks, and company
// scopes with three bulk queries. ListTasks used to issue these queries once per
// task, making startup and task-list hydration grow as 3N+1 database round trips.
// 한국어: 작업 목록 전체의 모델 체인, source, 회사 범위를 세 번의 일괄 조회로 보완한다. 작업 수가 늘어도 추가 왕복 수가 3N으로 증가하지 않는다.
func (d *DB) hydrateTasksContext(tasks []*Task) error {
	if len(tasks) == 0 {
		return nil
	}

	ids := make([]int64, 0, len(tasks))
	contexts := make(map[int64]*taskBatchContext, len(tasks))
	byID := make(map[int64]*Task, len(tasks))
	for _, task := range tasks {
		if task == nil {
			continue
		}
		ids = append(ids, task.ID)
		contexts[task.ID] = &taskBatchContext{}
		byID[task.ID] = task
	}
	if len(ids) == 0 {
		return nil
	}

	rows, err := d.Query(`
SELECT t.id, t.llm_profile_id, t.active_llm_profile_id, t.llm_chain_revision,
       p.profile_id, p.position, p.status, COALESCE(p.last_error,''), p.exhausted_at
FROM tasks t
LEFT JOIN task_llm_profiles p ON p.task_id=t.id
WHERE t.id=ANY($1::bigint[])
ORDER BY t.id, p.position NULLS LAST`, ids)
	if err != nil {
		return err
	}
	for rows.Next() {
		var taskID, revision int64
		var legacy, active, profileID, position sql.NullInt64
		var status, lastError sql.NullString
		var exhaustedAt sql.NullTime
		if err := rows.Scan(
			&taskID, &legacy, &active, &revision,
			&profileID, &position, &status, &lastError, &exhaustedAt,
		); err != nil {
			rows.Close()
			return err
		}
		context := contexts[taskID]
		if context == nil {
			continue
		}
		context.revision = revision
		if legacy.Valid {
			id := legacy.Int64
			context.legacyProfileID = &id
		}
		if active.Valid {
			id := active.Int64
			context.activeProfileID = &id
		}
		if profileID.Valid {
			entry := TaskLLMProfile{
				ProfileID: profileID.Int64,
				Position:  int(position.Int64),
				Status:    status.String,
				LastError: lastError.String,
			}
			if exhaustedAt.Valid {
				ts := exhaustedAt.Time
				entry.ExhaustedAt = &ts
			}
			context.chain = append(context.chain, entry)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	rows, err = d.Query(`
SELECT task_id, source_task_id
FROM task_relations
WHERE task_id=ANY($1::bigint[])
ORDER BY task_id, created_at, source_task_id`, ids)
	if err != nil {
		return err
	}
	for rows.Next() {
		var taskID, sourceID int64
		if err := rows.Scan(&taskID, &sourceID); err != nil {
			rows.Close()
			return err
		}
		if context := contexts[taskID]; context != nil {
			context.sourceIDs = append(context.sourceIDs, sourceID)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	rows, err = d.Query(`
SELECT task_id, company_id
FROM task_scope
WHERE task_id=ANY($1::bigint[]) AND kind='company' AND company_id IS NOT NULL
ORDER BY task_id, id`, ids)
	if err != nil {
		return err
	}
	for rows.Next() {
		var taskID, companyID int64
		if err := rows.Scan(&taskID, &companyID); err != nil {
			rows.Close()
			return err
		}
		if context := contexts[taskID]; context != nil {
			context.companyIDs = append(context.companyIDs, companyID)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	for id, context := range contexts {
		task := byID[id]
		task.SourceTaskIDs = context.sourceIDs
		task.CompanyIDs = context.companyIDs
		applyTaskLLMContext(
			task,
			context.legacyProfileID,
			context.activeProfileID,
			context.revision,
			context.chain,
		)
	}
	return nil
}

// taskLLMContext reads the compatibility profile, current cursor, and ordered
// chain in one statement. Runtime chain edits commit atomically, and one SQL
// statement gives hydration one matching snapshot instead of a transient mix of
// an old task cursor and a newly replaced chain (or vice versa).
// 한국어: 프로필 호환 필드·활성 커서·체인·revision을 한 SQL 문장에서 읽는다. 체인 편집 직전/직후를 서로 다른 쿼리로 섞어 읽는 중간 상태를 피한다.
func (d *DB) taskLLMContext(taskID int64) (*int64, *int64, int64, []TaskLLMProfile, error) {
	rows, err := d.Query(`
SELECT t.llm_profile_id, t.active_llm_profile_id, t.llm_chain_revision,
       p.profile_id, p.position, p.status, COALESCE(p.last_error,''), p.exhausted_at
FROM tasks t
LEFT JOIN task_llm_profiles p ON p.task_id=t.id
WHERE t.id=$1
ORDER BY p.position NULLS LAST`, taskID)
	if err != nil {
		return nil, nil, 0, nil, err
	}
	defer rows.Close()
	var (
		legacyProfileID *int64
		activeProfileID *int64
		revision        int64
		chain           []TaskLLMProfile
		found           bool
	)
	for rows.Next() {
		var legacy, active, profileID, position sql.NullInt64
		var rowRevision int64
		var status, lastError sql.NullString
		var exhaustedAt sql.NullTime
		if err := rows.Scan(&legacy, &active, &rowRevision, &profileID, &position, &status, &lastError, &exhaustedAt); err != nil {
			return nil, nil, 0, nil, err
		}
		if !found {
			found = true
			revision = rowRevision
			if legacy.Valid {
				id := legacy.Int64
				legacyProfileID = &id
			}
			if active.Valid {
				id := active.Int64
				activeProfileID = &id
			}
		}
		if !profileID.Valid {
			continue
		}
		entry := TaskLLMProfile{
			ProfileID: profileID.Int64,
			Position:  int(position.Int64),
			Status:    status.String,
			LastError: lastError.String,
		}
		if exhaustedAt.Valid {
			ts := exhaustedAt.Time
			entry.ExhaustedAt = &ts
		}
		chain = append(chain, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, 0, nil, err
	}
	if !found {
		return nil, nil, 0, nil, sql.ErrNoRows
	}
	return legacyProfileID, activeProfileID, revision, chain, nil
}

func (d *DB) TaskSourceIDs(taskID int64) ([]int64, error) {
	rows, err := d.Query(`SELECT source_task_id FROM task_relations WHERE task_id=$1 ORDER BY created_at, source_task_id`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// TaskCompanyIDs returns the companies whose asset scopes are available to the
// task. The task_scope rows remain the single source of truth.
func (d *DB) TaskCompanyIDs(taskID int64) ([]int64, error) {
	rows, err := d.Query(`
SELECT company_id FROM task_scope
WHERE task_id=$1 AND kind='company' AND company_id IS NOT NULL
ORDER BY id`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (d *DB) TaskSources(taskID int64) ([]TaskSource, error) {
	rows, err := d.Query(`
SELECT t.id, t.exploration_id, t.description, t.goal, t.status
FROM task_relations r
JOIN tasks t ON t.id=r.source_task_id AND t.deleted_at IS NULL
WHERE r.task_id=$1
ORDER BY r.created_at, r.source_task_id`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TaskSource
	for rows.Next() {
		var source TaskSource
		if err := rows.Scan(&source.TaskID, &source.ExplorationID, &source.Description, &source.Goal, &source.Status); err != nil {
			return nil, err
		}
		out = append(out, source)
	}
	return out, rows.Err()
}

func (d *DB) TaskLLMProfiles(taskID int64) ([]TaskLLMProfile, error) {
	rows, err := d.Query(`SELECT profile_id, position, status, COALESCE(last_error,''), exhausted_at
FROM task_llm_profiles WHERE task_id=$1 ORDER BY position`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TaskLLMProfile
	for rows.Next() {
		var entry TaskLLMProfile
		if err := rows.Scan(&entry.ProfileID, &entry.Position, &entry.Status, &entry.LastError, &entry.ExhaustedAt); err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return out, rows.Err()
}

// ReplaceTaskLLMProfiles atomically replaces and resets the explicit task chain.
// activeProfileID=0 selects the first entry. An empty list restores the existing
// agent-binding/global fallback behavior.
// 终态(done/failed/timeout)任务同样允许改链:任务结束后主 Agent 对话仍会走这条链,
// 链上模型不可用时必须能换,否则已完成任务就再也没法交互了。
// 한국어: 작업을 잠그고 체인 전체와 활성 커서를 함께 바꾼다. 빈 체인은 에이전트/전역 모델 선택으로 돌아가며 종료된 작업의 후속 대화도 모델 변경을 허용한다.
func (d *DB) ReplaceTaskLLMProfiles(taskID int64, profileIDs []int64, activeProfileID int64) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Keep the task -> profile lock order shared by all task LLM mutations and
	// DeleteProfile. Inserts below may take KEY SHARE locks on llm_profiles for
	// their foreign keys, so the task row must be locked before any of them.
	var lockedID int64
	if err := tx.QueryRow(`SELECT id FROM tasks WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, taskID).Scan(&lockedID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("task %d not found", taskID)
		}
		return err
	}
	if _, err := tx.Exec(`DELETE FROM task_llm_profiles WHERE task_id=$1`, taskID); err != nil {
		return err
	}
	if err := insertTaskLLMProfiles(tx, taskID, profileIDs); err != nil {
		return err
	}
	if len(profileIDs) == 0 {
		activeProfileID = 0
	} else if activeProfileID == 0 {
		activeProfileID = profileIDs[0]
	}
	if activeProfileID > 0 {
		found := false
		for _, id := range profileIDs {
			found = found || id == activeProfileID
		}
		if !found {
			return fmt.Errorf("active LLM profile %d is not in the task chain", activeProfileID)
		}
	}
	var active any
	if activeProfileID > 0 {
		active = activeProfileID
	}
	if _, err := tx.Exec(`UPDATE tasks
SET active_llm_profile_id=$2, llm_profile_id=$2, llm_chain_revision=llm_chain_revision+1
WHERE id=$1`, taskID, active); err != nil {
		return err
	}
	return tx.Commit()
}

// MarkTaskLLMProfileQuotaExhausted advances the shared task cursor once. A late
// in-flight error from an older profile records that entry as exhausted but does
// not advance past the profile another call already selected.
func (d *DB) MarkTaskLLMProfileQuotaExhausted(taskID, profileID int64, reason string) (TaskLLMTransition, error) {
	return d.markTaskLLMProfileQuotaExhausted(taskID, profileID, 0, false, reason)
}

// MarkTaskLLMProfileQuotaExhaustedAtRevision applies a provider failure only
// when it belongs to the chain snapshot used to start that call.
// 한국어: 실패한 호출이 시작될 때의 체인 revision을 넘겨, 사용자가 이미 바꾼 새 체인에 옛 공급자 오류를 적용하지 않게 한다.
func (d *DB) MarkTaskLLMProfileQuotaExhaustedAtRevision(taskID, profileID, revision int64, reason string) (TaskLLMTransition, error) {
	return d.markTaskLLMProfileQuotaExhausted(taskID, profileID, revision, true, reason)
}

// 한국어: 프로필 소진 표시와 커서 이동을 함께 처리한다. 다른 호출이 이미 다음 프로필로 옮겼다면 오래된 실패 때문에 커서를 한 번 더 넘기지 않는다.
func (d *DB) markTaskLLMProfileQuotaExhausted(taskID, profileID, revision int64, checkRevision bool, reason string) (TaskLLMTransition, error) {
	var out TaskLLMTransition
	tx, err := d.Begin()
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	var (
		active          sql.NullInt64
		currentRevision int64
	)
	if err := tx.QueryRow(`SELECT active_llm_profile_id, llm_chain_revision FROM tasks WHERE id=$1 FOR UPDATE`, taskID).Scan(&active, &currentRevision); err != nil {
		return out, err
	}
	out.PreviousProfileID = profileID
	if checkRevision && revision != currentRevision {
		out.Stale = true
		return out, tx.Commit()
	}
	var (
		position    int
		entryStatus string
	)
	if err := tx.QueryRow(`SELECT position, status FROM task_llm_profiles WHERE task_id=$1 AND profile_id=$2`, taskID, profileID).
		Scan(&position, &entryStatus); err != nil {
		if err == sql.ErrNoRows {
			// The chain was edited while this request was in flight. Its provider
			// error remains valid for the caller, but it must not mutate the new chain.
			return out, tx.Commit()
		}
		return out, err
	}
	// Once the cursor reached the end, late failures must be idempotent. This also
	// prevents an older in-flight request from reviving a ready entry before a
	// manually selected starting position.
	if !active.Valid {
		out.ChainExhausted = true
		return out, tx.Commit()
	}
	reason = truncateUTF8(strings.TrimSpace(reason), 1000)
	if entryStatus != "quota_exhausted" {
		if _, err := tx.Exec(`UPDATE task_llm_profiles
SET status='quota_exhausted', last_error=$3, exhausted_at=now()
WHERE task_id=$1 AND profile_id=$2`, taskID, profileID, reason); err != nil {
			return out, err
		}
	}
	if active.Valid && active.Int64 != profileID {
		next := active.Int64
		out.NextProfileID = &next
		return out, tx.Commit()
	}
	var next int64
	err = tx.QueryRow(`SELECT profile_id FROM task_llm_profiles
WHERE task_id=$1 AND position>$2 AND status='ready'
ORDER BY position LIMIT 1`, taskID, position).Scan(&next)
	switch err {
	case nil:
		out.Advanced = true
		out.NextProfileID = &next
		if _, err := tx.Exec(`UPDATE tasks
				SET active_llm_profile_id=$2, llm_profile_id=$2, llm_chain_revision=llm_chain_revision+1
				WHERE id=$1`, taskID, next); err != nil {
			return out, err
		}
	case sql.ErrNoRows:
		out.Advanced = true
		out.ChainExhausted = true
		if _, err := tx.Exec(`UPDATE tasks
				SET active_llm_profile_id=NULL, llm_profile_id=NULL, llm_chain_revision=llm_chain_revision+1
				WHERE id=$1`, taskID); err != nil {
			return out, err
		}
	default:
		return out, err
	}
	return out, tx.Commit()
}

// 한국어: 바이트 제한을 지키되 UTF-8 문자 시작 경계에서 자른다. 잘못된 입력 바이트도 먼저 복구해 PostgreSQL text에 저장 가능한 값을 만든다.
func truncateUTF8(value string, maxBytes int) string {
	value = strings.ToValidUTF8(value, "\uFFFD")
	if maxBytes <= 0 {
		return ""
	}
	if len(value) <= maxBytes {
		return value
	}
	end := maxBytes
	for end > 0 && !utf8.ValidString(value[:end]) {
		end--
	}
	return value[:end]
}

func (s *ExplorationStore) SetIntentBlockedReason(id int64, reason string) error {
	_, err := s.db.Exec(`UPDATE exploration_nodes
SET state='blocked', blocked_reason=NULLIF($1,''), completed_at=now()
WHERE id=$2 AND exploration_id=$3 AND kind='intent'`, reason, id, s.expID)
	return err
}

// 한국어: 특정 사유로 막힌 intent만 다시 open으로 바꾼다. 예를 들어 모델 예산 복구가 다른 이유의 차단까지 무조건 해제하지 않도록 사유를 조건에 포함한다.
func (s *ExplorationStore) ReopenIntentsByBlockedReason(reason string) (int64, error) {
	res, err := s.db.Exec(`UPDATE exploration_nodes
SET state='open', blocked_reason=NULL, completed_at=NULL
WHERE exploration_id=$1 AND kind='intent' AND state='blocked' AND blocked_reason=$2`, s.expID, reason)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}
