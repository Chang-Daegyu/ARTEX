/* 한국어 해설: 루트 주소의 진입 경로
 * 웹사이트 루트 / 요청을 작업 목록 /function/tasks로 연결하는 작은 서버 페이지다.
 * 자체 화면이나 작업 실행 기능을 갖지 않고 Next의 redirect를 사용한다.
 * 로그인 필요 여부는 이동한 경로의 레이아웃과 서버 인증 계층에서 처리된다.
 * 자세한 안내: docs/ko/modules/web-pages.md
 */

import { redirect } from "next/navigation";

export default function Home() {
  redirect("/function/tasks");
  return <>Coming Soon</>;
}
