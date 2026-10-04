"use client";

/**
 * 한국어 해설 — src/lib/sort-preference.ts
 * 표의 정렬 열/방향을 브라우저에 보관하는 제네릭 훅.
 * 저장값을 JSON으로 읽은 뒤 현재 허용 열 목록과 asc/desc를 확인한다.
 * hydrated가 true가 되기 전에는 저장하지 않아 초기 기본값이 기존 사용자 선택을 덮어쓰는 것을 막는다.
 */

import * as React from "react";

import { getLocalStorageValue, setLocalStorageValue } from "@/lib/local-storage.client";

export type SortDirection = "asc" | "desc";

export interface SortPreference<Field extends string> {
  field: Field;
  direction: SortDirection;
}

// 정렬 값은 먼저 기본값으로 렌더하고 마운트 뒤 저장값을 검증해 복원한다.
// 복원이 끝난 이후의 상태 변경만 저장하므로 기존 설정이 초기 렌더에 덮어써지지 않는다.
export function useStoredSortPreference<Field extends string>(
  key: string,
  fields: readonly Field[],
  defaultField: Field,
  defaultDirection: SortDirection,
) {
  const [preference, setPreference] = React.useState<SortPreference<Field>>({
    field: defaultField,
    direction: defaultDirection,
  });
  const [hydrated, setHydrated] = React.useState(false);

  React.useEffect(() => {
    const raw = getLocalStorageValue(key);
    if (raw) {
      try {
        const parsed = JSON.parse(raw) as Partial<SortPreference<string>>;
        const field = fields.find((candidate) => candidate === parsed.field);
        const direction = parsed.direction === "asc" || parsed.direction === "desc" ? parsed.direction : null;
        if (field && direction) setPreference({ field, direction });
      } catch {
        // Ignore malformed or legacy preferences and retain the current default.
      }
    }
    setHydrated(true);
  }, [fields, key]);

  React.useEffect(() => {
    if (!hydrated) return;
    setLocalStorageValue(key, JSON.stringify(preference));
  }, [hydrated, key, preference]);

  return [preference, setPreference] as const;
}
