/* 한국어 해설: 사이드바의 지원 안내 카드
 * 지원 안내 문구와 외부 링크를 정적으로 렌더링하는 작은 UI 조각이다.
 * 실행 상태나 서버 데이터를 조회하지 않으므로 내용 변경은 이 JSX와 링크를 수정하는 것으로 충분하다.
 * 원본 템플릿의 문구와 연결 주소는 실행 동작 보존을 위해 그대로 남겼다.
 * 자세한 안내: docs/ko/modules/web-pages.md
 */

import Link from "next/link";

import { siX } from "simple-icons";

import { SimpleIcon } from "@/components/simple-icon";
import { Card, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export function SidebarSupportCard() {
  return (
    <Card size="sm" className="overflow-hidden shadow-none group-data-[collapsible=icon]:hidden">
      <CardHeader className="min-w-0 px-4">
        <CardTitle className="truncate text-sm">Looking for something more?</CardTitle>
        <CardDescription className="line-clamp-2">
          Open an issue or do reach out to me on&nbsp;
          <Link
            href="https://x.com/arhamkhnz"
            target="_blank"
            rel="noreferrer"
            aria-label="Reach out on X"
            className="inline-flex items-center text-foreground"
          >
            <SimpleIcon icon={siX} aria-hidden className="size-3 fill-current" />
          </Link>
          .
        </CardDescription>
      </CardHeader>
    </Card>
  );
}
