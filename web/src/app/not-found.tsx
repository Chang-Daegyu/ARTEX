"use client";

/* 한국어 해설: 존재하지 않는 경로의 안내 화면
 * Next가 일치하는 페이지를 찾지 못했을 때 보여 주는 404 화면이다.
 * 오류를 데이터 API로 재조회하지 않고 안내 문구와 작업 목록 /function/tasks로 돌아가는 링크를 렌더링한다.
 * 경로를 신설할 때 이 화면이 보이면 app 디렉터리의 실제 page.tsx 위치와 정적 내보내기 결과를 먼저 확인한다.
 * 자세한 안내: docs/ko/modules/web-pages.md
 */

import Link from "next/link";

import { Button } from "@/components/ui/button";

export default function NotFound() {
  return (
    <div className="flex h-dvh flex-col items-center justify-center space-y-2 text-center">
      <h1 className="font-semibold text-2xl">Page not found.</h1>
      <p className="text-muted-foreground">The page you are looking for could not be found.</p>
      <Link prefetch={false} replace href="/function/tasks">
        <Button variant="outline">Go back home</Button>
      </Link>
    </div>
  );
}
