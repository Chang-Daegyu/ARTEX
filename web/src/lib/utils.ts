/**
 * 한국어 해설 — src/lib/utils.ts
 * UI 공통 도우미: Tailwind 클래스 병합, 팝업 닫힘 조정, 복사, 이니셜, 통화 포맷.
 * cn은 clsx의 조건부 클래스와 twMerge의 충돌 해결을 결합한다.
 * Radix 중첩 팝업은 pointerdown 캡처 단계의 열림 상태를 기억하여 안쪽 팝업을 닫는 클릭이 바깥 다이얼로그까지 닫지 않게 돕는다.
 * copyText는 보안 컨텍스트의 Clipboard API를 우선 사용하고 실패하면 임시 textarea 복사로 대체한다.
 */

import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

// 조건부 클래스 배열/객체를 clsx로 펼친 뒤 Tailwind 충돌을 twMerge로 해소한다.
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

// 抽屉/对话框(Sheet/Dialog)的 onInteractOutside 关闭判定辅助。
//
// 背景:抽屉内的 Radix 弹层(Select 下拉、DropdownMenu、Popover 等)会 portal 到抽屉
// 之外。开着弹层时点遮罩/抽屉外想收起它,这一次 pointerdown 会被 Select 和 Sheet 两个
// DismissableLayer 同时处理;Select 先关闭且是 discrete 事件、React 会同步 flush,于是
// 轮到 Sheet 的处理器时弹层的 data-state 早已翻成 closed —— 在"当下"检测弹层是否打开
// 天然不可靠(实测已验证)。
//
// 正确做法:Radix 的 pointerdown 监听在冒泡阶段;我们在 capture 阶段(早于它)先把
// "此刻有没有弹层开着"记录下来,onInteractOutside 再读这个记录值来决定是否放行关闭。
// 열림 상태의 트리거·목록·포퍼 요소가 DOM에 있는지 조회한다. 서버에서는 false다.
function isRadixOverlayOpenNow(): boolean {
  if (typeof document === "undefined") return false;
  return !!document.querySelector(
    [
      "[data-slot='select-trigger'][data-state='open']",
      "[data-slot='select-content'][data-state='open']",
      "[role='listbox'][data-state='open']",
      "[data-radix-popper-content-wrapper]",
      "[aria-expanded='true'][data-state='open']",
    ].join(","),
  );
}

let overlayOpenAtLastPointerDown = false;
if (typeof document !== "undefined") {
  document.addEventListener(
    "pointerdown",
    () => {
      overlayOpenAtLastPointerDown = isRadixOverlayOpenNow();
    },
    true, // capture:抢在 Radix 冒泡阶段的 pointerdown 处理器之前记录
  );
}

// radixOverlayWasOpenAtPointerDown 返回"最近一次 pointerdown 发生时是否有 Radix 弹层
// 开着"。抽屉/对话框据此:开着弹层时点遮罩 → 只收弹层、不关自身。
// 가장 최근 pointerdown 캡처 시점의 값을 읽는다. 이벤트 처리 중 팝업이 먼저 닫혀도 이전 상태를 잃지 않는다.
export function radixOverlayWasOpenAtPointerDown(): boolean {
  return overlayOpenAtLastPointerDown;
}

// copyText 把文本写入剪贴板,返回是否成功。
// 背景:navigator.clipboard 仅在安全上下文(HTTPS / localhost)可用;通过 IP + HTTP
// 访问时它为 undefined,此时降级到 execCommand("copy")。
// 복사 성공 여부만 반환한다. Clipboard API 실패 시 화면 밖 textarea를 잠시 붙여 선택/복사하고 제거한다.
export async function copyText(text: string): Promise<boolean> {
  if (navigator.clipboard && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      // 继续走降级方案
    }
  }
  try {
    const textarea = document.createElement("textarea");
    textarea.value = text;
    textarea.style.position = "fixed";
    textarea.style.left = "-9999px";
    textarea.style.top = "0";
    document.body.appendChild(textarea);
    textarea.focus();
    textarea.select();
    const ok = document.execCommand("copy");
    document.body.removeChild(textarea);
    return ok;
  } catch {
    return false;
  }
}

export const getInitials = (str: string): string => {
  if (typeof str !== "string" || !str.trim()) return "?";

  return (
    str
      .trim()
      .split(/\s+/)
      .filter(Boolean)
      .map((word) => word[0])
      .join("")
      .toUpperCase() || "?"
  );
};

// Intl.NumberFormat에 통화/로케일/소수점 옵션을 넘겨 표시 문자열을 만든다. 환율 변환이나 금액 계산은 하지 않는다.
export function formatCurrency(
  amount: number,
  opts?: {
    currency?: string;
    locale?: string;
    minimumFractionDigits?: number;
    maximumFractionDigits?: number;
    noDecimals?: boolean;
  },
) {
  const { currency = "USD", locale = "en-US", minimumFractionDigits, maximumFractionDigits, noDecimals } = opts ?? {};

  const formatOptions: Intl.NumberFormatOptions = {
    style: "currency",
    currency,
    minimumFractionDigits: noDecimals ? 0 : minimumFractionDigits,
    maximumFractionDigits: noDecimals ? 0 : maximumFractionDigits,
  };

  return new Intl.NumberFormat(locale, formatOptions).format(amount);
}
