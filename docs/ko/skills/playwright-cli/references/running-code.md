> **사람용 한국어 번역·해설입니다. ARTEX가 이 문서를 자동으로 로드하지 않습니다.**
> 원본: [references/running-code.md](../../../../../skills/playwright-cli/references/running-code.md). 실행용 원본 파일은 그대로 유지합니다.
> 본문·제목·제약을 한국어로 옮기고, 명령/코드 블록은 원문 그대로 보존했습니다. 예제 안의 설명은 마지막 한국어 보충 해설에서도 풀어 씁니다.

# 사용자 정의 Playwright 코드 실행

CLI 명령만으로 표현하기 어려운 고급 작업은 `run-code`로 Playwright 코드를 실행합니다.

## 구문

```bash
playwright-cli run-code "async page => {
  // Your Playwright code here
  // Access page.context() for browser context operations
}"
```

함수를 파일에서 읽어 실행할 수도 있습니다.

```bash
playwright-cli run-code --filename=./my-script.js
```


코드는 하나의 함수 표현식이어야 합니다. 전달한 코드를 `(...)`로 감싼 뒤 평가합니다.
`import`/`export`/`require` 구문은 지원하지 않습니다.

## 위치 정보

```bash
# Grant geolocation permission and set location
playwright-cli run-code "async page => {
  await page.context().grantPermissions(['geolocation']);
  await page.context().setGeolocation({ latitude: 37.7749, longitude: -122.4194 });
}"

# Set location to London
playwright-cli run-code "async page => {
  await page.context().grantPermissions(['geolocation']);
  await page.context().setGeolocation({ latitude: 51.5074, longitude: -0.1278 });
}"

# Clear geolocation override
playwright-cli run-code "async page => {
  await page.context().clearPermissions();
}"
```

## 브라우저 권한

```bash
# Grant multiple permissions
playwright-cli run-code "async page => {
  await page.context().grantPermissions([
    'geolocation',
    'notifications',
    'camera',
    'microphone'
  ]);
}"

# Grant permissions for specific origin
playwright-cli run-code "async page => {
  await page.context().grantPermissions(['clipboard-read'], {
    origin: 'https://example.com'
  });
}"
```

## 미디어 환경 모사

```bash
# Emulate dark color scheme
playwright-cli run-code "async page => {
  await page.emulateMedia({ colorScheme: 'dark' });
}"

# Emulate light color scheme
playwright-cli run-code "async page => {
  await page.emulateMedia({ colorScheme: 'light' });
}"

# Emulate reduced motion
playwright-cli run-code "async page => {
  await page.emulateMedia({ reducedMotion: 'reduce' });
}"

# Emulate print media
playwright-cli run-code "async page => {
  await page.emulateMedia({ media: 'print' });
}"
```

## 대기 전략

```bash
# Wait for network idle
playwright-cli run-code "async page => {
  await page.waitForLoadState('networkidle');
}"

# Wait for specific element
playwright-cli run-code "async page => {
  await page.locator('.loading').waitFor({ state: 'hidden' });
}"

# Wait for function to return true
playwright-cli run-code "async page => {
  await page.waitForFunction(() => window.appReady === true);
}"

# Wait with timeout
playwright-cli run-code "async page => {
  await page.locator('.result').waitFor({ timeout: 10000 });
}"
```

## 프레임과 iframe

```bash
# Work with iframe
playwright-cli run-code "async page => {
  const frame = page.locator('iframe#my-iframe').contentFrame();
  await frame.locator('button').click();
}"

# Get all frames
playwright-cli run-code "async page => {
  const frames = page.frames();
  return frames.map(f => f.url());
}"
```

## 파일 다운로드

```bash
# Handle file download
playwright-cli run-code "async page => {
  const downloadPromise = page.waitForEvent('download');
  await page.getByRole('link', { name: 'Download' }).click();
  const download = await downloadPromise;
  await download.saveAs('./downloaded-file.pdf');
  return download.suggestedFilename();
}"
```

## 클립보드

```bash
# Read clipboard (requires permission)
playwright-cli run-code "async page => {
  await page.context().grantPermissions(['clipboard-read']);
  return await page.evaluate(() => navigator.clipboard.readText());
}"

# Write to clipboard
playwright-cli run-code "async page => {
  await page.evaluate(text => navigator.clipboard.writeText(text), 'Hello clipboard!');
}"
```

## 페이지 정보

```bash
# Get page title
playwright-cli run-code "async page => {
  return await page.title();
}"

# Get current URL
playwright-cli run-code "async page => {
  return page.url();
}"

# Get page content
playwright-cli run-code "async page => {
  return await page.content();
}"

# Get viewport size
playwright-cli run-code "async page => {
  return page.viewportSize();
}"
```

## JavaScript 실행

```bash
# Execute JavaScript and return result
playwright-cli run-code "async page => {
  return await page.evaluate(() => {
    return {
      userAgent: navigator.userAgent,
      language: navigator.language,
      cookiesEnabled: navigator.cookieEnabled
    };
  });
}"

# Pass arguments to evaluate
playwright-cli run-code "async page => {
  const multiplier = 5;
  return await page.evaluate(m => document.querySelectorAll('li').length * m, multiplier);
}"
```

## 오류 처리

```bash
# Try-catch in run-code
playwright-cli run-code "async page => {
  try {
    await page.getByRole('button', { name: 'Submit' }).click({ timeout: 1000 });
    return 'clicked';
  } catch (e) {
    return 'element not found';
  }
}"
```

## 복합 작업 흐름

```bash
# Login and save state
playwright-cli run-code "async page => {
  await page.goto('https://example.com/login');
  await page.getByRole('textbox', { name: 'Email' }).fill('user@example.com');
  await page.getByRole('textbox', { name: 'Password' }).fill('secret');
  await page.getByRole('button', { name: 'Sign in' }).click();
  await page.waitForURL('**/dashboard');
  await page.context().storageState({ path: 'auth.json' });
  return 'Login successful';
}"

# Scrape data from multiple pages
playwright-cli run-code "async page => {
  const results = [];
  for (let i = 1; i <= 3; i++) {
    await page.goto(\`https://example.com/page/\${i}\`);
    const items = await page.locator('.item').allTextContents();
    results.push(...items);
  }
  return results;
}"
```

## 한국어 예제 해설

| 구역 | 코드가 하는 일 |
|---|---|
| 구문 | `page`를 인자로 받는 함수 하나를 실행. 파일도 모듈 전체가 아니라 함수 표현식이어야 함 |
| 위치 | 컨텍스트에 geolocation 권한을 주고 샌프란시스코/런던 좌표를 지정. 마지막 예제는 `clearPermissions()`로 권한 재정의를 해제 |
| 권한 | 위치·알림·카메라·마이크 권한을 함께 지정하거나 특정 origin의 클립보드 읽기를 허용 |
| 미디어 | dark/light, 동작 축소, print 매체를 모사하여 반응형 스타일 확인 |
| 대기 | 네트워크 상태, 로딩 요소가 숨겨짐, `appReady`, 결과 요소의 제한시간을 기다리는 각각의 예제 |
| iframe | 해당 iframe의 contentFrame에서 버튼을 찾거나 모든 frame URL을 조회 |
| 다운로드 | 클릭 전에 download 이벤트를 기다릴 Promise를 등록하고, 클릭 뒤 파일을 저장하여 이벤트 경쟁 방지 |
| 클립보드 | 읽기 권한을 지정한 뒤 값을 읽거나 evaluate 인자로 텍스트를 넘겨 쓰기 |
| 페이지 정보 | 제목, URL, HTML 본문, viewport 크기 반환 |
| evaluate | 브라우저 문맥에서 userAgent/언어/쿠키 지원을 읽거나 외부 인자를 전달 |
| 오류 | 1초 안에 Submit 버튼을 클릭하지 못하면 catch에서 대체 결과 반환 |
| 로그인 흐름 | 로그인 이동→이메일/비밀번호 입력→Sign in→dashboard 확인→상태 파일 저장 |
| 다중 페이지 | 1~3 페이지를 순회하며 `.item` 텍스트를 한 결과 배열에 합치기 |

**원문 사이의 적용 범위:** 이 문서에는 `networkidle` 대기 예제가 있지만 [test-generation.md](test-generation.md)의 실패 복구 절은 `networkidle` 사용을 금지합니다. 두 원문을 모두 보존했습니다. 테스트 복구 지침을 적용하는 문맥에서는 요소의 관찰 가능한 상태나 검증문을 기준으로 대기하는 해당 절의 규칙을 함께 읽어야 합니다.
