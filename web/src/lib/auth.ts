/**
 * 한국어 해설 — src/lib/auth.ts
 * 브라우저 인증 토큰 저장과 표시용 JWT 해석.
 * localStorage는 API의 Bearer 헤더에, 동기화한 쿠키는 Next.js 페이지 이동 판단에 사용된다.
 * getCurrentUser는 payload의 sub를 화면에 표시할 뿐 서명을 확인하지 않는다. 서버 인증 결과와 구분한다.
 * 토큰 저장 기간은 7일이며, 이 코드가 쓰는 쿠키는 JavaScript에서 접근 가능한 쿠키다.
 */

const TOKEN_KEY = "artex_token";
const COOKIE_MAX_AGE = 7 * 24 * 60 * 60; // 7 天（秒）

export interface CurrentUser {
  id: string;
  name: string;
  username: string;
  email: string;
  avatar: string;
  role: string;
}

export const auth = {
  getToken(): string | null {
    if (typeof window === "undefined") return null;
    // Mock demo：无真实登录，返回一个假 token 让路由守卫放行、直接进主界面。
    return localStorage.getItem(TOKEN_KEY) ?? (process.env.NEXT_PUBLIC_MOCK === "1" ? "mock-demo" : null);
  },

  // 하나의 토큰을 localStorage와 쿠키에 동기화한다.
  // 쿠키는 페이지 이동용이며 HttpOnly 쿠키를 만드는 서버 응답은 아니다.
  setToken(token: string): void {
    localStorage.setItem(TOKEN_KEY, token);
    // 同步写 cookie，供 Next.js middleware 服务端读取
    document.cookie = `${TOKEN_KEY}=${encodeURIComponent(token)}; path=/; max-age=${COOKIE_MAX_AGE}; SameSite=Lax`;
  },

  // 브라우저의 두 저장 위치를 함께 지워 로그아웃 이후 페이지/API의 상태를 맞춘다.
  clearToken(): void {
    localStorage.removeItem(TOKEN_KEY);
    document.cookie = `${TOKEN_KEY}=; path=/; max-age=0`;
  },

  // 从 JWT payload 的 sub 字段解析当前用户，仅用于展示，不做签名验证。
  // JWT payload를 base64url 디코딩해 sub를 읽는 표시 도우미다.
  // 서명·만료 검증 없이 읽은 정보이므로 권한 판단의 근거로 사용하면 안 된다.
  getCurrentUser(): CurrentUser | null {
    const token = this.getToken();
    if (!token) return null;
    try {
      const parts = token.split(".");
      if (parts.length !== 3) return null;
      // base64url → base64
      const payload = JSON.parse(atob(parts[1].replace(/-/g, "+").replace(/_/g, "/")));
      const username: string = payload.sub ?? "ARTEX";
      return { id: "1", name: username, username, email: "", avatar: "", role: "operator" };
    } catch {
      return null;
    }
  },
};
