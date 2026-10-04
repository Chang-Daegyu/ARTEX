// [한국어 길잡이] 프로세스 종료 원인의 회귀 테스트
// 신호 컨텍스트를 취소하고 shutdownContext 결과에서 AbortShutdown의 명명된 사유가 보존되는지 검사한다.
// 단순 context.Canceled가 먼저 전파되는 경쟁을 막는 시작점 설계의 의도를 설명하는 테스트다.
// 전체 모듈 설명과 파일별 읽기 순서: docs/ko/modules/server.md
package main

import (
	"context"
	"testing"

	"github.com/Autumn-27/artex/agent"
)

func TestShutdownContextPreservesNamedCause(t *testing.T) {
	signalCtx, signalCancel := context.WithCancel(context.Background())
	ctx, shutdown := shutdownContext(signalCtx)
	defer shutdown(nil)
	signalCancel()
	<-ctx.Done()
	if code, _, _, ok := agent.AbortReason(ctx); !ok || code != "shutdown" {
		t.Fatalf("code=%q ok=%v, want shutdown", code, ok)
	}
}
