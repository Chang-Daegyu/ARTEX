> **사람용 한국어 번역·해설입니다. ARTEX가 이 문서를 자동으로 로드하지 않습니다.**
> 원본: [references/test-generation.md](../../../../../skills/playwright-cli/references/test-generation.md). 실행용 원본 파일은 그대로 유지합니다.
> 본문·제목·제약을 한국어로 옮기고, 명령/코드 블록은 원문 그대로 보존했습니다. 예제 안의 설명은 마지막 한국어 보충 해설에서도 풀어 씁니다.

# 테스트 계획 → 생성 → 복구

`playwright-cli`로 Playwright 테스트를 작성하고 유지하는 전체 작업 흐름입니다. `playwright-cli`의 각 조작은 대응하는 Playwright TypeScript를 출력하며 이 코드가 테스트의 재료가 됩니다. 아래 구역은 각각 독립적으로 사용할 수도 있습니다.

- **생성 원리** — 조작을 TypeScript로 바꾸는 공통 원리와 검증문 추가 방법입니다.
- **계획** — 앱을 탐색하고 무엇을 검증할지 명세 파일에 기록합니다.
- **생성** — 명세를 Playwright 테스트 파일로 옮깁니다. 모호하거나 오래된 명세는 갱신합니다.
- **복구** — 실패한 테스트를 진단하고 코드를 수정하며 명세와 실제 동작을 맞춥니다.

계획·생성·복구는 같은 연결 방식을 사용합니다. `npx playwright test --debug=cli`를 백그라운드에서 실행하고 `playwright-cli attach tw-XXXX`로 일시 중지된 페이지에 연결합니다. 디버그/연결 방식은 [playwright-tests.md](playwright-tests.md)를 참고합니다.

---

## 0. 생성 원리

`playwright-cli`에서 수행하는 각 조작은 대응하는 Playwright TypeScript 코드를 생성합니다. 출력된 코드를 테스트 파일에 바로 복사할 수 있습니다.

```bash
# Start a session
playwright-cli open https://example.com/login

# Take a snapshot to see elements
playwright-cli snapshot
# Output shows: e1 [textbox "Email"], e2 [textbox "Password"], e3 [button "Sign In"]

# Fill form fields - generates code automatically
playwright-cli fill e1 "user@example.com"
# Ran Playwright code:
# await page.getByRole('textbox', { name: 'Email' }).fill('user@example.com');

playwright-cli fill e2 "password123"
# Ran Playwright code:
# await page.getByRole('textbox', { name: 'Password' }).fill('password123');

playwright-cli click e3
# Ran Playwright code:
# await page.getByRole('button', { name: 'Sign In' }).click();
```

### 테스트 파일 구성

생성된 코드를 다음처럼 Playwright 테스트에 모읍니다.

```typescript
import { test, expect } from '@playwright/test';

test('login flow', async ({ page }) => {
  // Generated code from playwright-cli session:
  await page.goto('https://example.com/login');
  await page.getByRole('textbox', { name: 'Email' }).fill('user@example.com');
  await page.getByRole('textbox', { name: 'Password' }).fill('password123');
  await page.getByRole('button', { name: 'Sign In' }).click();

  // Add assertions
  await expect(page).toHaveURL(/.*dashboard/);
});
```

### 의미 기반 locator 사용

생성 코드는 가능하면 역할 기반 locator를 사용합니다. 원본은 이를 더 안정적인 선택 방식으로 권장합니다.

```typescript
// Generated (good - semantic)
await page.getByRole('button', { name: 'Submit' }).click();

// Avoid (fragile - CSS selectors)
await page.locator('#submit-btn').click();
```

### 기록 전에 페이지 탐색

조작을 기록하기 전에 스냅샷으로 페이지 구조를 확인합니다.

```bash
playwright-cli open https://example.com
playwright-cli snapshot
# Review the element structure
playwright-cli click e5
```

### 검증문 직접 추가

생성 코드는 조작을 기록하지만 검증문까지 자동으로 만들지는 않습니다. 테스트에는 다음 matcher 등으로 기대 결과를 명시합니다.

- `toBeVisible()` — 요소가 렌더링되어 보이는지 확인합니다.
- `toHaveText(text)` — 요소의 텍스트가 일치하는지 확인합니다.
- `toHaveValue(value) / toBeEmpty()` — 입력/select 값 또는 빈 상태를 확인합니다.
- `toBeChecked() / toBeUnchecked()` — 체크박스 선택 상태를 확인합니다.
- `toMatchAriaSnapshot(snapshot)` — 페이지 또는 locator가 부분 접근성 스냅샷과 일치하는지 확인합니다.

`playwright-cli generate-locator <target>`으로 검증문에 쓸 locator 표현식을 만들고, snapshot/eval 명령으로 기대값을 확인합니다.

텍스트 내용을 검증할 때 locator 자체에 검증할 요소의 텍스트를 넣지 않도록 합니다. `getByTestId()`나 `getByLabel()`은 텍스트 검증과 함께 사용하기 좋습니다. 텍스트로 찾는 locator라면 `toBeVisible()`을 우선 고려합니다.

검증할 스냅샷에 페이지의 모든 정보를 넣을 필요는 없습니다. 검증에 필요한 부분만 포함하고 바뀔 수 있는 값은 정규식으로 표현할 수 있습니다.

```bash
# Get a stable locator for an element ref to use in the assertion
playwright-cli --raw generate-locator e5
# getByRole('button', { name: 'Submit' })

# Capture expected text content for toHaveText
playwright-cli --raw eval "el => el.textContent" e5

# Capture expected input value for toHaveValue/toBeEmpty
playwright-cli --raw eval "el => el.value" e5

# Capture expected aria snapshot for toMatchAriaSnapshot/toBeChecked
# (whole page, or use a ref to scope to a region)
playwright-cli --raw snapshot
playwright-cli --raw snapshot e5
```

```typescript
// Generated action
await page.getByRole('button', { name: 'Submit' }).click();

// Manual assertions using the outputs above:
await expect(page.getByRole('alert', { name: 'Success' })).toBeVisible();
await expect(page.getByTestId('main-header')).toHaveText('Welcome, user');
await expect(page.getByRole('textbox', { name: 'Email' })).toHaveValue('user@example.com');
await expect(page.getByRole('checkbox', { name: 'Enable notifications' })).toBeChecked();

// toMatchAriaSnapshot on the whole page, finds a matching region
await expect(page).toMatchAriaSnapshot(`
  - heading "Welcome, user"
  - link /\\d+ new messages?/
  - button "Sign out"
`);

// toMatchAriaSnapshot scoped to a region
await expect(page.getByRole('navigation')).toMatchAriaSnapshot(`
  - link "Home"
  - link /\\d+ new messages?/
  - link "Profile"
`);
```

---

## 1. 계획

목표는 테스트할 시나리오를 나열한 명세 파일, 예를 들어 `specs/<feature>.plan.md`를 만드는 것입니다. 원본은 명세를 **반드시 파일로 작성**하도록 안내합니다.

### 1.1 선행 조건: 작업 공간

먼저 작업 공간에 Playwright가 설치되어 있는지 확인합니다.

```bash
# Either of these confirms a workspace:
test -f playwright.config.ts || test -f playwright.config.js
npx --no-install playwright --version
```

Playwright가 없다면 초기화를 실행하고 사용자가 기본 옵션을 선택하도록 합니다.

```bash
npm init playwright@latest
```

### 1.2 선행 조건: 시작점이 되는 seed 테스트

**seed 테스트**는 모든 시나리오가 시작할 상태로 페이지를 만드는 최소 테스트입니다. 앱 이동, 필요한 로그인, 기능 플래그 등을 포함합니다. 각 시나리오는 seed가 끝난 뒤의 새 상태에서 출발한다고 가정합니다. `--debug=cli`는 이 테스트 안에서 멈추므로 계획과 생성 세션의 시작점이 됩니다.

최소 seed 예제:

```ts
// tests/seed.spec.ts
import { test } from '@playwright/test';

test('seed', async ({ page }) => {
  await page.goto('https://example.com/');
});
```

여러 시나리오가 재사용할 수 있도록 페이지 이동을 fixture에 넣는 방식을 권장합니다.

```ts
// tests/fixtures.ts
import { test as baseTest } from '@playwright/test';
export { expect } from '@playwright/test';

export const test = baseTest.extend({
  page: async ({ page }, use) => {
    await page.goto('https://example.com/');
    await use(page);
  },
});
```

```ts
// tests/seed.spec.ts
import { test } from './fixtures';

test('seed', async ({ page }) => {
  // Fixture already navigates. This empty body tells agents where to start.
});
```

seed가 없다면 적어도 앱으로 이동하는 seed 테스트를 만듭니다.

### 1.3 앱 탐색

seed를 통해 앱을 백그라운드에서 실행하고 연결합니다.

```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test tests/seed.spec.ts --debug=cli
# wait for "Debugging Instructions" and the session name tw-XXXX
playwright-cli attach tw-XXXX
```

seed가 실행되도록 재개한 다음 앱을 살펴봅니다.

```bash
playwright-cli resume                   # resume so that seed test runs fully
playwright-cli snapshot                 # inventory of interactive elements
playwright-cli click e5                 # follow a flow
playwright-cli eval "location.href"     # read URL / state
playwright-cli show --annotate          # ask the user to point at something
```

다음 항목을 정리합니다.

- 조작 가능한 화면 요소: 폼, 버튼, 목록, 필터, 모달.
- 주요 사용자 작업의 시작부터 끝까지의 흐름.
- 경계 사례: 빈 상태, 검증 오류, 매우 긴 입력, 경계값.
- 지속성: 새로고침, local/session storage, URL fragment.
- 탐색: 어떤 조작이 URL을 바꾸는지, 뒤로/앞으로 이동의 동작.

**중요:** playwright-cli로 앱 URL만 바로 여는 대신 항상 테스트를 거쳐야 합니다. 테스트에 있는 사용자 정의 초기화가 적용되기 때문입니다.
**중요:** 탐색을 마치면 백그라운드 테스트를 중지합니다.

### 1.4 명세 파일 작성

`specs/<feature>.plan.md` 아래에 다음 구조로 저장합니다.

```markdown
# <Feature> Test Plan

## Application Overview

<One paragraph describing what the feature does and why it matters.>

## Test Scenarios

### 1. <Group Name>

**Seed:** `tests/seed.spec.ts`

#### 1.1. <kebab-case-scenario-name>

**File:** `tests/<group>/<kebab-case-scenario-name>.spec.ts`

**Steps:**
  1. <Concrete user step>
    - expect: <observable outcome>
    - expect: <another observable outcome>
  2. <Next step>
    - expect: <outcome>

#### 1.2. <next-scenario>
...

### 2. <Next Group>

**Seed:** `tests/seed.spec.ts`
...
```

작성 지침:

- 각 시나리오는 독립적이며 seed의 새 상태에서 시작합니다. 앞 시나리오의 결과에 이어 붙이지 않습니다.
- 시나리오 이름은 kebab-case로 쓰고 파일명과 맞춥니다. 예: `should-add-single-todo` → `should-add-single-todo.spec.ts`.
- 정상 흐름, 경계 사례, 입력 검증, 실패/거부 흐름, 지속성을 포함합니다.
- “`fill` 호출” 같은 API 수준 대신 “입력창에 Buy milk 입력”처럼 사용자 동작 수준으로 단계를 씁니다.
- 관찰할 수 있는 결과를 `- expect:` 항목으로 기록합니다. 생성 단계에서 각각이 검증문이 됩니다.

---

## 2. 생성

목표는 명세 파일에서 Playwright 테스트 파일을 만드는 것입니다. 실제 앱과 차이가 생긴 명세는 필요하면 갱신합니다.

### 2.1 입력 자료

- **명세 파일:** 예를 들어 `specs/basic-operations.plan.md`.
- **대상:** 시나리오 하나(`1.2`), 그룹 전체(`1`), 또는 전체.
- **seed 파일:** 해당 시나리오 그룹의 `**Seed:**` 줄에서 읽습니다.

### 2.2 시나리오 하나 생성

각 대상 시나리오를 순서대로 처리합니다. 이 구역에서 원본은 seed 세션을 공유하므로 병렬 실행하지 말라고 안내합니다.

```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test <seed-file> --debug=cli   # background
playwright-cli attach tw-XXXX
# resume
```

playwright-cli로 앱 URL만 바로 열지 말고 **항상 테스트를 거쳐** 사용자 정의 초기화가 적용되게 합니다.

명세를 계획으로 삼고 실제 앱을 현재 동작의 기준으로 삼아 `Steps:`를 하나씩 playwright-cli로 수행합니다. “버튼 클릭”처럼 대상이 모호하거나 요소가 사라졌거나 실제 동작과 모순되면 판단하여 명세를 실제 동작에 맞추고 계속 진행합니다. 원본은 생성 중 명세를 고치는 것을 예상된 과정으로 설명합니다.

각 동작은 대응하는 Playwright TypeScript를 출력합니다. [생성 원리](#0-생성-원리)를 참고합니다.

```bash
playwright-cli snapshot                         # find refs
playwright-cli fill e3 "John Doe"               # -> page.getByRole('textbox', {...}).fill(...)
playwright-cli press Enter
playwright-cli click e7
```

모든 `- expect:` 항목에 명시적인 검증문을 추가합니다. 자세한 내용은 [생성 원리](#0-생성-원리)를 참고합니다.

생성된 코드를 모아 명세에서 지정한 경로에 테스트 파일을 작성합니다.

```ts
// spec: specs/basic-operations.plan.md
// seed: tests/seed.spec.ts
import { test, expect } from './fixtures';   // or '@playwright/test' if no fixtures file

test.describe('Signing in and out', () => {
  test('should sign in', async ({ page }) => {
    // 1. Navigate to the application
    // (handled by the seed fixture)

    // 2. Type 'John Doe' into the username field
    await page.getByRole('textbox', { name: 'username' }).fill('John Doe');

    // 3. Type password
    await page.getByRole('textbox', { name: 'password' }).fill('TestPassword');

    // 4. Press Enter to submit
    await page.getByRole('textbox', { name: 'password' }).press('Enter');

    await expect(page.getByRole('heading')).toContainText('Welcome, John Doe!');
  });
});
```

규칙:

- **파일 하나에 테스트 하나**를 작성합니다. 파일 경로, describe 이름, test 이름은 순번을 제외하고 명세 그대로 사용합니다.
- 각 번호 단계의 동작 앞에 `// N. <step text>` 주석을 붙입니다.
- describe 그룹 이름은 `1.` 같은 순번 없이 명세의 이름을 그대로 사용합니다.
- 프로젝트에 fixture가 있으면 `./fixtures`에서, 없으면 `@playwright/test`에서 가져옵니다.
- **중요:** 다음 시나리오로 넘어가기 전에 CLI 세션을 닫고 백그라운드 테스트를 중지합니다.

### 2.3 여러 시나리오 생성

대상 시나리오마다 2.2를 반복하고, 매번 seed를 다시 시작해 깨끗한 페이지에서 출발하게 합니다. 이 문단의 원문은 고유한 생성 세션 이름을 쓰면 병렬화도 안전하다고 설명하며, 각 테스트 실행을 반드시 중지하라고 덧붙입니다. 세션을 공유하는 2.2의 순차 실행 안내와 함께 아래 번역자 해설을 참고합니다.

### 2.4 생성한 테스트 실행

생성 후 새 테스트를 한 번 실행합니다.

```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test tests/<group>/<scenario>.spec.ts
```

실패하면 3절로 진행합니다.

---

## 3. 실패 복구

목표는 실패한 테스트를 수정하고, 앱의 의도된 동작이 바뀌었다면 명세도 갱신하는 것입니다.

### 3.1 실패한 테스트 찾기

```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test
```

실패한 `<file>:<line>` 목록을 기록하고 하나씩 처리합니다. 공유 상태와 단일 CLI 세션으로 인해 불안정해질 수 있으므로 병렬 수정은 하지 않습니다.

### 3.2 실패 하나 디버깅

실패한 테스트 하나를 백그라운드 디버그 모드로 실행한 뒤 연결합니다.

```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test tests/<group>/<scenario>.spec.ts:<line> --debug=cli
# wait for "Debugging Instructions" and the tw-XXXX session name
playwright-cli attach tw-XXXX
```

테스트는 시작 위치에서 멈춰 있습니다. 단계별로 진행하거나 실패 동작/검증 바로 전까지 실행한 뒤 진단합니다.

```bash
playwright-cli snapshot                # did the element change / move / rename?
playwright-cli console                 # app-side errors?
playwright-cli requests                # failed request? wrong payload?
playwright-cli show --annotate         # ask the user to point somewhere
```

흔한 원인은 선택자 변화, 새 래퍼 요소, label/ARIA 이름 변경, 전환/비동기 로드의 시점, 앱의 표시 문구 변경, 실행 간 테스트 데이터 누수입니다.

playwright-cli로 수정한 동작을 다시 수행하고 출력되는 코드를 테스트에 반영합니다.

### 3.3 수정 적용

테스트 파일의 locator, 검증문, 단계 순서, 입력값을 올바른 동작에 맞춥니다. 백그라운드 디버그 실행을 중지하고 해당 테스트를 다시 실행해 통과를 확인합니다.

해결책으로 훅을 건너뛰거나 임의의 sleep을 추가하지 않습니다. 이 실패 복구 절의 원문은 `networkidle` 사용도 금지합니다.

### 3.4 명세와 맞추기

테스트 파일의 `// spec:` 헤더가 가리키는 명세를 열고 해당 시나리오를 찾습니다.

- **기술적인 수정만 한 경우:** locator 변화 또는 더 나은 검증 방식이며 명세의 사용자 동작은 여전히 앱과 같다면 명세를 유지합니다.
- **사용자에게 보이는 단계·입력·순서·기대 결과를 바꾼 경우:** 명세를 실제 동작에 맞춥니다. 시나리오 ID와 파일 경로는 유지하고 step/expect 줄만 바꿉니다.
- **의도된 앱 변경으로 명세가 오래된 것인지, 앱의 회귀 버그로 테스트가 맞는 것인지 불명확한 경우:** 원문은 **중단하고 사용자에게 확인**하도록 안내합니다. 다음을 제공합니다.
  - 시나리오 ID, 예: `2.3`.
  - 더 이상 맞지 않는 명세 줄.
  - 관찰한 앱 동작: 스냅샷의 짧은 부분 또는 구체적인 결과.

사용자의 답을 받은 뒤 의도된 변경이면 명세를 갱신하고, 회귀 버그라면 해당 테스트가 버그를 검증하고 있음을 이슈/표시로 남깁니다.

### 3.5 반복과 보류 처리

- 실패는 하나씩 고치고 매번 다시 실행합니다.
- 충분히 조사한 뒤 테스트가 맞고 앱이 잘못되었다고 확신하며 사용자가 버그임을 확인한 경우, 사용자 결정이나 이슈 링크를 주석에 남기고 `test.fixme(...)`로 표시합니다. 조용히 건너뛰지 않습니다.

---

## 관련 문서

| 주제 | 참고 |
|---|---|
| `--debug=cli`와 attach 연결 방식 | [playwright-tests.md](playwright-tests.md) |
| 탐색/생성 중 요청 모킹 | [request-mocking.md](request-mocking.md) |
| CLI 브라우저 세션 관리 | [session-management.md](session-management.md) |

## 번역자 해설: 원문에서 구분해서 읽을 지점

이 문서의 명령, TypeScript, Markdown 명세 예제는 원문 그대로 보존했습니다. 예제 안의 `Seed`는 모든 시나리오의 공통 초기 상태를 만드는 테스트/fixture, `Steps`는 사용자가 수행하는 순서, `expect`는 확인할 관찰 결과입니다. `spec:`/`seed:` 주석은 테스트에서 해당 계획과 시작점을 다시 찾는 연결 정보입니다.

생성 원리는 **조작 기록 + 수동 검증문**입니다. 버튼을 클릭했다는 코드만 있어서는 기대한 화면 변화까지 확인하지 못합니다. 텍스트 자체로 요소를 찾은 뒤 같은 텍스트를 다시 검증하면 검증이 약해질 수 있어, 원문은 고정 test ID 또는 label과 실제 텍스트 기대값을 분리하도록 안내합니다.

원문 2.2는 seed 세션을 공유하므로 순차 실행을 요구하고, 2.3은 고유 세션 이름의 테스트 실행을 병렬화할 수 있다고 설명합니다. 이 번역에서는 두 문장을 삭제하거나 하나의 새로운 규칙으로 바꾸지 않았습니다. 읽는 관점에서 **같은 세션을 공유하는 조작**과 **완전히 분리한 실행 세션**의 구분이 필요합니다. 실제 자동화에서는 초기 데이터·계정·파일 경로도 공유될 수 있으므로 세션 이름만 다르다는 사실로 모든 독립성이 보장되지는 않습니다.

생성 중 명세를 바꿀 수 있다는 2.2의 안내도, 회귀인지 의도된 변경인지 불명확하면 사용자에게 확인하라는 3.4와 함께 읽어야 합니다. 현재 관찰한 동작을 무조건 정답으로 바꾸어 테스트를 통과시키라는 의미로 사용하면 안 됩니다.

실패 복구에서는 임의 sleep과 `networkidle`을 금지하지만 [영상 녹화](video-recording.md)는 사람이 보기 좋은 시연을 위해 장면 간 표시 시간을 넣습니다. 기능 검증의 대기 조건과 영상 연출의 시간은 목적이 다릅니다. [running-code.md](running-code.md)의 대기 예제와도 적용 문맥을 구분합니다.
