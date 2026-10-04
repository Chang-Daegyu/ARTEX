/**
 * 한국어 해설 — src/hooks/use-mobile.ts
 * 화면 너비가 768px 미만인지 구독하여 모바일 UI 분기를 결정한다.
 * SSR/최초 렌더에서는 브라우저에 접근하지 않고, effect에서 너비를 읽은 뒤 change 이벤트를 구독한다.
 * 사이드바의 모바일 Sheet 전환도 이 훅의 결과를 사용한다.
 */

import * as React from "react";

const MOBILE_BREAKPOINT = 768;

// 모바일 기준값을 반환한다. undefined 초기 상태는 boolean 변환으로 false가 된다.
export function useIsMobile() {
  const [isMobile, setIsMobile] = React.useState<boolean | undefined>(undefined);

  React.useEffect(() => {
    const mql = window.matchMedia(`(max-width: ${MOBILE_BREAKPOINT - 1}px)`);
    const onChange = () => {
      setIsMobile(window.innerWidth < MOBILE_BREAKPOINT);
    };
    mql.addEventListener("change", onChange);
    setIsMobile(window.innerWidth < MOBILE_BREAKPOINT);
    return () => mql.removeEventListener("change", onChange);
  }, []);

  return !!isMobile;
}
