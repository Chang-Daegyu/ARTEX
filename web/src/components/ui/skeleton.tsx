/**
 * 한국어 해설 — src/components/ui/skeleton.tsx
 * 데이터 로딩 중 자리와 크기를 보여주는 펄스 애니메이션 요소.
 * 로드 상태를 판단하거나 실제 콘텐츠를 조회하지 않고 호출자가 지정한 치수를 유지한다.
 * 컴포넌트의 data-slot은 스타일/조합의 연결점이고, className 및 나머지 props는 원래 요소로 전달된다.
 */

import { cn } from "@/lib/utils"

function Skeleton({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="skeleton"
      className={cn("animate-pulse rounded-md bg-muted", className)}
      {...props}
    />
  )
}

export { Skeleton }
