> **사람용 한국어 번역·해설입니다. ARTEX가 이 문서를 자동으로 로드하지 않습니다.**
> 원본: [references/storage-state.md](../../../../../skills/playwright-cli/references/storage-state.md). 실행용 원본 파일은 그대로 유지합니다.
> 본문·제목·제약을 한국어로 옮기고, 명령/코드 블록은 원문 그대로 보존했습니다. 예제 안의 설명은 마지막 한국어 보충 해설에서도 풀어 씁니다.

# 브라우저 저장소 관리

쿠키, localStorage, sessionStorage와 브라우저 저장 상태를 관리합니다.

## 저장 상태

쿠키와 저장소를 포함한 브라우저 상태를 저장하고 다시 불러옵니다.

### 저장 상태 내보내기

```bash
# Save to auto-generated filename (storage-state-{timestamp}.json)
playwright-cli state-save

# Save to specific filename
playwright-cli state-save my-auth-state.json
```

### 저장 상태 복원

```bash
# Load storage state from file
playwright-cli state-load my-auth-state.json

# Reload page to apply cookies
playwright-cli open https://example.com
```

### 저장 상태 파일 형식

저장 파일은 다음과 같은 내용을 담습니다.

```json
{
  "cookies": [
    {
      "name": "session_id",
      "value": "abc123",
      "domain": "example.com",
      "path": "/",
      "expires": 1893456000,
      "httpOnly": true,
      "secure": true,
      "sameSite": "Lax"
    }
  ],
  "origins": [
    {
      "origin": "https://example.com",
      "localStorage": [
        { "name": "theme", "value": "dark" },
        { "name": "user_id", "value": "12345" }
      ]
    }
  ]
}
```

## 쿠키

### 모든 쿠키 조회

```bash
playwright-cli cookie-list
```

### 도메인으로 쿠키 필터링

```bash
playwright-cli cookie-list --domain=example.com
```

### 경로로 쿠키 필터링

```bash
playwright-cli cookie-list --path=/api
```

### 특정 쿠키 조회

```bash
playwright-cli cookie-get session_id
```

### 쿠키 설정

```bash
# Basic cookie
playwright-cli cookie-set session abc123

# Cookie with options
playwright-cli cookie-set session abc123 --domain=example.com --path=/ --httpOnly --secure --sameSite=Lax

# Cookie with expiration (Unix timestamp)
playwright-cli cookie-set remember_me token123 --expires=1893456000
```

### 쿠키 삭제

```bash
playwright-cli cookie-delete session_id
```

### 모든 쿠키 비우기

```bash
playwright-cli cookie-clear
```

### 고급: 여러 쿠키 또는 사용자 옵션

여러 쿠키를 한 번에 추가하는 등 복잡한 경우에는 `run-code`를 사용합니다.

```bash
playwright-cli run-code "async page => {
  await page.context().addCookies([
    { name: 'session_id', value: 'sess_abc123', domain: 'example.com', path: '/', httpOnly: true },
    { name: 'preferences', value: JSON.stringify({ theme: 'dark' }), domain: 'example.com', path: '/' }
  ]);
}"
```

## LocalStorage

### 모든 localStorage 항목 조회

```bash
playwright-cli localstorage-list
```

### 값 하나 조회

```bash
playwright-cli localstorage-get token
```

### 값 설정

```bash
playwright-cli localstorage-set theme dark
```

### JSON 값 설정

```bash
playwright-cli localstorage-set user_settings '{"theme":"dark","language":"en"}'
```

### 항목 하나 삭제

```bash
playwright-cli localstorage-delete token
```

### 모든 localStorage 비우기

```bash
playwright-cli localstorage-clear
```

### 고급: 여러 작업 수행

여러 값을 한 번에 지정하는 등 복잡한 경우에는 `run-code`를 사용합니다.

```bash
playwright-cli run-code "async page => {
  await page.evaluate(() => {
    localStorage.setItem('token', 'jwt_abc123');
    localStorage.setItem('user_id', '12345');
    localStorage.setItem('expires_at', Date.now() + 3600000);
  });
}"
```

## SessionStorage

### 모든 sessionStorage 항목 조회

```bash
playwright-cli sessionstorage-list
```

### 값 하나 조회

```bash
playwright-cli sessionstorage-get form_data
```

### 값 설정

```bash
playwright-cli sessionstorage-set step 3
```

### 항목 하나 삭제

```bash
playwright-cli sessionstorage-delete step
```

### sessionStorage 비우기

```bash
playwright-cli sessionstorage-clear
```

## IndexedDB

### 데이터베이스 목록

```bash
playwright-cli run-code "async page => {
  return await page.evaluate(async () => {
    const databases = await indexedDB.databases();
    return databases;
  });
}"
```

### 데이터베이스 삭제

```bash
playwright-cli run-code "async page => {
  await page.evaluate(() => {
    indexedDB.deleteDatabase('myDatabase');
  });
}"
```

## 자주 쓰는 방식

### 로그인 상태 재사용

```bash
# Step 1: Login and save state
playwright-cli open https://app.example.com/login
playwright-cli snapshot
playwright-cli fill e1 "user@example.com"
playwright-cli fill e2 "password123"
playwright-cli click e3

# Save the authenticated state
playwright-cli state-save auth.json

# Step 2: Later, restore state and skip login
playwright-cli state-load auth.json
playwright-cli open https://app.example.com/dashboard
# Already logged in!
```

### 저장 후 복원하기

```bash
# Set up authentication state
playwright-cli open https://example.com
playwright-cli eval "() => { document.cookie = 'session=abc123'; localStorage.setItem('user', 'john'); }"

# Save state to file
playwright-cli state-save my-session.json

# ... later, in a new session ...

# Restore state
playwright-cli state-load my-session.json
playwright-cli open https://example.com
# Cookies and localStorage are restored!
```

## 보안 관련 원문 안내

- 인증 토큰이 들어 있는 저장 상태 파일을 커밋하지 않습니다.
- `.gitignore`에 `*.auth-state.json`을 추가합니다.
- 자동화가 끝나면 상태 파일을 삭제합니다.
- 민감한 값은 환경 변수를 사용합니다.
- 기본 세션은 메모리 모드로 실행되며 원문은 이를 민감한 작업에 더 적합한 기본값으로 설명합니다.

## 한국어 예제 해설

저장 파일 예제는 `cookies`와 origin별 `localStorage`를 보여 줍니다. 쿠키에는 domain/path/만료 시각/httpOnly/secure/sameSite가 있고, localStorage는 각 origin 아래 이름/값 목록입니다. `state-save`/`state-load`가 모든 종류의 브라우저 데이터나 세션 상태를 무조건 복제한다고 확대 해석하지 말고, 원문 예제의 저장 형식과 사용하는 CLI 버전의 실제 지원 범위를 구분해야 합니다.

쿠키의 만료 예제는 Unix 시각을 쓰고, `run-code`의 여러 쿠키 예제는 같은 컨텍스트에 두 쿠키를 추가합니다. localStorage의 JSON 예제는 JSON 객체 자체가 아니라 직렬화한 문자열을 값으로 저장합니다. sessionStorage 명령은 별도의 저장소를 대상으로 합니다. IndexedDB는 `run-code`와 페이지의 JavaScript로 목록을 읽거나 지정 데이터베이스를 삭제합니다.

인증 재사용 예제의 흐름은 로그인→상태 저장→나중에 불러오기→로그인 이후 화면 이동입니다. 마지막 왕복 예제는 쿠키와 localStorage 값을 직접 넣은 뒤 저장/복원하여 형식을 이해하도록 합니다. 예제의 `abc123`, `jwt_abc123`, `password123`은 설명용 문자열입니다. 원문의 보안 안내는 위에 그대로 한국어로 옮겼습니다.
