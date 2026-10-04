/**
 * 한국어 해설 — src/scripts/theme-boot.tsx
 * React hydration 전에 테마·글꼴·레이아웃을 적용하는 초기 스크립트 생성 컴포넌트.
 * 환경설정 저장 방식과 기본값을 JSON으로 직렬화하여 head에서 실행할 코드 문자열에 넣는다.
 * 쿠키/localStorage를 먼저 읽어 html의 class/data-* 속성을 바꾸므로 정적 HTML의 첫 화면 깜빡임을 줄인다.
 * code 변수 안은 브라우저에서 실행될 문자열이며, 설명 주석은 그 밖에 둬 원래 실행 바이트를 유지한다.
 */

/**
 * Boot script that reads user preference values (theme mode, theme preset,
 * content layout, navbar style) from cookies or localStorage based on the
 * configured persistence mode.
 *
 * Runs early in <head> to apply the correct data attributes before hydration,
 * preventing layout or theme flicker and keeping RootLayout fully static.
 */
import { PREFERENCE_DEFAULTS, PREFERENCE_PERSISTENCE } from "@/lib/preferences/preferences-config";

// 빌드 시 설정을 직렬화하고 브라우저 head에서 즉시 실행할 독립 코드 문자열을 만든다.
// 문자열 내부 코드는 React 훅 없이 동작하며 hydration 이후 Provider가 그 DOM 상태를 읽는다.
export function ThemeBootScript() {
  const persistence = JSON.stringify({
    theme_mode: PREFERENCE_PERSISTENCE.theme_mode,
    theme_preset: PREFERENCE_PERSISTENCE.theme_preset,
    font: PREFERENCE_PERSISTENCE.font,
    content_layout: PREFERENCE_PERSISTENCE.content_layout,
    navbar_style: PREFERENCE_PERSISTENCE.navbar_style,
    sidebar_variant: PREFERENCE_PERSISTENCE.sidebar_variant,
    sidebar_collapsible: PREFERENCE_PERSISTENCE.sidebar_collapsible,
  });

  const defaults = JSON.stringify({
    theme_mode: PREFERENCE_DEFAULTS.theme_mode,
    theme_preset: PREFERENCE_DEFAULTS.theme_preset,
    font: PREFERENCE_DEFAULTS.font,
    content_layout: PREFERENCE_DEFAULTS.content_layout,
    navbar_style: PREFERENCE_DEFAULTS.navbar_style,
    sidebar_variant: PREFERENCE_DEFAULTS.sidebar_variant,
    sidebar_collapsible: PREFERENCE_DEFAULTS.sidebar_collapsible,
  });

  // 아래 템플릿 문자열 전체가 브라우저로 전달될 실행 코드다.
  // 쿠키/localStorage 읽기 → 기본값 선택 → system 해석 → html 속성 반영 순서로 초기 화면을 결정한다.
  const code = `
    (function () {
      try {
        var root = document.documentElement;
        var PERSISTENCE = ${persistence};
        var DEFAULTS = ${defaults};

        function readCookie(name) {
          var match = document.cookie.split("; ").find(function(c) {
            return c.startsWith(name + "=");
          });
          return match ? decodeURIComponent(match.split("=")[1]) : null;
        }

        function readLocal(name) {
          try {
            return window.localStorage.getItem(name);
          } catch (e) {
            return null;
          }
        }

        function readPreference(key, fallback) {
          var mode = PERSISTENCE[key];
          var value = null;

          if (mode === "localStorage") {
            value = readLocal(key);
          }

          if (!value && (mode === "client-cookie" || mode === "server-cookie")) {
            value = readCookie(key);
          }

          if (!value || typeof value !== "string") {
            return fallback;
          }

          return value;
        }

        var rawMode = readPreference("theme_mode", DEFAULTS.theme_mode);
        var rawPreset = readPreference("theme_preset", DEFAULTS.theme_preset);
        var rawFont = readPreference("font", DEFAULTS.font);
        var rawContentLayout = readPreference("content_layout", DEFAULTS.content_layout);
        var rawNavbarStyle = readPreference("navbar_style", DEFAULTS.navbar_style);
        var rawSidebarVariant = readPreference("sidebar_variant", DEFAULTS.sidebar_variant);
        var rawSidebarCollapsible = readPreference("sidebar_collapsible", DEFAULTS.sidebar_collapsible);

        var isValidMode = rawMode === "dark" || rawMode === "light" || rawMode === "system";
        var mode = isValidMode ? rawMode : DEFAULTS.theme_mode;
        var resolvedMode =
          mode === "system" && window.matchMedia
            ? (window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light")
            : mode;
        var preset = rawPreset || DEFAULTS.theme_preset;
        var font = rawFont || DEFAULTS.font;
        var contentLayout = rawContentLayout || DEFAULTS.content_layout;
        var navbarStyle = rawNavbarStyle || DEFAULTS.navbar_style;
        var sidebarVariant = rawSidebarVariant || DEFAULTS.sidebar_variant;
        var sidebarCollapsible = rawSidebarCollapsible || DEFAULTS.sidebar_collapsible;

        root.classList.toggle("dark", resolvedMode === "dark");
        root.setAttribute("data-theme-mode", mode);
        root.setAttribute("data-theme-preset", preset);
        root.setAttribute("data-font", font);
        root.setAttribute("data-content-layout", contentLayout);
        root.setAttribute("data-navbar-style", navbarStyle);
        root.setAttribute("data-sidebar-variant", sidebarVariant);
        root.setAttribute("data-sidebar-collapsible", sidebarCollapsible);

        root.style.colorScheme = resolvedMode === "dark" ? "dark" : "light";

      } catch (e) {
        console.warn("ThemeBootScript error:", e);
      }
    })();
  `;

  /* biome-ignore lint/security/noDangerouslySetInnerHtml: required for pre-hydration boot script */
  return <script dangerouslySetInnerHTML={{ __html: code }} />;
}
