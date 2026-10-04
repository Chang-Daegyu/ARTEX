// [한국어 파일 안내] intercept/trace.go
// 도구 시작·승인·완료 이벤트를 같은 실행 ID에 연결하여 감사의 출처를 보존한다.
// SDK hook에서 도구 ID를 직접 받지 못하는 경우 이름과 정규화한 인자 해시를 함께 사용한다.
// 동시에 동일한 호출이 여럿이면 FIFO로 추측하지 않고 ambiguous로 남긴다. 결과 누락도 성공으로 간주하지 않는다.
// 감사용 짧은 대화 스냅샷은 review_context.go의 심사기 입력과 다른 데이터다.
package intercept

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Autumn-27/artex/db"
)

const (
	contextLimit = 24
	entryLimit   = 8 * 1024
	promptLimit  = 32 * 1024
	outputLimit  = 64 * 1024
)

type traceKey struct{}
type callKey struct{}
type completion func(status, output string, truncated bool)

type tracedCall struct {
	key       string
	audit     db.InterceptAudit
	claimed   bool
	ambiguous bool
	complete  completion
}

// Trace belongs to ONE Prompt invocation. SDK v0.3.6 hooks omit the tool ID;
// correlate only when exactly one outstanding event has matching input. Never
// guess between simultaneous identical requests, even when results arrive FIFO.
// 한국어 자료형: 한 번의 모델 실행에 속한 도구 이벤트와 문맥을 관리하는 메모리 객체다. 실행 간 ID를 공유하지 않는다.
type Trace struct {
	mu      sync.Mutex
	runID   string
	user    string
	userCut bool
	entries []db.InterceptContextEntry
	cut     bool
	calls   map[string]*tracedCall
}

// 한국어 해설: 한 번의 Prompt 호출에 해당하는 runID와 제한된 이전 감사 문맥을 만들고 context에 넣는다.
func WithTrace(ctx context.Context, user string, prior []db.InterceptContextEntry) (context.Context, *Trace) {
	t := &Trace{runID: rand.Text(), calls: make(map[string]*tracedCall)}
	t.user, t.userCut = bounded(user, promptLimit)
	for _, e := range prior {
		t.append(e)
	}
	return context.WithValue(ctx, traceKey{}, t), t
}

// 한국어 해설: 항목 본문을 8 KiB, 문맥 목록을 최근 24개로 제한하고 잘림 여부를 함께 누적한다. 내부 호출자는 잠금을 관리한다.
func (t *Trace) append(e db.InterceptContextEntry) {
	var cut bool
	e.Text, cut = bounded(e.Text, entryLimit)
	e.Truncated = e.Truncated || cut
	t.cut = t.cut || e.Truncated
	t.entries = append(t.entries, e)
	if len(t.entries) > contextLimit {
		t.entries = append([]db.InterceptContextEntry(nil), t.entries[len(t.entries)-contextLimit:]...)
		t.cut = true
	}
}

// 한국어 해설: 외부에서 문맥 항목을 추가할 때 mutex를 잡아 병렬 도구 이벤트의 쓰기를 직렬화한다.
func (t *Trace) Append(e db.InterceptContextEntry) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.append(e)
}

// 한국어 해설: 도구 ID에 실행 직전 문맥 스냅샷과 입력 해시를 연결한 뒤 tool_use 항목을 문맥에 추가한다.
func (t *Trace) Start(id, tool string, input []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.calls[id] = &tracedCall{key: tool + ":" + digestInput(input), audit: db.InterceptAudit{
		RunID: t.runID, ToolUseID: id, Correlation: "exact", InputDigest: digestInput(input),
		UserMessage: t.user, UserTruncated: t.userCut, CapturedAt: time.Now().UTC(),
		Context: append([]db.InterceptContextEntry{}, t.entries...), ContextTruncated: t.cut,
	}}
	t.append(db.InterceptContextEntry{Kind: "tool_use", Tool: tool, ToolUseID: id, Text: string(input)})
}

// WithCall claims the event before model review starts, so subsequent tools
// cannot change this approval's context while the judge is running.
// 한국어 해설: 아직 연결되지 않은 동일 이름·인자 후보가 정확히 하나일 때만 승인과 도구 ID를 결합한다. 여러 후보는 모두 모호함을 유지한다.
func WithCall(ctx context.Context, tool string, input []byte) context.Context {
	t, _ := ctx.Value(traceKey{}).(*Trace)
	a := db.InterceptAudit{Correlation: "unavailable", InputDigest: digestInput(input), CapturedAt: time.Now().UTC()}
	if t != nil {
		t.mu.Lock()
		var candidates []*tracedCall
		for _, c := range t.calls {
			if !c.claimed && c.key == tool+":"+a.InputDigest {
				candidates = append(candidates, c)
			}
		}
		if len(candidates) == 1 && !candidates[0].ambiguous {
			candidates[0].claimed = true
			a = candidates[0].audit
		} else {
			a.RunID, a.UserMessage, a.UserTruncated = t.runID, t.user, t.userCut
			if len(candidates) > 0 {
				a.Correlation = "ambiguous"
				for _, c := range candidates {
					c.ambiguous = true
				}
			}
		}
		t.mu.Unlock()
	}
	return context.WithValue(ctx, callKey{}, a)
}

// 한국어 해설: 정확한 도구 ID에 결과 저장 콜백을 등록한다.
func (t *Trace) bind(id string, f completion) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if c := t.calls[id]; c != nil {
		c.complete = f
	}
}

// 한국어 해설: 완료 도구를 map에서 제거하고 결과를 64 KiB로 제한하여 succeeded/failed 상태를 저장한다. 다른 ID의 결과는 붙이지 않는다.
func (t *Trace) Complete(id, output string, isError bool) {
	t.mu.Lock()
	c := t.calls[id]
	delete(t.calls, id)
	t.mu.Unlock()
	if c == nil || c.complete == nil {
		return
	}
	status := "succeeded"
	if isError {
		status = "failed"
	}
	out, cut := bounded(output, outputLimit)
	c.complete(status, out, cut)
}

// Finish marks missing results unknown, never successful. The tool may have
// been interrupted or its final event lost; this is distinct from a tool error.
// 한국어 해설: 호출은 시작했지만 결과가 도착하지 않은 연결들을 unknown으로 닫는다. 중단·이벤트 누락과 명시적 실패를 구분한다.
func (t *Trace) Finish() {
	t.mu.Lock()
	calls := t.calls
	t.calls = make(map[string]*tracedCall)
	t.mu.Unlock()
	for _, c := range calls {
		if c.complete != nil {
			c.complete("unknown", "执行结束但未收到工具结果", false)
		}
	}
}

// 한국어 해설: 규칙/모델 출처와 실제 적용 행동을 감사 구조체로 합친다. 자동 허용은 큰 원시 문맥을 버리지만 실제 모델 입력은 보존한다.
func auditFor(ctx context.Context, dec Decision, input []byte, status string) *db.InterceptAudit {
	a, ok := ctx.Value(callKey{}).(db.InterceptAudit)
	if !ok {
		a = db.InterceptAudit{Correlation: "unavailable", InputDigest: digestInput(input), CapturedAt: time.Now().UTC()}
	}
	a.InitialAction, a.InitialReason = dec.Action, dec.Message
	a.ModelFallback = dec.ModelFallback
	a.ModelInput, a.ModelInputDigest = dec.ModelInput, dec.ModelInputDigest
	a.RuleName, a.ConfigDigest, a.ProfileID = dec.RuleName, dec.ConfigDigest, dec.ProfileID
	a.ExecutionStatus = "not_started"
	if status == "allowed" {
		a.EffectiveAction, a.ExecutionStatus = "allow", "awaiting_result"
		if a.Correlation != "exact" {
			a.ExecutionStatus = "unknown"
		}
		// Keep the exact model input for EVERY model verdict, including automatic
		// allows. Raw audit history is not model input. Preserve the existing
		// lightweight allow-retention policy; render the saved input directly.
		a.UserMessage, a.UserTruncated = "", false
		a.Context, a.ContextTruncated = nil, false
	}
	if status == "denied" {
		a.EffectiveAction, a.ExecutionStatus = "deny", "not_executed"
	}
	return &a
}

// 한국어 해설: 정확히 연결된 감사 항목에만 DB 완료 저장을 등록한다. 연결이 모호하면 잘못된 실행 결과를 덮어쓰지 않는다.
func (i *Interceptor) bindResult(ctx context.Context, id int64, audit *db.InterceptAudit) {
	t, _ := ctx.Value(traceKey{}).(*Trace)
	if t == nil || audit.Correlation != "exact" || audit.ToolUseID == "" {
		return
	}
	t.bind(audit.ToolUseID, func(status, output string, cut bool) {
		_ = i.db.CompleteIntercept(id, audit.RunID, audit.ToolUseID, status, output, cut)
	})
}

// 한국어 해설: JSON 숫자를 float로 바꾸지 않고 정규화한 뒤 SHA-256을 계산한다. 키 순서·공백 차이를 동일 입력으로 취급한다.
func digestInput(input []byte) string {
	var value any
	d := json.NewDecoder(bytes.NewReader(input))
	d.UseNumber()
	if d.Decode(&value) == nil {
		if canonical, err := json.Marshal(value); err == nil {
			input = canonical
		}
	}
	h := sha256.Sum256(input)
	return hex.EncodeToString(h[:])
}

// 한국어 해설: 바이트 한도까지 자르되 UTF-8 문자 중간을 피하고 잘림 여부를 함께 반환한다.
func bounded(s string, limit int) (string, bool) {
	if len(s) <= limit {
		return s, false
	}
	end := limit
	for end > 0 && !utf8.RuneStart(s[end]) {
		end--
	}
	return s[:end], true
}
