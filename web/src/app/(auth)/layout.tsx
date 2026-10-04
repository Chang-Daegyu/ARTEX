/* 한국어 해설: 인증 화면의 공통 배치
 * 로그인과 최초 비밀번호 설정 페이지의 children을 Fragment로 그대로 전달한다.
 * 이 레이아웃에는 토큰 검증이나 API 요청이 없으며 각 인증 페이지가 초기 상태 조회와 이동을 담당한다.
 * 괄호로 묶인 (auth)는 경로에 나타나지 않는 Next 라우트 그룹이다.
 * 자세한 안내: docs/ko/modules/web-pages.md
 */

import type { ReactNode } from "react";

export default function AuthLayout({ children }: { readonly children: ReactNode }) {
  return <>{children}</>;
}
