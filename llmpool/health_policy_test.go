package llmpool

// 한국어 파일 해설: llmpool/health_policy_test.go
// 운영자 설정이 회로 차단기의 임계값과 냉각 시간을 바꾸는지 검사한다.
// 음수 임계값은 일시적 실패 차단만 끄고 확정적 실패는 유지하며 0은 기본 정책을 유지해야 한다.
// 테스트 이름은 아래의 각 Test 함수가 확인하는 구체적 경계를 나타낸다.
// 한국어 문서화 변경은 기존 테스트의 입력·예상값·실행 조건을 수정하지 않는다.

import (
	"testing"
	"time"
)

// A configured soft-trip threshold replaces the default: two transient failures
// are enough, and the fixed cooldown replaces the 1/5/30min ladder.
func TestSetPolicyOverridesThresholdAndCooldown(t *testing.T) {
	reg := NewRegistry(nil, nil)
	reg.SetPolicy(2, 90*time.Second)
	if reg.Trip(1, "429", false) {
		t.Fatal("tripped on the first transient failure, want the second")
	}
	if !reg.Trip(1, "429", false) {
		t.Fatal("should trip on transient failure #2")
	}
	st := reg.Get(1)
	if d := time.Until(st.OpenUntil); d < 80*time.Second || d > 90*time.Second {
		t.Fatalf("cooldown=%v, want ~90s", d)
	}
	// Every later trip keeps the same fixed window instead of climbing the ladder.
	reg.Trip(1, "429", true)
	st = reg.Get(1)
	if d := time.Until(st.OpenUntil); d < 80*time.Second || d > 90*time.Second {
		t.Fatalf("second cooldown=%v, want ~90s (fixed)", d)
	}
}

// A negative threshold turns off soft tripping entirely: transient failures never
// open the breaker, deterministic ones still do immediately.
func TestSetPolicyDisablesSoftTrip(t *testing.T) {
	reg := NewRegistry(nil, nil)
	reg.SetPolicy(-1, 0)
	for i := range 10 {
		if reg.Trip(1, "429", false) {
			t.Fatalf("transient failure #%d tripped the breaker, want never", i+1)
		}
	}
	if reg.IsOpen(1) {
		t.Fatal("breaker should stay closed for transient failures")
	}
	if !reg.Trip(1, "no credit", true) {
		t.Fatal("a hard failure must still trip immediately")
	}
	if d := time.Until(reg.Get(1).OpenUntil); d < 50*time.Second || d > 60*time.Second {
		t.Fatalf("cooldown=%v, want the default first rung (~1min)", d)
	}
}

// The zero policy is the historical behaviour: 3 transient failures, ladder cooldown.
func TestZeroPolicyKeepsDefaults(t *testing.T) {
	reg := NewRegistry(nil, nil)
	reg.SetPolicy(0, 0)
	for i := 1; i < softTripAfter; i++ {
		if reg.Trip(1, "429", false) {
			t.Fatalf("tripped after %d transient failures, want %d", i, softTripAfter)
		}
	}
	if !reg.Trip(1, "429", false) {
		t.Fatalf("should trip on failure #%d", softTripAfter)
	}
	if d := time.Until(reg.Get(1).OpenUntil); d < 50*time.Second || d > 60*time.Second {
		t.Fatalf("cooldown=%v, want the first ladder rung (~1min)", d)
	}
}
