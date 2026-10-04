// [한국어 길잡이] task_metadata_test.go의 테스트 읽기
// 작업 이름·pin PATCH의 응답과 영속 변경, 대화 일괄 삭제의 사라진 ID별 결과를 검증한다.
// 테스트를 읽을 때는 임시 데이터/가짜 provider 준비 → 핸들러 또는 함수 호출 → 응답·DB·컨텍스트 검증 순서로 따라간다.
// 주요 진입점: TestTaskMetadataPatchReturnsRenameAndPin, TestConversationBatchDeleteReportsMissing.
// DB를 쓰는 사례는 별도의 테스트용 PostgreSQL에서 실행한다. 테스트 파일의 Skip 분기는 환경이 없을 때의 건너뛰기이지 기능 검증 성공을 뜻하지 않는다.
// 전체 모듈 설명과 파일별 읽기 순서: docs/ko/modules/server.md
package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestTaskMetadataPatchReturnsRenameAndPin(t *testing.T) {
	m, err := NewManager(t.TempDir(), "")
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	defer m.Close()
	task, err := m.CreateTask("metadata patch", "goal", nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	taskID, _ := strconv.ParseInt(task.ID, 10, 64)
	defer func() { _ = m.pg.DeleteTask(taskID) }()
	s := New(context.Background(), m, t.TempDir(), t.TempDir(), t.TempDir())
	token, err := signJWT(s.jwtKey)
	if err != nil {
		t.Fatal(err)
	}

	body, _ := json.Marshal(map[string]any{"name": "  renamed task  ", "pinned": true})
	req := httptest.NewRequest(http.MethodPatch, "/api/tasks/"+task.ID, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status=%d body=%s", rec.Code, rec.Body.String())
	}
	var updated struct {
		Name     string `json:"name"`
		Pinned   bool   `json:"pinned"`
		PinnedAt string `json:"pinned_at"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Name != "renamed task" || !updated.Pinned || updated.PinnedAt == "" {
		t.Fatalf("unexpected task patch response: %+v", updated)
	}

	body, _ = json.Marshal(map[string]any{"name": strings.Repeat("任", maxTaskNameRunes+1)})
	req = httptest.NewRequest(http.MethodPatch, "/api/tasks/"+task.ID, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("overlong task name status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestConversationBatchDeleteReportsMissing(t *testing.T) {
	m, err := NewManager(t.TempDir(), "")
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	defer m.Close()
	s := New(context.Background(), m, t.TempDir(), t.TempDir(), t.TempDir())
	first, err := m.pg.CreateConversation("mainagent", "batch-http-first", nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := m.pg.CreateConversation("mainagent", "batch-http-second", nil)
	if err != nil {
		_ = m.pg.DeleteConversation(first.ID)
		t.Fatal(err)
	}
	defer func() {
		_ = m.pg.DeleteConversation(first.ID)
		_ = m.pg.DeleteConversation(second.ID)
	}()
	token, err := signJWT(s.jwtKey)
	if err != nil {
		t.Fatal(err)
	}
	missing := int64(1<<62 - 1)
	body, _ := json.Marshal(map[string]any{"ids": []int64{first.ID, missing, second.ID}})
	req := httptest.NewRequest(http.MethodPost, "/api/conversations/delete/batch", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("batch delete status=%d body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		Items []conversationDeleteItem `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Items) != 3 || !response.Items[0].OK || response.Items[1].OK || !response.Items[2].OK {
		t.Fatalf("batch delete response=%+v", response.Items)
	}
	for _, id := range []int64{first.ID, second.ID} {
		if conversation, err := m.pg.GetConversation(id); err != nil || conversation != nil {
			t.Fatalf("conversation %d still exists: %+v err=%v", id, conversation, err)
		}
	}
	if response.Items[1].Error == "" {
		t.Fatalf("missing conversation %s must include an error", fmt.Sprint(missing))
	}
}
