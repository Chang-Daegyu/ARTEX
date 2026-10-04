// [한국어 길잡이] conversation_status_test.go의 테스트 읽기
// 대화 목록의 running 표시가 서버의 실제 busy 상태와 일치하고 특정 Agent 종류에만 고정되지 않는지 확인한다.
// 테스트를 읽을 때는 임시 데이터/가짜 provider 준비 → 핸들러 또는 함수 호출 → 응답·DB·컨텍스트 검증 순서로 따라간다.
// 주요 진입점: TestConversationListRunningState.
// DB를 쓰는 사례는 별도의 테스트용 PostgreSQL에서 실행한다. 테스트 파일의 Skip 분기는 환경이 없을 때의 건너뛰기이지 기능 검증 성공을 뜻하지 않는다.
// 전체 모듈 설명과 파일별 읽기 순서: docs/ko/modules/server.md
package server

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestConversationListRunningState(t *testing.T) {
	s, _ := newRetestServer(t)
	var ids []int64
	for _, key := range []string{"auto", "reporter", "retester"} {
		c, err := s.m.pg.CreateConversation(key, "runtime status test", nil)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, c.ID)
	}
	t.Cleanup(func() {
		s.chatMu.Lock()
		clear(s.chatBusy)
		s.chatMu.Unlock()
		for _, id := range ids {
			_, _ = s.m.pg.Exec(`DELETE FROM conversations WHERE id=$1`, id)
		}
	})
	check := func(want map[int64]bool) {
		t.Helper()
		w := retestRequest(s.pgListConversations, http.MethodGet, 0, "")
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body)
		}
		var body struct {
			Conversations []conversationListItem `json:"conversations"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		found := 0
		for _, item := range body.Conversations {
			if expected, ok := want[item.ID]; ok {
				found++
				if item.Running != expected || item.Title != "runtime status test" {
					t.Fatalf("conversation %d: running=%v want=%v title=%q", item.ID, item.Running, expected, item.Title)
				}
			}
		}
		if found != len(want) {
			t.Fatalf("found %d of %d conversations", found, len(want))
		}
	}
	check(map[int64]bool{ids[0]: false, ids[1]: false, ids[2]: false})
	s.chatMu.Lock()
	s.chatBusy[s.convBusyKey(ids[1])] = true
	s.chatBusy[s.convBusyKey(ids[2])] = true
	s.chatBusy["unrelated-task"] = true
	s.chatMu.Unlock()
	check(map[int64]bool{ids[0]: false, ids[1]: true, ids[2]: true})
	s.chatMu.Lock()
	clear(s.chatBusy)
	s.chatMu.Unlock()
	check(map[int64]bool{ids[0]: false, ids[1]: false, ids[2]: false})
}
