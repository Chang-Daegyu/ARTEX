// [한국어 파일 안내] traffic/traffic_store_test.go
// 신규 SQLite+blob 저장 형식의 기록·본문 검색·목록 필터·동시 쓰기 계약을 검증한다.
// newFlow로 메모리 요청/응답을 만들고 임시 저장소의 실제 SQL 및 파일을 검사한다. 외부 대상 스캔을 실행하지 않는다.
// 미리보기·전체 본문·색인 텍스트의 범위가 서로 다름을 테스트로 따라 읽을 수 있다.
package traffic

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	mproxy "github.com/lqqyt2423/go-mitmproxy/proxy"
)

// flowOpt tweaks the synthetic flow built by newFlow.
type flowOpt func(*mproxy.Flow)

// 한국어 해설: 합성 응답의 Content-Type만 바꿔 텍스트/바이너리 분류 경로를 선택한다.
func withRespType(ct string) flowOpt {
	return func(f *mproxy.Flow) { f.Response.Header.Set("Content-Type", ct) }
}

// newFlow builds the minimal flow record() needs: a request with a URL, method
// and body, plus a response with a status and body.
// 한국어 해설: record가 필요로 하는 URL·메서드·헤더·본문만 갖춘 가짜 프록시 교환을 만든다.
func newFlow(host, method, path string, reqBody, respBody []byte, opts ...flowOpt) *mproxy.Flow {
	u, err := url.Parse("http://" + host + path)
	if err != nil {
		panic(err)
	}
	f := &mproxy.Flow{
		Request: &mproxy.Request{
			Method: method,
			URL:    u,
			Proto:  "HTTP/1.1",
			Header: http.Header{"Content-Type": []string{"application/json"}},
			Body:   reqBody,
		},
		Response: &mproxy.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       respBody,
		},
	}
	for _, o := range opts {
		o(f)
	}
	return f
}

// 한국어 해설: 테스트마다 임시 트래픽 디렉터리를 열고 정리 시 Close를 호출하도록 등록한다.
func openTraffic(t *testing.T) (*Traffic, string) {
	t.Helper()
	dir := t.TempDir()
	tr, err := Open(dir, "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tr.Close() })
	return tr, dir
}

// 한국어 해설: 한 건만 저장한 fixture에서 교환 ID를 읽어 이후 본문/메타데이터 검증에 사용한다.
func onlyExchangeID(t *testing.T, tr *Traffic) string {
	t.Helper()
	var id string
	if err := tr.DB().QueryRow(`SELECT id FROM exchanges`).Scan(&id); err != nil {
		t.Fatalf("读取 exchange id: %v", err)
	}
	return id
}

// TestRecordKeepsBodiesInIndex is the core of the storage change: a recorded
// exchange produces no per-request directory at all, and its bodies are served
// back out of SQLite.
// 한국어 해설: 신규 기록이 호스트별 요청 폴더를 만들지 않고 SQLite에서 요청 Host와 본문까지 되돌려 주는지 확인한다.
func TestRecordKeepsBodiesInIndex(t *testing.T) {
	tr, dir := openTraffic(t)

	tr.record(newFlow("api.example.com", "POST", "/v1/login",
		[]byte(`{"user":"admin","password":"P@ssw0rd"}`),
		[]byte(`{"token":"abc123","note":"内网测试账号"}`)))

	// The URL-mirroring tree is gone: no host directory, no nested path segments.
	if _, err := os.Stat(filepath.Join(dir, "api.example.com")); !os.IsNotExist(err) {
		t.Fatalf("record 仍在磁盘上创建 host 目录（stat err=%v）", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "_") {
			t.Fatalf("data 目录下出现非内部目录 %q，说明仍在写文件树", e.Name())
		}
	}

	id := onlyExchangeID(t, tr)
	req, resp, err := tr.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"POST /v1/login HTTP/1.1", "Host: api.example.com", `"password":"P@ssw0rd"`} {
		if !strings.Contains(req, want) {
			t.Fatalf("请求原文缺少 %q，实际：\n%s", want, req)
		}
	}
	for _, want := range []string{"HTTP 200", `"token":"abc123"`, "内网测试账号"} {
		if !strings.Contains(resp, want) {
			t.Fatalf("响应原文缺少 %q，实际：\n%s", want, resp)
		}
	}
}

// TestFullTextSearchMatchesBodies covers what the trigram index buys over the
// previous URL-only search: arbitrary substrings and CJK, across request and
// response bodies.
// 한국어 해설: 본문 전체 단어·단어 내부 부분 문자열·중국어가 검색되고 3문자 미만 도구 검색은 명시적으로 실패하는지 검증한다.
func TestFullTextSearchMatchesBodies(t *testing.T) {
	tr, _ := openTraffic(t)
	if !tr.fts {
		t.Skip("驱动未启用 FTS5")
	}
	const host = "api.example.com"
	tr.record(newFlow(host, "POST", "/v1/login",
		[]byte(`{"user":"admin","password":"P@ssw0rd"}`),
		[]byte(`{"token":"abc123","note":"内网测试账号"}`)))
	tr.record(newFlow(host, "GET", "/v1/health", nil, []byte(`{"status":"ok"}`)))

	hits := func(term string) int {
		t.Helper()
		rows, err := tr.query(host, "", term, 0, 10)
		if err != nil {
			t.Fatalf("按正文搜索 %q 出错：%v", term, err)
		}
		return len(rows)
	}
	if n := hits("password"); n != 1 {
		t.Fatalf("搜 password 命中 %d 条，应为 1", n)
	}
	// Substring inside a token — the default unicode61 tokenizer cannot do this.
	if n := hits("ssw0r"); n != 1 {
		t.Fatalf("搜子串 ssw0r 命中 %d 条，应为 1", n)
	}
	if n := hits("内网测试"); n != 1 {
		t.Fatalf("搜中文命中 %d 条，应为 1", n)
	}
	if n := hits("nonexistent-marker"); n != 0 {
		t.Fatalf("无关关键词命中 %d 条，应为 0", n)
	}

	// Too-short terms are reported, not silently treated as "no match".
	if _, err := tr.query(host, "", "ab", 0, 10); err == nil {
		t.Fatal("两字符正文关键词应返回明确错误")
	}
}

// TestLargeBodySpillsButStaysSearchable is the case that motivated indexing from
// memory: the body lives in the blob store, only a preview is inline, and the
// part past the preview is still findable.
// 한국어 해설: 큰 본문은 해시 파일로 빠져도 미리보기 이후의 색인 범위가 검색되고 BlobRange로 해당 원문을 읽을 수 있는지 확인한다.
func TestLargeBodySpillsButStaysSearchable(t *testing.T) {
	tr, dir := openTraffic(t)
	if !tr.fts {
		t.Skip("驱动未启用 FTS5")
	}
	const host = "dump.example.com"
	const marker = "DB_PASSWORD=hunter2"
	// Marker sits far past blobPreview, so only the full-text index can find it.
	big := []byte(strings.Repeat("-- MySQL dump\n", maxInlineBody/14+2000) + marker)
	if len(big) <= maxInlineBody+blobPreview {
		t.Fatalf("测试数据不够大：%d 字节", len(big))
	}
	tr.record(newFlow(host, "GET", "/backup.sql", nil, big, withRespType("application/sql")))

	// Stored under a single bucket level, named by hash.
	var hash string
	if err := tr.DB().QueryRow(`SELECT resp_blob FROM exchange_bodies`).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	if len(hash) != 64 {
		t.Fatalf("resp_blob=%q，应为 64 位 sha256", hash)
	}
	blob := filepath.Join(dir, "_blobs", "sha256", hash[:2], hash+".bin")
	st, err := os.Stat(blob)
	if err != nil {
		t.Fatalf("blob 未落盘到单层桶 %s：%v", blob, err)
	}
	if st.Size() != int64(len(big)) {
		t.Fatalf("blob 大小 %d，应为 %d", st.Size(), len(big))
	}

	// The reference is registered, which is what GC consults.
	var refs int
	if err := tr.DB().QueryRow(`SELECT COUNT(*) FROM blob_refs WHERE hash=?`, hash).Scan(&refs); err != nil {
		t.Fatal(err)
	}
	if refs != 1 {
		t.Fatalf("blob_refs 行数 %d，应为 1", refs)
	}

	// Inline: a readable preview plus the pointer, not the whole body.
	_, resp, err := tr.Get(onlyExchangeID(t, tr))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp, "-- MySQL dump") {
		t.Fatalf("响应缺少头部预览：\n%s", clip(resp, 300))
	}
	if !strings.Contains(resp, "@blob sha256:"+hash) {
		t.Fatalf("响应缺少 blob 指针：\n%s", clip(resp, 300))
	}
	if strings.Contains(resp, marker) {
		t.Fatal("预览不应包含超出 blobPreview 的内容")
	}
	if len(resp) > blobPreview*2 {
		t.Fatalf("内联内容 %d 字节，远超预览上限", len(resp))
	}

	// Searchable despite living on disk — the index was fed from memory.
	rows, err := tr.query(host, "", marker, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("超大正文中的关键词命中 %d 条，应为 1", len(rows))
	}

	// And retrievable in pages.
	data, total, err := tr.BlobRange(hash, int64(len(big)-len(marker)), 100)
	if err != nil {
		t.Fatal(err)
	}
	if total != int64(len(big)) {
		t.Fatalf("BlobRange total=%d，应为 %d", total, len(big))
	}
	if string(data) != marker {
		t.Fatalf("BlobRange 读到 %q，应为 %q", data, marker)
	}
	if _, _, err := tr.BlobRange("../../etc/passwd", 0, 10); err == nil {
		t.Fatal("非法 hash 应被拒绝")
	}
}

// TestBinaryBodyStaysOutOfIndex keeps the index spend on things worth searching:
// binary payloads contribute nothing but a type tag.
// 한국어 해설: 바이너리 본문의 marker는 FTS에 들어가지 않고 미리보기는 형식·magic 표시로 대체되는지 확인한다.
func TestBinaryBodyStaysOutOfIndex(t *testing.T) {
	tr, _ := openTraffic(t)
	if !tr.fts {
		t.Skip("驱动未启用 FTS5")
	}
	const host = "cdn.example.com"
	const marker = "SECRETINIMAGE"
	png := append([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00},
		[]byte(strings.Repeat("x", maxInlineBody)+marker)...)
	tr.record(newFlow(host, "GET", "/logo.png", nil, png, withRespType("image/png")))

	rows, err := tr.query(host, "", marker, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("二进制正文不应进入全文索引，却命中 %d 条", len(rows))
	}
	_, resp, err := tr.Get(onlyExchangeID(t, tr))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp, "[binary image/png") || !strings.Contains(resp, "magic=89504e47") {
		t.Fatalf("二进制正文应展示类型与魔数，实际：\n%s", clip(resp, 300))
	}
}

// TestGetFallsBackToLegacyTree keeps pre-migration captures readable: their rows
// carry a path and their bodies are still .http files on disk.
// 한국어 해설: path가 남아 있는 과거 행은 .http 파일을 읽어 이전 수집 기록을 유지하는지 검증한다.
func TestGetFallsBackToLegacyTree(t *testing.T) {
	tr, dir := openTraffic(t)
	const host = "old.example.com"
	const id = "1-0001"
	rel := filepath.Join(host, "GET", id)
	exDir := filepath.Join(dir, rel)
	if err := os.MkdirAll(exDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(exDir, "request.http"), []byte("GET / HTTP/1.1\nHost: old.example.com\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(exDir, "response.http"), []byte("HTTP 200\n\nlegacy body"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := tr.DB().Exec(`INSERT INTO exchanges(id,ts,host,method,url_template,url,status,content_type,req_len,resp_len,path)
VALUES(?,?,?,?,?,?,?,?,?,?,?)`, id, 1, host, "GET", "/", "http://"+host+"/", 200, "text/html", 0, 11, rel); err != nil {
		t.Fatal(err)
	}

	req, resp, err := tr.Get(id)
	if err != nil {
		t.Fatalf("历史记录应仍可读取：%v", err)
	}
	if !strings.Contains(req, "Host: old.example.com") {
		t.Fatalf("历史请求原文错误：%q", req)
	}
	if !strings.Contains(resp, "legacy body") {
		t.Fatalf("历史响应原文错误：%q", resp)
	}
}

// TestGCCollectsBlobsAndEmptyBuckets covers both halves of the collector: the
// reference lookup now comes from blob_refs, and emptied buckets are removed
// instead of accumulating forever.
// 한국어 해설: 마지막 참조 삭제 후 blob 파일과 빈 버킷, 관련 본문·FTS 행이 함께 정리되는지 확인한다.
func TestGCCollectsBlobsAndEmptyBuckets(t *testing.T) {
	tr, dir := openTraffic(t)
	const host = "dump.example.com"
	big := []byte(strings.Repeat("A", maxInlineBody+1024))
	tr.record(newFlow(host, "GET", "/big.bin", nil, big, withRespType("application/sql")))

	var hash string
	if err := tr.DB().QueryRow(`SELECT resp_blob FROM exchange_bodies`).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	bucket := filepath.Join(dir, "_blobs", "sha256", hash[:2])
	if _, err := os.Stat(filepath.Join(bucket, hash+".bin")); err != nil {
		t.Fatal(err)
	}

	if n, err := tr.DeleteHostsExact([]string{host}); err != nil || n != 1 {
		t.Fatalf("DeleteHostsExact=(%d,%v)，应为 (1,nil)", n, err)
	}
	if _, err := os.Stat(filepath.Join(bucket, hash+".bin")); !os.IsNotExist(err) {
		t.Fatalf("失去引用的 blob 未被回收：%v", err)
	}
	if _, err := os.Stat(bucket); !os.IsNotExist(err) {
		t.Fatalf("空桶目录未被清理：%v", err)
	}
	// Bodies and full-text rows go with the exchange.
	for _, q := range []string{
		`SELECT COUNT(*) FROM exchange_bodies`,
		`SELECT COUNT(*) FROM blob_refs`,
	} {
		var c int
		if err := tr.DB().QueryRow(q).Scan(&c); err != nil {
			t.Fatal(err)
		}
		if c != 0 {
			t.Fatalf("%s = %d，应为 0", q, c)
		}
	}
	if tr.fts {
		var c int
		if err := tr.DB().QueryRow(`SELECT COUNT(*) FROM ex_fts WHERE ex_fts MATCH ?`, ftsQuote("AAAA")).Scan(&c); err != nil {
			t.Fatal(err)
		}
		if c != 0 {
			t.Fatalf("全文索引残留 %d 条", c)
		}
	}
}

// TestPageSearchesBodies checks the UI-facing search box picks up the full-text
// index too, not just metadata columns.
// 한국어 해설: 웹 목록의 넓은 검색창이 메타데이터뿐 아니라 캡처된 본문도 찾는지 확인한다.
func TestPageSearchesBodies(t *testing.T) {
	tr, _ := openTraffic(t)
	if !tr.fts {
		t.Skip("驱动未启用 FTS5")
	}
	tr.record(newFlow("api.example.com", "POST", "/v1/login", nil, []byte(`{"error":"invalid credentials"}`)))
	tr.record(newFlow("api.example.com", "GET", "/v1/health", nil, []byte(`{"status":"ok"}`)))

	rows, total, err := tr.Page(PageQuery{Query: "invalid credentials", RespMin: -1, RespMax: -1}, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(rows) != 1 {
		t.Fatalf("正文关键词命中 total=%d rows=%d，应为 1/1", total, len(rows))
	}
	// Metadata matching still works alongside it.
	if _, total, err := tr.Page(PageQuery{Query: "health", RespMin: -1, RespMax: -1}, 0, 100); err != nil || total != 1 {
		t.Fatalf("URL 关键词 total=%d err=%v，应为 1", total, err)
	}
}

// TestPageFiltersAndSort covers the issue #177 additions: status-class/exact
// filtering, response-size bounds, path (url_template) filtering, and
// server-side sorting by resp_len.
// 한국어 해설: 상태 코드/구간·응답 길이·경로 조건과 길이 오름/내림차순 정렬을 서로 다른 세 교환으로 검증한다.
func TestPageFiltersAndSort(t *testing.T) {
	tr, _ := openTraffic(t)
	status := func(code int) flowOpt { return func(f *mproxy.Flow) { f.Response.StatusCode = code } }
	// Three exchanges with distinct status codes and response sizes.
	tr.record(newFlow("api.example.com", "GET", "/api/users", nil, make([]byte, 10), status(200)))
	tr.record(newFlow("api.example.com", "GET", "/api/admin", nil, make([]byte, 100), status(404)))
	tr.record(newFlow("api.example.com", "GET", "/api/users/1", nil, make([]byte, 50), status(500)))

	// Status class band.
	if rows, _, err := tr.Page(PageQuery{Status: "4xx", RespMin: -1, RespMax: -1}, 0, 100); err != nil || len(rows) != 1 || rows[0].Status != 404 {
		t.Fatalf("status=4xx 应命中 1 条 404，得 %d 条 err=%v", len(rows), err)
	}
	// Exact status.
	if rows, _, err := tr.Page(PageQuery{Status: "500", RespMin: -1, RespMax: -1}, 0, 100); err != nil || len(rows) != 1 || rows[0].Status != 500 {
		t.Fatalf("status=500 应命中 1 条，得 %d 条 err=%v", len(rows), err)
	}
	// Response-size lower bound (>=60 keeps only the 100-byte row).
	if rows, _, err := tr.Page(PageQuery{RespMin: 60, RespMax: -1}, 0, 100); err != nil || len(rows) != 1 || rows[0].RespLen != 100 {
		t.Fatalf("resp_min=60 应命中 1 条 100B，得 %d 条 err=%v", len(rows), err)
	}
	// Path (url_template) filter narrows to the /api/admin exchange.
	if rows, _, err := tr.Page(PageQuery{Path: "/api/admin", RespMin: -1, RespMax: -1}, 0, 100); err != nil || len(rows) != 1 || rows[0].Status != 404 {
		t.Fatalf("path=/api/admin 应命中 1 条，得 %d 条 err=%v", len(rows), err)
	}
	// Sort by response length, ascending then descending.
	asc, _, err := tr.Page(PageQuery{RespMin: -1, RespMax: -1, Sort: "resp_len", Order: "asc"}, 0, 100)
	if err != nil || len(asc) != 3 {
		t.Fatalf("resp_len asc 应返回 3 条，得 %d 条 err=%v", len(asc), err)
	}
	if asc[0].RespLen != 10 || asc[1].RespLen != 50 || asc[2].RespLen != 100 {
		t.Fatalf("resp_len asc 顺序错误：%d,%d,%d", asc[0].RespLen, asc[1].RespLen, asc[2].RespLen)
	}
	desc, _, err := tr.Page(PageQuery{RespMin: -1, RespMax: -1, Sort: "resp_len", Order: "desc"}, 0, 100)
	if err != nil || len(desc) != 3 || desc[0].RespLen != 100 || desc[2].RespLen != 10 {
		t.Fatalf("resp_len desc 顺序错误 err=%v", err)
	}
}

// 한국어 해설: 같은 호스트의 다른 포트가 섞이지 않고 URL·IPv6 입력도 같은 검색 계약을 따르는지 확인한다.
func TestQueryHostPortAndURLForms(t *testing.T) {
	tr, _ := openTraffic(t)
	tr.record(newFlow("api.example.com:8082", "GET", "/admin", nil, []byte("8082")))
	tr.record(newFlow("api.example.com:8088", "GET", "/admin", nil, []byte("8088")))
	tr.record(newFlow("[2001:db8::1]:8443", "GET", "/admin", nil, []byte("8443")))

	for _, tc := range []struct {
		name string
		host string
		want int
	}{
		{name: "bare host", host: "api.example.com", want: 2},
		{name: "host and port", host: "api.example.com:8082", want: 1},
		{name: "full URL", host: "http://api.example.com:8088/admin", want: 1},
		{name: "IPv6 host and port", host: "[2001:db8::1]:8443", want: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := tr.query(tc.host, "", "", 0, 10)
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != tc.want {
				t.Fatalf("query(%q) returned %d rows, want %d", tc.host, len(rows), tc.want)
			}
		})
	}
}

// 한국어 해설: 대소문자 도메인·포트·대괄호 IPv6를 hostname/port 두 값으로 올바르게 나누는지 확인한다.
func TestNormalizeSearchHost(t *testing.T) {
	for _, tc := range []struct {
		raw, host, port string
	}{
		{raw: "API.Example.com", host: "api.example.com"},
		{raw: "api.example.com:8088", host: "api.example.com", port: "8088"},
		{raw: "https://[2001:db8::1]:8443/path", host: "2001:db8::1", port: "8443"},
		{raw: "[2001:db8::1]", host: "2001:db8::1"},
	} {
		host, port, err := normalizeSearchHost(tc.raw)
		if err != nil {
			t.Fatalf("normalizeSearchHost(%q): %v", tc.raw, err)
		}
		if host != tc.host || port != tc.port {
			t.Fatalf("normalizeSearchHost(%q)=(%q,%q), want (%q,%q)", tc.raw, host, port, tc.host, tc.port)
		}
	}
}

// TestTruncateUTF8 guards the preview cut: never split a multi-byte rune.
// 한국어 해설: 각 바이트 경계에서 한글과 같은 다중 바이트 문자가 쪼개지지 않는 짧은 prefix를 반환하는지 검사한다.
func TestTruncateUTF8(t *testing.T) {
	s := "内网测试账号"
	for n := 0; n <= len(s); n++ {
		got := truncateUTF8([]byte(s), n)
		if !strings.HasPrefix(s, got) {
			t.Fatalf("n=%d 截断结果 %q 不是原串前缀", n, got)
		}
		if len(got) > n {
			t.Fatalf("n=%d 截断后 %d 字节，超出上限", n, len(got))
		}
	}
	if got := truncateUTF8([]byte("abc"), 10); got != "abc" {
		t.Fatalf("短于上限时应原样返回，得到 %q", got)
	}
}

// TestIsBinaryBody documents the classification: declared binary types, and the
// NUL backstop for anything mislabeled.
// 한국어 해설: 선언된 바이너리 MIME와 NUL 보완 검사, SQL/HTML/JSON 텍스트 예외를 표로 검증한다.
func TestIsBinaryBody(t *testing.T) {
	cases := []struct {
		ct   string
		body string
		want bool
	}{
		{"application/json", `{"a":1}`, false},
		{"text/html; charset=utf-8", "<html>", false},
		{"application/sql", "-- dump", false},
		{"", "plain text", false},
		{"image/png", "whatever", true},
		{"application/zip", "PK", true},
		{"APPLICATION/PDF", "%PDF", true},
		{"text/plain", "has\x00nul", true},
	}
	for _, c := range cases {
		if got := isBinaryBody(c.ct, []byte(c.body)); got != c.want {
			t.Errorf("isBinaryBody(%q, %q)=%v，应为 %v", c.ct, c.body, got, c.want)
		}
	}
}

// TestRecordConcurrent exercises the write path under contention: ids stay
// unique and every exchange lands in all three tables.
// 한국어 해설: 50개 goroutine의 합성 기록 후 교환과 본문 수가 모두 50인지 확인한다. 임의 고부하의 ID 충돌 불가능성을 증명하는 테스트는 아니다.
func TestRecordConcurrent(t *testing.T) {
	tr, _ := openTraffic(t)
	const n = 50
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			tr.record(newFlow("api.example.com", "GET", fmt.Sprintf("/item/%d", i),
				nil, fmt.Appendf(nil, `{"id":%d}`, i)))
		})
	}
	wg.Wait()
	var exchanges, bodies int
	if err := tr.DB().QueryRow(`SELECT COUNT(*) FROM exchanges`).Scan(&exchanges); err != nil {
		t.Fatal(err)
	}
	if err := tr.DB().QueryRow(`SELECT COUNT(*) FROM exchange_bodies`).Scan(&bodies); err != nil {
		t.Fatal(err)
	}
	if exchanges != n || bodies != n {
		t.Fatalf("并发写入后 exchanges=%d bodies=%d，应各为 %d", exchanges, bodies, n)
	}
}
