// [한국어 길잡이] 영속 저장소와 실행 중 작업 핸들의 소유자
// Manager는 PostgreSQL·공유 자산·트래픽·승인기를 연결하고 Task 핸들을 관리한다. Task는 탐색 저장소 및 동시 접근용 상태 스냅샷을 가진다.
// 작업 수·Worker 수·기록 토글·프록시·LLM 관련 설정을 저장하며, 실행 슬롯 배정과 상태 갱신은 DB 커밋 뒤 메모리에 반영한다.
// 삭제는 파일·트래픽을 먼저 복구 가능한 위치로 옮기고 DB 삭제 실패 시 되돌린다. DB 커밋 이후 물리 정리 실패는 이미 삭제된 작업을 되살리지 않는다.
// Notify 계열 함수는 그래프 변경의 이유를 모은 뒤 비차단 채널로 Planner를 깨운다. 이 알림 자체는 영속 작업 큐가 아니다.
// 트래픽/LLM 원시 기록은 기본 비활성이다. enrich 인스턴스 생성만으로 모든 자산에 자동 DNS·HTTP 보강 호출이 연결되었다고 볼 수 없다.
// 전체 모듈 설명과 파일별 읽기 순서: docs/ko/modules/server.md
package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Autumn-27/artex/agent"
	pgdb "github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/enrich"
	"github.com/Autumn-27/artex/guard"
	"github.com/Autumn-27/artex/intercept"
	"github.com/Autumn-27/artex/traffic"
	actool "github.com/Autumn-27/norma/tool"
)

// Task is one engagement: a description + goal + its own exploration store,
// sharing the process-wide asset store. ID is the PG task id as a string; ExpID
// is the exploration the task owns.
type Task struct {
	ID           string `json:"id"`
	ExpID        int64  `json:"exploration_id"`
	Name         string `json:"name"` // 可选任务名称;空=未命名
	CategoryID   *int64 `json:"category_id,omitempty"`
	CategoryName string `json:"category_name,omitempty"`
	PinnedAt     int64  `json:"pinned_at,omitempty"`
	Description  string `json:"description"`
	Goal         string `json:"goal"`
	CreatedAt    int64  `json:"created_at"`
	CompletedAt  int64  `json:"completed_at,omitempty"` // 进入终态的 unix 秒;0=未完成
	Paused       bool   `json:"paused"`
	Queued       bool   `json:"queued"` // 因并发上限被挂起、等待空位自动启动;true=尚未开跑
	// QueuedAt is an internal Unix-nanosecond ordering key. It is deliberately
	// finer than CreatedAt so several tasks enqueued in the same second retain
	// their real FIFO order.
	QueuedAt           int64   `json:"queued_at,omitempty"`
	QueueMode          string  `json:"queue_mode,omitempty"`
	ParentRef          string  `json:"parent_ref,omitempty"`     // 父任务 id(编排 spawn 记录)
	LLMProfileID       *int64  `json:"llm_profile_id,omitempty"` // 指定运行本任务 planner/worker 的 LLM 配置;nil=用全局激活配置
	LLMProfileIDs      []int64 `json:"llm_profile_ids,omitempty"`
	ActiveLLMProfileID *int64  `json:"active_llm_profile_id,omitempty"`
	LLMChainRevision   int64   `json:"-"`
	LLMFailoverState   string  `json:"llm_failover_state,omitempty"`
	LLMFailoverReason  string  `json:"llm_failover_reason,omitempty"`
	SourceTaskIDs      []int64 `json:"source_task_ids,omitempty"`
	CompanyIDs         []int64 `json:"company_ids,omitempty"`
	Status             string  `json:"status"` // persisted lifecycle status (done/failed/timeout 为终态；空/其它则由运行态推导)
	// 任务级超时(见 docs/任务级超时与收尾设计.md)。DeadlineAt/FirstRunAt 为 unix 秒,0=未设/未运行。
	TimeoutSeconds       int                    `json:"timeout_seconds"`
	PlanHeartbeatSeconds int                    `json:"plan_heartbeat_seconds"` // planner 心跳触发间隔(秒)
	CoverageEnabled      bool                   `json:"coverage_enabled"`       // 资产覆盖度功能开关(创建时定,默认开)
	FirstRunAt           int64                  `json:"first_run_at,omitempty"`
	DeadlineAt           int64                  `json:"deadline_at,omitempty"`
	Store                *pgdb.ExplorationStore `json:"-"`
	Guard                *guard.Guard           `json:"-"`
	notify               chan struct{}
	lifecycleMu          sync.RWMutex
	llmMu                sync.RWMutex

	// pendingTriggers accumulates the concrete changes (worker done / finding) that
	// fired planning rounds since the last one consumed them. The debounce coalesces
	// a burst into one round, so several may pile up before drainTriggers() clears them.
	trigMu          sync.Mutex
	pendingTriggers []agent.TriggerEvent
}

// taskLifecycleState is an internally consistent view of the mutable task
// lifecycle and inherited-scope context. Callers must use lifecycleSnapshot and
// updateLifecycle instead of reading or writing the corresponding Task fields
// directly after the task has been published by Manager.
type taskLifecycleState struct {
	Name          string
	PinnedAt      int64
	Status        string
	Paused        bool
	Queued        bool
	QueuedAt      int64
	QueueMode     string
	CompletedAt   int64
	FirstRunAt    int64
	DeadlineAt    int64
	SourceTaskIDs []int64
	CompanyIDs    []int64
	CategoryID    *int64
	CategoryName  string
}

// [한국어 함수 설명] 읽기 잠금 안에서 관련 필드를 한 시점에 복사한다. 슬라이스와 포인터도 복사해 호출자의 변경이 공유 상태로 새지 않게 한다.
func (t *Task) lifecycleSnapshot() taskLifecycleState {
	if t == nil {
		return taskLifecycleState{}
	}
	t.lifecycleMu.RLock()
	defer t.lifecycleMu.RUnlock()
	return t.lifecycleSnapshotLocked()
}

func (t *Task) lifecycleSnapshotLocked() taskLifecycleState {
	return taskLifecycleState{
		Name:          t.Name,
		PinnedAt:      t.PinnedAt,
		Status:        t.Status,
		Paused:        t.Paused,
		Queued:        t.Queued,
		QueuedAt:      t.QueuedAt,
		QueueMode:     t.QueueMode,
		CompletedAt:   t.CompletedAt,
		FirstRunAt:    t.FirstRunAt,
		DeadlineAt:    t.DeadlineAt,
		// 슬라이스를 복사하지 않으면 스냅샷 사용자가 공유 backing array를 바꿔 잠금 보호를 우회할 수 있다.
		SourceTaskIDs: append([]int64(nil), t.SourceTaskIDs...),
		CompanyIDs:    append([]int64(nil), t.CompanyIDs...),
		CategoryID:    cloneInt64Ptr(t.CategoryID),
		CategoryName:  t.CategoryName,
	}
}

// [한국어 함수 설명] 잠금 아래 복사한 상태를 수정 콜백에 넘긴 뒤 모든 필드를 한 번에 반영한다. 여러 필드를 서로 다른 시점으로 읽는 경쟁을 줄인다.
func (t *Task) updateLifecycle(update func(*taskLifecycleState)) {
	if t == nil || update == nil {
		return
	}
	t.lifecycleMu.Lock()
	state := t.lifecycleSnapshotLocked()
	update(&state)
	t.Name = state.Name
	t.PinnedAt = state.PinnedAt
	t.Status = state.Status
	t.Paused = state.Paused
	t.Queued = state.Queued
	t.QueuedAt = state.QueuedAt
	t.QueueMode = state.QueueMode
	t.CompletedAt = state.CompletedAt
	t.FirstRunAt = state.FirstRunAt
	t.DeadlineAt = state.DeadlineAt
	t.SourceTaskIDs = append(t.SourceTaskIDs[:0], state.SourceTaskIDs...)
	t.CompanyIDs = append(t.CompanyIDs[:0], state.CompanyIDs...)
	t.CategoryID = cloneInt64Ptr(state.CategoryID)
	t.CategoryName = state.CategoryName
	t.lifecycleMu.Unlock()
}

type taskLLMState struct {
	ProfileID      *int64
	ProfileIDs     []int64
	ActiveID       *int64
	ChainRevision  int64
	FailoverState  string
	FailoverReason string
}

func cloneInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func (t *Task) llmStateSnapshot() taskLLMState {
	t.llmMu.RLock()
	defer t.llmMu.RUnlock()
	return taskLLMState{
		ProfileID:      cloneInt64Ptr(t.LLMProfileID),
		ProfileIDs:     append(make([]int64, 0, len(t.LLMProfileIDs)), t.LLMProfileIDs...),
		ActiveID:       cloneInt64Ptr(t.ActiveLLMProfileID),
		ChainRevision:  t.LLMChainRevision,
		FailoverState:  t.LLMFailoverState,
		FailoverReason: t.LLMFailoverReason,
	}
}

// [한국어 함수 설명] LLM 체인 revision을 비교해 늦게 도착한 이전 상태가 최신 체인을 덮어쓰는 것을 막는다. 프로필 ID 슬라이스는 외부와 공유하지 않는다.
func (t *Task) setLLMState(profileID, activeID *int64, profileIDs []int64, revision int64, state, reason string) bool {
	t.llmMu.Lock()
	defer t.llmMu.Unlock()
	if revision < t.LLMChainRevision {
		return false
	}
	t.LLMProfileID = cloneInt64Ptr(profileID)
	t.ActiveLLMProfileID = cloneInt64Ptr(activeID)
	t.LLMProfileIDs = append(t.LLMProfileIDs[:0], profileIDs...)
	t.LLMChainRevision = revision
	t.LLMFailoverState = state
	t.LLMFailoverReason = reason
	return true
}

// DeleteTaskOptions controls cleanup of data stored outside the task's own
// exploration graph. All options default to false for backward compatibility.
type DeleteTaskOptions struct {
	DeleteAssets     bool `json:"delete_assets"`
	DeleteTraffic    bool `json:"delete_traffic"`
	DeleteFiles      bool `json:"delete_files"`
	DeleteFindings   bool `json:"delete_findings"`
	DeleteLLMRecords bool `json:"delete_llm_records"`
}

// DeleteTaskResult makes destructive cleanup auditable to API callers.
type DeleteTaskResult struct {
	Deleted           string `json:"deleted"`
	AssetsDeleted     int64  `json:"assets_deleted"`
	AssetsDetached    int64  `json:"assets_detached"`
	TrafficDeleted    int64  `json:"traffic_deleted"`
	FilesDeleted      bool   `json:"files_deleted"`
	FindingsDeleted   int64  `json:"findings_deleted"`
	LLMRecordsDeleted int64  `json:"llm_records_deleted"`
	CleanupWarning    string `json:"cleanup_warning,omitempty"`
}

// Manager owns the PostgreSQL data source (asset graph + every task's exploration
// graph + config) and the in-memory set of task handles.
type Manager struct {
	dir         string
	pg          *pgdb.DB
	assets      *pgdb.AssetStore
	traffic     *traffic.Traffic       // process-wide recording proxy (may be nil)
	enrich      *enrich.Engine         // engine-side asset auto-completion (DNS/HTTP)
	interceptor *intercept.Interceptor // user-configured tool-call interception rules

	companyMu sync.Mutex // serializes task/company-scope commits with live handle registration
	// taskStateMu preserves commit order between PostgreSQL lifecycle writes and
	// their in-memory mirrors. lifecycleMu makes snapshots race-free, but without
	// this outer write lock an older request could commit first and publish last.
	taskStateMu sync.Mutex
	mu          sync.RWMutex
	tasks       map[string]*Task
	active      string
	trafficOn   bool // 流量捕获开关（默认关；settings.traffic_capture）
	llmRecOn    bool // LLM 录制开关（默认关；settings.llm_record）
	// 联网搜索开关与来源（默认关；settings.web_search_*）。brave-free 需要 braveKey；tavily 需要 tavilyKey。
	// webSearchProxy 是独立出口代理(http/https/socks5)，与记录流量的 MITM 代理无关。
	webSearchOn      bool
	webSearchBackend string
	braveKey         string
	tavilyKey        string
	webSearchProxy   string
	// globalProxy is the egress proxy all target traffic routes through
	// (http/https/socks5, optional user:pass). Empty = direct. When traffic
	// capture is on it becomes the MITM's upstream; when capture is off it is
	// injected into agent bash env / WebFetch directly. See ProxyAddr.
	globalProxy string
}

// Settings keys the UI toggles at runtime.
const (
	settingTrafficCapture      = "traffic_capture"
	settingAgentTrafficBinding = "agent_traffic_binding"
	settingWebSearchOn         = "web_search_enabled"
	settingWebSearchBackend    = "web_search_backend"
	settingBraveKey            = "brave_search_api_key"
	settingTavilyKey           = "tavily_search_api_key"
	settingWebSearchProxy      = "web_search_proxy"
	// settingGlobalProxy is the global egress proxy for all target traffic
	// (http/https/socks5). Empty = direct. Distinct from web_search_proxy (which
	// only routes the search backend) and the per-profile LLM proxy.
	settingGlobalProxy = "global_proxy"
	settingWorkers     = "workers"
	settingLLMRecord   = "llm_record"
	// LLM 轮询(故障转移)。默认关闭——开启后走「全局激活配置」的 agent 在当前配置
	// 不可用(余额不足/key 失效/限流/服务异常)时自动切到下一个配置。
	// settingLLMPoolBindFallback 仅在轮询开启时有意义:默认关闭,即 agent/任务显式
	// 绑定了某个配置就只用它、失败即失败;开启后绑定的配置失败也会回落到轮询链。
	settingLLMPoolOn           = "llm_pool_enabled"
	settingLLMPoolBindFallback = "llm_pool_bind_fallback"
	// 任务并发上限:开关 + 上限数。默认关闭;开启后默认上限 5(见 defaultConcurrencyLimit)。
	settingConcurrencyOn    = "task_concurrency_enabled"
	settingConcurrencyLimit = "task_concurrency_limit"
	// 实验功能:noa 模型驱动上下文压缩(norma v0.4.0)。默认关闭——开启后平台接入的四类
	// agent(planner/worker/主 agent/对话)由 noa 接管上下文压缩,取代内置 compaction。
	// 每 run 读一次,切换只影响之后启动的 run。
	settingNoaCompaction = "noa_compaction"
	// defaultWebSearchBackend is used when web search is on but no backend was picked.
	defaultWebSearchBackend = "ddgs"
	// deepSeekWebSearchBackend borrows the active LLM profile instead of its own
	// key, so it only works on an anthropic-format profile pointed at DeepSeek.
	deepSeekWebSearchBackend = "deepseek"
	// defaultWorkers is the concurrent work-agent count when the setting is unset.
	defaultWorkers = 3
	// defaultConcurrencyLimit is the simultaneous-running-task cap when the feature
	// is enabled but no explicit limit was saved.
	defaultConcurrencyLimit = 5
)

// ConcurrencyLimit returns whether the simultaneous-running-task cap is enabled and
// its limit (default 5 when enabled but unset). limit is always >=1 when enabled.
func (m *Manager) ConcurrencyLimit() (enabled bool, limit int) {
	on, _, _ := m.pg.GetSetting(settingConcurrencyOn)
	if strings.TrimSpace(on) != "true" {
		return false, 0
	}
	limit = defaultConcurrencyLimit
	if v, ok, _ := m.pg.GetSetting(settingConcurrencyLimit); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n >= 1 {
			limit = n
		}
	}
	return true, limit
}

// SetConcurrency persists the running-task concurrency cap. limit<1 is clamped to 1.
func (m *Manager) SetConcurrency(enabled bool, limit int) error {
	if limit < 1 {
		limit = defaultConcurrencyLimit
	}
	if err := m.pg.SetSetting(settingConcurrencyLimit, strconv.Itoa(limit)); err != nil {
		return err
	}
	return m.pg.SetSetting(settingConcurrencyOn, strconv.FormatBool(enabled))
}

// Workers returns the configured concurrent work-agent count (default 3). Read
// per-task at engine.Run, so a change applies to tasks started afterwards.
func (m *Manager) Workers() int {
	v, ok, err := m.pg.GetSetting(settingWorkers)
	if err != nil || !ok {
		return defaultWorkers
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n <= 0 {
		return defaultWorkers
	}
	return n
}

// SetWorkers persists the concurrent work-agent count. Values <=0 are rejected.
func (m *Manager) SetWorkers(n int) error {
	if n <= 0 {
		return fmt.Errorf("workers 必须 >0")
	}
	return m.pg.SetSetting(settingWorkers, strconv.Itoa(n))
}

// Enrich returns the asset auto-completion engine (may be nil if init failed).
func (m *Manager) Enrich() *enrich.Engine { return m.enrich }

// NewManager connects to PostgreSQL and, if proxyAddr is non-empty, starts the
// traffic-recording proxy. PostgreSQL is required (it is the single data source).
// [한국어 함수 설명] 필수 PostgreSQL을 연 뒤 복구·보조 테이블·승인기를 준비하고, 선택적으로 트래픽 저장소/프록시와 보강 엔진을 연결한다.
func NewManager(dir, proxyAddr string) (*Manager, error) {
	// Resolve the data dir to an ABSOLUTE path up front. Every data path derives
	// from it — notably the MITM CA cert, whose path is injected into worker shells
	// (SSL_CERT_FILE/CURL_CA_BUNDLE) and read by WebFetch. A relative path (the
	// default is "./data" under `go run`) only resolves when the current working
	// directory happens to match, so curl/WebFetch in a different CWD fail to load
	// the CA → TLS to the proxy breaks (curl 000 / EOF). Absolute makes it CWD-proof.
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	dsn, source, err := pgdb.DSN()
	if err != nil {
		return nil, err
	}
	log.Printf("[pg] 数据库配置来源: %s", source)
	pg, err := pgdb.Open(dsn)
	if err != nil {
		return nil, err
	}
	if err := pg.RecoverFindingRetests(); err != nil {
		pg.Close()
		return nil, fmt.Errorf("recover finding retests: %w", err)
	}
	if err := pg.EnsureLLMRecordsTable(); err != nil {
		log.Printf("[llmrec] create table: %v", err)
	}
	if err := pg.EnsureLLMUsageTable(); err != nil {
		log.Printf("[llmusage] create table: %v", err)
	}
	m := &Manager{dir: dir, pg: pg, assets: pg.Assets(), tasks: map[string]*Task{}, interceptor: intercept.New(pg)}
	if proxyAddr != "" {
		tr, err := traffic.Open(filepath.Join(dir, "traffic"), proxyAddr)
		if err != nil {
			log.Printf("[traffic] disabled: %v", err)
		} else {
			err = tr.RecoverHostDeleteStages(func(_ int64, taskID int64) (bool, error) {
				if taskID <= 0 {
					return false, errors.New("归档流量暂存日志缺少任务 ID")
				}
				task, taskErr := pg.GetTask(taskID)
				if taskErr != nil {
					return false, taskErr
				}
				// Archived/permanently deleted tasks are hidden from GetTask. A
				// restored task is visible and needs the staged traffic put back.
				return task == nil, nil
			})
			if err != nil {
				_ = tr.Close()
				_ = pg.Close()
				return nil, fmt.Errorf("recover traffic delete staging: %w", err)
			}
		}
		if tr != nil {
			m.traffic = tr
			go func() {
				log.Printf("[traffic] recording proxy on %s (set HTTP_PROXY=%s + trust _ca CA)", proxyAddr, tr.ProxyAddr())
				if err := tr.Start(); err != nil {
					log.Printf("[traffic] proxy stopped: %v", err)
				}
			}()
		}
	}
	// Asset auto-completion engine (§5): HTTP probes routed through the recording
	// proxy (via m.ProxyAddr, which honors the traffic-capture toggle).
	// 프록시 객체 존재와 Agent가 기록 프록시로 연결하도록 설정된 상태는 다르다. 실제 도구/환경 주입은 이 토글을 따른다.
	m.trafficOn = pg.GetBool(settingTrafficCapture, false)
	// LLM 录制开关（默认关）。录制器每次调用时读取此标志。
	m.llmRecOn = pg.GetBool(settingLLMRecord, false)
	// Load persisted web-search config (default: off, ddgs).
	m.webSearchOn = pg.GetBool(settingWebSearchOn, false)
	if v, ok, _ := pg.GetSetting(settingWebSearchBackend); ok && v != "" {
		m.webSearchBackend = v
	} else {
		m.webSearchBackend = defaultWebSearchBackend
	}
	if v, ok, _ := pg.GetSetting(settingBraveKey); ok {
		m.braveKey = v
	}
	if v, ok, _ := pg.GetSetting(settingTavilyKey); ok {
		m.tavilyKey = v
	}
	if v, ok, _ := pg.GetSetting(settingWebSearchProxy); ok {
		m.webSearchProxy = v
	}
	// Global egress proxy (default: direct). When capture is on, feed it to the
	// MITM as its upstream so recorded traffic exits through it; when capture is
	// off, ProxyAddr hands it to agents directly (bash env / WebFetch).
	if v, ok, _ := pg.GetSetting(settingGlobalProxy); ok {
		m.globalProxy = strings.TrimSpace(v)
	}
	if m.traffic != nil {
		if err := m.traffic.SetUpstreamProxy(m.globalProxy); err != nil {
			log.Printf("[proxy] 全局代理 %q 无效，已忽略: %v", m.globalProxy, err)
		}
	}
	m.enrich = enrich.New(m.assets, m.ProxyAddr, 4)
	// Reconcile the seeded browser MCP with the persisted capture state, so a
	// restart with capture already on keeps Playwright routed through the proxy.
	m.syncBrowserMCPProxy()
	return m, nil
}

// TrafficEnabled reports whether traffic capture is on (default off). When off,
// no proxy/traffic tools/prompt are injected into agents (nothing is recorded).
func (m *Manager) TrafficEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.trafficOn
}

// SetTrafficEnabled persists and applies the traffic-capture toggle. Callers must
// rebuild the agents (applyLLM) afterwards so the new proxy/tools/prompt take hold.
// [한국어 함수 설명] 기록 토글을 저장하고 프록시 관련 설정을 반영한다. Agent 도구/프롬프트/환경은 호출자가 런타임을 다시 조립해야 새 값으로 바뀐다.
func (m *Manager) SetTrafficEnabled(on bool) error {
	if err := m.pg.SetBool(settingTrafficCapture, on); err != nil {
		return err
	}
	m.mu.Lock()
	m.trafficOn = on
	m.mu.Unlock()
	// Inject (on) or strip (off) the recording proxy + CA on the browser MCP so
	// Playwright routes through the MITM. Must run after the flag flip above, since
	// ProxyAddr/ProxyCACert honor it. putSettings rebuilds agents next (applyLLM),
	// which re-spawns the MCP with the new args/env.
	m.syncBrowserMCPProxy()
	return nil
}

// LLMRecordEnabled reports whether LLM request/response recording is on
// (默认关；settings.llm_record). The recorder consults this per call, so the
// toggle takes effect immediately without rebuilding agents.
// [한국어 함수 설명] 원시 LLM 요청 기록 토글을 읽는다. 요청마다 조회하므로 이미 만든 provider에서도 다음 호출부터 변경을 반영할 수 있다.
func (m *Manager) LLMRecordEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.llmRecOn
}

// SetLLMRecordEnabled persists and applies the LLM-record toggle. Effective at
// once — no applyLLM needed, since the recorder reads the flag on every call.
func (m *Manager) SetLLMRecordEnabled(on bool) error {
	if err := m.pg.SetBool(settingLLMRecord, on); err != nil {
		return err
	}
	m.mu.Lock()
	m.llmRecOn = on
	m.mu.Unlock()
	return nil
}

// NoaCompactionEnabled reports whether the experimental noa context-compression
// mechanism is on (默认关；settings.noa_compaction). Read per agent run via the
// injected resolver, so a toggle takes effect on the next run without rebuild.
func (m *Manager) NoaCompactionEnabled() bool {
	return m.pg.GetBool(settingNoaCompaction, false)
}

// SetNoaCompaction persists the noa toggle. Effective on the next agent run —
// the resolver reads it per run, so no rebuild is needed.
func (m *Manager) SetNoaCompaction(on bool) error {
	return m.pg.SetBool(settingNoaCompaction, on)
}

// LLMPoolEnabled reports whether LLM failover ("轮询") is on (默认关；
// settings.llm_pool_enabled). Read when the provider chain is built (applyLLM),
// so a change requires a rebuild — putSettings does that.
func (m *Manager) LLMPoolEnabled() bool {
	if m.pg == nil {
		return false
	}
	return m.pg.GetBool(settingLLMPoolOn, false)
}

// SetLLMPoolEnabled persists the failover toggle. Callers rebuild agents
// (applyLLM) afterwards so it takes effect.
func (m *Manager) SetLLMPoolEnabled(on bool) error { return m.pg.SetBool(settingLLMPoolOn, on) }

// LLMPoolBindFallback reports whether an agent/task that is BOUND to a specific
// profile still falls back to the chain when that profile fails (默认关：绑定即
// 独占，失败即失败). Only meaningful while LLMPoolEnabled.
func (m *Manager) LLMPoolBindFallback() bool {
	if m.pg == nil {
		return false
	}
	return m.pg.GetBool(settingLLMPoolBindFallback, false)
}

// SetLLMPoolBindFallback persists the bound-profile fallback toggle. Callers
// rebuild agents (applyLLM) afterwards.
func (m *Manager) SetLLMPoolBindFallback(on bool) error {
	return m.pg.SetBool(settingLLMPoolBindFallback, on)
}

// WebSearch returns the current web-search config: whether it is enabled, the
// backend ("ddgs" | "brave-free" | "tavily"), the Brave API key, the Tavily API
// key (each empty unless set), and the dedicated egress proxy (empty = direct).
func (m *Manager) WebSearch() (on bool, backend, braveKey, tavilyKey, proxy string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	backend = m.webSearchBackend
	if backend == "" {
		backend = defaultWebSearchBackend
	}
	return m.webSearchOn, backend, m.braveKey, m.tavilyKey, m.webSearchProxy
}

// WebSearchOpts returns the config as the agent-package struct the server pushes
// into each agent. Disabled when off, or when a keyed backend is selected without
// its key (so a half-configured backend never silently drops the tool at session build).
func (m *Manager) WebSearchOpts() agent.WebSearchOpts {
	on, backend, braveKey, tavilyKey, proxy := m.WebSearch()
	if on && backend == "brave-free" && strings.TrimSpace(braveKey) == "" {
		on = false
	}
	if on && backend == "tavily" && strings.TrimSpace(tavilyKey) == "" {
		on = false
	}
	o := agent.WebSearchOpts{Enabled: on, Backend: backend, BraveKey: braveKey, TavilyKey: tavilyKey, Proxy: proxy}
	if backend == deepSeekWebSearchBackend {
		o.DeepSeekBaseURL, o.DeepSeekAPIKey, o.DeepSeekModel = m.deepSeekSearchCreds()
	}
	return o
}

// deepSeekSearchCreds resolves the credentials the "deepseek" search backend
// borrows from the active LLM profile (it has no key of its own). Whether that
// profile can actually drive server-side search — DeepSeek exposes it only on
// the Anthropic-format endpoint — is deliberately NOT validated here: the UI
// states the requirement and the user decides. A profile that can't serve it
// simply fails at search time (or at the settings page's 测试 button), which is
// the same feedback every other backend gives for a bad key.
func (m *Manager) deepSeekSearchCreds() (baseURL, apiKey, model string) {
	p, err := m.pg.ActiveProfile()
	if err != nil || p == nil {
		return "", "", ""
	}
	return p.BaseURL, p.APIKey, p.Model
}

// SetWebSearch persists and applies the web-search settings. braveKey, tavilyKey, and
// proxy are each left untouched when nil (so toggling the switch doesn't wipe a saved
// key/proxy; pass a pointer to "" to clear). Callers must rebuild agents (applyLLM)
// afterwards so the settings take effect.
func (m *Manager) SetWebSearch(on bool, backend string, braveKey, tavilyKey, proxy *string) error {
	backend = strings.TrimSpace(backend)
	if backend == "" {
		backend = defaultWebSearchBackend
	}
	if err := m.pg.SetBool(settingWebSearchOn, on); err != nil {
		return err
	}
	if err := m.pg.SetSetting(settingWebSearchBackend, backend); err != nil {
		return err
	}
	m.mu.Lock()
	m.webSearchOn = on
	m.webSearchBackend = backend
	m.mu.Unlock()
	if braveKey != nil {
		if err := m.pg.SetSetting(settingBraveKey, *braveKey); err != nil {
			return err
		}
		m.mu.Lock()
		m.braveKey = *braveKey
		m.mu.Unlock()
	}
	if tavilyKey != nil {
		if err := m.pg.SetSetting(settingTavilyKey, *tavilyKey); err != nil {
			return err
		}
		m.mu.Lock()
		m.tavilyKey = *tavilyKey
		m.mu.Unlock()
	}
	if proxy != nil {
		p := strings.TrimSpace(*proxy)
		if err := m.pg.SetSetting(settingWebSearchProxy, p); err != nil {
			return err
		}
		m.mu.Lock()
		m.webSearchProxy = p
		m.mu.Unlock()
	}
	return nil
}

// browserMCPName is the seeded Playwright MCP whose proxy args + CA env are kept
// in sync with the traffic-capture toggle.
const browserMCPName = "browser"

// syncBrowserMCPProxy reconciles the seeded browser MCP's proxy args + CA env with
// the current traffic-capture state: capture on → route Playwright through the
// recording proxy (--proxy-server) and trust its MITM CA (NODE_EXTRA_CA_CERTS);
// capture off → strip both. Idempotent, and a no-op if the user deleted/renamed the
// MCP. Must be called WITHOUT m.mu held (ProxyAddr/ProxyCACert take the lock).
// [한국어 함수 설명] 기록 토글에 맞춰 기본 브라우저 MCP의 프록시 인자와 신뢰 CA 환경을 정리한다. 중복 플래그를 제거한 후 현재 값만 다시 넣는다.
func (m *Manager) syncBrowserMCPProxy() {
	servers, err := m.pg.ListMCP()
	if err != nil {
		log.Printf("[mcp] browser 代理同步: 读取 MCP 列表失败: %v", err)
		return
	}
	var srv *pgdb.MCPServer
	for _, s := range servers {
		if s.Name == browserMCPName {
			srv = s
			break
		}
	}
	if srv == nil {
		return // user removed/renamed it — leave it alone
	}

	proxy := m.ProxyAddr()  // "" when capture off
	cert := m.ProxyCACert() // "" when capture off

	args := stripProxyArgs(decodeStrSlice(srv.Args))
	env := decodeStrMap(srv.Env)
	delete(env, "NODE_EXTRA_CA_CERTS")
	if proxy != "" {
		args = append(args, "--proxy-server", proxy)
		if cert != "" {
			env["NODE_EXTRA_CA_CERTS"] = cert
		}
	}
	srv.Args = encodeJSON(args)
	srv.Env = encodeJSON(env)
	if _, err := m.pg.SaveMCP(srv); err != nil {
		log.Printf("[mcp] browser 代理同步失败: %v", err)
		return
	}
	if proxy != "" {
		log.Printf("[mcp] browser MCP 已挂捕获代理 %s (CA %s)", proxy, cert)
	} else {
		log.Printf("[mcp] browser MCP 已移除捕获代理配置")
	}
}

// stripProxyArgs removes any --proxy-server/--proxy-bypass flags (both "--flag val"
// and "--flag=val" forms) so they can be re-added cleanly from current state,
// without mutating the input slice.
func stripProxyArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--proxy-server" || a == "--proxy-bypass" {
			i++ // skip the following value too
			continue
		}
		if strings.HasPrefix(a, "--proxy-server=") || strings.HasPrefix(a, "--proxy-bypass=") {
			continue
		}
		out = append(out, a)
	}
	return out
}

func decodeStrSlice(raw json.RawMessage) []string {
	var out []string
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return out
}

func decodeStrMap(raw json.RawMessage) map[string]string {
	out := map[string]string{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return out
}

func encodeJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}

// HostTools are runtime host-provided tools added to EVERY agent's base list (via
// ToolAugment); the tools table then filters them per-agent binding. Currently the
// traffic tools, gated by the global capture switch: empty when capture is off, so
// no agent gets traffic_search/traffic_get regardless of binding.
func (m *Manager) HostTools() []actool.CoreTool {
	if m.traffic == nil || !m.TrafficEnabled() {
		return nil
	}
	return m.traffic.Tools()
}

func (m *Manager) Assets() *pgdb.AssetStore  { return m.assets }
func (m *Manager) PG() *pgdb.DB              { return m.pg }
func (m *Manager) Traffic() *traffic.Traffic { return m.traffic }

// ProxyAddr returns the egress proxy address agents route target traffic through:
//   - capture ON  → the recording MITM proxy (which itself exits via the global
//     proxy when one is set); agents also get its CA (see ProxyCACert).
//   - capture OFF → the global egress proxy directly (empty CA — real target
//     certs), or "" when no global proxy is set (direct, no recording).
//
// So the global proxy takes effect in both modes: at the MITM's upstream when
// capturing, in the agent's own bash env / WebFetch when not.
// [한국어 함수 설명] 기록 중에는 MITM 주소, 기록이 꺼져 있으면 전역 egress 프록시를 반환한다. 기록 여부와 외부 프록시 사용 여부는 별개 설정이다.
func (m *Manager) ProxyAddr() string {
	if m.traffic != nil && m.TrafficEnabled() {
		return m.traffic.ProxyAddr()
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.globalProxy
}

// ProxyCACert returns the CA cert path agents must trust to verify HTTPS through
// the egress proxy. Non-empty ONLY when traffic capture is on (the MITM re-signs
// certs): the global proxy used directly (capture off) is a plain forwarder that
// preserves real target certs, so no custom CA is needed there. Its emptiness is
// also the worker's "recording off" signal (see workerSystem).
func (m *Manager) ProxyCACert() string {
	if m.traffic == nil || !m.TrafficEnabled() {
		return ""
	}
	return m.traffic.CACertPath()
}

// GlobalProxy returns the configured global egress proxy URL (empty = direct).
func (m *Manager) GlobalProxy() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.globalProxy
}

// SetGlobalProxy validates, persists and applies the global egress proxy
// (http/https/socks5, optional user:pass; empty = direct). It updates the MITM's
// upstream immediately; callers must rebuild agents (applyLLM) afterwards so the
// capture-off path (bash env / WebFetch) picks up the change too.
func (m *Manager) SetGlobalProxy(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw != "" {
		if _, err := traffic.ValidateProxyURL(raw); err != nil {
			return err
		}
	}
	if err := m.pg.SetSetting(settingGlobalProxy, raw); err != nil {
		return err
	}
	m.mu.Lock()
	m.globalProxy = raw
	m.mu.Unlock()
	if m.traffic != nil {
		if err := m.traffic.SetUpstreamProxy(raw); err != nil {
			return err
		}
	}
	// Keep the browser MCP's egress in sync with the new global proxy too.
	m.syncBrowserMCPProxy()
	return nil
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.traffic != nil {
		m.traffic.Close()
	}
	return m.pg.Close()
}

// isTerminalStatus reports whether a task status is terminal (done/failed/timeout).
// Package-local shim over db.IsTerminal so all server files share one definition.
func isTerminalStatus(status string) bool { return pgdb.IsTerminal(status) }

// unixOrZero returns t's unix seconds, or 0 when the time is nil.
func unixOrZero(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return t.Unix()
}

func unixNanoOrZero(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return t.UnixNano()
}

func taskFromPG(pt *pgdb.Task, store *pgdb.ExplorationStore, ic *intercept.Interceptor) *Task {
	return &Task{
		ID: strconv.FormatInt(pt.ID, 10), ExpID: pt.ExplorationID,
		Name:       pt.Name,
		CategoryID: cloneInt64Ptr(pt.CategoryID), CategoryName: pt.CategoryName,
		PinnedAt:    unixOrZero(pt.PinnedAt),
		Description: pt.Description, Goal: pt.Goal, CreatedAt: pt.CreatedAt.Unix(), Paused: pt.Paused, Queued: pt.Queued,
		QueuedAt: unixNanoOrZero(pt.QueuedAt), QueueMode: pt.QueueMode,
		CompletedAt: unixOrZero(pt.CompletedAt), Status: pt.Status, ParentRef: pt.ParentRef,
		LLMProfileID:  pt.LLMProfileID,
		LLMProfileIDs: append([]int64(nil), pt.LLMProfileIDs...), ActiveLLMProfileID: pt.ActiveLLMProfileID,
		LLMChainRevision: pt.LLMChainRevision,
		LLMFailoverState: pt.LLMFailoverState, LLMFailoverReason: pt.LLMFailoverReason,
		SourceTaskIDs:  append([]int64(nil), pt.SourceTaskIDs...),
		CompanyIDs:     append([]int64(nil), pt.CompanyIDs...),
		TimeoutSeconds: pt.TimeoutSeconds, PlanHeartbeatSeconds: pt.PlanHeartbeatSeconds,
		CoverageEnabled: pt.CoverageEnabled,
		FirstRunAt:      unixOrZero(pt.FirstRunAt), DeadlineAt: unixOrZero(pt.DeadlineAt),
		Store: store, Guard: guard.NewWithInterceptor(ic), notify: make(chan struct{}, 1),
	}
}

// UpdateTaskMetadata changes list-only task metadata without interrupting any
// planner, main-agent, or worker call.
func (m *Manager) UpdateTaskMetadata(taskID string, patch pgdb.TaskPatch) (*Task, error) {
	m.taskStateMu.Lock()
	defer m.taskStateMu.Unlock()
	id, err := strconv.ParseInt(taskID, 10, 64)
	if err != nil || id <= 0 {
		return nil, nil
	}
	updated, err := m.pg.UpdateTask(id, patch)
	if err != nil || updated == nil {
		return nil, err
	}
	m.mu.RLock()
	task := m.tasks[taskID]
	m.mu.RUnlock()
	if task == nil {
		return nil, nil
	}
	task.updateLifecycle(func(state *taskLifecycleState) {
		state.Name = updated.Name
		state.PinnedAt = unixOrZero(updated.PinnedAt)
	})
	return task, nil
}

// CreateTask creates a task + its exploration and makes it active.
// timeoutSeconds is the task-level wall-clock budget (0 = 不限时).
func (m *Manager) CreateTask(description, goal string, llmProfileID *int64, timeoutSeconds, planHeartbeatSeconds int) (*Task, error) {
	var ids []int64
	if llmProfileID != nil {
		ids = []int64{*llmProfileID}
	}
	return m.CreateTaskWithOptions(description, goal, pgdb.TaskCreateOptions{
		LLMProfileIDs: ids, TimeoutSeconds: timeoutSeconds, PlanHeartbeatSeconds: planHeartbeatSeconds,
	})
}

// [한국어 함수 설명] DB에 작업과 탐색 저장소를 만든 뒤 실행 중 레지스트리에 Task를 등록한다. 실제 Planner/Worker 시작은 Server의 launchTask가 맡는다.
func (m *Manager) CreateTaskWithOptions(description, goal string, opts pgdb.TaskCreateOptions) (*Task, error) {
	if len(opts.CompanyIDs) > 0 {
		m.companyMu.Lock()
		defer m.companyMu.Unlock()
	}
	pt, err := m.pg.CreateTaskWithOptions(description, goal, opts)
	if err != nil {
		return nil, err
	}
	t := taskFromPG(pt, m.pg.Exploration(pt.ExplorationID), m.interceptor)
	m.mu.Lock()
	m.tasks[t.ID] = t
	m.active = t.ID
	m.mu.Unlock()
	return t, nil
}

// RenameTaskCategory persists a category name and refreshes every live task DTO
// that references it. taskStateMu keeps this ordered with task reassignment.
func (m *Manager) RenameTaskCategory(id int64, name string) (*pgdb.TaskCategory, error) {
	m.taskStateMu.Lock()
	defer m.taskStateMu.Unlock()
	category, err := m.pg.RenameTaskCategory(id, name)
	if err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, task := range m.tasks {
		task.updateLifecycle(func(state *taskLifecycleState) {
			if state.CategoryID != nil && *state.CategoryID == id {
				state.CategoryName = category.Name
			}
		})
	}
	return category, nil
}

// DeleteTaskCategory moves all affected live tasks to the uncategorized bucket.
func (m *Manager) DeleteTaskCategory(id int64) (bool, error) {
	m.taskStateMu.Lock()
	defer m.taskStateMu.Unlock()
	deleted, err := m.pg.DeleteTaskCategory(id)
	if err != nil || !deleted {
		return deleted, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, task := range m.tasks {
		task.updateLifecycle(func(state *taskLifecycleState) {
			if state.CategoryID != nil && *state.CategoryID == id {
				state.CategoryID = nil
				state.CategoryName = ""
			}
		})
	}
	return true, nil
}

// SetTaskCategory updates one live task without interrupting its runtime.
func (m *Manager) SetTaskCategory(taskID string, categoryID *int64) (*pgdb.TaskCategory, error) {
	m.taskStateMu.Lock()
	defer m.taskStateMu.Unlock()
	id, err := strconv.ParseInt(taskID, 10, 64)
	if err != nil || id <= 0 {
		return nil, pgdb.ErrTaskCategoryTaskNotFound
	}
	category, err := m.pg.SetTaskCategory(id, categoryID)
	if err != nil {
		return nil, err
	}
	m.mu.RLock()
	task := m.tasks[taskID]
	m.mu.RUnlock()
	if task != nil {
		task.updateLifecycle(func(state *taskLifecycleState) {
			state.CategoryID = cloneInt64Ptr(categoryID)
			state.CategoryName = ""
			if category != nil {
				state.CategoryName = category.Name
			}
		})
	}
	return category, nil
}

// SetTasksCategory applies one category change to several tasks at once. The
// database write and the in-memory refresh share taskStateMu, so a concurrent
// single-task update cannot interleave and leave a live DTO stale. The returned
// set holds the ids that were actually moved; callers report the rest as gone.
func (m *Manager) SetTasksCategory(taskIDs []string, categoryID *int64) (map[string]bool, *pgdb.TaskCategory, error) {
	m.taskStateMu.Lock()
	defer m.taskStateMu.Unlock()
	numeric := make([]int64, 0, len(taskIDs))
	for _, taskID := range taskIDs {
		id, err := strconv.ParseInt(taskID, 10, 64)
		if err != nil || id <= 0 {
			return nil, nil, pgdb.ErrTaskCategoryTaskNotFound
		}
		numeric = append(numeric, id)
	}
	updatedIDs, category, err := m.pg.SetTasksCategory(numeric, categoryID)
	if err != nil {
		return nil, nil, err
	}
	categoryName := ""
	if category != nil {
		categoryName = category.Name
	}
	updated := make(map[string]bool, len(updatedIDs))
	m.mu.RLock()
	tasks := make([]*Task, 0, len(updatedIDs))
	for _, id := range updatedIDs {
		taskID := strconv.FormatInt(id, 10)
		updated[taskID] = true
		if task := m.tasks[taskID]; task != nil {
			tasks = append(tasks, task)
		}
	}
	m.mu.RUnlock()
	for _, task := range tasks {
		task.updateLifecycle(func(state *taskLifecycleState) {
			state.CategoryID = cloneInt64Ptr(categoryID)
			state.CategoryName = categoryName
		})
	}
	return updated, category, nil
}

// DeleteCompanyWithAssets keeps the database cascade and live task handles in
// one manager-level critical section. This closes the gap where a task could
// commit its company scope immediately before registration and miss the
// post-delete in-memory sweep.
func (m *Manager) DeleteCompanyWithAssets(id int64, deleteAssets bool) (int64, error) {
	m.companyMu.Lock()
	defer m.companyMu.Unlock()
	assetsDeleted, err := m.pg.Companies().DeleteCompanyWithAssets(id, deleteAssets)
	if err != nil {
		return 0, err
	}
	m.mu.Lock()
	for _, task := range m.tasks {
		task.updateLifecycle(func(state *taskLifecycleState) {
			companyIDs := make([]int64, 0, len(state.CompanyIDs))
			for _, companyID := range state.CompanyIDs {
				if companyID != id {
					companyIDs = append(companyIDs, companyID)
				}
			}
			state.CompanyIDs = companyIDs
		})
	}
	m.mu.Unlock()
	return assetsDeleted, nil
}

// ReplaceTaskLLMProfiles resets a task's ordered provider chain and mirrors the
// committed state onto the live task handle. Terminal tasks are editable too —
// their 主 Agent 对话 keeps running on the chain after the task finishes.
func (m *Manager) ReplaceTaskLLMProfiles(id string, profileIDs []int64, activeProfileID int64) (int64, error) {
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, err
	}
	if err := m.pg.ReplaceTaskLLMProfiles(n, profileIDs, activeProfileID); err != nil {
		return 0, err
	}
	pt, err := m.pg.GetTask(n)
	if err != nil || pt == nil {
		return 0, err
	}
	m.mu.Lock()
	if task := m.tasks[id]; task != nil {
		task.setLLMState(pt.LLMProfileID, pt.ActiveLLMProfileID, pt.LLMProfileIDs, pt.LLMChainRevision, pt.LLMFailoverState, pt.LLMFailoverReason)
	}
	m.mu.Unlock()
	// 终态任务不重开额度阻塞意图:任务已经没有 worker 在跑,重开只会把它们从
	// blocked 挪到 open——那里既没人执行,也不再满足「重跑意图」的可重跑条件,
	// 反而变成死状态。终态任务想接着跑,走重跑意图/新增目标,那条路会把任务重新
	// admit 回运行态。
	if pgdb.IsTerminal(pt.Status) {
		return 0, nil
	}
	if task, ok := m.Task(id); ok {
		reopened, reopenErr := task.Store.ReopenIntentsByBlockedReason(pgdb.IntentBlockedLLMQuota)
		if reopenErr != nil {
			return reopened, reopenErr
		}
		return reopened, nil
	}
	return 0, nil
}

// LoadExisting rebuilds in-memory task handles from the PG task registry.
// [한국어 함수 설명] DB 작업 목록으로 메모리 핸들을 다시 만든다. 이전 프로세스의 goroutine이나 컨텍스트를 복원하는 것이 아니라 저장된 상태로 새로 구성한다.
func (m *Manager) LoadExisting() []*Task {
	pts, err := m.pg.ListTasks()
	if err != nil {
		log.Printf("[manager] reload: %v", err)
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var loaded []*Task
	for _, pt := range pts {
		id := strconv.FormatInt(pt.ID, 10)
		if _, ok := m.tasks[id]; ok {
			continue
		}
		t := taskFromPG(pt, m.pg.Exploration(pt.ExplorationID), m.interceptor)
		m.tasks[id] = t
		loaded = append(loaded, t)
	}
	if m.active == "" {
		var newest *Task
		for _, t := range m.tasks {
			if newest == nil || t.CreatedAt > newest.CreatedAt {
				newest = t
			}
		}
		if newest != nil {
			m.active = newest.ID
		}
	}
	if len(loaded) > 0 {
		log.Printf("[manager] reloaded %d task(s) from PG", len(loaded))
	}
	return loaded
}

// SetTaskPaused persists a task's paused state.
func (m *Manager) SetTaskPaused(id string, paused bool) error {
	m.taskStateMu.Lock()
	defer m.taskStateMu.Unlock()
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return err
	}
	if err := m.pg.SetPaused(n, paused); err != nil {
		return err
	}
	m.mu.Lock()
	if t := m.tasks[id]; t != nil {
		t.updateLifecycle(func(state *taskLifecycleState) {
			state.Paused = paused
		})
	}
	m.mu.Unlock()
	return nil
}

// ApplyTaskAdmission atomically commits the lifecycle fields controlled by the
// concurrency scheduler. Keeping status, paused and queue metadata in one UPDATE
// prevents a failed resume from leaving a task half-revived (for example running
// but still user-paused, or dequeued without an Engine start).
//
// preservePosition applies only when the row is already queued. A repeated
// admission keeps its FIFO timestamp; a task that was explicitly paused and is
// now re-queued receives a fresh tail position.
// [한국어 함수 설명] 상태·pause·queue 정보를 같은 DB 변경으로 커밋하고 메모리를 따라가게 한다. 예상 상태 조건으로 다른 종료 경로와의 경쟁을 감지한다.
func (m *Manager) ApplyTaskAdmission(id, expectedStatus, status string, queued bool, mode string, preservePosition bool) error {
	m.taskStateMu.Lock()
	defer m.taskStateMu.Unlock()
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return err
	}
	if queued {
		if mode != "bootstrap" && mode != "resume" {
			return fmt.Errorf("invalid queue mode %q", mode)
		}
	} else {
		mode = ""
	}

	var queuedAt, completedAt, firstRunAt, deadlineAt sql.NullTime
	var committedMode string
	err = m.pg.QueryRow(`UPDATE tasks
	SET status=$2,
	    completed_at=CASE
	        WHEN $2 IN ('done','failed','timeout') THEN COALESCE(completed_at, now())
	        ELSE NULL
	    END,
	    paused=false,
	    queued=$3,
	    queued_at=CASE
	        WHEN NOT $3 THEN NULL
	        WHEN $5 AND queued THEN COALESCE(queued_at, now())
	        ELSE now()
	    END,
	    queue_mode=CASE
	        WHEN NOT $3 THEN ''
	        WHEN ($5 AND queued AND queue_mode='bootstrap') OR $4='bootstrap' THEN 'bootstrap'
	        ELSE 'resume'
	    END,
	    first_run_at=CASE
	        WHEN $6='timeout' AND $2 NOT IN ('done','failed','timeout') THEN NULL
	        ELSE first_run_at
	    END,
	    deadline_at=CASE
	        WHEN $6='timeout' AND $2 NOT IN ('done','failed','timeout') THEN NULL
	        ELSE deadline_at
	    END
	WHERE id=$1 AND deleted_at IS NULL AND status=$6
	RETURNING queued_at, queue_mode, completed_at, first_run_at, deadline_at`, n, status, queued, mode, preservePosition, expectedStatus).
		Scan(&queuedAt, &committedMode, &completedAt, &firstRunAt, &deadlineAt)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("task %s lifecycle changed before admission (expected status %q)", id, expectedStatus)
	}
	if err != nil {
		return err
	}

	m.mu.Lock()
	if t := m.tasks[id]; t != nil {
		t.updateLifecycle(func(state *taskLifecycleState) {
			state.Status = status
			state.Paused = false
			state.Queued = queued
			state.QueueMode = committedMode
			state.QueuedAt = 0
			if queuedAt.Valid {
				state.QueuedAt = queuedAt.Time.UnixNano()
			}
			state.CompletedAt = 0
			if completedAt.Valid {
				state.CompletedAt = completedAt.Time.Unix()
			}
			state.FirstRunAt = 0
			if firstRunAt.Valid {
				state.FirstRunAt = firstRunAt.Time.Unix()
			}
			state.DeadlineAt = 0
			if deadlineAt.Valid {
				state.DeadlineAt = deadlineAt.Time.Unix()
			}
		})
	}
	m.mu.Unlock()
	return nil
}

// ApplyTaskPause atomically removes a task from the admission queue and records
// the user pause. queue_mode is intentionally retained so resuming a never-run
// bootstrap task still performs goal decomposition, but the next enqueue receives
// a new queued_at timestamp and therefore moves to the FIFO tail.
// [한국어 함수 설명] FIFO에서 제거하는 동작과 사용자 pause를 함께 저장한다. bootstrap 필요 여부는 남겨 다음 재개 때 목표 분해를 생략하지 않게 한다.
func (m *Manager) ApplyTaskPause(id string) error {
	m.taskStateMu.Lock()
	defer m.taskStateMu.Unlock()
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return err
	}
	var mode string
	err = m.pg.QueryRow(`UPDATE tasks
		SET paused=true, queued=false, queued_at=NULL
		WHERE id=$1 AND deleted_at IS NULL AND paused=false
		  AND status NOT IN ('done','failed','timeout')
		RETURNING COALESCE(queue_mode,'')`, n).Scan(&mode)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("task %s is unavailable for pause", id)
	}
	if err != nil {
		return err
	}
	m.mu.Lock()
	if t := m.tasks[id]; t != nil {
		t.updateLifecycle(func(state *taskLifecycleState) {
			state.Paused = true
			state.Queued = false
			state.QueuedAt = 0
			state.QueueMode = mode
		})
	}
	m.mu.Unlock()
	return nil
}

// EnqueueTask persists the concurrency hold and syncs the in-memory handle.
func (m *Manager) EnqueueTask(id, mode string) error {
	m.taskStateMu.Lock()
	defer m.taskStateMu.Unlock()
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return err
	}
	if mode != "bootstrap" && mode != "resume" {
		return fmt.Errorf("invalid queue mode %q", mode)
	}
	var queuedAt time.Time
	var committedMode string
	err = m.pg.QueryRow(`UPDATE tasks
		SET queued=true,
		    queued_at=CASE WHEN queued THEN COALESCE(queued_at, now()) ELSE now() END,
		    queue_mode=CASE
		        WHEN queue_mode='bootstrap' OR $2='bootstrap' THEN 'bootstrap'
		        ELSE 'resume'
		    END
		WHERE id=$1 AND deleted_at IS NULL
		RETURNING queued_at, queue_mode`, n, mode).Scan(&queuedAt, &committedMode)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("task %s is unavailable for enqueue", id)
	}
	if err != nil {
		return err
	}
	m.mu.Lock()
	if t := m.tasks[id]; t != nil {
		t.updateLifecycle(func(state *taskLifecycleState) {
			state.QueuedAt = queuedAt.UnixNano()
			state.Queued = true
			state.QueueMode = committedMode
		})
	}
	m.mu.Unlock()
	return nil
}

// DequeueTask removes the concurrency hold. clearMode=false is used when a user
// pauses a queued task so a later resume still knows whether bootstrap is needed.
func (m *Manager) DequeueTask(id string, clearMode bool) error {
	m.taskStateMu.Lock()
	defer m.taskStateMu.Unlock()
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return err
	}
	if err := m.pg.Dequeue(n, clearMode); err != nil {
		return err
	}
	m.mu.Lock()
	if t := m.tasks[id]; t != nil {
		t.updateLifecycle(func(state *taskLifecycleState) {
			state.Queued = false
			state.QueuedAt = 0
			if clearMode {
				state.QueueMode = ""
			}
		})
	}
	m.mu.Unlock()
	return nil
}

// TaskStatus returns a task's current in-memory status (empty if unknown).
func (m *Manager) TaskStatus(id string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if t := m.tasks[id]; t != nil {
		return t.lifecycleSnapshot().Status
	}
	return ""
}

// StampTaskFirstRun stamps first_run_at + deadline_at on the first real run (idempotent
// in DB) and mirrors deadline_at on the live handle. Returns the deadline unix (0 = 不限).
// [한국어 함수 설명] DB에서 첫 실제 실행 시각을 한 번만 기록하고 deadline을 메모리로 복사한다. 재시작 뒤에도 절대 종료 시각을 이어서 사용할 수 있다.
func (m *Manager) StampTaskFirstRun(id string) (int64, error) {
	m.taskStateMu.Lock()
	defer m.taskStateMu.Unlock()
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, err
	}
	m.mu.RLock()
	timeout := 0
	if t := m.tasks[id]; t != nil {
		timeout = t.TimeoutSeconds
	}
	m.mu.RUnlock()
	dl, err := m.pg.StampFirstRun(n, timeout)
	if err != nil {
		return 0, err
	}
	var dlUnix int64
	if dl != nil {
		dlUnix = dl.Unix()
	}
	m.mu.Lock()
	if t := m.tasks[id]; t != nil {
		t.updateLifecycle(func(state *taskLifecycleState) {
			if state.FirstRunAt == 0 {
				state.FirstRunAt = time.Now().Unix()
			}
			state.DeadlineAt = dlUnix
		})
	}
	m.mu.Unlock()
	return dlUnix, nil
}

// SetTaskStatusGuarded sets a TERMINAL status only if the task isn't already terminal
// (resolves the completed↔timeout race — first terminal writer wins). Reflects the
// won status on the live handle. won=false means another terminal already stuck.
// [한국어 함수 설명] 이미 종료 상태인 작업을 다른 종료 상태로 덮어쓰지 않는 조건부 기록이다. 정상 완료와 timeout이 동시에 결론을 내릴 때 먼저 기록된 상태를 보존한다.
func (m *Manager) SetTaskStatusGuarded(id, status string) (won bool, err error) {
	m.taskStateMu.Lock()
	defer m.taskStateMu.Unlock()
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return false, err
	}
	won, err = m.pg.SetTerminalStatusGuarded(n, status)
	if err != nil || !won {
		return won, err
	}
	m.mu.Lock()
	if t := m.tasks[id]; t != nil {
		t.updateLifecycle(func(state *taskLifecycleState) {
			state.Status = status
			if state.CompletedAt == 0 {
				state.CompletedAt = time.Now().Unix()
			}
		})
	}
	m.mu.Unlock()
	return true, nil
}

// SetTaskStatus persists a task's lifecycle status (e.g. "done") and reflects it
// on the in-memory handle so the derived DTO status shows it without a reload.
func (m *Manager) SetTaskStatus(id, status string) error {
	m.taskStateMu.Lock()
	defer m.taskStateMu.Unlock()
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return err
	}
	if err := m.pg.SetStatus(n, status); err != nil {
		return err
	}
	m.mu.Lock()
	if t := m.tasks[id]; t != nil {
		t.updateLifecycle(func(state *taskLifecycleState) {
			state.Status = status
			// Mirror the DB's completed_at stamp on the live handle so the DTO shows
			// the finish time without a reload (terminal -> stamp once; else clear).
			if pgdb.IsTerminal(status) {
				if state.CompletedAt == 0 {
					state.CompletedAt = time.Now().Unix()
				}
			} else {
				state.CompletedAt = 0
			}
		})
	}
	m.mu.Unlock()
	return nil
}

// DeleteTask removes a task and optionally its related global data. Traffic has
// no task-id column, so related exchanges are resolved by exact hosts from the
// task's asset rows. Files are staged before the database operation; traffic is
// staged while PostgreSQL excludes asset/anchor writers. Both are restored on a
// database failure and purged only after its commit.
// [한국어 함수 설명] 작업 외부 데이터의 선택적 정리 범위를 계산하고 파일/트래픽 staging 후 DB 삭제를 시도한다. 커밋 전 실패는 복원, 커밋 후 정리 실패는 별도 오류로 구분한다.
func (m *Manager) DeleteTask(id string, opts DeleteTaskOptions) (DeleteTaskResult, error) {
	result := DeleteTaskResult{Deleted: id}
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return result, err
	}
	registered, err := m.pg.GetTask(n)
	if err != nil {
		return result, err
	}
	if registered == nil {
		m.forgetTask(id, n)
		return result, nil
	}

	var fileStage *taskFileDeleteStage
	if opts.DeleteFiles {
		fileStage, err = stageTaskFiles(m.dir, id, registered.ExplorationID)
		if err != nil {
			return result, err
		}
		result.FilesDeleted = fileStage.deleted
	}

	var trafficStage *traffic.HostDeleteStage
	var prepare func(pgdb.TaskDeletePreparation) error
	if opts.DeleteTraffic && m.traffic != nil {
		prepare = func(p pgdb.TaskDeletePreparation) error {
			if len(p.TrafficHosts) == 0 {
				return nil
			}
			trafficStage, err = m.traffic.StageDeleteHostsExact(p.TrafficHosts)
			if err != nil {
				return err
			}
			result.TrafficDeleted = trafficStage.Deleted()
			return nil
		}
	}

	dbResult, err := m.pg.DeleteTaskCascadePrepared(
		n, opts.DeleteAssets, opts.DeleteFindings, opts.DeleteLLMRecords, prepare,
	)
	if err != nil {
		return result, rollbackTaskDelete(err, trafficStage, fileStage)
	}
	result.AssetsDeleted = dbResult.AssetsDeleted
	result.AssetsDetached = dbResult.AssetsDetached
	result.FindingsDeleted = dbResult.FindingsDeleted
	result.LLMRecordsDeleted = dbResult.LLMRecordsDeleted

	// PostgreSQL is now authoritative: finalize the staged external deletion and
	// forget the live task even if a final purge reports an error. Such errors are
	// typed so the HTTP layer can still tear down the task runtime instead of
	// incorrectly reviving a task whose database row is already gone.
	var finalizeErrs []error
	if trafficStage != nil {
		if err := trafficStage.Commit(); err != nil {
			finalizeErrs = append(finalizeErrs, fmt.Errorf("finalize traffic deletion: %w", err))
		}
	}
	if fileStage != nil {
		if err := fileStage.commit(); err != nil {
			finalizeErrs = append(finalizeErrs, fmt.Errorf("finalize task file deletion: %w", err))
		}
	}
	m.forgetTask(id, n)
	if err := errors.Join(finalizeErrs...); err != nil {
		return result, &taskDeleteCommittedError{err: err}
	}
	return result, nil
}

// taskDeleteCommittedError means PostgreSQL deletion succeeded but purging one
// of the recoverable staging directories failed. The task must stay deleted.
type taskDeleteCommittedError struct{ err error }

func (e *taskDeleteCommittedError) Error() string {
	return "task deletion committed; external cleanup incomplete: " + e.err.Error()
}

func (e *taskDeleteCommittedError) Unwrap() error { return e.err }

func rollbackTaskDelete(cause error, trafficStage *traffic.HostDeleteStage, fileStage *taskFileDeleteStage) error {
	errs := []error{cause}
	// Reverse the preparation order. Both restorations are attempted even if the
	// first one fails, and errors.Join preserves the original PostgreSQL error.
	if trafficStage != nil {
		if err := trafficStage.Rollback(); err != nil {
			errs = append(errs, fmt.Errorf("restore traffic after task delete failure: %w", err))
		}
	}
	if fileStage != nil {
		if err := fileStage.rollback(); err != nil {
			errs = append(errs, fmt.Errorf("restore task files after task delete failure: %w", err))
		}
	}
	return errors.Join(errs...)
}

func (m *Manager) forgetTask(id string, numericID int64) {
	m.mu.Lock()
	delete(m.tasks, id)
	for _, task := range m.tasks {
		task.updateLifecycle(func(state *taskLifecycleState) {
			kept := make([]int64, 0, len(state.SourceTaskIDs))
			for _, sourceID := range state.SourceTaskIDs {
				if sourceID != numericID {
					kept = append(kept, sourceID)
				}
			}
			state.SourceTaskIDs = kept
		})
	}
	if m.active == id {
		m.active = ""
		for _, t := range m.tasks {
			m.active = t.ID
			break
		}
	}
	m.mu.Unlock()
}

type stagedTaskPath struct {
	source string
	staged string
}

type taskFileDeleteStage struct {
	stageDir string
	moves    []stagedTaskPath
	deleted  bool
	done     bool
}

// stageTaskFiles atomically renames the task workspace and owned transcripts to
// a same-filesystem staging directory. The trailing dash in the transcript
// prefix is significant: exploration 12 must not match exploration 123.
// [한국어 함수 설명] 작업 디렉터리와 해당 exploration의 transcript 접두사만 staging으로 이동한다. exp12와 exp123을 혼동하지 않도록 경계 문자를 포함해 선택한다.
func stageTaskFiles(dataDir, taskID string, explorationID int64) (*taskFileDeleteStage, error) {
	stage := &taskFileDeleteStage{}
	var targets []string
	taskDir := filepath.Join(dataDir, "tasks", taskID)
	if _, err := os.Lstat(taskDir); err == nil {
		targets = append(targets, taskDir)
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	transcriptDir := filepath.Join(dataDir, "transcripts")
	entries, err := os.ReadDir(transcriptDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		entries = nil
	}
	prefix := fmt.Sprintf("exp%d-", explorationID)
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) || (!entry.IsDir() && !strings.HasSuffix(name, ".jsonl")) {
			continue
		}
		targets = append(targets, filepath.Join(transcriptDir, name))
	}
	if len(targets) == 0 {
		stage.done = true
		return stage, nil
	}

	parent := filepath.Join(dataDir, ".delete-staging")
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return nil, err
	}
	stage.stageDir, err = os.MkdirTemp(parent, "task-"+taskID+"-")
	if err != nil {
		return nil, err
	}
	for _, source := range targets {
		staged := filepath.Join(stage.stageDir, fmt.Sprintf("%d-%s", len(stage.moves), filepath.Base(source)))
		if err := os.Rename(source, staged); err != nil {
			cause := fmt.Errorf("stage task file %s: %w", source, err)
			if restoreErr := stage.rollback(); restoreErr != nil {
				return nil, errors.Join(cause, fmt.Errorf("restore partially staged task files: %w", restoreErr))
			}
			return nil, cause
		}
		stage.moves = append(stage.moves, stagedTaskPath{source: source, staged: staged})
	}
	stage.deleted = true
	return stage, nil
}

func (s *taskFileDeleteStage) commit() error {
	if s == nil || s.done {
		return nil
	}
	err := os.RemoveAll(s.stageDir)
	s.done = true
	return err
}

func (s *taskFileDeleteStage) rollback() error {
	if s == nil || s.done {
		return nil
	}
	var errs []error
	for i := len(s.moves) - 1; i >= 0; i-- {
		move := s.moves[i]
		if _, err := os.Lstat(move.source); err == nil {
			errs = append(errs, fmt.Errorf("restore destination already exists: %s", move.source))
			continue
		} else if !os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("inspect restore destination %s: %w", move.source, err))
			continue
		}
		if err := os.MkdirAll(filepath.Dir(move.source), 0o755); err != nil {
			errs = append(errs, fmt.Errorf("create restore parent for %s: %w", move.source, err))
			continue
		}
		if err := os.Rename(move.staged, move.source); err != nil {
			errs = append(errs, fmt.Errorf("restore %s: %w", move.source, err))
		}
	}
	if len(errs) == 0 && s.stageDir != "" {
		if err := os.RemoveAll(s.stageDir); err != nil {
			errs = append(errs, fmt.Errorf("remove task file stage: %w", err))
		}
	}
	s.done = true
	return errors.Join(errs...)
}

// deleteTaskFiles retains the standalone helper contract used by focused tests.
func deleteTaskFiles(dataDir, taskID string, explorationID int64) (bool, error) {
	stage, err := stageTaskFiles(dataDir, taskID, explorationID)
	if err != nil {
		return false, err
	}
	deleted := stage.deleted
	if err := stage.commit(); err != nil {
		return deleted, err
	}
	return deleted, nil
}

func (m *Manager) Task(id string) (*Task, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tasks[id]
	return t, ok
}

// ActiveTask returns the currently active task (or nil).
func (m *Manager) ActiveTask() *Task {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.active == "" {
		return nil
	}
	return m.tasks[m.active]
}

// SetActive switches the active task. Returns false if the id is unknown.
func (m *Manager) SetActive(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.tasks[id]; !ok {
		return false
	}
	m.active = id
	return true
}

func (m *Manager) ResolveTask(id string) *Task {
	if id == "" || id == "active" {
		return m.ActiveTask()
	}
	t, _ := m.Task(id)
	return t
}

func (m *Manager) List() []*Task {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Task, 0, len(m.tasks))
	for _, t := range m.tasks {
		out = append(out, t)
	}
	// 置顶任务优先，组内按置顶时间倒序；普通任务按 id 倒序。m.tasks 是 map，
	// 每次轮询都必须重排，id 则为同刻创建任务提供稳定且唯一的兜底顺序。
	sort.Slice(out, func(i, j int) bool {
		iState := out[i].lifecycleSnapshot()
		jState := out[j].lifecycleSnapshot()
		if (iState.PinnedAt > 0) != (jState.PinnedAt > 0) {
			return iState.PinnedAt > 0
		}
		if iState.PinnedAt != jState.PinnedAt {
			return iState.PinnedAt > jState.PinnedAt
		}
		ai, _ := strconv.ParseInt(out[i].ID, 10, 64)
		aj, _ := strconv.ParseInt(out[j].ID, 10, 64)
		return ai > aj
	})
	return out
}

// Notify signals that the asset/exploration graph changed (debounced consumer
// wakes the planner). Non-blocking.
// [한국어 함수 설명] 크기 제한 채널에 비차단 신호를 보낸다. 이미 신호가 대기 중이면 하나 더 보내지 않아 연속 변경을 한 번의 Planner 깨어남으로 합칠 수 있다.
func (t *Task) Notify() {
	select {
	case t.notify <- struct{}{}:
	default:
	}
}

// NotifyDone is Notify plus a hint: a worker just finished intentID and that is
// what triggered this wake-up. The planner reads the accumulated triggers next
// round so it can spell out which intent finished (+ its output). Events pile up
// (debounce) until the round drains them via drainTriggers.
// [한국어 함수 설명] 끝난 intent ID를 구체적인 trigger로 누적한 뒤 Notify한다. Planner는 다음 라운드에 어떤 실행 결과 때문에 깨어났는지 알 수 있다.
func (t *Task) NotifyDone(intentID int64) {
	if intentID > 0 {
		t.trigMu.Lock()
		t.pendingTriggers = append(t.pendingTriggers, agent.TriggerEvent{Kind: "done", IntentID: intentID})
		t.trigMu.Unlock()
	}
	t.Notify()
}

// NotifyFinding records that a worker reported a finding on intentID (summary),
// then wakes the planner — so the round spells out which intent found what.
// [한국어 함수 설명] 의도 ID와 발견 요약을 누적해 다음 Planner 문맥에 전달한다. 발견 저장 자체와 이 깨우기 신호는 구분되는 동작이다.
func (t *Task) NotifyFinding(intentID int64, summary string) {
	t.trigMu.Lock()
	t.pendingTriggers = append(t.pendingTriggers, agent.TriggerEvent{Kind: "finding", IntentID: intentID, Detail: summary})
	t.trigMu.Unlock()
	t.Notify()
}

// NotifyGoal records that one OR MORE goals were added in a single set_goals call —
// by the human via the main agent — then wakes the planner, so the next round spells
// out "人新增了 N 个目标：…" instead of the planner having to spot new open goals in
// the overview. One call → one trigger event (set_goals 的一次批量算一条，不逐条刷屏).
// The event survives an early-returning terminal round (drain happens after the gate),
// so a set_goals that revives a done task still surfaces it once the task is running.
func (t *Task) NotifyGoal(texts []string) {
	if len(texts) == 0 {
		return
	}
	t.trigMu.Lock()
	t.pendingTriggers = append(t.pendingTriggers, agent.TriggerEvent{Kind: "goal", Goals: texts})
	t.trigMu.Unlock()
	t.Notify()
}

// NotifyHint records that one OR MORE hints were added in a single add_hint call —
// by the human via the main agent, or by cross-task orchestration — then wakes the
// planner, so the next round is told "人新增了 N 条战略提示：…" and looks at them
// directly instead of having to spot the new hint folded into the graph overview.
// One call → one trigger event (a batched add_hint counts as one, not one per hint).
func (t *Task) NotifyHint(texts []string) {
	if len(texts) == 0 {
		return
	}
	t.trigMu.Lock()
	t.pendingTriggers = append(t.pendingTriggers, agent.TriggerEvent{Kind: "hint", Hints: texts})
	t.trigMu.Unlock()
	t.Notify()
}

// NotifyGoalDeleted records that the human deleted a goal (via 总览的目标管理), then
// wakes the planner so the next round spells out which goal was removed. The event
// survives an early-returning terminal round (drain happens after the gate).
func (t *Task) NotifyGoalDeleted(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	t.trigMu.Lock()
	t.pendingTriggers = append(t.pendingTriggers, agent.TriggerEvent{Kind: "goal_deleted", Detail: text})
	t.trigMu.Unlock()
	t.Notify()
}

// NotifyGoalEdited records that the human edited a goal (via 总览的目标管理), then wakes
// the planner so the next round spells out the old→new change. The event survives an
// early-returning terminal round (drain happens after the gate).
func (t *Task) NotifyGoalEdited(oldText, newText string) {
	oldText, newText = strings.TrimSpace(oldText), strings.TrimSpace(newText)
	if newText == "" {
		return
	}
	t.trigMu.Lock()
	t.pendingTriggers = append(t.pendingTriggers, agent.TriggerEvent{Kind: "goal_edited", OldGoal: oldText, NewGoal: newText})
	t.trigMu.Unlock()
	t.Notify()
}

// NotifyCancelled records that the human deleted intentID (reason = 删除原因), then
// wakes the planner so the next round spells out which intent was removed and why.
// summary is the intent's text captured before deletion — needed for hard delete,
// where the node is gone by the time the planner reads the trigger. Applies to both
// soft (state='deleted') and hard (physical cascade) delete.
func (t *Task) NotifyCancelled(intentID int64, summary, reason string) {
	if intentID > 0 {
		t.trigMu.Lock()
		t.pendingTriggers = append(t.pendingTriggers, agent.TriggerEvent{Kind: "cancelled", IntentID: intentID, Summary: summary, Detail: reason})
		t.trigMu.Unlock()
	}
	t.Notify()
}

// drainTriggers returns and clears the trigger events accumulated since the last round.
// [한국어 함수 설명] 잠금 아래 누적 원인 목록을 가져가고 비운다. debounce 동안 여러 Worker가 만든 변경을 한 라운드 입력으로 묶는 지점이다.
func (t *Task) drainTriggers() []agent.TriggerEvent {
	t.trigMu.Lock()
	defer t.trigMu.Unlock()
	ev := t.pendingTriggers
	t.pendingTriggers = nil
	return ev
}
