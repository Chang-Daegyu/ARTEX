// [한국어 길잡이] engine_llm_calls_test.go의 테스트 읽기
// 여러 goroutine이 BeginLLMCall/EndLLMCall을 호출해도 작업별 실행 중 LLM 카운터가 일치하는지 검증한다.
// 테스트를 읽을 때는 임시 데이터/가짜 provider 준비 → 핸들러 또는 함수 호출 → 응답·DB·컨텍스트 검증 순서로 따라간다.
// 주요 진입점: TestEngineActiveLLMCallsConcurrent.
// 이 테스트의 fixture와 단언을 함께 읽으면 일반 경로 외의 예외·경쟁 조건을 이해할 수 있다.
// 전체 모듈 설명과 파일별 읽기 순서: docs/ko/modules/server.md
package server

import (
	"sync"
	"testing"
)

func TestEngineActiveLLMCallsConcurrent(t *testing.T) {
	e := NewEngine(nil)
	const taskID = "42"
	const calls = 64

	var started sync.WaitGroup
	var release sync.WaitGroup
	var finished sync.WaitGroup
	started.Add(calls)
	finished.Add(calls)
	release.Add(1)
	for range calls {
		go func() {
			defer finished.Done()
			e.BeginLLMCall(taskID)
			started.Done()
			release.Wait()
			e.EndLLMCall(taskID)
		}()
	}
	started.Wait()
	if got := e.ActiveLLMCalls(taskID); got != calls {
		t.Fatalf("active calls = %d, want %d", got, calls)
	}
	release.Done()
	finished.Wait()
	if got := e.ActiveLLMCalls(taskID); got != 0 {
		t.Fatalf("active calls after completion = %d, want 0", got)
	}

	// A defensive extra end must never expose a negative UI count.
	e.EndLLMCall(taskID)
	if got := e.ActiveLLMCalls(taskID); got != 0 {
		t.Fatalf("active calls after extra end = %d, want 0", got)
	}
}
