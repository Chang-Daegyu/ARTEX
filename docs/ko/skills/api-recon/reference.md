> **사람이 읽기 위한 한국어 번역·해설입니다. ARTEX가 이 문서를 Skill로 자동 로드하지 않습니다.** 원문 [skills/api-recon/reference.md](../../../../skills/api-recon/reference.md)의 모든 절 A–J와 스크립트 표, 설정, 예시를 설명합니다. 실제 조사의 경계와 단계 순서는 [한국어 주 안내](SKILL.md)를 먼저 읽습니다. 아래 명령·설정은 설명을 위해 원문을 보존했으며 실행하지 않았습니다.

# api-recon 참고 설명서

이 문서는 grep 레시피, `config.json` 템플릿과 문제 해결을 보충한다. 원문의 grep은 내려받은 `js/` 디렉터리에 적용한다. bundle이 한 줄이면 `js-beautify` 또는 `sed 's/}/}\n/g'`로 보기 좋게 나눌 수 있지만, 원문은 보통 앞뒤 문맥을 제한한 raw grep으로 충분하다고 설명한다. 주 Skill의 큰 파일·출력량 제한과 OUTDIR 사본 규칙은 계속 적용된다.

## 스크립트별 조정 지점

`scripts/`의 모든 파일은 참고 템플릿이다. 대상 사이트에 맞는지 읽고 사본을 조정한 다음 사용한다.

| 파일 | 주로 확인할 설정 |
| --- | --- |
| `harvest_static.py` | endpoint 정규식, webpack/Vite manifest 해석, 마이크로 프런트엔드 publicPath, 재시도와 동시성 |
| `runtime_harvest.js` | neutralize 필드/성공값, stub 일치 규칙/body, routes 출처, WS 기록, `waitUntil`, `routeTimeout`, `proxy` |
| `preload.js` | `loginPathRe`, L1 stub, `neutralize.fields`, `apiPattern`, L3 여부, `recordDetail`, `observe.*`, `neutralizeVueRouter` |
| `spider_mpa.py` | 파괴적인 링크를 거르는 `--exclude`, cookie, depth/max, 같은 도메인 조건 |
| `extract_route_map.py` | `routeMap`, `routeLink` 문법과 KEY 패턴 |
| `build_perm_tree.py` | `userRouteAuth` 해석, ROOTS/PREFIX_PARENT 계층 규칙, stub의 바깥 필드 이름 |
| `config.json` | 위 대상별 설정을 모으는 입력 |

## A. 프런트 화면을 결정하는 세 조건

### A1. 렌더링 조건: 로그인 상태를 어떻게 판단하는가

다음 레시피는 로그인 상태 함수, 저장소 읽기, 디코딩 흔적을 찾는다.

```bash
grep -rhoaE '.{0,40}(isLogin|isAuthenticated|loggedIn|hasLogin|requireAuth)\b.{0,80}' js | head
grep -rhoaE 'function (getUser|getToken|getAuth)[0-9]?\([^)]*\)\{.{0,200}' js | head
grep -rhoaE '(localStorage|sessionStorage)\.getItem\("[^"]+"\)' js | sort -u
grep -rhoaE '(Cookies?|cookie)\.(get|load)\("[^"]+"\)' js | sort -u
grep -rhoaE '\batob\(|JSON\.parse\(|jwt|decode' js | head
```

`isLogin = f(getUser())`에서 `getUser = decode(storage.read(KEY))`로 이어지는 호출을 찾아 **저장 키**, **Cookie와 localStorage 중 어느 컨테이너인지**, **인코딩 방식**을 확인한다.

| 프런트가 기대하는 표현 | 원문의 config 구성 예시 |
| --- | --- |
| 일반 문자열, `"1"`, token 문자열 | `"value": "anything-truthy"` |
| `JSON.parse(x)` | `"value": "json:{\"id\":1,\"username\":\"admin\"}"` |
| `JSON.parse(atob(x))` | `"value": "b64json:{\"id\":1,\"username\":\"admin\"}"` |
| JWT 모양 데이터 | 원문은 서명 없는/`alg:none` 형태 또는 bundle에 있는 키를 언급함 |
| SM2/AES/RSA 등으로 감싼 값 | 원문은 코드에 남은 키와 디코딩 조건을 확인하고 프런트 표시용 blob으로 충분할 때만 구성하며, 그렇지 않으면 정적 분석 결과를 남김 |

이 표의 목적은 **클라이언트가 읽을 수 있는 mock 상태**를 구성하는 것이다. 서버의 서명 검증이나 인증을 통과한 자격증명을 얻었다는 의미가 아니며, 주 Skill의 실제 로그인/자격증명 추측 금지를 바꾸지 않는다. 확인한 값은 `cookies` 또는 `localStorage` 설정에 기록한다.

### A2. 응답 인터셉터 조건: 무엇이 `/login` 이동을 유발하는가

```bash
grep -rhoaE '.{0,60}(interceptors\.response|axios|request\.use).{0,120}' js | head
grep -rhoaE '.{0,40}(response_code|errcode|errno|\bcode\b|\bret\b|\bstatus\b)\s*[=!]==?\s*[\-0-9]{1,4}.{0,60}' js | head -20
grep -rhoaE '.{0,40}(未登录|请重新登录|登录已过期|unauthorized|登录失效|授权|token.{0,10}invalid).{0,40}' js | head
grep -rhoaE '.{0,30}(location\.href|router\.(push|replace)|navigate)\([^)]*login[^)]*\)' js | head
```

응답에서 어떤 필드를 읽는지, 성공값은 무엇인지, 어떤 실패값에서 이동하는지 확인한다. 원문은 성공값의 흔한 예로 0 또는 200을 제시한다. grep의 중국어 문자열은 “로그인 안 됨”, “다시 로그인”, “로그인 만료” 같은 원본 오류 문구를 찾기 위한 문자열이다.

원문에는 임의 session 값으로 보호 API의 오류 응답 형식을 관찰하는 다음 예시가 있다. `<fakekey>`, `<protected>`, `target`은 자리표시자다. 이 예시가 서버의 인증을 성공시키는 방법을 의미하지는 않는다.

```bash
curl -sk -X POST -H 'Cookie: <fakekey>=junk' https://target/api/<protected> -d '{}' -H 'Content-Type: application/json'
```

관찰한 필드와 성공값을 `neutralize.fields`, `neutralize.success`에 넣는다. 이 설정이 바꾸는 것은 후킹한 브라우저에 전달되는 응답 해석이다.

### A3. 콘텐츠 조건: 메뉴와 권한은 어디서 오는가

```bash
grep -rhoaE '"/api[^"]*(permission|perm|role|menu|acl|resource|nav)[^"]*"' js | sort -u
grep -rhoaE '.{0,30}(menus|permissions|menuList|routeList|authList|role_permissions)\b.{0,120}' js | head
grep -rhoaE 'userRouteAuth|getResultTree|routeMap|routeLink|hasPermission|checkAuth' js | head
grep -rhoaE '([A-Z_][A-Z0-9_]*):\{name:"[^"]*",link:"/[^"]+"\}' js | head
```

기업용 관리 화면에서 흔히 다음 자료를 함께 소비한다는 설명이다.

| 위치 | 전형적인 payload | 소비하는 프런트 코드 |
| --- | --- | --- |
| `.../role_permissions` | `{ permissions: string[], role_type }` | 라우트 가드, 버튼별 ACL |
| `.../permissions/all` | `tree[{ code, position, children }]` | 사이드바 메뉴 렌더링 |
| bundle의 `userRouteAuth` | `{ CODE: { url, name? } }` | 권한 code에서 실제 경로로 변환 |
| bundle의 `routeMap` | `{ KEY: { name, link } }` | webpack의 `o.DASHBOARD` 같은 별칭 해석 |

`getResultTree(tree, permissions)`가 어떤 필드를 비교해 거르는지, `v-if` 또는 `hasAuth(code)`가 무엇을 확인하는지 읽는다. 작은 사이트에서는 표시 조건에 맞는 mock payload를 직접 구성할 수 있고, 단순 메뉴 모양으로 모듈이 열리지 않는 대형 사이트는 I절의 트리 복원이 필요할 수 있다.

## B. `config.json` 템플릿

다음은 원문 전체 템플릿이다. `faketoken`, `anything-truthy` 계열 값과 example route는 실제 인증 정보가 아니라 프런트 렌더링을 관찰하는 mock 예시다.

```json
{
  "baseUrl": "https://target/",
  "runtimeMode": "both",
  "chromium": "/usr/bin/chromium",

  "cookies": [
    { "name": "auth", "value": "b64json:{\"id\":1,\"username\":\"admin\",\"role\":\"admin\",\"func\":{},\"permissions\":[\"*\"]}" }
  ],
  "localStorage": { "token": "faketoken", "isLogin": "1" },

  "neutralize": {
    "fields": ["response_code", "code", "errno", "ret", "status"],
    "success": 0,
    "flags": { "success": true, "message": "ok" }
  },
  "forward": true,
  "loginUrlPattern": "/login",
  "apiPattern": "/api/|/rest/|/graphql",

  "mockTier": "L1+L2",
  "recordDetail": true,
  "observe": {
    "storageReads": false,
    "cookieReads": false,
    "xhrHeaders": true
  },
  "neutralizeVueRouter": true,
  "stubs": [
    {
      "match": "permissions/all|/menu|role_permissions",
      "body": {
        "response_code": 0, "code": 0,
        "data": {
          "permissions": ["*"],
          "menus": [
            { "name": "dashboard", "path": "/dashboard", "show": true, "children": [] },
            { "name": "alert", "path": "/alert", "show": true, "children": [] }
          ]
        }
      }
    }
  ],

  "explore": {
    "clickTabs": true,
    "clickTables": true,
    "pushStateFallback": true,
    "maxMenuItems": 50
  },

  "routes": ["/dashboard", "/alert", "/asset", "/device", "/report", "/config", "/system"],
  "waitMs": 1500, "perRouteMs": 900, "headless": true,
  "waitUntil": "domcontentloaded",
  "routeTimeout": 12000,
  "proxy": "",

  "captureResponses": true, "recordWs": true, "respMax": 600
}
```

| 필드 | 한국어 설명 |
| --- | --- |
| `baseUrl` | 대상 앱의 기준 주소 |
| `runtimeMode` | `depth`는 Puppeteer, `coverage`는 브라우저 조작, `both`는 두 단계 |
| `chromium` | 사용할 Chromium 실행 파일 경로 |
| `cookies[].value` | `b64json:`은 JSON의 base64 인코딩, `json:`은 JSON 문자열, 접두가 없으면 원문 문자열 |
| `localStorage` | 프런트가 읽는 저장 키와 mock 값 |
| `neutralize.fields` | 응답에서 성공/실패 코드를 판별하는 필드 목록 |
| `neutralize.success`, `flags` | 프런트에 전달할 성공 코드와 관련 플래그 |
| `forward` | 참이면 실제 요청을 전달하고 응답 코드를 수정할 수 있음; 거짓이면 요청을 오프라인 stub으로 처리하는 설명 |
| `loginUrlPattern` | 로그인 경로를 식별하는 패턴 |
| `apiPattern` | 기록/처리할 API 경로를 식별하는 패턴 |
| `mockTier` | coverage preload에서 사용할 L1/L2/L3 조합 |
| `recordDetail` | 상세 요청 기록 활성화 |
| `observe.*` | 저장소·Cookie 읽기와 XHR 헤더 관찰 여부 |
| `neutralizeVueRouter` | Vue 라우트 가드와 경로 관찰에 사용하는 기능 |
| `stubs` | URL 일치 패턴과 그때 반환할 body |
| `explore` | 탭/표 클릭, pushState 대안, 메뉴 개수 등 탐색 설정 예시 |
| `routes` | 정적 routes 파일의 경로; 메뉴 mock 뒤 발견한 `<a href>`도 보완 가능 |
| `waitMs`, `perRouteMs` | 첫 진입과 경로별 관찰 대기 시간 설정 |
| `waitUntil` | 대형 SPA는 `domcontentloaded`를 써 지속 통신으로 인한 `networkidle2` 대기를 피하는 설명 |
| `routeTimeout` | 경로별 `page.goto` 시간 제한, 밀리초 |
| `proxy` | Puppeteer `--proxy-server`; 원문은 `HTTP_PROXY`/`HTTPS_PROXY`도 언급 |
| `captureResponses`, `recordWs` | 원문상 depth에서만 적용되는 응답/WS 기록 |
| `respMax` | 응답 미리보기 길이를 제한하는 설정 예시 |

이 표는 원문 템플릿의 의미를 설명한다. 실제 지원 여부와 기본값은 실행할 **사본 스크립트가 어느 필드를 읽는지**로 확인한다. 특히 `forward: true`는 “모든 요청이 외부로 나가지 않는 mock”과 같은 뜻이 아니다.

### B1. 권한 목록과 권한 트리의 두 stub

`role_permissions`에는 평탄한 권한 코드, `permissions/all`에는 계층 구조를 넣는 원문 예시다.

```json
"stubs": [
  {
    "match": "role_permissions",
    "body": {
      "response_code": 0,
      "data": {
        "permissions": ["MONITOR", "MONITOR_ALERT", "THREAT", "ASSETS_RISK"],
        "role_type": "SUPER_ADMIN"
      }
    }
  },
  {
    "match": "permissions/all",
    "body": {
      "response_code": 0,
      "data": [
        {
          "code": "MONITOR",
          "position": 1,
          "children": [
            { "code": "MONITOR_ALERT", "position": 1, "children": [] }
          ]
        }
      ]
    }
  }
]
```

바깥 `response_code`/`code`/`data` 구조는 A2에서 확인한 소비 코드와 맞아야 한다. permissions 목록이 트리의 모든 leaf code를 포함하는지도 확인한다. `SUPER_ADMIN`은 mock 응답의 예시 필드값이며 서버 권한을 실제로 부여한 증거가 아니다.

## C. coverage의 preload 설정

원문은 `scripts/preload.js` 상단의 CONFIG를 편집하거나 CDP 주입 전 교체한다고 설명한다. 주 Skill의 사본 규칙에 따라 **OUTDIR의 preload 사본**에 적용할 설정으로 읽는다.

```javascript
const CONFIG = {
  loginPathRe: /\/(login|signin)(\/|$|\?)/i,
  mockTier: 'L1+L2',
  forward: true,
  recordDetail: true,
  extractUrlsFromResponse: true,
  neutralizeVueRouter: true,
  observe: { storageReads: false, cookieReads: false, xhrHeaders: true },
  neutralize: { fields: ['response_code', 'code'], success: 0 },
  stubs: [ /* 同 config.json stubs */ ],
  apiPattern: /\/(api|apis|v\d+|dev|internal|graphql)\//i,
};
```

주석 `同 config.json stubs`는 “config.json의 stubs와 동일한 배열”이라는 뜻이다. `window.__API_RECON_PRELOAD__ === true`인지, 경로가 로그인 화면으로 다시 바뀌지 않는지 확인한다.

기록을 내보내는 원문 예시는 다음과 같다.

```javascript
JSON.stringify({
  apis: [...window.__API_RECON_LOG__],
  detail: window.__API_RECON_DETAIL__,
  routes: [...(window.__API_RECON_ROUTES__ || [])],
  observe: window.__API_RECON_OBSERVE__,
}, null, 2)
```

`apis`는 발견 URL/메서드 기록, `detail`은 상세, `routes`는 관찰된 프런트 경로, `observe`는 헤더/저장소 읽기 등의 보충 기록이다. 원문의 출력 객체는 브라우저에서 만들어진 관찰 결과이며 실제 서버의 전체 API 명세와 동일하다고 단정하지 않는다.

## D. preload와 runtime Hook의 지원 범위

| 기능 | 얻을 수 있는 정보 | 원문의 지원 설명 |
| --- | --- | --- |
| fetch / XHR.open Hook | URL와 메서드 | `recordDetail`, `__API_RECON_LOG__` |
| XHR.setRequestHeader Hook | Authorization 등 요청 헤더 | `observe.xhrHeaders` |
| localStorage/Cookie 읽기 Hook | 세션용 저장 키 | 선택적 `observe.storageReads/cookieReads` |
| Vue 라우트 읽기 | 이미 로드된 프런트 경로 | `__API_RECON_ROUTES__` |
| Vue 가드/로그인 이동 제어 | 모듈 렌더링과 요청 발생 보조 | `neutralizeVueRouter`와 브라우저 이동 처리 |
| React 라우트 읽기 | 라우트 목록 보완 | 전용 Hook이 없어 정적 분석과 클릭 조합 |
| 로그인 경로 이동 차단 | 화면을 유지하며 관찰 | 업무 이동 전체가 아닌 로그인 경로만 대상으로 함 |
| CryptoJS/SM 등 암호화 함수 | 암호화 전 payload | 내장되지 않음; 필요한 입력 관찰은 별도 대상별 분석 |
| 안티디버깅 처리 | 런타임 관찰 가능성 | 내장되지 않음; 해결하지 못하면 정적 결과를 남길 수 있음 |

지원 표시가 있다는 것과 모든 프레임워크·버전에 완전히 적용된다는 주장은 다르다. 대상의 소비 코드와 실제 기록으로 확인하는 절차가 필요하다.

## E. 정적 endpoint 추출이 부족할 때

`harvest_static.py` 사본의 `extract_endpoints` 정규식을 넓힌다. 원문은 다음 보충 패턴도 제시한다. 적용 시 주 Skill의 출력량 제한을 함께 사용한다.

```bash
grep -rhoaE '"/[a-z][A-Za-z0-9_/\-]{3,}"' js | sort -u
grep -rhoaE '/api/[a-zA-Z0-9_./-]+' js | sort -u
```

넓은 경로 패턴은 API 외의 문자열도 잡을 수 있으므로 결과가 늘었다는 사실만으로 정확도가 높아졌다고 보지 않는다. path의 실제 callsite와 런타임 기록으로 보완한다.

## F. 문제 해결 표

| 증상 | 원문의 원인 해석과 대응 |
| --- | --- |
| 정적 API가 적음 | endpoint 문법 불일치 → 정규식 확장; 정규식 설명은 E절 |
| chunk 수가 manifest보다 훨씬 적음 | CSS 전용 또는 미배포 chunk 여부, 404 재시도 확인 |
| 런타임도 로그인 화면 | A1의 키 이름·컨테이너·인코딩·domain 재확인 |
| UI 틀에는 들어갔지만 모듈이 비어 있음 | A3 메뉴 자료와 routes의 실제 path 확인 |
| 모든 경로에서 bootstrap/locale만 발생 | I절 권한 트리와 두 권한 stub 확인 |
| 사이드바는 있지만 하위 페이지가 비어 있음 | 중간 트리 노드 누락, code와 userRouteAuth 불일치 |
| 모든 API 뒤 로그인으로 이동 | neutralize 필드 확인; 중첩 필드는 walk 로직 조정 필요 |
| WS 프레임이 0 | 사용자 상호작용 후 구독하는지 확인하고 perRouteMs 조정 |
| 응답 본문이 비어 있음 | 실제 응답은 forward를 통해 얻는지, stub 결과인지 구분 |
| Chromium 없음 | 설치 경로, `config.chromium`, `CHROMIUM` 확인 |
| mock을 많이 넣어도 로그인으로 이동 | Hook이 늦거나 이동 처리 지점이 누락됨 → document-start 확인 |
| 모든 목록이 비어 있음 | L3의 빈 배열은 예상 결과; 탭/설정/상세의 요청 계속 관찰 |
| Redux action을 라우트로 오인 | get/set/change/clear/toggle/upload가 섞인 내부 path 후보 구분 |
| Vue가 계속 로그인으로 이동 | preload 주입 시점과 neutralizeVueRouter 설정 확인 |
| 응답에 URL이 있는데 log에는 없음 | extractUrlsFromResponse 또는 상세 기록에서 확인 |
| Authorization 헤더 이름을 모름 | observe.xhrHeaders 또는 DevTools 요청 헤더 관찰 |
| 런타임이 매우 느리거나 시간 초과 | domcontentloaded와 routeTimeout 조정; 지속 통신 시 networkidle2 회피 |
| 프록시 연결 실패 | proxy/환경변수와 Puppeteer·curl의 프록시 포트 일치 확인 |

원문 표는 endpoint 정규식 설명을 D절로 가리키지만 실제 정규식 절은 E이므로 이 번역의 연결을 맞췄다. 기능이나 스크립트를 고친 것은 아니다.

## G. 서버 검증이 강한 대상

원문은 위조할 수 없는 서명 쿠키, 서버에서만 만드는 메뉴처럼 클라이언트 mock으로 재현할 수 없는 조건이 있으면 런타임이 UI 틀에서 더 진행되지 않을 수 있다고 설명한다. 이 경우 코드에 남은 모듈 path를 정적 endpoint 결과로 남길 수 있다.

이 절의 원문에는 “허가된다면 실제 세션으로 같은 harness를 사용하고 `forward: true`로 실제 메서드/매개변수/응답을 기록한다”는 추가 예시가 있다. 그러나 주 `SKILL.md`는 실제 로그인·자격증명 요구와 실제 세션 의존을 금지한다. **참고 문서에 서로 다른 범위의 설명이 남아 있다는 사실을 구분해야 한다.** 이 번역은 새 자격증명을 얻거나 로그인하라는 지시를 추가하지 않으며, 실제 작업에서는 주 Skill의 명시적 범위와 사용자 권한 조건을 먼저 따른다.

## H. 한 번의 작업 체크리스트

1. 허가된 범위를 확인한다.
2. harvest 스크립트를 읽고 사본을 조정해 정적 API와 routes를 확인한다.
3. Phase 1b의 path 주변과 UI 바인딩에서 `param_candidates.json`을 만든다.
4. A1/A2/A3 결과로 사이트 전용 config를 작성한다.
5. runtime/preload 사본을 읽고 조정한다.
6. depth는 필요한 의존성을 설치한 뒤 조정한 runtime 스크립트를 사용한다.
7. coverage/both는 document-start preload와 UI 열거, 매개변수 발생 행렬을 수행한다.
8. 모듈이 렌더링되지 않으면 I절로 stub을 보완하고 반복한다.
9. 여러 표본의 차이와 허용 범위의 오류 근거로 `params_merged.json`을 만든다.
10. `site_map.json`, `api_merged.txt`로 합치고 미확인 범위와 스크립트 수정점을 명시한다.

## I. 권한 트리 복원: Phase 4의 상세

단순히 `menus: [{ path, show: true }]`를 주어도 하위 모듈이 마운트되지 않을 때 확인하는 절이다.

### I1. auth 모듈 찾기

```bash
grep -l 'userRouteAuth' js/*.js
grep -l 'routeMap\|routeLink' js/*.js
grep -rhoaE 'getResultTree|role_permissions|permissions/all' js | head
```

권한 API path, 응답 필드 이름, 이를 소비하는 chunk 파일을 기록한다.

### I2. routeMap 추출

원문 명령은 `scripts/` 경로를 쓰지만 실제 워크플로에서는 주 Skill이 정한 OUTDIR 사본을 조정하여 사용한다. 원문 코드 아래의 중국어 주석은 결과가 `recon/route_map.json`이라는 뜻이다.

```bash
python3 scripts/extract_route_map.py recon/js recon/
# 产出 recon/route_map.json
```

`[!] no routeMap pattern found`이면 사본의 패턴을 대상 문법에 맞추거나 다음 구조를 확인한다.

```bash
grep -rhoaE '([A-Z_][A-Z0-9_]*):\{name:"[^"]*",link:"/[^"]+"\}' js | head -20
```

### I3. 권한 트리와 stub 구성

```bash
python3 scripts/build_perm_tree.py recon/js recon/ --config recon/config.json
```

원문이 설명하는 알고리즘은 다음과 같다.

1. `userRouteAuth={MONITOR:{url:...},...}`를 읽는다. webpack 별칭 `He=o.DASHBOARD`도 포함한다.
2. route_map으로 별칭을 실제 path에 대응시킨다.
3. `MONITOR_ALERT` → `MONITOR`처럼 코드 접두를 이용해 부모 후보를 추정한다.
4. `permissions_tree.json`, `permissions_all_stub.json`, `role_permissions_stub.json`을 만든다.
5. `--config`가 있으면 config의 stubs와 routes도 보완한다.

이 부모 추정은 사이트별 관례에 대한 휴리스틱이다. `DEFAULT_ROOTS`, `DEFAULT_PREFIX_PARENT`, `DEFAULT_EXTRA_PARENT`를 통해 최상위 코드, 접두→부모, 접두 규칙에 맞지 않는 고아 노드를 조정한다.

### I4. stub의 일관성 확인

아래 원문 주석의 의미는 “permissions 수가 userRouteAuth 항목 수와 비슷한가”, “routes에 route_map의 모든 link가 있는가”이다.

```bash
# permissions 数量应 ≈ userRouteAuth 条目数
wc -l recon/perm_codes_all.txt
# routes 应覆盖 route_map 全部 link
python3 -c "import json; m=json.load(open('recon/route_map.json')); r=set(json.load(open('recon/config.json'))['routes']); print('missing', [v['link'] for v in m.values() if v['link'] not in r])"
```

수만 같다고 관계가 맞는 것은 아니므로 code와 tree의 실제 대응, 중간 부모 노드, 바깥 응답 필드도 함께 확인한다.

### I5. 런타임을 다시 실행해 비교

```bash
node recon/runtime_harvest.js recon/config.json
# 对比 forge 前后 runtime_api.json 条数；检查 /attack、/asset 等是否出现模块 API
```

주석은 수정 전후 `runtime_api.json`의 수와 `/attack`, `/asset` 등 모듈 API 발생 여부를 비교하라는 뜻이다. 다음 수치는 **원문이 든 예시**이며 모든 사이트가 이 수치에 도달해야 한다는 완료 기준은 아니다.

| 보완 전 예시 | 보완이 적용된 뒤의 예시 |
| --- | --- |
| 모든 경로에서 같은 bootstrap 3–5개 | 경로별로 서로 다른 모듈 API |
| `/api/locale/language`만 보임 | `/api/web/...` 같은 업무 endpoint 발생 |
| routes가 한 자리 수 | route_map 기반 80–110개 이상의 경로가 있는 사례 |

### I6. 여전히 안 되는 경우

- coverage에서 사이드바와 탭을 확인한다. 권한 판단이 상호작용 뒤 이루어질 수 있다.
- stub의 중첩 구조가 소비 코드가 기대하는 모양인지 확인한다. 원문은 실제 세션 응답 비교도 언급하지만, 그 내용은 G절에서 설명한 주 Skill과의 범위 차이를 고려해야 한다.
- `hasPermission`, `checkRole`, `func.` 같은 버튼별 조건을 확인하고 mock의 권한 구조를 보완한다.
- 해결하지 못하면 정적 API, 매개변수 후보, 이미 기록한 표본을 보존하고 런타임 미확인 범위를 적는다.

## J. 매개변수 역추적: Phase 1b / 5b / 5c

범용 추출기 하나가 아닌 방법론이다. path 정규식만으로 요청의 필드 구조를 알 수 없으므로 path 주변, UI 바인딩, 여러 표본, 검증 오류를 연결한다.

### J1. path를 기준점으로 요청 조립 객체 찾기

아래 주석은 “Phase 1에서 이미 알고 있는 path를 기준점으로 사용한다”는 뜻이다.

```bash
# 以 Phase 1 已知 path 为锚
grep -n '"/api/user/list"' js/*.js
grep -rhoaE '.{0,120}("/api[^"]+").{0,200}' js | head
grep -rhoaE '(params|data|body|payload)\s*:\s*\{' js | head
grep -rhoaE '(get|post|put|delete|patch)\([^,]+,\s*\{' js | head
```

주변의 params/data/body/payload 객체와 HTTP 메서드 호출을 찾아 입력이 어디서 만들어지는지 연결한다.

### J2. 래퍼와 전송 형태

다음 레시피는 순서대로 axios/공통 request, GraphQL, FormData/multipart, 경로 매개변수를 찾는다.

```bash
# axios / 统一 request
grep -rhoaE '(axios|request)\.(get|post|put|delete|patch)\(' js | head
grep -rhoaE 'interceptors\.(request|response)' js | head

# GraphQL
grep -rhoaE '(query|mutation)\s+\w+|gql`|graphql\(' js | head
grep -rhoaE '\$[a-zA-Z_]+\s*:\s*(Int|String|Boolean|\[)' js | head

# FormData / multipart
grep -rhoaE 'FormData|\.append\(' js | head

# 路径参数
grep -rhoaE 'path:\s*"/[^"]*:[^"]+"' js | head
grep -rhoaE 'useParams|route\.params|\$route\.params' js | head
```

GraphQL은 operation 이름만으로 끝내지 않고 `variables` 선언과 실제 JSON 값을 확인한다. multipart는 `.append`의 키와 값 출처를 본다. 경로 값은 일반 body 필드와 분리한다.

### J3. 필수·형식·enum 검증

```bash
grep -rhoaE '(required|message|pattern|enum|validator)\s*:' js | head
grep -rhoaE 'yup\.|zod\.|async-validator|Form\.Item|a-form-item|el-form-item' js | head
grep -rhoaE 'rules\s*:\s*\[|name:\s*["\'][a-zA-Z_]+["\']' js | head
grep -rhoaE 'label.*value|options\s*:\s*\[' js | head
```

프런트 validation은 필드 후보와 조건을 알려 주지만 서버가 동일하게 검증하는지까지 자동으로 확정하지 않는다. 필수 여부의 근거가 UI 규칙인지, 실제 요청 표본인지, 서버 오류인지 구분해 기록한다.

### J4. 폼에서 API까지의 바인딩

```bash
grep -rhoaE 'onFinish|handleSubmit|getFieldsValue|validateFields' js | head
grep -rhoaE '(pick|omit|transform|dayjs|moment)\(' js | head
```

`onFinish`, `handleSubmit`, `getFieldsValue`, `validateFields` 이후의 `pick`, `omit`, `transform`, 날짜 변환을 따라간다. 동적 분석에서는 Network의 Initiator/호출 스택에서 fetch/send보다 상위의 조립 함수를 찾는다. 화면의 field name과 서버에 전송되는 key가 같다는 보장은 없기 때문이다.

### J5. 암호화된 매개변수

```bash
grep -rhoaE 'encrypt|decrypt|sign|CryptoJS|sm2|sm3|sm4|RSA|AES' js | head
```

원문은 암호문에서 필드 이름을 추측하지 말고, 허가된 클라이언트 코드 분석에서 암호화 함수 **입력 전** payload를 관찰하라고 설명한다. 이 Hook은 자동 지원 기능이 아니므로 대상별 분석이 필요하다. 확인한 결론은 config와 param_candidates에 기록하고 관찰하지 못한 부분은 남겨 둔다.

### J6. 매개변수 발생 행렬

| UI 동작 | 비교할 내용 |
| --- | --- |
| 목록 첫 화면 | 기본 페이지 값 |
| 검색 | keyword, filters |
| 고급 필터 | 선택적으로 붙는 필드 |
| 생성/편집 UI | 엔터티의 전체 구조 |
| 일괄 선택/내보내기 UI | `ids[]`, `exportType` |
| 정렬/페이지 이동 | `sortField`, `order` |

주 Skill이 금지한 삭제·민감 데이터 반출·대량 쓰기를 실제 수행한다는 뜻은 아니다. mock 환경의 UI에서 어떤 outbound 요청이 만들어지는지 기록하는 경계를 유지한다.

표본 구조는 `param_samples.json`의 `[{ "path", "method", "action": "search", "body", "query", "headers" }]`로 설명되어 있다. 이것은 필드 목록의 개념적 표기이며 그 자체가 값까지 채워진 유효 JSON 예시는 아니다.

### J7. 신뢰도 규칙

| 신뢰도 | 근거 |
| --- | --- |
| 높음 | 정적 callsite와 둘 이상의 런타임 표본이 일치 |
| 중간 | 정적 근거만 있거나 런타임 표본이 하나 |
| 낮음 | 응답/오류에서 추론했고 추가 확인이 없음 |
| 발생 미확인 | 정적 필드는 알려졌지만 UI/권한 경로를 아직 실행하지 못함 |

이 등급은 원문의 조사용 분류다. 필드가 보였다는 신뢰도와 취약점이 존재한다는 신뢰도는 다른 주장이다.

### J8. 상황별 접근 순서

| 상황 | 원문의 권장 순서 |
| --- | --- |
| REST 목록 | J1 조립 객체 → J6의 여러 동작 비교 → J3 규칙 |
| 생성/편집 폼 | J3 Form name → J4 제출 연결 → 런타임 표본과 허용 범위의 빈 값 오류 |
| GraphQL | J2 변수 선언 → operation별 variables 표본 |
| 암호화 body | J5 입력 관찰 → 암호화 전 필드 기록 |

### J9. 전체 단계와의 대응

| api-recon 단계 | 매개변수 조사 |
| --- | --- |
| Phase 1 | J1의 path 주변 분석 |
| Phase 2 A2 | tenantId, sign 등 전역 주입 필드 |
| Phase 3 | J6 행렬과 param_samples |
| Phase 4 | 모듈별 폼이 실제로 나타나는지와 권한 자료 보완 |
| Phase 5 | params_merged와 신뢰도; 한 표본으로 필수 여부 단정 금지 |

### J10. 매개변수 조사 문제 해결

| 증상 | 대응 |
| --- | --- |
| 정적 필드가 런타임에서 한 번도 안 나타남 | 발생 미확인으로 표시; 권한 트리·고급 필터·연동 select 조건 확인 |
| 같은 path가 다른 body 구조를 사용 | action별로 따로 기록; 하나의 schema에 억지로 합치지 않음 |
| stub 응답은 가짜인데 params를 보고 싶음 | outbound body/headers를 기록; stub 응답에서 역추론하지 않음 |
| 400이 중첩 필드를 지적 | data, bizData, variables 등 바깥 포장 확인 |
| GraphQL operation 이름만 보임 | variables JSON과 정적 `$var: Type` 선언 확인 |

## 번역 범위

원문의 스크립트 표, A1–A3, B/B1, C–H, I1–I6, J1–J10을 모두 포함했다. 20개 명령/설정 코드 블록을 원문 그대로 보존하고 중국어 주석·자리표시자의 의미를 주변 한국어 문장으로 설명했다. 주 문서와 참고 문서의 경계 차이 및 잘못된 절 참조는 명시적으로 표시했으며 실행용 원문을 변경하지 않았다.
