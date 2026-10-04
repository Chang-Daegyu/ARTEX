package sidequestion

// 한국어 파일 해설: sidequestion/service.go
// 별도 질문 하나를 직접 Provider에 보내 답을 누적하는 가장 작은 실행 계층이다.
// Agent Session, 도구 실행기, 주 transcript 기록기, 작업 모델 장애 전환 루프를 소유하지 않는다.
// 스트림 텍스트와 usage를 누적해 update에 알리고 message_stop 없이 종료되면 중단 오류로 표시한다.
// 모델이 tool_use를 생성해도 실행하지 않고 ToolUse 표시만 보관한다.
// 도구 호출만 있고 본문이 없으면 주 대화에서 요청하라는 안내를 반환한다.
// 취소·실패 전에 받은 답과 사용량은 반환 구조체에 남겨 서버가 최종 상태를 보존할 수 있다.

import (
	"context"
	"errors"
	"strings"

	"github.com/Autumn-27/norma/llm"
)

// SideQuestionService has no harness, tool executor, transcript writer or model
// failover chain. Answer is one completion; Respond adds bounded preparation
// and at most one context-overflow recovery around that completion.
type SideQuestionService struct{ Provider llm.Provider }

type Answer struct {
	Text    string
	Usage   llm.Usage
	ToolUse bool
}

// 한국어: 도구 사용 요청을 수신하더라도 호출할 executor가 없으므로 부작용은 실행되지 않는다.
// 한국어: update는 누적 답을 전달하며 취소가 나면 이미 받은 usage와 텍스트를 유지한다.
func (s SideQuestionService) Answer(ctx context.Context, req llm.CompletionRequest, streaming bool, update func(Answer)) (out Answer, err error) {
	if streaming {
		complete := false
		for ev, streamErr := range s.Provider.Stream(ctx, req) {
			if streamErr != nil {
				err = streamErr
				break
			}
			switch ev.Type {
			case llm.SETextDelta:
				out.Text += ev.Text
			case llm.SEToolUseStart:
				out.ToolUse = true
			case llm.SEMessageStart, llm.SEMessageDelta:
				out.Usage.Add(ev.Usage)
			case llm.SEMessageStop:
				complete = true
			}
			if update != nil {
				update(out)
			}
			if ctx.Err() != nil {
				err = ctx.Err()
				break
			}
		}
		if err == nil && !complete {
			err = errors.New("模型响应中断，请重新提问")
		}
	} else {
		var msg llm.Message
		msg, _, out.Usage, err = s.Provider.Complete(ctx, req)
		out.Text, out.ToolUse = msg.Text(), len(msg.ToolUses()) > 0
	}
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	if err == nil && strings.TrimSpace(out.Text) == "" {
		if out.ToolUse {
			out.Text = "当前旁路提问不能执行工具操作，请在主会话中发出操作请求。"
		} else {
			err = errors.New("模型没有返回回答")
		}
	}
	return out, err
}
