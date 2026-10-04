// [한국어 길잡이] chat_llm_resolve_test.go의 테스트 읽기
// 모델 미설정과 미활성 상태의 설명을 구분하고, 대화에 고정된 프로필이 실제 채팅 Agent 선택에 반영되는지 확인한다.
// 테스트를 읽을 때는 임시 데이터/가짜 provider 준비 → 핸들러 또는 함수 호출 → 응답·DB·컨텍스트 검증 순서로 따라간다.
// 주요 진입점: TestChatUnavailableReasonDistinguishesStates, TestResolveChatAgentHonoursConversationProfile.
// DB를 쓰는 사례는 별도의 테스트용 PostgreSQL에서 실행한다. 테스트 파일의 Skip 분기는 환경이 없을 때의 건너뛰기이지 기능 검증 성공을 뜻하지 않는다.
// 전체 모듈 설명과 파일별 읽기 순서: docs/ko/modules/server.md
package server

import (
	"context"
	"strings"
	"testing"

	"github.com/Autumn-27/artex/db"
)

// TestChatUnavailableReasonDistinguishesStates pins the operator-facing message
// to the actual configuration state: nothing configured vs. configured-but-not-
// activated must read differently so the user knows what to do.
func TestChatUnavailableReasonDistinguishesStates(t *testing.T) {
	m, err := NewManager(t.TempDir(), "")
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	defer m.Close()
	td := t.TempDir()
	s := New(context.Background(), m, td, td, td)

	// Start from a clean profile table; other tests in the shared DB may have left rows.
	existing, _ := m.pg.ListProfiles()
	for _, p := range existing {
		if p.IsDefault {
			// The active profile cannot be deleted directly; clear the flag first.
			_, _ = m.pg.Exec(`UPDATE llm_profiles SET is_default=false WHERE id=$1`, p.ID)
		}
		_ = m.pg.DeleteProfile(p.ID)
	}

	if reason := s.chatUnavailableReason(); !strings.Contains(reason, "尚未配置") {
		t.Fatalf("no-profile reason=%q, want 尚未配置", reason)
	}

	id, err := m.pg.SaveProfile(&db.LLMProfile{Name: "p1", Format: "anthropic", Model: "claude-x", APIKey: "sk-test"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = m.pg.Exec(`UPDATE llm_profiles SET is_default=false WHERE id=$1`, id)
		_ = m.pg.DeleteProfile(id)
	})

	// Profile exists but is not activated.
	if reason := s.chatUnavailableReason(); !strings.Contains(reason, "没有已激活") {
		t.Fatalf("inactive reason=%q, want 没有已激活", reason)
	}

	// Once activated, the reason no longer claims a missing/inactive config.
	if err := m.pg.SetActiveProfile(id); err != nil {
		t.Fatal(err)
	}
	if reason := s.chatUnavailableReason(); strings.Contains(reason, "尚未配置") || strings.Contains(reason, "没有已激活") {
		t.Fatalf("active reason=%q should not report missing/inactive", reason)
	}
}

// TestResolveChatAgentHonoursConversationProfile is the regression guard for the
// reported bug: a conversation that picked a valid profile must resolve a chat
// agent even when NO global profile is active, so the send precheck stops
// rejecting it with "LLM 未配置".
func TestResolveChatAgentHonoursConversationProfile(t *testing.T) {
	m, err := NewManager(t.TempDir(), "")
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	defer m.Close()
	td := t.TempDir()
	s := New(context.Background(), m, td, td, td)

	// Force the global fallback to nil so a non-nil result can ONLY come from the
	// conversation's own profile — this is exactly the situation the user hit
	// (no usable global config) and isolates the fix from the test host's env.
	s.cfgMu.Lock()
	s.chatAgent = nil
	s.cfgMu.Unlock()

	id, err := m.pg.SaveProfile(&db.LLMProfile{Name: "conv-pick", Format: "anthropic", Model: "claude-x", APIKey: "sk-test"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.pg.DeleteProfile(id) })

	// A conversation that pinned this profile resolves a chat agent despite the
	// nil global fallback; without the fix the send precheck rejected it outright.
	conv := &db.Conversation{AgentKey: "mainagent", LLMProfileID: &id}
	if s.resolveChatAgent(conv) == nil {
		t.Fatalf("resolveChatAgent(conversation with valid profile) = nil; global fallback wrongly gates the pick")
	}

	// With neither a pick nor a global fallback there is genuinely nothing to run.
	if s.resolveChatAgent(&db.Conversation{AgentKey: "mainagent"}) != nil {
		t.Fatalf("resolveChatAgent with neither pick nor global config should be nil")
	}
}
