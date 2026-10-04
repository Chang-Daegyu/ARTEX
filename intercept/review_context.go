// [한국어 파일 안내] intercept/review_context.go
// 모델 심사기에 보낼 현재 도구 호출의 입력 봉투를 만든다.
// 실행 감사 기록과 심사 입력을 분리하며, 명시적으로 선택한 실제 사용자 메시지만 background에 포함한다.
// worker 요약·전체 탐색 상태·이전 도구 결과를 자동으로 배경에 섞지 않고 현재 arguments는 잘리지 않은 JSON으로 보존한다.
package intercept

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const reviewTextLimit = 4000

const BackgroundUserMessage = "user_message"

// ReviewBackground is explicitly bound from the current human message. Generated
// Worker summaries are not accepted. Background cannot override review policy.
// 한국어 자료형: 프로그램이 선택한 배경의 출처·본문·잘림 여부다. 이 배경이 승인 정책을 덮어쓸 권한은 없다.
type ReviewBackground struct {
	Source    string `json:"source"`
	Text      string `json:"text"`
	Truncated bool   `json:"truncated,omitempty"`
}

// ReviewInput contains only the current call and explicitly selected background.
// Execution history and call correlation belong to the separate audit record.
// 한국어 자료형: 현재 도구 이름, 완전한 JSON 인자, 작업 디렉터리와 선택적 사용자 배경만 담는 심사 계약이다.
type ReviewInput struct {
	Version    int               `json:"version"`
	WorkingDir string            `json:"working_directory,omitempty"`
	Background *ReviewBackground `json:"background,omitempty"`
	Tool       string            `json:"tool_name"`
	Arguments  json.RawMessage   `json:"arguments"`
}

type reviewContextKey struct{}
type reviewEnvironment struct {
	workingDir string
	background ReviewBackground
}

// WithReviewContext explicitly binds the permitted background for one run. Never
// fall back to the raw turn transcript: it may contain the full scheduler prompt.
// This is application wiring, not a model-callable tool.
// 한국어 해설: 작업 디렉터리와 출처가 지정된 배경을 context에 묶는다. 이 함수는 모델 도구가 아니라 애플리케이션 조립용이다.
func WithReviewContext(ctx context.Context, workingDir string, background ReviewBackground) context.Context {
	return context.WithValue(ctx, reviewContextKey{}, reviewEnvironment{workingDir, background})
}

// WithReviewWorkingDirectory preserves only explicitly selected background.
// Chat runs can be human-initiated or scheduled, so the Agent must not infer
// message provenance from the text it receives.
// 한국어 해설: 기존에 선택한 배경 출처를 유지한 채 디렉터리만 갱신한다. 전달 문자열을 사용자 원문이라고 추측하지 않는다.
func WithReviewWorkingDirectory(ctx context.Context, workingDir string) context.Context {
	env, _ := ctx.Value(reviewContextKey{}).(reviewEnvironment)
	env.workingDir = workingDir
	return context.WithValue(ctx, reviewContextKey{}, env)
}

// 한국어 해설: 유효한 현재 인자를 복사하여 version=4 입력을 만든다. background는 user_message 출처만 허용하고 4,000바이트로 제한한다.
func BuildReviewInput(ctx context.Context, tool string, arguments json.RawMessage) (ReviewInput, error) {
	if !json.Valid(arguments) {
		return ReviewInput{}, fmt.Errorf("工具参数不是有效 JSON")
	}
	in := ReviewInput{Version: 4, Tool: tool, Arguments: append(json.RawMessage(nil), arguments...)}
	if env, ok := ctx.Value(reviewContextKey{}).(reviewEnvironment); ok {
		in.WorkingDir = env.workingDir
		background := env.background
		if background.Source == BackgroundUserMessage && strings.TrimSpace(background.Text) != "" {
			var cut bool
			background.Text, cut = bounded(background.Text, reviewTextLimit)
			background.Truncated = background.Truncated || cut
			in.Background = &background
		}
	}
	return in, nil
}
