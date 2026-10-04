package db

// 한국어 읽기 안내
// 도구 실행 시도마다 도구명·에이전트·작업·결과를 기록하는 가벼운 사용 장부다.
// 도구별 횟수 조회는 활성 기록과 아카이브 집계를 합친다. activity의 상세 입력/출력을 대체하지 않는다.
// 도구 등록/설정은 tools.go, 실제 실행은 agent와 확장 도구 런타임에서 담당한다.

// ToolUsage is one catalog tool invocation. It stores attribution dimensions only:
// tool arguments and results are deliberately excluded from the ledger.
type ToolUsage struct {
	ToolKey       string `json:"tool_key"`
	AgentKey      string `json:"agent_key"`
	TaskID        int64  `json:"task_id"`
	ExplorationID int64  `json:"exploration_id"`
	IntentID      int64  `json:"intent_id"`
	SessionID     string `json:"session_id"`
}

// InsertToolUsage appends one ledger row. Runtime callers treat metering as
// best-effort so a statistics failure never interrupts the tool itself.
func (d *DB) InsertToolUsage(u *ToolUsage) error {
	_, err := d.Exec(`
INSERT INTO tool_usage(tool_key, agent_key, task_id, exploration_id, intent_id, session_id)
VALUES ($1, NULLIF($2,''), $3, $4, $5, NULLIF($6,''))`,
		u.ToolKey, u.AgentKey, nullIfZero(u.TaskID), nullIfZero(u.ExplorationID),
		nullIfZero(u.IntentID), u.SessionID)
	return err
}

// ToolUsageCounts returns invocation totals keyed by catalog tool key. Entries
// without calls are absent; API callers merge the result into the tools catalog.
func (d *DB) ToolUsageCounts() (map[string]int, error) {
	rows, err := d.Query(`
SELECT tool_key, COUNT(*)
FROM tool_usage
GROUP BY tool_key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var key string
		var calls int
		if err := rows.Scan(&key, &calls); err != nil {
			return nil, err
		}
		out[key] = calls
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	archived, err := d.archivedTaskAggregates()
	if err != nil {
		return nil, err
	}
	for _, aggregate := range archived {
		for key, calls := range aggregate.Tools {
			out[key] += calls
		}
	}
	return out, nil
}
