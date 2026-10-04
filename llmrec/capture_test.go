package llmrec

// 한국어 파일 해설: llmrec/capture_test.go
// nil 캡처의 무동작, 단일 응답 원문 보존, 재시도 응답 순서·상태 보존, HTTP가 없었던 호출의 빈 캡처를 검사한다.
// 원문 바이트와 재구성 JSON을 구별하는 작은 단위 테스트이다.
// 테스트 이름은 아래의 각 Test 함수가 확인하는 구체적 경계를 나타낸다.
// 한국어 문서화 변경은 기존 테스트의 입력·예상값·실행 조건을 수정하지 않는다.

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestCaptureFromNilAndMissing(t *testing.T) {
	var nilCtx context.Context // the transport reaches here on any unrecorded call
	if CaptureFrom(nilCtx) != nil {
		t.Fatal("nil context should yield no capture")
	}
	if CaptureFrom(context.Background()) != nil {
		t.Fatal("bare context should yield no capture")
	}
}

// A nil *Capture is the "recording off" path: the transport calls these on every
// request, so they must all be no-ops rather than panics.
func TestNilCaptureIsInert(t *testing.T) {
	var c *Capture
	c.SetRequest("body")
	if got := c.RawRequest(); got != "" {
		t.Fatalf("RawRequest()=%q want empty", got)
	}
	if got := c.RawResponse(); got != "" {
		t.Fatalf("RawResponse()=%q want empty", got)
	}
	rc := io.NopCloser(strings.NewReader("x"))
	if c.TeeResponse(200, rc) != rc {
		t.Fatal("nil capture must pass the body through untouched")
	}
}

func TestCaptureSingleAttemptKeepsBytesVerbatim(t *testing.T) {
	ctx, c := NewCapture(context.Background())
	if CaptureFrom(ctx) != c {
		t.Fatal("capture not retrievable from its own context")
	}
	c.SetRequest(`{"model":"x"}`)
	// Retries re-send identical bytes; only the first is kept.
	c.SetRequest(`{"model":"ignored"}`)

	const sse = "event: message_start\ndata: {\"type\":\"message_start\"}\n\nevent: message_stop\ndata: {}\n\n"
	body := c.TeeResponse(200, io.NopCloser(strings.NewReader(sse)))
	got, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != sse {
		t.Fatal("tee altered the stream delivered to the caller")
	}
	if err := body.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if c.RawRequest() != `{"model":"x"}` {
		t.Fatalf("RawRequest()=%q", c.RawRequest())
	}
	// A lone attempt must stay byte-identical — no headers, no framing added, so
	// it can be replayed as-is.
	if c.RawResponse() != sse {
		t.Fatalf("RawResponse()=%q want verbatim SSE", c.RawResponse())
	}
}

// norma's doStream discards the bodies of retried attempts (retry.go closes them
// unread), so the capture is the only place a 429 body survives.
func TestCaptureRetriedAttemptsAreAllKept(t *testing.T) {
	_, c := NewCapture(context.Background())

	first := c.TeeResponse(429, io.NopCloser(strings.NewReader(`{"error":"rate_limit"}`)))
	if _, err := io.ReadAll(first); err != nil {
		t.Fatalf("read first: %v", err)
	}
	second := c.TeeResponse(200, io.NopCloser(strings.NewReader("data: ok\n")))
	if _, err := io.ReadAll(second); err != nil {
		t.Fatalf("read second: %v", err)
	}

	raw := c.RawResponse()
	if !strings.Contains(raw, `{"error":"rate_limit"}`) {
		t.Fatalf("dropped the retried attempt body: %q", raw)
	}
	if !strings.Contains(raw, "data: ok") {
		t.Fatalf("dropped the final attempt body: %q", raw)
	}
	if !strings.Contains(raw, "attempt 1/2 — HTTP 429") || !strings.Contains(raw, "attempt 2/2 — HTTP 200") {
		t.Fatalf("attempts not delimited: %q", raw)
	}
	if strings.Index(raw, "rate_limit") > strings.Index(raw, "data: ok") {
		t.Fatal("attempts stored out of order")
	}
}

func TestCaptureEmptyWhenNoHTTPHappened(t *testing.T) {
	_, c := NewCapture(context.Background())
	if c.RawRequest() != "" || c.RawResponse() != "" {
		t.Fatal("unused capture should be empty")
	}
}
