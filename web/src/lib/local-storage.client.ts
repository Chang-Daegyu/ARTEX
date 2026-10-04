"use client";

/**
 * 한국어 해설 — src/lib/local-storage.client.ts
 * 브라우저 localStorage 접근 실패를 흡수하는 환경설정 저장 도우미.
 * 읽기가 막히면 null을 반환하고, 쓰기 실패는 개발 모드에서만 콘솔에 남긴다.
 * 개인정보 보호 설정이나 저장 공간 제한 때문에 UI 전체가 중단되지 않도록 경계를 둔다.
 */

// 저장이 막혀도 화면 동작은 계속되도록 예외를 잡으며 운영 모드에서는 오류 로그도 생략한다.
export function setLocalStorageValue(key: string, value: string) {
  try {
    window.localStorage.setItem(key, value);
  } catch (error) {
    if (process.env.NODE_ENV !== "production") {
      console.error("[localStorage] Failed to write value:", error);
    }
  }
}

// 저장소 접근이 불가능하거나 키가 없으면 호출자가 기본값을 고를 수 있도록 null을 반환한다.
export function getLocalStorageValue(key: string): string | null {
  try {
    return window.localStorage.getItem(key);
  } catch {
    return null;
  }
}
