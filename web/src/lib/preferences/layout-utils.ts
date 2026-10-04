/**
 * 한국어 해설 — src/lib/preferences/layout-utils.ts
 * 레이아웃 선택을 document.documentElement의 data-* 속성으로 적용한다.
 * 콘텐츠 폭·상단바·사이드바 형태·접기 방식·글꼴의 시각 효과는 이 속성을 읽는 CSS/컴포넌트가 담당한다.
 * 여기서는 상태 저장이나 API 호출을 하지 않는다. 지속 저장은 persistPreference를 통해 별도로 수행한다.
 */

// html의 data-content-layout을 바꿔 중앙 정렬/전체 폭 선택을 CSS에 알린다.
export function applyContentLayout(value: "centered" | "full-width") {
  const root = document.documentElement;
  root.setAttribute("data-content-layout", value);
}

// 상단바의 sticky/scroll 선택을 DOM 속성으로 전달한다.
export function applyNavbarStyle(value: "sticky" | "scroll") {
  const root = document.documentElement;
  root.setAttribute("data-navbar-style", value);
}

// 사이드바의 sidebar/inset/floating 변형을 적용하는 DOM 연결점이다.
export function applySidebarVariant(value: string) {
  const root = document.documentElement;
  root.setAttribute("data-sidebar-variant", value);
}

// 사이드바의 icon/offcanvas 접기 방식을 DOM 속성으로 전달한다.
export function applySidebarCollapsible(value: string) {
  const root = document.documentElement;
  root.setAttribute("data-sidebar-collapsible", value);
}

// 글꼴 레지스트리의 선택 키를 data-font로 반영한다.
export function applyFont(value: string) {
  const root = document.documentElement;
  root.setAttribute("data-font", value);
}
