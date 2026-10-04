// [한국어 길잡이] 도구 호출 직전 사용량 기록
// meteredTool은 CoreTool을 포함해 원래 스키마와 권한 정보를 유지하며 Call만 감싼다.
// 실행 직전에 도구 키·Agent·작업·세션을 사용량 테이블에 기록한 뒤 원 도구를 호출한다.
// 계량 저장 실패는 경고 로그로 남기고 도구 호출은 계속한다. 이 카운터는 성공한 결과만의 개수가 아니다.
// 전체 모듈 설명과 파일별 읽기 순서: docs/ko/modules/server.md
package server

import (
	"context"
	"encoding/json"
	"log"

	actool "github.com/Autumn-27/norma/tool"

	"github.com/Autumn-27/artex/agent"
	"github.com/Autumn-27/artex/db"
)

type toolUsageRecorder interface {
	InsertToolUsage(*db.ToolUsage) error
}

// meteredTool records one row immediately before a catalog tool is invoked. It
// embeds the resolved tool so schema overrides, defaults and permission behavior
// remain unchanged.
type meteredTool struct {
	actool.CoreTool
	recorder toolUsageRecorder
	toolKey  string
	agentKey string
	ri       agent.RunInfo
}

func meterTool(t actool.CoreTool, recorder toolUsageRecorder, toolKey, agentKey string, ri agent.RunInfo) actool.CoreTool {
	if recorder == nil {
		return t
	}
	return &meteredTool{CoreTool: t, recorder: recorder, toolKey: toolKey, agentKey: agentKey, ri: ri}
}

// [한국어 함수 설명] 기록 시도 후 원래 CoreTool.Call에 같은 ctx/input/ToolContext를 전달한다. 계량 실패는 로그로 알리며 실제 도구의 오류/결과는 변형하지 않는다.
func (m *meteredTool) Call(ctx context.Context, input json.RawMessage, tc *actool.ToolContext) (actool.Result, error) {
	if err := m.recorder.InsertToolUsage(&db.ToolUsage{
		ToolKey:       m.toolKey,
		AgentKey:      m.agentKey,
		TaskID:        m.ri.TaskID,
		ExplorationID: m.ri.ExplorationID,
		IntentID:      m.ri.IntentID,
		SessionID:     m.ri.SessionID,
	}); err != nil {
		log.Printf("[toolusage] insert %s: %v", m.toolKey, err)
	}
	return m.CoreTool.Call(ctx, input, tc)
}
