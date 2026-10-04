// [한국어 길잡이] 작업과 공유 자산의 연결 관리
// 공유 자산을 특정 작업에 붙이거나 떼고 의도가 참조하는 자산을 조회한다.
// 연결을 없애는 동작과 전역 자산 자체를 삭제하는 동작은 구분된다. 전자는 작업 문맥의 참조 관계를 바꾼다.
// 잘못된 자산·작업 입력은 저장소 오류 종류에 맞춰 HTTP 응답으로 변환하며 삭제 장벽을 사용한다.
// 전체 모듈 설명과 파일별 읽기 순서: docs/ko/modules/server.md
package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Autumn-27/artex/db"
)

const maxTaskAssetRequestBytes = 512 << 10

func writeTaskAssetError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, db.ErrTaskAssetInvalid):
		writeErr(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, db.ErrTaskAssetTaskNotFound), errors.Is(err, db.ErrTaskAssetAssetNotFound):
		writeErr(w, http.StatusNotFound, err.Error())
	default:
		writeErr(w, http.StatusInternalServerError, err.Error())
	}
}

// [한국어 함수 설명] 기존 공유 자산 ID를 작업 문맥에 연결한다. 동일 자산을 새로 복제하는 과정이 아니므로 전역 정체성과 작업별 출처를 함께 보존한다.
func (s *Server) attachTaskAssets(w http.ResponseWriter, r *http.Request) {
	task, ok := s.m.Task(r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "task not found")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxTaskAssetRequestBytes)
	var request struct {
		AssetIDs      []int64            `json:"asset_ids"`
		SourceSummary string             `json:"source_summary"`
		Scope         companyScopeInputs `json:"scope"`
	}
	if err := decode(r, &request); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeErr(w, http.StatusRequestEntityTooLarge, "请求正文过大")
		} else {
			writeErr(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	request.SourceSummary = strings.TrimSpace(request.SourceSummary)
	taskID, _ := parseTaskID(task.ID)
	if request.Scope != nil && len(request.AssetIDs) > 0 {
		writeErr(w, http.StatusBadRequest, "scope 与 asset_ids 不能同时提交")
		return
	}
	if request.Scope != nil {
		mutation, err := s.m.Assets().RegisterTaskAssetScopes(taskID, request.Scope)
		if err != nil {
			writeTaskAssetError(w, err)
			return
		}
		task.Notify()
		writeJSON(w, http.StatusOK, mutation)
		return
	}
	mutation, err := s.m.Assets().AttachAssetsToTask(taskID, request.AssetIDs, request.SourceSummary)
	if err != nil {
		writeTaskAssetError(w, err)
		return
	}
	task.Notify()
	writeJSON(w, http.StatusOK, mutation)
}

// [한국어 함수 설명] 작업에서의 연결만 제거한다. 공유 자산 자체를 지우는 API와 구분해 다른 작업의 참조를 유지한다.
func (s *Server) detachTaskAsset(w http.ResponseWriter, r *http.Request) {
	task, ok := s.m.Task(r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "task not found")
		return
	}
	assetID, ok := pathInt(r, "assetID")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad asset id")
		return
	}
	taskID, _ := parseTaskID(task.ID)
	detached, err := s.m.Assets().DetachAssetFromTask(taskID, assetID)
	if err != nil {
		writeTaskAssetError(w, err)
		return
	}
	if !detached {
		writeErr(w, http.StatusNotFound, "asset is not associated with this task")
		return
	}
	task.Notify()
	writeJSON(w, http.StatusOK, map[string]any{"detached": assetID})
}

func (s *Server) taskIntentAssets(w http.ResponseWriter, r *http.Request) {
	task, ok := s.m.Task(r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "task not found")
		return
	}
	taskID, _ := parseTaskID(task.ID)
	assets, err := s.m.Assets().IntentAssets(taskID)
	if err != nil {
		writeTaskAssetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"assets": assets})
}
