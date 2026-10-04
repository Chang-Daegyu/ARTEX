> **사람용 한국어 번역·해설입니다. ARTEX가 이 문서를 자동으로 로드하지 않습니다.**
> 원본: [references/tracing.md](../../../../../skills/playwright-cli/references/tracing.md). 실행용 원본 파일은 그대로 유지합니다.
> 본문·제목·제약을 한국어로 옮기고, 명령/코드 블록은 원문 그대로 보존했습니다. 예제 안의 설명은 마지막 한국어 보충 해설에서도 풀어 씁니다.

# 실행 추적

디버깅과 분석을 위해 자세한 실행 trace를 수집합니다. 원문은 DOM 스냅샷, 스크린샷, 네트워크 활동, 콘솔 로그를 포함한다고 설명합니다.

## 기본 사용

```bash
# Start trace recording
playwright-cli tracing-start

# Perform actions
playwright-cli open https://example.com
playwright-cli click e1
playwright-cli fill e2 "test"

# Stop trace recording
playwright-cli tracing-stop
```

## 추적 출력 파일

원문에 따르면 추적을 시작하면 Playwright는 여러 파일이 들어 있는 `traces/` 디렉터리를 만듭니다.

### `trace-{timestamp}.trace` — 동작 기록

**동작 로그:** 주 추적 파일이며 다음을 포함합니다.

- 수행한 각 동작: 클릭, 입력, 페이지 이동.
- 각 동작 전후의 DOM 스냅샷.
- 단계별 스크린샷.
- 소요 시간 정보.
- 콘솔 메시지.
- 소스 위치.

### `trace-{timestamp}.network` — 네트워크 기록

**네트워크 로그:** 네트워크 활동의 상세 정보입니다.

- HTTP 요청과 응답.
- 요청 헤더와 본문.
- 응답 헤더와 본문.
- DNS, 연결, TLS, 첫 바이트 도착 시간(TTFB), 다운로드 시간.
- 리소스 크기.
- 실패한 요청과 오류.

### `resources/` — 리소스

**리소스 디렉터리:** 캐시에 보관한 자료입니다.

- 이미지, 글꼴, 스타일시트, 스크립트.
- 재생용 응답 본문.
- 페이지 상태를 재구성하는 데 필요한 자산.

## trace가 기록하는 내용

| 구분 | 내용 |
|----------|---------|
| **조작** | 클릭, 입력, hover, 키보드 입력, 페이지 이동 |
| **DOM** | 각 동작 전후의 DOM 스냅샷 |
| **스크린샷** | 단계별 시각 상태 |
| **네트워크** | 요청, 응답, 헤더, 본문, 시간 |
| **콘솔** | console.log, warn, error 메시지 |
| **시간** | 각 작업의 상세 시간 |

## 활용 사례

### 실패한 조작 디버깅

```bash
playwright-cli tracing-start
playwright-cli open https://app.example.com

# This click fails - why?
playwright-cli click e5

playwright-cli tracing-stop
# Open trace to see DOM state when click was attempted
```

### 성능 분석

```bash
playwright-cli tracing-start
playwright-cli open https://slow-site.com
playwright-cli tracing-stop

# View network waterfall to identify slow resources
```

### 작업 증거 수집

```bash
# Record a complete user flow for documentation
playwright-cli tracing-start

playwright-cli open https://app.example.com/checkout
playwright-cli fill e1 "4111111111111111"
playwright-cli fill e2 "12/25"
playwright-cli fill e3 "123"
playwright-cli click e4

playwright-cli tracing-stop
# Trace shows exact sequence of events
```

## Trace·영상·스크린샷 비교

| 특성 | Trace | 영상 | 스크린샷 |
|---------|-------|-------|------------|
| **형식** | .trace 파일 | .webm 영상 | .png/.jpeg 이미지 |
| **DOM 확인** | 가능 | 불가 | 불가 |
| **네트워크 상세** | 가능 | 불가 | 불가 |
| **단계별 재생** | 가능 | 연속 영상 | 한 프레임 |
| **파일 크기** | 중간 | 큼 | 작음 |
| **주요 용도** | 디버깅 | 데모 | 빠른 화면 기록 |

## 권장 사용법

### 1. 문제가 생기기 전에 추적 시작

```bash
# Trace the entire flow, not just the failing step
playwright-cli tracing-start
playwright-cli open https://example.com
# ... all steps leading to the issue ...
playwright-cli tracing-stop
```

### 2. 오래된 trace 정리

trace가 많은 디스크 공간을 차지할 수 있으므로 원문은 다음 정리 예제를 제시합니다.

```bash
# Remove traces older than 7 days
find .playwright-cli/traces -mtime +7 -delete
```

## 제약

- 추적에는 추가 실행 비용이 듭니다.
- 큰 trace는 디스크 공간을 많이 사용할 수 있습니다.
- 일부 동적 콘텐츠는 완벽하게 재생되지 않을 수 있습니다.

## 한국어 예제 해설

실패 조사에서는 문제가 발생하기 전부터 기록해, 클릭 직전 DOM과 네트워크 상황을 함께 봅니다. 성능 예제는 네트워크 waterfall에서 느린 리소스를 찾는 용도입니다. 증거 예제는 결제 폼 입력부터 제출까지의 순서를 기록합니다. 입력값은 원문 시연 문자열로 유지했습니다.

Trace는 조사 가능한 구조화된 기록, 영상은 눈에 보이는 시간 흐름, 스크린샷은 특정 순간의 이미지라는 목적 차이가 있습니다. 파일 크기는 원문의 정성적 비교이며 실제 크기는 페이지와 기록 길이에 따라 달라집니다. 마지막 `find … -delete`는 오래된 파일을 실제 삭제하는 정리 명령 예제이므로 단순 trace 조회와 구분해서 읽습니다.
