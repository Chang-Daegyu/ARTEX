/**
 * 한국어 해설 — src/proxy.disabled.ts
 * 항상 요청을 통과시키는 Next.js Proxy 예제 파일.
 * 이 파일명은 활성 진입점 proxy.ts와 다르며 현재 인증 리다이렉트 구현은 같은 폴더의 proxy.ts에 있다.
 * 예제를 활성 파일로 바꾸면 기존 라우팅 정책이 교체되므로 단순 참고용 코드로 먼저 읽는다.
 */

// Proxy disabled.
// Rename this file to `proxy.ts` to enable it.
import { type NextRequest, NextResponse } from "next/server";

/**
 * Runs before requests complete.
 * Use for rewrites, redirects, or header changes.
 * Refer to Next.js Proxy docs for more examples.
 */
export function proxy(_req: NextRequest) {
  // Example: redirect to dashboard if user is logged in
  // const token = req.cookies.get("session_token")?.value;
  // if (token && req.nextUrl.pathname === "/auth/login")
  //   return NextResponse.redirect(new URL("/dashboard", req.url));

  return NextResponse.next();
}

/**
 * Matcher runs for all routes.
 * To skip assets or APIs, use a negative matcher from docs.
 */
export const config = {
  matcher: "/:path*",
};
