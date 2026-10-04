// [한국어 길잡이] prompt_vars_test.go의 테스트 읽기
// 공통 프롬프트 변수를 합칠 때 같은 이름은 중복하지 않고 서로 다른 변수는 유지하는지 확인한다.
// 테스트를 읽을 때는 임시 데이터/가짜 provider 준비 → 핸들러 또는 함수 호출 → 응답·DB·컨텍스트 검증 순서로 따라간다.
// 주요 진입점: TestWithGlobalVarsDedupesCollidingNames, TestWithGlobalVarsKeepsDistinctVars.
// DB를 쓰는 사례는 별도의 테스트용 PostgreSQL에서 실행한다. 테스트 파일의 Skip 분기는 환경이 없을 때의 건너뛰기이지 기능 검증 성공을 뜻하지 않는다.
// 전체 모듈 설명과 파일별 읽기 순서: docs/ko/modules/server.md
package server

import (
	"testing"

	"github.com/Autumn-27/artex/db"
)

// A stored catalog entry that collides with a global runtime var (e.g. a legacy
// seeded goals."Now") must not produce a duplicate: the global wins and the name
// appears exactly once, so the UI never sees duplicate React keys.
func TestWithGlobalVarsDedupesCollidingNames(t *testing.T) {
	t.Parallel()
	stored := []db.PromptVar{
		{Name: "EngagementDescription", Description: "task", Source: "exploration"},
		{Name: "Now", Description: "stale per-agent copy", Source: "runtime"},
	}
	out := withGlobalVars(stored)

	counts := map[string]int{}
	var now db.PromptVar
	for _, v := range out {
		counts[v.Name]++
		if v.Name == "Now" {
			now = v
		}
	}
	if counts["Now"] != 1 {
		t.Fatalf("Now appeared %d times, want 1: %+v", counts["Now"], out)
	}
	if counts["EngagementDescription"] != 1 {
		t.Fatalf("non-colliding stored var was dropped or duplicated: %+v", out)
	}
	// The surviving Now must be the authoritative global definition, not the stale
	// stored one.
	if now.Description == "stale per-agent copy" {
		t.Fatalf("stored var shadowed the global instead of the other way around: %+v", now)
	}
	// Every global is present exactly once.
	for _, g := range globalPromptVars {
		if counts[g.Name] != 1 {
			t.Fatalf("global %q present %d times, want 1", g.Name, counts[g.Name])
		}
	}
}

// The common case (no collisions) keeps every stored var and appends all globals.
func TestWithGlobalVarsKeepsDistinctVars(t *testing.T) {
	t.Parallel()
	stored := []db.PromptVar{{Name: "Goal", Source: "exploration"}}
	out := withGlobalVars(stored)
	if len(out) != len(stored)+len(globalPromptVars) {
		t.Fatalf("len=%d, want %d: %+v", len(out), len(stored)+len(globalPromptVars), out)
	}
	if out[0].Name != "Goal" {
		t.Fatalf("stored var order not preserved: %+v", out)
	}
}
