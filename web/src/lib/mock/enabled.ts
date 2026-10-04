/**
 * 한국어 해설 — src/lib/mock/enabled.ts
 * NEXT_PUBLIC_MOCK=1인 빌드에서만 데모 데이터 경로를 여는 공통 플래그.
 * 브라우저에 포함되는 환경 변수이며 실제 서버의 인증·LLM 실행 상태를 나타내는 값이 아니다.
 */

// Mock 开关。构建期注入的公开变量（NEXT_PUBLIC_ 前缀才在浏览器可读）。
// Vercel 上设 NEXT_PUBLIC_MOCK=1 即整站走 mock、无需后端。
export const MOCK = process.env.NEXT_PUBLIC_MOCK === "1";
