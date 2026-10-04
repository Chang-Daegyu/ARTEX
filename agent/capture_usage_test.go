package agent

// 한국어 파일 해설: agent/capture_usage_test.go
// Provider 실패와 context 취소 직전까지 받은 usage가 최종 Activity에도 남는지 검사한다.
// 성공한 답변만 세면 실패한 모델 요청의 비용이 사라지는 문제를 막기 위한 테스트이다.
// 테스트 이름은 아래의 각 Test 함수가 확인하는 구체적 경계를 나타낸다.
// 한국어 문서화 변경은 기존 테스트의 입력·예상값·실행 조건을 수정하지 않는다.

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/norma/agentcore"
	"github.com/Autumn-27/norma/llm"
)

type captureUsageProvider struct {
	stream func(context.Context, func(llm.StreamEvent, error) bool)
}

func (p captureUsageProvider) Stream(ctx context.Context, _ llm.CompletionRequest) iter.Seq2[llm.StreamEvent, error] {
	return func(yield func(llm.StreamEvent, error) bool) { p.stream(ctx, yield) }
}

func (p captureUsageProvider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.Message, string, llm.Usage, error) {
	acc := llm.NewAccumulator()
	for ev, err := range p.Stream(ctx, req) {
		if err != nil {
			return llm.Message{}, "", llm.Usage{}, err
		}
		acc.Add(ev)
	}
	return acc.Message(), acc.StopReason, acc.Usage, nil
}

func TestCaptureRunPersistsUsageOnProviderFailure(t *testing.T) {
	wantErr := errors.New("provider failed after reporting usage")
	provider := captureUsageProvider{stream: func(_ context.Context, yield func(llm.StreamEvent, error) bool) {
		// 遵守迭代器协议:yield 返回 false 后立即停止,不再调用它。
		if !yield(llm.StreamEvent{Type: llm.SEMessageStart, Usage: llm.Usage{InputTokens: 11, CacheReadTokens: 3}}, nil) {
			return
		}
		if !yield(llm.StreamEvent{Type: llm.SETextDelta, Text: "partial"}, nil) {
			return
		}
		if !yield(llm.StreamEvent{Type: llm.SEMessageDelta, Usage: llm.Usage{OutputTokens: 7, CacheWriteTokens: 2}}, nil) {
			return
		}
		yield(llm.StreamEvent{}, wantErr)
	}}

	var activities []db.Activity
	_, _, err := captureRun(context.Background(), agentcore.Options{Provider: provider, MaxTurns: 1}, "test", func(a db.Activity) {
		activities = append(activities, a)
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("captureRun error=%v, want %v", err, wantErr)
	}
	assertCapturedResultUsage(t, activities, 11, 7, 3, 2)
}

func TestCaptureRunPersistsUsageOnCancellation(t *testing.T) {
	started := make(chan struct{})
	provider := captureUsageProvider{stream: func(ctx context.Context, yield func(llm.StreamEvent, error) bool) {
		// 遵守迭代器协议:yield 返回 false 后立即停止,不再调用它。
		if !yield(llm.StreamEvent{Type: llm.SEMessageStart, Usage: llm.Usage{InputTokens: 13, CacheReadTokens: 5}}, nil) {
			return
		}
		if !yield(llm.StreamEvent{Type: llm.SETextDelta, Text: "partial"}, nil) {
			return
		}
		if !yield(llm.StreamEvent{Type: llm.SEMessageDelta, Usage: llm.Usage{OutputTokens: 9, CacheWriteTokens: 4}}, nil) {
			return
		}
		close(started)
		<-ctx.Done()
		yield(llm.StreamEvent{}, ctx.Err())
	}}

	ctx, cancel := context.WithCancel(context.Background())
	var activities []db.Activity
	done := make(chan error, 1)
	go func() {
		_, _, err := captureRun(ctx, agentcore.Options{Provider: provider, MaxTurns: 1}, "test", func(a db.Activity) {
			activities = append(activities, a)
		})
		done <- err
	}()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("captureRun error=%v, want context canceled", err)
	}
	assertCapturedResultUsage(t, activities, 13, 9, 5, 4)
}

func assertCapturedResultUsage(t *testing.T, activities []db.Activity, input, output, read, write int) {
	t.Helper()
	results := 0
	for _, activity := range activities {
		if activity.Kind != "result" {
			continue
		}
		results++
		if activity.InputTokens == nil || *activity.InputTokens != input ||
			activity.OutputTokens == nil || *activity.OutputTokens != output ||
			activity.CacheReadTokens == nil || *activity.CacheReadTokens != read ||
			activity.CacheWriteTokens == nil || *activity.CacheWriteTokens != write {
			t.Fatalf("result usage=%+v, want input=%d output=%d read=%d write=%d", activity, input, output, read, write)
		}
	}
	if results != 1 {
		t.Fatalf("result activity count=%d, activities=%+v", results, activities)
	}
}
