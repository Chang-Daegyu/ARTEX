// [한국어 파일 안내] notify/dingtalk.go
// DingTalk 봇에 Markdown 또는 상세 링크 버튼이 있는 ActionCard를 전송한다.
// HTTP 200 뒤의 errcode도 확인하여 플랫폼 거절을 성공으로 오인하지 않는다.
// 선택적 서명은 밀리초 시각과 secret으로 만든 메시지를 secret 키의 HMAC-SHA256에 넣는다.
// 아래 기본 속도와 크기 값은 이 소스가 채택한 설정값이며 향후 플랫폼 정책의 자동 갱신 정보는 아니다.
package notify

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"
)

// dingTalkChannel 实现钉钉自定义机器人。
//
// 平台特性（决定了这里的实现取舍）：
//   - 单机器人限流 20 条/分钟，超发会被静默丢弃（HTTP 仍可能 200），
//     所以限流必须在客户端做，见 DefaultRatePerMin。
//   - 安全设置三选一：加签 / 自定义关键词 / IP 白名单。加签是唯一不依赖
//     消息内容的方案，所以只支持加签（也支持三者都不开的裸 webhook）。
//   - 成功/失败都返回 HTTP 200，靠 body 里的 errcode 区分——不检查 errcode
//     会把投递失败记成成功。
// 한국어 자료형: DingTalk의 메시지 형식·서명·업무 오류를 공통 Channel 계약에 맞추는 무상태 어댑터다.
type dingTalkChannel struct{}

// 한국어 해설: DingTalk의 고정 채널 ID를 반환한다. DB 설정 및 registry 키와 일치해야 한다.
func (dingTalkChannel) Kind() string { return KindDingTalk }

// 한국어 해설: 새 채널 설정에 사용할 기본값을 분당 20로 제안한다. 0은 알려진 제한을 두지 않는다는 내부 계약이다.
func (dingTalkChannel) DefaultRatePerMin() int { return 20 }

// 钉钉的 Webhook 地址里带 access_token，本身就是凭据，因此整体掩码。
// 한국어 해설: API에서 마스킹할 비밀 필드를 webhook와 secret로 선언한다. 실제 저장값 암호화 기능은 아니다.
func (dingTalkChannel) SecretKeys() []string { return []string{"webhook", "secret"} }

// 目标是钉钉的 Webhook 地址本身；改地址必须同时对新地址重新表态加签密钥。
// 한국어 해설: 자격 증명 수신 위치를 결정하는 webhook를 선언한다. 이 값이 바뀌면 기존 비밀의 묵시적 재사용을 막는다.
func (dingTalkChannel) DestinationKeys() []string { return []string{"webhook"} }

// 한국어 해설: Webhook URL의 존재·스킴·기본 목적지 조건을 확인한다.
func (dingTalkChannel) Validate(cfg map[string]any) error {
	hook := cfgString(cfg, "webhook")
	if hook == "" {
		return errors.New("缺少 Webhook 地址")
	}
	if err := validateHTTPURL(hook); err != nil {
		return fmt.Errorf("Webhook 地址无效: %w", err)
	}
	return nil
}

// Send 投递一次消息。有回链且是单条时用 ActionCard（带按钮），否则用 markdown。
// 한국어 해설: 단일 항목에 상세 링크가 있으면 ActionCard, 그 외에는 Markdown으로 보내고 errcode를 검사한다.
func (c dingTalkChannel) Send(ctx context.Context, cfg map[string]any, m Message) (int, error) {
	hook := cfgString(cfg, "webhook")
	if err := c.Validate(cfg); err != nil {
		return 0, Permanent(err)
	}
	endpoint, err := dingTalkSignedURL(hook, cfgString(cfg, "secret"), time.Now())
	if err != nil {
		return 0, Permanent(err)
	}

	title := markdownTitle(m)
	// 钉钉 markdown 正文无明确字节上限，但仍做上限保护，避免证据字段异常膨胀。
	text, kept := markdownBody(m, 20000)

	var payload any
	if !m.Batch && len(m.Items) == 1 && m.Items[0].DetailURL != "" {
		payload = map[string]any{
			"msgtype": "actionCard",
			"actionCard": map[string]any{
				"title":          title,
				"text":           text,
				"btnOrientation": "0",
				"singleTitle":    "查看详情",
				"singleURL":      m.Items[0].DetailURL,
			},
		}
	} else {
		payload = map[string]any{
			"msgtype":  "markdown",
			"markdown": map[string]any{"title": title, "text": text},
		}
	}

	raw, err := doJSON(ctx, "POST", endpoint, nil, payload)
	if err != nil {
		return 0, err
	}
	// 钉钉把业务错误藏在 200 响应里。
	var res struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return 0, fmt.Errorf("解析钉钉响应失败: %w (%s)", err, snippet(raw))
	}
	if res.ErrCode != 0 {
		// 301000 是签名校验失败、310000 是关键词不匹配——都是配置错误，
		// 重试不会自愈。
		return 0, Permanent(fmt.Errorf("钉钉返回错误 %d: %s", res.ErrCode, res.ErrMsg))
	}
	return kept, nil
}

// dingTalkSignedURL 按官方加签规则给 webhook 追加 timestamp 与 sign 参数。
//
// 规则：待签串 = timestamp + "\n" + secret，HMAC-SHA256 的**密钥也是 secret**，
// 结果 base64 后 URL 编码。timestamp 是毫秒。secret 为空时原样返回，
// 以支持未开启加签的机器人。
// 한국어 해설: secret을 키로 timestamp+개행+secret을 HMAC-SHA256 처리한 뒤 base64와 URL 쿼리 인코딩으로 서명을 붙인다.
func dingTalkSignedURL(hook, secret string, now time.Time) (string, error) {
	if secret == "" {
		return hook, nil
	}
	ts := strconv.FormatInt(now.UnixMilli(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "\n" + secret))
	sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	u, err := url.Parse(hook)
	if err != nil {
		// 不透传 err：url.Parse 的错误文本里带完整地址（含 access_token）。
		return "", fmt.Errorf("解析 Webhook 地址失败: %s", redactRequestTarget(hook))
	}
	q := u.Query()
	q.Set("timestamp", ts)
	q.Set("sign", sign)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// validateHTTPURL 校验地址可用、协议受支持，并对字面 IP 目标做内网判断。
//
// 两点讲究：
//
//  1. **错误信息必须脱敏**。url.Parse 自己返回的是 *url.Error，它的 Error() 带
//     **完整原始地址**，而本功能这几家的地址里就嵌着凭据（钉钉 access_token、
//     企微 key、Telegram 的 bot token、飞书 hook id）。曾经这里直接 `return err`，
//     于是「地址格式非法」这条错误就把凭据带了出去，流向测试接口的 400 响应、
//     每次投递落库的 last_error、服务端日志与投递历史接口。
//
//  2. **字面 IP 直接判内网**，域名留给拨号阶段判（blockInternalDial 才是最终
//     生效点，也能覆盖 DNS 重绑定）。这里做一次是为了让保存配置时就能得到提示，
//     而不是等到第一次投递失败。
//
// 限制协议是防御性的：file:///gopher:// 之类会让 http.Client 产生意料之外的
// 行为（虽已被 scheme 检查挡下，但没有理由放开这个面）。
// 한국어 해설: http/https와 호스트를 요구하고 문자 IP의 제한 범위를 미리 검사한다. DNS 해석 뒤 최종 검사는 연결 hook이 맡는다.
func validateHTTPURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("地址无法解析（%s）", redactRequestTarget(raw))
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("只支持 http/https，收到 %q", u.Scheme)
	}
	if u.Host == "" {
		return errors.New("缺少主机名")
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil && isBlockedDialIP(ip) && !allowLocalTargets() {
		return fmt.Errorf("拒绝投递到本机/链路本地地址 %s（如确需投递到本机服务，设置 %s=1）", ip, AllowLocalTargetsEnv)
	}
	return nil
}
