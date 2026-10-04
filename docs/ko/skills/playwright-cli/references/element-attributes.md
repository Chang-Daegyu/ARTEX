> **사람용 한국어 번역·해설입니다. ARTEX가 이 문서를 자동으로 로드하지 않습니다.**
> 원본: [references/element-attributes.md](../../../../../skills/playwright-cli/references/element-attributes.md). 실행용 원본 파일은 그대로 유지합니다.
> 본문·제목·제약을 한국어로 옮기고, 명령/코드 블록은 원문 그대로 보존했습니다. 예제 안의 설명은 마지막 한국어 보충 해설에서도 풀어 씁니다.

# 요소 속성 확인

스냅샷에서 요소의 `id`, `class`, `data-*` 속성 또는 다른 DOM 속성이 보이지 않으면 `eval`로 확인합니다.

## 예제

```bash
playwright-cli snapshot
# snapshot shows a button as e7 but doesn't reveal its id or data attributes

# get the element's id
playwright-cli eval "el => el.id" e7

# get all CSS classes
playwright-cli eval "el => el.className" e7

# get a specific attribute
playwright-cli eval "el => el.getAttribute('data-testid')" e7
playwright-cli eval "el => el.getAttribute('aria-label')" e7

# get a computed style property
playwright-cli eval "el => getComputedStyle(el).display" e7
```

## 한국어 예제 해설

예제는 스냅샷에서 버튼 ref `e7`을 확인한 뒤 스냅샷에 생략된 DOM 정보를 읽습니다. `el.id`는 ID, `el.className`은 CSS 클래스 문자열, `getAttribute`는 `data-testid`/`aria-label` 같은 특정 속성입니다. `getComputedStyle(el).display`는 HTML에 적힌 원문이 아니라 실제 계산된 CSS 표시 값을 확인합니다. `eval` 뒤에 ref를 주면 함수의 `el`이 해당 요소가 됩니다.
