// [한국어 길잡이] Skill 호출 계량용 래퍼
// 원래 Skill 도구를 감싸 호출 입력에서 Skill 이름을 읽고 RunInfo의 작업·세션·역할에 사용량을 귀속한다.
// 조회 가능한 Skill 메타데이터로 기록을 보강하되 원 도구의 실행 결과·스키마·권한 동작을 유지한다.
// 도구 사용 횟수는 과제 성공률이나 Skill 내용의 정확도를 의미하지 않는다.
// 전체 모듈 설명과 파일별 읽기 순서: docs/ko/modules/server.md
package server

import (
	"context"
	"encoding/json"
	"log"
	"path/filepath"
	"strings"

	"github.com/Autumn-27/norma/skill"
	actool "github.com/Autumn-27/norma/tool"

	"github.com/Autumn-27/artex/agent"
	"github.com/Autumn-27/artex/db"
)

// maxLedgerSkillName caps the skill name stored for a MISS — that string comes
// straight from the model and is otherwise unbounded.
const maxLedgerSkillName = 128

// meteredSkill wraps the Skill meta-tool to append one skill_usage row per call.
// The metering lives here rather than in Registry.OnInvoke because OnInvoke only
// receives the resolved Skill: it never fires for an unknown skill name (norma
// returns a tool error before the hook) and it cannot see the caller's args. Both
// matter — a miss tells you which procedure the agents wished existed.
type meteredSkill struct {
	actool.CoreTool
	pg       *db.DB
	reg      *skill.Registry
	agentKey string
	ri       agent.RunInfo
}

// meterSkillTool returns t unchanged when there is no DB to meter into.
func meterSkillTool(t actool.CoreTool, pg *db.DB, reg *skill.Registry, agentKey string, ri agent.RunInfo) actool.CoreTool {
	if pg == nil {
		return t
	}
	return &meteredSkill{CoreTool: t, pg: pg, reg: reg, agentKey: agentKey, ri: ri}
}

func (m *meteredSkill) Call(ctx context.Context, input json.RawMessage, tc *actool.ToolContext) (actool.Result, error) {
	m.record(input)
	return m.CoreTool.Call(ctx, input, tc)
}

// record appends the ledger row. Best-effort by design: a metering failure must
// never turn into a failed skill invocation, so every error path just logs.
// [한국어 함수 설명] Skill 이름을 입력에서 읽어 실제 레지스트리 메타데이터와 RunInfo를 이용해 사용량 행을 만든다. 계량 실패는 원 Skill 호출을 막지 않는다.
func (m *meteredSkill) record(input json.RawMessage) {
	var in struct {
		Name string `json:"name"`
		Args string `json:"args"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return // malformed call; norma will reject it too
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return
	}
	// Resolve to the directory name so the ledger keys match agent_skill_visibility
	// and the skills page (a skill's display Name can differ from its directory).
	found := false
	if m.reg != nil {
		if s, ok := m.reg.Get(name); ok {
			found = true
			if s.Dir != "" {
				name = filepath.Base(s.Dir)
			}
		}
	}
	if len(name) > maxLedgerSkillName {
		name = name[:maxLedgerSkillName]
	}
	err := m.pg.InsertSkillUsage(&db.SkillUsage{
		Skill:         name,
		AgentKey:      m.agentKey,
		TaskID:        m.ri.TaskID,
		ExplorationID: m.ri.ExplorationID,
		IntentID:      m.ri.IntentID,
		SessionID:     m.ri.SessionID,
		ArgsLen:       len(in.Args),
		Found:         found,
	})
	if err != nil {
		log.Printf("[skillusage] insert: %v", err)
	}
}
