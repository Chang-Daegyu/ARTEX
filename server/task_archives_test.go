// [한국어 길잡이] task_archives_test.go의 테스트 읽기
// 아카이브 요청이 즉시 실행 결과가 아닌 큐 항목으로 생성되고 목록·요청 한도가 API 계약을 지키는지 검사한다.
// 테스트를 읽을 때는 임시 데이터/가짜 provider 준비 → 핸들러 또는 함수 호출 → 응답·DB·컨텍스트 검증 순서로 따라간다.
// 주요 진입점: TestTaskArchiveAPIQueueListAndLimits.
// DB를 쓰는 사례는 별도의 테스트용 PostgreSQL에서 실행한다. 테스트 파일의 Skip 분기는 환경이 없을 때의 건너뛰기이지 기능 검증 성공을 뜻하지 않는다.
// 전체 모듈 설명과 파일별 읽기 순서: docs/ko/modules/server.md
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestTaskArchiveAPIQueueListAndLimits(t *testing.T) {
	m, err := NewManager(t.TempDir(), "")
	if err != nil {
		t.Skipf("postgres unavailable (%v) - skipping", err)
	}
	defer m.Close()
	task, err := m.CreateTask("archive api", "verify 202", nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	taskID, err := strconv.ParseInt(task.ID, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = m.pg.Exec(`DELETE FROM task_archives WHERE task_id=$1`, taskID)
		_ = m.pg.DeleteTask(taskID)
	}()

	// Start with an already-cancelled context so the background archive worker
	// exits before these route-contract assertions enqueue work.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	dataDir := t.TempDir()
	s := New(ctx, m, dataDir, dataDir, dataDir)
	s.archiveWG.Wait()
	token, err := signJWT(s.jwtKey)
	if err != nil {
		t.Fatal(err)
	}
	request := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		s.Handler().ServeHTTP(recorder, req)
		return recorder
	}

	recorder := request(http.MethodPost, "/api/tasks/"+task.ID+"/archive", "{}")
	if recorder.Code != http.StatusConflict {
		t.Fatalf("running archive status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if err := m.pg.SetPaused(taskID, true); err != nil {
		t.Fatal(err)
	}
	recorder = request(http.MethodPost, "/api/tasks/"+task.ID+"/archive", "{}")
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("archive queue status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var queued struct {
		ID    int64  `json:"id"`
		State string `json:"state"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &queued); err != nil {
		t.Fatal(err)
	}
	if queued.ID <= 0 || queued.State != "archive_queued" {
		t.Fatalf("unexpected queue response: %+v", queued)
	}

	recorder = request(http.MethodGet, "/api/task-archives?page=1&size=10&q="+task.ID, "")
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), fmt.Sprintf(`"task_id":%d`, taskID)) {
		t.Fatalf("archive list status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	ids := make([]string, 101)
	for index := range ids {
		ids[index] = strconv.Itoa(index + 1)
	}
	body, _ := json.Marshal(map[string]any{"task_ids": ids})
	recorder = request(http.MethodPost, "/api/tasks/archive/batch", string(body))
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "100") {
		t.Fatalf("oversized batch status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
