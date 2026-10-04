// [한국어 길잡이] task_control_routes_test.go의 테스트 읽기
// Worker 의도 제어 URL이 올바른 핸들러에 연결되고 잘못된/없는 의도를 의미 있는 상태로 응답하는지 확인한다.
// 테스트를 읽을 때는 임시 데이터/가짜 provider 준비 → 핸들러 또는 함수 호출 → 응답·DB·컨텍스트 검증 순서로 따라간다.
// 주요 진입점: TestWorkerControlRoutes.
// DB를 쓰는 사례는 별도의 테스트용 PostgreSQL에서 실행한다. 테스트 파일의 Skip 분기는 환경이 없을 때의 건너뛰기이지 기능 검증 성공을 뜻하지 않는다.
// 전체 모듈 설명과 파일별 읽기 순서: docs/ko/modules/server.md
package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWorkerControlRoutes(t *testing.T) {
	m, err := NewManager(t.TempDir(), "")
	if err != nil {
		t.Skipf("postgres unavailable (%v) - skipping", err)
	}
	defer m.Close()

	td := t.TempDir()
	s := New(context.Background(), m, td, td, td)
	h := s.Handler()
	token, err := signJWT(s.jwtKey)
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}

	request := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"action":"pause"}`))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	// The single-worker route is still registered and reaches its JSON handler.
	single := request("/api/tasks/missing/intents/1/control")
	if single.Code != http.StatusNotFound || !strings.Contains(single.Body.String(), `"error":"task not found"`) {
		t.Fatalf("single control route unavailable: status=%d body=%s", single.Code, single.Body.String())
	}

	// The former batch route must fall through the mux instead of reaching a task handler.
	batch := request("/api/tasks/missing/intents/control/batch")
	if batch.Code != http.StatusNotFound || !strings.Contains(batch.Body.String(), "404 page not found") {
		t.Fatalf("batch control route still registered: status=%d body=%s", batch.Code, batch.Body.String())
	}
}
