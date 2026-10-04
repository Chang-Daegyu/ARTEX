// [한국어 길잡이] 재사용 가능한 작업 생성 템플릿
// 작업 이름·설명·목표·분류·작업별 승인 규칙 묶음을 저장하고 관리한다.
// PATCH에서 포인터와 원시 JSON 키 집합을 함께 사용해 생략한 필드와 명시적으로 비운 필드를 구분한다.
// 본문 크기·문자 길이·규칙 검증 후 DB에 위임한다. 템플릿 저장 자체가 작업을 실행하는 것은 아니다.
// 전체 모듈 설명과 파일별 읽기 순서: docs/ko/modules/server.md
package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/Autumn-27/artex/db"
)

const maxTaskTemplateRequestBytes = 512 << 10

// taskTemplateRequest uses pointers so PATCH can distinguish omitted fields from
// explicit empty values. Empty values are still rejected by the DB validator.
// category_id / intercept_rules presence is detected via the raw key set (see
// decodeTaskTemplateRequest) so a PATCH can clear category to null.
type taskTemplateRequest struct {
	Name           *string                `json:"name"`
	Description    *string                `json:"description"`
	Goal           *string                `json:"goal"`
	CategoryID     *int64                 `json:"category_id"`
	InterceptRules []taskInterceptRuleReq `json:"intercept_rules"`
}

// decodeTaskTemplateRequest decodes the body into req and returns the set of
// top-level keys present in the JSON (for PATCH presence detection).
// [한국어 함수 설명] 요청 크기를 제한하고 typed 포인터 필드와 원시 키 집합을 함께 읽는다. 생략과 null/빈 값의 차이를 유지하는 PATCH 처리다.
func decodeTaskTemplateRequest(w http.ResponseWriter, r *http.Request, req *taskTemplateRequest) (map[string]struct{}, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxTaskTemplateRequestBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeErr(w, http.StatusRequestEntityTooLarge, "请求正文过大")
		} else {
			writeErr(w, http.StatusBadRequest, err.Error())
		}
		return nil, false
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return nil, false
	}
	if err := json.Unmarshal(body, req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return nil, false
	}
	present := make(map[string]struct{}, len(raw))
	for k := range raw {
		present[k] = struct{}{}
	}
	return present, true
}

func validateTaskTemplateRequest(req taskTemplateRequest) error {
	checks := []struct {
		name  string
		value *string
		limit int
	}{
		{name: "name", value: req.Name, limit: db.MaxTaskTemplateNameRunes},
		{name: "description", value: req.Description, limit: db.MaxTaskTemplateTextRunes},
		{name: "goal", value: req.Goal, limit: db.MaxTaskTemplateTextRunes},
	}
	for _, check := range checks {
		if check.value == nil {
			continue
		}
		value := strings.TrimSpace(*check.value)
		if check.name == "name" {
			value = strings.Join(strings.Fields(value), " ")
		}
		if utf8.RuneCountInString(value) > check.limit {
			return fmt.Errorf("%s 最多 %d 个字符", check.name, check.limit)
		}
	}
	return nil
}

func writeTaskTemplateErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, db.ErrTaskTemplateInvalid):
		writeErr(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, db.ErrTaskTemplateNameConflict):
		writeErr(w, http.StatusConflict, "模板名称已存在")
	case errors.Is(err, db.ErrTaskTemplateNotFound):
		writeErr(w, http.StatusNotFound, "task template not found")
	default:
		writeErr(w, http.StatusInternalServerError, err.Error())
	}
}

func (s *Server) pgListTaskTemplates(w http.ResponseWriter, _ *http.Request) {
	pg := s.pg(w)
	if pg == nil {
		return
	}
	templates, err := pg.ListTaskTemplates()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"templates": templates})
}

func (s *Server) pgCreateTaskTemplate(w http.ResponseWriter, r *http.Request) {
	pg := s.pg(w)
	if pg == nil {
		return
	}
	var req taskTemplateRequest
	if _, ok := decodeTaskTemplateRequest(w, r, &req); !ok {
		return
	}
	if err := validateTaskTemplateRequest(req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	rules, err := buildTaskInterceptRules(req.InterceptRules)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "拦截/允许规则无效："+err.Error())
		return
	}
	template, err := pg.CreateTaskTemplate(db.TaskTemplateInput{
		Name:           stringValue(req.Name),
		Description:    stringValue(req.Description),
		Goal:           stringValue(req.Goal),
		CategoryID:     req.CategoryID,
		InterceptRules: rules,
	})
	if err != nil {
		writeTaskTemplateErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, template)
}

func (s *Server) pgUpdateTaskTemplate(w http.ResponseWriter, r *http.Request) {
	pg := s.pg(w)
	if pg == nil {
		return
	}
	id, ok := pathInt(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad task template id")
		return
	}
	var req taskTemplateRequest
	present, ok := decodeTaskTemplateRequest(w, r, &req)
	if !ok {
		return
	}
	if err := validateTaskTemplateRequest(req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	_, catPresent := present["category_id"]
	_, rulesPresent := present["intercept_rules"]
	if req.Name == nil && req.Description == nil && req.Goal == nil && !catPresent && !rulesPresent {
		writeErr(w, http.StatusBadRequest, "至少需要提供 name、description、goal、category_id 或 intercept_rules")
		return
	}
	patch := db.TaskTemplatePatch{Name: req.Name, Description: req.Description, Goal: req.Goal}
	if catPresent {
		patch.SetCategoryID = true
		patch.CategoryID = req.CategoryID
	}
	if rulesPresent {
		rules, err := buildTaskInterceptRules(req.InterceptRules)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "拦截/允许规则无效："+err.Error())
			return
		}
		patch.SetInterceptRules = true
		patch.InterceptRules = rules
	}
	template, err := pg.PatchTaskTemplate(id, patch)
	if err != nil {
		writeTaskTemplateErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, template)
}

func (s *Server) pgDeleteTaskTemplate(w http.ResponseWriter, r *http.Request) {
	pg := s.pg(w)
	if pg == nil {
		return
	}
	id, ok := pathInt(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad task template id")
		return
	}
	deleted, err := pg.DeleteTaskTemplate(id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !deleted {
		writeErr(w, http.StatusNotFound, "task template not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": id})
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
