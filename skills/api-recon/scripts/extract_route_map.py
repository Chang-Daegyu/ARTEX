#!/usr/bin/env python3
# [한국어 해설] 다운로드된 번들에서 화면 라우트 사전 추출
# 각 JS 파일에서 KEY/name/link 패턴을 찾고 가장 많은 일치 항목을 가진 파일을 route_map.json으로 저장합니다.
# 전체 파일 결과를 병합하지 않고 가장 큰 후보 하나를 택합니다. build_perm_tree.py의 번들 별칭 해석에 사용하는 보조 입력입니다.
"""
extract_route_map.py <JSDIR> <OUTDIR>

Scan downloaded JS chunks for routeMap / routeLink style objects:
  KEY:{name:"...",link:"/path"}

Writes route_map.json — used by build_perm_tree.py to resolve alias refs.
"""
import re, json, os, sys, glob

# [한국어 흐름] 입력 폴더의 JS들을 정규식으로 검사해 최대 일치 후보를 저장합니다. 일치가 없으면 성공처럼 빈 결과를 만들지 않고 실패로 종료합니다.
def main():
    if len(sys.argv) < 3:
        print("usage: extract_route_map.py <JSDIR> <OUTDIR>")
        sys.exit(1)
    jsdir, outdir = sys.argv[1], sys.argv[2]
    os.makedirs(outdir, exist_ok=True)

    best = {}
    best_file = None
    pat = re.compile(r'([A-Z_][A-Z0-9_]*):\{name:"([^"]*)",link:"([^"]+)"')

    for fp in glob.glob(os.path.join(jsdir, '*.js')):
        try:
            txt = open(fp, encoding='utf-8', errors='ignore').read()
        except Exception:
            continue
        hits = pat.findall(txt)
        if len(hits) > len(best):
            best = {k: {'name': n, 'link': l} for k, n, l in hits}
            best_file = fp

    if not best:
        print('[!] no routeMap pattern found — widen regex or grep manually')
        sys.exit(1)

    out = os.path.join(outdir, 'route_map.json')
    json.dump(best, open(out, 'w', encoding='utf-8'), ensure_ascii=False, indent=2)
    print(f'[+] {len(best)} routes from {os.path.basename(best_file)} -> {out}')

if __name__ == '__main__':
    main()
