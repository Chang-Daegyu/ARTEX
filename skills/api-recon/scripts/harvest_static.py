#!/usr/bin/env python3
# [한국어 해설] SPA 정적 JavaScript 자료 수집
# HTML의 스크립트 URL에서 시작해 webpack/Vite 청크 정보를 확장하고 API 후보 경로와 화면 라우트를 파일로 저장합니다.
# 정규식·manifest 형식은 사이트별 참고용입니다. HTTP 메서드나 실제 접근 권한은 정적 경로만으로 확정하지 않습니다.
# TLS 검증을 끄는 원본 설정과 다운로드 병렬 수를 유지합니다. main을 실행하면 네트워크 요청과 로컬 파일 쓰기가 발생합니다.
"""
harvest_static.py <BASE_URL> <OUTDIR>

参考模板 — 非通用成品。执行前须按目标站点调整，常见改动：
  - extract_endpoints 正则（endpoint 方言）
  - webpack/Vite manifest 解析逻辑
  - 微前端 publicPath、重试策略

Static SPA bundle harvester. Framework-agnostic; tuned for webpack + Vite.
  1. Fetch entry HTML, collect script/module references.
  2. Parse the runtime chunk manifest(s) and download EVERY chunk (not just the
     ones in HTML), looping until no new chunk ids appear. Handles multiple
     micro-frontend runtimes, each with its own publicPath.
  3. Extract API endpoints and route paths from all downloaded JS.

Stdlib only. TLS verification is disabled (recon against self-signed/internal hosts).
"""
import sys, os, re, ssl, json, urllib.request, urllib.parse
from concurrent.futures import ThreadPoolExecutor

CTX = ssl.create_default_context(); CTX.check_hostname = False; CTX.verify_mode = ssl.CERT_NONE
UA = "Mozilla/5.0 (spa-api-recon)"

# [한국어 흐름] 지정 URL을 읽어 본문과 상태를 반환합니다. 오류는 빈 본문과 상태 코드로 바꾸므로 호출자가 다운로드 성공 여부를 구분해야 합니다.
def fetch(url, binary=False):
    try:
        req = urllib.request.Request(url, headers={"User-Agent": UA})
        with urllib.request.urlopen(req, context=CTX, timeout=25) as r:
            data = r.read()
            return (data if binary else data.decode("utf-8", "ignore")), r.status
    except Exception as e:
        code = getattr(e, "code", 0)
        return "", code

# [한국어 흐름] scheme과 netloc만 조합해 청크 상대경로의 기준 출처를 만듭니다.
def origin_of(u):
    p = urllib.parse.urlparse(u)
    return f"{p.scheme}://{p.netloc}"

# ---- chunk manifest parsing -------------------------------------------------
# [한국어 흐름] 닫는 중괄호에서 역방향 깊이 카운트로 객체 구간을 찾습니다. 완전한 JavaScript 구문 파서는 아닙니다.
def matched_block(s, end_brace_idx):
    """Walk back from a '}' index to its matching '{' and return the object body."""
    depth = 0; j = end_brace_idx
    while j >= 0:
        if s[j] == '}': depth += 1
        elif s[j] == '{':
            depth -= 1
            if depth == 0: return s[j:end_brace_idx+1]
        j -= 1
    return ""

# [한국어 흐름] webpack 청크 ID/해시와 publicPath 후보, Vite assets 경로를 정규식으로 추출합니다.
def parse_chunk_maps(js_text):
    """Return list of (publicPath_hint, {id: hash}) for every `}[x]+".js"` map
    (webpack __webpack_require__.u) found in the text."""
    maps = []
    # publicPath hints in this file: .p="..."  or publicPath="..."
    pubs = re.findall(r'(?:\.p|publicPath)\s*=\s*"([^"]*)"', js_text)
    for m in re.finditer(r'\}\[[A-Za-z_$]\]\s*\+\s*"\.js"', js_text):
        body = matched_block(js_text, m.start())
        pairs = re.findall(r'(\d+):"([0-9a-fA-F]{6,16})"', body)
        if pairs:
            maps.append((pubs, dict(pairs)))
    # Vite style: __vite__mapDeps map of "assets/xx.js"
    for fn in re.findall(r'"(assets/[^"]+\.js)"', js_text):
        maps.append((["/"], {"__vite__": fn}))
    return maps

# [한국어 흐름] 앞서 찾은 manifest의 ID·해시·publicPath를 절대 URL 후보 집합으로 변환합니다.
def chunk_urls(base, js_text):
    """Yield absolute chunk URLs reconstructable from this file's manifest(s)."""
    origin = origin_of(base)
    out = set()
    for pubs, idmap in parse_chunk_maps(js_text):
        paths = pubs or ["/assets/", "/"]
        for cid, h in idmap.items():
            if cid == "__vite__":
                fn = h  # already "assets/xx.js"
                for pp in paths:
                    out.add(urllib.parse.urljoin(origin + "/", fn))
                continue
            for pp in paths:
                if pp.startswith("http"): bareorigin = ""; prefix = pp
                else: bareorigin = origin; prefix = pp if pp.startswith("/") else "/"+pp
                if not prefix.endswith("/"): prefix += "/"
                out.add(f"{bareorigin}{prefix}{cid}.{h}.js")
    return out

# ---- endpoint / route extraction -------------------------------------------
API_HINT = re.compile(r'/(?:api|rest|service|services|gateway|graphql|v\d|web|admin|backend|open)\b', re.I)
ASSET_EXT = re.compile(r'\.(js|css|png|jpe?g|svg|gif|woff2?|ttf|ico|map|json|mp4|webp)(\?|$)', re.I)

# [한국어 흐름] 문자열 경로와 문자열 연결 접두어를 추출하고 정적 자원 확장자를 제외해 API 후보와 기타 경로를 나눕니다.
def extract_endpoints(js_text):
    paths = set()
    # quoted ('/...'), double-quoted, and backtick template paths
    for m in re.findall(r'''["'`](/[A-Za-z0-9_\-./{}$:]+)["'`]''', js_text):
        paths.add(m)
    # concatenation heads:  "/api/x/" + var
    for m in re.findall(r'''["'](/[A-Za-z0-9_\-./]+/)["']\s*\+''', js_text):
        paths.add(m)
    api, other = set(), set()
    for p in paths:
        if ASSET_EXT.search(p): continue
        if p.count('/') < 2 and not API_HINT.search(p): continue
        (api if API_HINT.search(p) else other).add(p)
    return api, other

# [한국어 흐름] path/to/redirect/href 리터럴 중 API나 정적 자원으로 보이지 않는 항목을 화면 경로 후보로 모읍니다.
def extract_routes(js_text):
    r = set()
    for key in ('path', 'to', 'redirect', 'href'):
        for m in re.findall(key + r'''\s*:\s*["'](/[A-Za-z0-9_\-/:]*)["']''', js_text):
            if not ASSET_EXT.search(m) and not API_HINT.search(m):
                r.add(m)
    return r

# ---- main -------------------------------------------------------------------
# [한국어 흐름] HTML→초기 JS→최대 6회 청크 확장→실패 후보 재시도→경로 추출→정렬된 산출물 저장 순서의 진입점입니다.
def main():
    if len(sys.argv) < 3:
        print("usage: harvest_static.py <BASE_URL> <OUTDIR>"); sys.exit(1)
    base, outdir = sys.argv[1], sys.argv[2]
    if not base.startswith("http"): base = "https://" + base
    jsdir = os.path.join(outdir, "js"); os.makedirs(jsdir, exist_ok=True)

    print(f"[*] entry: {base}")
    html, status = fetch(base)
    open(os.path.join(outdir, "index.html"), "w").write(html)
    origin = origin_of(base)

    # initial scripts from HTML
    srcs = set(re.findall(r'<script[^>]+src="([^"]+\.js[^"]*)"', html))
    srcs |= set(re.findall(r'(?:src|href)="([^"]*\.js)"', html))
    seed = set()
    for s in srcs:
        seed.add(s if s.startswith("http") else urllib.parse.urljoin(base, s))
    print(f"[*] {len(seed)} scripts referenced in HTML")

    have = {}  # url -> local path
    # [한국어 흐름] 이미 시도한 URL을 확인하고 JS 응답을 로컬 basename 파일로 저장합니다. URL이 달라도 같은 파일명을 쓰는 경우를 고려해야 하는 참고 구현입니다.
    def dl(url):
        fn = os.path.basename(urllib.parse.urlparse(url).path)
        if not fn.endswith(".js"): return None
        lp = os.path.join(jsdir, fn)
        if url in have: return have[url]
        data, st = fetch(url, binary=True)
        if st == 200 and data and not data[:15].lstrip().startswith(b"<"):
            open(lp, "wb").write(data); have[url] = lp; return lp
        return None

    with ThreadPoolExecutor(max_workers=20) as ex:
        list(ex.map(dl, seed))

    # iteratively expand via chunk manifests (chunks reference more chunks)
    seen_urls = set(have.keys()); frontier = list(have.values())
    rounds = 0
    while frontier and rounds < 6:
        rounds += 1
        new_urls = set()
        for lp in frontier:
            try: txt = open(lp, encoding="utf-8", errors="ignore").read()
            except: continue
            for cu in chunk_urls(base, txt):
                if cu not in seen_urls: new_urls.add(cu)
        seen_urls |= new_urls
        if not new_urls: break
        print(f"[*] round {rounds}: {len(new_urls)} new chunk urls from manifest")
        before = set(have.values())
        with ThreadPoolExecutor(max_workers=24) as ex:
            list(ex.map(dl, new_urls))
        frontier = [p for p in have.values() if p not in before]

    # retry-once any manifest chunk that 404'd (transient failures are real)
    all_manifest = set()
    for lp in list(have.values()):
        try: all_manifest |= chunk_urls(base, open(lp, encoding="utf-8", errors="ignore").read())
        except: pass
    missing = [u for u in all_manifest if u not in have]
    if missing:
        with ThreadPoolExecutor(max_workers=24) as ex:
            list(ex.map(dl, missing))
        still = [u for u in all_manifest if u not in have]
        print(f"[*] manifest chunks: {len(all_manifest)} | downloaded {len(have)} | "
              f"unreachable {len(still)} (CSS-only / undeployed)")

    print(f"[+] total JS downloaded: {len(have)}")

    # extract from everything
    api, other, routes = set(), set(), set()
    for lp in have.values():
        try: txt = open(lp, encoding="utf-8", errors="ignore").read()
        except: continue
        a, o = extract_endpoints(txt); api |= a; other |= o
        routes |= extract_routes(txt)

    # [한국어 흐름] 중복 제거 집합을 정렬하여 출력 파일로 만듭니다. 출력 순서를 안정적으로 해 두면 결과 비교가 쉽습니다.
    def dump(name, items):
        path = os.path.join(outdir, name)
        open(path, "w").write("\n".join(sorted(items)))
        return path
    dump("api_static.txt", api)
    dump("paths_other.txt", other)
    dump("routes.txt", routes)
    dump("chunkmap.txt", sorted(os.path.basename(u) for u in all_manifest))

    print(f"[+] api endpoints: {len(api)}  (api_static.txt)")
    print(f"[+] other paths:   {len(other)} (paths_other.txt)")
    print(f"[+] route paths:   {len(routes)} (routes.txt)")
    print(f"[+] outdir: {outdir}")
    print("\n[next] reverse the 3 gate facts (see reference.md), fill config.json, "
          "then: node runtime_harvest.js config.json")

if __name__ == "__main__":
    main()
