/**
 * 한국어 해설 — src/hooks/use-lg.ts
 * 화면 너비가 1024px 이상인지 구독하는 반응형 레이아웃 훅.
 * 첫 렌더는 false이고 마운트 후 matchMedia의 change 이벤트와 innerWidth로 실제 값을 반영한다.
 * 컴포넌트가 해제될 때 이벤트 리스너를 제거하여 중복 구독을 방지한다.
 */

import * as React from "react";

const LG_BREAKPOINT = 1024;

// effect에서 너비 기준을 적용하고 뷰포트가 경계값을 넘을 때 상태를 갱신한다.
export function useIsLg() {
  const [isLg, setIsLg] = React.useState<boolean | undefined>(undefined);

  React.useEffect(() => {
    const mql = window.matchMedia(`(min-width: ${LG_BREAKPOINT}px)`);
    const onChange = () => {
      setIsLg(window.innerWidth >= LG_BREAKPOINT);
    };
    mql.addEventListener("change", onChange);
    setIsLg(window.innerWidth >= LG_BREAKPOINT);
    return () => mql.removeEventListener("change", onChange);
  }, []);

  return !!isLg;
}
