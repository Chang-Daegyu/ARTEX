// Package notify 实现漏洞发现的 IM / 邮件推送渠道适配层。
//
// 分层：本包是**叶子包**，只依赖标准库。它不认识数据库、不认识 server。渠道配置
// 以 map[string]any 传入（对应 notification_channels.config 这一 JSONB 列），
// 待推送内容以 Message 传入。这样拆开的好处是：签名计算、UTF-8 截断、过滤匹配这些
// 真正容易出错的地方可以脱离 PostgreSQL 单测，宿主只需在 server 侧做编排。
//
// 并发约定：Channel 的实现必须**无状态**。同一个 Channel 实例会被多个渠道配置
// （甚至同一渠道的多个机器人实例）并发复用，所有凭据一律从 cfg 参数传入，
// 不允许把 webhook URL 之类的东西缓存进实现自身的字段。
// [한국어 파일 안내] notify/notify.go
// 발견 알림의 공통 채널 ID·이벤트 ID·심각도/처리 상태 표시 규칙이다.
// notify는 표준 라이브러리만 사용하는 말단 패키지다. DB 이벤트 생성·큐·재시도 스케줄링은 db/server가 맡는다.
// 채널 구현은 무상태로 재사용하며 매 전송의 자격 증명은 cfg로 전달한다.
// 이 파일의 중국어 라벨은 실제 알림 출력이므로 원문을 보존하고 의미를 한국어 주석으로 안내한다.
package notify

// 渠道类型标识。取值同时是 notification_channels.kind 的合法集合，由 server 侧
// 白名单校验（与 findings.status 同理，不用 DB CHECK，方便后续加渠道）。
const (
	KindDingTalk = "dingtalk" // 钉钉自定义机器人
	KindFeishu   = "feishu"   // 飞书(含 Lark)自定义机器人
	KindWeCom    = "wecom"    // 企业微信群机器人
	KindWebhook  = "webhook"  // 通用 Webhook：自定义方法/头/JSON 模板
	KindTelegram = "telegram" // Telegram Bot API
	KindEmail    = "email"    // SMTP 邮件
)

// 事件类型，对应 notification_events.kind。
const (
	EventFindingCreated       = "finding_created"
	EventFindingStatusChanged = "finding_status_changed"
)

// InitKind 是 config 里为空的 kind 的兜底值。
const InitKind = KindDingTalk

// severityRank 把漏洞级别映射成可比较的序数。未知级别返回 0，因此任何
// min_severity 设置都会把未知级别挡在外面——存疑时不推，避免误报刷屏。
var severityRank = map[string]int{
	"low":      1,
	"medium":   2,
	"high":     3,
	"critical": 4,
}

// SeverityRank 返回级别的序数；未知级别返回 0。
// 한국어 해설: low=1부터 critical=4까지 심각도 순서를 숫자로 바꾼다. 알려지지 않은 등급은 map의 0값이다.
func SeverityRank(severity string) int { return severityRank[severity] }

// SeverityLabel 返回带 emoji 的中文级别名，用于消息标题与卡片配色。
// 未知级别原样回显，不臆造。
// 한국어 해설: 알려진 심각도에 이모지와 원본 중국어 라벨을 붙이고 알 수 없는 값은 그대로 표시한다.
func SeverityLabel(severity string) string {
	switch severity {
	case "critical":
		return "🔴 严重"
	case "high":
		return "🟠 高危"
	case "medium":
		return "🟡 中危"
	case "low":
		return "🔵 低危"
	default:
		return severity
	}
}

// StatusLabel 把处置状态翻译成中文，用于状态变更消息。
// 한국어 해설: pending·fixed·false_positive 등 처리 상태 식별자를 원본의 읽기 쉬운 라벨로 바꾼다.
func StatusLabel(status string) string {
	switch status {
	case "pending":
		return "待处理"
	case "in_progress":
		return "处理中"
	case "confirmed":
		return "已确认"
	case "resolved":
		return "已处理"
	case "fixed":
		return "已修复"
	case "false_positive":
		return "误报"
	case "ignored":
		return "忽略"
	case "duplicate":
		return "重复"
	case "risk_accepted":
		return "风险接受"
	default:
		return status
	}
}

// AtLeast 判断 severity 是否达到 min 门槛。min 为空表示不设门槛，一律通过。
// 注意未知 severity 的序数为 0，会被任何非空 min 拒掉（见 severityRank 注释）。
// 한국어 해설: 최소 등급이 비어 있으면 통과시키고, 아니면 숫자 순위를 비교한다. 잘못된 min 값의 거절은 저장 시 Validate가 맡는다.
func AtLeast(severity, min string) bool {
	if min == "" {
		return true
	}
	return SeverityRank(severity) >= SeverityRank(min)
}
