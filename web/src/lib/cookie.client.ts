/**
 * 한국어 해설 — src/lib/cookie.client.ts
 * 브라우저 document.cookie에 대한 작은 읽기·쓰기·삭제 도우미.
 * 환경설정 쿠키는 기본 7일 동안 path=/에 저장하고 삭제는 과거 만료일을 기록하는 방식이다.
 * 문자열 직렬화를 그대로 쓰므로 호출자가 값의 인코딩 조건을 이해해야 한다. 인증 전용 처리는 auth.ts에 별도로 있다.
 */

// Client-side cookie utilities.
// These functions manage cookies in the browser only.
// Server actions handle cookie updates on the server side.

// 쿠키 대입을 한 함수에 모아 브라우저 전용 접근과 lint 예외의 범위를 분명히 한다.
function writeClientCookie(serializedCookie: string) {
  // biome-ignore lint/suspicious/noDocumentCookie: This project still uses document.cookie for broad browser support.
  document.cookie = serializedCookie;
}

// 일수를 만료 시각으로 바꿔 사이트 루트 경로의 쿠키를 기록한다.
export function setClientCookie(key: string, value: string, days = 7) {
  const expires = new Date(Date.now() + days * 864e5).toUTCString();
  writeClientCookie(`${key}=${value}; expires=${expires}; path=/`);
}

// 쿠키 목록에서 이름이 일치하는 항목의 값을 읽는다. URI 디코딩은 여기서 하지 않는다.
export function getClientCookie(key: string) {
  return document.cookie
    .split("; ")
    .find((row) => row.startsWith(`${key}=`))
    ?.split("=")[1];
}

// 같은 path에 과거 만료일을 써 브라우저가 해당 쿠키를 제거하도록 한다.
export function deleteClientCookie(key: string) {
  writeClientCookie(`${key}=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/`);
}
