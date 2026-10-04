// Package intercept implements the user-configurable tool-call interception layer.
// Rules are loaded from the database, cached in memory, and evaluated in priority
// order (highest first) on every PreToolUse event. Three actions are supported:
//
//   - allow: immediately permits the call, skipping lower-priority rules.
//   - deny:  blocks the call and returns a message to the model.
//   - ask:   blocks the call, creates an intercept_pending record, writes an
//     activity to the active conversation, then waits for the user to approve or
//     deny via the /api/intercept/pending/{id}/decide endpoint.
//
// The timeout behaviour is configurable at runtime via SetTimeoutConfig.
// [한국어 파일 안내] intercept/intercept.go
// 도구 호출 승인 상태를 DB 규칙 → 선택적 LLM 판정 → 필요 시 사람의 결정 순서로 처리한다.
// 설정은 PostgreSQL에서 읽고 규칙·활성 도구 목록은 메모리에 캐시한다. 규칙 편집 후 Invalidate로 갱신한다.
// allow/deny도 이미 결정된 이력으로 남기고 ask만 pending 채널을 만들어 호출을 대기시킨다.
// 기본 LLM 심사기는 꺼져 있고, 켜진 심사기의 오류 기본 정책은 allow다. 규칙 미일치가 자동 거부를 뜻하지 않는다.
// 이 패키지는 도구 hook에서 호출되는 승인 조정기이며 패킷 필터·프로세스 격리·발견 진위 검증기는 아니다.
package intercept

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Autumn-27/artex/db"
)

// ctxKey is the unexported context key type to avoid collisions.
type ctxKey int

// ConvIDKey stores the active conversation ID in a context.Context so the
// interceptor can associate "ask" pending records with the right conversation.
const ConvIDKey ctxKey = 0

// WithConvID returns a child context carrying convID.
// 한국어 해설: 승인 이력이 연결될 대화 ID를 context에 넣는다. 모델 입력 문자열에 ID를 섞지 않는다.
func WithConvID(ctx context.Context, convID int64) context.Context {
	return context.WithValue(ctx, ConvIDKey, convID)
}

// ConvIDFromContext extracts the conversation ID (0 if absent).
// 한국어 해설: context의 대화 ID를 읽고 없으면 0을 반환한다. 0인 배경 작업도 task 문맥으로 이력을 남길 수 있다.
func ConvIDFromContext(ctx context.Context) int64 {
	v, _ := ctx.Value(ConvIDKey).(int64)
	return v
}

// taskCtxKey is a separate unexported key type for task context values.
type taskCtxKey int

const (
	taskInfoCtxKey taskCtxKey = 1
	taskEmitCtxKey taskCtxKey = 2
)

type taskCtxInfo struct{ taskID, agentName string }

// WithTaskContext injects task metadata and an emit function into ctx so that
// HandleAsk can tag pending records and write intercept_request activities to
// the task's exploration stream (making them appear inline in session transcripts).
// 한국어 해설: 작업 ID·에이전트 이름과 활동 발행 함수를 context에 묶어 승인 카드를 올바른 작업 스트림에 연결한다.
func WithTaskContext(ctx context.Context, taskID, agentName string, emit func(db.Activity)) context.Context {
	ctx = context.WithValue(ctx, taskInfoCtxKey, taskCtxInfo{taskID, agentName})
	if emit != nil {
		ctx = context.WithValue(ctx, taskEmitCtxKey, emit)
	}
	return ctx
}

// 한국어 해설: 작업 출처가 명시된 context에서만 taskID와 agentName을 꺼낸다.
func taskInfoFromCtx(ctx context.Context) (taskID, agentName string) {
	if v, ok := ctx.Value(taskInfoCtxKey).(taskCtxInfo); ok {
		return v.taskID, v.agentName
	}
	return "", ""
}

// 한국어 해설: 배경 worker의 승인 요청을 작업 활동에 표시할 콜백을 꺼낸다.
func taskEmitFromCtx(ctx context.Context) func(db.Activity) {
	f, _ := ctx.Value(taskEmitCtxKey).(func(db.Activity))
	return f
}

// compiledRule is an InterceptRule with the regex pre-compiled (nil for string rules).
// 한국어 자료형: DB 규칙에 컴파일된 정규식을 붙인 캐시 항목이다. 정규식 외의 규칙은 re가 nil이다.
type compiledRule struct {
	db.InterceptRule
	re *regexp.Regexp
}

// pendingManager tracks in-flight "ask" requests via per-request channels.
// 한국어 자료형: 같은 프로세스 안에서 ask 호출을 기다리게 하는 채널 관리자다. 영속적인 승인 상태 자체는 DB가 보관한다.
type pendingManager struct {
	mu sync.Mutex
	ch map[int64]chan bool
}

// 한국어 해설: 현재 프로세스에서 대기 중인 승인 ID와 버퍼 채널을 관리할 map을 만든다.
func newPendingManager() *pendingManager { return &pendingManager{ch: map[int64]chan bool{}} }

// 한국어 해설: 승인 ID별 1칸 버퍼 채널을 등록한다. 사람이 결정한 결과를 대기 중인 호출에 한 번 전달한다.
func (p *pendingManager) add(id int64) chan bool {
	ch := make(chan bool, 1)
	p.mu.Lock()
	p.ch[id] = ch
	p.mu.Unlock()
	return ch
}

// 한국어 해설: map에서 대기 채널을 제거하고 허용 여부를 보낸다. mutex를 놓은 뒤 전송하여 다른 승인을 막지 않는다.
func (p *pendingManager) resolve(id int64, allowed bool) {
	p.mu.Lock()
	ch, ok := p.ch[id]
	delete(p.ch, id)
	p.mu.Unlock()
	if ok {
		ch <- allowed
	}
}

// 한국어 해설: 호출 종료·취소 후 해당 승인 ID의 메모리 대기 항목을 정리한다.
func (p *pendingManager) remove(id int64) {
	p.mu.Lock()
	delete(p.ch, id)
	p.mu.Unlock()
}

// Reviewer runs the LLM fallback judge for one tool call and returns its verdict
// as a Decision (Action ∈ allow|ask|deny; empty Action means the reply could not
// be parsed). It is injected by the server layer so the intercept package stays
// free of any llm dependency. profileID == 0 means "use the active/default profile".
// 한국어 자료형: LLM 라이브러리를 직접 의존하지 않도록 server에서 주입하는 판정 함수 계약이다.
type Reviewer func(ctx context.Context, profileID int64, prompt string, input ReviewInput) (Decision, error)

// Interceptor loads intercept rules from the database and evaluates them on
// tool calls. It is safe for concurrent use.
// 한국어 자료형: DB 설정 캐시와 동시 승인 요청을 관리한다. 상태 저장과 실제 모델 호출을 분리한다.
type Interceptor struct {
	db           *db.DB
	mu           sync.RWMutex
	cached       []compiledRule  // sorted by priority DESC; nil means not loaded yet
	enabledTools map[string]bool // nil means not loaded yet
	pending      *pendingManager
	reviewer     Reviewer // nil = LLM fallback judge not wired
}

// SetReviewer installs the LLM fallback judge callback. Passing nil disables it.
// 한국어 해설: server가 제공하는 모델 판정 함수를 주입한다. nil은 심사기 실행을 연결하지 않는 상태다.
func (i *Interceptor) SetReviewer(r Reviewer) {
	i.mu.Lock()
	i.reviewer = r
	i.mu.Unlock()
}

// defaultEnabledTools is the hard-coded set of tools that enter the intercept
// rule system when no intercept_enabled_tools setting has been saved.
var defaultEnabledTools = []string{
	"Bash", "WebFetch", "web_search",
	"shell_open", "shell_send",
	"Write", "Edit", "MultiEdit",
}

// New creates an Interceptor backed by d. The rule cache is lazy-loaded on
// first use.
// 한국어 해설: DB와 pending 관리자만 준비하고 규칙 읽기는 실제 사용 시점까지 미룬다.
func New(d *db.DB) *Interceptor {
	return &Interceptor{db: d, pending: newPendingManager()}
}

// Invalidate clears the in-memory rule cache and the enabled-tools cache.
// The next call to Match or IsToolEnabled will reload from the database.
// Call this after any CRUD operation on rules or tool config.
// 한국어 해설: 규칙과 활성 도구 캐시를 모두 비운다. 이후 호출이 새 DB 설정을 다시 읽게 한다.
func (i *Interceptor) Invalidate() {
	i.mu.Lock()
	i.cached = nil
	i.enabledTools = nil
	i.mu.Unlock()
}

// 한국어 해설: 활성 규칙의 정규식을 미리 컴파일하고 도구 목록 설정을 읽는다. 잘못된 정규식은 현재 구현에서 건너뛴다.
// 도구 설정 JSON이 손상되면 빈 활성 목록이 되어 규칙 경로가 적용되지 않을 수 있으므로 오류 정책과 함께 이해한다.
func (i *Interceptor) loadLocked() error {
	rules, err := i.db.ListInterceptRules()
	if err != nil {
		return err
	}
	var out []compiledRule
	for _, r := range rules {
		if !r.Enabled {
			continue
		}
		cr := compiledRule{InterceptRule: r}
		if r.MatchType == "regex" {
			re, err := regexp.Compile(r.Pattern)
			if err != nil {
				continue // skip rules with bad regex rather than crashing
			}
			cr.re = re
		}
		out = append(out, cr)
	}
	i.cached = out

	// Load enabled-tools set from settings, falling back to hard-coded defaults.
	val, ok, _ := i.db.GetSetting("intercept_enabled_tools")
	if !ok {
		m := make(map[string]bool, len(defaultEnabledTools))
		for _, n := range defaultEnabledTools {
			m[n] = true
		}
		i.enabledTools = m
	} else {
		var names []string
		if json.Unmarshal([]byte(val), &names) != nil {
			i.enabledTools = map[string]bool{}
		} else {
			m := make(map[string]bool, len(names))
			for _, n := range names {
				m[n] = true
			}
			i.enabledTools = m
		}
	}
	return nil
}

// 한국어 해설: 읽기 잠금으로 캐시를 먼저 확인하고 비어 있을 때만 쓰기 잠금 아래 이중 확인 후 로드한다.
func (i *Interceptor) rules() ([]compiledRule, error) {
	i.mu.RLock()
	if i.cached != nil {
		out := i.cached
		i.mu.RUnlock()
		return out, nil
	}
	i.mu.RUnlock()

	i.mu.Lock()
	defer i.mu.Unlock()
	if i.cached != nil {
		return i.cached, nil
	}
	if err := i.loadLocked(); err != nil {
		return nil, err
	}
	return i.cached, nil
}

// IsToolEnabled returns true if the named tool is in the intercept-enabled set
// (i.e. it should enter the rule-matching path). Uses the same double-check lock
// pattern as rules().
// 한국어 해설: 해당 이름이 승인 규칙 적용 대상인지 확인한다. 모든 도구가 자동으로 포함되는 구조가 아니다.
func (i *Interceptor) IsToolEnabled(name string) bool {
	i.mu.RLock()
	if i.enabledTools != nil {
		v := i.enabledTools[name]
		i.mu.RUnlock()
		return v
	}
	i.mu.RUnlock()

	i.mu.Lock()
	defer i.mu.Unlock()
	if i.enabledTools == nil {
		_ = i.loadLocked()
	}
	return i.enabledTools[name]
}

// GetEnabledTools returns the ordered list of tool names that are currently
// configured to enter the intercept rule system. When the setting has never been
// saved the hard-coded default list is returned.
// 한국어 해설: API에 보여 줄 활성 도구 배열을 반환한다. 저장값이 없으면 기본 목록, JSON 오류면 빈 목록을 사용한다.
func (i *Interceptor) GetEnabledTools() ([]string, error) {
	val, ok, err := i.db.GetSetting("intercept_enabled_tools")
	if err != nil {
		return nil, err
	}
	if !ok {
		out := make([]string, len(defaultEnabledTools))
		copy(out, defaultEnabledTools)
		return out, nil
	}
	var names []string
	if err := json.Unmarshal([]byte(val), &names); err != nil {
		return []string{}, nil
	}
	return names, nil
}

// SetEnabledTools persists the list of tool names that should enter the intercept
// rule system, then invalidates the cache so the next call picks up the new list.
// 한국어 해설: 활성 도구 이름 배열을 JSON으로 저장하고 캐시를 무효화하여 다음 실행에 반영한다.
func (i *Interceptor) SetEnabledTools(tools []string) error {
	b, err := json.Marshal(tools)
	if err != nil {
		return err
	}
	if err := i.db.SetSetting("intercept_enabled_tools", string(b)); err != nil {
		return err
	}
	i.Invalidate()
	return nil
}

// Decision is the outcome of a successful rule match.
// 한국어 자료형: 최종 행동과 함께 근거·규칙/모델 출처·입력/설정 해시·사람 승인 시간 제한을 전달한다.
type Decision struct {
	ModelInput       json.RawMessage
	ModelInputDigest string
	ModelFallback    bool
	RuleName         string
	ConfigDigest     string
	ProfileID        int64
	Action           string // "allow" | "deny" | "ask"
	Message          string
	RuleID           int64
	TimeoutEnabled   bool
	TimeoutSeconds   int
	TimeoutAction    string // "deny" | "allow"
}

// --- LLM fallback judge ---

// Judge settings keys (stored in the settings KV table). See docs §3.
const (
	settingJudgeEnabled          = "llm_judge_enabled"
	settingJudgeProfileID        = "llm_judge_profile_id"
	settingJudgePrompt           = "llm_judge_prompt"
	settingJudgeTimeoutSecs      = "llm_judge_timeout_seconds"
	settingJudgeFailAction       = "llm_judge_fail_action"
	settingJudgeAskTimeoutSecs   = "llm_judge_ask_timeout_seconds"
	settingJudgeAskTimeoutAction = "llm_judge_ask_timeout_action"
)

// Judge default values.
const (
	defaultJudgeTimeoutSecs      = 15
	defaultJudgeFailAction       = "allow"
	defaultJudgeAskTimeoutSecs   = 300
	defaultJudgeAskTimeoutAction = "deny"
)

// JudgeConfig is the resolved LLM-fallback-judge configuration. Prompt is always
// non-empty (falls back to DefaultJudgePrompt).
// 한국어 자료형: 저장된 KV 설정을 기본값까지 적용해 해석한 심사기 설정이다. ProfileID=0은 활성 기본 프로필을 따른다.
type JudgeConfig struct {
	Enabled           bool   `json:"enabled"`
	ProfileID         int64  `json:"profile_id"` // 0 = follow active/default
	Prompt            string `json:"prompt"`
	TimeoutSeconds    int    `json:"timeout_seconds"`
	FailAction        string `json:"fail_action"` // allow|ask|deny
	AskTimeoutSeconds int    `json:"ask_timeout_seconds"`
	AskTimeoutAction  string `json:"ask_timeout_action"` // allow|deny
}

// judgeConfig reads the judge configuration from settings, applying defaults for
// missing/invalid keys. Read fresh on each fallback judgement — the LLM call that
// follows dwarfs a few KV reads, and freshness avoids a cache-invalidation path.
// 한국어 해설: 판정마다 최신 설정을 읽어 누락·잘못된 값에 기본값을 적용한다. 기본 모델 대기 15초, 사람 대기 300초, 사람 시간 초과는 deny다.
func (i *Interceptor) judgeConfig() JudgeConfig {
	c := JudgeConfig{
		Enabled:           i.db.GetBool(settingJudgeEnabled, false),
		ProfileID:         int64(i.getSettingInt(settingJudgeProfileID, 0)),
		TimeoutSeconds:    i.getSettingInt(settingJudgeTimeoutSecs, defaultJudgeTimeoutSecs),
		FailAction:        i.getSettingChoice(settingJudgeFailAction, defaultJudgeFailAction, "allow", "ask", "deny"),
		AskTimeoutSeconds: i.getSettingInt(settingJudgeAskTimeoutSecs, defaultJudgeAskTimeoutSecs),
		AskTimeoutAction:  i.getSettingChoice(settingJudgeAskTimeoutAction, defaultJudgeAskTimeoutAction, "allow", "deny"),
	}
	// Prompt: stored value if non-empty, else the built-in template.
	if v, ok, _ := i.db.GetSetting(settingJudgePrompt); ok && strings.TrimSpace(v) != "" {
		c.Prompt = v
	} else {
		c.Prompt = DefaultJudgePrompt
	}
	if c.TimeoutSeconds <= 0 {
		c.TimeoutSeconds = defaultJudgeTimeoutSecs
	}
	if c.AskTimeoutSeconds <= 0 {
		c.AskTimeoutSeconds = defaultJudgeAskTimeoutSecs
	}
	return c
}

// 한국어 해설: 설정 문자열을 정수로 바꾸고 없거나 읽기/변환 실패이면 호출자가 정한 기본값을 사용한다.
func (i *Interceptor) getSettingInt(key string, def int) int {
	v, ok, err := i.db.GetSetting(key)
	if err != nil || !ok {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return def
	}
	return n
}

// 한국어 해설: 설정값을 주어진 선택지 중 하나로 제한한다. 값이 목록에 없으면 기본 선택을 반환한다.
func (i *Interceptor) getSettingChoice(key, def string, allowed ...string) string {
	v, ok, err := i.db.GetSetting(key)
	if err != nil || !ok {
		return def
	}
	v = strings.TrimSpace(v)
	for _, a := range allowed {
		if v == a {
			return v
		}
	}
	return def
}

// GetJudgeConfig returns the resolved judge configuration for the API/UI. Prompt
// is the effective prompt (built-in template when unset), so the UI can prefill.
// 한국어 해설: UI가 실제 적용될 모델 판정 설정과 기본/사용자 프롬프트를 확인할 수 있게 한다.
func (i *Interceptor) GetJudgeConfig() JudgeConfig { return i.judgeConfig() }

// SetJudgeConfig persists the judge configuration. An empty Prompt clears the
// override (the built-in template is used again).
// 한국어 해설: 심사기 설정 키들을 차례로 저장한다. 기본 프롬프트와 같은 값은 빈 override로 두어 이후 원본 기본값 갱신을 따르게 한다.
func (i *Interceptor) SetJudgeConfig(c JudgeConfig) error {
	if err := i.db.SetBool(settingJudgeEnabled, c.Enabled); err != nil {
		return err
	}
	if err := i.db.SetSetting(settingJudgeProfileID, strconv.FormatInt(c.ProfileID, 10)); err != nil {
		return err
	}
	// Store the prompt only when it differs from the built-in template, so version
	// updates to DefaultJudgePrompt flow through for users who never customized it.
	promptToStore := ""
	if strings.TrimSpace(c.Prompt) != "" && strings.TrimSpace(c.Prompt) != strings.TrimSpace(DefaultJudgePrompt) {
		promptToStore = c.Prompt
	}
	if err := i.db.SetSetting(settingJudgePrompt, promptToStore); err != nil {
		return err
	}
	if err := i.db.SetSetting(settingJudgeTimeoutSecs, strconv.Itoa(c.TimeoutSeconds)); err != nil {
		return err
	}
	if err := i.db.SetSetting(settingJudgeFailAction, c.FailAction); err != nil {
		return err
	}
	if err := i.db.SetSetting(settingJudgeAskTimeoutSecs, strconv.Itoa(c.AskTimeoutSeconds)); err != nil {
		return err
	}
	if err := i.db.SetSetting(settingJudgeAskTimeoutAction, c.AskTimeoutAction); err != nil {
		return err
	}
	return nil
}

// Judge runs the LLM fallback judge for a tool call that matched no rule. It
// returns (Decision, true) when the judge produced a terminal verdict, or
// (Decision{}, false) when the fallback is disabled or not wired (caller then
// keeps the current behavior: allow). On model error or an unparseable reply it
// falls back to the configured FailAction. Ask verdicts carry the human-approval
// timeout so the existing HandleAsk consumes them unchanged.
// 한국어 해설: 규칙이 맞지 않은 현재 도구 호출을 선택적으로 모델에 심사시킨다. 꺼짐/미연결이면 false를 반환한다.
// 유효하지 않은 인자는 ask, 모델 오류·파싱 불가 응답은 설정한 FailAction으로 처리하고 입력·설정 해시를 감사에 남긴다.
func (i *Interceptor) Judge(ctx context.Context, tool string, arguments json.RawMessage) (Decision, bool) {
	cfg := i.judgeConfig()
	i.mu.RLock()
	rv := i.reviewer
	i.mu.RUnlock()
	if !cfg.Enabled || rv == nil {
		return Decision{}, false
	}

	cctx := ctx
	if cfg.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		cctx, cancel = context.WithTimeout(ctx, time.Duration(cfg.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	input, contextErr := BuildReviewInput(cctx, tool, arguments)
	cfg.Prompt = EffectiveJudgePrompt(cfg.Prompt)
	var out Decision
	var err error
	var modelInput []byte
	if contextErr != nil {
		// Invalid current arguments cannot be reviewed faithfully, regardless of
		// the configured model-failure strategy. A human must resolve the input.
		out = Decision{Action: "ask", ModelFallback: true, Message: "审查上下文不完整，需要人工确认：" + contextErr.Error()}
	} else {
		modelInput, _ = json.Marshal(input)
		out, err = rv(cctx, cfg.ProfileID, cfg.Prompt, input)
	}
	if err != nil {
		out = Decision{ProfileID: out.ProfileID, ModelFallback: true, Action: cfg.FailAction, Message: "模型审批失败,按失败策略处理: " + err.Error()}
	}
	switch out.Action {
	case "allow", "ask", "deny":
		// valid verdict
	default:
		out = Decision{ProfileID: out.ProfileID, ModelFallback: true, Action: cfg.FailAction, Message: "模型输出无法解析,按失败策略处理"}
	}
	// A model verdict never carries a rule; keep RuleID 0 (→ NULL) for history.
	out.RuleID = 0
	if len(modelInput) > 0 {
		out.ModelInput = modelInput
		out.ModelInputDigest = digestInput(modelInput)
	}
	if out.ProfileID != 0 {
		cfg.ProfileID = out.ProfileID
	}
	out.ProfileID = cfg.ProfileID
	configJSON, _ := json.Marshal(cfg)
	out.ConfigDigest = digestInput(configJSON)
	if out.Message == "" {
		out.Message = "[模型] " + judgeActionLabel(out.Action)
	} else if !strings.HasPrefix(out.Message, "[模型]") {
		out.Message = "[模型] " + out.Message
	}
	if out.Action == "ask" {
		out.TimeoutEnabled = true
		out.TimeoutSeconds = cfg.AskTimeoutSeconds
		out.TimeoutAction = cfg.AskTimeoutAction
	}
	return out, true
}

// 한국어 해설: 내부 allow/ask/deny 값을 모델 판정 이력의 중국어 표시 문자열로 바꾼다. 식별자와 표시어는 별개다.
func judgeActionLabel(action string) string {
	switch action {
	case "allow":
		return "放行"
	case "deny":
		return "拦截"
	case "ask":
		return "转人工审批"
	default:
		return action
	}
}

// Match evaluates the rule list (priority DESC) against a tool call.
// Returns (Decision, true) for the first matching enabled rule, or
// (Decision{}, false) if no rule matches.
// 한국어 해설: DB가 정렬한 우선순위 순서에서 첫 번째 일치 규칙의 행동을 반환한다. 하위 규칙을 합산하거나 다수결하지 않는다.
func (i *Interceptor) Match(toolName string, input []byte) (Decision, bool) {
	rules, err := i.rules()
	if err != nil || len(rules) == 0 {
		return Decision{}, false
	}
	for _, r := range rules {
		if ruleMatches(r, toolName, input) {
			msg := r.Message
			if msg == "" {
				msg = defaultMessage(r.Action, r.Name)
			}
			configJSON, _ := json.Marshal(r.InterceptRule)
			return Decision{
				RuleName: r.Name, ConfigDigest: digestInput(configJSON),
				Action:         r.Action,
				Message:        msg,
				RuleID:         r.ID,
				TimeoutEnabled: r.TimeoutEnabled,
				TimeoutSeconds: r.TimeoutSeconds,
				TimeoutAction:  r.TimeoutAction,
			}, true
		}
	}
	return Decision{}, false
}

// 한국어 해설: 도구 이름 또는 전체 JSON 인자를 대상으로 정규식/부분 문자열 일치를 검사한다. JSON의 실제 명령 의미를 분석하는 함수는 아니다.
func ruleMatches(r compiledRule, toolName string, input []byte) bool {
	var subject string
	switch r.MatchTarget {
	case "tool_name":
		subject = toolName
	case "tool_input":
		subject = string(input)
	default:
		return false
	}
	if r.MatchType == "regex" {
		return r.re != nil && r.re.MatchString(subject)
	}
	return strings.Contains(subject, r.Pattern)
}

// 한국어 해설: 메시지를 직접 지정하지 않은 deny/ask 규칙에 이름이 포함된 기본 설명을 붙인다.
func defaultMessage(action, name string) string {
	switch action {
	case "deny":
		return "拦截规则 [" + name + "] 禁止执行此工具"
	case "ask":
		return "拦截规则 [" + name + "] 要求用户审批，请等待"
	default:
		return ""
	}
}

// Log records an allow/deny rule or model decision into intercept_pending as an ALREADY-decided
// row (status = "allowed" | "denied"), for observability. Unlike HandleAsk it does NOT
// block and needs no user action — it makes explicit review decisions visible on the history
// page (GET /api/intercept/history) and the task's intercept list. Best-effort: a DB
// error is swallowed so logging never changes the tool call's outcome. The pending list
// (status='pending') is unaffected, so it still shows only asks awaiting a decision.
// 한국어 해설: 이미 허용/거부된 판정을 이력에 남기고 정확히 연결된 도구 결과 콜백을 등록한다. DB 기록 실패는 실행 판정을 바꾸지 않는 best-effort 경로다.
func (i *Interceptor) Log(ctx context.Context, convID int64, dec Decision, toolName string, input []byte, status string) {
	taskID, agentName := taskInfoFromCtx(ctx)
	audit := auditFor(ctx, dec, input, status)
	id, err := i.db.CreateDecidedIntercept(dec.RuleID, convID, taskID, agentName, toolName, input, status, dec.Message, audit)
	if err == nil {
		i.bindResult(ctx, id, audit)
	}
}

// HandleAsk creates a pending approval record and blocks until the user decides
// (via /api/intercept/pending/{id}/decide) or the per-rule timeout elapses.
// Returns true if the user approved.
//
// convID == 0 means no active conversation (background pentest task). The
// pending record is still created (conversation_id = NULL) so the approvals
// page shows it and the sidebar badge lights up. The worker thread blocks just
// like in a chat session — the user must visit the approvals page to unblock it.
// 한국어 해설: pending 행을 만든 다음 메모리 채널을 등록하고 DB를 다시 읽어 등록 직전의 빠른 결정을 놓치지 않는다.
// 사람의 결정·context 취소·규칙별 시간 초과 중 먼저 온 결과를 처리하며 DB의 조건부 결정 결과를 재확인한다.
func (i *Interceptor) HandleAsk(ctx context.Context, convID int64, dec Decision, toolName string, input []byte) bool {
	ruleID := dec.RuleID
	taskID, agentName := taskInfoFromCtx(ctx)
	taskEmit := taskEmitFromCtx(ctx)

	audit := auditFor(ctx, dec, input, "pending")
	pendingID, err := i.db.CreateInterceptPending(ruleID, convID, taskID, agentName, toolName, input, dec.Message, audit)
	if err != nil {
		return false
	}

	i.bindResult(ctx, pendingID, audit)
	ch := i.pending.add(pendingID)
	defer i.pending.remove(pendingID)
	// Polling clients can decide after INSERT commits but before the channel is
	// registered. Re-read after registration so that decision cannot be lost;
	// later decisions will be delivered through ch.
	if saved, err := i.db.GetInterceptDetail(pendingID); err == nil && saved != nil && saved.Status != "pending" {
		return saved.Status == "allowed" || (saved.Audit != nil && saved.Audit.EffectiveAction == "allow")
	}

	detail, _ := json.Marshal(map[string]any{
		"pending_id": pendingID,
		"tool":       toolName,
		"input":      json.RawMessage(input),
	})
	activity := db.Activity{
		Kind:    "intercept_request",
		Summary: fmt.Sprintf("工具 %s 请求审批 (#%d)", toolName, pendingID),
		Detail:  string(detail),
	}

	if convID != 0 {
		// Chat session: write inline card to the conversation stream.
		_, _ = i.db.AppendConvActivity(convID, activity)
	} else if taskEmit != nil {
		// Task worker: emit to the exploration activity stream so it appears
		// inline in the session transcript (the emit fn stamps NodeID + Worker).
		taskEmit(activity)
	}

	if !dec.TimeoutEnabled {
		// No timeout: wait indefinitely until user decides or worker stops.
		select {
		case allowed := <-ch:
			return allowed
		case <-ctx.Done():
			_, _ = i.db.ResolveIntercept(pendingID, "denied", "deny", "工作已取消")
			_ = i.db.CompleteIntercept(pendingID, audit.RunID, audit.ToolUseID, "not_executed", "执行前工作已取消", false)
			return false
		}
	}

	secs := dec.TimeoutSeconds
	if secs <= 0 {
		secs = 60
	}
	timer := time.NewTimer(time.Duration(secs) * time.Second)
	defer timer.Stop()
	select {
	case allowed := <-ch:
		return allowed
	case <-timer.C:
		allowed := dec.TimeoutAction == "allow"
		action := "deny"
		if allowed {
			action = "allow"
		}
		resolved, err := i.db.ResolveIntercept(pendingID, "timeout", action, "审批超时，按超时策略处理")
		if err != nil {
			return false
		}
		if !resolved {
			detail, err := i.db.GetInterceptDetail(pendingID)
			return err == nil && detail != nil && (detail.Status == "allowed" || (detail.Audit != nil && detail.Audit.EffectiveAction == "allow"))
		}
		return allowed
	case <-ctx.Done():
		_, _ = i.db.ResolveIntercept(pendingID, "denied", "deny", "工作已取消")
		_ = i.db.CompleteIntercept(pendingID, audit.RunID, audit.ToolUseID, "not_executed", "执行前工作已取消", false)
		return false
	}
}

var ErrAlreadyDecided = errors.New("审批已处理或不存在，请刷新记录")

// Decide resolves a pending request. Called by the HTTP decide endpoint.
// 한국어 해설: API로 들어온 사람의 결정을 DB에서 먼저 확정한 후 대기 중인 채널을 깨운다. 이미 처리된 ID는 ErrAlreadyDecided로 구분한다.
func (i *Interceptor) Decide(pendingID int64, allowed bool) error {
	status := "denied"
	if allowed {
		status = "allowed"
	}
	action, reason := "deny", "人工拒绝执行"
	if allowed {
		action, reason = "allow", "人工允许执行"
	}
	resolved, err := i.db.ResolveIntercept(pendingID, status, action, reason)
	if err != nil {
		return err
	}
	if !resolved {
		return ErrAlreadyDecided
	}
	i.pending.resolve(pendingID, allowed)
	return nil
}
