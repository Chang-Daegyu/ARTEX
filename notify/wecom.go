// [한국어 파일 안내] notify/wecom.go
// WeCom 그룹 봇의 Markdown 채널이다.
// Webhook URL 안의 key가 자격 증명이므로 주소 전체를 마스킹하며 별도의 서명 필드는 없다.
// 4,096바이트 예산에 항목을 넣고 errcode=45009인 일시 제한은 재시도 가능한 오류로 구분한다.
package notify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// weComMarkdownLimit 是企微群机器人 markdown content 的硬上限（字节，非字符）。
// 这是全部六个渠道里最紧的限制，也是 TruncateBytes 存在的主要原因。
const weComMarkdownLimit = 4096

// weComChannel 实现企业微信群机器人。
//
// 平台特性：
//   - 唯一通过 URL 上的 key 鉴权，不支持加签——所以 webhook 地址本身就是全部凭据。
//   - markdown content 上限 4096 **字节**，超长整条被拒（不是截断）。中文 3 字节/字，
//     意味着正文只有一千多字可写，必须客户端截断。
//   - 限流 20 条/分钟，同样靠客户端限流兜住。
// 한국어 자료형: URL 자격 증명과 errcode 응답을 사용하는 WeCom 봇의 무상태 구현이다.
type weComChannel struct{}

// 한국어 해설: WeCom의 고정 채널 ID를 반환한다. DB 설정 및 registry 키와 일치해야 한다.
func (weComChannel) Kind() string { return KindWeCom }

// 한국어 해설: 새 채널 설정에 사용할 기본값을 분당 20로 제안한다. 0은 알려진 제한을 두지 않는다는 내부 계약이다.
func (weComChannel) DefaultRatePerMin() int { return 20 }

// 企业微信只有 Webhook 一处凭据（URL 上的 key），且它不支持加签——
// 整个地址就是全部凭据，没有别的字段需要掩码。
// 한국어 해설: API에서 마스킹할 비밀 필드를 webhook 전체로 선언한다. 실제 저장값 암호화 기능은 아니다.
func (weComChannel) SecretKeys() []string { return []string{"webhook"} }

// 企微只有 Webhook 一处字段，它既是目的地也是凭据，因此没有「改地址后残留的凭据」可言。
// 한국어 해설: 자격 증명 수신 위치를 결정하는 webhook를 선언한다. 이 값이 바뀌면 기존 비밀의 묵시적 재사용을 막는다.
func (weComChannel) DestinationKeys() []string { return []string{"webhook"} }

// 한국어 해설: Webhook 필수값과 공통 HTTP 목적지 규칙을 검사한다.
func (weComChannel) Validate(cfg map[string]any) error {
	hook := cfgString(cfg, "webhook")
	if hook == "" {
		return errors.New("缺少 Webhook 地址")
	}
	if err := validateHTTPURL(hook); err != nil {
		return fmt.Errorf("Webhook 地址无效: %w", err)
	}
	return nil
}

// 한국어 해설: UTF-8을 보존하면서 4,096바이트 안에 담은 Markdown을 전송하고 실제 포함 항목 수를 반환한다. 45009 외 업무 거절은 영구 오류로 분류한다.
func (c weComChannel) Send(ctx context.Context, cfg map[string]any, m Message) (int, error) {
	if err := c.Validate(cfg); err != nil {
		return 0, Permanent(err)
	}
	// 汇总批可能很长（50 条 × 每条一行 + 前缀），4096 字节很容易超。
	// 截断在这里做而不是靠平台报错：被拒意味着这一批全丢，而截断至少送达前若干条。
	content, kept := markdownBody(m, weComMarkdownLimit)
	payload := map[string]any{
		"msgtype":  "markdown",
		"markdown": map[string]any{"content": content},
	}
	raw, err := doJSON(ctx, "POST", cfgString(cfg, "webhook"), nil, payload)
	if err != nil {
		return 0, err
	}
	var res struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return 0, fmt.Errorf("解析企业微信响应失败: %w (%s)", err, snippet(raw))
	}
	if res.ErrCode != 0 {
		// 45009 是接口调用超过限制——平台的限流窗口会滚动，退避后重试是有效的，
		// 所以显式归为可重试。走到这里说明客户端 rate_per_min 配得过于激进，
		// 重试只是兜底，真正的修法是调低该渠道的限流值。
		if res.ErrCode == 45009 {
			return 0, fmt.Errorf("企业微信限流 %d: %s", res.ErrCode, res.ErrMsg)
		}
		// 93000 是 webhook key 无效——永久失败，重试不会自愈。
		return 0, Permanent(fmt.Errorf("企业微信返回错误 %d: %s", res.ErrCode, res.ErrMsg))
	}
	return kept, nil
}
