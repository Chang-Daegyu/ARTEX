# Skill 문서와 참고 스크립트 읽기

[문서 안내](README.md) · [에이전트 도구 조립](modules/agents.md)

## 1. Skill은 사람이 보는 설명서이자 모델에 들어가는 지시다

ARTEX의 `skills/`는 에이전트가 필요할 때 읽는 절차 문서입니다. 프롬프트·도구 schema와
마찬가지로 실행 결과에 영향을 줄 수 있습니다. 이번 이관에서는 이 원문을 보존하고,
사람이 읽을 한국어판을 **`docs/ko/skills/`**에 두었습니다. 한국어 동반 문서는 ARTEX가
자동으로 로드하지 않습니다.

코드와의 연결은 [server/assembly.go](../../server/assembly.go),
[agent/deferred.go](../../agent/deferred.go), [agent/assembly.go](../../agent/assembly.go)에
있습니다. Skill 이름·frontmatter·MCP 선언·도구 이름은 조회·노출 계약이므로 마음대로
번역해 식별자를 바꾸면 안 됩니다.

## 2. 전체 문서 안내

| 원본 | 한국어판 | 역할 |
| --- | --- | --- |
| `skills/api-recon/SKILL.md` | [API Recon 절차](skills/api-recon/SKILL.md) | 정적 코드와 실행 UI에서 인터페이스 구조 관찰 |
| `skills/api-recon/reference.md` | [API Recon 참고](skills/api-recon/reference.md) | 대상별 설정·추출 방법·문제 해결 |
| `skills/scopesentry/SKILL.md` | [ScopeSentry MCP](skills/scopesentry/SKILL.md) | 프로젝트·작업·자산·템플릿 API 사용 안내 |
| `skills/playwright-cli/SKILL.md` | [Playwright CLI](skills/playwright-cli/SKILL.md) | 브라우저 명령·snapshot 참조·세션 관리 |
| `references/element-attributes.md` | [요소 속성](skills/playwright-cli/references/element-attributes.md) | snapshot에 없는 속성 확인 |
| `references/playwright-tests.md` | [테스트 연결](skills/playwright-cli/references/playwright-tests.md) | Playwright 테스트 문맥의 활용 |
| `references/request-mocking.md` | [요청 모의 응답](skills/playwright-cli/references/request-mocking.md) | 테스트에서 요청 가로채기·응답 대체 |
| `references/running-code.md` | [코드 실행](skills/playwright-cli/references/running-code.md) | 페이지 평가와 Playwright 코드 실행 구분 |
| `references/session-management.md` | [세션 관리](skills/playwright-cli/references/session-management.md) | 브라우저 세션과 분리·정리 |
| `references/storage-state.md` | [저장 상태](skills/playwright-cli/references/storage-state.md) | 쿠키·웹 저장소 상태 보관·복원 |
| `references/test-generation.md` | [테스트 생성](skills/playwright-cli/references/test-generation.md) | 관찰을 반복 가능한 테스트로 전환 |
| `references/tracing.md` | [트레이싱](skills/playwright-cli/references/tracing.md) | 실행 추적 자료 수집과 읽기 |
| `references/video-recording.md` | [영상 기록](skills/playwright-cli/references/video-recording.md) | 브라우저 실행 기록의 시작·종료·활용 |

위 `references/` 원본의 기준 디렉터리는 `skills/playwright-cli/`입니다.
한국어판은 원문 절차를 이해하는 자료이며, 현재 설치된 도구 버전에서 모든 명령이
성공함을 이번 작업으로 검증했다는 의미는 아닙니다.

## 3. API Recon의 두 관찰 방식

정적 관찰은 HTML과 JS 번들을 읽어 API 경로·라우트 후보를 찾습니다. 실제 HTTP 메서드,
실행 시 조립하는 URL, 어떤 입력이 필수인지는 이것만으로 확정하기 어렵습니다.
실행 관찰은 브라우저에서 UI를 렌더링하고 어떤 요청이 생성되는지 기록합니다.
모의 응답으로 렌더링한 UI의 결과와 실제 서버에서 검증한 결과는 구분해야 합니다.

원본 Skill은 정적 수집, 설정 확인, 실행 관찰, 메뉴 구조 보강, 파라미터 비교와 통합 산출물의
순서를 제시합니다. 포함된 스크립트는 **범용 완제품이 아니라 대상에 맞춰 읽고 수정하는
참고 템플릿**이라고 명시합니다. 특히 응답 바꾸기, Cookie/localStorage, 라우터 조정은
페이지 렌더링의 문맥을 만드는 방법이며 서버의 인증 성공을 증명하지 않습니다.

## 4. 포함된 6개 실행 스크립트

| 스크립트 | 입력 → 출력 | 주요 읽기 지점 |
| --- | --- | --- |
| [harvest_static.py](../../skills/api-recon/scripts/harvest_static.py) | HTML/JS URL → 내려받은 JS, API·라우트 후보 | manifest 해석, URL 조합, 병렬 다운로드, 최대 6회 확장 |
| [spider_mpa.py](../../skills/api-recon/scripts/spider_mpa.py) | 페이지 URL → 폼 메서드·필드·링크 | HTMLParser 콜백, BFS, 깊이·방문 수·필터 |
| [extract_route_map.py](../../skills/api-recon/scripts/extract_route_map.py) | 다운로드된 JS → `route_map.json` | 최대 일치 파일 선택, KEY/name/link 정규식 |
| [build_perm_tree.py](../../skills/api-recon/scripts/build_perm_tree.py) | 권한 모듈·라우트 사전 → 메뉴/권한 stub | 별칭 해석, 접두어 부모 규칙, 트리 재귀, 설정 덮어쓰기 |
| [runtime_harvest.js](../../skills/api-recon/scripts/runtime_harvest.js) | 설정 파일·브라우저 → 실행 요청·라우트별 목록 | 요청 가로채기, stub 우선, 실제 forward 분기, WS/SSE |
| [preload.js](../../skills/api-recon/scripts/preload.js) | document-start 주입 → 페이지 전역 관찰 목록 | fetch/XHR 래핑, 기록 제한, mock tier, 라우터 훅 |

이 여섯 파일은 파일·함수별 한국어 주석을 직접 넣었습니다. Python docstring과 JS 문자열,
네트워크 옵션, 정규식, 실행 코드는 원문을 유지합니다.

### 정적 자료 흐름

HTML → 초기 JS → 청크 manifest → 추가 JS → API/라우트 후보라는 흐름입니다.
`harvest_static.py`는 프레임워크별 manifest 모양을 정규식으로 해석합니다. 따라서
모든 JavaScript 문법을 분석하는 파서가 아니며 미지원 번들 형태는 놓칠 수 있습니다.

`extract_route_map.py`는 여러 파일을 합치는 대신 가장 많은 일치 항목을 찾은 후보를
선택합니다. `build_perm_tree.py`는 코드 접두어와 예외 규칙으로 계층을 추정하므로
실제 서버 권한 모델을 정확히 복원했다고 자동 판단하면 안 됩니다.

### 실행 자료 흐름

`runtime_harvest.js`는 설정을 읽어 브라우저를 열고 API 패턴에 맞는 요청을 기록합니다.
명시적인 stub을 먼저 적용하고, SSE는 긴 연결로 멈추지 않도록 별도 처리합니다.
**`forward` 기본값은 true**입니다. “모의 응답을 사용한다”는 말만 보고 전체 실행이
네트워크 없이 돌아간다고 해석하면 안 됩니다.

`preload.js`는 페이지 컨텍스트에 주입되므로 Node 코드와 사용하는 전역이 다릅니다.
`window.__API_RECON_*`는 페이지 내부의 관찰 버퍼이며 ARTEX의 영속 증거 저장소 자체가
아닙니다. 후속 도구가 읽어 산출물로 정리해야 합니다.

## 5. JSON 두 파일

`skills/api-recon/scripts/package.json`은 Puppeteer Core 의존성을 설명하고,
`package-lock.json`은 그 해석 결과와 무결성 정보를 고정합니다. 둘 다 원문을 보존합니다.
원본에서는 두 파일에 기록된 패키지 이름이 다르므로 이름이 일치한다고 가정하지 마세요.
설치 결과에 영향을 주는 잠금 파일을 단순 번역 대상으로 취급하지 않습니다.

## 6. Skill을 다른 분야에 바꾸어 쓰는 방법

Skill의 재사용할 요소는 **입력 조건 → 수행 절차 → 결과 형식 → 완료 기준 → 한계 명시**입니다.
예를 들어 품질 측정 Skill이라면 입력은 펌웨어 버전·시험 신호, 도구는 측정기 API,
출력은 측정 표와 원본 파일, 완료 기준은 측정 조건 충족과 오차 허용치가 됩니다.

도구 이름·JSON schema는 프로그램 계약으로 두고 설명 언어를 분리하면 한국어 문서를
추가하기 쉽습니다. 다만 사람이 읽는 번역을 런타임 Skill로 채택하는 순간에는 모델 행동이
바뀔 수 있으므로 고정한 입력으로 도구 선택·출력 형식·완료 판단을 검증해야 합니다.
