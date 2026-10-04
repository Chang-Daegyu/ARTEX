//go:build embedui

// [한국어 길잡이] 정적 웹 UI를 Go 바이너리에 포함
// embedui 빌드 태그가 있을 때만 선택되는 구현이다. server/webui/dist에 미리 만든 Next 정적 내보내기 결과가 필요하다.
// go:embed의 all: 접두사는 _next처럼 밑줄로 시작하는 자산도 포함한다. 이 지시문은 그대로 유지해야 한다.
// 공개 정적 파일을 제공하고 경로별 HTML 또는 루트 HTML로 대응한다. 실제 API 보호는 별도의 인증 미들웨어가 담당한다.
// 전체 모듈 설명과 파일별 읽기 순서: docs/ko/modules/server.md
package server

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// webuiDist holds the statically-exported frontend (built with
// `cd web && npm run build:static`, then copied into server/webui/dist). Compiled
// into the binary only when building with `-tags embedui`. The `all:` prefix is
// required so Next's `_next/` asset dir (leading underscore) is included.
//
//go:embed all:webui/dist
var webuiDist embed.FS

// webuiHandler serves the embedded SPA. Public (no JWT) — auth is enforced
// client-side and on /api. Serves the exported per-route index.html files and
// falls back to index.html so client-side routing still resolves unknown paths.
// [한국어 함수 설명] embed.FS에서 정적 export 루트를 꺼내 요청 경로별 파일/HTML을 찾고 필요한 경우 index.html로 돌아간다. API 인증은 이 파일 서버 바깥에서 적용된다.
func (s *Server) webuiHandler() http.Handler {
	root, err := fs.Sub(webuiDist, "webui/dist")
	if err != nil {
		panic(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" {
			p = "index.html"
		}
		// 1) exact file (assets: _next/*, favicon.ico, ...)
		// 2) route dir → <p>/index.html (trailingSlash export)
		// 3) <p>.html
		// 4) SPA 兜底 → index.html（交给客户端路由）
		for _, cand := range []string{p, p + "/index.html", p + ".html"} {
			if serveFileIfExists(w, r, root, cand) {
				return
			}
		}
		http.ServeFileFS(w, r, root, "index.html")
	})
}

// serveFileIfExists serves name from fsys when it exists as a regular file.
func serveFileIfExists(w http.ResponseWriter, r *http.Request, fsys fs.FS, name string) bool {
	f, err := fsys.Open(name)
	if err != nil {
		return false
	}
	st, statErr := f.Stat()
	_ = f.Close()
	if statErr != nil || st.IsDir() {
		return false
	}
	http.ServeFileFS(w, r, fsys, name)
	return true
}
