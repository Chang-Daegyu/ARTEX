package agent

// 한국어 파일 해설: agent/finding_recorder.go
// Agent와 증거 저장소 사이의 의존성을 작게 유지하는 FindingRecorder 인터페이스이다.
// Agent는 RecordFindingInput과 검증할 TrafficRef 목록만 전달하고 HTTP 본문 복사는 수행하지 않는다.
// 실제 구현은 원본 트래픽 확인·증거 스냅샷 보존·DB 원자적 기록을 소유한다.
// Setter는 Worker/Planner/MainAgent/ToolSet에 동일한 기록기를 주입한다.
// 트래픽은 선택 사항으로, 비 HTTP 관찰이나 미수집 상황은 텍스트·명령 증거로 기록할 수 있다.
// 상수의 중국어 문장은 모델에게 전달되는 실행 안내이므로 원본 동작을 위해 유지한다.

import (
	"context"

	"github.com/Autumn-27/artex/db"
)

// FindingRecorder is injected by the host; agents never synthesize or copy
// evidence bodies themselves. Its implementation owns the atomic write.
type FindingRecorder interface {
	Record(context.Context, db.RecordFindingInput, []db.TrafficRef) (*db.RecordedFinding, error)
}

// Tool-use guidance is appended without replacing the user's editable prompt.
// It does not require capture or claim that unavailable traffic tools exist.
const findingTrafficGuidance = "\n\n**漏洞流量证据（可选）**：调用 report_finding 上报漏洞时，如有已查看并确认支持漏洞结论的 HTTP 请求/响应，可用 traffic_refs 按复现顺序绑定真实 ID；域名和时间只作候选筛选，不推定关联。TCP 等非 HTTP 漏洞、未采集或无确切匹配时省略或传 []，在 evidence 保留命令输出、日志等其他可验证证据，建议说明未绑定原因。不要猜测 ID，也不要仅为补包重复探测。"

func (t *ToolSet) SetFindingRecorder(r FindingRecorder)   { t.findingRecorder = r }
func (w *Worker) SetFindingRecorder(r FindingRecorder)    { w.findingRecorder = r }
func (p *Planner) SetFindingRecorder(r FindingRecorder)   { p.findingRecorder = r }
func (m *MainAgent) SetFindingRecorder(r FindingRecorder) { m.findingRecorder = r }
