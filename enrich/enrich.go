// Package enrich is the engine-side (non-AI) asset auto-completion layer described
// in docs/资产模型与自动关联设计.md §5: an async worker pool that resolves domains
// (dnsx) and probes web assets (HTTP, through the recording proxy) and writes the
// results back into the asset graph — creating IP/port nodes, resolves/exposes
// edges, and filling attrs.dns / attrs.http. DNS is ungated; HTTP probing is gated
// by RoE (§5.2).
// [한국어 파일 안내] enrich/enrich.go
// AI 호출 없이 DNS 및 HTTP 메타데이터를 보충할 수 있는 비동기 작업 풀이다.
// ResolveDomain/ProbeSite로 들어온 작업을 큐에서 꺼내 자산 UPSERT로 저장하고 같은 kind:id에는 5분 냉각 시간을 둔다.
// 원본 상단의 RoE·resolves/exposes edge 설명은 역사적 설계 설명이다. 아래 현재 doHTTP에는 별도 RoE 판정 호출이 없다.
// 또한 엔진을 생성했다는 사실만으로 자동 보충이 실행되지 않는다. 호출부가 실제 enqueue 함수를 호출하는지 함께 추적해야 한다.
package enrich

import (
	"crypto/tls"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Autumn-27/artex/db"

	"github.com/miekg/dns"
	"github.com/projectdiscovery/dnsx/libs/dnsx"
)

type jobKind int

const (
	jobDNS  jobKind = iota // resolve a domain
	jobHTTP                // probe a web asset (site)
)

// 한국어 자료형: 비동기 보충 한 건의 종류, 자산 ID, 도메인 또는 URL 인자다.
type job struct {
	kind jobKind
	id   int64  // asset id (domain for DNS, site for HTTP)
	arg  string // host (DNS) or url (HTTP)
}

// Engine owns the resolver, the proxy-routed HTTP client, and the worker pool.
// 한국어 자료형: DNS 리졸버·HTTP 클라이언트·메모리 큐·냉각 기록의 수명을 소유한다.
type Engine struct {
	as     *db.AssetStore
	resolv *dnsx.DNSX
	client *http.Client

	jobs   chan job
	cool   sync.Map // dedup/cooldown: "kind:id" -> time.Time (last run)
	once   sync.Once
	closed chan struct{}
}

const (
	cooldown    = 5 * time.Minute
	httpTimeout = 12 * time.Second
	queueSize   = 1024
)

// New builds the engine. proxy() returns the recording-proxy address to route HTTP
// probes through (so they land in the traffic store), evaluated per request so the
// runtime traffic-capture toggle takes effect live; "" = direct. Returns a usable
// engine even if the resolver fails to init (DNS becomes a no-op).
// 한국어 해설: DNSX와 HTTP 클라이언트 및 1,024칸 큐를 만들고 기본 4개 worker를 띄운다. DNS 초기화만 실패하면 HTTP 처리는 남는다.
func New(as *db.AssetStore, proxy func() string, workers int) *Engine {
	if workers <= 0 {
		workers = 4
	}
	resolv, err := dnsx.New(dnsx.Options{
		BaseResolvers: dnsx.DefaultResolvers,
		MaxRetries:    3,
		QuestionTypes: []uint16{dns.TypeA, dns.TypeAAAA, dns.TypeCNAME},
		Timeout:       4 * time.Second,
	})
	if err != nil {
		log.Printf("[enrich] dnsx 初始化失败，DNS 解析停用：%v", err)
		resolv = nil
	}
	e := &Engine{
		as:     as,
		resolv: resolv,
		client: buildClient(proxy),
		jobs:   make(chan job, queueSize),
		closed: make(chan struct{}),
	}
	for i := 0; i < workers; i++ {
		go e.worker()
	}
	return e
}

// buildClient returns an HTTP client that dials via the recording proxy (resolved
// per-request via proxy(), so the traffic-capture toggle applies live) and skips
// TLS verification (the proxy re-signs with its MITM CA; targets are often
// self-signed — this is a pentest probe).
// 한국어 해설: 요청 때마다 proxy 콜백을 읽고 12초 제한·TLS 검증 생략·자동 리다이렉트 중지를 적용한 probe 클라이언트를 만든다.
func buildClient(proxy func() string) *http.Client {
	tr := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives:   true,
		TLSHandshakeTimeout: httpTimeout,
	}
	if proxy != nil {
		tr.Proxy = func(*http.Request) (*url.URL, error) {
			p := proxy()
			if p == "" {
				return nil, nil // direct
			}
			return url.Parse(p)
		}
	}
	return &http.Client{
		Transport: tr,
		Timeout:   httpTimeout,
		// cap redirects; keep them within scope by re-checking at probe time
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// ResolveDomain enqueues a DNS resolution for a domain asset (ungated). host is the
// FQDN. No-op for an empty engine.
// 한국어 해설: 도메인 자산 ID와 FQDN을 DNS 보충 작업으로 큐에 넣는다. 호출만으로 동기 결과가 반환되지는 않는다.
func (e *Engine) ResolveDomain(id int64, host string) { e.enqueue(job{jobDNS, id, host}) }

// ProbeSite enqueues an HTTP probe for a web-asset (site). rawURL is the site URL.
// 한국어 해설: 웹 자산 ID와 URL을 HTTP GET 보충 작업으로 큐에 넣는다.
func (e *Engine) ProbeSite(id int64, rawURL string) { e.enqueue(job{jobHTTP, id, rawURL}) }

// 한국어 해설: nil 엔진·잘못된 ID·빈 인자를 무시하고 큐가 가득 차면 작업을 버린다. 영속 작업 큐가 아닌 best-effort 보충이다.
func (e *Engine) enqueue(j job) {
	if e == nil || id0(j.id) || j.arg == "" {
		return
	}
	select {
	case e.jobs <- j:
	default: // queue full → drop (best-effort enrichment)
		log.Printf("[enrich] 队列已满，丢弃任务 kind=%d id=%d", j.kind, j.id)
	}
}

// 한국어 해설: 0 이하 자산 ID를 아직 유효하지 않은 참조로 판별한다.
func id0(id int64) bool { return id <= 0 }

// Close stops the workers (idempotent).
// 한국어 해설: 종료 채널을 한 번만 닫아 worker가 다음 선택에서 빠져나오도록 한다. 진행 중 요청 완료를 기다리는 WaitGroup은 없다.
func (e *Engine) Close() {
	if e == nil {
		return
	}
	e.once.Do(func() { close(e.closed) })
}

// 한국어 해설: 종료 신호 또는 보충 작업을 기다리고 냉각 기간이 아니면 DNS/HTTP별 처리기로 분기한다.
func (e *Engine) worker() {
	for {
		select {
		case <-e.closed:
			return
		case j := <-e.jobs:
			if e.onCooldown(j) {
				continue
			}
			switch j.kind {
			case jobDNS:
				e.doDNS(j.id, j.arg)
			case jobHTTP:
				e.doHTTP(j.id, j.arg)
			}
		}
	}
}

// onCooldown returns true (skip) if this (kind,id) ran within the cooldown window.
// 한국어 해설: kind와 자산 ID별 최근 실행 시각을 메모리에 보관한다. Load/Store 조합이므로 엄밀한 원자 중복 실행 방지 락으로 해석하지 않는다.
func (e *Engine) onCooldown(j job) bool {
	key := string(rune(j.kind)) + ":" + itoa(j.id)
	if v, ok := e.cool.Load(key); ok {
		if t, ok := v.(time.Time); ok && time.Since(t) < cooldown {
			return true
		}
	}
	e.cool.Store(key, time.Now())
	return false
}

// 한국어 해설: int64 자산 ID를 십진 문자열로 바꾸어 냉각 키를 구성한다.
func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// ---- DNS ----

// 한국어 해설: A·AAAA·CNAME 응답을 조회하고 IP 및 subdomain UPSERT에 전달한다. 개별 저장 오류는 현재 경로에서 무시된다.
func (e *Engine) doDNS(id int64, host string) {
	if e.resolv == nil {
		return
	}
	data, err := e.resolv.QueryMultiple(host)
	if err != nil || data == nil {
		return
	}
	ips := uniq(append(append([]string{}, data.A...), data.AAAA...))
	// Upsert resolved IPs into the asset store.
	for _, ip := range ips {
		_, _ = e.as.UpsertIP(db.UpsertIPReq{
			IP:           ip,
			BoundDomains: []string{host},
		})
	}
	// Record A records as subdomains if host looks like a subdomain.
	for _, a := range data.A {
		_, _ = e.as.UpsertSubdomain(db.UpsertSubdomainReq{
			Domain:      host,
			RecordType:  "A",
			RecordValue: []string{a},
		})
	}
	for _, aaaa := range data.AAAA {
		_, _ = e.as.UpsertSubdomain(db.UpsertSubdomainReq{
			Domain:      host,
			RecordType:  "AAAA",
			RecordValue: []string{aaaa},
		})
	}
	for _, cname := range data.CNAME {
		_, _ = e.as.UpsertSubdomain(db.UpsertSubdomainReq{
			Domain:      host,
			RecordType:  "CNAME",
			RecordValue: []string{cname},
		})
	}
}

// ---- HTTP probe ----

var reTitle = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)

// 한국어 해설: 한 번의 GET 결과에서 최대 1 MiB만 읽어 상태 코드·읽힌 길이·페이지 제목을 HTTPService에 저장한다.
// 이 ContentLength는 원본 응답 전체 길이가 아니라 실제 제한해서 읽은 바이트 수일 수 있다.
func (e *Engine) doHTTP(id int64, rawURL string) {
	host := hostOf(rawURL)
	if host == "" {
		return
	}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", "artex-enrich/1.0")
	resp, err := e.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // cap 1 MiB
	statusCode := resp.StatusCode
	bodyLen := int64(len(body))
	title := extractTitle(body)
	_, _ = e.as.UpsertHTTPService(db.UpsertHTTPServiceReq{
		URL:           rawURL,
		StatusCode:    &statusCode,
		ContentLength: &bodyLen,
		PageTitle:     title,
	})
}

// 한국어 해설: HTML title 요소를 정규식으로 찾고 엔티티와 주변 공백을 정리한다. 완전한 DOM 파서가 아니다.
func extractTitle(body []byte) string {
	m := reTitle.FindSubmatch(body)
	if m == nil {
		return ""
	}
	return strings.TrimSpace(html.UnescapeString(string(m[1])))
}

// 한국어 해설: URL에서 호스트 이름을 꺼내며 파싱 실패는 빈 문자열로 반환한다.
func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// 한국어 해설: nil·빈 문자열·일부 숫자 0을 빈 값으로 판별하는 보조 함수다. 현재 보충 흐름에서 실제 사용 여부는 호출부를 확인한다.
func isBlank(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return x == ""
	case float64:
		return x == 0
	case int:
		return x == 0
	}
	return false
}

// 한국어 해설: 원래 순서를 유지한 채 중복 문자열을 제거한다. 입력 슬라이스의 backing array를 재사용한다.
func uniq(in []string) []string {
	seen := map[string]struct{}{}
	out := in[:0]
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
