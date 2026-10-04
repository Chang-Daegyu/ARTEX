package agent

// 한국어 파일 해설: agent/runinfo.go
// 도구 조립 시 해당 호출이 어느 작업·탐색·intent·대화에 속하는지 전달한다.
// AugmentTools의 입력에 모든 ID를 추가하는 대신 context 값으로 RunInfo를 주입한다.
// 서버의 Skill 사용 기록 등은 이 정보를 읽어 정확한 소유 작업에 활동을 귀속시킨다.
// TaskID와 ExplorationID는 서로 다른 테이블의 식별자이며 같은 숫자라고 가정하면 안 된다.
// IntentID는 Worker 실행에서만 설정하고 SessionID는 작업 외 대화에서 사용한다.
// 0 또는 빈 문자열은 미지정 의미이며 explorationID는 nil 저장소도 허용한다.

import (
	"context"

	"github.com/Autumn-27/artex/db"
)

// RunInfo identifies WHICH run a tool call belongs to. Tool assembly only receives
// (ctx, agentKey) — the task/exploration ids live in the caller's arguments, not the
// ctx — so anything wired at assembly time (currently the Skill ledger in
// server/assembly.go) has no way to attribute a call to a task. Each run attaches
// its own RunInfo before calling AugmentTools; the wiring closure reads it once and
// captures it, so per-run attribution stays correct without threading parameters
// through the tool layer. Same pattern as TaskClock (see taskclock.go).
//
// Zero value = attribution unknown; every consumer must treat it as optional.
type RunInfo struct {
	TaskID        int64  // task registry id; 0 for non-task runs (chat sessions)
	ExplorationID int64  // exploration id; 0 when unknown
	IntentID      int64  // worker's intent node; 0 for planner/mainagent/chat
	SessionID     string // chat conversation id; empty for task runs
}

// explorationID reads a store's exploration id, tolerating a nil store (planner and
// worker runs can be driven without one in tests).
func explorationID(ts *db.ExplorationStore) int64 {
	if ts == nil {
		return 0
	}
	return ts.ID()
}

type runInfoKey struct{}

// WithRunInfo attaches run attribution to ctx.
// 한국어: 도구가 어떤 run에 속하는지 context에 붙인다. 서로 다른 작업 간 전역 가변 ID를 공유하지 않는다.
func WithRunInfo(ctx context.Context, ri RunInfo) context.Context {
	return context.WithValue(ctx, runInfoKey{}, ri)
}

// RunInfoFrom reads the RunInfo (zero value if none attached).
func RunInfoFrom(ctx context.Context) RunInfo {
	if v, ok := ctx.Value(runInfoKey{}).(RunInfo); ok {
		return v
	}
	return RunInfo{}
}
