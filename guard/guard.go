// Package guard implements the safety boundary layer (docs §11): an audit log,
// user-configured intercept-rule evaluation, and Observer/G5 failure attribution.
// Every tool call passes through the PreToolUse hook before executing.
// (The RoE authorization-scope mechanism was removed; a replacement may be added
// later.) Destructive/exfil gating is no longer hard-coded here — it lives in the
// DB intercept rules (seeded as ordinary [内置] rules, so users can disable or
// delete them), evaluated via applyIntercept.
// [한국어 파일 안내] guard/guard.go
// Norma의 도구 실행 전·후 hook을 ARTEX 승인 규칙 및 감사 기록에 연결한다.
// 실제로 Hooks()가 세션 옵션에 연결된 경로에서만 동작하므로, 이 패키지 존재만으로 모든 역할·모든 네트워크 호출이 통제되지는 않는다.
// 인터셉터가 없거나 해당 도구가 비활성 상태이고 별도 거부가 없으면 기본 동작은 통과다.
// 원본의 안전 경계라는 표현은 hook 수준 책임을 가리키며 운영체제 격리나 강제 네트워크 scope를 구현한 것은 아니다.
package guard

import (
	"context"
	"encoding/json"
	"regexp"
	"sync"
	"time"

	"github.com/Autumn-27/artex/intercept"
	"github.com/Autumn-27/norma/hook"
)

// AuditEntry records one gated tool call.
// 한국어 자료형: 도구·시각·허용/차단·이유를 담는 메모리 감사 항목이다. 대상 시스템의 실제 결과 로그와 구분한다.
type AuditEntry struct {
	TS      int64  `json:"ts"`
	Tool    string `json:"tool"`
	Action  string `json:"action"` // allow|block
	Reason  string `json:"reason,omitempty"`
	Command string `json:"command,omitempty"`
}

// Guard enforces the side-effect policy via agent-core hooks.
// 한국어 자료형: hook 레지스트리, 선택적 인터셉터, 메모리 감사와 Bash 결과 통계의 동시 접근을 관리한다.
type Guard struct {
	mu          sync.Mutex
	audit       []AuditEntry
	attrib      map[string]int // failure attribution counts (Observer / G5)
	reg         *hook.Registry
	interceptor *intercept.Interceptor // optional; nil disables user-configured rules
}

// New creates a Guard without user-configured intercept rules (used for pentest
// tasks where the Interceptor is not yet available).
// 한국어 해설: 인터셉터 없이 감사 기록 및 결과 분류 hook만 만든다. 이 생성 경로 자체에 고정된 위험 명령 차단 목록은 없다.
func New() *Guard { return newGuard(nil) }

// NewWithInterceptor creates a Guard with user-configured intercept rules.
// 한국어 해설: DB 규칙과 선택적 모델 판정을 사용할 수 있도록 Interceptor를 주입한다.
func NewWithInterceptor(ic *intercept.Interceptor) *Guard { return newGuard(ic) }

// 한국어 해설: 실행 전 승인 처리와 실행 후 결과 분류를 하나의 hook.Registry에 등록한다.
func newGuard(ic *intercept.Interceptor) *Guard {
	g := &Guard{attrib: map[string]int{}, interceptor: ic}
	g.reg = hook.NewRegistry().
		On(hook.PreToolUse, g.preToolUse).
		On(hook.PostToolUse, g.postToolUse)
	return g
}

// Hooks returns the hook registry to attach to an agent session.
// 한국어 해설: 호출자가 Norma 세션에 명시적으로 붙여야 할 hook 레지스트리를 반환한다.
func (g *Guard) Hooks() *hook.Registry { return g.reg }

// 한국어 해설: Bash와 대화형 셸 입력에서 명령 표면을 뽑아 감사 목록에 먼저 남긴 뒤 규칙 판정을 요청한다.
// 초기 allow 기록은 실행 성공을 뜻하지 않으며 이후 차단 기록이 추가될 수 있다.
func (g *Guard) preToolUse(ctx context.Context, ev hook.Event) hook.Result {
	// Extract the shell-command surface for the audit log: Bash + the interactive-shell
	// tools (shell_open's command, shell_send's text). Destructive/exfil gating is no
	// longer hard-coded here — it now lives in the DB intercept rules, evaluated by
	// applyIntercept below. Other tools record an empty command.
	var cmd string
	switch ev.ToolName {
	case "Bash", "shell_open":
		var in struct {
			Command string `json:"command"`
		}
		_ = json.Unmarshal(ev.Input, &in)
		cmd = in.Command
	case "shell_send":
		var in struct {
			Text string `json:"text"`
		}
		_ = json.Unmarshal(ev.Input, &in)
		cmd = in.Text
	}
	g.record(ev.ToolName, "allow", "", cmd)
	return g.applyIntercept(ctx, ev)
}

// applyIntercept evaluates user-configured intercept rules against the tool call.
// Both rules and the fallback judge receive the complete tool input.
// 한국어 해설: 활성 도구의 우선순위 규칙을 먼저 평가하고 미일치일 때만 선택적 Judge를 호출한다.
// deny는 즉시 차단, allow는 기록 후 통과, ask는 승인 결과까지 기다린다. 취소된 작업의 신규 대기는 만들지 않는다.
func (g *Guard) applyIntercept(ctx context.Context, ev hook.Event) hook.Result {
	if g.interceptor == nil {
		return hook.Result{}
	}
	if !g.interceptor.IsToolEnabled(ev.ToolName) {
		return hook.Result{}
	}
	ctx = intercept.WithCall(ctx, ev.ToolName, ev.Input)
	dec, matched := g.interceptor.Match(ev.ToolName, ev.Input)
	if !matched {
		// No rule matched. Ask the LLM fallback judge (if enabled); when it is off
		// or unwired, keep current behavior and allow.
		d, judged := g.interceptor.Judge(ctx, ev.ToolName, ev.Input)
		if !judged {
			return hook.Result{}
		}
		dec = d
	}
	switch dec.Action {
	case "deny":
		// 观测:deny 命中不阻塞审批,直接记一条 denied（历史/任务拦截页可见）。
		g.interceptor.Log(ctx, intercept.ConvIDFromContext(ctx), dec, ev.ToolName, ev.Input, "denied")
		return g.block(ev.ToolName, systemBlockMessage(dec.Message), "")
	case "allow":
		// Record explicit rule and model approvals so review details remain auditable.
		g.interceptor.Log(ctx, intercept.ConvIDFromContext(ctx), dec, ev.ToolName, ev.Input, "allowed")
		return hook.Result{}
	case "ask":
		// If the worker context is already cancelled (task stopped / killed), block
		// immediately without creating a pending record — avoids orphaned DB entries
		// and makes execOne complete fast, reducing the race against drainSynthetic.
		if ctx.Err() != nil {
			return g.block(ev.ToolName, systemBlockMessage("工作已取消，平台安全管控阻止执行"), "")
		}
		convID := intercept.ConvIDFromContext(ctx)
		if !g.interceptor.HandleAsk(ctx, convID, dec, ev.ToolName, ev.Input) {
			return g.block(ev.ToolName, systemBlockMessage("人工审批未通过（用户拒绝或审批超时）"), "")
		}
		return hook.Result{}
	}
	return hook.Result{}
}

// systemBlockMessage frames an intercept block as an ARTEX platform-governance
// decision so the agent does not mistake it for a target-side defense.
//
// The bare reasons ("禁止执行此工具" / "用户拒绝") read exactly like a WAF/403 on
// the target, so a pentest agent's instinct is to bypass them — rewrite the
// command, swap the payload, re-encode, retry. That is both futile (the platform
// blocks the class of action, not one string) and wrong (it's a policy decision,
// not an obstacle to defeat). This prefix states plainly that the block comes
// from the platform, is not the target's protection, and that the operation is
// forbidden — so the agent pivots to another approach instead of evading it.
// Audit/history rows keep the raw reason (see Interceptor.Log); only the
// model-facing tool_result carries this framing.
// 한국어 해설: 차단 이유가 대상 서버의 방어가 아니라 ARTEX 정책 결정임을 모델 결과에 명시한다. 원문 문자열은 프롬프트 동작을 보존하기 위해 유지한다.
func systemBlockMessage(reason string) string {
	return "【ARTEX 平台管控·非目标防御】此调用被平台拦截。" +
		"原因：" + reason + "。此操作被禁止。"
}

var reBlocked = regexp.MustCompile(`(?i)\b(403|forbidden|waf|blocked|rate.?limit|429|captcha|denied)\b`)

// postToolUse is the Observer failure-attribution hook (G5): it classifies tool
// results into blocked / error / ok so the planner can change strategy instead
// of giving up at a WAF.
// 한국어 해설: Bash 결과 문자열의 차단 단서와 IsError로 blocked/error/ok를 집계한다. 실제 공격 성공 여부를 판단하는 검증기는 아니다.
func (g *Guard) postToolUse(_ context.Context, ev hook.Event) hook.Result {
	if ev.ToolName != "Bash" {
		return hook.Result{}
	}
	class := "ok"
	switch {
	case reBlocked.Match(ev.Result):
		class = "blocked"
	case ev.IsError:
		class = "error"
	}
	g.mu.Lock()
	g.attrib[class]++
	g.mu.Unlock()
	return hook.Result{}
}

// Attributions returns failure-attribution counts (Observer / G5).
// 한국어 해설: mutex 아래 분류 통계의 복사본을 반환하여 호출자가 내부 map을 바꾸지 못하게 한다.
func (g *Guard) Attributions() map[string]int {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make(map[string]int, len(g.attrib))
	for k, v := range g.attrib {
		out[k] = v
	}
	return out
}

// 한국어 해설: 차단 감사를 추가하고 Norma가 실행을 막을 수 있는 hook.Result를 반환한다.
func (g *Guard) block(tool, reason, cmd string) hook.Result {
	g.record(tool, "block", reason, cmd)
	return hook.Result{Decision: "block", Message: reason}
}

// 한국어 해설: 도구별 감사 이벤트를 최근 2,000건까지 메모리에 보관한다. DB 영속 승인 이력과 별개의 짧은 버퍼다.
func (g *Guard) record(tool, action, reason, cmd string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.audit = append(g.audit, AuditEntry{TS: time.Now().Unix(), Tool: tool, Action: action, Reason: reason, Command: cmd})
	if len(g.audit) > 2000 {
		g.audit = g.audit[len(g.audit)-2000:]
	}
}

// Audit returns a snapshot of recent gated calls (most recent last).
// 한국어 해설: 최근 감사 항목을 오래된 것부터 정렬된 현재 순서의 복사본으로 반환한다.
func (g *Guard) Audit() []AuditEntry {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]AuditEntry, len(g.audit))
	copy(out, g.audit)
	return out
}
