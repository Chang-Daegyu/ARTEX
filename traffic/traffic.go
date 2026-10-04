// Package traffic implements the request-recording subsystem (docs §10): an
// embedded go-mitmproxy proxy whose addon writes every target HTTP exchange into
// a human-browsable file tree, with a sidecar SQLite index for paged queries.
// Full capture, plaintext (no redaction), target HTTP only.
// [한국어 파일 안내] traffic/traffic.go
// HTTP 트래픽 수집기의 생성·기록·검색·삭제·공간 회수를 한 파일에서 연결한다.
// 현재 신규 기록은 SQLite의 exchanges/exchange_bodies와 대형 본문용 _blobs에 저장된다.
// 위 원본 패키지 주석의 요청별 파일 트리는 이전 저장 형식이며, 현재는 읽기/삭제 호환 경로로 남아 있다.
// 수집은 이 프록시를 통과하고 MITM 처리에 성공한 HTTP 교환에 한정된다. 투명 통과 호스트는 기록되지 않는다.
// 본문은 마스킹 없이 보관되며 SHA-256 파일명은 중복 제거용이다. 암호화나 취약점의 사실 검증을 뜻하지 않는다.
// 읽기 순서: Open → sink.Response → record/spill → query/Get/BlobRange → 삭제 stage/Commit/GC.
// 자세한 저장소 관계와 실패 시 동작은 docs/ko/modules/services.md를 함께 읽는다.
package traffic

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/norma/permission"
	actool "github.com/Autumn-27/norma/tool"
	mproxy "github.com/lqqyt2423/go-mitmproxy/proxy"
	_ "modernc.org/sqlite"
)

const indexSchema = `
CREATE TABLE IF NOT EXISTS exchanges (
  id           TEXT PRIMARY KEY,
  ts           INTEGER,
  host         TEXT,
  method       TEXT,
  url_template TEXT,
  url          TEXT,
  status       INTEGER,
  content_type TEXT,
  req_len      INTEGER,
  resp_len     INTEGER,
  path         TEXT
);
CREATE INDEX IF NOT EXISTS idx_ex_host ON exchanges(host);
CREATE INDEX IF NOT EXISTS idx_ex_tmpl ON exchanges(host, url_template);
CREATE INDEX IF NOT EXISTS idx_ex_ts   ON exchanges(ts);
CREATE INDEX IF NOT EXISTS idx_ex_status ON exchanges(status);
CREATE INDEX IF NOT EXISTS idx_ex_resp   ON exchanges(resp_len);

CREATE TABLE IF NOT EXISTS exchange_bodies (
  id        TEXT PRIMARY KEY,
  req_head  TEXT NOT NULL DEFAULT '',
  req_body  BLOB,
  req_blob  TEXT,
  resp_head TEXT NOT NULL DEFAULT '',
  resp_body BLOB,
  resp_blob TEXT
);

CREATE TABLE IF NOT EXISTS blob_refs (
  hash        TEXT NOT NULL,
  exchange_id TEXT NOT NULL,
  PRIMARY KEY (hash, exchange_id)
);
CREATE INDEX IF NOT EXISTS idx_blob_refs_ex ON blob_refs(exchange_id);
`

// ftsSchema is applied separately from indexSchema: a driver build without FTS5
// must degrade to "no full-text search" rather than take the whole recorder down.
// trigram (not the default unicode61) is required for two reasons this subsystem
// depends on: it matches arbitrary substrings — "ssw0r" finds "P@ssw0rd" — and it
// handles CJK, which unicode61 does not tokenize. Contentless (content=”) keeps
// only the index, since the text itself lives in exchange_bodies; contentless_delete
// lets rows be deleted without replaying the original text back in.
const ftsSchema = `CREATE VIRTUAL TABLE IF NOT EXISTS ex_fts USING fts5(
  content, tokenize='trigram', content='', contentless_delete=1
);`

const (
	// maxInlineBody is the cutoff between a body stored inline (SQLite column) and
	// one spilled to the content-addressed blob store.
	maxInlineBody = 256 * 1024
	// blobPreview is how much of a spilled body stays inline so readers can tell
	// what it is (JSON shape, SQL dump header, ZIP magic) without fetching it.
	blobPreview = 8 * 1024
	// maxIndexBody caps a single body's contribution to the full-text index. Text
	// is indexed in full below this; binary never reaches the index at all.
	maxIndexBody = 4 * 1024 * 1024
	// minTrigram is the shortest term the trigram tokenizer can match; shorter
	// queries fall back to metadata LIKE.
	minTrigram = 3
	// maxBlobRead caps one traffic_blob read so paging through a large body never
	// floods the agent's context.
	maxBlobRead = 8 * 1024
	// autoVacuumIncremental is SQLite's numeric value for auto_vacuum=incremental.
	autoVacuumIncremental = 2
	// reclaimChunkPages bounds how much index space one locked step returns to the
	// filesystem (pages are 4KiB, so ~32MB). Reclamation holds the write lock, and
	// record() runs before go-mitmproxy replies to the client, so an unbounded
	// pass would stall the very requests being recorded.
	reclaimChunkPages = 8192
	// reclaimMergePages bounds the full-text merge done alongside each chunk.
	// Deleting from a contentless_delete index only writes tombstones; merging is
	// what discards them, and left undone the index grows on every deletion.
	reclaimMergePages = 256
	// reclaimMergeSteps bounds how many merges one reclamation performs. fts5
	// offers no way to ask whether an index has settled — its special INSERT
	// registers a row change of its own, so change counting cannot tell a real
	// merge from a no-op — so the budget is simply spent down, and whatever is
	// left over falls to the next deletion.
	reclaimMergeSteps = 16
	// reclaimMaxSteps backstops the loop. Both merging and incremental_vacuum are
	// documented to stop making progress eventually, but a reclamation that cannot
	// converge must give up rather than spin holding the write lock.
	reclaimMaxSteps = 512
	// reclaimBudget caps one background reclamation. Deleting a busy host can free
	// gigabytes, and the next deletion resumes wherever this one stopped.
	reclaimBudget = 5 * time.Minute
)

// TrafficSearchDescription is persisted into the tool catalog for new and
// upgraded installations. Keep it in the traffic package so the runtime tool
// and the catalog migration cannot drift apart.
const TrafficSearchDescription = "查询记录代理已抓取的目标流量（必须指定 host；支持裸主机、主机:端口或完整 URL，可再按 URL 子串或正文关键词过滤）。指定端口时只返回该服务的流量，避免同一 IP 的不同端口串包。body_contains 会在已抓取的请求/响应头与正文中做全文搜索，支持任意子串和中文（至少 3 个字符）。仅返回极轻量索引(id/method/url/status/resp_len)，不含响应内容；结果非空后必须用 traffic_get 逐条核实请求/响应，再把确实支持当前漏洞的 ID 交给 bind_finding_traffic。默认只返回 3 条、每页最多 10 条；结果多时用 page 翻页。"

// Traffic runs the recording proxy and owns the file tree + index.
// 한국어 자료형: SQLite 인덱스, MITM 프록시, blob 파일, 기록/삭제 잠금과 회수 goroutine의 수명을 소유하는 중심 객체다.
type Traffic struct {
	dir   string
	addr  string
	db    *sql.DB
	wmu   sync.Mutex // serializes record() vs DeleteHost (incl. blob GC)
	seq   atomic.Int64
	proxy *mproxy.Proxy
	// fts reports whether the full-text index is available. False on a driver
	// build without FTS5: recording and metadata search still work, body search
	// degrades to unsupported rather than erroring.
	fts bool
	// reaping tracks background reclamation of legacy trees and index space, so
	// shutdown and tests can wait for it instead of racing it.
	reaping sync.WaitGroup
	// incrementalVacuum reports whether the index can return freed pages to the
	// filesystem on its own. False on a database created before this was the
	// default: PRAGMA incremental_vacuum is a silent no-op there, so deletions
	// reclaim nothing until an explicit full compaction converts the file.
	incrementalVacuum bool
	// reclaiming keeps a single background reclamation in flight. Concurrent ones
	// would only contend for wmu, never finish sooner.
	reclaiming atomic.Bool
	// closed is shut by Close so a reclamation abandons its remaining budget
	// instead of holding shutdown open for minutes. Nil on the zero value, which
	// stopping() treats as "not closing".
	closed    chan struct{}
	closeOnce sync.Once
	// pass is the set of hosts whose MITM interception failed for a proxy/protocol
	// reason; connections to them are tunneled transparently (fail-open) so the
	// request still reaches the target — unrecorded — instead of being killed.
	pass sync.Map // hostname(string) -> struct{}
	// upstream is the global egress proxy every captured request is forwarded
	// through (nil = dial targets directly). Both the intercepted and the
	// transparent-passthrough paths honor it (go-mitmproxy's getUpstreamConn),
	// so no host escapes it. Hot-swappable at runtime via SetUpstreamProxy.
	upstream atomic.Pointer[url.URL]
}

// Open initializes the traffic tree, blob store and SQLite index under dir.
// 한국어 해설: 저장 디렉터리와 SQLite를 열고 MITM 프록시를 구성한다. 반환만으로 리스닝하지 않으며 Start 호출이 필요하다.
// 연결별 busy_timeout을 DSN에 넣고, 출구 프록시는 환경값 대신 upstream 포인터의 현재 값을 사용한다.
func Open(dir, addr string) (*Traffic, error) {
	for _, d := range []string{dir, filepath.Join(dir, "_index"), filepath.Join(dir, "_blobs"), filepath.Join(dir, "_ca")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}
	// busy_timeout is a per-connection setting, so it belongs in the DSN rather
	// than in a one-off Exec: the pool opens connections on demand, and an Exec
	// only configures whichever one happened to serve it — leaving every other
	// connection to fail instantly the moment a writer holds the database.
	// The driver splits the DSN at the first '?', so a data directory containing
	// one would silently name a different file; that path falls back to a bare
	// DSN, where initIndex still applies the pragmas to its own connection.
	index := filepath.Join(dir, "_index", "index.sqlite")
	dsn := index
	if !strings.ContainsRune(index, '?') {
		dsn = "file:" + index + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	t := &Traffic{dir: dir, addr: addr, db: db, closed: make(chan struct{})}
	if err := t.initIndex(); err != nil {
		db.Close()
		return nil, err
	}

	p, err := mproxy.NewProxy(&mproxy.Options{
		Addr:        addr,
		SslInsecure: true,
		CaRootPath:  filepath.Join(dir, "_ca"),
	})
	if err != nil {
		db.Close()
		return nil, err
	}
	// Upstream selection. By default (no global egress proxy set) targets are
	// dialed DIRECTLY: go-mitmproxy's own default upstream uses
	// http.ProxyFromEnvironment, so an HTTP_PROXY/HTTPS_PROXY in the environment
	// (a VPN/system proxy) would make it forward target requests through that
	// external proxy — which can't reach the target → 502. Returning nil forces
	// a direct dial. When a global egress proxy IS configured (SetUpstreamProxy),
	// every captured request — intercepted AND transparently tunneled — is
	// forwarded through it instead, so no host leaks the real source IP.
	p.SetUpstreamProxy(func(*http.Request) (*url.URL, error) { return t.upstream.Load(), nil })
	// Fail-open: MITM every host by default, EXCEPT ones a prior request proved we
	// can't intercept without breaking (see maybePassthrough). Those are tunneled
	// transparently so the request still reaches the target instead of being killed.
	p.SetShouldInterceptRule(func(req *http.Request) bool {
		_, tunnel := t.pass.Load(hostOnly(req.Host))
		return !tunnel
	})
	p.AddAddon(&sink{t: t})
	t.proxy = p
	return t, nil
}

// initIndex applies the schema on a single pinned connection. Pinning is what
// makes auto_vacuum reliable: it can only be set while the database still holds
// no tables, and only the VACUUM that follows writes it into the file header —
// two steps a pooled *sql.DB is free to route to different connections, which
// would silently drop the setting.
//
// auto_vacuum=incremental is what lets a deletion hand freed pages back to the
// filesystem. Without it SQLite merely chains them onto its freelist, so the
// index file never shrinks no matter how much traffic is deleted — and since
// every body below maxInlineBody lives in that file, plus a trigram index
// roughly twice the size of the text it covers, a capture-heavy install ends up
// holding gigabytes for traffic it no longer has. On a database that already has
// tables the pragma is a documented no-op, so installs created before this keep
// auto_vacuum=0 until a full compaction converts the file; incrementalVacuum
// records that so reclaim can say so instead of pretending to reclaim.
// 한국어 해설: 한 연결을 고정하여 WAL·스키마·FTS5를 초기화한다. 새 DB에서만 incremental auto-vacuum을 설정한다.
// FTS5 생성 실패는 본문 검색 비활성화로 처리하므로 메타데이터 기록과 조회까지 중단되지는 않는다.
func (t *Traffic) initIndex() error {
	ctx := context.Background()
	conn, err := t.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	// Re-applied rather than left to the DSN so the bare-DSN fallback in Open is
	// still correct: journal_mode persists in the file header, which is what every
	// later connection reads.
	for _, p := range []string{`PRAGMA journal_mode=WAL`, `PRAGMA busy_timeout=5000`} {
		if _, err := conn.ExecContext(ctx, p); err != nil {
			return err
		}
	}
	var tables int
	if err := conn.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table'`).Scan(&tables); err != nil {
		return err
	}
	if tables == 0 {
		for _, p := range []string{`PRAGMA auto_vacuum=incremental`, `VACUUM`} {
			if _, err := conn.ExecContext(ctx, p); err != nil {
				return err
			}
		}
	}
	var mode int
	if err := conn.QueryRowContext(ctx, `PRAGMA auto_vacuum`).Scan(&mode); err != nil {
		return err
	}
	t.incrementalVacuum = mode == autoVacuumIncremental
	if !t.incrementalVacuum {
		log.Printf("[traffic] 索引库未启用增量回收（auto_vacuum=%d）：删除流量不会缩小 index.sqlite，需要执行一次存储压缩来转换", mode)
	}
	if _, err := conn.ExecContext(ctx, indexSchema); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, ftsSchema); err != nil {
		log.Printf("[traffic] 全文索引不可用，正文搜索将被禁用（元数据搜索不受影响）：%v", err)
		return nil
	}
	t.fts = true
	return nil
}

// hostOnly strips an optional :port, so passthrough keys match whether the host
// arrives as "example.com:443" (CONNECT) or "example.com" (request URL).
// 한국어 해설: CONNECT 주소의 포트를 분리하여 요청 URL의 호스트와 동일한 투명 통과 키를 만든다.
func hostOnly(hostport string) string {
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		return h
	}
	return hostport
}

// ProxyAddr returns the address workers should set as HTTP(S)_PROXY. A bare
// ":port" means "bind all interfaces" (legacy default), so map it to loopback.
// 한국어 해설: 에이전트가 HTTP_PROXY/HTTPS_PROXY에 넣을 URL을 반환한다. :포트 표기는 접속용 루프백 주소로 바꾼다.
func (t *Traffic) ProxyAddr() string {
	if strings.HasPrefix(t.addr, ":") {
		return "http://127.0.0.1" + t.addr
	}
	return "http://" + t.addr
}

// SetUpstreamProxy points every captured request at a global egress proxy
// (http/https/socks5, optional user:pass in the URL). An empty raw string clears
// it, restoring direct dialing. The change is atomic and takes effect on the next
// connection — no restart, no proxy rebuild. go-mitmproxy dials all three schemes
// itself, so socks5 works uniformly here regardless of the target tool.
// 한국어 해설: 출구 프록시 URL을 검증한 뒤 원자 포인터로 교체한다. 빈 문자열은 직접 연결로 복귀하며 잘못된 값은 기존 설정을 바꾸지 않는다.
func (t *Traffic) SetUpstreamProxy(raw string) error {
	if strings.TrimSpace(raw) == "" {
		t.upstream.Store(nil)
		return nil
	}
	u, err := ValidateProxyURL(raw)
	if err != nil {
		return err
	}
	t.upstream.Store(u)
	return nil
}

// ValidateProxyURL parses and checks a proxy URL (http/https/socks5, optional
// user:pass), returning the parsed URL. Exposed so callers can validate a global
// proxy before persisting it even when the traffic proxy itself is disabled.
// 한국어 해설: HTTP·HTTPS·SOCKS5 스킴과 호스트 존재를 확인한다. 프록시의 실제 도달성이나 인증 성공을 검사하는 함수는 아니다.
func ValidateProxyURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("解析代理地址 %q: %w", raw, err)
	}
	switch u.Scheme {
	case "http", "https", "socks5":
	case "":
		return nil, fmt.Errorf("代理 %q 缺少协议(用 http://、https:// 或 socks5://)", raw)
	default:
		return nil, fmt.Errorf("不支持的代理协议 %q(用 http、https 或 socks5)", u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("代理 %q 缺少主机地址", raw)
	}
	return u, nil
}

// CACertPath returns the PEM CA cert clients must trust to verify HTTPS through
// the MITM proxy (go-mitmproxy writes it here on first start).
// 한국어 해설: HTTPS를 가로채어 기록할 때 클라이언트가 신뢰해야 하는 로컬 CA 인증서 경로를 알려 준다.
func (t *Traffic) CACertPath() string {
	return filepath.Join(t.dir, "_ca", "mitmproxy-ca-cert.pem")
}

// Start runs the proxy (blocking); run in a goroutine.
// 한국어 해설: 설정된 MITM 프록시의 리스닝 루프에 진입한다. 블로킹 호출이므로 서버 조립부가 별도 goroutine에서 실행한다.
func (t *Traffic) Start() error { return t.proxy.Start() }

// Close waits for background tree reclamation to finish before closing the
// index, so shutdown never leaves a goroutine unlinking files out from under a
// removed data directory. Index-space reclamation is signalled to stop first:
// it holds a whole minutes-long budget, and finishing it is never worth delaying
// shutdown for — the next deletion resumes it.
// 한국어 해설: 공간 회수 루프에 종료를 알리고 이미 시작한 파일 정리를 기다린 다음 SQLite를 닫는다.
func (t *Traffic) Close() error {
	if t.closed != nil {
		t.closeOnce.Do(func() { close(t.closed) })
	}
	t.reaping.Wait()
	return t.db.Close()
}

// stopping reports whether Close has been called. A nil channel (the zero value)
// is never ready, so this reads as "not closing" without a separate guard.
// 한국어 해설: 닫힘 채널을 비차단으로 읽어 종료 요청 여부를 확인한다. 초기화되지 않은 nil 채널은 종료되지 않은 상태로 취급한다.
func (t *Traffic) stopping() bool {
	select {
	case <-t.closed:
		return true
	default:
		return false
	}
}
// 한국어 해설: 통합 처리와 테스트에서 사용할 SQLite 핸들을 반환한다. PostgreSQL의 작업·발견 저장소와 구분한다.
func (t *Traffic) DB() *sql.DB { return t.db }

// sink is the go-mitmproxy addon that records completed exchanges.
type sink struct {
	mproxy.BaseAddon
	t *Traffic
}

// 한국어 해설: 프록시가 완성한 요청과 응답이 모두 있을 때만 하나의 교환으로 기록한다.
func (s *sink) Response(f *mproxy.Flow) {
	if f.Request == nil || f.Response == nil {
		return
	}
	s.t.record(f)
}

// RequestError fires when a request through an established MITM tunnel fails. If
// the failure looks proxy/protocol-caused (h2 quirks, HEAD-with-body, protocol
// errors) — not a plain target-unreachable error — we flag the host for
// transparent passthrough so future requests to it succeed instead of dying.
// 한국어 해설: MITM 요청 처리 오류를 분류하여 필요하면 해당 호스트의 이후 연결을 투명 통과로 전환한다.
func (s *sink) RequestError(f *mproxy.Flow, err error) { s.t.maybePassthrough(f, err) }

// maybePassthrough marks a host to be tunneled transparently on the next
// connection, but only for errors the proxy itself caused — a target that is
// simply down/filtered would fail without us too, and must stay MITM'd+recorded.
// 한국어 해설: 프로토콜·프록시 원인으로 보이는 오류만 호스트별 pass 집합에 등록한다. 대상의 일반 연결 거절은 수집 해제 사유로 보지 않는다.
// 투명 통과는 요청의 도달성을 우선하는 정책이며, 이후 해당 호스트 본문은 이 수집기에 남지 않는다.
func (t *Traffic) maybePassthrough(f *mproxy.Flow, err error) {
	if err == nil || f == nil || f.Request == nil || f.Request.URL == nil || !proxyCausedErr(err) {
		return
	}
	host := f.Request.URL.Hostname()
	if host == "" {
		return
	}
	if _, loaded := t.pass.LoadOrStore(host, struct{}{}); !loaded {
		log.Printf("[traffic] 与 %s 的 MITM 出错，改为透传（该 host 后续直连目标、不再记录，但请求照常）：%v", host, err)
	}
}

// proxyCausedErr reports whether err indicates the interception layer (not the
// target) is at fault — HTTP/2 handling, HEAD-with-body, or a protocol violation.
// 한국어 해설: 오류 문자열에서 HTTP/2·HEAD·malformed 등의 단서를 찾는 휴리스틱이다. 네트워크 원인을 증명하는 정밀 판별기는 아니다.
func proxyCausedErr(err error) bool {
	s := strings.ToLower(err.Error())
	for _, p := range []string{"head request", "http2", "http/2", "protocol error", "protocol_error", "malformed"} {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

// record persists one exchange entirely in SQLite: metadata, bodies and the
// full-text index. Nothing is written to a per-request directory — only bodies
// above maxInlineBody spill to the content-addressed blob store.
// 한국어 해설: 하나의 교환을 메타데이터·본문·blob 참조·FTS 인덱스에 같은 SQLite 트랜잭션으로 기록한다.
// 파일 쓰기는 SQL 트랜잭션 전에 처리하고 wmu는 그 전체 구간을 보호하여 삭제/GC와의 경쟁을 막는다.
// ID는 초 단위 시각과 seq%10000을 조합하므로 임의 규모에서 충돌이 불가능한 UUID로 해석하면 안 된다.
func (t *Traffic) record(f *mproxy.Flow) {
	// The write lock covers blob + index writes, so DeleteHost (and its blob GC)
	// can run under the same lock without racing a concurrent record.
	t.wmu.Lock()
	defer t.wmu.Unlock()
	host := f.Request.URL.Hostname()
	method := f.Request.Method
	tmpl := db.TemplatePath(f.Request.URL.EscapedPath())
	n := t.seq.Add(1)
	now := time.Now()
	id := fmt.Sprintf("%d-%04d", now.Unix(), n%10000)

	ct := f.Response.Header.Get("Content-Type")
	url := f.Request.URL.String()
	reqHead := fmt.Sprintf("%s %s %s\n%s", method, f.Request.URL.RequestURI(), f.Request.Proto, requestHeaderLines(f.Request))
	respHead := fmt.Sprintf("HTTP %d\n%s", f.Response.StatusCode, headerLines(f.Response.Header))
	// Bodies are spilled before the transaction opens: blob writes are filesystem
	// work and must not sit inside the SQLite write lock.
	reqB := t.spill(f.Request.Body, f.Request.Header.Get("Content-Type"))
	respB := t.spill(f.Response.Body, ct)

	tx, err := t.db.Begin()
	if err != nil {
		log.Printf("[traffic] 记录 %s 失败（开启事务）：%v", url, err)
		return
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	// path stays empty for db-resident exchanges; a non-empty path marks a legacy
	// row whose bodies still live in the old on-disk tree (see Get).
	res, err := tx.Exec(`INSERT OR REPLACE INTO exchanges(id,ts,host,method,url_template,url,status,content_type,req_len,resp_len,path)
VALUES(?,?,?,?,?,?,?,?,?,?,'')`,
		id, now.Unix(), host, method, tmpl, url, f.Response.StatusCode, ct,
		len(f.Request.Body), len(f.Response.Body))
	if err != nil {
		log.Printf("[traffic] 记录 %s 失败（写索引）：%v", url, err)
		return
	}
	rowid, err := res.LastInsertId()
	if err != nil {
		log.Printf("[traffic] 记录 %s 失败（取 rowid）：%v", url, err)
		return
	}

	if _, err := tx.Exec(`INSERT OR REPLACE INTO exchange_bodies(id,req_head,req_body,req_blob,resp_head,resp_body,resp_blob)
VALUES(?,?,?,?,?,?,?)`,
		id, reqHead, reqB.inline, nullIfEmpty(reqB.hash), respHead, respB.inline, nullIfEmpty(respB.hash)); err != nil {
		log.Printf("[traffic] 记录 %s 失败（写正文）：%v", url, err)
		return
	}

	for _, h := range []string{reqB.hash, respB.hash} {
		if h == "" {
			continue
		}
		if _, err := tx.Exec(`INSERT OR IGNORE INTO blob_refs(hash,exchange_id) VALUES(?,?)`, h, id); err != nil {
			log.Printf("[traffic] 记录 %s 失败（登记 blob 引用）：%v", url, err)
			return
		}
	}

	if t.fts {
		// The index is fed from memory, so a body that spilled to _blobs is still
		// fully searchable even though only its preview is stored inline.
		idx := strings.Join([]string{url, reqHead, reqB.index, respHead, respB.index}, "\n")
		if _, err := tx.Exec(`INSERT INTO ex_fts(rowid,content) VALUES(?,?)`, rowid, idx); err != nil {
			log.Printf("[traffic] 记录 %s 失败（写全文索引）：%v", url, err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[traffic] 记录 %s 失败（提交）：%v", url, err)
	}
}

// storedBody is one body after the inline/spill decision: inline is what goes in
// the SQLite column (the whole body, or a preview when spilled), hash names the
// blob when it spilled, and index is the text handed to FTS (empty for binary).
// 한국어 자료형: inline은 실제 SQL 본문/미리보기, hash는 외부 파일 키, index는 검색용 텍스트다. 세 값의 크기와 보존 범위는 서로 다르다.
type storedBody struct {
	inline []byte
	hash   string
	index  string
}

// spill decides where one body lives. Small bodies stay inline. Large ones are
// written to the content-addressed store and keep a readable preview inline so a
// reader can identify them without fetching the blob. Either way, text bodies
// are handed to the full-text index in full (up to maxIndexBody) — indexing is
// independent of where the bytes end up.
// 한국어 해설: 256 KiB 이하 본문은 SQLite 안에, 큰 본문은 SHA-256 기반 파일에 둔다. 텍스트 미리보기는 8 KiB, 색인 기여분은 최대 4 MiB다.
// blob 쓰기 실패 시 현재 구현은 짧은 미리보기로 진행하므로 저장 길이와 실제 바이트의 일치 여부는 증거 복사 단계에서도 확인한다.
func (t *Traffic) spill(body []byte, contentType string) storedBody {
	if len(body) == 0 {
		return storedBody{}
	}
	text := !isBinaryBody(contentType, body)
	indexText := func() string {
		if !text {
			return ""
		}
		return string(clipBytes(body, maxIndexBody))
	}
	if len(body) <= maxInlineBody {
		return storedBody{inline: body, index: indexText()}
	}

	sum := sha256.Sum256(body)
	h := hex.EncodeToString(sum[:])
	// One bucket level (256 buckets) is enough to keep any single directory small;
	// the store only ever holds bodies above maxInlineBody, deduplicated by hash.
	blobDir := filepath.Join(t.dir, "_blobs", "sha256", h[:2])
	if err := os.MkdirAll(blobDir, 0o755); err != nil {
		log.Printf("[traffic] 创建 blob 目录失败：%v", err)
		return storedBody{inline: clipBytes(body, blobPreview), index: indexText()}
	}
	blobPath := filepath.Join(blobDir, h+".bin")
	if _, err := os.Stat(blobPath); os.IsNotExist(err) {
		if err := os.WriteFile(blobPath, body, 0o644); err != nil {
			log.Printf("[traffic] 写 blob %s 失败：%v", h, err)
			return storedBody{inline: clipBytes(body, blobPreview), index: indexText()}
		}
	}
	sb := storedBody{hash: h, index: indexText()}
	if text {
		sb.inline = []byte(truncateUTF8(body, blobPreview))
	} else {
		sb.inline = []byte(binaryTag(contentType, body))
	}
	return sb
}

// binaryTypes are content-type prefixes whose bodies are never worth indexing or
// previewing as text. Anything not listed is treated as text (with a NUL-byte
// check as backstop), so unusual-but-searchable types like application/sql or a
// bare text/* are not silently dropped from the index.
var binaryTypes = []string{
	"image/", "audio/", "video/", "font/",
	"application/octet-stream", "application/zip", "application/gzip",
	"application/x-gzip", "application/x-tar", "application/x-7z-compressed",
	"application/x-rar", "application/pdf", "application/x-msdownload",
	"application/vnd.android.package-archive", "application/java-archive",
	"application/wasm", "application/x-shockwave-flash",
}

// 한국어 해설: Content-Type 접두어와 앞 512바이트의 NUL 유무로 텍스트 색인 대상인지 추정한다.
func isBinaryBody(contentType string, body []byte) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	for _, p := range binaryTypes {
		if strings.HasPrefix(ct, p) {
			return true
		}
	}
	// Backstop for mislabeled or absent content types: real text does not carry
	// NUL bytes, so a NUL in the head of the body means binary regardless.
	return bytes.IndexByte(clipBytes(body, 512), 0) >= 0
}

// binaryTag describes a spilled binary body in one line, including the leading
// magic bytes so a reader can recognize the format without downloading it.
// 한국어 해설: 큰 바이너리 본문 대신 형식·전체 길이·앞 4바이트의 magic 값을 한 줄로 보여 준다.
func binaryTag(contentType string, body []byte) string {
	ct := strings.TrimSpace(contentType)
	if ct == "" {
		ct = "application/octet-stream"
	}
	magic := hex.EncodeToString(clipBytes(body, 4))
	return fmt.Sprintf("[binary %s, %d bytes, magic=%s]", ct, len(body), magic)
}

// 한국어 해설: 바이트 슬라이스의 앞부분만 반환하는 저수준 절단 함수다. 문자 경계 보존이 필요한 곳에서는 truncateUTF8을 사용한다.
func clipBytes(b []byte, n int) []byte {
	if len(b) <= n {
		return b
	}
	return b[:n]
}

// truncateUTF8 cuts b to at most n bytes without splitting a multi-byte rune,
// so a preview never ends in half a CJK character.
// 한국어 해설: 미리보기 끝이 한글·중국어·이모지의 중간 바이트에 걸리지 않도록 UTF-8 경계까지 뒤로 이동한다.
func truncateUTF8(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	b = b[:n]
	// A rune is at most 4 bytes, so backing off 3 bytes always finds the boundary
	// (unless the input was already invalid UTF-8, in which case we keep the cut).
	for i := 0; i < utf8.UTFMax-1 && len(b) > 0; i++ {
		if r, size := utf8.DecodeLastRune(b); r != utf8.RuneError || size != 1 {
			break
		}
		b = b[:len(b)-1]
	}
	return string(b)
}

// 한국어 해설: 빈 blob 해시를 SQL NULL로 표현하여 본문이 인라인 저장되었다는 상태를 구분한다.
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// 한국어 해설: 복수 값 헤더를 각각 한 줄씩 직렬화한다. map 순회 순서 자체는 정렬하지 않는다.
func headerLines(h map[string][]string) string {
	var b strings.Builder
	for k, vs := range h {
		for _, v := range vs {
			b.WriteString(k)
			b.WriteString(": ")
			b.WriteString(v)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// requestHeaderLines restores the HTTP Host header, which net/http stores on
// Request.Host rather than in Header. Keeping it in request.http makes the raw
// capture complete and directly replayable.
// 한국어 해설: Go HTTP 구조체에서 일반 Header와 분리된 Host를 다시 넣어 재현 가능한 요청 헤더를 구성한다.
func requestHeaderLines(req *mproxy.Request) string {
	headers := req.Header.Clone()
	headers.Del("Host")
	host := req.URL.Host
	if raw := req.Raw(); raw != nil && strings.TrimSpace(raw.Host) != "" {
		host = raw.Host
	}
	var b strings.Builder
	if host = strings.TrimSpace(host); host != "" {
		b.WriteString("Host: ")
		b.WriteString(host)
		b.WriteByte('\n')
	}
	b.WriteString(headerLines(headers))
	return b.String()
}

// 한국어 해설: 이전 저장 방식의 디렉터리 이름에 맞게 경로 특수문자를 치환하고 길이를 제한한다. 자산 URL 정규화 함수와는 다르다.
func sanitize(s string) string {
	r := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "?", "_", "*", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	out := r.Replace(s)
	if out == "" || out == "_" {
		return "root"
	}
	if len(out) > 120 {
		out = out[:120]
	}
	return out
}

var blobHashRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

// blobPath locates a stored blob. Current writes use one bucket level; blobs
// written before that change used two, so both layouts are probed rather than
// migrated.
// 한국어 해설: 64자리 16진수 해시만 받아 현재 1단계 버킷과 과거 2단계 버킷을 차례로 찾는다. 임의 경로를 인자로 받지 않는다.
func (t *Traffic) blobPath(hash string) (string, error) {
	hash = strings.ToLower(strings.TrimSpace(hash))
	// Validated as pure hex before touching the filesystem, so a crafted hash can
	// never traverse out of the blob directory.
	if !blobHashRe.MatchString(hash) {
		return "", fmt.Errorf("非法的 blob hash")
	}
	for _, p := range []string{
		filepath.Join(t.dir, "_blobs", "sha256", hash[:2], hash+".bin"),
		filepath.Join(t.dir, "_blobs", "sha256", hash[:2], hash[2:4], hash+".bin"),
	} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	return "", fmt.Errorf("blob %s 不存在", hash)
}

// Blob opens a spilled body for streaming; the caller must close the file.
// Streaming matters here: the driver exposes no incremental BLOB API, so keeping
// large bodies on disk is what lets them be served without loading them whole.
// 한국어 해설: 대형 본문을 스트리밍할 파일과 전체 크기를 반환한다. 반환된 파일을 닫는 책임은 호출자에게 있다.
func (t *Traffic) Blob(hash string) (*os.File, int64, error) {
	p, err := t.blobPath(hash)
	if err != nil {
		return nil, 0, err
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, 0, err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, 0, err
	}
	return f, st.Size(), nil
}

// BlobRange reads at most length bytes of a blob from offset, and reports the
// blob's total size. Used by the agent tool to page through a large body without
// pulling all of it into the model's context.
// 한국어 해설: 파일 오프셋부터 지정 바이트를 읽고 전체 길이도 반환한다. 에이전트용 8 KiB 한도는 이 함수가 아니라 Tools의 래퍼에서 적용한다.
func (t *Traffic) BlobRange(hash string, offset, length int64) (data []byte, total int64, err error) {
	f, size, err := t.Blob(hash)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	if offset < 0 {
		offset = 0
	}
	if offset >= size {
		return nil, size, nil
	}
	if length <= 0 || offset+length > size {
		length = size - offset
	}
	buf := make([]byte, length)
	n, err := f.ReadAt(buf, offset)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, size, err
	}
	return buf[:n], size, nil
}

// ExchangeMeta is one row of the index (returned by Search).
// 한국어 자료형: 목록에 필요한 교환 식별자·시각·호스트·URL·상태 등을 담는다. 본문은 이 DTO에 넣지 않는다.
type ExchangeMeta struct {
	ID          string `json:"id"`
	TS          int64  `json:"ts"`
	Host        string `json:"host"`
	Method      string `json:"method"`
	URLTemplate string `json:"url_template"`
	URL         string `json:"url"`
	Status      int    `json:"status"`
	ContentType string `json:"content_type"`
	RespLen     int    `json:"resp_len"`
	Path        string `json:"path"`
}

// Search returns paged exchange metadata (never bodies).
// 한국어 해설: 본문 없이 메타데이터만 페이지로 반환하는 기본 조회다. 도구용 query와 페이지 크기 및 필터 계약이 다르다.
func (t *Traffic) Search(host string, page, size int) ([]ExchangeMeta, error) {
	if size <= 0 || size > 500 {
		size = 100
	}
	q := `SELECT id,ts,host,method,url_template,url,status,content_type,resp_len,path FROM exchanges`
	args := []any{}
	if host != "" {
		q += ` WHERE host=?`
		args = append(args, host)
	}
	q += ` ORDER BY ts DESC LIMIT ? OFFSET ?`
	args = append(args, size, page*size)
	rows, err := t.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ExchangeMeta
	for rows.Next() {
		var m ExchangeMeta
		if err := rows.Scan(&m.ID, &m.TS, &m.Host, &m.Method, &m.URLTemplate, &m.URL, &m.Status, &m.ContentType, &m.RespLen, &m.Path); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ftsFilter builds the SQL condition restricting rows to those whose indexed
// text matches term. ok is false when full-text search cannot serve the term —
// no FTS index on this build, or fewer than three characters, which the trigram
// tokenizer cannot match — leaving callers to fall back to metadata matching.
// 한국어 해설: FTS가 준비되어 있고 검색어가 3문자 이상일 때에만 MATCH 조건과 바인딩 값을 만든다.
func (t *Traffic) ftsFilter(term string) (cond string, arg any, ok bool) {
	term = strings.TrimSpace(term)
	if !t.fts || utf8.RuneCountInString(term) < minTrigram {
		return "", nil, false
	}
	return `rowid IN (SELECT rowid FROM ex_fts WHERE ex_fts MATCH ?)`, ftsQuote(term), true
}

// ftsQuote wraps a term as an FTS5 string literal so that punctuation and query
// operators inside it (quotes, AND/OR/NEAR, *, ^) are matched literally instead
// of being parsed as query syntax.
// 한국어 해설: 입력의 따옴표와 연산자를 FTS 문자열 리터럴로 처리하여 검색어가 질의 문법으로 해석되지 않게 한다.
func ftsQuote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// PageQuery bundles the optional filters and sort order for Page. Every filter
// is combined with AND; an empty string skips that filter, and RespMin/RespMax
// of -1 skip the response-size bounds. An empty Sort/Order defaults to
// newest-first by timestamp.
// 한국어 자료형: 웹 목록의 선택 필터다. 응답 크기의 -1은 제한 없음이며 0은 실제 0바이트 조건이 될 수 있다.
type PageQuery struct {
	Host    string // host substring
	Method  string // exact method (case-insensitive)
	Query   string // broad search box: metadata columns OR captured text
	Body    string // content box: captured request/response text only (full-text)
	Path    string // url_template substring — the vulnerability's path
	Status  string // exact code ("404") or class bucket ("4xx")
	RespMin int64  // minimum resp_len, or -1 when unset
	RespMax int64  // maximum resp_len, or -1 when unset
	Sort    string // ts | status | resp_len (default ts)
	Order   string // asc | desc (default desc)
}

// sortColumns whitelists the columns Page may order by, so the caller-supplied
// Sort can never reach the SQL as anything but one of these fixed names.
var sortColumns = map[string]string{"ts": "ts", "status": "status", "resp_len": "resp_len"}

// statusFilter turns a status token into a SQL condition: an exact code ("404")
// matches that status, an "Nxx" class ("4xx") matches the whole hundreds band.
// ok is false for an empty or unrecognized token.
// 한국어 해설: 404 같은 정확한 코드 또는 4xx 같은 백 단위 구간을 SQL 조건으로 변환한다.
func statusFilter(s string) (cond string, args []any, ok bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "", nil, false
	}
	if len(s) == 3 && s[0] >= '1' && s[0] <= '5' && s[1] == 'x' && s[2] == 'x' {
		base := int(s[0]-'0') * 100
		return "status>=? AND status<?", []any{base, base + 100}, true
	}
	if n, err := strconv.Atoi(s); err == nil {
		return "status=?", []any{n}, true
	}
	return "", nil, false
}

// Page returns one page of exchange metadata for the traffic list, filtered by
// the criteria in f (all optional, combined with AND) and ordered by the
// requested column. Query is the broad search box (metadata columns OR, when the
// term is long enough for the trigram index, captured request/response text);
// Body narrows to exchanges whose captured text matches, via that same index.
// Also returns the total number of rows matching the filter (for the UI's
// pagination). Bodies are never included in the rows.
// 한국어 해설: 웹 트래픽 목록의 AND 필터·개수·정렬·페이지 조회를 담당한다. 정렬 열은 고정 허용 목록에서만 선택한다.
// 일반 Query는 메타데이터와 FTS를 합치며, Body가 너무 짧으면 현재 구현에서는 해당 본문 필터를 추가하지 않는다.
func (t *Traffic) Page(f PageQuery, page, size int) (rows []ExchangeMeta, total int, err error) {
	if size <= 0 || size > 500 {
		size = 100
	}
	if page < 0 {
		page = 0
	}
	where := ""
	var args []any
	add := func(cond string, vs ...any) {
		if where == "" {
			where = " WHERE "
		} else {
			where += " AND "
		}
		where += cond
		args = append(args, vs...)
	}
	if h := strings.TrimSpace(f.Host); h != "" {
		add("host LIKE ?", "%"+h+"%")
	}
	if m := strings.TrimSpace(f.Method); m != "" {
		add("method=?", strings.ToUpper(m))
	}
	if p := strings.TrimSpace(f.Path); p != "" {
		add("url_template LIKE ?", "%"+p+"%")
	}
	if cond, sargs, ok := statusFilter(f.Status); ok {
		add(cond, sargs...)
	}
	if f.RespMin >= 0 {
		add("resp_len>=?", f.RespMin)
	}
	if f.RespMax >= 0 {
		add("resp_len<=?", f.RespMax)
	}
	if s := strings.TrimSpace(f.Query); s != "" {
		like := "%" + s + "%"
		const meta = "host LIKE ? OR url LIKE ? OR url_template LIKE ? OR method LIKE ? OR content_type LIKE ? OR CAST(status AS TEXT) LIKE ?"
		// Metadata match OR full-text match: one search box, widest recall. Terms
		// too short for trigram silently fall back to metadata only.
		if cond, arg, ok := t.ftsFilter(s); ok {
			add("(("+meta+") OR "+cond+")", like, like, like, like, like, like, arg)
		} else {
			add("("+meta+")", like, like, like, like, like, like)
		}
	}
	// Body is the dedicated content box: it only searches captured request/response
	// text, so it goes straight to the full-text index with no metadata fallback. A
	// term too short for the trigram tokenizer cannot be served and is skipped
	// rather than guessed at — the UI hints at the three-character minimum.
	if b := strings.TrimSpace(f.Body); b != "" {
		if cond, arg, ok := t.ftsFilter(b); ok {
			add(cond, arg)
		}
	}
	if err = t.db.QueryRow(`SELECT COUNT(*) FROM exchanges`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	col := sortColumns[strings.ToLower(strings.TrimSpace(f.Sort))]
	if col == "" {
		col = "ts"
	}
	dir := "DESC"
	if strings.EqualFold(strings.TrimSpace(f.Order), "asc") {
		dir = "ASC"
	}
	// id embeds capture timestamp + sequence, so a trailing id DESC makes the order
	// total and pagination stable even when the sort column has many ties.
	sel := `SELECT id,ts,host,method,url_template,url,status,content_type,resp_len,path FROM exchanges` +
		where + ` ORDER BY ` + col + ` ` + dir + `, id DESC LIMIT ? OFFSET ?`
	qargs := append(append([]any{}, args...), size, page*size)
	rs, err := t.db.Query(sel, qargs...)
	if err != nil {
		return nil, 0, err
	}
	defer rs.Close()
	for rs.Next() {
		var m ExchangeMeta
		if err := rs.Scan(&m.ID, &m.TS, &m.Host, &m.Method, &m.URLTemplate, &m.URL, &m.Status, &m.ContentType, &m.RespLen, &m.Path); err != nil {
			return nil, 0, err
		}
		rows = append(rows, m)
	}
	return rows, total, rs.Err()
}

// Get returns the full request/response text of one exchange. Bodies come from
// the database; rows recorded before bodies moved into SQLite carry a non-empty
// path and are read from the legacy on-disk tree instead, so history stays
// readable without being migrated.
// 한국어 해설: SQLite의 헤더·인라인 본문을 우선 읽고, 과거 path 행만 request.http/response.http로 되돌아간다.
// 대형 본문의 반환 문자열에는 전체 파일 대신 미리보기와 @blob 포인터가 들어간다.
func (t *Traffic) Get(id string) (req, resp string, err error) {
	var reqHead, respHead string
	var reqBody, respBody []byte
	var reqBlob, respBlob sql.NullString
	var reqLen, respLen int
	err = t.db.QueryRow(`SELECT b.req_head,b.req_body,b.req_blob,b.resp_head,b.resp_body,b.resp_blob,e.req_len,e.resp_len
FROM exchange_bodies b JOIN exchanges e ON e.id=b.id WHERE b.id=?`, id).
		Scan(&reqHead, &reqBody, &reqBlob, &respHead, &respBody, &respBlob, &reqLen, &respLen)
	if err == nil {
		return assembleRaw(reqHead, reqBody, reqBlob, reqLen), assembleRaw(respHead, respBody, respBlob, respLen), nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", "", err
	}

	var rel string
	if err = t.db.QueryRow(`SELECT path FROM exchanges WHERE id=?`, id).Scan(&rel); err != nil {
		return "", "", err
	}
	if strings.TrimSpace(rel) == "" {
		return "", "", fmt.Errorf("exchange %s 无正文记录", id)
	}
	rb, _ := os.ReadFile(filepath.Join(t.dir, rel, "request.http"))
	pb, _ := os.ReadFile(filepath.Join(t.dir, rel, "response.http"))
	return string(rb), string(pb), nil
}

// assembleRaw rebuilds one side's raw HTTP text: head, blank line, body. A body
// that spilled to the blob store shows its inline preview followed by the blob
// pointer, so the reader can see what it is and fetch the rest by hash.
// 한국어 해설: 헤더·빈 줄·본문을 결합하고 외부 blob이 있으면 전체 길이와 해시 포인터를 추가한다.
func assembleRaw(head string, body []byte, blob sql.NullString, total int) string {
	var b strings.Builder
	b.WriteString(head)
	b.WriteByte('\n')
	b.Write(body)
	if blob.Valid && blob.String != "" {
		fmt.Fprintf(&b, "\n…[truncated] @blob sha256:%s (len=%d)", blob.String, total)
	}
	return b.String()
}

// HostCount is one distinct recorded host plus its exchange count.
type HostCount struct {
	Host  string `json:"host"`
	Count int    `json:"count"`
}

// Hosts returns distinct recorded hosts with exchange counts, most recent
// activity first — powers the page's target picker.
// 한국어 해설: 호스트별 교환 수와 최근 시각을 집계하여 웹 화면의 대상 선택기를 채운다.
func (t *Traffic) Hosts() ([]HostCount, error) {
	rows, err := t.db.Query(`SELECT host, COUNT(*) AS n, MAX(ts) AS last FROM exchanges GROUP BY host ORDER BY last DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []HostCount
	for rows.Next() {
		var h HostCount
		var last int64
		if err := rows.Scan(&h.Host, &h.Count, &last); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// Count returns total recorded exchanges.
// 한국어 해설: 현재 exchanges 행의 총수를 센다. 증거 저장소의 보존본 수와는 별개다.
func (t *Traffic) Count() (int, error) {
	var n int
	err := t.db.QueryRow(`SELECT COUNT(*) FROM exchanges`).Scan(&n)
	return n, err
}

// DeleteHost removes every recorded exchange whose host contains the given
// substring — mirroring the UI's host filter, so what you filtered is what gets
// deleted. Everything lives in SQLite, so the deletion is one transaction across
// the index, bodies, full-text index and blob references. Content-addressed
// blobs are shared across hosts and so are garbage-collected afterwards rather
// than deleted by host. Returns rows deleted.
// 한국어 해설: 호스트 부분 문자열에 맞는 행들을 같은 SQL 트랜잭션에서 제거한다. 원본 파일 트리는 먼저 이동하여 커밋 전 되돌릴 수 있게 한다.
// blob은 여러 호스트가 공유할 수 있으므로 호스트 이름만으로 지우지 않고 마지막 참조 여부를 확인한다.
func (t *Traffic) DeleteHost(host string) (int64, error) {
	like := "%" + host + "%"
	t.wmu.Lock()
	defer t.wmu.Unlock()
	// Resolved before the rows go away: with a substring match, the index is the
	// only record of which host directories the filter actually hit.
	legacy, err := t.hostTrees(`host LIKE ?`, like)
	if err != nil {
		return 0, err
	}
	tx, err := t.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed
	n, err := t.deleteWhere(tx, `host LIKE ?`, like)
	if err != nil {
		return 0, err
	}
	// Staged before the commit so a filesystem failure can still abort the whole
	// deletion, and before gcBlobs so reference collection sees the correct live
	// set — a staged tree is invisible to it.
	stageDir, moves, err := t.stageTrees(legacy)
	if err != nil {
		return 0, errors.Join(err, restoreTrees(stageDir, moves))
	}
	if err := tx.Commit(); err != nil {
		return 0, errors.Join(fmt.Errorf("提交流量索引删除: %w", err), restoreTrees(stageDir, moves))
	}
	t.reapStage(stageDir)
	if n > 0 {
		t.reclaim()
		if err := t.gcBlobs(); err != nil {
			return n, err
		}
	}
	return n, nil
}

// DeleteAll removes every recorded exchange — the page's clear-everything
// action. It differs from the host-scoped deletions in two ways. It sweeps host
// directories without consulting the index, because "clear everything" should
// leave nothing behind and a directory whose rows are already gone would
// otherwise survive. And it ends with a full compaction: VACUUM costs what it
// keeps, so an emptied index is the one moment it is free, and it is also the
// only way to switch on auto_vacuum for a database created without it.
//
// Evidence already bound to a finding is untouched. Those bodies were copied
// into the evidence store when they were bound, precisely so that disposable
// traffic could be cleared without taking proof with it.
//
// Returns the number of exchanges deleted and how many bytes of index the
// compaction handed back.
// 한국어 해설: 모든 원시 트래픽과 고아인 과거 호스트 디렉터리를 지우고 빈 SQLite에 전체 VACUUM을 수행한다.
// 이미 finding에 복사·바인딩한 evidence 저장소는 별도로 남는다.
func (t *Traffic) DeleteAll() (deleted int64, reclaimed int64, err error) {
	t.wmu.Lock()
	defer t.wmu.Unlock()
	before := t.indexBytes()
	trees, err := t.allHostTrees()
	if err != nil {
		return 0, 0, err
	}
	tx, err := t.db.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed
	if deleted, err = t.deleteWhere(tx, `1=1`); err != nil {
		return 0, 0, err
	}
	// Staged before the commit so a filesystem failure can still abort the whole
	// deletion, exactly as in DeleteHost.
	stageDir, moves, err := t.stageTrees(trees)
	if err != nil {
		return 0, 0, errors.Join(err, restoreTrees(stageDir, moves))
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, errors.Join(fmt.Errorf("提交流量索引删除: %w", err), restoreTrees(stageDir, moves))
	}
	t.reapStage(stageDir)
	if err := t.gcBlobs(); err != nil {
		return deleted, 0, err
	}
	if err := t.compactIndex(); err != nil {
		// The deletion is already durable; compaction is disk space, not
		// correctness, so it must not turn a completed purge into a failed one.
		log.Printf("[traffic] 压实索引失败：%v", err)
		return deleted, 0, nil
	}
	return deleted, before - t.indexBytes(), nil
}

// allHostTrees lists every host directory left by the pre-SQLite layout. Unlike
// hostTrees it does not go through the index, so it also finds directories whose
// rows are already gone. Underscore-prefixed entries are the store's own
// (_index, _blobs, _ca, _delete_staging) and are never hosts.
// 한국어 해설: 인덱스에 없는 과거 호스트 폴더까지 찾는다. _index·_blobs·_ca처럼 밑줄로 시작하는 내부 폴더는 제외한다.
func (t *Traffic) allHostTrees() ([]string, error) {
	entries, err := os.ReadDir(t.dir)
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), "_") {
			dirs = append(dirs, filepath.Join(t.dir, e.Name()))
		}
	}
	return dirs, nil
}

// compactIndex rewrites the index into a fresh file, which is what actually
// returns its pages to the filesystem. VACUUM's runtime and temporary space
// scale with the content it keeps, so this is only reached right after DeleteAll
// has emptied the index — never as a routine step. It doubles as the conversion
// path for a database created before auto_vacuum=incremental: that pragma only
// takes effect through the VACUUM that follows it, and both must run on the same
// connection. Callers must hold wmu.
// 한국어 해설: 삭제로 비워진 DB를 다시 써서 실제 파일 공간을 반환하고 옛 DB의 auto_vacuum 방식도 전환한다. 호출자는 wmu를 보유해야 한다.
func (t *Traffic) compactIndex() error {
	ctx := context.Background()
	conn, err := t.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if t.fts {
		// A full merge, not the bounded one reclaim uses: with the index emptied
		// there is nothing left to merge, so this only discards the tombstones.
		if _, err := conn.ExecContext(ctx, `INSERT INTO ex_fts(ex_fts) VALUES('optimize')`); err != nil {
			return fmt.Errorf("合并全文索引: %w", err)
		}
	}
	if _, err := conn.ExecContext(ctx, `PRAGMA auto_vacuum=incremental`); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, `VACUUM`); err != nil {
		return fmt.Errorf("压实索引: %w", err)
	}
	var mode int
	if err := conn.QueryRowContext(ctx, `PRAGMA auto_vacuum`).Scan(&mode); err != nil {
		return err
	}
	// A legacy database has just been converted, so later deletions can reclaim
	// space on their own instead of waiting for another purge.
	t.incrementalVacuum = mode == autoVacuumIncremental
	if _, err := conn.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return fmt.Errorf("截断 WAL: %w", err)
	}
	return nil
}

// deleteWhere removes every trace of the exchanges matching the condition:
// full-text index rows, bodies, blob references, and finally the index rows.
// Order matters — each sub-select reads exchanges, so that table is emptied last.
// 한국어 해설: FTS → 본문 → blob 참조 → 메타데이터 순서로 삭제한다. 앞선 하위 질의들이 exchanges를 사용하므로 마지막에 지운다.
func (t *Traffic) deleteWhere(tx *sql.Tx, where string, args ...any) (int64, error) {
	if t.fts {
		// ex_fts is contentless and addressed by rowid, hence the rowid sub-select.
		if _, err := tx.Exec(`DELETE FROM ex_fts WHERE rowid IN (SELECT rowid FROM exchanges WHERE `+where+`)`, args...); err != nil {
			return 0, fmt.Errorf("删除全文索引: %w", err)
		}
	}
	if _, err := tx.Exec(`DELETE FROM exchange_bodies WHERE id IN (SELECT id FROM exchanges WHERE `+where+`)`, args...); err != nil {
		return 0, fmt.Errorf("删除正文: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM blob_refs WHERE exchange_id IN (SELECT id FROM exchanges WHERE `+where+`)`, args...); err != nil {
		return 0, fmt.Errorf("删除 blob 引用: %w", err)
	}
	res, err := tx.Exec(`DELETE FROM exchanges WHERE `+where, args...)
	if err != nil {
		return 0, fmt.Errorf("删除索引行: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// hostTrees returns the on-disk directory of every host matching the condition.
// Exchanges recorded since bodies moved into SQLite have no directory at all, so
// most of these paths simply will not exist — staging skips those. The lookup is
// deliberately not restricted to rows with a path: a host whose index rows are
// already gone can still have an orphaned directory, and deleting the host should
// take that with it.
// 한국어 해설: 삭제 조건에 맞는 호스트를 SQL에서 찾은 뒤 이전 버전의 폴더 이름으로 변환한다.
func (t *Traffic) hostTrees(where string, args ...any) ([]string, error) {
	rows, err := t.db.Query(`SELECT DISTINCT host FROM exchanges WHERE `+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var dirs []string
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, err
		}
		dirs = append(dirs, filepath.Join(t.dir, sanitize(h)))
	}
	return dirs, rows.Err()
}

type stagedTrafficPath struct {
	source string
	staged string
}

type hostDeleteStageJournal struct {
	Version   int                   `json:"version"`
	ArchiveID int64                 `json:"archive_id,omitempty"`
	TaskID    int64                 `json:"task_id,omitempty"`
	Hosts     []string              `json:"hosts,omitempty"`
	Moves     []hostDeleteStageMove `json:"moves"`
}

type hostDeleteStageMove struct {
	Source string `json:"source"`
	Staged string `json:"staged"`
}

const hostDeleteStageJournalName = "journal.json"

// stageTrees moves host directories aside onto the same filesystem. The rename is
// atomic and instant, which buys two things the unlink cannot: the deletion stays
// reversible until the transaction commits, and the tree stops being visible to
// blob reference collection right away (legacyBlobRefs skips underscore-prefixed
// directories, so anything under _delete_staging is already out of the live set).
// An empty stageDir is returned when there was nothing to stage.
// 한국어 해설: 일반 호스트 폴더 삭제를 같은 파일시스템의 임시 이동 단계로 위임한다. 아직 영구 삭제를 완료하지 않는다.
func (t *Traffic) stageTrees(dirs []string) (stageDir string, moves []stagedTrafficPath, err error) {
	return t.stageTreesForArchive(dirs, nil, 0, 0)
}

// 한국어 해설: 삭제 계획을 journal.json에 먼저 기록하고 기존 폴더를 임시 위치로 rename한다.
// 폴더가 없는 신규 SQLite 형식도 호스트 목록을 남겨 PostgreSQL 커밋 뒤 프로세스가 죽었을 때 미완료 삭제를 복구한다.
func (t *Traffic) stageTreesForArchive(dirs, hosts []string, archiveID, taskID int64) (stageDir string, moves []stagedTrafficPath, err error) {
	seen := make(map[string]struct{}, len(dirs))
	planned := make([]stagedTrafficPath, 0, len(dirs))
	for _, source := range dirs {
		if _, dup := seen[source]; dup {
			continue
		}
		seen[source] = struct{}{}
		if _, err := os.Lstat(source); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return stageDir, moves, fmt.Errorf("检查历史流量目录 %s: %w", source, err)
		}
		planned = append(planned, stagedTrafficPath{source: source})
	}
	// 归档路径即使一个历史 host 目录都没有(新装机的流量只落在 SQLite + _blobs)
	// 也必须留下 journal：崩溃点若落在 PostgreSQL 提交与 SQLite 提交之间，重启后
	// SQLite 事务被回滚，只有这份 journal 能让恢复流程补做 host 行的删除。少了它，
	// 已转冷任务的独占流量会永久留在热库里。
	uniqueHosts := uniqueArchiveHosts(hosts)
	if len(planned) == 0 && len(uniqueHosts) == 0 {
		return "", nil, nil
	}
	parent := filepath.Join(t.dir, "_delete_staging")
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return "", nil, fmt.Errorf("创建流量暂存目录: %w", err)
	}
	if stageDir, err = os.MkdirTemp(parent, "hosts-"); err != nil {
		return "", nil, fmt.Errorf("创建流量暂存目录: %w", err)
	}
	journal := hostDeleteStageJournal{Version: 1, ArchiveID: archiveID, TaskID: taskID, Hosts: uniqueHosts}
	for i := range planned {
		planned[i].staged = filepath.Join(stageDir, fmt.Sprintf("%d-%s", i, filepath.Base(planned[i].source)))
		journal.Moves = append(journal.Moves, hostDeleteStageMove{Source: planned[i].source, Staged: planned[i].staged})
	}
	if err := writeHostDeleteStageJournal(filepath.Join(stageDir, hostDeleteStageJournalName), journal); err != nil {
		_ = os.RemoveAll(stageDir)
		return "", nil, err
	}
	for _, move := range planned {
		source, staged := move.source, move.staged
		if err := os.Rename(source, staged); err != nil {
			return stageDir, moves, fmt.Errorf("移出历史流量目录 %s: %w", source, err)
		}
		moves = append(moves, stagedTrafficPath{source: source, staged: staged})
	}
	return stageDir, moves, nil
}

// 한국어 해설: 재시작 복구에 필요한 삭제 계획을 새 파일로 쓰고 Sync한다. 기존 journal을 조용히 덮어쓰지 않는다.
func writeHostDeleteStageJournal(path string, journal hostDeleteStageJournal) error {
	raw, err := json.Marshal(journal)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(raw); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

// restoreTrees puts staged directories back where they came from, newest move
// first, and drops the staging directory once everything is home.
// 한국어 해설: 이동 순서를 역순으로 되돌린다. 목적지 충돌이 생기면 오류와 임시 파일을 남겨 복구 실패를 숨기지 않는다.
func restoreTrees(stageDir string, moves []stagedTrafficPath) error {
	var errs []error
	for i := len(moves) - 1; i >= 0; i-- {
		move := moves[i]
		if _, err := os.Lstat(move.source); err == nil {
			errs = append(errs, fmt.Errorf("restore destination already exists: %s", move.source))
			continue
		} else if !os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("inspect restore destination %s: %w", move.source, err))
			continue
		}
		if err := os.Rename(move.staged, move.source); err != nil {
			errs = append(errs, fmt.Errorf("restore %s: %w", move.source, err))
		}
	}
	if len(errs) == 0 && stageDir != "" {
		if err := os.RemoveAll(stageDir); err != nil {
			errs = append(errs, fmt.Errorf("remove traffic stage: %w", err))
		}
	}
	return errors.Join(errs...)
}

// reapStage unlinks a committed staging directory in the background. This is the
// step that used to stall the recorder: a legacy tree mirrors the URL path per
// request and can hold hundreds of thousands of small files, and the whole unlink
// ran while the write lock was held. The rows are already gone by now, so losing
// this to a shutdown leaves garbage under _delete_staging, never inconsistent
// state.
// 한국어 해설: SQL 커밋 후 필요 없어진 임시 폴더를 백그라운드에서 제거하여 대규모 파일 삭제가 수집기를 오래 막지 않게 한다.
func (t *Traffic) reapStage(stageDir string) {
	if stageDir == "" {
		return
	}
	t.reaping.Go(func() {
		if err := os.RemoveAll(stageDir); err != nil {
			log.Printf("[traffic] 清理历史流量目录 %s 失败：%v", stageDir, err)
		}
	})
}

// DeleteHostsExact removes recorded exchanges for a set of EXACT hosts (the
// batch path for the page's multi-select delete): index rows + each host's file
// tree, then one blob-GC pass. Exact match — unlike DeleteHost's substring —
// so picking "api.example.com" never sweeps "api.example.com.cn". Returns rows
// deleted. Duplicate hosts are harmless (idempotent deletes, single tree pass).
// 한국어 해설: 선택한 호스트의 정확한 이름만 일괄 삭제한다. 부분 문자열 삭제인 DeleteHost와 달리 비슷한 도메인을 함께 지우지 않는다.
func (t *Traffic) DeleteHostsExact(hosts []string) (int64, error) {
	stage, err := t.StageDeleteHostsExact(hosts)
	if err != nil {
		return 0, err
	}
	deleted := stage.Deleted()
	if err := stage.Commit(); err != nil {
		return deleted, err
	}
	return deleted, nil
}

// HostDeleteStage keeps the SQLite delete transaction open so a task deletion
// can be coordinated with PostgreSQL: the traffic side is staged here and only
// committed once the caller's own transaction has succeeded. The Traffic write
// lock is held until Commit or Rollback, so no recorder can add rows or a blob
// reference for a host that is mid-deletion. Rolling back is just a transaction
// rollback now that nothing is moved on disk.
// 한국어 자료형: SQL 삭제와 과거 폴더 이동을 보류한 상태다. 생명주기 동안 Traffic 쓰기 잠금을 잡으므로 종료 책임이 중요하다.
type HostDeleteStage struct {
	traffic  *Traffic
	tx       *sql.Tx
	stageDir string
	moves    []stagedTrafficPath
	deleted  int64
	done     bool
}

// Deleted reports the number of exchange rows selected by this stage.
// 한국어 해설: 준비된 삭제 단계가 선택한 행 수를 반환한다. Commit 성공 여부 자체를 뜻하지는 않는다.
func (s *HostDeleteStage) Deleted() int64 {
	if s == nil {
		return 0
	}
	return s.deleted
}

// StageDeleteHostsExact prepares a reversible exact-host deletion. Callers must
// finish every successful stage with Commit or Rollback.
// 한국어 해설: 정확한 호스트 목록의 삭제를 준비한다. 성공 시 반드시 Commit 또는 Rollback으로 종료하여 쓰기 잠금을 해제해야 한다.
func (t *Traffic) StageDeleteHostsExact(hosts []string) (*HostDeleteStage, error) {
	return t.stageDeleteHostsExact(hosts, 0, 0)
}

// StageDeleteHostsExactForArchive ties the reversible traffic deletion to a
// persistent task archive. Startup recovery uses archiveCommitted to decide
// whether an interrupted stage must be completed or rolled back.
// 한국어 해설: 영속 archive/task ID를 삭제 일지에 연결하여 재시작 시 PostgreSQL 상태와 맞추어 처리할 수 있게 한다.
func (t *Traffic) StageDeleteHostsExactForArchive(hosts []string, archiveID, taskID int64) (*HostDeleteStage, error) {
	if archiveID <= 0 || taskID <= 0 {
		return nil, errors.New("archive and task ids must be positive")
	}
	return t.stageDeleteHostsExact(hosts, archiveID, taskID)
}

// 한국어 해설: wmu와 열린 SQLite 트랜잭션을 유지한 채 중복 제거한 호스트 행 삭제 및 파일 이동을 준비한다.
func (t *Traffic) stageDeleteHostsExact(hosts []string, archiveID, taskID int64) (*HostDeleteStage, error) {
	t.wmu.Lock()
	stage := &HostDeleteStage{traffic: t}
	fail := func(cause error) (*HostDeleteStage, error) {
		if rollbackErr := stage.rollbackLocked(); rollbackErr != nil {
			return nil, errors.Join(cause, fmt.Errorf("回滚流量删除: %w", rollbackErr))
		}
		return nil, cause
	}
	abort := func(cause error) (*HostDeleteStage, error) {
		t.wmu.Unlock()
		stage.done = true
		return nil, cause
	}

	unique := make([]string, 0, len(hosts))
	seen := make(map[string]struct{}, len(hosts))
	for _, host := range hosts {
		if _, ok := seen[host]; ok {
			continue
		}
		seen[host] = struct{}{}
		unique = append(unique, host)
	}

	// Exact hosts are known up front, so their directories are derived directly
	// rather than looked up: a host whose index rows are already gone can still
	// own an orphaned directory that this deletion should take with it.
	legacy := make([]string, 0, len(unique))
	for _, h := range unique {
		legacy = append(legacy, filepath.Join(t.dir, sanitize(h)))
	}

	tx, err := t.db.Begin()
	if err != nil {
		return abort(err)
	}
	stage.tx = tx
	for _, h := range unique {
		n, err := t.deleteWhere(tx, `host=?`, h)
		if err != nil {
			return fail(err)
		}
		stage.deleted += n
	}

	stage.stageDir, stage.moves, err = t.stageTreesForArchive(legacy, unique, archiveID, taskID)
	if err != nil {
		return fail(err)
	}
	return stage, nil
}

// RecoverHostDeleteStages resolves filesystem moves left by a process exit.
// SQLite rolls open transactions back on restart, while the supplied callback
// identifies task archives whose PostgreSQL compaction already committed.
// 한국어 해설: 재시작 후 남은 일지를 읽고 PostgreSQL 아카이브 커밋 여부에 따라 삭제를 마치거나 폴더를 복원한다.
// 이는 두 DB의 단일 분산 트랜잭션이 아니라 일지를 이용해 서로 다른 커밋 시점을 보정하는 복구 절차다.
func (t *Traffic) RecoverHostDeleteStages(archiveCommitted func(int64, int64) (bool, error)) error {
	if t == nil {
		return nil
	}
	t.wmu.Lock()
	defer t.wmu.Unlock()
	parent := filepath.Join(t.dir, "_delete_staging")
	entries, err := os.ReadDir(parent)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var errs []error
	needsGC := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		stageDir := filepath.Join(parent, entry.Name())
		raw, err := os.ReadFile(filepath.Join(stageDir, hostDeleteStageJournalName))
		if os.IsNotExist(err) {
			// Older versions did not persist enough information for lossless
			// recovery. Keep the directory for manual inspection.
			continue
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		var journal hostDeleteStageJournal
		if err := json.Unmarshal(raw, &journal); err != nil {
			errs = append(errs, fmt.Errorf("读取流量暂存日志 %s: %w", stageDir, err))
			continue
		}
		if journal.Version != 1 {
			errs = append(errs, fmt.Errorf("流量暂存日志 %s 的版本 %d 不受支持", stageDir, journal.Version))
			continue
		}
		moves := make([]stagedTrafficPath, 0, len(journal.Moves))
		for _, move := range journal.Moves {
			if !pathWithin(t.dir, move.Source) || !pathWithin(stageDir, move.Staged) {
				errs = append(errs, fmt.Errorf("流量暂存日志包含越界路径: %s", stageDir))
				moves = nil
				break
			}
			if _, err := os.Lstat(move.Staged); err == nil {
				moves = append(moves, stagedTrafficPath{source: move.Source, staged: move.Staged})
			} else if !os.IsNotExist(err) {
				errs = append(errs, err)
				moves = nil
				break
			}
		}
		if moves == nil {
			continue
		}
		committed := false
		if journal.ArchiveID > 0 {
			if archiveCommitted == nil {
				errs = append(errs, fmt.Errorf("流量归档 %d 无状态解析器", journal.ArchiveID))
				continue
			}
			committed, err = archiveCommitted(journal.ArchiveID, journal.TaskID)
			if err != nil {
				errs = append(errs, err)
				continue
			}
		} else if len(journal.Hosts) > 0 {
			committed, err = t.hostsHaveNoExchanges(journal.Hosts)
			if err != nil {
				errs = append(errs, err)
				continue
			}
		}
		if !committed {
			if err := restoreTrees(stageDir, moves); err != nil {
				errs = append(errs, err)
			}
			continue
		}
		if err := t.deleteArchivedHosts(journal.Hosts); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := os.RemoveAll(stageDir); err != nil {
			errs = append(errs, err)
			continue
		}
		needsGC = true
	}
	if needsGC {
		t.reclaim()
		if err := t.gcBlobs(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// 한국어 해설: 일지에 있는 모든 호스트의 SQL 행이 이미 사라졌는지 확인하여 일반 삭제의 커밋 여부를 추론한다.
func (t *Traffic) hostsHaveNoExchanges(hosts []string) (bool, error) {
	for _, host := range hosts {
		var count int
		if err := t.db.QueryRow(`SELECT count(*) FROM exchanges WHERE host=?`, host).Scan(&count); err != nil {
			return false, err
		}
		if count > 0 {
			return false, nil
		}
	}
	return true, nil
}

// 한국어 해설: 복구 시 아카이브로 이동이 확정된 호스트의 원시 트래픽 행을 하나의 SQLite 트랜잭션으로 제거한다.
func (t *Traffic) deleteArchivedHosts(hosts []string) error {
	tx, err := t.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	for _, host := range hosts {
		if _, err := t.deleteWhere(tx, `host=?`, host); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// 한국어 해설: 절대 경로와 상대 경로 관계로 일지 경로가 루트 밖인지 검사한다. 심볼릭 링크의 실제 대상을 해석하는 검사는 아니다.
func pathWithin(root, candidate string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	candidateAbs, err := filepath.Abs(candidate)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(rootAbs, candidateAbs)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

// Rollback discards the staged deletion, leaving the traffic store untouched.
// 한국어 해설: 미완료 삭제의 SQL 변경과 파일 이동을 되돌린다. nil 또는 이미 종료한 stage는 다시 처리하지 않는다.
func (s *HostDeleteStage) Rollback() error {
	if s == nil || s.done {
		return nil
	}
	return s.rollbackLocked()
}

// 한국어 해설: SQLite 롤백과 폴더 복원을 모두 시도하고 오류들을 합친 뒤 stage를 종료하고 wmu를 놓는다.
func (s *HostDeleteStage) rollbackLocked() error {
	if s == nil || s.done {
		return nil
	}
	var errs []error
	if s.tx != nil {
		if err := s.tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			errs = append(errs, fmt.Errorf("回滚流量索引: %w", err))
		}
	}
	if err := restoreTrees(s.stageDir, s.moves); err != nil {
		errs = append(errs, err)
	}
	s.done = true
	s.traffic.wmu.Unlock()
	return errors.Join(errs...)
}

// Commit makes the staged deletion permanent. SQLite is committed only after
// the caller has committed its PostgreSQL task deletion.
// 한국어 해설: 호출자의 PostgreSQL 처리가 성공한 후 SQLite 삭제를 확정하고 파일·blob·빈 공간 정리를 이어 간다.
// 정리 오류가 나더라도 이미 확정한 SQL 삭제까지 되돌렸다고 해석하면 안 된다.
func (s *HostDeleteStage) Commit() error {
	if s == nil || s.done {
		return nil
	}
	if err := s.tx.Commit(); err != nil {
		// A failed SQLite commit normally rolls the transaction back. Restore the
		// trees so the traffic store remains internally consistent and recoverable.
		restoreErr := restoreTrees(s.stageDir, s.moves)
		s.done = true
		s.traffic.wmu.Unlock()
		return errors.Join(fmt.Errorf("提交流量索引删除: %w", err), restoreErr)
	}
	// Unlinking the staged trees is what used to hold the write lock for hours;
	// it now runs in the background, while collection below only needs them to be
	// out of the live tree — which the staging rename already guaranteed.
	s.traffic.reapStage(s.stageDir)
	s.traffic.reclaim()
	var errs []error
	if err := s.traffic.gcBlobs(); err != nil {
		errs = append(errs, fmt.Errorf("回收流量 blob: %w", err))
	}
	s.done = true
	s.traffic.wmu.Unlock()
	return errors.Join(errs...)
}

// indexBytes is the index's real footprint on disk: the database plus its
// write-ahead log and shared-memory file, since those are what an operator sees
// the data directory holding.
// 한국어 해설: SQLite 본체뿐 아니라 WAL·SHM 파일도 더하여 운영자가 보는 실제 인덱스 디스크 사용량을 계산한다.
func (t *Traffic) indexBytes() int64 {
	base := filepath.Join(t.dir, "_index", "index.sqlite")
	var total int64
	for _, p := range []string{base, base + "-wal", base + "-shm"} {
		if st, err := os.Stat(p); err == nil {
			total += st.Size()
		}
	}
	return total
}

// reclaim hands space freed by a deletion back to the filesystem. Deleting rows
// only makes them invisible: SQLite chains their pages onto a freelist, and
// deleting from the contentless_delete full-text index writes tombstones rather
// than removing the postings they hide. Neither shrinks a single byte on disk,
// and since bodies below maxInlineBody live in that same file, an install that
// captures heavily ends up holding far more than the traffic it still has.
//
// The work is done in the background, in bounded steps that drop the write lock
// between them, because it is proportional to what was deleted — freeing several
// gigabytes under one lock would stall record(), which go-mitmproxy calls before
// it replies to the client, and so would stall the requests being recorded.
// Safe to call with wmu held: the background pass simply waits for it.
//
// Best-effort throughout. Failing to reclaim costs disk space, never
// correctness, so errors are logged and the next deletion resumes the work.
// 한국어 해설: 삭제 후 FTS의 tombstone 병합과 빈 페이지 반환을 백그라운드의 작은 구간으로 나눈다.
// 단일 실행·최대 512단계·5분 예산으로 수집 쓰기를 장시간 독점하지 않으며 종료 시 남은 회수를 포기한다.
func (t *Traffic) reclaim() {
	if !t.reclaiming.CompareAndSwap(false, true) {
		return // one pass at a time; a second would only contend for the lock
	}
	t.reaping.Go(func() {
		defer t.reclaiming.Store(false)
		// One pinned connection for the whole pass: a long reclamation issues
		// hundreds of statements, and letting the pool hand each one a different
		// connection would both churn connections and split the freelist readings
		// that decide when to stop away from the vacuum they measure.
		ctx := context.Background()
		conn, err := t.db.Conn(ctx)
		if err != nil {
			log.Printf("[traffic] 回收索引空间失败（获取连接）：%v", err)
			return
		}
		defer conn.Close()
		deadline := time.Now().Add(reclaimBudget)
		merges := reclaimMergeSteps
		for step := 0; ; step++ {
			t.wmu.Lock()
			progressed, err := t.reclaimChunk(ctx, conn, &merges)
			t.wmu.Unlock()
			if err != nil {
				log.Printf("[traffic] 回收索引空间失败：%v", err)
				return
			}
			if !progressed {
				break
			}
			if t.stopping() {
				return // shutdown must not wait out the remaining budget
			}
			if step+1 >= reclaimMaxSteps {
				log.Printf("[traffic] 索引空间回收未做完（已用满 %d 步上限），下次删除时继续", reclaimMaxSteps)
				return
			}
			if time.Now().After(deadline) {
				log.Printf("[traffic] 索引空间回收未做完（已用满 %s 预算），下次删除时继续", reclaimBudget)
				return
			}
		}
		// Truncating the log is what makes the reclamation visible on disk: in WAL
		// mode the freed pages are recorded there first, and a PASSIVE checkpoint
		// would leave the log itself sitting at its high-water mark.
		t.wmu.Lock()
		defer t.wmu.Unlock()
		if _, err := conn.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
			log.Printf("[traffic] 截断 WAL 失败：%v", err)
		}
	})
}

// reclaimChunk does one bounded step and reports whether it made progress, which
// is what the caller loops on. merges carries the remaining full-text merge
// budget and is spent down here. Callers must hold wmu.
// 한국어 해설: 한 번의 FTS 병합 및 incremental_vacuum을 수행하고 진전 여부를 반환한다. 옛 auto_vacuum=0 DB에서는 병합만 가능하다.
func (t *Traffic) reclaimChunk(ctx context.Context, conn *sql.Conn, merges *int) (bool, error) {
	progressed := false
	if t.fts && *merges > 0 {
		// A negative rank is fts5's page budget for one incremental merge.
		if _, err := conn.ExecContext(ctx, `INSERT INTO ex_fts(ex_fts, rank) VALUES('merge', ?)`, -reclaimMergePages); err != nil {
			return false, fmt.Errorf("合并全文索引: %w", err)
		}
		*merges--
		progressed = true
	}
	if !t.incrementalVacuum {
		// incremental_vacuum is a silent no-op on a database created with
		// auto_vacuum=0; only a full compaction can convert one. Merging the
		// full-text index above still pays off, so stop here rather than earlier.
		return progressed, nil
	}
	var before, after int
	if err := conn.QueryRowContext(ctx, `PRAGMA freelist_count`).Scan(&before); err != nil {
		return false, err
	}
	if before == 0 {
		return progressed, nil
	}
	// The budget is a constant and PRAGMA arguments cannot be bound as parameters.
	if _, err := conn.ExecContext(ctx, fmt.Sprintf(`PRAGMA incremental_vacuum(%d)`, reclaimChunkPages)); err != nil {
		return false, fmt.Errorf("回收索引空闲页: %w", err)
	}
	if err := conn.QueryRowContext(ctx, `PRAGMA freelist_count`).Scan(&after); err != nil {
		return false, err
	}
	// Progress, not an empty freelist, is the right stopping condition:
	// incremental_vacuum can only release pages it manages to move to the end of
	// the file, so a residual freelist it cannot shrink is a normal outcome.
	return progressed || after < before, nil
}

// gcBlobs removes blobs that no remaining exchange references. Live references
// come from blob_refs, so collection is one query plus a sweep of the blob
// directory — no exchange body is ever read. Emptied bucket directories are
// removed as well; the previous file-scanning collector deleted only files and
// left the buckets behind permanently. Best-effort: walk errors only skip the
// affected entries. Callers must hold wmu so this never races a concurrent
// record() writing a fresh blob + reference.
// 한국어 해설: SQL blob_refs와 과거 파일 포인터의 살아 있는 해시를 모아 무참조 파일 및 빈 버킷을 정리한다. 호출 중에는 wmu가 필요하다.
func (t *Traffic) gcBlobs() error {
	refs := make(map[string]struct{})
	rows, err := t.db.Query(`SELECT DISTINCT hash FROM blob_refs`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			rows.Close()
			return err
		}
		refs[h] = struct{}{}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if err := t.legacyBlobRefs(refs); err != nil {
		return err
	}

	root := filepath.Join(t.dir, "_blobs", "sha256")
	var buckets []string
	if err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if d.IsDir() {
			if p != root {
				buckets = append(buckets, p)
			}
			return nil
		}
		h := strings.TrimSuffix(d.Name(), ".bin")
		if _, ok := refs[h]; !ok {
			os.Remove(p)
		}
		return nil
	}); err != nil {
		return err
	}
	// Deepest first, so a two-level bucket left by the old layout collapses fully.
	// Remove fails harmlessly on a non-empty directory — exactly the guard needed
	// to avoid deleting a bucket that still holds live blobs.
	sort.Slice(buckets, func(i, j int) bool { return len(buckets[i]) > len(buckets[j]) })
	for _, b := range buckets {
		os.Remove(b)
	}
	return nil
}

// legacyBlobRefs adds hashes referenced by pre-SQLite exchanges, whose bodies are
// still .http files on disk holding "@blob sha256:<hex>" pointers. Without this
// the first collection after the upgrade would delete blobs that history still
// points at. Skipped entirely once no legacy rows remain — the steady state — so
// the tree walk is transitional rather than a permanent cost.
// 한국어 해설: 과거 path 행이 남아 있을 때에만 .http 파일의 @blob 포인터를 읽어 이전 데이터의 공유 본문이 지워지지 않도록 한다.
func (t *Traffic) legacyBlobRefs(refs map[string]struct{}) error {
	var n int
	if err := t.db.QueryRow(`SELECT COUNT(*) FROM exchanges WHERE path<>''`).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return nil
	}
	blobRe := regexp.MustCompile(`@blob sha256:([0-9a-f]{64})`)
	return filepath.WalkDir(t.dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() {
			// _blobs/_index/_ca never contain references; skip their subtrees.
			if p != t.dir && strings.HasPrefix(d.Name(), "_") {
				return filepath.SkipDir
			}
			return nil
		}
		if nm := d.Name(); nm != "request.http" && nm != "response.http" {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		for _, m := range blobRe.FindAllSubmatch(b, -1) {
			refs[string(m[1])] = struct{}{}
		}
		return nil
	})
}

// query returns one page of exchange metadata filtered by host and/or a url
// substring. Default page size is intentionally small (3) to keep tool results
// lightweight and capped at 10; page is 0-based (page*limit offset).
// 한국어 해설: 에이전트 검색은 호스트를 정확히 제한하고 기본 3개·최대 10개의 가벼운 메타데이터만 반환한다.
// 포트가 지정되면 원래 URL의 authority도 검사하여 같은 IP의 다른 서비스를 섞지 않는다.
func (t *Traffic) query(host, contains, bodyContains string, page, limit int) ([]ExchangeMeta, error) {
	if limit <= 0 {
		limit = 3
	}
	if limit > 10 {
		limit = 10
	}
	if page < 0 {
		page = 0
	}
	q := `SELECT id,ts,host,method,url_template,url,status,content_type,resp_len,path FROM exchanges WHERE 1=1`
	args := []any{}
	if host != "" {
		hostName, port, err := normalizeSearchHost(host)
		if err != nil {
			return nil, err
		}
		q += ` AND host=?`
		args = append(args, hostName)
		if port != "" {
			// Recorded rows historically store URL.Hostname() (without the port),
			// so constrain the original URL authority as a compatibility fallback.
			// New and old captures therefore share the same search contract.
			authority := net.JoinHostPort(hostName, port)
			q += ` AND (url LIKE ? OR url LIKE ? OR url LIKE ?)`
			args = append(args,
				"%://"+authority+"/%",
				"%://"+authority+"?%",
				"%://"+authority,
			)
		}
	}
	if contains != "" {
		q += ` AND (url LIKE ? OR url_template LIKE ?)`
		args = append(args, "%"+contains+"%", "%"+contains+"%")
	}
	if b := strings.TrimSpace(bodyContains); b != "" {
		cond, arg, ok := t.ftsFilter(b)
		if !ok {
			if !t.fts {
				return nil, fmt.Errorf("当前实例未启用全文索引，无法按正文搜索")
			}
			return nil, fmt.Errorf("正文搜索关键词至少需要 %d 个字符（当前 %d 个）", minTrigram, utf8.RuneCountInString(b))
		}
		q += ` AND ` + cond
		args = append(args, arg)
	}
	q += ` ORDER BY ts DESC LIMIT ? OFFSET ?`
	args = append(args, limit, page*limit)
	rows, err := t.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ExchangeMeta
	for rows.Next() {
		var m ExchangeMeta
		if err := rows.Scan(&m.ID, &m.TS, &m.Host, &m.Method, &m.URLTemplate, &m.URL, &m.Status, &m.ContentType, &m.RespLen, &m.Path); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// normalizeSearchHost accepts the forms agents commonly have at hand while
// keeping the stored host (URL.Hostname()) as the canonical host key. A port,
// when supplied, is applied to the URL authority so captures from different
// services on the same IP cannot be mixed.
// 한국어 해설: URL·host:port·IPv6 표기를 저장 키인 소문자 hostname과 별도 port로 나눈다. 포트는 1~65535인지 검사한다.
func normalizeSearchHost(raw string) (host, port string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", errors.New("host 为必填参数")
	}
	if strings.Contains(raw, "://") {
		u, parseErr := url.Parse(raw)
		if parseErr != nil || u.Host == "" {
			return "", "", fmt.Errorf("无法解析 host：%q", raw)
		}
		host, port = u.Hostname(), u.Port()
	} else if h, p, splitErr := net.SplitHostPort(raw); splitErr == nil {
		host, port = h, p
	} else if strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]") {
		host = strings.TrimSuffix(strings.TrimPrefix(raw, "["), "]")
	} else {
		host = raw
	}
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if host == "" {
		return "", "", fmt.Errorf("无法解析 host：%q", raw)
	}
	if port != "" {
		p, parseErr := strconv.Atoi(port)
		if parseErr != nil || p < 1 || p > 65535 {
			return "", "", fmt.Errorf("端口无效：%q", port)
		}
		port = strconv.Itoa(p)
	}
	return strings.ToLower(host), port, nil
}

// Tools exposes traffic lookup to work agents so they query already-captured
// traffic instead of re-curling the same resource (token + dedup win).
// 한국어 해설: traffic_search → traffic_get → traffic_blob의 점진적 읽기 도구를 만든다. 먼저 ID를 찾고 필요한 원문만 읽어 모델 문맥을 절약한다.
// 이 함수의 도구 설명·JSON 스키마·중국어 결과 문자열은 모델과 프로그램의 실행 계약이므로 원문을 유지한다.
func (t *Traffic) Tools() []actool.CoreTool {
	allow := func(context.Context, json.RawMessage, permission.Context) permission.Decision {
		return permission.Allowed()
	}
	ro := func(json.RawMessage) bool { return true }

	search := actool.Build(actool.Spec{
		Name:        "traffic_search",
		Description: TrafficSearchDescription,
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"host":          map[string]any{"type": "string", "description": "按主机过滤（必填；如 '107.172.96.177'、'107.172.96.177:8082' 或 'http://107.172.96.177:8082/path'）"},
				"contains":      map[string]any{"type": "string", "description": "URL 子串过滤（可选，如 'api' / 'login'）"},
				"body_contains": map[string]any{"type": "string", "description": "正文全文搜索（可选，至少 3 个字符），匹配请求/响应的头与正文，如 'password' / 'root:x:0' / '内网测试'"},
				"limit":         map[string]any{"type": "integer", "description": "每页条数，默认 3，最大 10"},
				"page":          map[string]any{"type": "integer", "description": "页码，从 0 开始，默认 0（按 ts 倒序分页）"},
			},
			"required": []any{"host"},
		},
		ReadOnly:    ro,
		Permissions: allow,
		Run: func(_ context.Context, in json.RawMessage, _ *actool.ToolContext) (actool.Result, error) {
			var a struct {
				Host, Contains string
				BodyContains   string `json:"body_contains"`
				Limit          int
				Page           int
			}
			_ = json.Unmarshal(in, &a)
			if strings.TrimSpace(a.Host) == "" {
				return actool.Errorf("host 为必填参数：请指定裸主机、主机:端口或完整 URL，避免全库扫描。"), nil
			}
			rows, err := t.query(a.Host, a.Contains, a.BodyContains, a.Page, a.Limit)
			if err != nil {
				return actool.Errorf(err.Error()), nil
			}
			if len(rows) == 0 {
				return actool.Text("无匹配流量。"), nil
			}
			// 精简为最小索引：仅保留定位所需字段 + 响应码/长度，不带任何响应内容。
			type liteRow struct {
				ID      string `json:"id"`
				Method  string `json:"method"`
				URL     string `json:"url"`
				Status  int    `json:"status"`
				RespLen int    `json:"resp_len"`
			}
			lite := make([]liteRow, 0, len(rows))
			for _, r := range rows {
				lite = append(lite, liteRow{ID: r.ID, Method: r.Method, URL: r.URL, Status: r.Status, RespLen: r.RespLen})
			}
			b, _ := json.Marshal(lite)
			return actool.Text(string(b)), nil
		},
	})

	get := actool.Build(actool.Spec{
		Name:        "traffic_get",
		Description: "按 id 取一条已抓流量的请求/响应原文（过大会截断）。配合 traffic_search 用，避免重复 curl。",
		Schema: map[string]any{
			"type":       "object",
			"properties": map[string]any{"id": map[string]any{"type": "string", "description": "traffic_search 返回的 id"}},
			"required":   []any{"id"},
		},
		ReadOnly:    ro,
		Permissions: allow,
		Run: func(_ context.Context, in json.RawMessage, _ *actool.ToolContext) (actool.Result, error) {
			var a struct{ ID string }
			_ = json.Unmarshal(in, &a)
			req, resp, err := t.Get(a.ID)
			if err != nil {
				return actool.Errorf(err.Error()), nil
			}
			return actool.Text("=== REQUEST ===\n" + clip(req, 2500) + "\n\n=== RESPONSE ===\n" + clip(resp, 4000)), nil
		},
	})

	blob := actool.Build(actool.Spec{
		Name:        "traffic_blob",
		Description: "分段读取超大请求/响应体的原文。traffic_get 里显示为 '…[truncated] @blob sha256:<hash>' 的部分即存放于此，把该 hash 传进来即可取完整内容。单次最多返回 8KB，用 offset 继续往后读（返回结果会给出总长度）。适合翻阅备份文件、源码泄露、大 JSON 导出等超过内联阈值的响应。",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"hash":   map[string]any{"type": "string", "description": "traffic_get 中 @blob sha256: 后面的 64 位十六进制值"},
				"offset": map[string]any{"type": "integer", "description": "起始字节偏移，默认 0"},
				"length": map[string]any{"type": "integer", "description": "本次读取字节数，默认且最大 8192"},
			},
			"required": []any{"hash"},
		},
		ReadOnly:    ro,
		Permissions: allow,
		Run: func(_ context.Context, in json.RawMessage, _ *actool.ToolContext) (actool.Result, error) {
			var a struct {
				Hash   string
				Offset int64
				Length int64
			}
			_ = json.Unmarshal(in, &a)
			if a.Length <= 0 || a.Length > maxBlobRead {
				a.Length = maxBlobRead
			}
			data, total, err := t.BlobRange(a.Hash, a.Offset, a.Length)
			if err != nil {
				return actool.Errorf(err.Error()), nil
			}
			if len(data) == 0 {
				return actool.Text(fmt.Sprintf("偏移 %d 已超出内容长度（总长 %d 字节）。", a.Offset, total)), nil
			}
			head := fmt.Sprintf("[offset=%d 本次=%d 总长=%d]\n", a.Offset, len(data), total)
			if isBinaryBody("", data) {
				return actool.Text(head + "二进制内容，以十六进制展示前 512 字节：\n" + hex.EncodeToString(clipBytes(data, 512))), nil
			}
			return actool.Text(head + truncateUTF8(data, len(data))), nil
		},
	})

	return []actool.CoreTool{search, get, blob}
}

// SeedToolMetas returns the traffic tools built on a ZERO receiver, for seeding the
// tools catalog (metadata only — Name/Description/InputSchema). The handlers close
// over the nil receiver but are never invoked on this instance, so it is safe.
// 한국어 해설: 실제 수집 DB 없이 도구 이름·설명·스키마를 카탈로그에 등록하기 위한 생성 경로다. 반환 핸들러를 실행하는 용도가 아니다.
func SeedToolMetas() []actool.CoreTool { return (&Traffic{}).Tools() }

// 한국어 해설: 도구 출력 문자열을 바이트 수로 짧게 만든다. UTF-8 경계 보존이나 원문 복원은 보장하지 않으므로 전체 본문은 저장소에서 읽는다.
func clip(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + fmt.Sprintf("\n... [截断，共 %d 字节；完整在流量文件树] ...", len(s))
}
