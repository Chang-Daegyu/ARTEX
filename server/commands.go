// [한국어 길잡이] 도구 이력·LLM 사용량 조회 API
// activity 기반 도구 실행 이력과 LLM 요청 기록·모델별 토큰·사용 통계를 페이지 및 작업 필터로 조회한다.
// commandTaskFilter의 nil과 숫자 포인터는 전체 조회와 특정 작업 조회를 구분하는 계약이다.
// 원시 LLM 기록 상세와 계량 통계는 별도 저장 경로다. 기록 삭제 API를 읽을 때 비용 통계까지 같은 데이터라고 가정하지 않는다.
// 전체 모듈 설명과 파일별 읽기 순서: docs/ko/modules/server.md
package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Autumn-27/artex/db"
)

// pgListCommands returns tool executions (any tool) from the activity table.
func (s *Server) pgListCommands(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page := atoiDefault(q.Get("page"), 0)
	size := atoiDefault(q.Get("size"), 50)
	keyword := q.Get("q")

	records, total, err := s.m.PG().ListCommands(commandTaskFilter(q.Get("task")), keyword, page, size)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"commands": records, "total": total})
}

// pgToolStats returns per-tool call counts for the tool-execution history, under
// the same task/keyword filters the list takes — the summary describes the whole
// filtered set, not the page on screen.
func (s *Server) pgToolStats(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	pg := s.m.PG()
	if pg == nil {
		writeJSON(w, 200, map[string]any{"stats": []db.ToolStat{}})
		return
	}
	stats, err := pg.ToolStats(commandTaskFilter(q.Get("task")), q.Get("q"))
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"stats": stats})
}

// commandTaskFilter parses the ?task= exploration id. Absent or unparsable → nil
// (no filter), matching the list endpoint's long-standing lenient behaviour.
// [한국어 함수 설명] 빈 값은 전체 작업 조회를 뜻하는 nil이고 숫자 값은 특정 작업 필터다. URL에서 전달된 표시값을 DB 질의의 의미로 바꾼다.
func commandTaskFilter(v string) *int64 {
	if v == "" {
		return nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil
	}
	return &n
}

// pgListLLMRecords returns paginated LLM call records.
func (s *Server) pgListLLMRecords(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page := atoiDefault(q.Get("page"), 0)
	size := atoiDefault(q.Get("size"), 50)
	model := q.Get("model")
	session := q.Get("session")
	task := q.Get("task")

	records, total, err := s.m.PG().ListLLMRecords(model, session, task, page, size)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"records": records, "total": total})
}

// pgTokenByModel returns a task's LLM token usage grouped by model, from the
// always-on llm_usage metering ledger. Accurate even with per-agent model bindings,
// pool rotation/failover, and interrupted runs (every call is metered on success or
// error). No recording toggle required.
func (s *Server) pgTokenByModel(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("task"))
	if id == "" {
		writeErr(w, 400, "missing task")
		return
	}
	pg := s.m.PG()
	if pg == nil {
		writeJSON(w, 200, map[string]any{"models": []db.ModelTokenStat{}})
		return
	}
	models, err := pg.TokenByModel(id)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"models": models})
}

// pgUsageStats returns global llm_usage aggregates for the dashboard's "new" token
// view: per-profile totals (all-time) + per-(profile, day) buckets for the daily
// chart. Sourced from the always-on metering ledger, so it is accurate across
// per-agent bindings, pool rotation, and interrupted runs.
func (s *Server) pgUsageStats(w http.ResponseWriter, r *http.Request) {
	pg := s.m.PG()
	if pg == nil {
		writeJSON(w, 200, map[string]any{"by_profile": []db.ProfileUsage{}, "daily": []db.ProfileDayUsage{}})
		return
	}
	days := atoiDefault(r.URL.Query().Get("days"), 365)
	byProfile, err := pg.UsageByProfile()
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	daily, err := pg.UsageDaily(days)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"by_profile": byProfile, "daily": daily})
}

// pgLLMTasks returns distinct recorded tasks with counts, for the page's task
// picker (pick a task → filter the list, then delete its conversations).
func (s *Server) pgLLMTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.m.PG().LLMTasks()
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"tasks": tasks})
}

// pgDeleteLLMRecords removes all LLM records for one exact task_id (the same
// match the task picker uses). Empty task → 400. Returns rows deleted.
func (s *Server) pgDeleteLLMRecords(w http.ResponseWriter, r *http.Request) {
	task := strings.TrimSpace(r.URL.Query().Get("task"))
	if task == "" {
		writeErr(w, 400, "missing task")
		return
	}
	n, err := s.m.PG().DeleteLLMRecords(task)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"deleted": n})
}

// pgGetLLMRecord returns a single LLM record with full request/response bodies.
// [한국어 함수 설명] 목록에서 제외한 큰 원시 요청/응답 내용을 필요한 한 건에 대해서만 읽는다. 이력이 많을 때 목록 API가 모든 원문을 운반하지 않도록 분리한 경로다.
func (s *Server) pgGetLLMRecord(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, 400, "invalid id")
		return
	}
	rec, err := s.m.PG().GetLLMRecord(id)
	if err != nil {
		writeErr(w, 404, "not found")
		return
	}
	writeJSON(w, 200, rec)
}
