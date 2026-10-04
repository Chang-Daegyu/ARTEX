// [한국어 길잡이] 목표 초기화와 작업 실행 슬롯 배정
// HTTP 작업 생성과 spawn_task 도구가 launchTask를 공유한다. 초기 자산·선택적 seed intent 이후 실행 슬롯을 배정한다.
// bootstrap 경로는 목표 분해가 끝난 뒤 Engine.Run을 호출해 목표가 없는 순간 Planner가 실행되는 경쟁을 피한다.
// 동시 작업 한도가 켜져 있으면 DB에 FIFO 대기를 저장한다. 이미 실행 중인 작업·LLM 불가 작업·일시 정지·시간 초과 재시작을 구분한다.
// createGoals는 LLM 분해에 결과가 없을 때 원래 목표 전체를 단일 goal로 사용한다. 쉼표나 URL을 규칙으로 잘라 분해하지 않는다.
// 전체 모듈 설명과 파일별 읽기 순서: docs/ko/modules/server.md
package server

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/Autumn-27/artex/agent"
	"github.com/Autumn-27/artex/db"
)

type goalSpec struct {
	Text      string
	VulnClass string
}

// launchTask runs the shared post-creation sequence for a task created via ANY
// path (HTTP createTask 或 orchestration spawn_task),避免两处复制粘贴:
//  1. seed 根资产,喂给事件驱动 loop;
//  2. 可选种子意图,worker 免等首轮 planner 直接开跑;
//  3. 后台异步做目标分解(发「第 0 轮目标拆解」round + LLM 分解步骤 + 逐条 goal,页面可见),
//     分解完再 engine.Run —— 引擎在 goal 节点就绪后才启动,避免 planner 抢在 goal 之前跑的竞态。
//
// 异步(goroutine)所以调用方立即返回,两条路径行为一致:秒建任务、后台拆目标。
// [한국어 함수 설명] HTTP 생성과 Agent spawn_task의 공통 진입점이다. 먼저 seed를 기록하고, 실제 실행은 bootstrap 모드의 admission으로 넘긴다.
func (s *Server) launchTask(t *Task, seedText string, seedFirstIntent bool) {
	if !s.engine.beginTaskOperation(t.ID) {
		return
	}
	s.seed(t, seedText)
	if seedFirstIntent {
		s.seedFirstIntent(t)
	}
	s.engine.decInflight(t.ID)
	if _, err := s.admitTask(t, "bootstrap"); err != nil {
		log.Printf("[concurrency] task %s 启动失败: %v", t.ID, err)
	}
}

// [한국어 함수 설명] 0번째 목표 분해 라운드를 활동으로 남기고 goal 생성이 끝난 뒤 Engine.Run을 호출한다. 각 단계 사이에서 취소/삭제를 다시 검사한다.
func (s *Server) startTaskEngine(t *Task) {
	ctx := s.engine.execContextFor(s.ctx, t.ID)
	if ctx.Err() != nil || s.engine.IsDeleting(t.ID) {
		return
	}
	s.engine.emitActivity(t, db.Activity{Worker: "planner", Kind: "round",
		Summary: "第 0 轮目标拆解"})
	goals := s.createGoals(ctx, t, func(r db.Activity) {
		s.engine.emitActivity(t, r)
	})
	if ctx.Err() != nil || s.engine.IsDeleting(t.ID) {
		return
	}
	for _, g := range goals {
		summary := g.Text
		if g.VulnClass != "" {
			summary = fmt.Sprintf("[%s] %s", g.VulnClass, g.Text)
		}
		s.engine.emitActivity(t, db.Activity{Worker: "planner", Kind: "text", Summary: summary})
	}
	if ctx.Err() != nil || s.engine.IsDeleting(t.ID) {
		return
	}
	s.engine.Run(s.ctx, t)
}

// [한국어 함수 설명] FIFO 대기·사용자 pause·종료 상태를 제외하고 실제 실행 가능한 작업을 센다. 삭제 커밋 전에는 슬롯을 유지해 실패 복원 시 초과 실행을 피한다.
func (s *Server) occupiesConcurrencySlot(t *Task) bool {
	if t == nil {
		return false
	}
	lifecycle := t.lifecycleSnapshot()
	if lifecycle.Queued || lifecycle.Paused || isTerminalStatus(lifecycle.Status) {
		return false
	}
	// Deletion temporarily pauses the Engine but has not committed yet. Preserve
	// the task's slot until PostgreSQL deletion succeeds; otherwise FIFO promotion
	// during the drain window could over-admit if deletion later aborts and the
	// persisted running task is restored.
	if s.engine.IsDeleting(t.ID) {
		return true
	}
	return !s.engine.IsPaused(t.ID) && (s.engine.ReadyFor(t) || s.engine.ActiveLLMCalls(t.ID) > 0)
}

func (s *Server) runningTaskCount(excludeID string) int {
	count := 0
	for _, task := range s.m.List() {
		if task.ID != excludeID && s.occupiesConcurrencySlot(task) {
			count++
		}
	}
	return count
}

// admitTask is the single admission path for new, resumed, rerun and follow-up
// work. It atomically either starts the task or appends it to the persistent FIFO
// queue. mode is bootstrap for a freshly-created task and resume otherwise.
func (s *Server) admitTask(t *Task, mode string) (queued bool, err error) {
	return s.admitTaskWhen(t, mode, false)
}

// admitPausedTask is the task-control resume path. The paused precondition is
// checked under the same scheduler lock as admission so a concurrent pause,
// dequeue or FIFO promotion cannot leave database and Engine state divergent.
func (s *Server) admitPausedTask(t *Task) (queued bool, err error) {
	return s.admitTaskWhen(t, "resume", true)
}

// [한국어 함수 설명] 동시성 잠금 아래 현재 Task 핸들과 상태를 재확인하고 슬롯 부여 또는 FIFO 대기를 DB에 원자적으로 저장한다. 커밋 전에 실행 장벽을 설치하는 순서가 중요하다.
func (s *Server) admitTaskWhen(t *Task, mode string, requirePaused bool) (queued bool, err error) {
	if t == nil {
		return false, fmt.Errorf("task not found")
	}
	if mode != "bootstrap" {
		mode = "resume"
	}
	// 슬롯 계산·FIFO 결정·DB 기록을 같은 잠금 흐름에서 진행해야 두 요청이 마지막 한 자리를 동시에 받지 않는다.
	s.concMu.Lock()
	defer s.concMu.Unlock()
	// Delete installs its barrier under concMu as well. Re-resolve after acquiring
	// the lock so a request that captured a task pointer before successful deletion
	// cannot revive that stale handle after StopTask clears its Engine maps.
	current, exists := s.m.Task(t.ID)
	if !exists || current != t || s.engine.IsDeleting(t.ID) {
		return false, fmt.Errorf("task is being deleted")
	}
	if !s.engine.beginTaskOperation(t.ID) {
		return false, fmt.Errorf("task is being deleted")
	}
	defer s.engine.decInflight(t.ID)
	lifecycle := t.lifecycleSnapshot()
	if requirePaused {
		if isTerminalStatus(lifecycle.Status) {
			return false, fmt.Errorf("终态任务不能执行继续")
		}
		if !lifecycle.Paused {
			return false, fmt.Errorf("仅已暂停的任务可以继续")
		}
		mode = s.resumeAdmissionMode(t)
	}
	wasTerminal := isTerminalStatus(lifecycle.Status)
	wasPaused := lifecycle.Paused || s.engine.IsPaused(t.ID)
	wasQueued := lifecycle.Queued
	engineWasPaused := s.engine.IsPaused(t.ID)
	enabled, limit := s.m.ConcurrencyLimit()
	ready := s.engine.ReadyFor(t)

	// Work added to a task that is already admitted only needs a wake-up. This
	// matters when the configured limit was lowered below the current running
	// count: an existing task must not suddenly mark itself queued while its
	// planner/workers are still live. Still compare-and-commit the persisted
	// status: a concurrent terminal transition must win instead of being silently
	// reported as a successful follow-up admission.
	if !wasTerminal && !wasPaused && !wasQueued && s.engine.Started(t.ID) && (!enabled || ready) {
		if err := s.m.ApplyTaskAdmission(t.ID, lifecycle.Status, lifecycle.Status, false, "resume", false); err != nil {
			return false, err
		}
		s.startAdmittedTask(t, "resume")
		return false, nil
	}

	// Preserve the original first-run mode across repeated admissions. Legacy
	// queued rows have an empty queue_mode, so infer bootstrap from their graph.
	if wasQueued {
		switch lifecycle.QueueMode {
		case "bootstrap":
			mode = "bootstrap"
		case "":
			mode = s.resumeAdmissionMode(t)
		}
	}

	readyBacklog := enabled && s.hasReadyQueuedTask(t.ID)
	atCapacity := enabled && s.runningTaskCount(t.ID) >= limit
	shouldQueue := enabled && (!ready || readyBacklog || atCapacity)

	// Install the execution barrier before reviving a terminal/paused task. Without
	// this ordering, its already-running worker loops can claim the newly-opened
	// intent in the gap between status=running and queued=true.
	if shouldQueue || wasTerminal || wasPaused || wasQueued {
		s.engine.Pause(t.ID, agent.Causef("queued_for_admission", "任务等待运行准入",
			"任务正在等待并发队列或准入状态提交，本次执行已停止；只有获得运行槽后才会重新领取意图"))
	}

	status := lifecycle.Status
	if wasTerminal {
		status = "running"
	}
	if err := s.m.ApplyTaskAdmission(t.ID, lifecycle.Status, status, shouldQueue, mode, wasQueued); err != nil {
		if !engineWasPaused && !wasPaused && !wasQueued {
			s.engine.Resume(t)
		}
		return false, err
	}
	if lifecycle.Status == "timeout" && status == "running" {
		// ApplyTaskAdmission reset this timed-out run's persisted clock. Clear the
		// matching Engine gates before either parking it in FIFO or starting work;
		// otherwise the old settling flag would make every worker skip forever.
		s.engine.resetTimeoutRevival(t.ID)
	}
	if shouldQueue {
		if !wasQueued {
			summary := fmt.Sprintf("已排队：达到并发上限 %d，等待空位后自动开始", limit)
			switch {
			case !ready:
				summary = "已排队：当前没有可运行的 LLM 配置，配置恢复后自动开始"
			case readyBacklog:
				summary = "已排队：已有更早的任务等待运行，将按 FIFO 顺序自动开始"
			}
			s.engine.emitActivity(t, db.Activity{Worker: "system", Kind: "text", Summary: summary})
		}
		return true, nil
	}
	s.startAdmittedTask(t, mode)
	return false, nil
}

func (s *Server) hasReadyQueuedTask(excludeID string) bool {
	for _, task := range s.m.List() {
		lifecycle := task.lifecycleSnapshot()
		if task.ID != excludeID && lifecycle.Queued && !lifecycle.Paused && !isTerminalStatus(lifecycle.Status) && s.engine.ReadyFor(task) {
			return true
		}
	}
	return false
}

// [한국어 함수 설명] bootstrap이면 목표 분해 goroutine을 시작하고, resume이면 기존 루프와 컨텍스트를 다시 활성화한다. 시간 초과 재시작은 새 deadline 조정기도 준비한다.
func (s *Server) startAdmittedTask(t *Task, mode string) {
	if mode == "bootstrap" {
		if !s.engine.beginTaskOperation(t.ID) {
			return
		}
		// A first-run task may have kept the Engine pause barrier while waiting
		// in the concurrency queue. Clear it only after operation admission, or
		// startTaskEngine's first execContextFor call would return a cancelled
		// context and silently strand the dequeued task.
		if s.engine.IsPaused(t.ID) {
			s.engine.Resume(t)
		}
		go func() {
			defer s.engine.decInflight(t.ID)
			s.startTaskEngine(t)
		}()
		return
	}
	s.engine.Run(s.ctx, t)
	s.engine.Resume(t)
	// Run returns early for an already-started task. Explicitly ensure a timeout
	// coordinator exists after resume; resetTimeoutRevival cleared the completed
	// coordinator's marker and the next real call will stamp a fresh deadline.
	s.engine.startDeadlineCoordinator(s.ctx, t)
}

// [한국어 함수 설명] LLM 불가 작업을 대기시켜 슬롯을 돌려주고, 영속 queued_at 순서로 실행 가능한 오래된 작업을 시작한다. 모델이 아직 없는 항목은 슬롯을 소비하지 않는다.
func (s *Server) reconcileConcurrency() {
	s.concMu.Lock()
	defer s.concMu.Unlock()
	enabled, limit := s.m.ConcurrencyLimit()

	// A task whose provider chain becomes unavailable cannot do useful work and
	// must not reserve a limited running slot forever. Park it persistently so a
	// later profile edit/recovery can re-enter through the same FIFO path.
	if enabled {
		for _, task := range s.m.List() {
			lifecycle := task.lifecycleSnapshot()
			if lifecycle.Queued || lifecycle.Paused || s.engine.IsPaused(task.ID) || s.engine.IsDeleting(task.ID) ||
				isTerminalStatus(lifecycle.Status) || s.engine.ReadyFor(task) || s.engine.ActiveLLMCalls(task.ID) > 0 {
				continue
			}
			mode := s.resumeAdmissionMode(task)
			s.engine.Pause(task.ID, agent.Causef("llm_unavailable_queued", "LLM 不可用，任务进入等待队列",
				"任务当前无法解析可运行的 Planner/Worker LLM，已释放并发槽；配置恢复后按队列顺序继续"))
			if err := s.m.EnqueueTask(task.ID, mode); err != nil {
				s.engine.Resume(task)
				log.Printf("[concurrency] task %s 因 LLM 不可用入队失败: %v", task.ID, err)
				continue
			}
			s.engine.emitActivity(task, db.Activity{Worker: "system", Kind: "text",
				Summary: "已排队：当前没有可运行的 LLM 配置，配置恢复后自动开始"})
		}
	}

	type queuedTask struct {
		task      *Task
		lifecycle taskLifecycleState
	}
	queued := []queuedTask{}
	for _, task := range s.m.List() {
		lifecycle := task.lifecycleSnapshot()
		if lifecycle.Queued && !isTerminalStatus(lifecycle.Status) {
			queued = append(queued, queuedTask{task: task, lifecycle: lifecycle})
		}
	}
	sort.SliceStable(queued, func(i, j int) bool {
		left, right := queued[i].lifecycle.QueuedAt, queued[j].lifecycle.QueuedAt
		if left == 0 {
			left = queued[i].task.CreatedAt * int64(1e9)
		}
		if right == 0 {
			right = queued[j].task.CreatedAt * int64(1e9)
		}
		if left != right {
			return left < right
		}
		// Old rows may not have queued_at and task creation timestamps are only
		// kept to second precision in memory. Task ids are monotonic, providing a
		// deterministic oldest-first fallback for those ties.
		leftID, leftErr := strconv.ParseInt(queued[i].task.ID, 10, 64)
		rightID, rightErr := strconv.ParseInt(queued[j].task.ID, 10, 64)
		if leftErr == nil && rightErr == nil {
			return leftID < rightID
		}
		return queued[i].task.ID < queued[j].task.ID
	})
	slots := len(queued)
	if enabled {
		slots = limit - s.runningTaskCount("")
	}
	for _, entry := range queued {
		if slots <= 0 {
			break
		}
		task := entry.task
		lifecycle := task.lifecycleSnapshot()
		// Keep FIFO order, but do not consume a concurrency slot for a task
		// whose explicit chain/global provider is unavailable. It will be retried
		// after the user configures or resets its LLM chain.
		if lifecycle.Paused || (enabled && !s.engine.ReadyFor(task)) {
			continue
		}
		mode := lifecycle.QueueMode
		if mode != "bootstrap" && mode != "resume" {
			mode = s.resumeAdmissionMode(task)
		}
		if mode != "bootstrap" {
			mode = "resume"
		}
		if !s.engine.beginTaskOperation(task.ID) {
			continue
		}
		if err := s.m.ApplyTaskAdmission(task.ID, lifecycle.Status, lifecycle.Status, false, mode, false); err != nil {
			s.engine.decInflight(task.ID)
			continue
		}
		s.startAdmittedTask(task, mode)
		s.engine.decInflight(task.ID)
		slots--
	}
}

// reviveTask 让一个已停下的任务重新跑起来:把终态(done/failed/timeout)拉回 running、
// 解除暂停,并(重)启动引擎循环 + 唤醒。已在 running 且未暂停的任务:只剩 Run 里的一次
// Notify,近乎无副作用。用于「主 agent set_goals 新增目标」和「重跑 blocked 意图」两处。
//
// 为什么必须显式复活:planner/worker 循环的终态门(engine.go)会吞掉普通 notify——光改
// 图 + Notify 唤不醒已判完成的任务;重启后终态任务的 goroutine 也可能已不在,故还要 Run。
// [한국어 함수 설명] 종료된 작업에 새 목표/후속 의도가 생겼을 때 단순 Notify 대신 공통 admission으로 재진입한다. FIFO와 시간 예산 초기화를 같은 규칙으로 적용한다.
func (s *Server) reviveTask(t *Task) {
	if t == nil {
		return
	}
	if _, err := s.admitTask(t, "resume"); err != nil {
		log.Printf("[revive] task %s 恢复失败: %v", t.ID, err)
	}
}

// createGoals materializes the goal node(s) under the task root (rel objective).
// Decomposition is done ENTIRELY by the LLM (the project requires an LLM). There is
// no rule-based fallback splitter — it only ever produced garbage (shredded URLs,
// meaningless 2-way splits). If the LLM yields nothing (an error), the raw task goal
// is used verbatim as a single goal so the task still has something to judge against.
// Returns the seeded specs so callers can emit activity records for them.
// emit, when non-nil, is forwarded to DecomposeGoals so LLM steps are visible in the UI.
// [한국어 함수 설명] 작업의 실제 LLM 라우터로 목표 분해를 요청한다. 결과가 없을 때만 원래 목표를 하나의 goal로 저장하며, 도구가 이미 쓴 분해 결과를 중복 저장하지 않는다.
func (s *Server) createGoals(ctx context.Context, t *Task, emit func(db.Activity)) []goalSpec {
	if t == nil {
		return nil
	}
	// Use the SAME LLM the task runs on (its pinned profile, else the active profile),
	// NOT agent.FromEnv() — the LLM is configured via the UI (DB profile), not env vars,
	// so FromEnv returned empty and every task silently fell back to the crude rule
	// splitter (which shredded URLs / made meaningless 2-way splits).
	var specs []goalSpec
	var as *db.AssetStore
	if s.m != nil {
		as = s.m.Assets()
	}
	taskID, _ := strconv.ParseInt(t.ID, 10, 64)
	s.engine.BeginLLMCall(t.ID)
	goalRuntime := s.agentsForTask(t).runtime
	decomposed := agent.DecomposeGoalsWithProvider(ctx, goalRuntime, s.m.dir, t.Goal, t.Description, as, t.Store, taskID, goalRuntime.nonStreaming(), goalRuntime.maxTokens(), emit)
	s.engine.EndLLMCall(t.ID)
	for _, g := range decomposed {
		if strings.TrimSpace(g.Text) != "" {
			specs = append(specs, goalSpec{Text: g.Text, VulnClass: g.VulnClass})
		}
	}
	// 분해용 모델/도구가 결과를 못 써도 판단 대상 하나는 남긴다. 원문 전체를 그대로 보존해 URL·문장이 임의 분할되지 않게 한다.
	if len(specs) == 0 {
		// No decomposed goals (LLM error / no provider): use the raw task goal verbatim
		// as a single goal so the task still has something to judge against. This is the
		// only path that writes here — decomposed goals are already persisted by the tool.
		if g := strings.TrimSpace(t.Goal); g != "" {
			log.Printf("[goals] task %s: LLM 目标拆解无产出，回退为「原始目标作为单目标」", t.ID)
			origin, _ := t.Store.OriginFactID()
			id, _ := t.Store.AddNode(db.KindGoal, map[string]any{"text": g}, 0, "open", "system", nil)
			if origin > 0 && id > 0 {
				_ = t.Store.Link(origin, db.RelSpawns, id) // goal descends from the task root (origin fact)
			}
			specs = []goalSpec{{Text: g}}
		}
	}
	return specs
}
