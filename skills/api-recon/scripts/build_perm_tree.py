#!/usr/bin/env python3
# [한국어 해설] 프런트 권한·메뉴 구조의 로컬 stub 생성
# 다운로드한 JS의 userRouteAuth와 route_map.json을 읽어 코드·경로·부모 관계를 추정합니다.
# 생성한 메뉴 트리와 권한 목록은 UI 렌더링을 위한 모의 응답입니다. 실제 서버 권한 부여나 로그인 성공을 의미하지 않습니다.
# 기본 루트/접두어/정규식은 특정 번들 형태의 휴리스틱이므로 다른 프런트엔드에는 그대로 맞지 않을 수 있습니다.
"""
build_perm_tree.py <JSDIR> <OUTDIR> [--config recon/config.json]

Rebuild a permissive permission/menu stub from frontend auth modules.

Typical consumer chain (TDP / enterprise admin SPAs):
  POST /api/web/user/role_permissions  -> { permissions: [...], role_type }
  POST /api/web/permissions/all        -> tree[{ code, position, children }]
  getResultTree(tree, permissions)     -> menu include lists
  userRouteAuth[code].url              -> frontend route path

This script:
  1. Locates userRouteAuth={MONITOR:{url:...},...} in js/
  2. Resolves webpack alias refs (He=o.DASHBOARD) via route_map.json
  3. Infers hierarchy from code prefixes (MONITOR_ -> MONITOR)
  4. Writes permissions_tree.json, permissions_all_stub.json,
     role_permissions_stub.json, userRouteAuth.json
  5. Optionally patches config.json stubs (permissions/all + role_permissions)

Adjust ROOTS / PREFIX_PARENT / EXTRA_PARENT per target if heuristics miss nodes.
"""
import re, json, os, sys, glob, argparse

DEFAULT_ROOTS = [
    'MONITOR', 'THREAT', 'ASSETS_RISK', 'INVESTIGATION', 'MANAGEMENT',
    'AGENT_EVIDENCE', 'MDR', 'PLATFORM',
]
DEFAULT_PREFIX_PARENT = {
    'MONITOR_': 'MONITOR', 'THREAT_': 'THREAT', 'ASSETS_RISK_': 'ASSETS_RISK',
    'INVESTIGATION_': 'INVESTIGATION', 'MANAGEMENT_': 'MANAGEMENT', 'MDR_': 'MDR',
    'PLATFORM_': 'PLATFORM', 'CUSTOM_': 'PLATFORM', 'FALSE_': 'PLATFORM',
}
DEFAULT_EXTRA_PARENT = {
    'LINKAGE_DISPOSAL': 'MANAGEMENT',
    'ASSETS_RISK_API': 'ASSETS_RISK',
    'ASSETS_RISK_WEAK_PWD': 'ASSETS_RISK',
}


# [한국어 흐름] userRouteAuth 문자열과 특정 객체 시작 패턴을 가진 JS 후보를 찾습니다. 완전한 모듈 해석 대신 소스 문자열을 검사합니다.
def find_auth_file(jsdir):
    best_fp, best_len = None, 0
    for fp in glob.glob(os.path.join(jsdir, '*.js')):
        try:
            txt = open(fp, encoding='utf-8', errors='ignore').read()
        except Exception:
            continue
        if 'userRouteAuth' not in txt:
            continue
        m = re.search(r'userRouteAuth=\{MONITOR:', txt)
        if m and len(txt) > best_len:
            best_fp, best_len = fp, len(m.group(0))
        elif 'userRouteAuth' in txt and best_fp is None:
            best_fp = fp
    return best_fp


# [한국어 흐름] 번들 앞부분에서 변수=o.KEY 모양을 수집해 축약 변수와 라우트 키를 연결합니다.
def parse_aliases(chunk):
    aliases = {}
    for m in re.finditer(r'([a-zA-Z_$][\w$]*)=o\.([A-Z_0-9]+)\b', chunk[:8000]):
        aliases[m.group(1)] = m.group(2)
    return aliases


# [한국어 흐름] 문자열 리터럴은 JSON으로 읽고 번들 별칭은 route_map의 link로 해석합니다. 미해결 참조는 토큰을 그대로 반환합니다.
def resolve_ref(token, aliases, route_map):
    token = token.strip()
    if token.startswith('"'):
        return json.loads(token)
    if token in aliases:
        key = aliases[token]
        return route_map.get(key, {}).get('link', key)
    return token


# [한국어 흐름] 권한 코드별 url/control 구조를 정규식으로 추출하고 별칭을 해석합니다. 다른 번들 구조에서는 정규식 조정이 필요한 지점입니다.
def parse_user_route_auth(txt, route_map):
    m = re.search(r'(?:t\.)?userRouteAuth=(\{MONITOR:.*?\})\},', txt, re.S)
    if not m:
        m = re.search(r'userRouteAuth=(\{[A-Z_0-9]+:\{url:', txt)
        if not m:
            return None, {}
        # greedy fallback — trim at next webpack module
        body = m.group(1)
        end = body.rfind('}')
        obj_src = body[: end + 1] if end > 0 else body
    else:
        obj_src = m.group(1)

    chunk_start = txt.find('userRouteAuth')
    chunk = txt[chunk_start:chunk_start + 35000]
    aliases = parse_aliases(chunk)

    entries = {}
    for em in re.finditer(
        r'([A-Z_0-9]+):\{url:([^,}]+)(?:,control:(\[.*?\]|[^,}]+))?\}', obj_src
    ):
        key = em.group(1)
        url = resolve_ref(em.group(2).strip(), aliases, route_map)
        controls = []
        if em.group(3):
            raw = em.group(3).strip()
            vars_ = re.findall(r'([A-Za-z_$][\w$]*)', raw) if raw.startswith('[') else [raw]
            controls = [aliases.get(v, v) for v in vars_]
        entries[key] = {'code': key, 'url': url, 'control': controls}
    return obj_src, entries


# [한국어 흐름] 명시적 부모 예외, 루트 여부, 접두어 규칙 순서로 부모를 결정합니다. 서버에서 검증한 계층이 아니라 추정 규칙입니다.
def parent_of(code, roots, prefix_parent, extra_parent):
    if code in extra_parent:
        return extra_parent[code]
    if code in roots:
        return None
    for pref, par in prefix_parent.items():
        if code.startswith(pref):
            return par
    return None


# [한국어 흐름] 코드별 자식 목록을 만들고 루트와 고아 코드를 메뉴 트리로 바꿉니다. 원본 기본 규칙에 맞는 입력을 전제로 합니다.
def build_tree(entries, roots, prefix_parent, extra_parent):
    children_of = {k: [] for k in entries}
    for code in entries:
        p = parent_of(code, roots, prefix_parent, extra_parent)
        if p:
            children_of.setdefault(p, []).append(code)

    # [한국어 흐름] 하나의 코드를 position/name/children 구조로 재귀 변환합니다. 노드 이름은 코드 자체를 사용합니다.
    def make_node(code):
        node = {
            'code': code,
            'position': 'top' if code in roots else 'left',
            'name': code,
        }
        kids = sorted(children_of.get(code, []))
        if kids:
            node['children'] = [make_node(c) for c in kids]
        return node

    tree = [make_node(r) for r in roots if r in entries or children_of.get(r)]
    orphans = [c for c in entries if parent_of(c, roots, prefix_parent, extra_parent) is None and c not in roots]
    for code in sorted(orphans):
        tree.append(make_node(code))
    return tree


# [한국어 흐름] 트리를 순회해 권한 코드를 평평한 목록으로 모읍니다. 역할별 모의 권한 응답의 입력입니다.
def flat_codes(nodes):
    out = []
    for n in nodes:
        out.append(n['code'])
        out.extend(flat_codes(n.get('children', [])))
    return out


# [한국어 흐름] 라우트 사전→권한 모듈→트리/권한 목록 파일 순서로 생성하고, 설정 파일이 있으면 관련 stub을 교체합니다.
def main():
    ap = argparse.ArgumentParser(description='Build permission tree stubs from JS auth modules')
    ap.add_argument('jsdir')
    ap.add_argument('outdir')
    ap.add_argument('--config', help='patch stubs into config.json')
    ap.add_argument('--role', default='SUPER_ADMIN', help='role_type in role_permissions stub')
    args = ap.parse_args()
    os.makedirs(args.outdir, exist_ok=True)

    route_map_path = os.path.join(args.outdir, 'route_map.json')
    if not os.path.exists(route_map_path):
        print('[*] route_map.json missing — run extract_route_map.py first')
        route_map = {}
    else:
        route_map = json.load(open(route_map_path, encoding='utf-8'))

    auth_fp = find_auth_file(args.jsdir)
    if not auth_fp:
        print('[!] userRouteAuth module not found in js/')
        sys.exit(1)
    print(f'[*] auth module: {os.path.basename(auth_fp)}')
    txt = open(auth_fp, encoding='utf-8', errors='ignore').read()
    _, entries = parse_user_route_auth(txt, route_map)
    if not entries:
        print('[!] failed to parse userRouteAuth object — adjust regex in script')
        sys.exit(1)

    tree = build_tree(entries, DEFAULT_ROOTS, DEFAULT_PREFIX_PARENT, DEFAULT_EXTRA_PARENT)
    all_codes = sorted(set(flat_codes(tree) + [c for e in entries.values() for c in e.get('control', [])]))

    perm_all = {'response_code': 0, 'verbose_msg': 'ok', 'data': tree}
    role_perm = {
        'response_code': 0,
        'verbose_msg': 'ok',
        'data': {'role_type': args.role, 'permissions': all_codes},
    }

    json.dump(entries, open(os.path.join(args.outdir, 'userRouteAuth.json'), 'w', encoding='utf-8'), ensure_ascii=False, indent=2)
    json.dump(tree, open(os.path.join(args.outdir, 'permissions_tree.json'), 'w', encoding='utf-8'), ensure_ascii=False, indent=2)
    open(os.path.join(args.outdir, 'perm_codes_all.txt'), 'w', encoding='utf-8').write('\n'.join(all_codes))
    json.dump(perm_all, open(os.path.join(args.outdir, 'permissions_all_stub.json'), 'w', encoding='utf-8'), ensure_ascii=False, indent=2)
    json.dump(role_perm, open(os.path.join(args.outdir, 'role_permissions_stub.json'), 'w', encoding='utf-8'), ensure_ascii=False, indent=2)

    print(f'[+] entries={len(entries)} codes={len(all_codes)} tree_roots={len(tree)}')

    cfg_path = args.config or os.path.join(args.outdir, 'config.json')
    if os.path.exists(cfg_path):
        cfg = json.load(open(cfg_path, encoding='utf-8'))
        stubs = [s for s in cfg.get('stubs', []) if not re.search(r'permissions/all|role_permissions', s.get('match', ''))]
        stubs = [
            {'match': 'permissions/all', 'body': perm_all},
            {'match': 'role_permissions', 'body': role_perm},
        ] + stubs
        cfg['stubs'] = stubs
        json.dump(cfg, open(cfg_path, 'w', encoding='utf-8'), ensure_ascii=False, indent=2)
        print(f'[+] patched stubs -> {cfg_path}')


if __name__ == '__main__':
    main()
