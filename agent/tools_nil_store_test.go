package agent

// 한국어 파일 해설: agent/tools_nil_store_test.go
// 작업 저장소가 없는 역할에 도메인 도구가 바인딩되어도 프로세스가 죽지 않는지 검사한다.
// task 필요 오류를 반환하고 예외적인 handler panic은 guardPanic이 도구 오류로 바꿔야 한다.
// 테스트 이름은 아래의 각 Test 함수가 확인하는 구체적 경계를 나타낸다.
// 한국어 문서화 변경은 기존 테스트의 입력·예상값·실행 조건을 수정하지 않는다.

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	actool "github.com/Autumn-27/norma/tool"
)

// The server-level ToolSet behind buildDomainReg carries a nil ExplorationStore,
// and the tools table can bind any of its tools to any agent — including agents
// that never run inside a task. Every domain tool must therefore survive being
// called with nil stores: a nil deref here runs on the harness's own goroutine,
// out of reach of every recover() in the server, and kills the whole process.
func TestDomainToolsSurviveNilStores(t *testing.T) {
	inputs := []string{
		`{}`,
		`{"id":379,"asset_id":1,"goal_id":1,"evidence_id":1,"intent_id":1,"node_id":1,"work_id":1,` +
			`"summary":"x","reason":"x","name":"x","severity":"low","vulnclass":"x","text":"x","q":"x"}`,
	}
	for _, tool := range NewToolSet(nil, "").AllDomainTools() {
		for _, in := range inputs {
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("%s panicked with nil stores on %s: %v", tool.Name(), in, r)
					}
				}()
				if _, err := tool.Call(context.Background(), json.RawMessage(in), nil); err != nil {
					t.Fatalf("%s returned a transport error: %v", tool.Name(), err)
				}
			}()
		}
	}
}

// node_detail is the one that took the process down; assert it now answers with a
// usable refusal rather than dying.
func TestExplorationToolRefusesWithoutTask(t *testing.T) {
	ts := NewToolSet(nil, "")
	res, err := ts.NodeDetailTool().Call(context.Background(), json.RawMessage(`{"id":379}`), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError || !strings.Contains(res.Flatten(), "任务上下文") {
		t.Fatalf("want an explanatory tool error, got IsError=%v %q", res.IsError, res.Flatten())
	}
}

// A panicking tool must degrade to a tool error: the run continues and the process
// survives, instead of the supervisor restarting into the same crash on replay.
func TestGuardPanicConvertsPanicToToolError(t *testing.T) {
	boom := actool.Build(actool.Spec{
		Name: "boom",
		Run: func(context.Context, json.RawMessage, *actool.ToolContext) (actool.Result, error) {
			panic("nil map write")
		},
	})
	res, err := guardPanic(boom).Call(context.Background(), json.RawMessage(`{}`), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError || !strings.Contains(res.Flatten(), "nil map write") {
		t.Fatalf("want the panic reported as a tool error, got IsError=%v %q", res.IsError, res.Flatten())
	}
}
