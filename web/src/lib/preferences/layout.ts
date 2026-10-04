/**
 * 한국어 해설 — src/lib/preferences/layout.ts
 * 사이드바 형태·접힘 방식·콘텐츠 폭·상단바 배치의 허용 선택값 정의.
 * OPTIONS는 UI 표시용, VALUES는 검증용 목록, typeof로 파생한 타입은 코드의 값 범위를 제한한다.
 * value 문자열은 CSS data-* 속성 및 저장된 환경설정과 연결되므로 바꾸면 함께 마이그레이션해야 한다.
 */

// Sidebar Variant
export const SIDEBAR_VARIANT_OPTIONS = [
  { label: "Sidebar", value: "sidebar" },
  { label: "Inset", value: "inset" },
  { label: "Floating", value: "floating" },
] as const;
export const SIDEBAR_VARIANT_VALUES = SIDEBAR_VARIANT_OPTIONS.map((v) => v.value);
export type SidebarVariant = (typeof SIDEBAR_VARIANT_VALUES)[number];

// Sidebar Collapsible
export const SIDEBAR_COLLAPSIBLE_OPTIONS = [
  { label: "Icon", value: "icon" },
  { label: "Offcanvas", value: "offcanvas" },
] as const;
export const SIDEBAR_COLLAPSIBLE_VALUES = SIDEBAR_COLLAPSIBLE_OPTIONS.map((v) => v.value);
export type SidebarCollapsible = (typeof SIDEBAR_COLLAPSIBLE_VALUES)[number];

// Content Layout
export const CONTENT_LAYOUT_OPTIONS = [
  { label: "Centered", value: "centered" },
  { label: "Full Width", value: "full-width" },
] as const;
export const CONTENT_LAYOUT_VALUES = CONTENT_LAYOUT_OPTIONS.map((v) => v.value);
export type ContentLayout = (typeof CONTENT_LAYOUT_VALUES)[number];

// Navbar Style
export const NAVBAR_STYLE_OPTIONS = [
  { label: "Sticky", value: "sticky" },
  { label: "Scroll", value: "scroll" },
] as const;
export const NAVBAR_STYLE_VALUES = NAVBAR_STYLE_OPTIONS.map((v) => v.value);
export type NavbarStyle = (typeof NAVBAR_STYLE_VALUES)[number];
