package agent

// 한국어 파일 해설: agent/side_questions_test.go
// 실제 ChatAgent와 가짜 모델·로컬 파일 도구를 조합해 /btw 체크포인트를 검증한다.
// 주 도구 결과는 스냅샷에 포함되지만 독립 질문은 주 transcript와 활동 스트림을 변경하지 않아야 한다.
// 테스트 이름은 아래의 각 Test 함수가 확인하는 구체적 경계를 나타낸다.
// 한국어 문서화 변경은 기존 테스트의 입력·예상값·실행 조건을 수정하지 않는다.

import (
	"context"
	"encoding/json"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/sidequestion"
	"github.com/Autumn-27/norma/llm"
	"github.com/Autumn-27/norma/transcript"
)

type sideAgentProvider struct {
	calls int
	path  string
}

func (p *sideAgentProvider) Stream(ctx context.Context, req llm.CompletionRequest) iter.Seq2[llm.StreamEvent, error] {
	return func(y func(llm.StreamEvent, error) bool) {
		msg, stop, _, err := p.Complete(ctx, req)
		if err != nil {
			y(llm.StreamEvent{}, err)
			return
		}
		for _, b := range msg.Content {
			if b.Type == llm.BlockToolUse {
				if !y(llm.StreamEvent{Type: llm.SEToolUseStart, ToolID: b.ID, ToolName: b.Name}, nil) {
					return
				}
				if !y(llm.StreamEvent{Type: llm.SEToolInputJSON, Text: string(b.Input)}, nil) {
					return
				}
			} else if !y(llm.StreamEvent{Type: llm.SETextDelta, Text: b.Text}, nil) {
				return
			}
		}
		if !y(llm.StreamEvent{Type: llm.SEMessageDelta, StopReason: stop}, nil) {
			return
		}
		y(llm.StreamEvent{Type: llm.SEMessageStop}, nil)
	}
}
func (p *sideAgentProvider) Complete(_ context.Context, req llm.CompletionRequest) (llm.Message, string, llm.Usage, error) {
	p.calls++
	if p.calls == 1 {
		input, _ := json.Marshal(map[string]string{"file_path": p.path})
		return llm.Message{Role: llm.RoleAssistant, Content: []llm.ContentBlock{{Type: llm.BlockToolUse, ID: "fixture-read", Name: "Read", Input: input}}}, "tool_use", llm.Usage{}, nil
	}
	return llm.Message{Role: llm.RoleAssistant, Content: []llm.ContentBlock{llm.TextBlock("main finished")}}, "end_turn", llm.Usage{}, nil
}

func TestSideActualChatCheckpointToolResultAndTranscriptIsolation(t *testing.T) {
	for _, streaming := range []bool{false, true} {
		t.Run(map[bool]string{true: "stream", false: "atomic"}[streaming], func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "asset.txt")
			if err := os.WriteFile(file, []byte("controlled-homepage-result"), 0600); err != nil {
				t.Fatal(err)
			}
			p := &sideAgentProvider{path: file}
			bound := sidequestion.Bind(p, sidequestion.Model{Model: "fixture", Streaming: streaming})
			store := transcript.NewStore(filepath.Join(dir, "transcripts"))
			chat := NewChatAgent(bound, "fixture", dir, store, 100000)
			chat.SetNonStreaming(func() bool { return !streaming })
			var snapshots []sidequestion.Snapshot
			var activities []db.Activity
			ctx := sidequestion.WithPublisher(t.Context(), func(s sidequestion.Snapshot) { snapshots = append(snapshots, s) })
			if _, err := chat.Chat(ctx, "mainagent", "conv-987654", "Read the local fixture", 5, time.Minute, false, func(a db.Activity) { activities = append(activities, a) }); err != nil {
				t.Fatal(err)
			}
			if len(snapshots) < 3 {
				t.Fatalf("actual Chat missed capture hooks: %d", len(snapshots))
			}
			last := snapshots[len(snapshots)-1]
			raw, _ := json.Marshal(last)
			if last.Parent.ConversationID != 987654 || !strings.Contains(string(raw), "controlled-homepage-result") || !strings.Contains(string(raw), "main finished") {
				t.Fatalf("missing real tool result/final reply: %s", raw)
			}
			before, err := os.ReadFile(store.MainPath("conv-987654"))
			if err != nil {
				t.Fatal(err)
			}
			count := len(activities)
			tools := 0
			for _, a := range activities {
				if a.Kind == "tool_use" {
					tools++
				}
			}
			if tools != 1 {
				t.Fatalf("main fixture tool executions: %d", tools)
			}
			req, err := sidequestion.BuildRequest(last, nil, "side-only-question")
			if err != nil {
				t.Fatal(err)
			}
			// Reset the fake to request another Read; the side executor cannot run it.
			p.calls = 0
			p.path = filepath.Join(dir, "nonexistent")
			answer, err := (sidequestion.SideQuestionService{Provider: bound}).Answer(t.Context(), req, streaming, nil)
			if err != nil || !answer.ToolUse || p.calls != 1 || !strings.Contains(answer.Text, "不能执行工具") {
				t.Fatalf("tool denial %+v %v", answer, err)
			}
			after, err := os.ReadFile(store.MainPath("conv-987654"))
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != string(after) || len(activities) != count {
				t.Fatal("side question modified main transcript/activity")
			}
			if len(snapshots) == 0 || snapshots[len(snapshots)-1].Version != last.Version {
				t.Fatal("side request replaced main checkpoint")
			}
		})
	}
}
