package agent

// 한국어 파일 해설: agent/side_questions.go
// 기존 Agent의 RunInfo를 /btw의 부모 리소스 식별자로 바꾸는 접착 코드이다.
// conv- 접두어가 있는 일반 대화와 작업·탐색·intent가 있는 실행을 각각 식별한다.
// 부모를 특정할 수 없으면 아무 변경 없이 원래 context를 반환한다.
// 식별 가능하면 sidequestion.Attach를 통해 Options.Deps의 모델 호출만 감싼다.
// 이 함수는 별도 질문을 실행하거나 대화 파일을 수정하지 않고 체크포인트 수집 연결만 담당한다.

import (
	"context"
	"strconv"
	"strings"

	"github.com/Autumn-27/artex/sidequestion"
	"github.com/Autumn-27/norma/agentcore"
)

func attachSideCapture(ctx context.Context, opts *agentcore.Options) context.Context {
	ri := RunInfoFrom(ctx)
	p := sidequestion.Parent{TaskID: ri.TaskID, ExplorationID: ri.ExplorationID, IntentID: ri.IntentID}
	if strings.HasPrefix(ri.SessionID, "conv-") {
		p.ConversationID, _ = strconv.ParseInt(strings.TrimPrefix(ri.SessionID, "conv-"), 10, 64)
	}
	if p.ConversationID == 0 && (p.TaskID == 0 || p.ExplorationID == 0) {
		return ctx
	}
	ctx, opts.Deps = sidequestion.Attach(ctx, p, opts.Deps, opts.Provider)
	return ctx
}
