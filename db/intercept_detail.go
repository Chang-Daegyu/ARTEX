package db

// 한국어 읽기 안내
// 승인 순간에 수집한 제한된 세션 사건과 도구 ID를 audit JSON으로 저장/조회한다. 기록된 맥락이며 모델 내부 추론을 복원하지 않는다.
// ResolveIntercept는 pending일 때만 결정을 확정해 이미 내려진 사람의 결정을 타임아웃이 덮어쓰지 않게 한다.
// CompleteIntercept는 허용된 정확한 run_id/tool_use_id만 결과와 연결한다. 차단된 도구 호출을 실행 실패로 표시하지 않는다.
// audit이 없는 예전 행은 실제로 없는 기록으로 남기며 목록 응답에 긴 audit을 일괄 포함하지 않는다.

import (
	"database/sql"
	"encoding/json"
	"time"
)

// InterceptContextEntry is a bounded, recorded session event, not model reasoning.
type InterceptContextEntry struct {
	Kind      string `json:"kind"`
	Tool      string `json:"tool,omitempty"`
	ToolUseID string `json:"tool_use_id,omitempty"`
	Text      string `json:"text"`
	IsError   bool   `json:"is_error,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
}

// InterceptAudit is captured at review time. It is deliberately excluded from
// polling/list responses; old rows have no audit instead of reconstructed data.
type InterceptAudit struct {
	RunID            string                  `json:"run_id,omitempty"`
	ToolUseID        string                  `json:"tool_use_id,omitempty"`
	Correlation      string                  `json:"correlation"` // exact | ambiguous | unavailable
	InputDigest      string                  `json:"input_digest"`
	UserMessage      string                  `json:"user_message"`
	UserTruncated    bool                    `json:"user_truncated,omitempty"`
	Context          []InterceptContextEntry `json:"context"`
	ContextTruncated bool                    `json:"context_truncated,omitempty"`
	CapturedAt       time.Time               `json:"captured_at"`
	ModelFallback    bool                    `json:"model_fallback,omitempty"`
	ModelInput       json.RawMessage         `json:"model_input,omitempty"`
	ModelInputDigest string                  `json:"model_input_digest,omitempty"`
	InitialAction    string                  `json:"initial_action"`
	InitialReason    string                  `json:"initial_reason"`
	EffectiveAction  string                  `json:"effective_action,omitempty"`
	DecisionReason   string                  `json:"decision_reason,omitempty"`
	RuleName         string                  `json:"rule_name,omitempty"`
	ConfigDigest     string                  `json:"config_digest,omitempty"`
	ProfileID        int64                   `json:"profile_id,omitempty"`
	ExecutionStatus  string                  `json:"execution_status"`
	Output           string                  `json:"output,omitempty"`
	OutputTruncated  bool                    `json:"output_truncated,omitempty"`
	ExecutionEndedAt *time.Time              `json:"execution_ended_at,omitempty"`
}

type InterceptDetail struct {
	InterceptApprovalRow
	Audit *InterceptAudit `json:"audit"`
}

func (d *DB) GetInterceptDetail(id int64) (*InterceptDetail, error) {
	var out InterceptDetail
	var raw []byte
	err := d.QueryRow(approvalRowSelectWithAudit+` WHERE ip.id=$1`, id).Scan(
		&out.ID, &out.RuleID, &out.ConversationID, &out.TaskID, &out.AgentName,
		&out.ToolName, &out.ToolInput, &out.Status, &out.Reason, &out.DecidedAt, &out.CreatedAt,
		&out.DecisionSource, &out.ConvTitle, &out.ConvAgentKey, &out.RuleName, &raw,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out.Audit); err != nil {
			return nil, err
		}
	}
	return &out, nil
}

// ResolveIntercept atomically settles a pending request. A timeout cannot
// overwrite a human decision and repeat decisions cannot rewrite history.
// 한국어: pending 조건을 포함한 UPDATE로 최초 결정만 확정한다. allow여도 정확한 실행 연결이 없으면 unknown으로 기록해 실행 결과를 아는 것처럼 표시하지 않는다.
func (d *DB) ResolveIntercept(id int64, status, action, reason string) (bool, error) {
	execution := "not_executed"
	if action == "allow" {
		execution = "awaiting_result"
	}
	patch, err := json.Marshal(map[string]any{
		"effective_action": action, "decision_reason": reason, "execution_status": execution,
	})
	if err != nil {
		return false, err
	}
	r, err := d.Exec(`UPDATE intercept_pending SET status=$2, decided_at=NOW(),
		audit=CASE WHEN audit IS NULL THEN NULL ELSE audit || $3::jsonb ||
        CASE WHEN $4='allow' AND audit->>'correlation' IS DISTINCT FROM 'exact'
        THEN '{"execution_status":"unknown"}'::jsonb ELSE '{}'::jsonb END END
        WHERE id=$1 AND status='pending'`, id, status, patch, action)
	if err != nil {
		return false, err
	}
	n, err := r.RowsAffected()
	return n == 1, err
}

// CompleteIntercept only updates the exact recorded call after it was allowed.
// A blocked tool_result must never be presented as a failed execution.
// 한국어: 저장된 run_id/tool_use_id가 같고 effective_action=allow, awaiting_result일 때만 결과를 붙인다. 허용 여부와 실제 실행 성공 여부를 분리한다.
func (d *DB) CompleteIntercept(id int64, runID, toolUseID, status, output string, truncated bool) error {
	patch, err := json.Marshal(map[string]any{
		"execution_status": status, "output": output, "output_truncated": truncated,
		"execution_ended_at": time.Now().UTC(),
	})
	if err != nil {
		return err
	}
	_, err = d.Exec(`UPDATE intercept_pending SET audit=audit || $4::jsonb
		WHERE id=$1 AND audit->>'run_id'=$2 AND audit->>'tool_use_id'=$3
		AND audit->>'effective_action'='allow' AND audit->>'execution_status'='awaiting_result'`,
		id, runID, toolUseID, patch)
	return err
}
