package agent

// 한국어 파일 해설: agent/deferred.go
// MCP 도구의 스키마를 처음부터 모두 보내지 않기 위한 프롬프트·해제 보조 코드이다.
// 전역 공개 도구 이름은 마지막 system 구간에 넣고 Skill 전용 이름은 그 구간에서 제외한다.
// DynamicBoundary는 세션 동안 고정된 system 구간 전체를 캐시할 수 있도록 지정한다.
// 새 Session이 이전 transcript를 Resume하면 과거 Skill 호출을 읽어 UnlockSet을 복원한다.
// 지연 대상은 도구 설명·스키마 노출이며 MCP 서버의 연결 자체가 지연된다는 뜻은 아니다.
// 도구 호출 허용 범위와 실제 MCP 연결 수명은 server/assembly.go 및 Norma 구현과 함께 확인한다.

import (
	"encoding/json"

	"github.com/Autumn-27/norma/llm"
	actool "github.com/Autumn-27/norma/tool"
)

// deferredSystem builds an agent's system-prompt segments and cache boundary from
// its DeferredInfo. When globally-available MCP tools are present, their names + a
// "prefer core tools" instruction render into a <available-deferred-tools> block
// placed as the LAST system-prompt segment, with DynamicBoundary set so the whole
// (session-fixed) system prompt — including the block — is cached (design doc
// §2.1 / C1). Skill-gated MCP names are NOT in this block; they surface when their
// skill loads. Returns a single plain segment + boundary 0 when there is no global
// block to add.
func deferredSystem(sysText string, def DeferredInfo) (system []string, boundary int) {
	sysText += def.FindingGuidance
	block := actool.RenderDeferredToolsBlock(def.GlobalNames)
	if block == "" {
		return []string{sysText}, 0
	}
	system = []string{sysText, block}
	boundary = len(system) // b >= len → whole system prompt cached (SDK guard)
	return system, boundary
}

// seedUnlockFromHistory replays prior Skill() invocations in the conversation so
// their skill-gated MCPs are re-unlocked on a resumed session (design doc C2). The
// main agent builds a fresh session each turn; its in-memory unlock set would
// otherwise reset, leaving the model able to see a skill-revealed tool name yet
// unable to call it. No-op when unlockSkill is nil (no deferred tools).
// 한국어: 과거의 Skill 호출 이름만 재생하여 메모리의 해제 집합을 복원한다.
// 한국어: Skill 자체나 과거 MCP 작업을 다시 실행하는 과정은 아니다.
func seedUnlockFromHistory(msgs []llm.Message, unlockSkill func(string)) {
	if unlockSkill == nil {
		return
	}
	for _, m := range msgs {
		for _, b := range m.ToolUses() {
			if b.Name != "Skill" {
				continue
			}
			var in struct {
				Name string `json:"name"`
			}
			if json.Unmarshal(b.Input, &in) == nil && in.Name != "" {
				unlockSkill(in.Name)
			}
		}
	}
}
