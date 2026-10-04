package db

// 한국어 읽기 안내
// 승인 이력에서 원래 도구 실행 위치로 이동할 수 있도록 activity/conversation_activities의 정확한 호출을 찾는다.
// audit의 correlation=exact와 tool_use_id가 필요하며 명령 텍스트 유사도로 실행을 추측하지 않는다.
// 후보 3개까지만 읽어 중복 ID를 감지하고, call/result의 worker·intent·main 세그먼트가 같은지도 검증한다.
// 삭제/아카이브된 작업, 사라진 세션, 모호한 연결은 각각 오류로 반환한다.

import (
	"errors"
	"fmt"
	"strconv"
)

var ErrInterceptTaskDeleted = errors.New("任务已被删除或归档")
var ErrInterceptSessionDeleted = errors.New("对应会话或执行记录已被删除或不存在")

var ErrInterceptExecutionUnavailable = errors.New("未找到可唯一关联的原始工具调用；记录可能已删除，或旧审批没有保存关联 ID")

// InterceptExecution is a navigation target read from original activity rows.
// It is not model context and never falls back to matching command text.
type InterceptExecution struct {
	ConversationID *int64     `json:"conversation_id,omitempty"`
	TaskID         *string    `json:"task_id,omitempty"`
	Session        string     `json:"session"`
	Seq            int64      `json:"seq"`
	Items          []Activity `json:"-"`
}

// 한국어: 승인 audit의 정확한 tool_use_id로 원래 실행 쌍을 찾는다. 3개 이상 후보, 다른 worker/intent/세그먼트, 없는 작업은 모호함/삭제 오류로 반환하고 문자열 유사도로 보완하지 않는다.
func (d *DB) GetInterceptExecution(id int64) (*InterceptExecution, error) {
	approval, err := d.GetInterceptDetail(id)
	if err != nil || approval == nil {
		return nil, err
	}
	audit := approval.Audit
	if audit == nil || audit.Correlation != "exact" || audit.ToolUseID == "" {
		return nil, ErrInterceptExecutionUnavailable
	}
	var query string
	var scope any
	if approval.ConversationID != nil {
		scope = *approval.ConversationID
		query = `SELECT id, NULL::bigint, COALESCE(worker,''), kind, COALESCE(tool,''), tool_use_id, is_error, COALESCE(summary,''), created_at, NULL::integer FROM conversation_activities WHERE conversation_id=$1`
	} else if approval.TaskID != nil {
		taskID, parseErr := strconv.ParseInt(*approval.TaskID, 10, 64)
		if parseErr != nil || taskID <= 0 {
			return nil, ErrInterceptExecutionUnavailable
		}
		var exists bool
		if err := d.QueryRow(`SELECT EXISTS(SELECT 1 FROM tasks WHERE id=$1 AND archived_at IS NULL AND deleted_at IS NULL)`, taskID).Scan(&exists); err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrInterceptTaskDeleted
		}
		scope = taskID
		query = `SELECT id, node_id, COALESCE(worker,''), kind, COALESCE(tool,''), tool_use_id, is_error, COALESCE(summary,''), created_at, main_seg FROM activity WHERE exploration_id=(SELECT exploration_id FROM tasks WHERE id=$1 AND archived_at IS NULL AND deleted_at IS NULL)`
	} else {
		return nil, ErrInterceptExecutionUnavailable
	}
	// Three matches suffice to detect duplicate IDs without loading a transcript.
	rows, err := d.Query(query+` AND tool_use_id=$2 AND kind IN ('tool_use','tool_result') ORDER BY id LIMIT 3`, scope, audit.ToolUseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := &InterceptExecution{ConversationID: approval.ConversationID, TaskID: approval.TaskID}
	for rows.Next() {
		var a Activity
		if err := rows.Scan(&a.ID, &a.NodeID, &a.Worker, &a.Kind, &a.Tool, &a.ToolUseID, &a.IsError, &a.Summary, &a.CreatedAt, &a.MainSeg); err != nil {
			return nil, err
		}
		out.Items = append(out.Items, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out.Items) == 0 {
		return nil, ErrInterceptSessionDeleted
	}
	if len(out.Items) > 2 {
		return nil, ErrInterceptExecutionUnavailable
	}
	call := out.Items[0]
	if call.Kind != "tool_use" || call.Tool != approval.ToolName {
		return nil, ErrInterceptExecutionUnavailable
	}
	if len(out.Items) == 2 {
		result := out.Items[1]
		if result.Kind != "tool_result" || result.Worker != call.Worker || (result.Tool != "" && result.Tool != call.Tool) || !sameOptionalInt64(call.NodeID, result.NodeID) || !sameMainSegment(call.MainSeg, result.MainSeg) {
			return nil, ErrInterceptExecutionUnavailable
		}
	}
	out.Seq = call.ID
	if approval.ConversationID == nil {
		switch {
		case call.Worker == "mainagent":
			seg := 0
			if call.MainSeg != nil {
				seg = *call.MainSeg
			}
			out.Session = fmt.Sprintf("main:%d", seg)
		case call.Worker == "planner":
			out.Session = "plan"
		case call.NodeID != nil:
			out.Session = fmt.Sprintf("intent:%d", *call.NodeID)
		default:
			return nil, ErrInterceptExecutionUnavailable
		}
	}
	return out, nil
}

func sameOptionalInt64(a, b *int64) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

// 한국어: 옛 기록의 nil 세그먼트를 원래 세그먼트 0과 같은 값으로 비교한다. 저장 형식 차이 때문에 같은 main 대화의 호출/결과가 분리되지 않게 한다.
func sameMainSegment(a, b *int) bool {
	av, bv := 0, 0
	if a != nil {
		av = *a
	}
	if b != nil {
		bv = *b
	}
	return av == bv
}
