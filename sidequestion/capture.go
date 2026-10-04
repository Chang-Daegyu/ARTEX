// Package sidequestion captures immutable main-agent checkpoints. It never owns
// an agent session or a tool executor.
package sidequestion

// 한국어 파일 해설: sidequestion/capture.go
// 주 Agent의 안정된 모델 요청 경계를 독립 질문에 쓸 불변 스냅샷으로 만든다.
// 부모 키는 conversation 또는 task/exploration/intent 조합으로 정하며 재사용 Worker 슬롯과 구분한다.
// Attach는 QueryDeps의 주 루프 호출에만 표식을 붙여 요약용 보조 요청이 스냅샷을 덮지 않게 한다.
// Bind는 실제 Provider 바로 바깥에 놓여 모델 라우팅 뒤 선택된 프로필의 신원을 기록한다.
// 요청 시작 및 정상 완료 경계만 JSON 깊은 복사로 발행하고 반쯤 생성된 답은 완료본으로 발행하지 않는다.
// Finish는 마지막 모델 요청 뒤 도착한 도구 결과를 정상 API 메시지 형태로 반영한다.
// Model에는 설정 식별 정보만 담으며 API 키와 프록시 주소를 복사하지 않는다.

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"strings"
	"sync"
	"time"

	"github.com/Autumn-27/norma/harness"
	"github.com/Autumn-27/norma/llm"
)

type Parent struct {
	ConversationID int64 `json:"conversation_id,omitempty"`
	TaskID         int64 `json:"task_id,omitempty"`
	ExplorationID  int64 `json:"exploration_id,omitempty"`
	IntentID       int64 `json:"intent_id,omitempty"`
}

func (p Parent) Key() string {
	if p.ConversationID > 0 {
		return fmt.Sprintf("conv-%d", p.ConversationID)
	}
	if p.IntentID > 0 {
		return fmt.Sprintf("task-%d-exp-%d-worker-i%d", p.TaskID, p.ExplorationID, p.IntentID)
	}
	return fmt.Sprintf("task-%d-exp-%d-main", p.TaskID, p.ExplorationID)
}

// Model contains only configuration identity, never credentials or proxy URLs.
type Model struct {
	ProfileID    int64  `json:"profile_id"`
	Name         string `json:"name"`
	Format       string `json:"format"`
	Model        string `json:"model"`
	Identity     string `json:"identity"`
	Streaming    bool   `json:"streaming"`
	WindowTokens int    `json:"window_tokens"`
}

type Snapshot struct {
	Parent     Parent                `json:"parent"`
	RunID      int64                 `json:"run_id"`
	Version    int64                 `json:"version"`
	CapturedAt time.Time             `json:"captured_at"`
	Model      Model                 `json:"model"`
	Request    llm.CompletionRequest `json:"request"`
}

func CloneRequest(req llm.CompletionRequest) (llm.CompletionRequest, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return llm.CompletionRequest{}, err
	}
	var out llm.CompletionRequest
	err = json.Unmarshal(b, &out)
	return out, err
}

type Publisher func(Snapshot)
type publisherKey struct{}
type captureKey struct{}
type attemptKey struct{}

func WithPublisher(ctx context.Context, publish Publisher) context.Context {
	return context.WithValue(ctx, publisherKey{}, publish)
}

type Capture struct {
	mu      sync.Mutex
	parent  Parent
	runID   int64
	version int64
	publish Publisher
	last    *Snapshot
}

// Attach uses QueryDeps rather than a global provider hook so compaction and
// other auxiliary completions cannot replace the main conversation checkpoint.
// 한국어: 주 모델 호출에만 attemptKey를 붙이고 보조 압축 요청은 구분한다.
// 한국어: 구체 Provider의 Bind가 이 표식을 발견할 때만 선택된 실제 모델과 요청을 발행한다.
func Attach(ctx context.Context, parent Parent, deps harness.QueryDeps, provider llm.Provider) (context.Context, harness.QueryDeps) {
	publish, _ := ctx.Value(publisherKey{}).(Publisher)
	if publish == nil {
		return ctx, deps
	}
	c := &Capture{parent: parent, runID: time.Now().UnixNano(), publish: publish}
	stream, complete := deps.CallModel, deps.CallModelSync
	if stream == nil {
		stream = provider.Stream
	}
	if complete == nil {
		complete = provider.Complete
	}
	deps.CallModel = func(ctx context.Context, req llm.CompletionRequest) iter.Seq2[llm.StreamEvent, error] {
		return stream(context.WithValue(ctx, attemptKey{}, c), req)
	}
	deps.CallModelSync = func(ctx context.Context, req llm.CompletionRequest) (llm.Message, string, llm.Usage, error) {
		return complete(context.WithValue(ctx, attemptKey{}, c), req)
	}
	return context.WithValue(ctx, captureKey{}, c), deps
}

// 한국어: 먼저 JSON 깊은 복사 후 잠금 안에서 버전을 증가시키고 잠금 밖에서 발행한다.
// 한국어: 스냅샷 기록 실패가 주 Agent 실행을 실패시키지 않도록 복사 오류는 이 경계에서 무시한다.
func (c *Capture) save(req llm.CompletionRequest, model Model) {
	copy, err := CloneRequest(req)
	if err != nil {
		return
	} // Observability must not break the main run.
	c.mu.Lock()
	c.version++
	s := Snapshot{Parent: c.parent, RunID: c.runID, Version: c.version, CapturedAt: time.Now().UTC(), Model: model, Request: copy}
	c.last = &s
	c.mu.Unlock()
	c.publish(s)
}

// Finish adds tool results that landed after the final model request. Norma's
// first API message is its host reminder, absent from Terminal.Messages.
// 한국어: 주 모델의 마지막 호출 뒤 도착한 도구 결과를 Terminal.Messages에서 반영한다.
// 한국어: Norma가 요청에만 붙이는 system-reminder 접두 메시지는 기존 스냅샷에서 보존한다.
func Finish(ctx context.Context, messages []llm.Message) {
	c, _ := ctx.Value(captureKey{}).(*Capture)
	if c == nil {
		return
	}
	c.mu.Lock()
	s := c.last
	c.mu.Unlock()
	if s == nil {
		return
	}
	req := s.Request
	var prefix []llm.Message
	if len(req.Messages) > 0 && strings.HasPrefix(req.Messages[0].Text(), "<system-reminder>") {
		prefix = append(prefix, req.Messages[0])
	}
	req.Messages = append(prefix, llm.MessagesForAPI(messages)...)
	c.save(req, s.Model)
}

type boundProvider struct {
	inner llm.Provider
	model Model
}

// Bind belongs immediately around a concrete provider, inside any routing pool.
func Bind(inner llm.Provider, model Model) llm.Provider { return &boundProvider{inner, model} }

// 한국어: 요청 시작점은 즉시 보존하되 완료된 응답은 message_stop·무오류·미취소 조건이 모두 맞을 때만 저장한다.
func (p *boundProvider) Stream(ctx context.Context, req llm.CompletionRequest) iter.Seq2[llm.StreamEvent, error] {
	return func(yield func(llm.StreamEvent, error) bool) {
		c, _ := ctx.Value(attemptKey{}).(*Capture)
		if c == nil {
			for ev, err := range p.inner.Stream(ctx, req) {
				if !yield(ev, err) {
					return
				}
			}
			return
		}
		c.save(req, p.model)
		acc := llm.NewAccumulator()
		failed := false
		complete := false
		for ev, err := range p.inner.Stream(ctx, req) {
			if err != nil {
				failed = true
			} else {
				acc.Add(ev)
				if ev.Type == llm.SEMessageStop {
					complete = true
				}
			}
			if !yield(ev, err) {
				return
			}
		}
		if !failed && complete && ctx.Err() == nil {
			msg := acc.Message()
			req.Messages = llm.MessagesForAPI(append(append([]llm.Message{}, req.Messages...), msg))
			c.save(req, p.model)
		}
	}
}

func (p *boundProvider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.Message, string, llm.Usage, error) {
	c, _ := ctx.Value(attemptKey{}).(*Capture)
	if c != nil {
		c.save(req, p.model)
	}
	msg, stop, usage, err := p.inner.Complete(ctx, req)
	if c != nil && err == nil && ctx.Err() == nil {
		req.Messages = llm.MessagesForAPI(append(append([]llm.Message{}, req.Messages...), msg))
		c.save(req, p.model)
	}
	return msg, stop, usage, err
}
