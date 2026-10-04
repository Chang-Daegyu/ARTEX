package agent

// 한국어 파일 해설: agent/assembly.go
// 역할별 도구 목록을 실제 실행 가능한 도구 묶음으로 조립한다.
// 입력은 역할 키(agentKey)와 기본 CoreTool 목록이며, 서버가 주입한 Skill/MCP 도구를 합친다.
// 그 뒤 ToolResolve로 DB의 활성화·역할 바인딩·설명·기본값을 반영하고,
// findingWorkflowTools로 증거 도구 계약을 맞춘 다음 모든 도구에 panic 복구 래퍼를 씌운다.
// DeferredInfo는 이름 노출, 스키마 지연 로딩, Skill 해제 상태를 세션 생성부로 전달한다.
// 반환된 cleanup은 MCP 연결 등의 자원을 닫으므로 호출자가 반드시 defer해야 한다.
// guardPanic은 프로세스 충돌 방지용이며, 권한 심사나 외부 명령 격리 기능은 아니다.

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime/debug"

	actool "github.com/Autumn-27/norma/tool"
)

// DeferredInfo carries the deferred-tools wiring an agent needs to build its
// Options: the MCP tool names whose schemas are withheld, the subset listed in the
// global system-prompt block (non-skill-gated), and the shared session unlock set.
// UnlockSkill unlocks a named skill's MCPs — hosts call it to rebuild the unlock set
// from history on a resumed session (design doc C2).
type DeferredInfo struct {
	FindingGuidance string            // derived from the final permitted tools, including DB overrides
	Deferred        []string          // all MCP tool names (schema withheld)
	GlobalNames     []string          // MCP names to list in the system-prompt block
	Unlock          *actool.UnlockSet // shared call-gate; nil when no MCP tools
	UnlockSkill     func(skillName string)
}

// ToolAugment, if set, returns the EXTRA tools an agent should see beyond its
// built-in base set — the agent's visible skills (packed into one Skill meta-tool)
// and visible MCP servers (expanded to mcp__server__tool). It also returns the
// DeferredInfo describing how those MCP tools are deferred/gated. The server wires
// it to the PG agent_visibility table. cleanup releases any spawned MCP clients.
//
// When nil, agents run with only their built-in tools — behavior is unchanged
// until a user assigns a skill/MCP to the agent in the UI.
var ToolAugment func(ctx context.Context, agentKey string) (extra []actool.CoreTool, def DeferredInfo, cleanup func())

// AugmentTools returns base plus the agent's visible skill/MCP tools, the
// DeferredInfo, and a cleanup func the caller must defer (closes MCP clients).
// Built-in base tools are kept as-is — never filtered (内置工具留代码层，不做可见性过滤).
// 한국어: 조립 순서가 중요하다: 추가 도구 → DB 최종 바인딩/기본값 → 증거 계약 → panic 복구이다.
// 한국어: cleanup은 성공 여부와 무관하게 호출자가 실행해야 연결 자원이 누수되지 않는다.
func AugmentTools(ctx context.Context, agentKey string, base []actool.CoreTool) ([]actool.CoreTool, DeferredInfo, func()) {
	var (
		def     DeferredInfo
		cleanup = func() {}
		out     = base
	)
	if ToolAugment != nil {
		var extra []actool.CoreTool
		var cl func()
		extra, def, cl = ToolAugment(ctx, agentKey)
		if cl != nil {
			cleanup = cl
		}
		if len(extra) > 0 {
			out = append(append([]actool.CoreTool{}, base...), extra...)
		}
	}
	// DB tools table has the final say on the built-in tools: drop the ones this
	// agent isn't bound to (or that are disabled) and swap in overridden
	// descriptions/schemas + default injection. MCP/skill/host tools have no row
	// and pass through untouched, so deferred/unlock wiring stays consistent.
	if ToolResolve != nil {
		out = ToolResolve(ctx, agentKey, out)
	}
	out, def.FindingGuidance = findingWorkflowTools(agentKey, out)
	for i, t := range out {
		out[i] = guardPanic(t)
	}
	return out, def, cleanup
}

// guardPanic turns a panicking tool handler into an ordinary tool error. The
// harness runs each tool on its own goroutine, so a panic inside a handler can't
// be recovered by the caller that started the run — it takes the whole process
// down, and on restart the agent replays the same call and crashes again. Applied
// last, so it covers every tool the agent can reach: domain, SDK, MCP and skill.
func guardPanic(t actool.CoreTool) actool.CoreTool { return &guardedTool{CoreTool: t} }

type guardedTool struct{ actool.CoreTool }

// 한국어: 개별 handler의 panic만 회복하여 모델이 읽을 수 있는 도구 오류로 돌린다.
// 한국어: 원래 도구의 권한·이름·스키마 메서드는 내장 CoreTool로 계속 위임된다.
func (g *guardedTool) Call(ctx context.Context, in json.RawMessage, tc *actool.ToolContext) (res actool.Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[tools] %s panic: %v\n%s", g.Name(), r, debug.Stack())
			res, err = actool.Errorf(fmt.Sprintf("工具 %s 内部错误：%v（本次调用已失败，可换个参数或改用别的工具）", g.Name(), r)), nil
		}
	}()
	return g.CoreTool.Call(ctx, in, tc)
}
