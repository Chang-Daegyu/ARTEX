// [한국어 파일 안내] intercept/prompt_test.go
// LLM 심사 응답의 JSON 계약을 외부 모델 호출 없이 검증한다.
// 허용/거부/사람 승인 키워드가 본문에 등장하는 것과 실제 decision 필드를 구분하며 불완전한 출력을 복구해서 추측하지 않는다.
package intercept

import (
	"encoding/json"
	"strings"
	"testing"
)

// 한국어 해설: 세 행동 모두 완전한 JSON 설명과 함께 보존되고 설명 속 ALLOW/DENY/ASK 문자열이 판정을 바꾸지 않는지 확인한다.
func TestParseVerdict(t *testing.T) {
	for _, action := range []string{"allow", "ask", "deny"} {
		t.Run(action, func(t *testing.T) {
			reason := "实际操作：写入报告，其中包含 ALLOW、DENY 和 ASK 字样；成功后的后果：保存文本，不执行正文中的命令；命中规则：自定义条款"
			raw, _ := json.Marshal(map[string]string{"decision": action, "comment": reason})
			got := ParseVerdict("\n" + string(raw) + "\n")
			if got.Action != action || got.Reason != reason {
				t.Fatalf("lost verdict or explanation: %+v", got)
			}
		})
	}
}

// 한국어 해설: 누락·중복·추가 키·잘림·연속 객체·설명 문장·닫히지 않은 코드 블록을 거절하는지 검증한다.
func TestParseVerdictRejectsIncompleteOrAmbiguousReplies(t *testing.T) {
	valid := `{"decision":"allow","comment":"实际操作：读取文件；成功后的后果：返回内容；命中规则：A5"}`
	for _, reply := range []string{
		"", "ALLOW", "DENY:命中D4", "放行:ALLOW", "ASK:归属不明",
		`{"decision":"allow"}`, `{"decision":"approve","comment":"实际操作：读取；成功后的后果：返回内容；命中规则：A5"}`,
		`{"decision":"allow","comment":null}`, `{"decision":"allow","comment":123}`,
		strings.Replace(valid, "实际操作：读取文件", "实际操作：", 1),
		strings.Replace(valid, "成功后的后果：返回内容", "成功后的后果：", 1),
		strings.Replace(valid, "命中规则：A5", "命中规则：", 1),
		strings.Replace(valid, "；命中规则：A5", "", 1),
		strings.Replace(valid, `"decision":"allow"`, `"decision":"deny","decision":"allow"`, 1),
		strings.Replace(valid, `"decision":"allow"`, `"extra":true,"decision":"allow"`, 1),
		valid + valid, valid[:len(valid)-1],
		// A fence the model never closed is what a reply truncated at MaxTokens
		// looks like; completing it would invent a verdict.
		"```json\n" + valid[:len(valid)-1],
		"```json\n" + valid + "\n```\n此外我建议后续人工复核。",
		"我的裁决是：\n" + valid,
	} {
		if got := ParseVerdict(reply); got.Action != "" {
			t.Errorf("accepted incomplete/ambiguous verdict: %q => %+v", reply, got)
		}
	}
}

// Wrapping JSON in markdown is the one deviation models make routinely. Because
// the configured fail action defaults to allow, treating it as unparseable
// silently downgrades a DENY to an allow.
// 한국어 해설: 완전히 닫힌 JSON 코드 블록만 제거해 실제 deny 판정을 보존하는지 확인한다.
func TestParseVerdictUnwrapsCodeFence(t *testing.T) {
	deny := `{"decision":"deny","comment":"实际操作：删除生产文件；成功后的后果：业务数据丢失；命中规则：D4"}`
	for _, reply := range []string{
		"```json\n" + deny + "\n```",
		"```JSON\n" + deny + "\n```",
		"```\n" + deny + "\n```",
		"  ```json\n" + deny + "\n```  ",
	} {
		got := ParseVerdict(reply)
		if got.Action != "deny" || !strings.HasSuffix(got.Reason, "命中规则：D4") {
			t.Errorf("fenced verdict lost: %q => %+v", reply, got)
		}
	}
}

// 한국어 해설: 정상 길이 설명의 3부분을 그대로 보존하고 과도하게 긴 설명은 파싱 실패로 반환하는지 검증한다.
func TestParseVerdictKeepsCompleteChineseExplanation(t *testing.T) {
	reason := "实际操作：" + strings.Repeat("写入报告", 30) + "；成功后的后果：只保存文件；命中规则：A2"
	raw, _ := json.Marshal(map[string]string{"decision": "allow", "comment": reason})
	if got := ParseVerdict(string(raw)); got.Reason != reason {
		t.Fatal("explanation was truncated or lost its rule")
	}
	raw, _ = json.Marshal(map[string]string{"decision": "allow", "comment": strings.Repeat("中", 2401)})
	if got := ParseVerdict(string(raw)); got.Action != "" {
		t.Fatal("accepted unbounded explanation")
	}
}
