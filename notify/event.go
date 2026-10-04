// [한국어 파일 안내] notify/event.go
// DB 이벤트 스냅샷과 채널 렌더링용 메시지의 데이터 계약을 정의한다.
// 발견이 나중에 수정되어도 당시 알림의 판단을 보존하도록 Snapshot에는 발생 시점 필드를 복사한다.
// 사람이 읽는 자산명·상세 링크는 server가 Item에 채우며 notify 자체는 DB를 조회하지 않는다.
package notify

// Snapshot 是 notification_events.snapshot 这一 JSONB 列的契约。写方是 db 层的
// 漏洞落库事务，读方是 server 层的投递引擎与过滤匹配。定义放在本包是因为它是
// 「通知领域」的载荷：db 只负责序列化，不理解字段含义。
//
// 为什么冗余存漏洞字段而不在渲染时回查：漏洞事后会被改名、改级别、改状态，
// 而推送内容应当反映**事发当时**的结论——回查会得到「事后被改成 low」的
// 危险误导。另外 fan-out 与渲染因此不必 JOIN findings/tasks/assets 三张表。
// 한국어 자료형: notification_events.snapshot JSONB에 저장하는 발생 시점의 발견 정보다. 실시간 findings 재조회 결과와 구분한다.
type Snapshot struct {
	// 事件类型：finding_created / finding_status_changed
	Kind      string  `json:"kind"`
	FindingID int64   `json:"finding_id"`
	TaskID    int64   `json:"task_id"`
	VulnClass string  `json:"vulnclass"`
	Name      string  `json:"name"`
	Severity  string  `json:"severity"`
	Summary   string  `json:"summary"`
	AssetIDs  []int64 `json:"asset_ids"`
	// 仅 kind=finding_status_changed 时非空。
	FromStatus string `json:"from_status,omitempty"`
	ToStatus   string `json:"to_status,omitempty"`
}

// Item 是一条待推送的漏洞，供渠道渲染。
// 한국어 자료형: 채널이 렌더링할 한 개 발견/상태 변경 항목이며 자산명과 상세 링크를 이미 포함한다.
type Item struct {
	FindingID int64
	Name      string
	VulnClass string
	Severity  string
	Summary   string
	// Assets 是解析后的资产展示名（如域名/IP）。由 server 层填充——
	// 本包不碰数据库，拿不到名字。
	Assets []string
	// DetailURL 是漏洞详情回链；为空表示未配 public_base_url，渲染时省略。
	DetailURL string
	// 状态变更事件专用；两项均非空时渲染成「待处理 → 已修复」。
	FromStatus string
	ToStatus   string
}

// IsStatusChange 报告该条目是否为状态变更事件。
// 한국어 해설: 이전 또는 이후 상태가 있으면 상태 변경 항목으로 본다. 두 필드가 모두 있어야 한다는 검증 함수는 아니다.
func (i Item) IsStatusChange() bool { return i.FromStatus != "" || i.ToStatus != "" }

// Title 返回条目的展示标题：优先人工命名的 name，回退漏洞类型 vulnclass，
// 两者都空时用一个占位符——绝不输出空标题。
// 한국어 해설: 사용자 이름을 우선하고 취약점 분류와 기본 제목으로 차례로 대체하여 빈 제목을 피한다.
func (i Item) Title() string {
	if i.Name != "" {
		return i.Name
	}
	if i.VulnClass != "" {
		return i.VulnClass
	}
	return "(未命名漏洞)"
}

// Message 是一次渠道发送的完整内容。
// 한국어 자료형: 단일/묶음 전송의 항목 목록·명시적 시간창·플랫폼 링크를 묶는다. 시간창을 다시 계산하지 않아 렌더링이 결정적이다.
type Message struct {
	// 单条推送时长度为 1；汇总推送（digest）时为一整批。
	// 空切片是非法的，调用方须保证至少一条。
	Items []Item
	// Batch=true 时按汇总消息渲染（换标题、带上时间窗与条数）。
	Batch bool
	// WindowMinutes 是汇总周期（分钟），仅 Batch=true 时用于文案「近 N 分钟」。
	// 刻意由配置显式传入而不是渲染时算 time.Since：渲染保持确定性，才好测。
	WindowMinutes int
	// HomeURL 是平台面板地址（全局 public_base_url）；空则不带面板入口。
	HomeURL string
}
