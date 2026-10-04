/**
 * 한국어 해설 — src/components/ui/spinner.tsx
 * 작업 중임을 표시하는 회전 아이콘.
 * status 역할과 스크린리더 레이블을 가진 SVG를 반환하며 시간 측정이나 작업 제어는 하지 않는다.
 * 컴포넌트의 data-slot은 스타일/조합의 연결점이고, className 및 나머지 props는 원래 요소로 전달된다.
 */

import { cn } from "@/lib/utils"
import { Loader2Icon } from "lucide-react"

function Spinner({ className, ...props }: React.ComponentProps<"svg">) {
  return (
    <Loader2Icon data-slot="spinner" role="status" aria-label="Loading" className={cn("size-4 animate-spin", className)} {...props} />
  )
}

export { Spinner }
