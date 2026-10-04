package agent

// 한국어 파일 해설: agent/review_context_test.go
// Worker의 여러 도구 호출을 거치며 승인 심사 문맥이 올바르게 갱신되는지 검사한다.
// 현재 동작과 이미 수행한 짝지어진 도구 증거를 구별하고 상위 실행의 광범위한 목적이 심사 배경으로 잘못 이어지지 않는지 확인한다.
// 테스트 이름은 아래의 각 Test 함수가 확인하는 구체적 경계를 나타낸다.
// 한국어 문서화 변경은 기존 테스트의 입력·예상값·실행 조건을 수정하지 않는다.

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/guard"
	"github.com/Autumn-27/artex/intercept"
	"github.com/Autumn-27/norma/llm"
	"github.com/Autumn-27/norma/tool"
)

// Real PostgreSQL + SDK hooks: Worker reviews receive the current call only.
// Intent summaries, inherited background and prior execution are excluded.
func TestWorkerReviewContextAcrossToolCalls(t *testing.T) {
	dsn, _, err := db.DSN()
	if err != nil {
		t.Skip("no test database configured")
	}
	d, err := db.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	expID, err := d.CreateExploration("只操作隔离测试目录", "验证创建和清理")
	if err != nil {
		t.Fatal(err)
	}
	ts := d.Exploration(expID)
	taskID := fmt.Sprint(expID)
	t.Cleanup(func() {
		_, _ = d.Exec(`DELETE FROM intercept_pending WHERE task_id=$1`, taskID)
		_, _ = d.Exec(`DELETE FROM explorations WHERE id=$1`, expID)
	})
	ic := intercept.New(d)
	priorTools, err := ic.GetEnabledTools()
	if err != nil {
		t.Fatal(err)
	}
	priorConfig := ic.GetJudgeConfig()
	t.Cleanup(func() { _ = ic.SetEnabledTools(priorTools); _ = ic.SetJudgeConfig(priorConfig) })
	const probeName = "ContextEvidenceProbe"
	if err := ic.SetEnabledTools([]string{probeName}); err != nil {
		t.Fatal(err)
	}
	if err := ic.SetJudgeConfig(intercept.JudgeConfig{Enabled: true}); err != nil {
		t.Fatal(err)
	}
	var inputs []intercept.ReviewInput
	ic.SetReviewer(func(_ context.Context, _ int64, _ string, in intercept.ReviewInput) (intercept.Decision, error) {
		inputs = append(inputs, in)
		action := "allow"
		if string(in.Arguments) == `{"step":2}` {
			action = "deny"
		}
		return intercept.Decision{Action: action, Message: "probe policy"}, nil
	})
	turn, executions := 0, 0
	provider := captureUsageProvider{stream: func(_ context.Context, yield func(llm.StreamEvent, error) bool) {
		turn++
		events := []llm.StreamEvent{{Type: llm.SETextDelta, Text: "done"}, {Type: llm.SEMessageDelta, StopReason: "end_turn"}}
		if turn <= 2 {
			events = []llm.StreamEvent{
				{Type: llm.SEToolUseStart, ToolID: fmt.Sprintf("call-%d", turn), ToolName: probeName},
				{Type: llm.SEToolInputJSON, Text: fmt.Sprintf(`{"step":%d}`, turn)},
				{Type: llm.SEMessageDelta, StopReason: "tool_use"},
			}
		}
		for _, event := range events {
			if !yield(event, nil) {
				return
			}
		}
	}}
	probe := tool.Build(tool.Spec{Name: probeName, Schema: map[string]any{"type": "object"},
		Run: func(context.Context, json.RawMessage, *tool.ToolContext) (tool.Result, error) {
			executions++
			_, err := ts.AddConstraint("deny", "禁止后续清理", "human")
			return tool.Text("Created a new fixture; no existing file overwritten."), err
		},
	})
	workDir := t.TempDir()
	ctx := intercept.WithTaskContext(t.Context(), taskID, "test-agent", nil)
	ctx = intercept.WithReviewContext(ctx, "/parent", intercept.ReviewBackground{Source: intercept.BackgroundUserMessage, Text: "PARENT_BACKGROUND_SENTINEL"})
	intentPayload := map[string]any{"summary": "创建并清理", "extra": "FULL_INTENT_SENTINEL"}
	intentID, err := ts.AddNode("intent", intentPayload, 0, "running", "planner", nil)
	if err != nil {
		t.Fatal(err)
	}
	rawIntent, _ := json.Marshal(intentPayload)
	worker := NewWorker(provider, "test-model", workDir, nil, 0, 3, probe)
	_, _, err = worker.Execute(ctx, "test-agent", expID, nil, ts, &db.Node{ID: intentID, Payload: rawIntent}, guard.NewWithInterceptor(ic).Hooks(), nil, nil, nil)
	runDir := ensureRunDir(workDir, expID, intentID)
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) != 2 || executions != 1 {
		t.Fatalf("reviews=%d executions=%d", len(inputs), executions)
	}
	first, second := inputs[0], inputs[1]
	for _, in := range inputs {
		if in.Version != 4 || in.Background != nil || in.WorkingDir != runDir {
			t.Fatalf("unexpected Worker background: %+v", in)
		}
		raw, _ := json.Marshal(in)
		for _, forbidden := range []string{"创建并清理", "PARENT_BACKGROUND_SENTINEL", `"background"`, "只操作隔离测试目录", "验证创建和清理", "禁止后续清理", "FULL_INTENT_SENTINEL", "全局探索态势", `"task_id"`, `"task"`, `"turn_input"`, `"worker_intent"`, `"history"`, `"history_truncated"`, `"correlation"`, "Created a new fixture"} {
			if strings.Contains(string(raw), forbidden) {
				t.Fatalf("unexpected review data: %s", forbidden)
			}
		}
	}
	constraints, err := ts.ListConstraints()
	if err != nil || len(constraints) != 1 {
		t.Fatal("Agent task constraints were unexpectedly changed")
	}
	if string(first.Arguments) != `{"step":1}` || string(second.Arguments) != `{"step":2}` {
		t.Fatal("review lost current parameters")
	}

	rows, err := d.ListTaskIntercepts(taskID)
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows=%d err=%v", len(rows), err)
	}
	for _, row := range rows {
		detail, err := d.GetInterceptDetail(row.ID)
		if err != nil || detail == nil || detail.Audit == nil || len(detail.Audit.ModelInput) == 0 {
			t.Fatalf("verdict lost model input: %+v err=%v", detail, err)
		}
		var saved intercept.ReviewInput
		if json.Unmarshal(detail.Audit.ModelInput, &saved) != nil || saved.Background != nil || saved.Version != 4 {
			t.Fatal("stored Worker review input retained a background")
		}
		if row.Status == "allowed" && (detail.Audit.ExecutionStatus != "succeeded" || saved.Version != 4) {
			t.Fatal("automatic allow lost execution result or its original background snapshot")
		}
		if detail.Audit.Correlation != "exact" {
			t.Fatal("audit lost call correlation")
		}
		if row.Status == "denied" {
			found := false
			for _, entry := range detail.Audit.Context {
				if entry.Kind == "tool_result" && entry.ToolUseID == "call-1" && strings.Contains(entry.Text, "Created a new fixture") {
					found = true
				}
			}
			if !found {
				t.Fatal("prior execution missing from separate audit")
			}
		}
		if row.Status == "denied" && detail.Audit.ExecutionStatus != "not_executed" {
			t.Fatal("denial recorded an execution")
		}
	}

}
