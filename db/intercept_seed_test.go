package db

// 한국어 테스트 안내
// 내장 삭제 경로 정규식이 tool_input JSON 형태에서 삭제 API를 탐지하고 비슷한 일반 단어는 통과시키는지 검증한다.
// DB에 규칙을 등록하는 테스트가 아니라 정규식 자체의 hit/miss 사례를 확인하는 순수 단위 테스트다.

import (
	"regexp"
	"testing"
)

// 内置「删除类接口路径」规则匹配的是整个 tool_input JSON 串，因此用例直接以
// JSON 形态给出，与 Interceptor 实际拿到的 subject 一致。
// 한국어 검증 목적: 삭제 경로는 HTTP 메서드와 무관하게 잡되 delivery/details 같은 다른 단어를 잘못 잡지 않는 정규식 사례다.
func TestDeleteEndpointPathPattern(t *testing.T) {
	re := regexp.MustCompile(deleteEndpointPathPattern)

	hit := []string{
		`{"command":"curl -s 'http://t.com/api/user/delete?id=1'"}`,    // GET 打删除接口
		`{"command":"curl -X POST http://t.com/admin/delete -d id=1"}`, // POST 打删除接口
		`{"command":"curl 'http://t.com/api/deleteAll'"}`,
		`{"command":"curl 'http://t.com/api/delete_user?id=1'"}`,
		`{"command":"curl 'http://t.com/api/delete-user?id=1'"}`,
		`{"url":"http://t.com/api/remove?id=1"}`,
		`{"command":"curl http://t.com/files/unlink/3"}`,
		`{"command":"curl http://t.com/api/del?id=2"}`,
		`{"command":"curl -X POST http://t/v1/erase"}`,
		`{"command":"curl http://t/admin/destroyAll"}`, // v1 的路径规则不允许后缀，这里补上
	}
	for _, s := range hit {
		if !re.MatchString(s) {
			t.Errorf("应命中却放行: %s", s)
		}
	}

	// 动词后必须跟分隔符，避免 /delivery、/details 这类只读路径被误拦。
	miss := []string{
		`{"command":"curl 'http://t.com/api/delivery?id=1'"}`,
		`{"command":"curl 'http://t.com/order/details'"}`,
		`{"command":"curl 'http://t.com/api/delta/sync'"}`,
		`{"command":"curl 'http://t.com/user/delegate'"}`,
		`{"command":"curl 'http://delete.example.com/'"}`, // 删除动词出现在域名而非路径
		`{"command":"curl 'http://t.com/remote/status'"}`,
		`{"command":"nmap -p80 10.0.0.1"}`,
	}
	for _, s := range miss {
		if re.MatchString(s) {
			t.Errorf("误拦: %s", s)
		}
	}
}
