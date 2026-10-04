"use client";

/**
 * 한국어 해설 — src/lib/preferences/preferences-storage.ts
 * 선택한 환경설정의 지속 저장만 담당하는 어댑터.
 * 설정표의 모드에 따라 무저장·쿠키·localStorage로 분기한다.
 * 정적 내보내기에는 Node Server Action이 없으므로 server-cookie도 client-cookie와 같은 브라우저 쓰기를 수행한다.
 */

import { setClientCookie } from "../cookie.client";
import { setLocalStorageValue } from "../local-storage.client";
import { PREFERENCE_PERSISTENCE, type PreferenceKey } from "./preferences-config";

// 키별 저장 정책을 읽는다. 이 함수는 저장만 하며 React store나 DOM을 직접 변경하지 않는다.
export async function persistPreference(key: PreferenceKey, value: string) {
  const mode = PREFERENCE_PERSISTENCE[key];

  switch (mode) {
    case "none":
      return;

    // 静态导出无 Node 服务端，server-cookie 退化为浏览器 cookie（写入语义等价）。
    case "server-cookie":
    case "client-cookie":
      setClientCookie(key, value);
      return;

    case "localStorage":
      setLocalStorageValue(key, value);
      return;

    default:
      return;
  }
}
