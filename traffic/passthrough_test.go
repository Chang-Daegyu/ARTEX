// [한국어 파일 안내] traffic/passthrough_test.go
// MITM 오류 뒤 투명 통과 전환 조건을 검증한다.
// 합성 Flow와 오류 문자열만 사용하여 프록시/프로토콜 문제와 대상 연결 장애를 구분한다.
// pass에 들어갔다는 것은 이후 연결이 기록되지 않는다는 뜻이므로 정상 대상 장애를 잘못 분류하면 안 된다.
package traffic

import (
	"errors"
	"net/url"
	"testing"

	mproxy "github.com/lqqyt2423/go-mitmproxy/proxy"
)

// 한국어 해설: CONNECT의 host:port와 일반 hostname이 같은 투명 통과 키로 정규화되는지 확인한다.
func TestHostOnly(t *testing.T) {
	cases := map[string]string{
		"example.com:443": "example.com",
		"example.com":     "example.com",
		"10.0.0.1:8080":   "10.0.0.1",
	}
	for in, want := range cases {
		if got := hostOnly(in); got != want {
			t.Errorf("hostOnly(%q)=%q want %q", in, got, want)
		}
	}
}

// 한국어 해설: HTTP/2·HEAD·malformed 오류는 프록시 단서로 분류하고 연결 거절·reset·시간 초과는 그렇게 분류하지 않는지 비교한다.
func TestProxyCausedErr(t *testing.T) {
	proxy := []string{
		"protocol error: received DATA on a HEAD request",
		"http2: server sent GOAWAY",
		"malformed HTTP response",
	}
	target := []string{ // target-side failures must NOT trigger passthrough
		"dial tcp 1.2.3.4:443: connect: connection refused",
		"read: connection reset by peer",
		"context deadline exceeded",
	}
	for _, s := range proxy {
		if !proxyCausedErr(errors.New(s)) {
			t.Errorf("expected proxy-caused: %q", s)
		}
	}
	for _, s := range target {
		if proxyCausedErr(errors.New(s)) {
			t.Errorf("expected NOT proxy-caused: %q", s)
		}
	}
}

// 한국어 해설: 일반 연결 오류에는 pass가 비고 프로토콜 오류 뒤에만 호스트가 등록되는지 검사한다.
func TestMaybePassthroughFlagsHostOnce(t *testing.T) {
	tr := &Traffic{}
	f := &mproxy.Flow{Request: &mproxy.Request{URL: &url.URL{Host: "target.test:443"}}}

	// Target-caused error → do NOT flag (keep MITM + recording).
	tr.maybePassthrough(f, errors.New("connection refused"))
	if _, ok := tr.pass.Load("target.test"); ok {
		t.Fatal("target-caused error must not flag passthrough")
	}

	// Proxy-caused error → flag the host for transparent passthrough.
	tr.maybePassthrough(f, errors.New("protocol error: received DATA on a HEAD request"))
	if _, ok := tr.pass.Load("target.test"); !ok {
		t.Fatal("proxy-caused error must flag passthrough")
	}

	// The shouldIntercept rule uses hostOnly(req.Host); the CONNECT host carries a
	// port, so it must resolve to the same flagged key → intercept=false (tunnel).
	if _, tunnel := tr.pass.Load(hostOnly("target.test:443")); !tunnel {
		t.Fatal("flagged host must be recognized for the CONNECT form with port")
	}
}
