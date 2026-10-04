"use client";

/**
 * 한국어 해설 — src/stores/preferences/preferences-provider.tsx
 * 컴포넌트 트리별 Zustand 저장소를 만들고 초기 DOM 테마와 동기화한다.
 * ThemeBootScript가 이미 설정한 data-* 값을 허용 목록으로 읽어 store에 반영한 뒤 isSynced를 켠다.
 * 사용자가 system 모드를 고른 동안만 OS 테마 변경을 구독하며, 모드 변경/해제 시 구독을 정리한다.
 * usePreferencesStore(selector)는 필요한 조각만 React에 구독시키고 Provider 밖 사용은 오류로 드러낸다.
 */

import { createContext, useContext, useEffect, useRef, useState } from "react";

import { type StoreApi, useStore } from "zustand";

import { type FontKey, fontRegistry } from "@/lib/fonts/registry";
import {
  CONTENT_LAYOUT_VALUES,
  NAVBAR_STYLE_VALUES,
  SIDEBAR_COLLAPSIBLE_VALUES,
  SIDEBAR_VARIANT_VALUES,
} from "@/lib/preferences/layout";
import { THEME_MODE_VALUES, THEME_PRESET_VALUES } from "@/lib/preferences/theme";
import { applyThemeMode, subscribeToSystemTheme } from "@/lib/preferences/theme-utils";

import { createPreferencesStore, type PreferencesState } from "./preferences-store";

const PreferencesStoreContext = createContext<StoreApi<PreferencesState> | null>(null);

const FONT_VALUES = Object.keys(fontRegistry) as FontKey[];

// DOM 속성은 문자열이므로 허용 목록에 포함되는 값만 저장소 타입으로 받아들인다.
function getSafeValue<T extends string>(raw: string | null, allowed: readonly T[]): T | undefined {
  if (!raw) return undefined;
  return allowed.includes(raw as T) ? (raw as T) : undefined;
}

// 초기 스크립트가 적용한 html 속성과 dark 클래스를 읽어 React store의 시작 상태로 변환한다.
function readDomState(): Partial<PreferencesState> {
  const root = document.documentElement;

  const themeModeAttr = getSafeValue(root.getAttribute("data-theme-mode"), THEME_MODE_VALUES);
  const resolvedMode = root.classList.contains("dark") ? "dark" : "light";

  return {
    themeMode: themeModeAttr ?? resolvedMode,
    resolvedThemeMode: resolvedMode,
    themePreset: getSafeValue(root.getAttribute("data-theme-preset"), THEME_PRESET_VALUES),
    font: getSafeValue(root.getAttribute("data-font"), FONT_VALUES),
    contentLayout: getSafeValue(root.getAttribute("data-content-layout"), CONTENT_LAYOUT_VALUES),
    navbarStyle: getSafeValue(root.getAttribute("data-navbar-style"), NAVBAR_STYLE_VALUES),
    sidebarVariant: getSafeValue(root.getAttribute("data-sidebar-variant"), SIDEBAR_VARIANT_VALUES),
    sidebarCollapsible: getSafeValue(root.getAttribute("data-sidebar-collapsible"), SIDEBAR_COLLAPSIBLE_VALUES),
  };
}

// useState의 지연 초기화로 마운트된 Provider당 store를 한 번 만들고 초기 props를 넣는다.
export const PreferencesStoreProvider = ({
  children,
  themeMode,
  themePreset,
  font,
  contentLayout,
  navbarStyle,
}: {
  children: React.ReactNode;
  themeMode: PreferencesState["themeMode"];
  themePreset: PreferencesState["themePreset"];
  font: PreferencesState["font"];
  contentLayout: PreferencesState["contentLayout"];
  navbarStyle: PreferencesState["navbarStyle"];
}) => {
  const [store] = useState<StoreApi<PreferencesState>>(() =>
    createPreferencesStore({
      themeMode,
      themePreset,
      font,
      contentLayout,
      navbarStyle,
    }),
  );

  const domSnapshotRef = useRef<Partial<PreferencesState> | null>(null);

  useEffect(() => {
    const domState = readDomState();
    domSnapshotRef.current = domState;

    store.setState((prev) => ({
      ...prev,
      ...domState,
      isSynced: true,
    }));
  }, [store]);

  useEffect(() => {
    let unsubscribeMedia: (() => void) | undefined;

    // 기존 OS 구독을 먼저 해제한 후 새 모드를 적용한다. system일 때만 미디어 구독을 다시 연결한다.
    const applyFromMode = (mode: PreferencesState["themeMode"]) => {
      unsubscribeMedia?.();
      const resolved = applyThemeMode(mode);
      store.setState((prev) => ({ ...prev, resolvedThemeMode: resolved }));

      if (mode === "system") {
        unsubscribeMedia = subscribeToSystemTheme(() => {
          const next = applyThemeMode("system");
          store.setState((prev) => ({ ...prev, resolvedThemeMode: next }));
        });
      }
    };

    const startMode = domSnapshotRef.current?.themeMode ?? store.getState().themeMode;
    applyFromMode(startMode);

    const unsubscribeStore = store.subscribe((s, p) => {
      if (s.themeMode !== p.themeMode) applyFromMode(s.themeMode);
    });

    return () => {
      unsubscribeMedia?.();
      unsubscribeStore();
    };
  }, [store]);

  return <PreferencesStoreContext.Provider value={store}>{children}</PreferencesStoreContext.Provider>;
};

// selector로 필요한 상태 조각만 구독한다. Provider가 없으면 잘못된 트리 구성을 즉시 알린다.
export const usePreferencesStore = <T,>(selector: (state: PreferencesState) => T): T => {
  const store = useContext(PreferencesStoreContext);
  if (!store) throw new Error("Missing PreferencesStoreProvider");
  return useStore(store, selector);
};
