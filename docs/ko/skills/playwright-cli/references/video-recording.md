> **사람용 한국어 번역·해설입니다. ARTEX가 이 문서를 자동으로 로드하지 않습니다.**
> 원본: [references/video-recording.md](../../../../../skills/playwright-cli/references/video-recording.md). 실행용 원본 파일은 그대로 유지합니다.
> 본문·제목·제약을 한국어로 옮기고, 명령/코드 블록은 원문 그대로 보존했습니다. 예제 안의 설명은 마지막 한국어 보충 해설에서도 풀어 씁니다.

# 영상 녹화

디버깅, 문서화, 검증을 위해 브라우저 자동화 세션을 영상으로 기록합니다. 원문은 WebM 형식과 VP8/VP9 코덱을 사용한다고 설명합니다.

## 기본 녹화

```bash
# Open browser first
playwright-cli open

# Start recording
playwright-cli video-start demo.webm

# Add a chapter marker for section transitions
playwright-cli video-chapter "Getting Started" --description="Opening the homepage" --duration=2000

# Navigate and perform actions
playwright-cli goto https://example.com
playwright-cli snapshot
playwright-cli click e1

# Add another chapter
playwright-cli video-chapter "Filling Form" --description="Entering test data" --duration=2000
playwright-cli fill e2 "test input"

# Stop and save
playwright-cli video-stop
```

## 권장 사용법

### 1. 내용을 설명하는 파일명 사용

```bash
# Include context in filename
playwright-cli video-start recordings/login-flow-2024-01-15.webm
playwright-cli video-start recordings/checkout-test-run-42.webm
```

### 2. 보여 주려는 전체 시나리오를 스크립트로 기록

사용자에게 보여 주거나 작업 수행을 증명하는 영상을 만들 때 원문은 코드 조각을 작성해 run-code로 실행하는 방식을 권장합니다.
이렇게 하면 동작 사이에 적절한 표시 시간을 두고 영상에 설명을 붙일 수 있습니다. 아래 예제는 이를 위한 Playwright API를 사용합니다.

1. 먼저 CLI로 시나리오를 수행하며 모든 locator와 동작을 기록합니다. 강조할 영역의 bounding box를 얻을 때 해당 locator가 필요합니다.
2. 아래 예제처럼 영상용 스크립트 파일을 만듭니다. 자연스러운 타이핑에는 지연을 둔 `pressSequentially`를 사용하고 보기 적절한 간격을 넣습니다.
3. `playwright-cli run-code --filename your-script.js`로 실행합니다.

**중요:** 오버레이는 `pointer-events: none`이므로 페이지 조작을 가로막지 않습니다. 설명 오버레이를 계속 보여 주면서 클릭·입력 등 다른 동작을 수행할 수 있습니다.

```js
async page => {
  await page.screencast.start({ path: 'video.webm', size: { width: 1280, height: 800 } });
  await page.goto('https://demo.playwright.dev/todomvc');

  // Show a chapter card — blurs the page and shows a dialog.
  // Blocks until duration expires, then auto-removes.
  // Use this for simple use cases, but always feel free to hand-craft your own beautiful
  // overlay via await page.screencast.showOverlay().
  await page.screencast.showChapter('Adding Todo Items', {
    description: 'We will add several items to the todo list.',
    duration: 2000,
  });

  // Perform action
  await page.getByRole('textbox', { name: 'What needs to be done?' }).pressSequentially('Walk the dog', { delay: 60 });
  await page.getByRole('textbox', { name: 'What needs to be done?' }).press('Enter');
  await page.waitForTimeout(1000);

  // Show next chapter
  await page.screencast.showChapter('Verifying Results', {
    description: 'Checking the item appeared in the list.',
    duration: 2000,
  });

  // Add a sticky annotation that stays while you perform actions.
  // Overlays are pointer-events: none, so they won't block clicks.
  const annotation = await page.screencast.showOverlay(`
    <div style="position: absolute; top: 8px; right: 8px;
      padding: 6px 12px; background: rgba(0,0,0,0.7);
      border-radius: 8px; font-size: 13px; color: white;">
      ✓ Item added successfully
    </div>
  `);

  // Perform more actions while the annotation is visible
  await page.getByRole('textbox', { name: 'What needs to be done?' }).pressSequentially('Buy groceries', { delay: 60 });
  await page.getByRole('textbox', { name: 'What needs to be done?' }).press('Enter');
  await page.waitForTimeout(1500);

  // Remove the annotation when done
  await annotation.dispose();

  // You can also highlight relevant locators and provide contextual annotations.
  const bounds = await page.getByText('Walk the dog').boundingBox();
  await page.screencast.showOverlay(`
    <div style="position: absolute;
      top: ${bounds.y}px;
      left: ${bounds.x}px;
      width: ${bounds.width}px;
      height: ${bounds.height}px;
      border: 1px solid red;">
    </div>
    <div style="position: absolute;
      top: ${bounds.y + bounds.height + 5}px;
      left: ${bounds.x + bounds.width / 2}px;
      transform: translateX(-50%);
      padding: 6px;
      background: #808080;
      border-radius: 10px;
      font-size: 14px;
      color: white;">Check it out, it is right above this text
    </div>
  `, { duration: 2000 });

  await page.screencast.stop();
}
```

오버레이를 이용하면 장면 설명과 강조를 다양하게 구성할 수 있습니다.

### 오버레이 API 요약

| 메서드 | 용도 |
|--------|----------|
| `page.screencast.showChapter(title, { description?, duration?, styleSheet? })` | 배경을 흐리게 하는 전체 화면 장면 제목 카드. 구간 전환에 사용 |
| `page.screencast.showOverlay(html, { duration? })` | 사용자 정의 HTML 오버레이. 설명, 이름표, 강조에 사용 |
| `disposable.dispose()` | duration 없이 추가한 지속 오버레이 제거 |
| `page.screencast.hideOverlays()` / `page.screencast.showOverlays()` | 모든 오버레이를 잠시 숨기거나 다시 표시 |

## trace와 영상 비교

| 특성 | 영상 | Trace |
|---------|-------|---------|
| 출력 | WebM 파일 | Trace Viewer에서 읽는 추적 파일 |
| 보여 주는 내용 | 시각적 녹화 | DOM 스냅샷, 네트워크, 콘솔, 조작 |
| 활용 | 데모, 문서화 | 디버깅, 분석 |
| 크기 | 상대적으로 큼 | 상대적으로 작음 |

## 제약

- 녹화는 자동화에 약간의 실행 비용을 더합니다.
- 큰 영상은 디스크 공간을 많이 사용할 수 있습니다.

## 한국어 예제 해설

먼저 브라우저를 열고 영상 기록을 시작합니다. `video-chapter`의 `duration=2000`은 제목 카드를 2초 표시하는 예입니다. CLI의 기본 녹화와 별개로 긴 JavaScript 예제는 `page.screencast`에서 영상 크기와 저장 경로를 설정한 뒤 TodoMVC 시나리오를 연출합니다.

예제의 동작 순서는 다음과 같습니다.

1. 1280×800 크기로 녹화를 시작하고 TodoMVC로 이동합니다.
2. 할 일을 추가한다는 장면 제목을 2초 보여 줍니다.
3. `pressSequentially`의 문자별 지연을 주어 입력하고 Enter를 누릅니다.
4. 결과 확인 장면 제목을 표시합니다.
5. 지속 오버레이를 만들고 그 상태로 두 번째 항목을 입력합니다.
6. `annotation.dispose()`로 지속 오버레이를 제거합니다.
7. 첫 항목의 bounding box를 얻어 해당 위치에 테두리와 설명을 배치합니다.
8. 녹화를 종료합니다.

오버레이는 포인터 이벤트를 받지 않으므로 설명이 보이는 동안에도 페이지를 클릭할 수 있습니다. `showChapter`는 구간 소개, `showOverlay`는 자유로운 HTML 설명/강조, `hideOverlays`/`showOverlays`는 일시 숨김/복원입니다. 이 예제의 표시용 `waitForTimeout`은 영상 연출 목적이며, 실패한 테스트를 임의 대기로 통과시키는 방법과는 구분합니다.
