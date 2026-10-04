// [한국어 파일 안내] notify/webhook.go
// 사용자가 정한 URL·HTTP 메서드·헤더·JSON 템플릿으로 외부 수신 시스템을 연결하는 범용 채널이다.
// 템플릿에는 데이터 전용 구조체와 json/jsons만 노출하며 파일/네트워크/셸 실행 함수를 제공하지 않는다.
// GET은 본문 없는 신호로 보내고 나머지 지원 메서드는 렌더링 결과가 유효한 JSON인지 확인한다.
// 전송 성공이 수신 시스템의 업무 처리를 검증하는 것은 아니며 채널은 HTTP 수락 결과를 반환한다.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"text/template"
	"time"
)

// webhookChannel 是通用 Webhook 适配器：用户自定 URL、方法、请求头与 JSON 模板。
// 它的存在让本功能不必为 Slack / Mattermost / Discord / 自建系统各写一个实现——
// 那些平台都能被一个可配模板覆盖。
// 한국어 자료형: 범용 JSON HTTP 전송 채널의 무상태 구현이다.
type webhookChannel struct{}

// 한국어 해설: 범용 Webhook의 고정 채널 ID를 반환한다. DB 설정 및 registry 키와 일치해야 한다.
func (webhookChannel) Kind() string { return KindWebhook }

// 通用 Webhook 没有官方限制，返回 0 表示默认不限流，由使用者按对端能力自定。
// 한국어 해설: 새 채널 설정에 사용할 기본값을 분당 0로 제안한다. 0은 알려진 제한을 두지 않는다는 내부 계약이다.
func (webhookChannel) DefaultRatePerMin() int { return 0 }

// 掩码 url 与 headers：目标地址本身常带 token，自定义头里通常放着鉴权凭据，
// 两者都会出现在接口回显里，所以都要挡。
// 代价是编辑时若想改动其中一个头，需要重新填整组头（掩码值会被解释为「保持原值」）——
// 这个取舍是刻意的：宁可多填一次，也不把凭据回显到浏览器。
// 한국어 해설: API에서 마스킹할 비밀 필드를 url과 headers 전체로 선언한다. 실제 저장값 암호화 기능은 아니다.
func (webhookChannel) SecretKeys() []string { return []string{"url", "headers"} }

// 目的地是 url。改 url 时必须重新表态 headers —— 否则原始 Authorization 头
// 会被原样发到新地址，这正是掩码绕过的主路径。
// 한국어 해설: 자격 증명 수신 위치를 결정하는 url를 선언한다. 이 값이 바뀌면 기존 비밀의 묵시적 재사용을 막는다.
func (webhookChannel) DestinationKeys() []string { return []string{"url"} }

// webhookDefaultTemplate 是未填模板时的兜底请求体：一个直白的 JSON 结构，
// 覆盖绝大多数「收一条 JSON 入库」的自建接收端。
const webhookDefaultTemplate = `{
  "title": {{json .Title}},
  "batch": {{.Batch}},
  "count": {{.Count}},
  "items": [
{{- range $i, $it := .Items}}
{{- if $i}},{{end}}
    {
      "finding_id": {{$it.FindingID}},
      "name": {{json $it.Name}},
      "vulnclass": {{json $it.VulnClass}},
      "severity": {{json $it.Severity}},
      "summary": {{json $it.Summary}},
      "assets": {{json $it.Assets}},
      "detail_url": {{json $it.DetailURL}}
    }
{{- end}}
  ]
}`

// webhookTemplateData 是暴露给用户模板的上下文。
// 한국어 자료형: 템플릿이 읽을 수 있는 명시적 공개 데이터다. 메서드를 추가하면 템플릿 호출 능력도 늘 수 있으므로 경계를 보존한다.
type webhookTemplateData struct {
	Title   string
	Batch   bool
	Count   int
	Items   []webhookItem
	HomeURL string
	// SentAt 是本次投递时间（RFC3339），供接收端记录。
	SentAt string
}

// 한국어 자료형: 템플릿에 노출할 발견 한 건의 순수 데이터 복사본이다.
type webhookItem struct {
	FindingID     int64
	Name          string
	VulnClass     string
	Severity      string
	SeverityLabel string
	Summary       string
	Assets        []string
	DetailURL     string
	FromStatus    string
	ToStatus      string
	// StatusLabel 是状态变更的可读描述，如「待处理 → 已修复」；非状态变更时为空。
	StatusLabel string
}

// 한국어 해설: URL, 허용 메서드 GET/POST/PUT/PATCH, 사용자 템플릿 문법을 저장 전에 검사한다.
func (webhookChannel) Validate(cfg map[string]any) error {
	raw := cfgString(cfg, "url")
	if raw == "" {
		return errors.New("缺少目标 URL")
	}
	if err := validateHTTPURL(raw); err != nil {
		return fmt.Errorf("目标 URL 无效: %w", err)
	}
	if m := strings.ToUpper(cfgString(cfg, "method")); m != "" && m != http.MethodGet && m != http.MethodPost && m != http.MethodPut && m != http.MethodPatch {
		return fmt.Errorf("不支持的方法 %s（可用 GET/POST/PUT/PATCH）", m)
	}
	if tpl := cfgString(cfg, "body_template"); tpl != "" {
		if _, err := parseWebhookTemplate(tpl); err != nil {
			return fmt.Errorf("请求体模板语法错误: %w", err)
		}
	}
	return nil
}

// 한국어 해설: 설정을 다시 확인하고 GET 외에는 템플릿 결과를 JSON RawMessage로 보내 이중 인코딩을 피한다. 성공 시 전체 항목 수를 반환한다.
func (c webhookChannel) Send(ctx context.Context, cfg map[string]any, m Message) (int, error) {
	if err := c.Validate(cfg); err != nil {
		return 0, Permanent(err)
	}
	method := strings.ToUpper(cfgString(cfg, "method"))
	if method == "" {
		method = http.MethodPost
	}

	// GET 不带请求体：把内容塞进 query 超出模板能力范围，也不符合 GET 语义，
	// 所以 GET 只适合「命中即触发钩子」这类接收端。
	var payload any
	if method != http.MethodGet {
		body, err := renderWebhookBody(cfgString(cfg, "body_template"), m)
		if err != nil {
			return 0, Permanent(err)
		}
		// 模板渲染出的是字符串形式的 JSON，这里转成 json.RawMessage 原样发出，
		// 避免二次转义把用户精心构造的结构套进一个 JSON 字符串里。
		if !json.Valid([]byte(body)) {
			return 0, Permanent(errors.New("请求体模板渲染结果不是合法 JSON"))
		}
		payload = json.RawMessage(body)
	}

	headers := cfgMap(cfg, "headers")
	if ct := cfgString(cfg, "content_type"); ct != "" {
		// 允许覆盖，但放在 headers 之后应用，保证显式配置优先。
		if headers == nil {
			headers = map[string]string{}
		}
		headers["Content-Type"] = ct
	}
	if _, err := doJSON(ctx, method, cfgString(cfg, "url"), headers, payload); err != nil {
		return 0, err
	}
	// 通用 Webhook 不截断正文（接收端是用户自己的服务，体积由 body_template 决定），
	// 因此整批都算送达。
	return len(m.Items), nil
}

// renderWebhookBody 用用户模板（或默认模板）渲染请求体。
// 한국어 해설: 빈 설정이면 기본 템플릿을 선택하고 제한된 데이터 문맥으로 요청 본문을 렌더링한다.
func renderWebhookBody(tpl string, m Message) (string, error) {
	if strings.TrimSpace(tpl) == "" {
		tpl = webhookDefaultTemplate
	}
	t, err := parseWebhookTemplate(tpl)
	if err != nil {
		return "", fmt.Errorf("请求体模板语法错误: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, newWebhookTemplateData(m)); err != nil {
		return "", fmt.Errorf("渲染请求体模板失败: %w", err)
	}
	return buf.String(), nil
}

// parseWebhookTemplate 解析模板。
//
// missingkey=zero 让缺失的 map 键渲染成零值而不是报错——但本文件的上下文是结构体，
// 主要作用是让 .Items 为空时 range 不出错。真正需要防的是 .Items 为 nil。
// 한국어 해설: 지정된 작은 함수 집합으로 text/template를 파싱한다. 알려지지 않은 구조체 필드 접근은 실행 단계에서 실패할 수 있다.
func parseWebhookTemplate(tpl string) (*template.Template, error) {
	return template.New("body").Funcs(webhookTemplateFuncs).Option("missingkey=zero").Parse(tpl)
}

// webhookTemplateFuncs 是暴露给模板的辅助函数。
var webhookTemplateFuncs = template.FuncMap{
	// json 把任意值序列化成 JSON。
	//
	// 这个函数不是锦上添花而是必需的：省去它，用户只能写 {{.Title}} 直接插值，
	// 而漏洞标题里只要有引号或换行，整段请求体就不再是合法 JSON——接收端会
	// 拒收，且报错信息指向「JSON 解析失败」，完全联想不到是标题里有个引号。
	"json": func(v any) (string, error) {
		raw, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(raw), nil
	},
	// jsons 用于把 JSON 片段嵌进另一段 JSON 字符串值内部（做一层字符串转义）。
	"jsons": func(v any) (string, error) {
		raw, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		quoted, err := json.Marshal(string(raw))
		if err != nil {
			return "", err
		}
		// 去掉外层引号：调用方自己决定要不要加引号。
		return string(quoted[1 : len(quoted)-1]), nil
	},
}

// 한국어 해설: Message를 메서드 없는 템플릿 DTO로 복사하고 현재 발송 시각·등급 라벨·상태 변경 설명을 채운다.
func newWebhookTemplateData(m Message) webhookTemplateData {
	d := webhookTemplateData{
		Title:   markdownTitle(m),
		Batch:   m.Batch,
		Count:   len(m.Items),
		HomeURL: m.HomeURL,
		SentAt:  time.Now().Format(time.RFC3339),
		Items:   make([]webhookItem, 0, len(m.Items)),
	}
	for _, it := range m.Items {
		wi := webhookItem{
			FindingID:     it.FindingID,
			Name:          it.Name,
			VulnClass:     it.VulnClass,
			Severity:      it.Severity,
			SeverityLabel: SeverityLabel(it.Severity),
			Summary:       it.Summary,
			Assets:        append([]string{}, it.Assets...),
			DetailURL:     it.DetailURL,
			FromStatus:    it.FromStatus,
			ToStatus:      it.ToStatus,
		}
		if it.IsStatusChange() {
			wi.StatusLabel = StatusLabel(it.FromStatus) + " → " + StatusLabel(it.ToStatus)
		}
		d.Items = append(d.Items, wi)
	}
	return d
}
