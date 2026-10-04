package db

// 한국어 테스트 안내
// Skill 호출 성공/누락/에이전트/작업별 집계 및 최근 호출 목록을 검증한다.
// 장부의 한 행이 어떤 통계에 포함되는지 입력 fixture와 기대값을 비교해 이해할 수 있다.

import (
	"testing"
)

// TestSkillUsageLedger writes a few rows and checks the three aggregates the
// skills page relies on: per-skill totals (resolved calls only), the miss list,
// and the recent-call list. Rows are namespaced by a unique skill name so the
// shared dev DB stays usable and the test cleans up after itself.
// 한국어 검증 목적: Skill 성공/누락 호출 장부와 전체·작업별·최근 사용 집계가 일관된지 확인한다.
func TestSkillUsageLedger(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	defer d.Close()

	const (
		skillA  = "zz-test-skill-a"
		missing = "zz-test-skill-missing"
	)
	cleanup := func() {
		_, _ = d.Exec(`DELETE FROM skill_usage WHERE skill IN ($1,$2)`, skillA, missing)
	}
	cleanup()
	defer cleanup()

	rows := []*SkillUsage{
		{Skill: skillA, AgentKey: "worker", TaskID: 991, ExplorationID: 5, IntentID: 7, ArgsLen: 12, Found: true},
		{Skill: skillA, AgentKey: "worker", TaskID: 991, ExplorationID: 5, ArgsLen: 0, Found: true},
		{Skill: skillA, AgentKey: "planner", TaskID: 992, ExplorationID: 6, Found: true},
		{Skill: skillA, AgentKey: "chatbot", SessionID: "conv-1", Found: true},
		{Skill: missing, AgentKey: "worker", TaskID: 991, Found: false},
		{Skill: missing, AgentKey: "worker", TaskID: 991, Found: false},
	}
	for _, r := range rows {
		if err := d.InsertSkillUsage(r); err != nil {
			t.Fatalf("insert %s: %v", r.Skill, err)
		}
	}

	stats, err := d.SkillStats()
	if err != nil {
		t.Fatal(err)
	}
	var got *SkillStat
	for i := range stats {
		if stats[i].Skill == skillA {
			got = &stats[i]
		}
		if stats[i].Skill == missing {
			t.Fatalf("misses must not appear in SkillStats: %+v", stats[i])
		}
	}
	if got == nil {
		t.Fatalf("skill %s absent from SkillStats", skillA)
	}
	if got.Calls != 4 {
		t.Errorf("calls: want 4, got %d", got.Calls)
	}
	// 991 + 992; the chat row has a NULL task_id and COUNT(DISTINCT) skips NULLs.
	if got.Tasks != 2 {
		t.Errorf("tasks: want 2, got %d", got.Tasks)
	}
	if len(got.Agents) != 3 {
		t.Errorf("agents: want 3 distinct, got %v", got.Agents)
	}
	if got.LastUsed == nil {
		t.Error("last_used must be set")
	}

	miss, err := d.MissingSkillStats(10)
	if err != nil {
		t.Fatal(err)
	}
	var missCalls int
	for _, m := range miss {
		if m.Skill == missing {
			missCalls = m.Calls
		}
	}
	if missCalls != 2 {
		t.Errorf("missing calls: want 2, got %d", missCalls)
	}

	calls, err := d.RecentSkillCalls(skillA, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 4 {
		t.Fatalf("recent calls: want 4, got %d", len(calls))
	}
	// newest first
	for i := 1; i < len(calls); i++ {
		if calls[i].TS.After(calls[i-1].TS) {
			t.Errorf("recent calls not newest-first at %d", i)
		}
	}

	byTask, err := d.SkillCallsByTask(991)
	if err != nil {
		t.Fatal(err)
	}
	var taskCalls int
	for _, s := range byTask {
		if s.Skill == skillA {
			taskCalls = s.Calls
		}
		if s.Skill == missing {
			t.Errorf("misses must not appear in SkillCallsByTask: %+v", s)
		}
	}
	if taskCalls != 2 {
		t.Errorf("task 991 calls: want 2, got %d", taskCalls)
	}
}
