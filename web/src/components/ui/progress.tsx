"use client"

/**
 * 한국어 해설 — src/components/ui/progress.tsx
 * 0~100 값에 대응하는 수평 진행 막대.
 * Indicator를 translateX로 이동시켜 남은 영역을 숨긴다. 실제 작업 진행률 계산·완료 판단은 외부에서 수행한다.
 * 컴포넌트의 data-slot은 스타일/조합의 연결점이고, className 및 나머지 props는 원래 요소로 전달된다.
 */

import * as React from "react"
import { Progress as ProgressPrimitive } from "radix-ui"

import { cn } from "@/lib/utils"

// value가 없으면 0으로 보고 100-value만큼 왼쪽으로 이동시켜 채워진 비율을 표시한다.
function Progress({
  className,
  value,
  ...props
}: React.ComponentProps<typeof ProgressPrimitive.Root>) {
  return (
    <ProgressPrimitive.Root
      data-slot="progress"
      className={cn(
        "relative flex h-1 w-full items-center overflow-x-hidden rounded-full bg-muted",
        className
      )}
      {...props}
    >
      <ProgressPrimitive.Indicator
        data-slot="progress-indicator"
        className="size-full flex-1 bg-primary transition-all"
        style={{ transform: `translateX(-${100 - (value || 0)}%)` }}
      />
    </ProgressPrimitive.Root>
  )
}

export { Progress }
