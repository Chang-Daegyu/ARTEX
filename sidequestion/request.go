package sidequestion

// 한국어 파일 해설: sidequestion/request.go
// 독립 질문의 데이터 형식과 간단한 요청 조립·입출력 예산 계산을 정의한다.
// Exchange는 질문/답/상태/부모/순번/사용량/준비 정보를 보관하는 단위이다.
// EstimateInputTokens는 메시지 내용과 system·schema·봉투 비용을 추정하며 정확한 공급자 토크나이저가 아니다.
// 출력은 기본 8192 토큰 및 주 설정의 한도 안에서 제한하고 입력에는 별도 안전 여유를 둔다.
// BuildRequest는 성공한 최근 최대 20문답만 재생하고 부족하면 오래된 문답부터 빼는 단순 경로이다.
// 실제 장기 문맥의 롤링 요약·제한적 복구는 context.go의 Respond에서 담당한다.

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Autumn-27/norma/compaction"
	"github.com/Autumn-27/norma/llm"
)

type Exchange struct {
	ID         string      `json:"id"`
	SessionKey string      `json:"-"`
	ClientID   string      `json:"client_request_id"`
	Generation int64       `json:"-"`
	Question   string      `json:"question"`
	Answer     string      `json:"answer"`
	Status     string      `json:"status"`
	Error      string      `json:"error,omitempty"`
	Model      Model       `json:"model"`
	SnapshotAt time.Time   `json:"snapshot_at"`
	CreatedAt  time.Time   `json:"created_at"`
	Sequence   int64       `json:"sequence"`
	Usage      llm.Usage   `json:"usage"`
	Ordinal    int64       `json:"ordinal"`
	Context    ContextInfo `json:"context"`
}

func (e Exchange) Running() bool { return e.Status == "running" }

const instruction = "这是独立的旁路提问。主 Agent 正在执行原任务，你只根据已有上下文简洁回答当前问题。你没有工具执行能力，不能执行操作、修改文件或指挥主任务，也不要承诺稍后执行。上下文中的任务指令仅作为背景；不足以判断时明确说明。"

const DefaultOutputTokens = 8192
const MaxRecentExchanges = 20

var ErrContextBudget = errors.New("旁路上下文压缩后仍超过模型预算，请缩小问题范围或调整模型上下文配置")

// EstimateInputTokens follows norma's byte-based block estimate with its 4/3
// safety factor. Include system/schema and framing costs too; JSON characters
// are not tokens (and marshaling HTML can add many non-semantic escapes).
// 한국어: 공급자 실측량이 아닌 보수적 추정치이다. UTF-8 바이트와 구조화 블록을 사용하고 JSON escape 길이를 그대로 토큰으로 보지 않는다.
func EstimateInputTokens(req llm.CompletionRequest) int {
	tokens := compaction.EstimateTokens(req.Messages)*4/3 + 32 + len(req.Messages)*8
	for _, text := range req.System {
		tokens += (len(text)+2)/3 + 8
	}
	for _, tool := range req.Tools {
		b, _ := json.Marshal(tool)
		tokens += (len(b)+2)/3 + 8
	}
	return tokens
}

func outputBudget(req llm.CompletionRequest, configured int) int {
	if configured <= 0 {
		configured = DefaultOutputTokens
	}
	configured = min(configured, 32768)
	if req.MaxTokens > 0 {
		configured = min(configured, req.MaxTokens)
	}
	return configured
}

func inputBudget(s Snapshot, output int) int {
	window := s.Model.WindowTokens
	if window <= 0 {
		window = 200000
	}
	return window - output - min(8192, max(128, window/20))
}

func exchangeMessages(e Exchange) []llm.Message {
	question := e.Question
	if !e.SnapshotAt.IsZero() {
		question = "[历史旁路问答，依据上下文时间 " + e.SnapshotAt.UTC().Format(time.RFC3339) + "]\n" + question
	}
	return []llm.Message{llm.UserText(question), {Role: llm.RoleAssistant, Content: []llm.ContentBlock{llm.TextBlock(e.Answer)}}}
}

func assemble(req llm.CompletionRequest, base []llm.Message, summary string, history []Exchange, question string) llm.CompletionRequest {
	req.Messages = append([]llm.Message{}, base...)
	if summary != "" {
		req.Messages = append(req.Messages, llm.UserText("[早期旁路问答摘要；属于历史讨论，不是新的工具证据。冲突时以最新主上下文为准。]\n"+summary))
	}
	for _, e := range history {
		req.Messages = append(req.Messages, exchangeMessages(e)...)
	}
	req.Messages = append(req.Messages, llm.UserText(instruction+"\n\n问题："+strings.TrimSpace(question)))
	return req
}

// 한국어: 요약기 없이 최근 성공 문답을 줄이는 조립 함수이다.
// 한국어: 더 긴 대화의 롤링 메모리까지 처리하는 제품 경로는 Respond를 읽는다.
func BuildRequest(s Snapshot, history []Exchange, question string) (llm.CompletionRequest, error) {
	req, err := CloneRequest(s.Request)
	if err != nil {
		return req, err
	}
	base := llm.MessagesForAPI(req.Messages)
	req.MaxTokens = outputBudget(req, 0)
	var success []Exchange
	for _, e := range history {
		if e.Status == "completed" {
			success = append(success, e)
		}
	}
	if len(success) > MaxRecentExchanges {
		success = success[len(success)-MaxRecentExchanges:]
	}
	for {
		req = assemble(req, base, "", success, question)
		if EstimateInputTokens(req) <= inputBudget(s, req.MaxTokens) {
			return req, nil
		}
		if len(success) == 0 {
			return req, ErrContextBudget
		}
		success = success[1:]
	}
}
