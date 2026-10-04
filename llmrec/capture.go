package llmrec

// 한국어 파일 해설: llmrec/capture.go
// 한 논리적 모델 요청에서 발생한 실제 HTTP 본문을 별도로 수집한다.
// 상위 Recorder가 context에 Capture를 넣으면 agent.quotaAwareTransport가 요청과 응답을 채운다.
// 원본 요청은 첫 값을 보존하고 연결 재시도의 각 응답은 status와 함께 차례로 기록한다.
// TeeResponse는 응답 전체를 먼저 읽지 않고 실제 소비되는 바이트를 복제해 SSE 흐름을 유지한다.
// 동시 읽기·쓰기에는 mutex를 사용하고 캡처가 꺼진 nil 수신자는 안전하게 아무 작업도 하지 않는다.
// 복제 대상은 원본 바이트이며 비밀정보를 자동 마스킹하거나 암호화하는 계층이 아니다.

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
)

type captureContextKey struct{}

// Capture collects the untouched wire bodies of one logical LLM call. The
// Recorder creates it and puts it on the context; the HTTP transport that norma
// dials through (agent.quotaAwareTransport) finds it there and fills it in.
//
// This exists because everything the Recorder itself sees is already normalized:
// llm.CompletionRequest is re-serialized rather than the body buildBody() sent,
// and the response arrives as decoded StreamEvents, not the SSE frames. For
// debugging a live provider, the bytes on the wire are the only ground truth.
//
// norma's doStream retries the request-establishment phase, so one Stream can
// issue several HTTP attempts. Each attempt is kept: the discarded ones (see
// norma/llm/retry.go, which closes non-final bodies unread) are exactly what
// makes rate-limit and gateway failures diagnosable.
//
// The transport writes from norma's stream-reading goroutine while the Recorder
// snapshots at stream end, so all state is mutex-guarded.
type Capture struct {
	mu       sync.Mutex
	request  string
	attempts []*attempt
}

type attempt struct {
	status int
	body   strings.Builder
}

// NewCapture returns a context carrying a fresh Capture, plus the Capture itself.
func NewCapture(ctx context.Context) (context.Context, *Capture) {
	c := &Capture{}
	return context.WithValue(ctx, captureContextKey{}, c), c
}

// CaptureFrom returns the Capture attached to ctx, or nil when raw capture is
// off. Callers must tolerate nil — recording is a toggle, and non-recorded
// providers dial through the same transport.
func CaptureFrom(ctx context.Context) *Capture {
	if ctx == nil {
		return nil
	}
	c, _ := ctx.Value(captureContextKey{}).(*Capture)
	return c
}

// SetRequest stores the outgoing request body. Retries re-send identical bytes,
// so only the first attempt's body is kept.
func (c *Capture) SetRequest(body string) {
	if c == nil || body == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.request == "" {
		c.request = body
	}
}

// TeeResponse opens a new attempt and wraps rc so everything read from it is
// mirrored into that attempt. It tees rather than reads because a successful
// response is an SSE stream that must keep streaming to the caller.
// 한국어: 응답을 미리 소비하지 않고 읽히는 만큼 복사한다.
// 한국어: 원본 reader를 소비하지 않은 구간까지 보장하는 저장 방식은 아니며 소비된 HTTP 시도들을 기록한다.
func (c *Capture) TeeResponse(status int, rc io.ReadCloser) io.ReadCloser {
	if c == nil || rc == nil {
		return rc
	}
	a := &attempt{status: status}
	c.mu.Lock()
	c.attempts = append(c.attempts, a)
	c.mu.Unlock()
	return &teeBody{rc: rc, c: c, a: a}
}

// RawRequest returns the request body as sent, or "" if nothing was captured.
func (c *Capture) RawRequest() string {
	if c == nil {
		return ""
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.request
}

// RawResponse returns the response bytes as received. A single attempt yields
// the untouched original (copy-pasteable straight into a replay); multiple
// attempts are concatenated behind per-attempt header lines so a retry sequence
// stays readable.
// 한국어: 한 번의 시도는 그대로 복사 가능한 원문을 반환하고 여러 번이면 구분 헤더를 추가한다.
// 한국어: 따라서 여러 시도의 반환 문자열 전체는 하나의 원본 HTTP 응답 본문이 아니다.
func (c *Capture) RawResponse() string {
	if c == nil {
		return ""
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	switch len(c.attempts) {
	case 0:
		return ""
	case 1:
		return c.attempts[0].body.String()
	}
	var b strings.Builder
	for i, a := range c.attempts {
		fmt.Fprintf(&b, "===== attempt %d/%d — HTTP %d =====\n", i+1, len(c.attempts), a.status)
		body := a.body.String()
		b.WriteString(body)
		if !strings.HasSuffix(body, "\n") {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// Attempt is one HTTP round trip's status code and response bytes.
type Attempt struct {
	Status int
	Body   string
}

// Attempts returns every round trip in order. Unlike RawResponse — which drops
// the header line for a lone attempt so the bytes stay replayable — this always
// carries the status code, for callers that must report "HTTP 401 + body".
func (c *Capture) Attempts() []Attempt {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Attempt, 0, len(c.attempts))
	for _, a := range c.attempts {
		out = append(out, Attempt{Status: a.status, Body: a.body.String()})
	}
	return out
}

// teeBody mirrors reads into a Capture attempt, guarded by the Capture's mutex
// so a snapshot taken mid-stream never races the writer.
type teeBody struct {
	rc io.ReadCloser
	c  *Capture
	a  *attempt
}

func (t *teeBody) Read(p []byte) (int, error) {
	n, err := t.rc.Read(p)
	if n > 0 {
		t.c.mu.Lock()
		t.a.body.Write(p[:n])
		t.c.mu.Unlock()
	}
	return n, err
}

func (t *teeBody) Close() error { return t.rc.Close() }
