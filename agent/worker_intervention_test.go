package agent

// 한국어 파일 해설: agent/worker_intervention_test.go
// Worker 슬롯 대신 exploration/intent ID로 transcript 키가 안정되게 생성되는지 검사한다.
// 인간 개입 request ID 마커는 정상 user 메시지에서만 인식하여 다른 역할의 텍스트를 중복 요청으로 오인하지 않는다.
// 테스트 이름은 아래의 각 Test 함수가 확인하는 구체적 경계를 나타낸다.
// 한국어 문서화 변경은 기존 테스트의 입력·예상값·실행 조건을 수정하지 않는다.

import (
	"testing"

	"github.com/Autumn-27/norma/llm"
)

func TestWorkerSessionIDIsStablePerIntent(t *testing.T) {
	if got := WorkerSessionID(12, 34); got != "exp12-worker-i34" {
		t.Fatalf("session id = %q, want exp12-worker-i34", got)
	}
}

func TestWorkerChatMarkerOnlyMatchesItsNormalUserTurn(t *testing.T) {
	requestID := "worker-message-123"
	messages := []llm.Message{
		{Role: llm.RoleAssistant, Content: []llm.ContentBlock{llm.TextBlock(workerChatMarker(requestID))}},
		llm.UserText(workerChatMarker("worker-message-other") + "\nother"),
		llm.UserText(workerChatMarker(requestID) + "\nnew intent"),
	}
	if !hasWorkerChatMessage(messages, requestID) {
		t.Fatal("expected the matching user turn to be detected")
	}
	if hasWorkerChatMessage(messages, "worker-message-missing") {
		t.Fatal("unrelated request id matched a Worker user turn")
	}
}
