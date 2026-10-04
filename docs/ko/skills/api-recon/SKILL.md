> **사람이 읽기 위한 한국어 번역·해설입니다. ARTEX가 이 파일을 Skill로 자동 로드하지 않습니다.** 실행용 원문은 [skills/api-recon/SKILL.md](../../../../skills/api-recon/SKILL.md)에 그대로 있습니다. 아래 명령은 원문의 절차를 설명하기 위한 예시이며 이 번역 작업에서 실행하지 않았습니다. 설정 예시와 보충 레시피는 [한국어 reference.md](reference.md)에서 이어집니다.

# API Recon: 프런트엔드 API 인터페이스 조사

원문의 Skill 이름은 `api-recon`이고 description은 “웹사이트 API 인터페이스를 수집할 때 이 Skill을 호출한다”는 뜻이다.

목적은 **허가된 범위**에서 백엔드 API의 경로·메서드·매개변수·응답, 프런트엔드 경로, 탭·대화상자·표의 행 조작과 같은 UI 기능의 요청 발생 지점을 가능한 범위까지 찾는 것이다. 원문은 이 단계를 취약점 공격과 구분하고, 실제 서버 권한을 획득하는 대신 프런트 코드를 분석하거나 모의 응답으로 화면을 렌더링해 **프런트가 어떤 요청을 만들려 하는지** 관찰한다.

## 경계와 금지 사항

이 Skill은 **API와 매개변수 표면을 조사하는 단계**다. 취약점 탐색·침투·악용 단계로 확대하지 않는 것이 원문의 전제다.

### 작업 범위

| 구분 | 원문이 허용하는 것 | 원문이 금지하는 것 |
| --- | --- | --- |
| 목표 | path, method, 매개변수, 라우트, UI 발생 지점 열거 | SQLi, XSS, 권한 우회, 무차별 대입, 취약점 fuzzing, 공격성 요청 변조, 파괴적 조작 |
| 인증 | Hook와 stub/mock으로 **클라이언트 화면의** 로그인 조건을 분석/대체 | 사용자에게 비밀번호를 요구하거나 추측하기, 실제 로그인 폼 제출 시도 |
| 런타임 | 자격증명 없이 인터페이스를 후킹하고 mock으로 SPA의 로그인 뒤 UI 틀을 렌더링 | 실제 서버 세션을 얻어야만 이어갈 수 있는 작업 흐름에 의존 |

여기서 “클라이언트 로그인 문턱”은 JavaScript가 화면을 표시할지 판단하는 조건이다. 이를 mock으로 통과시켜도 서버가 실제 인증이나 권한을 부여한 것은 아니다.

### 자격증명 없는 동적 분석: Phase 3의 기본값

1. `preload.js` / `runtime_harvest.js`로 로그인·권한·메뉴 등 초기 화면 API를 가로채 stub 응답을 제공한다.
2. 업무 조회 API에는 **구조가 맞고 업무 성공 코드를 가지며 내용은 비어 있어도 되는** mock body를 제공한다.
3. 서버가 없거나 401을 반환하는 상황에서도 SPA의 하위 경로와 컴포넌트가 렌더링되어 추가 XHR/fetch/WebSocket 요청을 만들게 한다.
4. 빈 데이터·빈 표·자리표시 UI는 예상되는 결과다. 데이터를 채우겠다는 이유로 실제 로그인이나 취약점 테스트로 전환하지 않는다.

따라서 관심 대상은 mock 응답의 진실성이 아니라 **프런트가 내보내는 요청의 URL·메서드·body·headers**다. 다만 그 요청 값 자체도 사용자가 넣은 입력과 mock으로 구성된 UI 상태에 영향을 받을 수 있으므로 실제 업무 데이터나 서버 검증 결과로 과장하지 않는다.

### 절차상 금지와 대안

| 하지 말아야 할 일 | 원문이 지정한 대안 |
| --- | --- |
| Phase 1 이전에 주 entry `index-*.js`를 grep/curl/Read로 읽어 API를 수작업 추출 | `OUTDIR/harvest_static.py` 실행 |
| `extract_apis.py` 같은 별도 임시 추출기로 harvest 대체 | OUTDIR의 harvest 사본을 수정하고 재실행 |
| 같은 grep/명령이 두 번 이상 실패했는데 그대로 반복 | tool_logs, harvest 수정, reference를 확인해 전략 변경 |
| A/B 준비 조건을 건너뛰고 원본 `scripts/` 그대로 실행 | OUTDIR로 복사해 대상별 설정 반영 |
| 실제 사용자명/비밀번호·OTP·OAuth 인증 시도 | stub/mock 사용 |
| 실제 데이터를 얻겠다고 권한 우회나 주입 시험 수행 | outbound 요청만 기록 |
| 삭제·민감 데이터 내보내기·대량 쓰기 등 되돌릴 수 없는 조작 | coverage 클릭에서도 같은 경계 준수 |
| 런타임과 동적 열거를 하지 않고 모든 페이지/API를 얻었다고 주장 | 완료 조건을 만족하거나 한계를 명시 |
| 매개변수 발생 행렬과 diff 없이 모든 매개변수를 안다고 주장 | Phase 3b 행렬과 Phase 5 비교 수행 |
| 런타임 표본 하나로 필수/선택 여부 단정 | 여러 표본 비교 또는 검증 규칙/오류 메시지 근거 사용 |

## 정적·런타임 두 층과 실행 모드

| 층 | 얻을 수 있는 것 | 한계 |
| --- | --- | --- |
| 정적 JS bundle | 코드에 보이는 endpoint 경로, 라우트 초안, 요청 조립 필드 후보 | 기본 경로 수집만으로 메서드와 매개변수 구조가 확정되지 않음; 런타임 URL 조합을 놓칠 수 있음 |
| 런타임 | 메서드·body·응답·동적 URL·WS/SSE, 여러 표본의 매개변수 차이 | 해당 화면이 실제로 렌더링되어야 요청이 발생함; 한 표본으로 필수/선택 확정 불가 |

| `runtimeMode` | 엔진 | 주 용도 |
| --- | --- | --- |
| `depth` | Puppeteer의 `runtime_harvest.js` | 반복 가능한 경로 순회와 상세 API/WS/SSE 기록 |
| `coverage` | browser와 `preload.js` | 탭·대화상자·표 조작 등 실제 UI 기능 발생 지점 확대 |
| `both` | depth 후 coverage | 두 결과를 합치는 대신 시간이 더 필요 |

매개변수 조사에는 완성된 범용 스크립트가 없다는 것이 원문의 설명이다. 경로는 harvest/정규식으로 찾고, 필드는 **알고 있는 path 주변 확장 → UI 바인딩 추적 → 여러 요청 표본의 차이 → 오류 메시지 역추론**으로 확인한다. 보충 설명은 [reference J](reference.md#j-매개변수-역추적-phase-1b--5b--5c)에 있다.

## 완료 조건

다음을 모두 충족해야 원문 기준으로 recon 완료라고 표현한다.

- [ ] 정적 단계에서 `api_static.txt`, `routes.txt`, `js/`를 얻었다.
- [ ] depth 또는 coverage 중 하나 이상을 실행했다. coverage/both라면 Hook 적용과 동적 열거를 확인했다.
- [ ] 업무 경로에 접근했을 때 `/login` 화면으로 돌아가지 않았다. hash 라우트도 고려했다.
- [ ] coverage/both에서 매개변수 발생 행렬과 `param_samples.json`을 만들었고 Phase 5의 `params_merged.json`을 작성했다.
- [ ] 모듈이 빈 화면이면 Phase 4 권한 트리 복원 후 다시 확인하여 locale/bootstrap뿐 아니라 모듈 API가 나타나는지 확인했다.
- [ ] Phase 5 결과물을 정리하고 `insert_assets`로 발견된 서비스와 endpoint 자산을 빠뜨리지 않고 등록했다.

이 체크리스트는 **Skill이 실행될 때 기대하는 결과**다. 문서를 번역했다는 이유로 현재 모든 항목을 실행한 것으로 간주하지 않는다.

## 스크립트 사본과 준비 조건

원본 `scripts/`는 **참고 템플릿**이다. 읽고, 대상에 맞춰 조정하고, `OUTDIR`에 사본을 두고, 변경 이유를 `CHANGES.md`에 기록한다. 대상과 맞지 않으면 구조만 참고하여 해당 사본을 수정한다.

| 준비 조건 | 시점 | 사본 대상 | 확인할 항목 |
| --- | --- | --- | --- |
| A: 정적 | Phase 0 이후 첫 harvest/spider 전 | `harvest_static.py`, `spider_mpa.py` | 대부분 기본 regex를 사용할 수 있으나 manifest/문법 차이가 있으면 endpoint regex, webpack/Vite publicPath, MPA exclude/cookie를 조정 |
| B: 런타임 | Phase 2 이후 depth/coverage 전 | `runtime_harvest.js`, `preload.js`, `config.json` | Cookie/localStorage 키, 성공 코드, stub 구조, login regex, API 접두, hash/history 라우팅 |

SPA의 순서는 강제되어 있다. Phase 0과 A에서 사본 준비를 끝내면 다음 실행 단계는 `python3 OUTDIR/harvest_static.py <URL> OUTDIR`다. 주 bundle을 수동 추출한 뒤 harvest 사용 여부를 결정하는 순서가 아니다. Phase 1 완료 전에는 `wc -l`로 결과를 확인하고 404나 endpoint 부족이 있으면 harvest를 수정한다. Phase 1b부터 저장된 `OUTDIR/js/*.js`에서 매개변수 조사를 한다.

MPA라면 Phase 0 다음 실행 단계는 `python3 OUTDIR/spider_mpa.py ...`다. 원문의 “다음 Bash”는 해당 조사 워크플로의 실행 순서를 뜻하며, 이 사람용 번역 파일이 자동 명령 실행을 지시하는 것은 아니다.

## 도구 사용과 출력량 제한

| 구분 | 원문 지침 |
| --- | --- |
| 큰 파일 | 100KB가 넘는 `index-*.js` 전체를 Read/grep 결과로 대화 맥락에 넣지 않고 사본 스크립트로 일괄 처리 |
| grep 결과 | `head -20` 또는 `-m 5` 등으로 제한; 대화에는 path 요약만 남기고 bundle 덩어리를 붙이지 않음 |
| 결과 확인 | `wc -l`, `ls \| wc -l`로 수를 확인하며 전체 디렉터리 내용을 읽어 넣지 않음 |
| 초기 regex 시험 | 선택 사항, 최대 한 번, 50KB 이하 작은 chunk나 HTML만; 정식 정적 결과는 harvest 기준 |
| 상세 참고 | 레시피·설정·문제 해결은 reference를 사용 |

## 실행 순서

1. **Phase 0:** SPA/MPA 분류, OUTDIR 생성.
2. **준비 A와 Phase 1:** 스크립트 사본 준비, harvest/spider 실행, 결과 수 확인.
3. **Phase 1b:** 경로 주변과 UI 바인딩에서 매개변수 후보 수집.
4. **Phase 2:** 렌더링·응답 인터셉터·메뉴/권한의 세 조건 분석, config 작성.
5. **준비 B와 Phase 3:** 런타임 사본 적용, depth/coverage/both와 매개변수 표본 기록.
6. **필요 시 Phase 4:** 권한 트리 복원, stub 수정, Phase 3 반복.
7. **Phase 5:** 경로·매개변수·기능·한계 보고서 작성, 발견된 모든 서비스/endpoint 자산을 `insert_assets`로 등록.

원문은 앞 단계가 완료되지 않은 채 다음 단계로 건너뛰지 않도록 한다.

## Phase 0 — 사이트 분류

진입 HTML을 읽고 `OUTDIR`을 만든다. Skill 원본 디렉터리의 스크립트는 수정하지 않는다.

- **SPA:** 앱 root와 chunk를 불러오는 빈 UI 틀이라면 Phase 1–5를 따른다.
- **MPA:** 서버 렌더링 HTML과 form 중심이고 endpoint bundle이 없으면 준비 A 뒤 spider를 사용한다.

원문의 명령 형식을 그대로 보존한다. `--cookie`가 예시에 존재하더라도 이 Skill의 자격증명을 요구/추측하거나 실제 로그인하지 않는 경계를 바꾸지는 않는다.

```bash
python3 recon/spider_mpa.py <BASE_URL> <OUTDIR> [--cookie "session=..."] [--max 300] [--depth 5] [--exclude "logout|delete|destroy"]
```

MPA 결과는 `forms.txt`, `links.txt`, `api_inline.txt`다. 실제로 SPA라 form이 거의 없다면 Phase 1로 전환한다.

## Phase 1 — 정적 수집

```bash
python3 recon/harvest_static.py <BASE_URL> <OUTDIR>
```

harvest는 HTML script, webpack/Vite manifest를 분석하고 지연 로딩 chunk를 내려받아 `js/`, `api_static.txt`, `routes.txt`, `chunkmap.txt`를 만든다. 선언된 chunk 수와 실제 내려받은 수를 비교한다.

```bash
wc -l OUTDIR/api_static.txt OUTDIR/routes.txt
ls OUTDIR/js | wc -l
```

404가 나면 개별 chunk를 수동 curl하는 대신 harvest 경로 해석을 수정하고 재시도한다. 정적 API가 지나치게 적으면 OUTDIR 사본의 endpoint 정규식을 넓히고 다시 수집한다.

### Phase 1b — 매개변수 역추적

path 목록과 매개변수 구조는 별개의 결과다. 주요 API에 대해 **필드명·전송 위치·추정 타입·필수 여부·예시 값·신뢰도**를 답할 수 있어야 한다.

#### 1b.0 — 전송 형태

| 형태 | 매개변수 위치 | 정적 코드에서 볼 곳 |
| --- | --- | --- |
| REST JSON | body와 query | path 주변 `params`, `data`, `body` 객체 |
| GraphQL | `variables` | gql 템플릿, `$page: Int` 같은 선언 |
| 전통 form | urlencoded | `<form>`, `FormData` |
| 파일 업로드 | multipart | `FormData.append` |
| 경로 매개변수 | `/user/:id` | 라우트 표, `useParams`, `$route.params` |
| 암호화/서명 | `sign`, `data` 등에 포장 | 암호화 함수에 들어가기 전 입력; reference D/J |

API마다 `transport: query|json|form|graphql|encrypted`를 기록한다.

#### 1b.1 — path 주변으로 분석 범위 넓히기

이미 찾은 path를 기준점으로 삼아 주변의 요청 조립 객체를 찾는다.

```bash
grep -n '"/api/user/list"' OUTDIR/js/*.js | head -20
grep -rhoaE '.{0,120}("/api[^"]+").{0,200}' OUTDIR/js/*.js | head -20
grep -rhoaE '(params|data|body|payload)\s*:\s*\{' OUTDIR/js/*.js | head -20
```

| 래퍼 | 매개변수 단서 |
| --- | --- |
| axios 인스턴스 | `data`, `params` |
| 공통 request 함수 | 인터셉터가 주입하는 전역 필드 |
| OpenAPI 생성 클라이언트 | 생성된 메서드 시그니처 |
| React Query / SWR | 훅에 전달하는 인수 |
| Vue composable | composable의 입력 |

`yup`, `zod`, 검증 rules, `Form.Item name=`, 포함된 Swagger에도 타입 단서가 남을 수 있다. 결과는 `param_candidates.json`의 `{ path, fields[], source: "static-callsite", confidence }` 형태로 정리한다.

#### 1b.2 — UI 바인딩 계층

폼 필드가 `onFinish`/`handleSubmit`에 들어가고, 변환 함수를 거친 뒤 API payload가 되는 연결을 추적한다.

| 시작점 | 추적 방식 |
| --- | --- |
| 폼 제출 | submit → transform → API |
| 표 검색 | `getFieldsValue()`에서 `params`로 이동 |
| 라우트 | `:id`, `?tab=` |
| 인터셉터 | `tenantId`, 공통 페이지 값, sign |
| 선택 상자 | options의 표시값과 실제 전송 enum 값 |

동적 분석에서는 DevTools 호출 스택의 `fetch`/`XHR.send`에서 상위 요청 조립 함수로 올라간다.

#### 1b.3 — 요청 조립의 세 질문

이 세 질문은 Phase 2의 인증 관련 세 조건과 다른 분류다.

| 질문 | 확인할 내용 |
| --- | --- |
| 조립 | payload는 어디서 만들어지고 어떤 transform을 거치는가 |
| 검증 | required, pattern, enum 규칙은 무엇인가 |
| 전송 | path, query, body, multipart, header 중 어디에 들어가는가 |

Phase 2의 인터셉터를 읽을 때 `Authorization`, `X-Tenant-Id`, sign 같은 전역 주입 필드도 함께 기록한다.

#### 1b.4 — 런타임과 연결

정적/바인딩 결과는 후보를 제공한다. 필수·선택·조건부 의존성은 Phase 3의 입력/동작 행렬과 여러 표본의 diff, Phase 5의 오류 근거로 보완한다.

## Phase 2 — 화면을 결정하는 세 조건

`OUTDIR/js/`에서 제한된 grep 결과를 읽고 결론을 `config.json`에 반영한다.

| 조건 | 질문 | 단서 |
| --- | --- | --- |
| 렌더링 | 프런트는 무엇을 보고 로그인했다고 판단하는가 | `isLogin`, `getToken`, Cookie/localStorage |
| 응답 인터셉터 | 무엇이 `/login` 이동을 유발하는가 | `response_code`, `errno`, axios interceptor |
| 콘텐츠 | 메뉴와 권한은 어디서 오는가 | menu, permission, role, acl, routes |

localStorage의 키 이름 자체를 실제 자격증명이라고 해석하지 않는다. chunk의 읽기/디코딩 함수와 요청 연결을 확인한다. 분석 결과를 config에 기록하고 `OUTDIR/runtime_harvest.js` / `preload.js`에 반영하는 것이 준비 B다.

### Phase 2b — 선택적인 API 관찰

| 설정 | 의미와 결과 |
| --- | --- |
| `recordDetail: true` | `__API_RECON_DETAIL__`에 상세 기록 |
| `observe.xhrHeaders: true` | 요청 헤더 관찰 |
| `extractUrlsFromResponse: true` | 응답 안의 추가 API URL 수집 |
| `observe.storageReads/cookieReads: true` | 세션에 사용하는 저장 키 확인 후 config 보완 |
| `neutralizeVueRouter: true` | Vue 라우트 관찰과 `__API_RECON_ROUTES__` 활용 |

coverage 각 회차에서 `__API_RECON_LOG__`, `__API_RECON_DETAIL__`, `__API_RECON_ROUTES__`, `__API_RECON_OBSERVE__`를 내보낸다.

## Phase 3 — 런타임 수집

준비 B와 자격증명 없는 mock 경계를 먼저 충족한다. config의 `runtimeMode`는 `depth`, `coverage`, `both` 중 하나다.

### Hook와 stub의 세 층

| 층 | 적용 대상 | 목적 |
| --- | --- | --- |
| L1 | 로그인/권한/bootstrap의 정확한 stub | 첫 화면 조건 충족 |
| L2 | JSON 응답의 실패 코드 | 프런트가 해석하는 미로그인 코드를 성공 코드로 바꿈 |
| L3 | L1에 맞지 않은 API 요청 | 비어 있는 성공 모양 응답으로 컴포넌트 렌더링 보조 |

depth는 가짜 프런트 상태, `forward`, `stubs`와 함께 hash/history의 routes를 순회하고 `runtime_api.json`을 만든다. `forward`가 참이면 실제 요청을 전달할 수 있다는 뜻이므로 완전 오프라인 mock과 구분해야 한다. 값의 상세 의미는 reference B를 읽는다.

coverage는 **document-start**, 즉 앱 코드가 실행되기 전에 `preload.js`를 주입한다. 원문 예시는 CDP의 `addScriptToEvaluateOnNewDocument` 또는 Userscript다. 실제 브라우저 자동화 도구의 지원 범위와 권한은 별도로 따라야 한다.

`window.__API_RECON_PRELOAD__` 존재와 업무 경로가 `/login`으로 되돌아가지 않는지로 적용을 확인한다.

```bash
cd recon && npm install
node runtime_harvest.js config.json
```

### 3b — coverage 동적 열거와 매개변수 행렬

1. 주요 탐색 메뉴와 사이드바를 열어 요청을 관찰하고 네트워크 반응을 1–3초 기다린다.
2. `role=tab`, `.ant-tabs-tab` 등 탭을 전환한다.
3. 표의 첫 행에 있는 보기·편집·상세를 살펴본다.
4. 툴바의 검색·새 항목·내보내기 관련 UI가 어떤 요청을 만들려 하는지 확인한다. **실제 삭제·민감 데이터 반출·대량 쓰기의 금지는 그대로 적용된다.**
5. 모듈에 들어갈 때마다 API와 라우트 결과를 합친다.
6. SPA에서만 `routes.txt`의 미확인 path를 통제된 `pushState`로 확인한다. MPA에는 적용하지 않는다.

각 모듈에서 서로 다른 동작의 표본을 기록해 차이를 비교한다.

| 동작 | 추가로 나타날 수 있는 필드 |
| --- | --- |
| 목록 첫 화면 | 페이지와 기본 필터 |
| 검색 | keyword, filter |
| 고급 필터 | 추가 선택 필드 |
| 생성/편집 UI | 엔터티의 전체 필드 |
| 일괄 선택/내보내기/정렬 | `ids[]`, `exportType`, `sortField` |

stub 응답을 쓸 때도 outbound body/headers를 기록할 수 있다. 응답에 임의로 넣은 mock 필드를 실제 요청 매개변수의 증거로 사용하지 않는다. 결과는 `scan_raw.json`, `param_samples.json`, `api_detail.json`이다.

Vue는 `neutralizeVueRouter: true`와 document-start preload, React는 정적 routes와 메뉴 클릭/`pushState`를 조합한다. both는 depth를 먼저 수행하고 coverage로 이어간다.

## Phase 4 — 권한 트리 복원

모듈이 빈 화면이거나 경로마다 locale/bootstrap 요청만 나온다면 콘텐츠 조건이 남아 있을 수 있다.

| 관찰 | 원문의 해석 |
| --- | --- |
| 로그인 뒤 UI 틀에는 들어감 | 렌더링과 응답 인터셉터 조건은 통과했을 수 있음 |
| 사이드바가 빠지거나 누르면 빈 화면 | stub 구조 또는 권한 코드가 부족할 수 있음 |
| 모든 경로에서 같은 소수 API만 발생 | `v-if permission` 같은 표시 조건이 충족되지 않았을 수 있음 |
| 정적 라우트 수가 bundle 내용보다 적음 | auth 모듈의 경로 매핑 보완 필요 |

```bash
grep -rhoaE '"/api[^"]*(permission|perm|role|menu|acl)[^"]*"' OUTDIR/js/*.js | sort -u | head -30
grep -rhoaE 'userRouteAuth|getResultTree|routeMap|routeLink|menuList|authList' OUTDIR/js/*.js | head -20
```

전형적인 연결은 `role_permissions`의 평탄한 코드 목록과 `permissions/all` 트리가 `getResultTree`로 합쳐지고, `userRouteAuth[CODE].url`로 실제 경로를 찾는 구조다. 서버 권한을 새로 부여하는 것이 아니라 프런트의 메뉴 소비 구조를 맞춘 mock 자료를 만드는 것이다.

```bash
python3 recon/extract_route_map.py recon/js recon/
python3 recon/build_perm_tree.py recon/js recon/ --config recon/config.json
```

중간 결과는 `route_map.json`, `userRouteAuth.json`, `permissions_tree.json`, `*_stub.json`, `perm_codes_all.txt`다. 외부 성공 코드 필드가 인터셉터 조건과 맞는지, 평탄한 코드와 트리 잎이 일치하는지, routes가 route_map의 link를 포함하는지 확인한다.

config를 수정하면 **Phase 3을 다시 실행**한다. 대형 SPA의 대기 조건은 `waitUntil`, `routeTimeout`, `perRouteMs`로 조정하는데, reference A3/I에서 맥락을 함께 읽는다.

## Phase 5 — 합치기와 보고

### 전체 결과물

| 결과 | 단계 | 내용 |
| --- | --- | --- |
| `js/`, `api_static.txt`, `routes.txt`, `chunkmap.txt` | 1 | 정적 bundle과 path |
| `param_candidates.json` | 1b | 정적 필드 후보 |
| `config.json` | 2 | 세 조건 분석과 런타임 설정 |
| `runtime_api.json` | 3a | depth 상세 기록, WS/SSE 포함 |
| `param_samples.json`, `scan_raw.json`, `api_detail.json` | 3b | 동작별 표본·클릭 기록·상세 |
| `route_map.json` 등 | 4 | 필요한 경우 권한 트리 중간 파일 |
| `params_merged.json` | 5 | 합친 매개변수와 신뢰도 |
| `api_merged.txt` | 5 | `METHOD /path [params] [static\|runtime\|both]` |
| `site_map.json` | 5 | 라우트·API·매개변수·기능·미확인 한계 |
| `insert_assets` | 5 | 찾은 모든 서비스와 endpoint를 자산 저장소에 등록 |

### 5b — 매개변수 병합

`param_samples.json`의 여러 표본을 비교한다. 원문은 모든 사이트에 적용되는 범용 병합 스크립트가 없다고 명시한다. 정적 callsite와 두 개 이상 런타임 표본이 일치하면 높은 신뢰도, 정적만 있거나 한 표본이면 중간, 오류/응답으로만 추론했으면 낮음, 아직 UI에서 만들지 못했으면 “발생 미확인”으로 구분한다.

### 5c — 오류 메시지로 보완

허가된 조사 범위 안에서 비파괴적인 불완전 요청의 400 응답이 필수 필드나 enum을 알려 줄 수 있다는 절차다. 이를 주입·권한 우회 등의 취약점 시험으로 확장하지 않는다. `field 'x' is required` 같은 메시지라도 바깥 `data`, GraphQL `variables`, 암호화 전 `bizData` 포장을 함께 확인한다.

보고서에는 `runtimeMode`, 정적/런타임 API 개수, 매개변수 신뢰도, 미확인 모듈, 원본 템플릿에서 무엇을 바꿨는지 `CHANGES.md` 요약을 적는다. 다음은 `site_map.json`의 원문 권장 구조다.

```json
{
  "site": "https://example.com",
  "runtimeMode": "both",
  "appType": "vue-spa",
  "routeGuardStrategy": ["nav-neutralize", "L1-auth", "L2-patch", "forward"],
  "apisFromStatic": [],
  "apisFromRuntime": [],
  "apis": [],
  "params": [{ "method": "POST", "path": "/api/user/list", "transport": "json", "fields": [] }],
  "frontendRoutes": [],
  "routesVerifiedByClick": [],
  "featuresTriggered": [],
  "limitations": ""
}
```

## 일반적인 적용 범위와 한계

- webpack, Vite, Angular 등의 지연 로딩에도 정적 chunk 수집과 런타임 관찰 원리를 적용한다.
- REST/JSON, GraphQL, WebSocket, SSE를 다루며 원문은 gRPC-web을 범위 밖으로 둔다.
- SSR에서도 클라이언트 fetch는 기록할 수 있지만 RSC와 Server Actions를 완전히 열거할 수 있다는 뜻은 아니다.
- JSVMP, WASM, 강한 HMAC/mTLS 검증처럼 현재 도구로 분석이 어려운 부분은 정적 결과와 한계를 남긴다.
- 조건부 입력 연동, 숨은 필드, WASM 내부 요청 조립은 “아직 발생시키지 못함” 또는 “현재 경로에서 도달 불가”로 기록한다.
- 런타임 화면이 막혀도 정적 코드에 보이는 endpoint 목록은 별도 결과로 남길 수 있다.

## 추가 자료와 원문 일관성

[reference.md](reference.md)는 grep 레시피, config, Hook 지원 범위, 문제 해결, 권한 트리, 매개변수 역추적을 상세히 설명한다.

원문의 주 Skill은 실제 로그인/자격증명 요구를 금지하지만 reference G와 I6에는 허가된 실제 세션 사용 예시가 남아 있다. 두 문서의 표현이 완전히 같지는 않다. 한국어판은 이 차이를 숨기지 않고 보존하여 설명하며, 참고 예시를 주 Skill의 명시적인 작업 경계를 확대하는 근거로 해석하지 않는다. 또한 reference의 `scripts/...` 명령은 참고 경로이므로 주 문서가 요구하는 OUTDIR 사본 준비와 조정을 생략하는 근거가 아니다.
