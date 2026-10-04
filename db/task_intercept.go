package db

// 한국어 읽기 안내
// 한 작업에 한정된 자산 allow/block 규칙의 CRUD와 생성 시 일괄 등록을 담당한다.
// 작업 ID와 규칙 ID를 함께 조건에 넣어 다른 작업의 규칙을 수정하지 않도록 범위를 고정한다.
// TaskInterceptRulesSplit은 저장된 action을 차단/허용 목록으로 분리한다. 실제 우선순위와 후보 비교는 EvaluateAssetGate를 읽는다.

import (
	"database/sql"
	"fmt"
)

// TaskInterceptRuleInput is one task-level rule supplied at task creation.
// Action: 'block'=拦截 'allow'=允许(白名单)；空视为 'block'。
type TaskInterceptRuleInput struct {
	Enabled bool   `json:"enabled"`
	Action  string `json:"action"`
	Kind    string `json:"kind"`
	Pattern string `json:"pattern"`
	Note    string `json:"note"`
}

const taskInterceptRuleCols = `id, enabled, action, kind, pattern, note, created_at, updated_at`

func scanTaskInterceptRule(row interface{ Scan(...any) error }) (AssetInterceptRule, error) {
	var r AssetInterceptRule
	err := row.Scan(&r.ID, &r.Enabled, &r.Action, &r.Kind, &r.Pattern, &r.Note, &r.CreatedAt, &r.UpdatedAt)
	return r, err
}

// ListTaskInterceptRules returns a task's rules (both block and allow) as
// AssetInterceptRule (Builtin always false; Action carries block/allow). taskID
// <= 0 returns nothing.
func (s *AssetStore) ListTaskInterceptRules(taskID int64) ([]AssetInterceptRule, error) {
	if taskID <= 0 {
		return nil, nil
	}
	rows, err := s.db.Query(`SELECT `+taskInterceptRuleCols+` FROM task_intercept_rules WHERE task_id=$1 ORDER BY action, id`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AssetInterceptRule
	for rows.Next() {
		r, err := scanTaskInterceptRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// TaskInterceptRulesSplit loads a task's rules and splits them into block and
// allow sets, for the enforcement gate.
func (s *AssetStore) TaskInterceptRulesSplit(taskID int64) (block, allow []AssetInterceptRule, err error) {
	rules, err := s.ListTaskInterceptRules(taskID)
	if err != nil {
		return nil, nil, err
	}
	for _, r := range rules {
		if r.Action == "allow" {
			allow = append(allow, r)
		} else {
			block = append(block, r)
		}
	}
	return block, allow, nil
}

func normalizeRuleAction(action string) string {
	if action == "allow" {
		return "allow"
	}
	return "block"
}

// CreateTaskInterceptRule inserts a rule under a task.
func (s *AssetStore) CreateTaskInterceptRule(taskID int64, action, kind, pattern, note string, enabled bool) (AssetInterceptRule, error) {
	row := s.db.QueryRow(`
INSERT INTO task_intercept_rules(task_id, enabled, action, kind, pattern, note)
VALUES ($1,$2,$3,$4,$5,$6)
RETURNING `+taskInterceptRuleCols,
		taskID, enabled, normalizeRuleAction(action), kind, pattern, note)
	return scanTaskInterceptRule(row)
}

// UpdateTaskInterceptRule replaces the editable fields of a task's rule (scoped
// by task_id so a rule can only be edited through its owning task).
func (s *AssetStore) UpdateTaskInterceptRule(taskID, ruleID int64, action, kind, pattern, note string, enabled bool) (AssetInterceptRule, error) {
	row := s.db.QueryRow(`
UPDATE task_intercept_rules
   SET enabled=$3, action=$4, kind=$5, pattern=$6, note=$7
WHERE id=$1 AND task_id=$2
RETURNING `+taskInterceptRuleCols,
		ruleID, taskID, enabled, normalizeRuleAction(action), kind, pattern, note)
	return scanTaskInterceptRule(row)
}

// DeleteTaskInterceptRule removes a task's rule. Returns false if not found.
func (s *AssetStore) DeleteTaskInterceptRule(taskID, ruleID int64) (bool, error) {
	res, err := s.db.Exec(`DELETE FROM task_intercept_rules WHERE id=$1 AND task_id=$2`, ruleID, taskID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// ToggleTaskInterceptRule flips the enabled state of a task's rule.
func (s *AssetStore) ToggleTaskInterceptRule(taskID, ruleID int64, enabled bool) error {
	_, err := s.db.Exec(`UPDATE task_intercept_rules SET enabled=$3 WHERE id=$1 AND task_id=$2`, ruleID, taskID, enabled)
	return err
}

// insertTaskInterceptRules inserts task-level rules within the task-creation
// transaction (mirrors insertTaskCompanies).
func insertTaskInterceptRules(tx *sql.Tx, taskID int64, rules []TaskInterceptRuleInput) error {
	for _, r := range rules {
		if _, err := tx.Exec(`
INSERT INTO task_intercept_rules(task_id, enabled, action, kind, pattern, note)
VALUES ($1,$2,$3,$4,$5,$6)`, taskID, r.Enabled, normalizeRuleAction(r.Action), r.Kind, r.Pattern, r.Note); err != nil {
			return fmt.Errorf("insert task intercept rule %q: %w", r.Pattern, err)
		}
	}
	return nil
}
