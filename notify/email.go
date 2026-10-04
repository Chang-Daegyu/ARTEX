// [한국어 파일 안내] notify/email.go
// SMTP 연결·선택적 인증·MAIL/RCPT/DATA 순서로 HTML 메일을 전송한다.
// TLS=true는 즉시 TLS로 연결하는 방식이며, false는 일반 연결 뒤 서버가 지원하면 STARTTLS로 올린다.
// 연결 10초와 세션 45초 한도를 두어 느린 SMTP가 전달 엔진을 계속 점유하지 않게 한다.
// 성공은 SMTP 서버가 메시지를 수락했다는 뜻이며 실제 사람의 수신함 도착·열람 확인과 구분한다.
package notify

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// emailDialTimeout / emailSessionTimeout 分别约束建连与整段 SMTP 会话。
// net/smtp 自身没有任何超时机制，不设这两道的话，一个卡住的对端会让
// 投递 goroutine 永久挂在那里——而 dispatcher 是单 goroutine 串行处理的，
// 等于整个通知系统停摆。
const (
	emailDialTimeout    = 10 * time.Second
	emailSessionTimeout = 45 * time.Second
)

// emailChannel 实现 SMTP 邮件投递。
// 한국어 자료형: SMTP 메일을 전송하는 무상태 채널 구현이다. 특정 계정의 자격 증명은 저장하지 않는다.
type emailChannel struct{}

// 한국어 해설: SMTP 메일의 고정 채널 ID를 반환한다. DB 설정 및 registry 키와 일치해야 한다.
func (emailChannel) Kind() string { return KindEmail }

// 邮件没有平台限流，但不该用它刷屏；给一个宽松的默认值。
// 한국어 해설: 새 채널 설정에 사용할 기본값을 분당 60로 제안한다. 0은 알려진 제한을 두지 않는다는 내부 계약이다.
func (emailChannel) DefaultRatePerMin() int { return 60 }

// 只掩码密码。SMTP 主机、账号、收件人都不算秘密，掩码它们只会让编辑变麻烦。
// 한국어 해설: API에서 마스킹할 비밀 필드를 password로 선언한다. 실제 저장값 암호화 기능은 아니다.
func (emailChannel) SecretKeys() []string { return []string{"password"} }

// host/port 决定把密码交给哪台服务器；tls 决定是否加密传输。三者任一变化都
// 要求重新表态密码——顺带让「关掉 TLS」这一步必须显式带上凭据，而不是顺手一改。
// 한국어 해설: 자격 증명 수신 위치를 결정하는 host·port·tls를 선언한다. 이 값이 바뀌면 기존 비밀의 묵시적 재사용을 막는다.
func (emailChannel) DestinationKeys() []string { return []string{"host", "port", "tls"} }

// 한국어 해설: SMTP 호스트·유효 포트·보내는 주소·최소 한 수신자를 확인한다. 실제 로그인 성공은 Send에서 결정된다.
func (emailChannel) Validate(cfg map[string]any) error {
	if cfgString(cfg, "host") == "" {
		return errors.New("缺少 SMTP 服务器地址")
	}
	port := cfgInt(cfg, "port")
	if port <= 0 || port > 65535 {
		return errors.New("SMTP 端口无效（应为 1-65535）")
	}
	if cfgString(cfg, "from") == "" {
		return errors.New("缺少发件人地址")
	}
	if len(cfgStrings(cfg, "to")) == 0 {
		return errors.New("至少需要一个收件人地址")
	}
	return nil
}

// 한국어 해설: 메일을 구성하고 TLS/인증·발신자·수신자·DATA 순서로 보낸다. DATA 제출 성공 뒤 Quit 실패는 이미 수락된 메시지를 실패로 바꾸지 않는다.
func (c emailChannel) Send(ctx context.Context, cfg map[string]any, m Message) (int, error) {
	if err := c.Validate(cfg); err != nil {
		return 0, Permanent(err)
	}
	host := cfgString(cfg, "host")
	port := cfgInt(cfg, "port")
	from := cfgString(cfg, "from")
	to := cfgStrings(cfg, "to")
	username := cfgString(cfg, "username")
	password := cfgString(cfg, "password")
	implicitTLS := cfgBool(cfg, "tls")

	msg, err := buildEmailMessage(from, to, m)
	if err != nil {
		return 0, Permanent(err)
	}

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	client, err := emailDial(ctx, addr, host, implicitTLS)
	if err != nil {
		return 0, err
	}
	defer client.Close()

	// STARTTLS：对端支持就升级。明文会话下不能发凭据（见下面的 auth 说明）。
	if !implicitTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
				return 0, fmt.Errorf("STARTTLS 失败: %w", err)
			}
		}
	}
	if username != "" {
		if err := client.Auth(smtp.PlainAuth("", username, password, host)); err != nil {
			// smtp.PlainAuth 会拒绝在未加密连接上发送凭据（除非目标是 localhost）。
			// 这是**正确**的安全行为，不能绕过，但需要把原因翻译清楚——
			// 否则使用者只会看到「unencrypted connection」而不知道该怎么办。
			if strings.Contains(err.Error(), "unencrypted connection") {
				return 0, Permanent(fmt.Errorf("拒发凭据：连接未加密。请启用 TLS，或改用 465 端口(隐式 TLS)，或把「启用 TLS」勾上 (%w)", err))
			}
			return 0, Permanent(fmt.Errorf("SMTP 认证失败: %w", err))
		}
	}
	if err := client.Mail(from); err != nil {
		return 0, smtpStageError(fmt.Sprintf("发件人 %s 被拒", from), err)
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return 0, smtpStageError(fmt.Sprintf("收件人 %s 被拒", rcpt), err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return 0, fmt.Errorf("SMTP DATA 失败: %w", err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return 0, fmt.Errorf("写入邮件正文失败: %w", err)
	}
	if err := w.Close(); err != nil {
		return 0, fmt.Errorf("提交邮件失败: %w", err)
	}
	// Quit 失败不影响「邮件已被服务器接收」这个事实，因此忽略其错误。
	_ = client.Quit()
	// 邮件没有长度截断（HTML 正文全部发送），整批都算送达。
	return len(m.Items), nil
}

// emailDial 建立 SMTP 连接。
//
// implicitTLS=true 走 465 这类「连上即 TLS」的方式；false 走 25/587 明文建连后再
// STARTTLS。两者不能混：对 465 端口发明文 greeting 会被直接断开。
//
// 会话期限在**建连处**就设好（而非事后补设），因为 net/smtp 的 Client 把底层
// 连接藏在未导出字段里，外部拿不到它；连接一旦交出去就只能靠预先设置的 deadline
// 兜底。这也顺带覆盖了握手阶段的阻塞。
// Control 挂 blockInternalDial 与 HTTP 系渠道共用同一道守卫。不挂的话 SMTP
// 就是整套 SSRF 防护的缺口：host 填 169.254.169.254 或 127.0.0.1 能直接连上，
// 而 smtp.NewClient 握手失败时会把对端返回的那一行包进错误、经 last_error
// 由投递历史接口回显，构成半盲读原语；「连接被拒 vs 超时」的耗时差异还能
// 用来探测端口。拨号阶段是最终生效点，也覆盖 DNS 重绑定。
// 한국어 해설: HTTP 채널과 같은 연결 IP 검사로 SMTP를 열고 전체 세션 deadline을 설정한다. TLS 경로는 최소 TLS 1.2와 서버 이름을 사용한다.
func emailDial(ctx context.Context, addr, host string, implicitTLS bool) (*smtp.Client, error) {
	d := &net.Dialer{Timeout: emailDialTimeout, Control: blockInternalDial}
	var conn net.Conn
	var err error
	if implicitTLS {
		conn, err = tls.DialWithDialer(d, "tcp", addr, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
	} else {
		conn, err = d.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return nil, fmt.Errorf("连接 SMTP 服务器失败: %w", err)
	}
	_ = conn.SetDeadline(time.Now().Add(emailSessionTimeout))
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("SMTP 握手失败: %w", err)
	}
	return client, nil
}

// smtpStageError 按 SMTP 应答码把某个阶段的失败分成「可重试」与「永久失败」。
//
// 为什么必须区分：SMTP 的 4xx 与 5xx 语义完全不同——
//   - 4xx（450 灰名单、451 本地错误、452 存储不足）是**临时**拒绝，
//     正规做法是稍后重试；尤其是灰名单，几乎每次首次投递都会遇到。
//   - 5xx（550 用户不存在、553 地址非法）是永久拒绝，重试没有意义。
//
// 若一律判永久失败，一个启用灰名单的邮件服务器会让**每一条**推送都在第一次
// 尝试后落入 failed——而这类失败恰恰是自动重试最该发挥作用的场景。
// 应答码取错误文本的前三位数字；取不到码时按可重试处理（宁可多试一次，
// 也不要因为解析不出就把可能的瞬时故障判死）。
// 한국어 해설: SMTP 5xx는 영구 거절, 4xx나 판독 불가 오류는 일시 실패로 표시한다. HTTP의 4xx 분류와 혼동하지 않는다.
func smtpStageError(what string, err error) error {
	code := smtpReplyCode(err.Error())
	if code >= 500 && code < 600 {
		return Permanent(fmt.Errorf("%s: %w", what, err))
	}
	return fmt.Errorf("%s: %w", what, err)
}

// smtpReplyCode 从 SMTP 错误文本里取前导的三位应答码，取不到返回 0。
// net/smtp 不导出错误码字段，只能从文本里取；格式为「450 4.7.1 ...」。
// 한국어 해설: 오류 문자열 맨 앞 3자리 숫자를 SMTP 응답 코드로 읽고 없으면 0을 반환한다.
func smtpReplyCode(text string) int {
	if len(text) < 3 {
		return 0
	}
	n, err := strconv.Atoi(text[:3])
	if err != nil {
		return 0
	}
	return n
}

// buildEmailMessage 组装完整的 RFC 5322 邮件。
//
// 正文用 base64 编码有两个原因：一是 SMTP 规定单行不超过 1000 字节，而 HTML
// 正文（尤其汇总邮件）很容易出现超长行；二是 base64 天然不会出现以 "." 开头
// 的行，省去 SMTP 点号转义的麻烦。
// 한국어 해설: UTF-8 제목을 MIME 인코딩하고 HTML 본문을 base64 76문자 줄로 나누어 전송 가능한 메일 원문을 만든다.
func buildEmailMessage(from string, to []string, m Message) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", strings.Join(to, ", "))
	// 中文主题必须做 RFC 2047 编码，否则会被客户端显示成乱码。
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", htmlTitle(m)))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n")
	// 邮件没有长度硬上限，因此不截断正文。
	b.WriteString("\r\n")
	encoded := base64.StdEncoding.EncodeToString([]byte(htmlBody(m, 0)))
	// base64 按 76 字符折行，符合 RFC 2045。
	for len(encoded) > 76 {
		b.WriteString(encoded[:76] + "\r\n")
		encoded = encoded[76:]
	}
	b.WriteString(encoded + "\r\n")
	return b.String(), nil
}
