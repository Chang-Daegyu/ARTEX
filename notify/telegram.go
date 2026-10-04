// [한국어 파일 안내] notify/telegram.go
// Telegram Bot API의 sendMessage에 HTML 형식 알림을 전송한다.
// 토큰은 /bot<token>/ 경로에 들어가므로 오류 URL과 base_url 변경 시 자격 증명 보호가 필요하다.
// 문자 수 예산을 사용하고 HTTP 성공 뒤 응답 ok와 error_code도 확인한다.
// base_url은 API 수신 서버를, chat_id는 그 서버 안의 수신 대화를 가리켜 설정 변경의 의미가 다르다.
package notify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// telegramTextLimit 是 Telegram sendMessage 的 text 字段上限（字符数）。
const telegramTextLimit = 4096

// telegramChannel 实现 Telegram Bot API。
//
// 平台特性：
//   - 鉴权全部在 URL path 里（/bot<token>/sendMessage），无需加签。
//   - 用 HTML 解析模式而不是 MarkdownV2：MarkdownV2 要求转义 `_*[]()~`>#+-=|{}.!`
//     共 18 个字符，漏一个就整条消息被拒；HTML 只需转义 & < > 三个。
//   - 业务错误同样藏在 HTTP 200 里，靠 ok 字段判断。
// 한국어 자료형: Bot API URL·JSON·HTML 렌더링과 오류 코드를 공통 채널 계약으로 제공한다.
type telegramChannel struct{}

// 한국어 해설: Telegram의 고정 채널 ID를 반환한다. DB 설정 및 registry 키와 일치해야 한다.
func (telegramChannel) Kind() string { return KindTelegram }

// Telegram 单聊约 1 条/秒、群组 20 条/分钟。取保守值。
// 한국어 해설: 새 채널 설정에 사용할 기본값을 분당 20로 제안한다. 0은 알려진 제한을 두지 않는다는 내부 계약이다.
func (telegramChannel) DefaultRatePerMin() int { return 20 }

// Bot Token 是完整凭据；chat_id 只是收件人，不算秘密（拿到它没有 Token 也发不了消息）。
// 한국어 해설: API에서 마스킹할 비밀 필드를 bot_token로 선언한다. 실제 저장값 암호화 기능은 아니다.
func (telegramChannel) SecretKeys() []string { return []string{"bot_token"} }

// base_url 决定 Token 被发往哪个 API 端点（如自建反代），改它必须重新表态 Token。
// 한국어 해설: 자격 증명 수신 위치를 결정하는 base_url를 선언한다. 이 값이 바뀌면 기존 비밀의 묵시적 재사용을 막는다.
func (telegramChannel) DestinationKeys() []string { return []string{"base_url"} }

// 한국어 해설: Bot Token·Chat ID를 요구하고 선택적 자체 API base_url을 공통 HTTP 규칙으로 검사한다.
func (telegramChannel) Validate(cfg map[string]any) error {
	if cfgString(cfg, "bot_token") == "" {
		return errors.New("缺少 Bot Token")
	}
	if cfgString(cfg, "chat_id") == "" {
		return errors.New("缺少 Chat ID")
	}
	if base := cfgString(cfg, "base_url"); base != "" {
		if err := validateHTTPURL(base); err != nil {
			return fmt.Errorf("API 地址无效: %w", err)
		}
	}
	return nil
}

// 한국어 해설: HTML 본문을 sendMessage로 보내고 ok이면 kept를 반환한다. 429는 재시도 가능하고 나머지 업무 거절은 영구 오류로 분류한다.
func (c telegramChannel) Send(ctx context.Context, cfg map[string]any, m Message) (int, error) {
	if err := c.Validate(cfg); err != nil {
		return 0, Permanent(err)
	}
	endpoint, err := telegramEndpoint(cfg)
	if err != nil {
		return 0, Permanent(err)
	}
	text, kept := telegramHTML(m)
	payload := map[string]any{
		"chat_id":                  cfgString(cfg, "chat_id"),
		"text":                     text,
		"parse_mode":               "HTML",
		"disable_web_page_preview": false,
	}
	raw, err := doJSON(ctx, "POST", endpoint, nil, payload)
	if err != nil {
		return 0, err
	}
	var res struct {
		OK          bool   `json:"ok"`
		ErrorCode   int    `json:"error_code"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return 0, fmt.Errorf("解析 Telegram 响应失败: %w (%s)", err, snippet(raw))
	}
	if res.OK {
		return kept, nil
	}
	// 429 是限流，退避后重试有效；其余（400 参数错、401 token 错、403 被拉黑、
	// 404 chat 不存在）都是配置问题，重试不会自愈。
	if res.ErrorCode == 429 {
		return 0, fmt.Errorf("Telegram 限流: %s", res.Description)
	}
	return 0, Permanent(fmt.Errorf("Telegram 返回错误 %d: %s", res.ErrorCode, res.Description))
}

// telegramEndpoint 拼出 sendMessage 地址。base_url 留空时用官方 API，
// 非空时用于自建 Bot API 反代（国内网络下的常见需求）。
// 한국어 해설: 기본/사용자 API 주소에 봇 토큰과 sendMessage 경로를 붙인다. 파싱 오류에는 전체 토큰 URL을 노출하지 않는다.
func telegramEndpoint(cfg map[string]any) (string, error) {
	base := cfgString(cfg, "base_url")
	if base == "" {
		base = "https://api.telegram.org"
	}
	base = strings.TrimSuffix(base, "/")
	token := cfgString(cfg, "bot_token")
	raw := base + "/bot" + token + "/sendMessage"
	u, err := url.Parse(raw)
	if err != nil {
		// 不透传 err：地址里含 Bot Token，且此时连 addr 都不该回显。
		return "", fmt.Errorf("拼接 API 地址失败（API 地址：%s）", redactRequestTarget(base))
	}
	return u.String(), nil
}

// telegramHTML 渲染 HTML 正文，返回正文与实际写入的条目数（见 Channel.Send）。
// 한국어 해설: 단일/묶음 HTML 메시지를 문자 예산에 맞추고 실제 포함 항목 수를 반환한다. 공통 제목의 Markdown 이스케이프는 가져오지 않는다.
func telegramHTML(m Message) (string, int) {
	var b strings.Builder
	b.WriteString("<b>" + telegramEscape(markdownTitle(m)) + "</b>\n")
	if m.Batch {
		// Telegram 的上限是**字符数**，所以打包也按字符计量（runeSize）。
		footer := ""
		if m.HomeURL != "" {
			footer = fmt.Sprintf("\n\n<a href=\"%s\">在平台中查看全部</a>", telegramEscapeAttr(m.HomeURL))
		}
		kept := packItemCount(m.Items, telegramTextLimit, telegramReservedRunes, footer, runeSize, func(it Item, idx int) string {
			return telegramBatchLine(it, idx+1)
		})
		items := m.Items[:kept]
		b.Reset()
		b.WriteString("<b>" + telegramEscape(telegramBatchTitle(m, items, len(m.Items))) + "</b>")
		for i, it := range items {
			b.WriteString("\n" + telegramEscape(telegramBatchLine(it, i+1)))
		}
		b.WriteString(footer)
		return TruncateHTML(b.String(), telegramTextLimit), kept
	}
	if len(m.Items) == 0 {
		return b.String(), 0
	}
	it := m.Items[0]
	if it.IsStatusChange() {
		b.WriteString(fmt.Sprintf("\n<b>状态变更</b>：%s → %s",
			telegramEscape(StatusLabel(it.FromStatus)), telegramEscape(StatusLabel(it.ToStatus))))
	}
	if it.VulnClass != "" && it.VulnClass != it.Title() {
		b.WriteString("\n<b>类型</b>：" + telegramEscape(it.VulnClass))
	}
	if a := assetLine(it.Assets, maxAssetsShown); a != "" {
		b.WriteString("\n<b>资产</b>：" + telegramEscape(a))
	}
	if s := OneLine(it.Summary, maxSummaryRunes); s != "" {
		b.WriteString("\n<b>摘要</b>：" + telegramEscape(s))
	}
	if it.DetailURL != "" {
		b.WriteString(fmt.Sprintf("\n\n<a href=\"%s\">查看详情</a>", telegramEscapeAttr(it.DetailURL)))
	}
	return TruncateHTML(b.String(), telegramTextLimit), 1
}

// telegramReservedRunes 预留给消息标题与可能出现的截断提示（按字符计）。
const telegramReservedRunes = 160

// telegramBatchLine 渲染汇总里的一条（未转义，由调用方统一转义）。
// 한국어 해설: 묶음 한 줄의 순번·등급·제목·자산 원문을 만들고 HTML 이스케이프는 상위 출력 단계에 맡긴다.
func telegramBatchLine(it Item, idx int) string {
	if a := assetLine(it.Assets, maxAssetsShown); a != "" {
		return fmt.Sprintf("%d. %s · %s — %s", idx, SeverityLabel(it.Severity), it.Title(), a)
	}
	return fmt.Sprintf("%d. %s · %s", idx, SeverityLabel(it.Severity), it.Title())
}

// telegramBatchTitle 渲染汇总消息的标题行。条数用的是**本条实际包含**的条数，
// 而不是本批总数——否则读者会以为消息头写的数字就是全部。
// 한국어 해설: 전체 항목 수와 이번 메시지 이후 남은 수를 구분하고 선택적 시간창을 표시한다.
func telegramBatchTitle(m Message, items []Item, total int) string {
	title := fmt.Sprintf("漏洞汇总 · 共 %d 条", total)
	if extra := total - len(items); extra > 0 {
		title += fmt.Sprintf("（显示前 %d 条，其余 %d 条下一条继续）", len(items), extra)
	}
	if m.WindowMinutes > 0 {
		title = fmt.Sprintf("近 %d 分钟 · %s", m.WindowMinutes, title)
	}
	return title
}

// telegramEscape 转义 HTML 文本内容。
// Telegram 只认这三种实体，转义后 &amp; 之类的已有实体会被二次转义——这正是
// 期望行为：我们要显示的是原始字符，不是让用户注入 HTML。
// 한국어 해설: 외부 텍스트의 &, <, >를 Telegram HTML 엔티티로 치환하여 태그 주입을 막는다.
func telegramEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// telegramEscapeAttr 转义 HTML 属性值。在文本转义之外还要处理引号——
// URL 里带引号会提前闭合 href 属性，把后面的内容变成注入点。
// 한국어 해설: href 속성에 들어갈 URL은 HTML 텍스트 처리 후 큰따옴표도 치환한다.
func telegramEscapeAttr(s string) string {
	s = telegramEscape(s)
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
