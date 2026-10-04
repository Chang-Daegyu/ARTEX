package agent

// 한국어 파일 해설: agent/provider_responses_test.go
// openai-responses UI 설정이 Norma 형식과 /responses를 제거한 API base로 변환되는지 검사한다.
// 기본 모델/형식 식별이 OpenAI Chat Completions 경로와 혼동되지 않는지 확인한다.
// 테스트 이름은 아래의 각 Test 함수가 확인하는 구체적 경계를 나타낸다.
// 한국어 문서화 변경은 기존 테스트의 입력·예상값·실행 조건을 수정하지 않는다.

import (
	"testing"

	"github.com/Autumn-27/norma/llm"
)

// TestConfigFromOpenAIResponses locks the openai-responses format wiring:
// ConfigFrom resolves the Responses format, strips a full /responses endpoint
// back to the API base, and Provider() round-trips the short name. NewProvider
// must build a working provider for it.
func TestConfigFromOpenAIResponses(t *testing.T) {
	c := ConfigFrom("openai-responses", "gpt-5", "https://gw.example/v1/responses", "sk-x", "")
	if c.Format != llm.FormatOpenAIResponses {
		t.Fatalf("format=%v, want FormatOpenAIResponses", c.Format)
	}
	if c.BaseURL != "https://gw.example/v1" {
		t.Fatalf("base_url=%q, want the /responses suffix stripped", c.BaseURL)
	}
	if c.Provider() != "openai-responses" {
		t.Fatalf("Provider()=%q", c.Provider())
	}
	if !c.Stream { // default streaming preserved
		t.Fatal("Stream should default true")
	}
	if _, err := c.NewProvider(); err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
}
