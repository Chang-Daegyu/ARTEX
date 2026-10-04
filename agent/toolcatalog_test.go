package agent

// 한국어 파일 해설: agent/toolcatalog_test.go
// 도구 카탈로그에 기본 역할별 바인딩이 맞게 합쳐지는지 검사한다.
// 설명/schema 덮어쓰기는 원래 handler를 유지하고 누락·null·빈 문자열에만 scalar 기본값을 넣는지 확인한다.
// 테스트 이름은 아래의 각 Test 함수가 확인하는 구체적 경계를 나타낸다.
// 한국어 문서화 변경은 기존 테스트의 입력·예상값·실행 조건을 수정하지 않는다.

import (
	"context"
	"encoding/json"
	"testing"

	actool "github.com/Autumn-27/norma/tool"
)

// TestBuiltinToolSeeds ensures the catalog builds from a nil-store ToolSet without
// panicking, has stable keys, unions agent bindings, and carries real descriptions.
func TestBuiltinToolSeeds(t *testing.T) {
	seeds := BuiltinToolSeeds()
	if len(seeds) == 0 {
		t.Fatal("no seeds")
	}
	byKey := map[string]ToolSeed{}
	for _, s := range seeds {
		if s.Key == "" || s.Desc == "" {
			t.Errorf("seed %q missing key/desc", s.Key)
		}
		byKey[s.Key] = s
	}
	// record_fact is bound to worker (also mainagent, which can log confirmed facts).
	rf, ok := byKey["record_fact"]
	hasWorker := false
	for _, a := range rf.Agents {
		if a == "worker" {
			hasWorker = true
		}
	}
	if !ok || !hasWorker {
		t.Errorf("record_fact agents = %v, want to include worker", rf.Agents)
	}
	// add_task_scope is bound to the planner (deliberate scope widening).
	if ts, ok := byKey["add_task_scope"]; !ok || len(ts.Agents) == 0 {
		t.Errorf("add_task_scope not seeded / has no agent binding: %v", ts.Agents)
	}
	// SDK generic tools (incl. sleep, now part of DefaultTools) are deliberately NOT
	// seeded — every agent owns them; they flow through ToolResolve untouched.
	for _, k := range []string{"Bash", "Read", "Write", "Edit", "Grep", "sleep"} {
		if _, ok := byKey[k]; ok {
			t.Errorf("SDK tool %q should not be seeded", k)
		}
	}
}

// TestDecorateToolInjectsDefaults verifies a schema "default" fills a missing param
// before the underlying handler runs, and an explicitly-provided value is kept.
func TestDecorateToolInjectsDefaults(t *testing.T) {
	var seen map[string]any
	base := actool.Build(actool.Spec{
		Name: "probe", Description: "orig",
		Run: func(_ context.Context, in json.RawMessage, _ *actool.ToolContext) (actool.Result, error) {
			_ = json.Unmarshal(in, &seen)
			return actool.Text("ok"), nil
		},
	})
	schema := map[string]any{"type": "object", "properties": map[string]any{
		"limit": map[string]any{"type": "integer", "description": "n", "default": float64(3)},
		"q":     map[string]any{"type": "string", "description": "query"},
	}}
	dec := DecorateTool(base, "new desc", schema)
	if dec.Description() != "new desc" {
		t.Errorf("description = %q", dec.Description())
	}

	// limit omitted → default 3 injected; q kept.
	if _, err := dec.Call(context.Background(), json.RawMessage(`{"q":"x"}`), nil); err != nil {
		t.Fatal(err)
	}
	if seen["limit"] != float64(3) || seen["q"] != "x" {
		t.Errorf("injected = %v, want limit=3 q=x", seen)
	}

	// limit provided → default does NOT override.
	if _, err := dec.Call(context.Background(), json.RawMessage(`{"limit":9}`), nil); err != nil {
		t.Fatal(err)
	}
	if seen["limit"] != float64(9) {
		t.Errorf("limit = %v, want 9 (no override)", seen["limit"])
	}
}
