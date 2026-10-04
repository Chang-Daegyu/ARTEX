package agent

// 한국어 파일 해설: agent/constraints.go
// task_constraints의 allow/deny 문장을 역할 프롬프트에 붙이는 렌더러이다.
// 빈 제약이나 저장소 부재·조회 실패 때는 빈 문자열을 반환한다.
// 허용 문장과 금지 문장을 따로 모아 모델이 각 행동 전에 확인하도록 안내한다.
// Planner와 Worker는 자신의 설정에 따라 이 블록을 system prompt에 추가한다.
// 이 계층은 자연어 지시이며 HTTP 프록시의 강제 범위 검사나 운영체제 권한 경계가 아니다.
// 실제 도구 승인과 자산 차단 여부는 guard/intercept 및 해당 도구의 검증 경로를 따로 읽어야 한다.

import (
	"strings"

	"github.com/Autumn-27/artex/db"
)

// constraintBlock renders this task's operation constraints (task_constraints) as a
// high-priority block appended to the planner/worker system prompt. allow/deny are
// grouped; empty string when there are no constraints (or ts is nil). The framing
// deliberately puts these ABOVE the exploration/expansion heuristics so a declared
// boundary wins the tug-of-war against "chase another entry surface".
// 한국어: 저장된 제약을 분류해 자연어 블록으로 만든다. 별도의 도구 승인 결정을 반환하는 함수는 아니다.
func constraintBlock(ts *db.ExplorationStore) string {
	if ts == nil {
		return ""
	}
	rows, err := ts.ListConstraints()
	if err != nil || len(rows) == 0 {
		return ""
	}
	var allow, deny []string
	for _, c := range rows {
		text := strings.TrimSpace(c.Text)
		if text == "" {
			continue
		}
		if c.Kind == "allow" {
			allow = append(allow, "- "+text)
		} else {
			deny = append(deny, "- "+text)
		}
	}
	if len(allow) == 0 && len(deny) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n【操作约束（最高优先级，凌驾于下方一切探索/拓面启发式；每生成一条意图、每执行一个动作前都必须先自检是否违反，违反即不得进行）】：")
	if len(allow) > 0 {
		b.WriteString("\n允许的操作：\n")
		b.WriteString(strings.Join(allow, "\n"))
	}
	if len(deny) > 0 {
		b.WriteString("\n禁止的操作：\n")
		b.WriteString(strings.Join(deny, "\n"))
	}
	b.WriteString("\n（发现约束之外的新目标/新端口/新主机，不等于获得授权：除非它落在上述允许范围内，否则记为 out-of-scope 事实并跳过，不得为其派生意图或执行动作。）")
	return b.String()
}
