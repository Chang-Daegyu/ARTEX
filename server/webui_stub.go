//go:build !embedui

// [한국어 길잡이] 정적 UI를 포함하지 않는 개발용 구현
// embedui 태그가 없으면 선택된다. 같은 webuiHandler 이름을 제공하지만 UI 요청에는 안내용 404를 반환한다.
// 개발 중 Next 서버를 별도로 실행하거나 정적 파일을 만들고 embedui 태그로 빌드해야 화면을 볼 수 있다.
// webui_embed.go와는 빌드 조건이 상호 배타적이므로 두 함수를 동시에 컴파일하지 않는다.
// 전체 모듈 설명과 파일별 읽기 순서: docs/ko/modules/server.md
package server

import "net/http"

// webuiHandler is the no-embed stub (default build). The frontend is NOT bundled
// into this binary — run it separately with `cd web && npm run dev` during
// development. Build the bundled single-binary with:
//
//	cd web && npm run build:static     # produces web/out
//	cp -r web/out server/webui/dist    # (or use the build script)
//	go build -tags embedui ./cmd/artex
func (s *Server) webuiHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "前端未内嵌到此二进制（开发用 next dev；发布用 -tags embedui 构建）", http.StatusNotFound)
	})
}
