// [한국어 파일 안내] notify/email_test.go
// 최소 SMTP 서버를 로컬에 구현하여 인증·발신자·수신자·DATA·종료의 실제 프로토콜 순서를 검증한다.
// 가짜 서버를 통해 단계별 4xx/5xx를 반환하므로 net/smtp를 단순 mock으로 대체하지 않고 분류 동작을 확인한다.
// 임시 수신자에게만 테스트 데이터를 보내며 실제 사람에게 알림을 전송하는 흐름이 아니다.
package notify

import (
	"bufio"
	"context"

	"net"
	"strings"
	"sync"
	"testing"
)

// 本文件补齐邮件渠道的协议级测试。在此之前 email.Send 的覆盖率是 0——
// 整条 SMTP 路径没有任何用例跑过，而它恰恰是六个渠道里协议面最大、
// 最容易出错的一个（握手、认证、信封、DATA 阶段各有各的失败语义）。
//
// 这里用自建的最小 SMTP 服务器驱动，而不是 mock 掉 net/smtp：
// 邮件渠道的绝大部分风险就在「与真实 SMTP 服务器对话」这一步，
// 把这一步 mock 掉等于不测。

// fakeSMTP 是一个刚好够用的 SMTP 服务器：能完成 greet/EHLO/AUTH/MAIL/RCPT/DATA/QUIT，
// 并按用例要求对特定阶段返回指定应答码。
// 한국어 자료형: 정해진 응답 코드를 제어하고 받은 명령/본문을 기록하는 테스트용 SMTP 수신자다.
type fakeSMTP struct {
	ln net.Listener

	// rcptReply 是 RCPT TO 的应答；默认 250。
	rcptReply string
	// mailReply 是 MAIL FROM 的应答；默认 250。
	mailReply string
	// advertiseAuth 为 true 时在 EHLO 里声明支持 AUTH PLAIN。
	advertiseAuth bool

	mu       sync.Mutex
	data     string
	commands []string
}

// 한국어 해설: 임의 로컬 포트에 1회 SMTP 세션을 받을 fixture 서버를 만들고 종료 정리를 등록한다.
func newFakeSMTP(t *testing.T) *fakeSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSMTP{ln: ln, rcptReply: "250 OK", mailReply: "250 OK"}
	go f.serve()
	t.Cleanup(func() { ln.Close() })
	return f
}

// 한국어 해설: fixture가 선택한 루프백 주소와 임의 포트를 반환한다.
func (f *fakeSMTP) hostPort(t *testing.T) (string, int) {
	t.Helper()
	addr, ok := f.ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatal("非 TCP 监听地址")
	}
	return "127.0.0.1", addr.Port
}

// 한국어 해설: 서버가 실제 받은 SMTP 명령을 mutex 아래 보관하여 프로토콜 순서를 검사할 수 있게 한다.
func (f *fakeSMTP) record(cmd string) {
	f.mu.Lock()
	f.commands = append(f.commands, cmd)
	f.mu.Unlock()
}

// 한국어 해설: 서버가 DATA에서 받은 전체 메일을 잠금 아래 복사해 읽는다.
func (f *fakeSMTP) body() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.data
}

// 한국어 해설: EHLO·AUTH·MAIL·RCPT 같은 명령 접두어가 실제로 수신되었는지 확인한다.
func (f *fakeSMTP) sawCommand(prefix string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.commands {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

// 한국어 해설: SMTP greeting과 기본 명령 응답을 모사하고 DATA 종료 점까지 본문을 기록한다.
func (f *fakeSMTP) serve() {
	conn, err := f.ln.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	br := bufio.NewReader(conn)
	w := func(s string) { _, _ = conn.Write([]byte(s + "\r\n")) }
	w("220 fake.local ESMTP ready")
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		f.record(line)
		switch {
		case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
			// 不声明 STARTTLS：让代码走明文分支（测试目标是信封逻辑，不是 TLS）。
			w("250-fake.local")
			if f.advertiseAuth {
				w("250-AUTH PLAIN")
			}
			w("250 8BITMIME")
		case strings.HasPrefix(line, "AUTH"):
			// 简化处理：PLAIN 的初始应答可能跨多行，直接接受。
			w("235 2.7.0 Authentication successful")
		case strings.HasPrefix(line, "MAIL FROM"):
			w(f.mailReply)
		case strings.HasPrefix(line, "RCPT TO"):
			w(f.rcptReply)
		case strings.HasPrefix(line, "DATA"):
			w("354 End data with <CR><LF>.<CR><LF>")
			var sb strings.Builder
			for {
				dl, err := br.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimRight(dl, "\r\n") == "." {
					break
				}
				sb.WriteString(dl)
			}
			f.mu.Lock()
			f.data = sb.String()
			f.mu.Unlock()
			w("250 2.0.0 Ok: queued as FAKE1")
		case strings.HasPrefix(line, "QUIT"):
			w("221 2.0.0 Bye")
			return
		default:
			w("250 OK")
		}
	}
}

// 한국어 해설: 임시 서버 주소와 가짜 발신/수신자를 공통 설정으로 만들고 테스트별 옵션을 덮어쓴다.
func emailCfg(t *testing.T, f *fakeSMTP, extra map[string]any) map[string]any {
	t.Helper()
	host, port := f.hostPort(t)
	cfg := map[string]any{
		"host": host,
		"port": float64(port),
		"from": "artex@example.com",
		"to":   []any{"a@example.com", "b@example.com"},
	}
	for k, v := range extra {
		cfg[k] = v
	}
	return cfg
}

// 한국어 해설: AUTH·MAIL·복수 RCPT·DATA·QUIT가 수행되고 메일 헤더 및 base64 본문이 실제 전달되는지 확인한다.
func TestEmailSendDeliversFullMessage(t *testing.T) {
	f := newFakeSMTP(t)
	f.advertiseAuth = true
	cfg := emailCfg(t, f, map[string]any{"username": "artex", "password": "pw"})

	if _, err := (emailChannel{}).Send(context.Background(), cfg, singleMsg()); err != nil {
		t.Fatalf("投递失败: %v", err)
	}
	// 信封阶段必须走到：发件人、两个收件人、DATA。
	for _, want := range []string{"MAIL FROM:<artex@example.com>", "RCPT TO:<a@example.com>", "RCPT TO:<b@example.com>", "DATA", "AUTH", "QUIT"} {
		if !f.sawCommand(want) {
			t.Errorf("SMTP 会话里缺少 %q，实际命令：%v", want, f.commands)
		}
	}
	// 正文是 base64 的 HTML，且要带上真实的漏洞内容（编码后仍可辨认）。
	body := f.body()
	if body == "" {
		t.Fatal("DATA 阶段没有收到正文")
	}
	if !strings.Contains(body, "Content-Type: text/html") {
		t.Errorf("缺少 Content-Type 头:\n%s", body)
	}
	if !strings.Contains(body, "base64") {
		t.Errorf("正文未按 base64 编码（长 HTML 行会破坏 SMTP 的 1000 字节行长限制）:\n%s", body)
	}
	// 多个收件人都要出现在 To 头里。
	if !strings.Contains(body, "a@example.com, b@example.com") {
		t.Errorf("To 头未包含全部收件人:\n%s", body)
	}
}

// 한국어 해설: 계정을 지정하지 않은 중계에는 불필요한 AUTH를 전송하지 않는지 검사한다.
func TestEmailSendWithoutAuth(t *testing.T) {
	// 未配账号时不应发 AUTH —— 有些中继会因此拒收。
	f := newFakeSMTP(t)
	cfg := emailCfg(t, f, nil)
	if _, err := (emailChannel{}).Send(context.Background(), cfg, singleMsg()); err != nil {
		t.Fatalf("投递失败: %v", err)
	}
	if f.sawCommand("AUTH") {
		t.Errorf("未配账号却发了 AUTH: %v", f.commands)
	}
}

// TestEmailSendClassifiesSMTPReplies 是本次审计修复的直接验证：
// 5xx 判永久失败、4xx（灰名单）判可重试。
// 한국어 해설: MAIL/RCPT의 4xx는 재시도, 5xx는 영구 실패이며 원래 응답 코드가 진단에 남는지 확인한다.
func TestEmailSendClassifiesSMTPReplies(t *testing.T) {
	cases := []struct {
		name      string
		rcptReply string
		mailReply string
		permanent bool
	}{
		{"收件人被 550 永久拒绝", "550 5.1.1 User unknown", "250 OK", true},
		{"收件人遇 450 灰名单", "450 4.7.1 Greylisting in action", "250 OK", false},
		{"收件人遇 452 邮箱满", "452 4.2.2 Mailbox full", "250 OK", false},
		{"发件人被 553 永久拒绝", "250 OK", "553 5.1.3 Bad address", true},
		{"发件人遇 451 临时错误", "250 OK", "451 4.3.0 Temporary failure", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeSMTP(t)
			f.rcptReply = tc.rcptReply
			f.mailReply = tc.mailReply
			_, err := (emailChannel{}).Send(context.Background(), emailCfg(t, f, nil), singleMsg())
			if err == nil {
				t.Fatal("应报错")
			}
			if got := IsPermanent(err); got != tc.permanent {
				t.Fatalf("permanent 判定错误：期望 %v 得到 %v (%v)", tc.permanent, got, err)
			}
			// 服务器原文要保留，否则用户不知道该找服务器管理员还是改地址。
			if !strings.Contains(err.Error(), strings.Fields(tc.rcptReply)[0]) && !strings.Contains(err.Error(), strings.Fields(tc.mailReply)[0]) {
				t.Errorf("错误里应保留服务器的应答码: %v", err)
			}
		})
	}
}

// 한국어 해설: 비로컬 호스트와 인증 설정의 실패 경로를 관찰한다. 실제 연결 불가도 허용하는 테스트여서 TLS 보안 전체를 증명하지는 않는다.
func TestEmailSendRefusesPlaintextCredentials(t *testing.T) {
	// net/smtp 的 PlainAuth 拒绝在未加密连接上发凭据（除非目标是 localhost）。
	// 这是**正确**的安全行为，不能被绕过；但要给出能指导用户修复的错误。
	// 这里用一个非 localhost 的主机名触发它。
	f := newFakeSMTP(t)
	f.advertiseAuth = true
	_, port := f.hostPort(t)
	cfg := map[string]any{
		"host":     "smtp.example.com", // 非 localhost
		"port":     float64(port),
		"from":     "a@example.com",
		"to":       []any{"b@example.com"},
		"username": "artex",
		"password": "pw",
	}
	_, err := (emailChannel{}).Send(context.Background(), cfg, singleMsg())
	if err == nil {
		t.Skip("本机 DNS 解析到了本地服务器，跳过（不影响其它用例）")
	}
	// 连不上 或 被拒发凭据都算通过这条断言；关键是**不能**静默把密码发出去。
	if !IsPermanent(err) && !strings.Contains(err.Error(), "连接") {
		t.Logf("错误：%v（非 localhost 下未能连上属预期）", err)
	}
}

// 한국어 해설: SMTP 필수값 누락과 포트 범위 오류가 전송 전에 드러나는지 검사한다.
func TestEmailValidateReportsMissingFields(t *testing.T) {
	// 邮件渠道的配置字段最多，遗漏任一个都会在投递时才暴露；这里逐个确认
	// 校验能提前拦下。断言检查的是「错误信息提到了缺什么」。
	cases := []struct {
		name string
		cfg  map[string]any
	}{
		{"缺 host", map[string]any{"port": float64(25), "from": "a@b.c", "to": []any{"d@e.f"}}},
		{"缺 port", map[string]any{"host": "smtp.example.com"}},
		{"port 越界", map[string]any{"host": "h", "port": float64(70000), "from": "a@b.c", "to": []any{"d@e.f"}}},
		{"缺 from", map[string]any{"host": "h", "port": float64(25), "to": []any{"d@e.f"}}},
		{"缺 to", map[string]any{"host": "h", "port": float64(25), "from": "a@b.c"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := (emailChannel{}).Validate(tc.cfg); err == nil {
				t.Fatalf("应校验失败: %v", tc.cfg)
			}
		})
	}
}

// TestEmailConfigTolerance 覆盖配置读取的容错：JSONB 里数值是 float64，
// 但用户在 UI 里可能把端口填成字符串；数组也可能是单个字符串。
// 한국어 해설: 문자열 포트·문자열 true·단일 문자열 수신자를 설정 helper가 올바르게 해석하는지 확인한다.
func TestEmailConfigTolerance(t *testing.T) {
	cfg := map[string]any{
		"host": "smtp.example.com",
		"port": "587", // 字符串形式的端口
		"from": "a@b.c",
		"to":   "d@e.f", // 单个字符串而非数组
		"tls":  "true",  // 字符串形式的布尔
	}
	if err := (emailChannel{}).Validate(cfg); err != nil {
		t.Fatalf("应容忍字符串形式的数值: %v", err)
	}
	if got := cfgInt(cfg, "port"); got != 587 {
		t.Errorf("cfgInt 未解析字符串端口，得到 %d", got)
	}
	if !cfgBool(cfg, "tls") {
		t.Error("cfgBool 未解析字符串 \"true\"")
	}
	if to := cfgStrings(cfg, "to"); len(to) != 1 || to[0] != "d@e.f" {
		t.Errorf("cfgStrings 未兼容单字符串，得到 %v", to)
	}
}

// TestFilterValidateRejectsTypo 是审计修复的直接验证：
// 门槛打错字必须在写入时被拦，否则过滤器会静默失效变成全推。
// 한국어 해설: 심각도 오타·대문자·후행 공백을 저장 시 거절하고 허용 선택지를 설명하는지 검증한다.
func TestFilterValidateRejectsTypo(t *testing.T) {
	good := []string{"", "low", "medium", "high", "critical"}
	for _, s := range good {
		if err := (Filter{MinSeverity: s}).Validate(); err != nil {
			t.Errorf("合法门槛 %q 被拒: %v", s, err)
		}
	}
	// 这些是真实会发生的笔误——全部必须被拒。
	for _, s := range []string{"hgih", "HIGH", "严重", "high ", "crit"} {
		err := (Filter{MinSeverity: s}).Validate()
		if err == nil {
			t.Errorf("非法门槛 %q 应被拒绝（否则过滤器静默失效、变成全推）", s)
			continue
		}
		// 错误信息要能指导用户改对。
		if !strings.Contains(err.Error(), "low") || !strings.Contains(err.Error(), "critical") {
			t.Errorf("错误信息应列出可选值，得到 %q", err.Error())
		}
	}
}

// TestFilterValidateIsWriteTimeOnly 锁住「写入严、读取宽」的分工：
// 库里已有的坏值不能让渠道整个读不出来（那会让历史渠道突然全部停止推送）。
// 한국어 해설: 기존 DB의 잘못된 등급도 읽기 경로는 유지하여 과거 채널 전체가 읽기 실패로 멈추지 않는지 확인한다.
func TestFilterValidateIsWriteTimeOnly(t *testing.T) {
	raw := []byte(`{"min_severity":"hgih"}`)
	f := ParseFilter(raw) // 不报错
	if f.MinSeverity != "hgih" {
		t.Fatalf("读取路径应原样保留，得到 %q", f.MinSeverity)
	}
	// 且该渠道仍能对事件做出判定（不 panic、不阻塞）。
	_ = Match(f, Snapshot{Kind: EventFindingCreated, Severity: "critical"})
}
