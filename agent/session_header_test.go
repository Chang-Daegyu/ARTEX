package agent

// 한국어 파일 해설: agent/session_header_test.go
// HTTP 요청 context의 transcript 세션 ID를 사용자 지정 헤더에 전달하는지 검사한다.
// 헤더 이름 또는 세션 ID가 비어 있을 때 헤더를 생성하지 않는 경계도 확인한다.
// 테스트 이름은 아래의 각 Test 함수가 확인하는 구체적 경계를 나타낸다.
// 한국어 문서화 변경은 기존 테스트의 입력·예상값·실행 조건을 수정하지 않는다.

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Autumn-27/norma/transcript"
)

// fakeRT records the request it saw and returns a minimal 200 response.
type fakeRT struct{ seen *http.Request }

func (f *fakeRT) RoundTrip(req *http.Request) (*http.Response, error) {
	f.seen = req
	return &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil
}

func newReq(ctx context.Context) *http.Request {
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.example.com/v1/messages", strings.NewReader("{}"))
	return req
}

func TestSessionHeaderInjectedFromContext(t *testing.T) {
	base := &fakeRT{}
	rt := quotaAwareTransport{base: base, sessionHeaderKey: "x-session-id"}
	ctx := transcript.WithSessionID(context.Background(), "conv-42")
	if _, err := rt.RoundTrip(newReq(ctx)); err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	if got := base.seen.Header.Get("x-session-id"); got != "conv-42" {
		t.Fatalf("x-session-id = %q, want conv-42", got)
	}
}

func TestSessionHeaderSkippedWhenKeyEmpty(t *testing.T) {
	base := &fakeRT{}
	rt := quotaAwareTransport{base: base} // no key configured
	ctx := transcript.WithSessionID(context.Background(), "conv-42")
	if _, err := rt.RoundTrip(newReq(ctx)); err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	// The header name is whatever the user would have set; with no key, nothing
	// session-related is added. Assert the common key stays absent.
	if got := base.seen.Header.Get("x-session-id"); got != "" {
		t.Fatalf("unexpected session header %q with empty key", got)
	}
}

func TestSessionHeaderSkippedWhenNoSessionID(t *testing.T) {
	base := &fakeRT{}
	rt := quotaAwareTransport{base: base, sessionHeaderKey: "x-session-id"}
	// Context carries no session id (transcript persistence off).
	if _, err := rt.RoundTrip(newReq(context.Background())); err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	if got := base.seen.Header.Get("x-session-id"); got != "" {
		t.Fatalf("x-session-id = %q, want empty when no session id on context", got)
	}
}
