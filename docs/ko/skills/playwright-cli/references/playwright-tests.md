> **사람용 한국어 번역·해설입니다. ARTEX가 이 문서를 자동으로 로드하지 않습니다.**
> 원본: [references/playwright-tests.md](../../../../../skills/playwright-cli/references/playwright-tests.md). 실행용 원본 파일은 그대로 유지합니다.
> 본문·제목·제약을 한국어로 옮기고, 명령/코드 블록은 원문 그대로 보존했습니다. 예제 안의 설명은 마지막 한국어 보충 해설에서도 풀어 씁니다.

# Playwright 테스트 실행

Playwright 테스트는 `npx playwright test` 또는 패키지 관리자 스크립트로 실행합니다. 대화형 HTML 보고서가 자동으로 열리지 않게 하려면 `PLAYWRIGHT_HTML_OPEN=never` 환경 변수를 사용합니다.

```bash
# Run all tests
PLAYWRIGHT_HTML_OPEN=never npx playwright test

# Run all tests through a custom npm script
PLAYWRIGHT_HTML_OPEN=never npm run special-test-command
```

# Playwright 테스트 디버깅

실패한 Playwright 테스트를 조사할 때는 `--debug=cli` 옵션으로 실행합니다. 테스트가 시작 지점에서 일시 중지되고 디버깅 안내가 출력됩니다.

**중요:** 명령을 백그라운드에서 실행하고 “Debugging Instructions”가 출력될 때까지 확인합니다. 작업이 끝나면 반드시 해당 실행을 중지합니다.

세션 이름이 포함된 안내가 출력되면 `playwright-cli`로 그 세션에 연결하여 페이지를 살펴봅니다.

```bash
# Run the test
PLAYWRIGHT_HTML_OPEN=never npx playwright test --debug=cli
# ...
# ... debugging instructions for "tw-abcdef" session ...
# ...

# Attach to the test
playwright-cli attach tw-abcdef
```

페이지를 살펴보고 수정 방법을 찾는 동안 테스트는 백그라운드에서 계속 실행 상태로 둡니다.
테스트는 시작 위치에서 멈춰 있으므로 단계 실행을 하거나 문제가 있을 것으로 보이는
특정 위치에 멈춰서 조사합니다.

`playwright-cli`에서 수행하는 각 동작은 대응하는 Playwright TypeScript 코드를 생성합니다.
이 코드는 출력에 나타나므로 테스트에 바로 복사할 수 있습니다. 대개 locator나 기대값을 수정하면 되지만, 애플리케이션 자체의 버그일 수도 있으므로 관찰 결과를 바탕으로 판단합니다.

테스트를 수정한 뒤 백그라운드 디버그 실행을 중지하고 다시 실행해 통과하는지 확인합니다.

## 한국어 예제 해설

일반 실행은 테스트를 끝까지 수행하고 결과를 확인하는 과정입니다. `--debug=cli`는 테스트 설정과 fixture를 그대로 사용하면서 시작 지점에서 멈춰 대화형으로 조사하는 과정입니다. 예제의 `tw-abcdef`는 출력에서 실제로 확인한 세션 이름으로 바꿔 읽습니다. 명령을 실행하지 않고 임의의 세션 이름을 추정하는 절차가 아닙니다.
