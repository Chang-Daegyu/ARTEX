package agent

// 한국어 파일 해설: agent/taskclock.go
// 서버의 작업 전체 마감 시각을 각 Planner/Worker run에 전달한다.
// DeadlineUnix가 0이면 작업 전체 시간 제한이 없고 Final은 최종 Planner 정리 회차를 뜻한다.
// clampMaxDuration은 run 자체 예산과 남은 작업 시간 중 더 작은 유효값을 택한다.
// SDK에서 0 이하 시간이 무제한으로 해석되므로 남은 시간이 없더라도 최소 1초로 맞춘다.
// 반환 clamped는 전체 작업 마감이 이번 run을 제한했는지 알려 수습 문구 선택에 쓰인다.
// 마감 정보 전달과 실제 context 취소 원인은 분리되어 있다.

import (
	"context"
	"time"
)

// TaskClock carries a task's absolute deadline into a worker/planner run so the run
// can clamp its own wall-clock budget to the task's remaining time and pick the
// right wrap-up words (per-run vs task-timeout). Attached to the run ctx by the
// engine. Zero value = no task-level timeout (behaves exactly as before).
type TaskClock struct {
	DeadlineUnix int64 // absolute deadline (unix seconds); 0 = no task timeout
	Final        bool  // coordinator-driven FINAL planner round (task ending now)
}

type taskClockKey struct{}

// WithTaskClock attaches a TaskClock to ctx for the run.
func WithTaskClock(ctx context.Context, tc TaskClock) context.Context {
	return context.WithValue(ctx, taskClockKey{}, tc)
}

// taskClockFrom reads the TaskClock (zero value if none attached).
func taskClockFrom(ctx context.Context) TaskClock {
	if v, ok := ctx.Value(taskClockKey{}).(TaskClock); ok {
		return v
	}
	return TaskClock{}
}

// clampMaxDuration folds a task deadline into a run's own wall-clock budget.
//   - ownBudget = the agent's own run_seconds (0 = unlimited).
//   - Returns eff = the MaxDuration to use (floored at 1s so we never pass ≤0, which
//     harness reads as "unlimited"), and clamped = whether the TASK deadline is the
//     binding constraint (remaining ≤ ownBudget, or ownBudget unlimited). When there
//     is no deadline, returns (ownBudget, false) unchanged.
// 한국어: 작업 마감과 run 예산의 최솟값을 계산하되 SDK의 0=무제한 규칙 때문에 최소 1초를 보장한다.
func clampMaxDuration(deadlineUnix int64, ownBudget time.Duration) (eff time.Duration, clamped bool) {
	if deadlineUnix <= 0 {
		return ownBudget, false
	}
	remaining := time.Until(time.Unix(deadlineUnix, 0))
	if remaining < time.Second {
		remaining = time.Second // max(1, …): never pass ≤0 (harness treats 0 as unlimited)
	}
	// clamped when the task deadline binds this run's time: remaining ≤ own budget,
	// or the agent has no own time budget (then remaining always binds).
	clamped = ownBudget <= 0 || remaining <= ownBudget
	eff = remaining
	if ownBudget > 0 && ownBudget < remaining {
		eff = ownBudget
	}
	return eff, clamped
}
