// [한국어 길잡이] 발견에 묶인 HTTP 증거의 조회·편집
// 원시 트래픽 ID와 영속 증거 스냅샷을 구분하고 evidence.Store를 통해 바인딩·메모·순서·본문 조회를 처리한다.
// 작업 상속 문맥에서는 읽기 가능 출처와 쓰기 권한을 따로 확인한다. 원시 트래픽을 나중에 삭제해도 독립 복사한 증거를 조회할 수 있는 구조다.
// 본문 미리 보기는 범위·UTF-8 경계·바이너리 여부를 처리한다. 요약 API와 원문 다운로드 API의 책임이 다르다.
// 전체 모듈 설명과 파일별 읽기 순서: docs/ko/modules/server.md
package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"unicode/utf8"

	"github.com/Autumn-27/artex/agent"
	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/evidence"
	actool "github.com/Autumn-27/norma/tool"
)

func (s *Server) evidenceStore() *evidence.Store {
	return evidence.New(s.m.pg, s.m.traffic, filepath.Join(s.m.dir, "evidence"))
}

// Add only new optional properties; preserve edited descriptions, existing
// properties, agent bindings and disabled flags. The one-time flag also keeps
// subsequent user unbinding of the evidence reader intact.
func (s *Server) seedFindingTrafficTools() {
	const flag = "finding_traffic_tools_v1"
	if v, _, _ := s.m.pg.GetSetting(flag); v == "true" {
		return
	}
	for _, key := range []string{"report_finding", "update_finding_report"} {
		var schema any
		if key == "update_finding_report" {
			schema = s.toolUpdateFindingReport().InputSchema()
		} else {
			for _, seed := range agent.BuiltinToolSeeds() {
				if seed.Key == key {
					schema = seed.Schema
					break
				}
			}
		}
		raw, err := json.Marshal(schema)
		if err != nil {
			log.Printf("[evidence] tool schema: %v", err)
			return
		}
		var obj map[string]json.RawMessage
		if err = json.Unmarshal(raw, &obj); err != nil {
			return
		}
		// BuiltinToolSeeds stores Schema as RawMessage; both forms marshal as JSON.
		var properties map[string]json.RawMessage
		if err = json.Unmarshal(obj["properties"], &properties); err != nil {
			return
		}
		name := "traffic_refs"
		if key == "update_finding_report" {
			name = "evidence_version"
		}
		if len(properties[name]) == 0 {
			return
		}
		_, err = s.m.pg.Exec(`UPDATE tools SET schema=jsonb_set(schema,ARRAY['properties',$2::text],$3::jsonb,true),updated_at=now()
WHERE key=$1 AND system AND NOT(COALESCE(schema->'properties','{}'::jsonb) ? $2)`, key, name, string(properties[name]))
		if err != nil {
			log.Printf("[evidence] upgrade tool %s: %v", key, err)
			return
		}
	}
	if reporter, _ := s.m.pg.GetAgentByKey("reporter"); reporter != nil {
		if err := s.m.pg.AddAgentToToolBinding("reporter", []string{"get_finding_traffic"}); err != nil {
			return
		}
	}
	_ = s.m.pg.SetSetting(flag, "true")
}

func (s *Server) registerFindingTraffic(mux *http.ServeMux) {
	base := "/api/exploration/findings/{id}/traffic"
	mux.HandleFunc("GET "+base, s.getFindingTraffic)
	mux.HandleFunc("POST "+base, s.bindFindingTraffic)
	mux.HandleFunc("PATCH "+base+"/{binding_id}", s.editFindingTraffic)
	mux.HandleFunc("DELETE "+base+"/{binding_id}", s.editFindingTraffic)
	mux.HandleFunc("PUT "+base+"/order", s.editFindingTraffic)
	mux.HandleFunc("GET "+base+"/{binding_id}", s.getFindingTrafficDetail)
	mux.HandleFunc("GET "+base+"/{binding_id}/body", s.getFindingTrafficBody)
}

func evidenceError(w http.ResponseWriter, err error) {
	status := http.StatusUnprocessableEntity
	if errors.Is(err, db.ErrEvidenceConflict) || errors.Is(err, db.ErrTaskArchiveState) {
		status = http.StatusConflict
	}
	if errors.Is(err, db.ErrFindingNotFound) || errors.Is(err, db.ErrEvidenceNotFound) {
		status = http.StatusNotFound
	}
	writeErr(w, status, err.Error())
}

// [한국어 함수 설명] 발견 ID와 작업 문맥을 검사하고 읽기 요청인지 쓰기 요청인지에 따라 상속 자료 접근을 달리 처리한다.
func (s *Server) findingTrafficAccess(w http.ResponseWriter, r *http.Request, write bool) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, 400, "invalid finding id")
		return 0, false
	}
	f, err := s.m.pg.GetFinding(id)
	if err != nil {
		writeErr(w, 500, err.Error())
		return 0, false
	}
	if f == nil {
		writeErr(w, 404, "finding not found")
		return 0, false
	}
	if taskID := r.URL.Query().Get("context_task"); taskID != "" {
		task := s.m.ResolveTask(taskID)
		if task == nil {
			writeErr(w, 404, "context task not found")
			return 0, false
		}
		_, inherited, allowed := findingProvenanceInTask(task, f.TaskID)
		if !allowed {
			writeErr(w, 404, "finding not available in task context")
			return 0, false
		}
		if write && inherited {
			writeErr(w, 403, "继承漏洞只读，请在来源任务中修改")
			return 0, false
		}
	}
	return id, true
}

func trafficSummary(in *db.FindingTraffic) *db.FindingTraffic {
	out := *in
	out.Bindings = append([]db.FindingTrafficBinding{}, in.Bindings...)
	for i := range out.Bindings {
		out.Bindings[i].Snapshot.ReqHead = ""
		out.Bindings[i].Snapshot.RespHead = ""
	}
	return &out
}

func (s *Server) getFindingTraffic(w http.ResponseWriter, r *http.Request) {
	id, ok := s.findingTrafficAccess(w, r, false)
	if !ok {
		return
	}
	out, err := s.m.pg.GetFindingTraffic(r.Context(), id)
	if err != nil {
		evidenceError(w, err)
		return
	}
	writeJSON(w, 200, trafficSummary(out))
}

// [한국어 함수 설명] 기존 원시 트래픽 참조를 발견의 영속 증거로 바인딩한다. 파일 복사·해시 검증·DB 쓰기의 세부 절차는 evidence.Store가 담당한다.
func (s *Server) bindFindingTraffic(w http.ResponseWriter, r *http.Request) {
	id, ok := s.findingTrafficAccess(w, r, true)
	if !ok {
		return
	}
	var body struct {
		Refs []db.TrafficRef `json:"traffic_refs"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		writeErr(w, 400, "invalid body")
		return
	}
	if len(body.Refs) == 0 {
		writeErr(w, 400, "请选择流量")
		return
	}
	out, err := s.evidenceStore().Bind(r.Context(), id, body.Refs)
	if err != nil {
		evidenceError(w, err)
		return
	}
	writeJSON(w, 200, trafficSummary(out))
}

func (s *Server) editFindingTraffic(w http.ResponseWriter, r *http.Request) {
	id, ok := s.findingTrafficAccess(w, r, true)
	if !ok {
		return
	}
	var body struct {
		Version *int64   `json:"version"`
		Role    *string  `json:"role"`
		Note    *string  `json:"note"`
		Order   []string `json:"binding_ids"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil || body.Version == nil {
		writeErr(w, 400, "version 和有效请求体必填")
		return
	}
	var order []int64
	bindingID := int64(0)
	if r.Method == http.MethodPut {
		if body.Order == nil {
			writeErr(w, 400, "binding_ids 必填")
			return
		}
		order = []int64{}
		for _, raw := range body.Order {
			v, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || v <= 0 {
				writeErr(w, 400, "invalid binding id")
				return
			}
			order = append(order, v)
		}
	} else {
		var err error
		bindingID, err = strconv.ParseInt(r.PathValue("binding_id"), 10, 64)
		if err != nil || bindingID <= 0 {
			writeErr(w, 400, "invalid binding id")
			return
		}
	}
	err := s.m.pg.EditFindingTraffic(r.Context(), id, bindingID, *body.Version, body.Role, body.Note, r.Method == http.MethodDelete, order)
	if err != nil {
		evidenceError(w, err)
		return
	}
	s.getFindingTraffic(w, r)
}

type evidencePreview struct {
	Content    string `json:"content"`
	Offset     int64  `json:"offset"`
	Total      int64  `json:"total"`
	NextOffset int64  `json:"next_offset"`
	Truncated  bool   `json:"truncated"`
	Binary     bool   `json:"binary"`
}

// [한국어 함수 설명] 증거 한쪽의 지정 구간을 읽어 텍스트/바이너리 미리 보기를 구성한다. UTF-8 글자 중간을 자르는 경우를 보정하고 큰 본문을 전부 보내지 않는다.
func readEvidencePreview(store *evidence.Store, snapshot db.TrafficEvidenceSnapshot, side string, offset, length int64) (out evidencePreview, err error) {
	if offset < 0 || length < 0 {
		return out, errors.New("offset / length 不能为负数")
	}
	if length == 0 || length > 8192 {
		length = 8192
	}
	f, total, err := store.OpenBody(snapshot, side)
	if err != nil {
		return out, err
	}
	defer f.Close()
	if offset > total {
		return out, errors.New("offset 超出正文长度")
	}
	if _, err = f.Seek(offset, io.SeekStart); err != nil {
		return out, err
	}
	raw, err := io.ReadAll(io.LimitReader(f, length))
	if err != nil {
		return out, err
	}
	// Leave an incomplete trailing UTF-8 rune for the next page. Explicit byte
	// offsets are still accepted; complete downloads always retain original bytes.
	if offset+int64(len(raw)) < total && len(raw) >= utf8.UTFMax {
		start := len(raw) - 1
		for start > 0 && raw[start]&0xc0 == 0x80 {
			start--
		}
		if !utf8.FullRune(raw[start:]) {
			raw = raw[:start]
		}
	}
	out = evidencePreview{Offset: offset, Total: total, NextOffset: offset + int64(len(raw)), Truncated: offset+int64(len(raw)) < total,
		Binary: bytes.IndexByte(raw, 0) >= 0 || (offset == 0 && !utf8.Valid(raw))}
	if out.Binary {
		out.Content = fmt.Sprintf("[二进制正文，%d 字节；请下载查看]", total)
	} else {
		out.Content = string(bytes.ToValidUTF8(raw, []byte("�")))
	}
	return out, nil
}

func (s *Server) getFindingTrafficDetail(w http.ResponseWriter, r *http.Request) {
	id, ok := s.findingTrafficAccess(w, r, false)
	if !ok {
		return
	}
	bid, err := strconv.ParseInt(r.PathValue("binding_id"), 10, 64)
	if err != nil || bid <= 0 {
		writeErr(w, 400, "invalid binding id")
		return
	}
	store := s.evidenceStore()
	var result any
	err = store.WithBinding(r.Context(), id, bid, func(b db.FindingTrafficBinding) error {
		req, err := readEvidencePreview(store, b.Snapshot, "request", 0, 8192)
		if err != nil {
			return err
		}
		resp, err := readEvidencePreview(store, b.Snapshot, "response", 0, 8192)
		if err != nil {
			return err
		}
		result = map[string]any{"binding": b, "request": req, "response": resp}
		return nil
	})
	if err != nil {
		evidenceError(w, err)
		return
	}
	writeJSON(w, 200, result)
}

// [한국어 함수 설명] 저장된 증거의 요청 또는 응답 원문을 다운로드한다. 목록/미리 보기의 잘린 표현과 별개로 보관된 바이트를 제공하는 경로다.
func (s *Server) getFindingTrafficBody(w http.ResponseWriter, r *http.Request) {
	id, ok := s.findingTrafficAccess(w, r, false)
	if !ok {
		return
	}
	bid, err := strconv.ParseInt(r.PathValue("binding_id"), 10, 64)
	if err != nil || bid <= 0 {
		writeErr(w, 400, "invalid binding id")
		return
	}
	side := r.URL.Query().Get("side")
	store := s.evidenceStore()
	// Resolve the binding under the evidence lock, then read the blob without it:
	// both paths below are O(body size) and would otherwise stall every evidence
	// write for as long as the client takes to receive the data.
	b, err := store.Binding(r.Context(), id, bid)
	if err != nil {
		evidenceError(w, err)
		return
	}
	if r.URL.Query().Get("download") == "1" {
		f, length, err := store.OpenBody(b.Snapshot, side)
		if err != nil {
			evidenceError(w, err)
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"evidence-%d-%s.bin\"", bid, side))
		w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
		_, _ = io.Copy(w, f)
		return
	}
	offset, length := int64(0), int64(8192)
	for name, dst := range map[string]*int64{"offset": &offset, "length": &length} {
		if raw := r.URL.Query().Get(name); raw != "" {
			v, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				evidenceError(w, err)
				return
			}
			*dst = v
		}
	}
	preview, err := readEvidencePreview(store, b.Snapshot, side, offset, length)
	if err != nil {
		evidenceError(w, err)
		return
	}
	writeJSON(w, 200, preview)
}

func (s *Server) toolGetFindingTraffic() actool.CoreTool {
	return roTool("get_finding_traffic", "读取漏洞已绑定的真实流量证据，不依赖捕获开关。finding_id 使用 report_finding JSON 返回的独立漏洞记录 ID（不是第一行的探索节点 ID）。先不传 binding_id 获取清单及 version；空清单是正常情况，TCP 等非 HTTP 漏洞或未采集时仍可依据文字/命令证据编写报告，不强制绑定。有绑定时按 binding_id、side(request/response)、offset 分段读取正文。写报告时将读取的 version 作为 evidence_version 传给 update_finding_report，后者 finding_id 仍使用探索节点 ID。",
		objSchema(map[string]any{"finding_id": strParam("独立漏洞记录 ID"), "binding_id": strParam("清单里的绑定 ID，省略则返回清单"), "side": strParam("request 或 response，默认 response"), "offset": map[string]any{"type": "integer"}, "length": map[string]any{"type": "integer"}}, "finding_id"),
		func(ctx context.Context, in json.RawMessage) (actool.Result, error) {
			var a struct {
				FindingID      json.RawMessage `json:"finding_id"`
				BindingID      json.RawMessage `json:"binding_id"`
				Side           string          `json:"side"`
				Offset, Length int64
			}
			if err := json.Unmarshal(in, &a); err != nil {
				return actool.Errorf(err.Error()), nil
			}
			id, bid := parseProfileID(a.FindingID), parseProfileID(a.BindingID)
			if err := s.agentFindingTrafficAccess(ctx, id, false); err != nil {
				return actool.Errorf(err.Error()), nil
			}
			if len(a.BindingID) == 0 {
				list, err := s.m.pg.GetFindingTraffic(ctx, id)
				if err != nil {
					return actool.Errorf(err.Error()), nil
				}
				return jsonResult(trafficSummary(list))
			}
			if a.Side == "" {
				a.Side = "response"
			}
			store := s.evidenceStore()
			var result any
			err := store.WithBinding(ctx, id, bid, func(b db.FindingTrafficBinding) error {
				preview, err := readEvidencePreview(store, b.Snapshot, a.Side, a.Offset, a.Length)
				if err != nil {
					return err
				}
				head := b.Snapshot.ReqHead
				if a.Side == "response" {
					head = b.Snapshot.RespHead
				}
				result = map[string]any{"binding_id": strconv.FormatInt(bid, 10), "head": head, "body": preview}
				return nil
			})
			if err != nil {
				return actool.Errorf(err.Error()), nil
			}
			return jsonResult(result)
		})
}
