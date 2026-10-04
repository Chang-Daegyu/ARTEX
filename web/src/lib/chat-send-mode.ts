"use client";

/**
 * 한국어 해설 — src/lib/chat-send-mode.ts
 * Enter/Ctrl+Enter 전송 방식의 브라우저별 환경설정과 키 입력 판정.
 * useSyncExternalStore로 같은 탭의 직접 알림과 다른 탭의 storage 이벤트를 함께 구독한다.
 * 서버 렌더는 기본값으로 시작하고 브라우저 저장값은 hydration 후 반영한다.
 * 한글 등 IME 조합 중 Enter는 전송으로 처리하지 않아 조합 확정과 메시지 발송이 충돌하지 않게 한다.
 */

import * as React from "react";

import { getLocalStorageValue, setLocalStorageValue } from "@/lib/local-storage.client";

// 会话输入框的发送/换行键位。纯前端偏好：只落 localStorage，不入库、不随账号同步，
// 因此换浏览器需要重设。见 issue #39——0.3.2 把 Ctrl+Enter 发送改成了 Enter 发送，
// 这里把旧键位还回来作为可选项。
export type ChatSendMode = "enter" | "ctrl-enter";

export const CHAT_SEND_MODE_KEY = "artex_chat_send_mode";
export const DEFAULT_CHAT_SEND_MODE: ChatSendMode = "enter";

export const CHAT_SEND_MODE_OPTIONS: { value: ChatSendMode; label: string }[] = [
  { value: "enter", label: "Enter 发送，Shift+Enter 换行" },
  { value: "ctrl-enter", label: "Ctrl+Enter 发送，Enter 换行" },
];

// 알려진 두 값만 허용하고 오래되거나 손상된 저장값은 기본 모드로 되돌린다.
function parseMode(raw: string | null): ChatSendMode {
  return raw === "ctrl-enter" || raw === "enter" ? raw : DEFAULT_CHAT_SEND_MODE;
}

// 同一标签页内的订阅者集合。localStorage 的 storage 事件只在「其他」标签页触发，
// 本页在设置里改完后要靠 emit 通知同页的输入框，否则得刷新才生效。
const listeners = new Set<() => void>();

// 현재 탭의 수동 알림 집합과 다른 탭의 storage 이벤트를 함께 등록하고 해제 함수를 반환한다.
function subscribe(listener: () => void) {
  listeners.add(listener);
  window.addEventListener("storage", listener);
  return () => {
    listeners.delete(listener);
    window.removeEventListener("storage", listener);
  };
}

// 返回的是字符串字面量，Object.is 按值比较，不会让 useSyncExternalStore 陷入循环。
// 문자열 리터럴을 반환하므로 React가 Object.is로 비교할 때 매번 새 객체가 생기지 않는다.
function getSnapshot(): ChatSendMode {
  return parseMode(getLocalStorageValue(CHAT_SEND_MODE_KEY));
}

// 服务端没有 localStorage，先渲染默认值，hydrate 后 getSnapshot 再纠正。
// SSR에서는 localStorage를 읽을 수 없으므로 기본값을 사용한다.
function getServerSnapshot(): ChatSendMode {
  return DEFAULT_CHAT_SEND_MODE;
}

// 외부 브라우저 저장소를 React 구독 모델과 연결하여 설정 화면과 입력창의 값을 동기화한다.
export function useChatSendMode(): ChatSendMode {
  return React.useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}

// 저장 후 같은 탭 리스너에도 직접 알린다. storage 이벤트만으로는 현재 탭이 갱신되지 않는다.
export function setChatSendMode(mode: ChatSendMode) {
  setLocalStorageValue(CHAT_SEND_MODE_KEY, mode);
  for (const listener of listeners) listener();
}

// shouldSubmitOnKey 判断一次按键是否应当发送。
// isComposing / keyCode 229 是中文等输入法正在选字，必须放行，否则回车选词会误发送。
// enter 模式只排除 Shift，与 0.3.2 的行为逐字保持一致——不改设置的用户手感不变。
// ctrl-enter 模式同时接受 Ctrl 与 Cmd（macOS）。
// Enter인지 먼저 확인한 다음 IME 조합 중인지 검사한다.
// ctrl-enter 모드는 Ctrl 또는 macOS Cmd를, enter 모드는 Shift가 없는 Enter를 전송으로 해석한다.
export function shouldSubmitOnKey(e: React.KeyboardEvent, mode: ChatSendMode): boolean {
  if (e.key !== "Enter") return false;
  if (e.nativeEvent.isComposing || e.nativeEvent.keyCode === 229) return false;
  if (mode === "ctrl-enter") return e.ctrlKey || e.metaKey;
  return !e.shiftKey;
}
