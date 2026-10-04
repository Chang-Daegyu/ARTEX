"use client";

/**
 * 한국어 해설 — src/hooks/use-current-user.ts
 * 브라우저 마운트 후 JWT의 표시용 사용자 정보를 화면 상태에 반영하는 훅.
 * 초기 렌더는 ARTEX 기본 사용자로 시작하며 auth.getCurrentUser가 성공하면 한 번 갱신한다.
 * 사용자 표시를 제공할 뿐 서명 검증·권한 확인·서버 사용자 조회를 수행하지 않는다.
 */

import { useEffect, useState } from "react";

import { auth, type CurrentUser } from "@/lib/auth";

const FALLBACK: CurrentUser = { id: "1", name: "ARTEX", username: "artex", email: "", avatar: "", role: "operator" };

// 사용자 표시 상태를 마운트 후 한 번 초기화한다. 토큰 변경을 실시간 구독하는 훅은 아니다.
export function useCurrentUser(): CurrentUser {
  const [user, setUser] = useState<CurrentUser>(FALLBACK);

  useEffect(() => {
    const u = auth.getCurrentUser();
    if (u) setUser(u);
  }, []);

  return user;
}
