// [한국어 파일 안내] notify/feishu.go
// Feishu/Lark 봇의 색상 있는 interactive 카드와 선택적 서명을 구성한다.
// 서명은 timestamp+개행+secret을 HMAC 키로 하고 메시지는 비워 둔다. DingTalk와 인자 배치가 다르다.
// 현재 기본 rate는 분당 100이며 원본 주석의 초당 수치와 단순 환산을 동일하다고 가정하지 않는다.
// code와 일부 구버전의 StatusCode 두 응답 필드를 함께 검사한다.
package notify

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// feishuChannel 实现飞书（含 Lark）自定义机器人，走交互式卡片。
//
// 平台特性：
//   - 加签算法与钉钉**不同**，且极易写错，见 feishuSign 注释。
//   - 与钉钉一样把业务错误塞在 HTTP 200 的 body 里（code != 0）。
//   - 卡片 header 支持颜色模板，用级别映射配色，让人在消息列表里一眼看出严重程度。
// 한국어 자료형: Feishu의 카드·서명·오류 코드를 공통 채널로 변환하는 무상태 구현이다.
type feishuChannel struct{}

// 한국어 해설: Feishu/Lark의 고정 채널 ID를 반환한다. DB 설정 및 registry 키와 일치해야 한다.
func (feishuChannel) Kind() string { return KindFeishu }

// 飞书自定义机器人约 5 次/秒，折合 100 次/分钟。
// 한국어 해설: 새 채널 설정에 사용할 기본값을 분당 100로 제안한다. 0은 알려진 제한을 두지 않는다는 내부 계약이다.
func (feishuChannel) DefaultRatePerMin() int { return 100 }

// Webhook 地址末段即机器人唯一标识，属凭据。
// 한국어 해설: API에서 마스킹할 비밀 필드를 webhook와 secret로 선언한다. 실제 저장값 암호화 기능은 아니다.
func (feishuChannel) SecretKeys() []string { return []string{"webhook", "secret"} }

// 同理：改 Webhook 地址必须对新地址重新表态签名密钥。
// 한국어 해설: 자격 증명 수신 위치를 결정하는 webhook를 선언한다. 이 값이 바뀌면 기존 비밀의 묵시적 재사용을 막는다.
func (feishuChannel) DestinationKeys() []string { return []string{"webhook"} }

// 한국어 해설: Webhook 주소를 저장/전송하기 전에 공통 HTTP URL 규칙으로 확인한다.
func (feishuChannel) Validate(cfg map[string]any) error {
	hook := cfgString(cfg, "webhook")
	if hook == "" {
		return errors.New("缺少 Webhook 地址")
	}
	if err := validateHTTPURL(hook); err != nil {
		return fmt.Errorf("Webhook 地址无效: %w", err)
	}
	return nil
}

// 한국어 해설: interactive 카드에 선택적 서명 필드를 붙여 POST하고 두 형태의 업무 오류 코드를 검사한다.
func (c feishuChannel) Send(ctx context.Context, cfg map[string]any, m Message) (int, error) {
	if err := c.Validate(cfg); err != nil {
		return 0, Permanent(err)
	}
	card, kept := feishuCard(m)
	payload := map[string]any{
		"msg_type": "interactive",
		"card":     card,
	}
	// 加签参数与消息同层，且只在配置了 secret 时出现。
	if secret := cfgString(cfg, "secret"); secret != "" {
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		payload["timestamp"] = ts
		payload["sign"] = feishuSign(ts, secret)
	}
	raw, err := doJSON(ctx, "POST", cfgString(cfg, "webhook"), nil, payload)
	if err != nil {
		return 0, err
	}
	var res struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		// 部分版本的飞书 hook 用这套字段名，一并兼容。
		StatusCode    int    `json:"StatusCode"`
		StatusMessage string `json:"StatusMessage"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return 0, fmt.Errorf("解析飞书响应失败: %w (%s)", err, snippet(raw))
	}
	if res.Code != 0 {
		return 0, Permanent(fmt.Errorf("飞书返回错误 %d: %s", res.Code, res.Msg))
	}
	if res.StatusCode != 0 {
		return 0, Permanent(fmt.Errorf("飞书返回错误 %d: %s", res.StatusCode, res.StatusMessage))
	}
	return kept, nil
}

// feishuSign 按飞书官方规则计算签名。
//
// 这里特别容易踩坑：官方样例是
//
//	hmac.new(string_to_sign.encode(), digestmod=sha256)
//
// 也就是 **key = timestamp + "\n" + secret，message 为空**，而不是直觉上的
// 「key=secret, message=stringToSign」——那正是钉钉的算法。两边算法刚好反过来，
// 照着另一家的实现写必然签名校验失败（报 19021）。
// 한국어 해설: timestamp와 secret을 합친 문자열을 HMAC 키로 쓰고 빈 메시지의 SHA-256 MAC을 base64로 반환한다.
func feishuSign(timestamp, secret string) string {
	stringToSign := timestamp + "\n" + secret
	mac := hmac.New(sha256.New, []byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// feishuSeverityTemplate 把漏洞级别映射到卡片 header 配色模板。
// 未知级别用 grey——不用 blue，免得和 low 混淆。
// 한국어 해설: 발견 심각도를 카드 header의 색상 템플릿 이름으로 바꾼다. 미분류는 grey다.
func feishuSeverityTemplate(severity string) string {
	switch severity {
	case "critical":
		return "red"
	case "high":
		return "orange"
	case "medium":
		return "yellow"
	case "low":
		return "blue"
	default:
		return "grey"
	}
}

// feishuMaxCardBytes 是卡片内容的保守上限。飞书对卡片有体积限制，超了整条被拒；
// 取一个明显低于官方上限的值，把 JSON 包装开销也算进来。
const feishuMaxCardBytes = 24000

// feishuCard 构造交互式卡片，返回卡片与**实际写入的条目数**。
// kept 的用途同 markdownBody：只有真正进了卡片的条目才该被标记为已送达。
// 한국어 해설: 단일/묶음 카드를 구성하고 묶음은 길이 예산에 들어간 항목 수를 반환한다. 카드 제목의 plain_text와 본문의 lark_md를 구분한다.
func feishuCard(m Message) (map[string]any, int) {
	elements := []any{}
	kept := 0
	if m.Batch {
		// 先按整条打包再拼头部：头部要写「其余 N 条将在下一条消息继续」，
		// N 必须来自实际装下的条数。
		kept = packItemCount(m.Items, feishuMaxCardBytes, markdownReservedBytes, "", byteSize, func(it Item, idx int) string {
			return feishuBatchLine(it, idx+1)
		})
		items := m.Items[:kept]
		elements = append(elements, feishuMarkdownDiv(markdownBatchIntro(m, items, len(m.Items))))
		for i, it := range items {
			elements = append(elements, feishuMarkdownDiv(feishuBatchLine(it, i+1)))
		}
		if m.HomeURL != "" {
			elements = append(elements, feishuButton("在平台中查看全部", m.HomeURL))
		}
	} else if len(m.Items) > 0 {
		kept = 1
		it := m.Items[0]
		elements = append(elements, feishuMarkdownDiv(feishuItemLines(it)))
		if it.DetailURL != "" {
			elements = append(elements, feishuButton("查看详情", it.DetailURL))
		}
	}

	card := map[string]any{
		"config":   map[string]any{"wide_screen_mode": true},
		"header":   map[string]any{"title": map[string]any{"tag": "plain_text", "content": markdownTitle(m)}},
		"elements": elements,
	}
	if len(m.Items) > 0 {
		card["header"].(map[string]any)["template"] = feishuSeverityTemplate(m.Items[0].Severity)
	}
	return card, kept
}

// 한국어 해설: 본문 문자열을 Feishu의 div/lark_md JSON 블록으로 감싼다.
func feishuMarkdownDiv(content string) map[string]any {
	return map[string]any{"tag": "div", "text": map[string]any{"tag": "lark_md", "content": content}}
}

// 한국어 해설: 표시 문구와 URL로 주 동작 버튼 JSON을 만든다.
func feishuButton(label, url string) map[string]any {
	return map[string]any{
		"tag": "action",
		"actions": []any{map[string]any{
			"tag":  "button",
			"text": map[string]any{"tag": "lark_md", "content": label},
			"url":  url,
			"type": "primary",
		}},
	}
}

// feishuItemLines 渲染单个漏洞的 lark_md 正文。
//
// lark_md 与 markdown 是同族的文本格式，同样会解析链接与强调，所以来自外部
// 的字段一律过 markdownText（单行化 + 转义）——否则一条漏洞标题就能在
// 飞书里变成可点击的外链。
// 한국어 해설: 단일 발견의 등급·상태·유형·자산·요약을 이스케이프된 lark_md로 렌더링한다.
func feishuItemLines(it Item) string {
	out := fmt.Sprintf("**%s · %s**", SeverityLabel(it.Severity), markdownText(it.Title(), 0))
	if it.IsStatusChange() {
		out += fmt.Sprintf("\n**状态变更**：%s → %s",
			markdownText(StatusLabel(it.FromStatus), 0), markdownText(StatusLabel(it.ToStatus), 0))
	}
	if it.VulnClass != "" && it.VulnClass != it.Title() {
		out += fmt.Sprintf("\n**类型**：%s", markdownText(it.VulnClass, 0))
	}
	if a := assetLine(it.Assets, maxAssetsShown); a != "" {
		out += fmt.Sprintf("\n**资产**：%s", markdownText(a, 0))
	}
	if it.Summary != "" {
		if s := markdownText(it.Summary, maxSummaryRunes); s != "" {
			out += fmt.Sprintf("\n**摘要**：%s", s)
		}
	}
	return out
}

// feishuBatchLine 渲染汇总卡片里的一条。
// 한국어 해설: 묶음에서 한 항목의 순번·등급·제목·자산을 짧게 표현한다.
func feishuBatchLine(it Item, index int) string {
	line := fmt.Sprintf("**%d. %s · %s**", index, SeverityLabel(it.Severity), markdownText(it.Title(), 0))
	if a := assetLine(it.Assets, maxAssetsShown); a != "" {
		line += " — " + markdownText(a, 0)
	}
	return line
}
