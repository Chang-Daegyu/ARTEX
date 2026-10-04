// [한국어 파일 안내] traffic/proxy_test.go
// 출구 프록시 주소 검증·설정 갱신·에이전트용 로컬 URL 변환을 검증한다.
// 주소를 실제 외부 프록시에 연결하는 테스트가 아니라 URL/원자 포인터 상태의 계약을 확인한다.
package traffic

import "testing"

// 한국어 해설: 지원 스킴과 인증 정보 포함 URL은 허용하고 스킴/호스트 누락 및 미지원 프로토콜은 거절하는지 확인한다.
func TestValidateProxyURL(t *testing.T) {
	ok := []string{
		"http://127.0.0.1:8080",
		"https://proxy.example.com:3128",
		"socks5://10.0.0.1:1080",
		"socks5://user:pass@10.0.0.1:1080",
	}
	for _, raw := range ok {
		if _, err := ValidateProxyURL(raw); err != nil {
			t.Errorf("ValidateProxyURL(%q) unexpected error: %v", raw, err)
		}
	}
	bad := []string{
		"127.0.0.1:8080",         // no scheme
		"ftp://host:21",          // unsupported scheme
		"http://",                // no host
		"socks4://10.0.0.1:1080", // unsupported scheme
	}
	for _, raw := range bad {
		if _, err := ValidateProxyURL(raw); err == nil {
			t.Errorf("ValidateProxyURL(%q) expected error, got nil", raw)
		}
	}
}

// 한국어 해설: 저장·인증 정보 보존·빈 값 초기화·잘못된 새 값의 비변경 동작을 순서대로 검증한다.
func TestSetUpstreamProxyStoreClear(t *testing.T) {
	tr := &Traffic{}
	if got := tr.upstream.Load(); got != nil {
		t.Fatalf("initial upstream = %v, want nil", got)
	}
	if err := tr.SetUpstreamProxy("socks5://user:pass@10.0.0.1:1080"); err != nil {
		t.Fatalf("SetUpstreamProxy: %v", err)
	}
	u := tr.upstream.Load()
	if u == nil || u.Scheme != "socks5" || u.Host != "10.0.0.1:1080" {
		t.Fatalf("stored upstream = %v, want socks5://10.0.0.1:1080", u)
	}
	if pw, _ := u.User.Password(); u.User.Username() != "user" || pw != "pass" {
		t.Fatalf("stored upstream lost credentials: %v", u)
	}
	// Empty clears back to direct.
	if err := tr.SetUpstreamProxy("  "); err != nil {
		t.Fatalf("SetUpstreamProxy(clear): %v", err)
	}
	if got := tr.upstream.Load(); got != nil {
		t.Fatalf("after clear upstream = %v, want nil", got)
	}
	// Invalid value is rejected and does not mutate current state.
	if err := tr.SetUpstreamProxy("nope://x"); err == nil {
		t.Fatal("SetUpstreamProxy(invalid) expected error")
	}
	if got := tr.upstream.Load(); got != nil {
		t.Fatalf("invalid set mutated upstream to %v, want nil", got)
	}
}

// 한국어 해설: 포트만 지정한 바인드 주소를 접속 가능한 127.0.0.1 URL로 바꾸고 명시적 호스트는 유지하는지 확인한다.
func TestProxyAddr(t *testing.T) {
	cases := []struct {
		addr string
		want string
	}{
		// Bare :port means "bind all interfaces" — the legacy default. The URL
		// agents consume must still point at loopback so they reach the local proxy.
		{":8788", "http://127.0.0.1:8788"},
		// Explicit loopback — the current default since #129 (open proxy exposure).
		{"127.0.0.1:8788", "http://127.0.0.1:8788"},
		// Explicit all-interface bind is still supported (remote capture via SSH).
		{"0.0.0.0:8788", "http://0.0.0.0:8788"},
	}
	for _, c := range cases {
		got := (&Traffic{addr: c.addr}).ProxyAddr()
		if got != c.want {
			t.Errorf("ProxyAddr(%q) = %q, want %q", c.addr, got, c.want)
		}
	}
}
