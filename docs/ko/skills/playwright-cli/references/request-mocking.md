> **사람용 한국어 번역·해설입니다. ARTEX가 이 문서를 자동으로 로드하지 않습니다.**
> 원본: [references/request-mocking.md](../../../../../skills/playwright-cli/references/request-mocking.md). 실행용 원본 파일은 그대로 유지합니다.
> 본문·제목·제약을 한국어로 옮기고, 명령/코드 블록은 원문 그대로 보존했습니다. 예제 안의 설명은 마지막 한국어 보충 해설에서도 풀어 씁니다.

# 네트워크 요청 모킹

네트워크 요청을 가로채고, 가짜 응답을 제공하거나 수정·차단합니다.

## CLI 라우트 명령

```bash
# Mock with custom status
playwright-cli route "**/*.jpg" --status=404

# Mock with JSON body
playwright-cli route "**/api/users" --body='[{"id":1,"name":"Alice"}]' --content-type=application/json

# Mock with custom headers
playwright-cli route "**/api/data" --body='{"ok":true}' --header="X-Custom: value"

# Remove headers from requests
playwright-cli route "**/*" --remove-header=cookie,authorization

# List active routes
playwright-cli route-list

# Remove a route or all routes
playwright-cli unroute "**/*.jpg"
playwright-cli unroute
```

## URL 패턴

```
**/api/users           - Exact path match
**/api/*/details       - Wildcard in path
**/*.{png,jpg,jpeg}    - Match file extensions
**/search?q=*          - Match query parameters
```

## run-code를 사용한 고급 모킹

조건별 응답, 요청 본문 검사, 실제 응답 수정, 지연 응답에는 다음 방식을 사용합니다.

### 요청 내용에 따라 응답 분기

```bash
playwright-cli run-code "async page => {
  await page.route('**/api/login', route => {
    const body = route.request().postDataJSON();
    if (body.username === 'admin') {
      route.fulfill({ body: JSON.stringify({ token: 'mock-token' }) });
    } else {
      route.fulfill({ status: 401, body: JSON.stringify({ error: 'Invalid' }) });
    }
  });
}"
```

### 실제 응답 수정

```bash
playwright-cli run-code "async page => {
  await page.route('**/api/user', async route => {
    const response = await route.fetch();
    const json = await response.json();
    json.isPremium = true;
    await route.fulfill({ response, json });
  });
}"
```

### 네트워크 장애 모사

```bash
playwright-cli run-code "async page => {
  await page.route('**/api/offline', route => route.abort('internetdisconnected'));
}"
# Options: connectionrefused, timedout, connectionreset, internetdisconnected
```

### 응답 지연

```bash
playwright-cli run-code "async page => {
  await page.route('**/api/slow', async route => {
    await new Promise(r => setTimeout(r, 3000));
    route.fulfill({ body: JSON.stringify({ data: 'loaded' }) });
  });
}"
```

## 한국어 예제 해설

CLI `route`의 첫 예제는 JPG 요청을 404로 만들고, 다음은 JSON 본문/Content-Type 또는 사용자 헤더를 반환합니다. `--remove-header=cookie,authorization`는 요청 헤더를 제거하는 예제입니다. `unroute`에 패턴을 주면 해당 규칙만, 주지 않으면 전체 규칙을 해제합니다.

| 원문 URL 패턴 | 의미 |
|---|---|
| `**/api/users` | 해당 끝 경로에 맞추기 |
| `**/api/*/details` | 경로 중간 한 부분에 와일드카드 사용 |
| `**/*.{png,jpg,jpeg}` | 파일 확장자 여러 개에 맞추기 |
| `**/search?q=*` | 쿼리 매개변수를 포함한 URL에 맞추기 |

조건부 응답 예제는 요청 JSON의 `username`을 읽어 admin이면 토큰, 아니면 401을 반환합니다. 실제 응답 수정은 `route.fetch()`로 먼저 원본을 받고 `json.isPremium`을 바꾼 다음 `fulfill`합니다. 장애 모사는 `abort`에 `connectionrefused`, `timedout`, `connectionreset`, `internetdisconnected` 같은 사유를 사용합니다. 지연 응답 예제는 3초 뒤에 가짜 본문을 반환합니다. 이들은 테스트의 응답 시나리오를 만드는 코드이며 실제 서비스가 해당 결과를 반환했다는 측정 기록은 아닙니다.
