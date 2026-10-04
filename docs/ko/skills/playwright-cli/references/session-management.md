> **사람용 한국어 번역·해설입니다. ARTEX가 이 문서를 자동으로 로드하지 않습니다.**
> 원본: [references/session-management.md](../../../../../skills/playwright-cli/references/session-management.md). 실행용 원본 파일은 그대로 유지합니다.
> 본문·제목·제약을 한국어로 옮기고, 명령/코드 블록은 원문 그대로 보존했습니다. 예제 안의 설명은 마지막 한국어 보충 해설에서도 풀어 씁니다.

# 브라우저 세션 관리

상태를 보관하면서 서로 분리된 브라우저 세션을 여러 개 동시에 실행합니다.

## 이름이 있는 브라우저 세션

브라우저 컨텍스트를 분리하려면 `-s` 플래그를 사용합니다.

```bash
# Browser 1: Authentication flow
playwright-cli -s=auth open https://app.example.com/login

# Browser 2: Public browsing (separate cookies, storage)
playwright-cli -s=public open https://example.com

# Commands are isolated by browser session
playwright-cli -s=auth fill e1 "user@example.com"
playwright-cli -s=public snapshot
```

## 브라우저 세션별 분리 항목

각 브라우저 세션은 다음 항목을 독립적으로 가집니다.

- 쿠키
- LocalStorage / SessionStorage
- IndexedDB
- 캐시
- 방문 기록
- 열린 탭

## 브라우저 세션 명령

```bash
# List all browser sessions
playwright-cli list

# Stop a browser session (close the browser)
playwright-cli close                # stop the default browser
playwright-cli -s=mysession close   # stop a named browser

# Stop all browser sessions
playwright-cli close-all

# Forcefully kill all daemon processes (for stale/zombie processes)
playwright-cli kill-all

# Delete browser session user data (profile directory)
playwright-cli delete-data                # delete default browser data
playwright-cli -s=mysession delete-data   # delete named browser data
```

## 환경 변수

환경 변수로 기본 브라우저 세션 이름을 지정할 수 있습니다.

```bash
export PLAYWRIGHT_CLI_SESSION="mysession"
playwright-cli open example.com  # Uses "mysession" automatically
```

## 자주 쓰는 방식

### 여러 사이트의 동시 수집

```bash
#!/bin/bash
# Scrape multiple sites concurrently

# Start all browsers
playwright-cli -s=site1 open https://site1.com &
playwright-cli -s=site2 open https://site2.com &
playwright-cli -s=site3 open https://site3.com &
wait

# Take snapshots from each
playwright-cli -s=site1 snapshot
playwright-cli -s=site2 snapshot
playwright-cli -s=site3 snapshot

# Cleanup
playwright-cli close-all
```

### A/B 테스트 세션

```bash
# Test different user experiences
playwright-cli -s=variant-a open "https://app.com?variant=a"
playwright-cli -s=variant-b open "https://app.com?variant=b"

# Compare
playwright-cli -s=variant-a screenshot
playwright-cli -s=variant-b screenshot
```

### 디스크에 보관하는 프로필

기본 브라우저 프로필은 메모리에만 있습니다. `open`에 `--persistent`를 전달하면 디스크에 보관합니다.

```bash
# Use persistent profile (auto-generated location)
playwright-cli open https://example.com --persistent

# Use persistent profile with custom directory
playwright-cli open https://example.com --profile=/path/to/profile
```

## 실행 중인 브라우저에 연결

새 브라우저를 시작하는 대신 이미 실행 중인 브라우저에 연결하려면 `attach`를 사용합니다.

### 채널 이름으로 연결

실행 중인 Chrome 또는 Edge의 채널 이름으로 연결할 수 있습니다. 대상 브라우저에서 원격 디버깅이 활성화되어 있어야 합니다. 원본은 `chrome://inspect/#remote-debugging`으로 이동하여 “Allow remote debugging for this browser instance” 항목을 켜도록 안내합니다.

```bash
# Attach to Chrome
playwright-cli attach --cdp=chrome

# Attach to Chrome Canary
playwright-cli attach --cdp=chrome-canary

# Attach to Microsoft Edge
playwright-cli attach --cdp=msedge

# Attach to Edge Dev
playwright-cli attach --cdp=msedge-dev
```

지원 채널: `chrome`, `chrome-beta`, `chrome-dev`, `chrome-canary`, `msedge`, `msedge-beta`, `msedge-dev`, `msedge-canary`.

`--session`을 생략하면 채널 이름이 세션 이름이 됩니다. 예를 들어 `--cdp=msedge`는 `msedge` 세션을 생성하므로 Chrome과 Edge에 동시에 연결해도 `default`에서 충돌하지 않습니다. 이름을 직접 정하려면 `--session=<name>`을 사용합니다.

### CDP 엔드포인트로 연결

Chrome DevTools Protocol 엔드포인트를 공개한 브라우저에는 다음처럼 연결합니다.

```bash
playwright-cli attach --cdp=http://localhost:9222
```

### 브라우저 확장 프로그램으로 연결

Playwright 확장 프로그램이 설치된 브라우저에 연결합니다.

```bash
playwright-cli attach --extension
```

### 연결 해제

외부 브라우저를 종료하지 않고 연결한 세션만 정리합니다.

```bash
# Detach the default attached session
playwright-cli detach

# Detach a specific attached session
playwright-cli -s=msedge detach
```

`detach`는 `attach`로 만든 세션에서만 동작합니다. `open`으로 만든 세션에는 `close`를 사용합니다.

## 기본 브라우저 세션

`-s`를 생략하면 명령은 기본 브라우저 세션을 사용합니다.

```bash
# These use the same default browser session
playwright-cli open https://example.com
playwright-cli snapshot
playwright-cli close  # Stops default browser
```

## 브라우저 세션 설정

브라우저를 열 때 세션별 설정을 지정할 수 있습니다.

```bash
# Open with config file
playwright-cli open https://example.com --config=.playwright/my-cli.json

# Open with specific browser
playwright-cli open https://example.com --browser=firefox

# Open in headed mode
playwright-cli open https://example.com --headed

# Open with persistent profile
playwright-cli open https://example.com --persistent
```

## 권장 사용법

### 1. 세션 이름에 목적을 담기

```bash
# GOOD: Clear purpose
playwright-cli -s=github-auth open https://github.com
playwright-cli -s=docs-scrape open https://docs.example.com

# AVOID: Generic names
playwright-cli -s=s1 open https://github.com
```

### 2. 사용이 끝나면 정리하기

```bash
# Stop browsers when done
playwright-cli -s=auth close
playwright-cli -s=scrape close

# Or stop all at once
playwright-cli close-all

# If browsers become unresponsive or zombie processes remain
playwright-cli kill-all
```

### 3. 오래된 브라우저 데이터 삭제

```bash
# Remove old browser data to free disk space
playwright-cli -s=oldsession delete-data
```

## 한국어 예제 해설

이름이 다른 세션은 로그인 쿠키와 저장소를 분리하므로 `auth` 세션에서 로그인해도 `public` 세션의 상태가 같아지지 않습니다. 동시 수집 예제의 `&`는 브라우저 시작을 백그라운드로 보내고 `wait`는 그 시작 명령들이 끝날 때까지 기다립니다. 이후 스냅샷도 각각의 `-s` 이름으로 읽습니다.

`open --persistent`는 디스크 프로필을 만들고, `attach`는 이미 실행 중인 외부 브라우저에 연결합니다. 서로 다른 생명주기이므로 전자는 `close`, 후자는 외부 브라우저를 유지하려면 `detach`를 사용합니다. `delete-data`는 단순 연결 종료와 달리 프로필 데이터를 삭제하는 명령입니다. 원문의 마지막 정리 예제들은 각 명령의 차이를 보여 주며, 실제 사용할 때는 현재 관리 중인 세션의 범위를 확인해야 합니다.

A/B 예제는 URL의 `variant=a/b`를 달리한 별도 세션에서 스크린샷을 비교합니다. `--headed`는 화면이 보이는 브라우저 실행 옵션입니다. 의미 있는 이름(`github-auth`, `docs-scrape`)을 쓰면 무엇을 종료/삭제할지 구분하기 쉽다는 것이 원문 권장 사항입니다.
