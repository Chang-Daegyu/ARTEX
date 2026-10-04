// [한국어 파일 안내] notify/http_test.go
// 로컬 가짜 HTTP 수신자로 공통 전송층의 상태 코드 분류·진단 길이 제한을 검사한다.
// 채널별 JSON 업무 오류와 HTTP 상태 오류는 별개이므로 공통 doJSON 경로를 독립 검증한다.
package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// 本文件覆盖 doJSON 的 HTTP 层错误分级。
//
// 为什么要单独测：各渠道适配器只管平台自己的业务错误码（钉钉 errcode、
// 飞书 code、Telegram ok 字段），而**HTTP 层**的分级是 doJSON 统一做的，
// 两者是两道独立的防线。少了这道，一个返回 503 的中转网关会被当成永久失败、
// 直接放弃重试；而一个 403 会被当成可重试、白白退避三轮。

// 한국어 해설: 지정한 상태 코드와 본문만 반환하는 임시 HTTP fixture를 만든다.
func replyServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// 한국어 해설: 2xx 성공, 408/429/5xx 일시 실패, 다른 4xx 영구 실패가 구분되고 코드가 진단에 남는지 확인한다.
func TestDoJSONClassifiesHTTPStatus(t *testing.T) {
	cases := []struct {
		name      string
		status    int
		permanent bool
	}{
		{"200 成功不算错误", 200, false},
		{"429 限流可重试", 429, false},
		{"408 请求超时可重试", 408, false},
		{"500 服务端错误可重试", 500, false},
		{"502 网关错误可重试", 502, false},
		{"503 服务不可用可重试", 503, false},
		{"400 参数错误永久失败", 400, true},
		{"401 鉴权失败永久失败", 401, true},
		{"403 禁止访问永久失败", 403, true},
		{"404 地址不存在永久失败", 404, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := replyServer(t, tc.status, `{"detail":"upstream says no"}`)
			_, err := doJSON(context.Background(), "GET", srv.URL, nil, nil)
			if tc.status < 300 {
				if err != nil {
					t.Fatalf("2xx 不应报错: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("非 2xx 应报错")
			}
			if got := IsPermanent(err); got != tc.permanent {
				t.Fatalf("HTTP %d 的 permanent 判定错误：期望 %v 得到 %v (%v)",
					tc.status, tc.permanent, got, err)
			}
			// 状态码必须出现在错误里，否则用户无从判断是自己配错了还是对端挂了。
			// 断言数字而不是 Go 的英文 StatusText：本包的文案是中文的
			// （与项目其它部分一致），数字才是语言无关、可稳定断言的部分。
			if !strings.Contains(err.Error(), strconv.Itoa(tc.status)) {
				t.Errorf("错误信息应带上 HTTP 状态码 %d，得到 %v", tc.status, err)
			}
		})
	}
}

// TestDoJSONIncludesResponseSnippet 覆盖 snippet：对端返回的错误说明要带回来，
// 否则用户只知道「失败了」，不知道对端为什么拒绝。
// 한국어 해설: 거절 응답의 짧은 실제 설명도 오류에 포함하여 원인을 파악할 수 있는지 확인한다.
func TestDoJSONIncludesResponseSnippet(t *testing.T) {
	srv := replyServer(t, 400, `{"error":"invalid webhook token"}`)
	_, err := doJSON(context.Background(), "GET", srv.URL, nil, nil)
	if err == nil {
		t.Fatal("应报错")
	}
	if !strings.Contains(err.Error(), "invalid webhook token") {
		t.Errorf("错误信息应带回对端的说明，得到 %v", err)
	}
}

// TestDoJSONSnippetIsSingleLineAndBounded 约束 snippet 的形态：
// 它对端响应会原样进 last_error 列与前端表格，多行/超长会破坏排版与载荷。
// 한국어 해설: 긴 다중 줄 오류 본문을 전달해 DB/UI에 들어갈 메시지가 한 줄·제한 길이로 줄어드는지 확인한다.
func TestDoJSONSnippetIsSingleLineAndBounded(t *testing.T) {
	// 带换行、制表符与 5000 字符超长内容的响应。
	long := strings.Repeat("x", 5000)
	srv := replyServer(t, 500, "line1\nline2\r\n\tline3 "+long)
	_, err := doJSON(context.Background(), "GET", srv.URL, nil, nil)
	if err == nil {
		t.Fatal("应报错")
	}
	msg := err.Error()
	if strings.ContainsAny(msg, "\r\n\t") {
		t.Errorf("错误信息应压成单行，得到 %q", msg)
	}
	// snippet 上限 200 字符 + 固定前缀，总量必须远小于原始响应。
	if len(msg) > 400 {
		t.Errorf("错误信息过长（%d 字节），应被 snippet 截断: %q", len(msg), msg)
	}
}

// TestDoJSONRejectsOversizedResponse 确认读取有上限：对端异常返回超大内容时
// 不能把整个响应读进内存（投递历史里每一条都会存一份 last_error）。
// 한국어 해설: 대형 오류 응답을 전체 저장하지 않고 제한한 진단만 반환하는지 확인한다. 크기 자체를 거절하는 HTTP 정책 테스트는 아니다.
func TestDoJSONRejectsOversizedResponse(t *testing.T) {
	huge := strings.Repeat("A", 1<<20) // 1 MiB
	srv := replyServer(t, 400, huge)
	_, err := doJSON(context.Background(), "GET", srv.URL, nil, nil)
	if err == nil {
		t.Fatal("应报错")
	}
	if len(err.Error()) > 400 {
		t.Errorf("超大响应应被限长读取并截断，错误信息长度 %d", len(err.Error()))
	}
}
