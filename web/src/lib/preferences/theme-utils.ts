/**
 * 한국어 해설 — src/lib/preferences/theme-utils.ts
 * 선택한 테마와 운영체제의 다크 모드를 실제 DOM 상태로 연결한다.
 * system은 matchMedia로 light/dark를 결정하고 dark 클래스·colorScheme·data-theme-mode를 일관되게 갱신한다.
 * 전환 중 잠시 disable-transitions를 붙인 뒤 다음 animation frame에서 제거한다.
 * OS 테마 변경 구독은 해제 함수를 반환하므로 Provider가 생명주기에 맞춰 정리할 수 있다.
 */

import type { ResolvedThemeMode, ThemeMode } from "./theme";

// system을 현재 운영체제 설정으로 해석하고 그 외 값은 명시적 light/dark로 정규화한다.
export function resolveThemeMode(mode: ThemeMode): ResolvedThemeMode {
  if (mode === "system") {
    const prefersDark = window.matchMedia?.("(prefers-color-scheme: dark)")?.matches;
    return prefersDark ? "dark" : "light";
  }
  return mode === "dark" ? "dark" : "light";
}

// 선택 모드와 실제 해석 모드를 분리해 data 속성에는 선택을, dark 클래스에는 실제 결과를 기록한다.
export function applyThemeMode(mode: ThemeMode): ResolvedThemeMode {
  const resolved = resolveThemeMode(mode);
  const doc = document.documentElement;
  doc.setAttribute("data-theme-mode", mode);
  doc.classList.add("disable-transitions");
  doc.classList.toggle("dark", resolved === "dark");
  doc.style.colorScheme = resolved;
  requestAnimationFrame(() => {
    doc.classList.remove("disable-transitions");
  });
  return resolved;
}

// 테마 CSS 선택자가 읽는 프리셋 키를 html에 설정한다.
export function applyThemePreset(value: string) {
  document.documentElement.setAttribute("data-theme-preset", value);
}

// matchMedia change를 구독하고 listener 제거 함수를 반환한다. 브라우저 API가 없으면 빈 해제 함수다.
export function subscribeToSystemTheme(onChange: (mode: ResolvedThemeMode) => void): () => void {
  if (typeof window === "undefined") return () => undefined;
  const media = window.matchMedia?.("(prefers-color-scheme: dark)");
  if (!media) return () => undefined;

  const listener = (event: MediaQueryListEvent) => {
    onChange(event.matches ? "dark" : "light");
  };

  media.addEventListener("change", listener);

  return () => {
    media.removeEventListener("change", listener);
  };
}
