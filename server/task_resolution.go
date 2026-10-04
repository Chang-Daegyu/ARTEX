// [한국어 길잡이] 현재 역할이 사용할 LLM의 설명 API
// Planner·Worker·Main 등 역할별로 모델이 어디서 결정되는지 프로필 ID·모델·출처·가용성·이유를 반환한다.
// 실제 런타임과 같은 바인딩/작업 체인/전역 우선순위를 설명해 UI 설정과 실행 모델 사이의 혼동을 줄인다.
// 프로필이 없거나 DB 조회가 실패하는 경우를 구분하며, 여기서 모델을 호출하거나 프로필 체인을 변경하지 않는다.
// 전체 모듈 설명과 파일별 읽기 순서: docs/ko/modules/server.md
package server

import (
	"fmt"
	"net/http"

	"github.com/Autumn-27/artex/db"
)

type taskLLMResolution struct {
	ProfileID *int64 `json:"profile_id,omitempty"`
	Name      string `json:"name"`
	Format    string `json:"format"`
	Model     string `json:"model"`
	Source    string `json:"source"`
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

func (s *Server) resolutionFromProfile(p *db.LLMProfile, source string) taskLLMResolution {
	if p == nil {
		return taskLLMResolution{Source: source, Reason: "LLM 配置不存在"}
	}
	id := p.ID
	result := taskLLMResolution{
		ProfileID: &id,
		Name:      p.Name,
		Format:    p.Format,
		Model:     p.Model,
		Source:    source,
	}
	if p.APIKey == "" {
		result.Reason = "LLM 配置未设置 API Key"
		return result
	}
	if _, _, ok := s.providerForProfile(p.ID); !ok {
		result.Reason = "LLM 配置格式或参数无效"
		return result
	}
	result.Available = true
	return result
}

// resolveTaskRoleLLM mirrors taskLLMRuntime.current without creating a provider:
// role Agent binding -> explicit task chain -> active/global environment config.
// Database failures are returned rather than represented as an unavailable model.
// [한국어 함수 설명] 런타임 선택과 같은 우선순위를 화면에 설명 가능한 출처/가용성 구조로 변환한다. 관리 UI가 잘못된 모델 이름을 보여 주지 않도록 실제 선택 코드와 비교한다.
func (s *Server) resolveTaskRoleLLM(t *Task, agentKey string) (taskLLMResolution, error) {
	if s == nil || s.m == nil || s.m.pg == nil {
		return taskLLMResolution{}, fmt.Errorf("database unavailable")
	}
	a, err := s.m.pg.GetAgentByKey(agentKey)
	if err != nil {
		return taskLLMResolution{}, fmt.Errorf("load agent %q: %w", agentKey, err)
	}
	if a != nil && a.LLMProfileID != nil {
		p, err := s.m.pg.ProfileByID(*a.LLMProfileID)
		if err != nil {
			return taskLLMResolution{}, fmt.Errorf("load agent LLM profile: %w", err)
		}
		if p != nil {
			resolved := s.resolutionFromProfile(p, "agent_binding")
			if resolved.Available {
				return resolved, nil
			}
		}
	}
	state := t.llmStateSnapshot()
	if len(state.ProfileIDs) > 0 {
		if state.ActiveID == nil {
			return taskLLMResolution{Source: "task_chain", Reason: "任务 LLM 配置链额度已耗尽"}, nil
		}
		p, err := s.m.pg.ProfileByID(*state.ActiveID)
		if err != nil {
			return taskLLMResolution{}, fmt.Errorf("load task LLM profile: %w", err)
		}
		return s.resolutionFromProfile(p, "task_chain"), nil
	}
	p, err := s.m.pg.ActiveProfile()
	if err != nil {
		return taskLLMResolution{}, fmt.Errorf("load global LLM profile: %w", err)
	}
	if p != nil {
		resolved := s.resolutionFromProfile(p, "global_profile")
		if resolved.Available {
			return resolved, nil
		}
	}
	s.cfgMu.Lock()
	on, name, cfg := s.llmOn, s.llmProf, s.llmCfg
	s.cfgMu.Unlock()
	if on {
		if name == "" {
			name = "全局配置"
		}
		return taskLLMResolution{
			Name: name, Format: cfg.Provider(), Model: cfg.Model,
			Source: "environment", Available: true,
		}, nil
	}
	return taskLLMResolution{Source: "global", Reason: "没有可用的 LLM 配置"}, nil
}

func (s *Server) taskLLMResolutionHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := s.m.Task(r.PathValue("id"))
	if !ok {
		writeErr(w, 404, "task not found")
		return
	}
	roles := map[string]taskLLMResolution{}
	for responseKey, agentKey := range map[string]string{
		"mainagent": "mainagent",
		"planner":   "planner",
		"worker":    "worker",
	} {
		resolved, err := s.resolveTaskRoleLLM(t, agentKey)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		roles[responseKey] = resolved
	}
	writeJSON(w, http.StatusOK, roles)
}
