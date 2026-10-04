"use client";

/* 한국어 해설: 밝음·어두움·시스템 테마 순환
 * THEME_CYCLE 순서에 따라 현재 테마 모드를 다음 값으로 바꾸고 환경설정 저장소의 setter를 호출한다.
 * 버튼 아이콘은 현재 모드와 연결되며 실제 CSS 적용과 지속성은 공통 테마 계층이 담당한다.
 * 이 컴포넌트는 서버 설정 API를 사용하지 않는다.
 * 자세한 안내: docs/ko/modules/web-pages.md
 */

import { Monitor, Moon, Sun } from "lucide-react";

import { Button } from "@/components/ui/button";
import { persistPreference } from "@/lib/preferences/preferences-storage";
import { usePreferencesStore } from "@/stores/preferences/preferences-provider";

const THEME_CYCLE = ["light", "dark", "system"] as const;

export function ThemeSwitcher() {
  const themeMode = usePreferencesStore((s) => s.themeMode);
  const setThemeMode = usePreferencesStore((s) => s.setThemeMode);

  const cycleTheme = () => {
    const currentIndex = THEME_CYCLE.indexOf(themeMode);
    const nextTheme = THEME_CYCLE[(currentIndex + 1) % THEME_CYCLE.length];

    setThemeMode(nextTheme);
    void persistPreference("theme_mode", nextTheme);
  };

  return (
    <Button size="icon" onClick={cycleTheme} aria-label={`Current theme: ${themeMode}. Click to cycle themes`}>
      {/* SYSTEM */}
      <Monitor className="hidden [html[data-theme-mode=system]_&]:block" />

      {/* DARK (resolved) */}
      <Sun className="hidden dark:block [html[data-theme-mode=system]_&]:hidden" />

      {/* LIGHT (resolved) */}
      <Moon className="block dark:hidden [html[data-theme-mode=system]_&]:hidden" />
    </Button>
  );
}
