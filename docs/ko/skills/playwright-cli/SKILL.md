> **사람용 한국어 번역·해설입니다. ARTEX가 이 문서를 자동으로 로드하지 않습니다.**
> 원본: [SKILL.md](../../../../skills/playwright-cli/SKILL.md). 실행용 원본 파일은 그대로 유지합니다.
> 본문·제목·제약을 한국어로 옮기고, 명령/코드 블록은 원문 그대로 보존했습니다. 예제 안의 설명은 마지막 한국어 보충 해설에서도 풀어 씁니다.

**원본 Skill 메타데이터**

```yaml
name: playwright-cli
description: Automate browser interactions, test web pages and work with Playwright tests.
allowed-tools: Bash(playwright-cli:*) Bash(npx:*) Bash(npm:*)
```

설명 번역: 브라우저 조작을 자동화하고 웹 페이지를 시험하며 Playwright 테스트를 다룹니다. `allowed-tools`는 원본 Skill의 선언으로 보존한 정보입니다.

# playwright-cli로 브라우저 자동화하기

## 빠른 시작

```bash
# open new browser
playwright-cli open
# navigate to a page
playwright-cli goto https://playwright.dev
# interact with the page using refs from the snapshot
playwright-cli click e15
playwright-cli type "page.click"
playwright-cli press Enter
# take a screenshot (rarely used, as snapshot is more common)
playwright-cli screenshot
# close the browser
playwright-cli close
```

## 명령

### 기본 조작

```bash
playwright-cli open
# open and navigate right away
playwright-cli open https://example.com/
playwright-cli goto https://playwright.dev
playwright-cli type "search query"
playwright-cli click e3
playwright-cli dblclick e7
# --submit presses Enter after filling the element
playwright-cli fill e5 "user@example.com"  --submit
playwright-cli drag e2 e8
# drop files or data onto an element (from outside the page)
playwright-cli drop e4 --path=./image.png
playwright-cli drop e4 --data="text/plain=hello world"
playwright-cli hover e4
playwright-cli select e9 "option-value"
playwright-cli upload ./document.pdf
playwright-cli check e12
playwright-cli uncheck e12
playwright-cli snapshot
# search the snapshot for text or a regexp, returns matching nodes with surrounding context
playwright-cli find "Sign in"
playwright-cli find --regex "Sign (in|up)"
# wrap the regexp in slashes to add flags, e.g. /i for case-insensitive
playwright-cli find --regex "/sign (in|up)/i"
playwright-cli eval "document.title"
playwright-cli eval "el => el.textContent" e5
# get element id, class, or any attribute not visible in the snapshot
playwright-cli eval "el => el.id" e5
playwright-cli eval "el => el.getAttribute('data-testid')" e5
playwright-cli dialog-accept
playwright-cli dialog-accept "confirmation text"
playwright-cli dialog-dismiss
playwright-cli resize 1920 1080
playwright-cli close
```

### 탐색

```bash
playwright-cli go-back
playwright-cli go-forward
playwright-cli reload
```

### 키보드

```bash
playwright-cli press Enter
playwright-cli press ArrowDown
playwright-cli keydown Shift
playwright-cli keyup Shift
```

### 마우스

```bash
playwright-cli mousemove 150 300
playwright-cli mousedown
playwright-cli mousedown right
playwright-cli mouseup
playwright-cli mouseup right
playwright-cli mousewheel 0 100
```

### 이미지/PDF로 저장

```bash
playwright-cli screenshot
playwright-cli screenshot e5
playwright-cli screenshot --filename=page.png
playwright-cli screenshot --hires
playwright-cli pdf --filename=page.pdf
```

### 탭

```bash
playwright-cli tab-list
playwright-cli tab-new
playwright-cli tab-new https://example.com/page
playwright-cli tab-close
playwright-cli tab-close 2
playwright-cli tab-select 0
```

### 저장소

```bash
playwright-cli state-save
playwright-cli state-save auth.json
playwright-cli state-load auth.json

# Cookies
playwright-cli cookie-list
playwright-cli cookie-list --domain=example.com
playwright-cli cookie-get session_id
playwright-cli cookie-set session_id abc123
playwright-cli cookie-set session_id abc123 --domain=example.com --httpOnly --secure
playwright-cli cookie-delete session_id
playwright-cli cookie-clear

# LocalStorage
playwright-cli localstorage-list
playwright-cli localstorage-get theme
playwright-cli localstorage-set theme dark
playwright-cli localstorage-delete theme
playwright-cli localstorage-clear

# SessionStorage
playwright-cli sessionstorage-list
playwright-cli sessionstorage-get step
playwright-cli sessionstorage-set step 3
playwright-cli sessionstorage-delete step
playwright-cli sessionstorage-clear
```

### 네트워크

```bash
playwright-cli route "**/*.jpg" --status=404
playwright-cli route "https://api.example.com/**" --body='{"mock": true}'
playwright-cli route-list
playwright-cli unroute "**/*.jpg"
playwright-cli unroute
```

### 개발자 도구

```bash
playwright-cli console
playwright-cli console warning
playwright-cli requests
playwright-cli request 5
playwright-cli run-code "async page => await page.context().grantPermissions(['geolocation'])"
playwright-cli run-code --filename=script.js
playwright-cli tracing-start
playwright-cli tracing-stop
playwright-cli video-start video.webm
playwright-cli video-chapter "Chapter Title" --description="Details" --duration=2000
playwright-cli video-stop

# annotate each subsequent action (click, type, ...) with a callout naming the action and highlighting the target
playwright-cli video-show-actions --duration=600 --position=top-right
playwright-cli video-hide-actions

# launch the dashboard for UI review / design feedback — user annotates the page, you receive the annotated screenshot, snapshot, and notes
playwright-cli show --annotate

# generate a Playwright locator for an element from its ref or selector
playwright-cli generate-locator e5 --raw

# show a persistent highlight overlay for an element, optionally with a custom style
playwright-cli highlight e5
playwright-cli highlight e5 --style="outline: 3px dashed red"
# hide a single element highlight, or all page highlights when no target is given
playwright-cli highlight e5 --hide
playwright-cli highlight --hide
```

## 가공하지 않은 결과 출력

전역 `--raw` 옵션은 페이지 상태, 생성된 코드, 스냅샷 구역을 출력에서 제거하고 결과값만 반환합니다. 명령 결과를 다른 도구에 파이프로 전달할 때 사용합니다. 원래 결과가 없는 명령은 아무것도 출력하지 않습니다.

```bash
playwright-cli --raw eval "JSON.stringify(performance.timing)" | jq '.loadEventEnd - .navigationStart'
playwright-cli --raw eval "JSON.stringify([...document.querySelectorAll('a')].map(a => a.href))" > links.json
playwright-cli --raw snapshot > before.yml
playwright-cli click e5
playwright-cli --raw snapshot > after.yml
diff before.yml after.yml
TOKEN=$(playwright-cli --raw cookie-get session_id)
playwright-cli --raw localstorage-get theme
```

각 응답을 JSON으로 감싼 구조화된 출력을 원하면 `--json`을 전달합니다.
```bash
playwright-cli list --json
```

## 브라우저 열기/연결 매개변수
```bash
# Use specific browser when creating session
playwright-cli open --browser=chrome
playwright-cli open --browser=firefox
playwright-cli open --browser=webkit
playwright-cli open --browser=msedge

# Emulate a generic mobile device (Pixel 10 for Chromium, iPhone 17 for WebKit).
# Prefer this when a mobile layout is acceptable: mobile pages are usually
# lighter, so snapshots are smaller and cheaper.
playwright-cli open --mobile
playwright-cli open --device="iPhone 15"

# Use persistent profile (by default profile is in-memory)
playwright-cli open --persistent
# Use persistent profile with custom directory
playwright-cli open --profile=/path/to/profile

# Connect to browser via Playwright Extension
playwright-cli attach --extension=chrome

# Connect to a running Chrome or Edge by channel name
playwright-cli attach --cdp=chrome
playwright-cli attach --cdp=msedge

# Connect to a running browser via CDP endpoint
playwright-cli attach --cdp=http://localhost:9222

# Start with config file
playwright-cli open --config=my-config.json

# Close the browser
playwright-cli close
# Detach from an attached browser (leaves the external browser running)
playwright-cli -s=msedge detach
# Delete user data for the default session
playwright-cli delete-data
```

## Windows에서 `&`가 포함된 URL

Windows의 `cmd.exe`와 PowerShell은 `&`를 명령 구분자로 해석하므로, 쿼리 매개변수가 여러 개인 URL이 `playwright-cli` 실행 전에 잘릴 수 있습니다. `cmd.exe`에서는 `^&`로 이스케이프하거나 PowerShell에서는 `--%`를 사용합니다.

```batch
playwright-cli goto "https://example.com/?a=1^&b=2"
```

```powershell
playwright-cli --% goto "https://example.com/?a=1&b=2"
```

## 스냅샷

각 명령 실행 후 playwright-cli는 현재 브라우저 상태의 스냅샷을 제공합니다.

```bash
> playwright-cli goto https://example.com
### Page
- Page URL: https://example.com/
- Page Title: Example Domain
### Snapshot
[Snapshot](.playwright-cli/page-2026-02-14T19-22-42-679Z.yml)
```

필요한 시점에 `playwright-cli snapshot`으로 직접 스냅샷을 얻을 수도 있습니다. 아래 옵션은 필요에 따라 조합할 수 있습니다.

```bash
# default - save to a file with timestamp-based name
playwright-cli snapshot

# save to file, use when snapshot is a part of the workflow result
playwright-cli snapshot --filename=after-click.yaml

# snapshot an element instead of the whole page
playwright-cli snapshot "#main"

# limit snapshot depth for efficiency, take a partial snapshot afterwards
playwright-cli snapshot --depth=4
playwright-cli snapshot e34

# include each element's bounding box as [box=x,y,width,height]
playwright-cli snapshot --boxes

# search a large snapshot instead of capturing it all — returns matching nodes
# with 3 lines of context around each match (like grep -C)
playwright-cli find "Add to cart"
playwright-cli find --regex "\\$[0-9]+\\.[0-9]{2}"
```

## 요소 지정

기본적으로 스냅샷에 표시된 ref를 사용하여 페이지 요소를 조작합니다.

```bash
# get snapshot with refs
playwright-cli snapshot

# interact using a ref
playwright-cli click e15
```

CSS 선택자 또는 Playwright locator도 사용할 수 있습니다.

```bash
# css selector
playwright-cli click "#main > button.submit"

# role locator
playwright-cli click "getByRole('button', { name: 'Submit' })"

# test id
playwright-cli click "getByTestId('submit-button')"
```

## 브라우저 세션

```bash
# create new browser session named "mysession" with persistent profile
playwright-cli -s=mysession open example.com --persistent
# same with manually specified profile directory (use when requested explicitly)
playwright-cli -s=mysession open example.com --profile=/path/to/profile
playwright-cli -s=mysession click e6
playwright-cli -s=mysession close  # stop a named browser
playwright-cli -s=mysession delete-data  # delete user data for persistent session

playwright-cli list
# Close all browsers
playwright-cli close-all
# Forcefully kill all browser processes
playwright-cli kill-all
```

## 설치

전역 `playwright-cli` 명령이 없다면 `npx playwright cli`로 로컬 버전을 사용할 수 있는지 먼저 확인합니다.

```bash
npx --no-install playwright --version
```

로컬 버전이 있으면 모든 명령에서 `npx playwright cli`를 사용합니다. 없으면 다음과 같이 `playwright-cli` 전역 명령을 설치합니다.

```bash
npm install -g @playwright/cli@latest
```

## 예제: 폼 제출

```bash
playwright-cli open https://example.com/form
playwright-cli snapshot

playwright-cli fill e1 "user@example.com"
playwright-cli fill e2 "password123"
playwright-cli click e3
playwright-cli snapshot
playwright-cli close
```

## 예제: 여러 탭을 사용하는 작업

```bash
playwright-cli open https://example.com
playwright-cli tab-new https://example.com/other
playwright-cli tab-list
playwright-cli tab-select 0
playwright-cli snapshot
playwright-cli close
```

## 예제: 개발자 도구로 디버깅

```bash
playwright-cli open https://example.com
playwright-cli click e4
playwright-cli fill e7 "test"
playwright-cli console
playwright-cli requests
playwright-cli close
```

```bash
playwright-cli open https://example.com
playwright-cli tracing-start
playwright-cli click e4
playwright-cli fill e7 "test"
playwright-cli tracing-stop
playwright-cli close
```

## 예제: 대화형 검토 세션

UI 검토나 디자인 피드백을 받을 때 사용자는 실제 페이지 위에 사각형을 그리고 의견을 입력합니다. 에이전트는 표시된 스크린샷, 표시 영역의 스냅샷, 사용자의 의견을 받습니다. 원본은 사용자가 “UI review”, “design feedback”, 또는 사용자의 생각·요구·의도를 물어보라고 요청한 경우에 이 기능을 사용하도록 안내합니다.

```bash
playwright-cli open https://example.com
playwright-cli show --annotate
```

## 세부 작업별 참고 문서

* **Playwright 테스트 실행과 디버깅** [references/playwright-tests.md](references/playwright-tests.md)
* **네트워크 요청 모킹** [references/request-mocking.md](references/request-mocking.md)
* **Playwright 코드 실행** [references/running-code.md](references/running-code.md)
* **브라우저 세션 관리** [references/session-management.md](references/session-management.md)
* **저장 상태: 쿠키와 localStorage** [references/storage-state.md](references/storage-state.md)
* **테스트 계획·생성·복구** [references/test-generation.md](references/test-generation.md)
* **실행 추적** [references/tracing.md](references/tracing.md)
* **영상 녹화** [references/video-recording.md](references/video-recording.md)
* **요소 속성 확인** [references/element-attributes.md](references/element-attributes.md)

## 한국어 보충 해설: 명령을 읽는 방법

아래는 원문 예제 안의 영어 주석과 명령 옵션을 한국어로 풀어 쓴 색인입니다. 위의 명령/코드 블록은 인자·따옴표·출력 예시를 포함하여 원문 그대로 두었습니다. 이 문서를 읽는 것만으로 브라우저가 시작되거나 아래 작업이 실행되지는 않습니다.

### 기본 조작과 상태 확인

| 명령/옵션 | 의미 |
|---|---|
| `open`, `open URL` | 새 브라우저 세션 열기, 선택적으로 즉시 URL 이동 |
| `goto`, `go-back`, `go-forward`, `reload` | 주소 이동, 방문 기록 뒤/앞으로, 새로고침 |
| `type` | 현재 입력 위치에 텍스트 입력 |
| `fill ref text --submit` | 특정 입력 요소 채우기, `--submit`이면 이어서 Enter 입력 |
| `click`, `dblclick`, `hover` | ref/선택자로 지정한 요소 클릭·두 번 클릭·포인터 올리기 |
| `drag source target` | 페이지 안의 한 요소에서 다른 요소로 드래그 |
| `drop ref --path=…`, `--data=…` | 페이지 밖의 파일 또는 지정 MIME 데이터를 요소에 드롭 |
| `select ref value` | 선택 요소에서 지정 값 고르기 |
| `upload path` | 파일 업로드 조작에 사용할 파일 지정 |
| `check`, `uncheck` | 체크 요소를 선택/해제 상태로 만들기 |
| `press`, `keydown`, `keyup` | 키 입력 또는 누른 상태/떼는 상태를 나눠 제어 |
| `mousemove`, `mousedown`, `mouseup`, `mousewheel` | 좌표 이동, 마우스 버튼 누르기/떼기, 휠 이동 |
| `snapshot` | 현재 페이지의 구조와 요소 ref 읽기 |
| `snapshot --filename=…` | 스냅샷을 결과물에 쓸 이름으로 저장 |
| `snapshot ref`, `--depth=4` | 요소/영역에 한정하거나 깊이를 제한해 작은 스냅샷 읽기 |
| `snapshot --boxes` | 요소별 `[box=x,y,width,height]` 위치 정보 포함 |
| `find text`, `find --regex …` | 큰 스냅샷에서 일치하는 노드와 주변 문맥 검색 |
| `find --regex "/pattern/i"` | 슬래시로 정규식을 감싸 대소문자 무시 같은 플래그 지정 |
| `eval expression`, `eval function ref` | 페이지 표현식 또는 특정 요소를 인자로 하는 함수를 평가 |
| `dialog-accept`, `dialog-dismiss` | 브라우저 대화상자 수락/취소. 수락 시 텍스트를 넘길 수도 있음 |
| `resize width height` | 브라우저 표시 영역 크기 변경 |
| `screenshot`, `screenshot ref` | 전체 페이지 또는 지정 요소의 이미지 저장 |
| `--filename`, `--hires`, `pdf` | 파일명 지정, 고해상도 이미지, PDF 저장 |
| `tab-list`, `tab-new`, `tab-close`, `tab-select` | 탭 목록·생성·닫기·선택. 예제의 숫자는 탭 인덱스 |

원문은 일반적인 요소 탐색에는 스크린샷보다 스냅샷을 더 자주 사용한다고 설명합니다. ref는 스냅샷에 실제로 보인 요소를 가리키며, CSS/role/test-id locator도 사용할 수 있습니다.

### 기록, 표시, 네트워크

| 명령 | 의미 |
|---|---|
| `state-save`, `state-load` | 브라우저 저장 상태 파일 내보내기/불러오기 |
| `cookie-*` | 쿠키 조회·설정·삭제. domain/path/httpOnly/secure 등의 옵션 가능 |
| `localstorage-*`, `sessionstorage-*` | 현재 저장소의 키 목록·단일 값·설정·삭제·전체 비우기 |
| `route`, `route-list`, `unroute` | URL 패턴별 가짜 응답/차단 규칙 등록·조회·해제 |
| `console`, `console warning` | 콘솔 기록을 읽고 필요하면 수준으로 좁히기 |
| `requests`, `request 5` | 요청 목록 또는 지정 요청 상세 조회 |
| `run-code`, `--filename` | Playwright 함수 표현식 또는 파일의 함수 실행 |
| `tracing-start`, `tracing-stop` | 동작·DOM·네트워크 등을 추적하는 기록 시작/종료 |
| `video-start`, `video-stop` | 영상 기록 시작/저장 |
| `video-chapter` | 제목·설명·표시 시간으로 구간 안내 카드 삽입 |
| `video-show-actions`, `video-hide-actions` | 이후 조작에 이름/대상 강조 표시 켜기·끄기 |
| `show --annotate` | 사용자가 실제 UI 위에 표시하고 피드백을 남기는 검토 화면 |
| `generate-locator ref --raw` | ref에 대응하는 Playwright locator 표현식만 얻기 |
| `highlight ref`, `--style` | 요소를 계속 강조하고 필요하면 테두리 스타일 지정 |
| `highlight ref --hide`, `highlight --hide` | 특정 요소 또는 모든 강조 표시 제거 |

`--raw` 예제는 JSON 결과를 `jq`에 넘기거나 파일로 저장하고, 조작 전후 YAML 스냅샷을 비교하는 방식을 보여 줍니다. 쿠키값을 쉘 변수에 담는 예제 역시 결과값만 출력한다는 의미입니다. `--json`은 전체 응답을 구조화된 JSON으로 감싼다는 점에서 다릅니다.

### 브라우저 선택과 프로필

| 옵션/명령 | 원문 설명 |
|---|---|
| `--browser=chrome/firefox/webkit/msedge` | 세션을 만들 브라우저 선택 |
| `--mobile` | 일반적인 모바일 기기 모사. 원문 기준 Chromium은 Pixel 10, WebKit은 iPhone 17 |
| `--device="iPhone 15"` | 기기 프리셋을 이름으로 지정 |
| `--persistent` | 메모리 기본 프로필 대신 디스크 프로필 사용 |
| `--profile=/path/to/profile` | 프로필 디렉터리를 명시. 원문은 사용자가 명시적으로 원할 때 지정하도록 안내 |
| `attach --extension…` | Playwright 확장 프로그램을 통한 기존 브라우저 연결 |
| `attach --cdp=chrome/msedge/URL` | 채널 이름 또는 CDP 엔드포인트로 연결 |
| `--config=…` | 설정 파일로 세션 시작 |
| `close` | CLI가 연 브라우저 종료 |
| `detach` | 연결만 해제하고 외부 브라우저는 유지 |
| `delete-data` | 세션의 디스크 사용자 데이터 삭제 |
| `-s=name` | 명령을 이름 있는 세션에 적용 |
| `list`, `close-all`, `kill-all` | 세션 목록, 모든 브라우저 닫기, 남은 프로세스 강제 종료 |

원문은 모바일 레이아웃으로 충분한 경우 페이지와 스냅샷이 작아질 수 있어 모바일 모드를 선호할 수 있다고 설명합니다. 이는 이 저장소 원문 문서의 안내이며, 이 번역 작업에서 설치된 CLI 버전별 지원 여부를 실행 검증하지는 않았습니다.
