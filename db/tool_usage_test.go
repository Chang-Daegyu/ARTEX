package db

// 한국어 테스트 안내
// 도구 사용 장부의 기록과 도구별 호출 횟수를 검증한다.
// 도구를 실제 실행하는 테스트가 아니라 도구 실행 계층이 넘긴 사용 메타데이터의 저장/집계 계약을 확인한다.

import "testing"

// 한국어 검증 목적: 도구 호출 메타데이터 한 행이 도구별 집계에 올바르게 반영되는지 확인한다.
func TestToolUsageLedger(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v) - skipping", err)
	}
	defer d.Close()

	const (
		toolA = "zz_test_tool_usage_a"
		toolB = "zz_test_tool_usage_b"
	)
	cleanup := func() {
		_, _ = d.Exec(`DELETE FROM tool_usage WHERE tool_key IN ($1,$2)`, toolA, toolB)
	}
	cleanup()
	defer cleanup()

	rows := []*ToolUsage{
		{ToolKey: toolA, AgentKey: "worker", TaskID: 991, ExplorationID: 5, IntentID: 7},
		{ToolKey: toolA, AgentKey: "planner", TaskID: 992, ExplorationID: 6},
		{ToolKey: toolA, AgentKey: "chatbot", SessionID: "conv-1"},
		{ToolKey: toolB, AgentKey: "worker", TaskID: 991},
	}
	for _, row := range rows {
		if err := d.InsertToolUsage(row); err != nil {
			t.Fatalf("insert %s: %v", row.ToolKey, err)
		}
	}

	counts, err := d.ToolUsageCounts()
	if err != nil {
		t.Fatal(err)
	}
	if got := counts[toolA]; got != 3 {
		t.Errorf("%s calls: want 3, got %d", toolA, got)
	}
	if got := counts[toolB]; got != 1 {
		t.Errorf("%s calls: want 1, got %d", toolB, got)
	}
}
