"use client"

/**
 * 한국어 해설 — src/components/ui/label.tsx
 * 입력 요소와 연결하는 Radix Label의 공통 스타일.
 * htmlFor 등 원래 속성을 전달하여 레이블 클릭과 입력 포커스의 접근성 관계를 유지한다.
 * 컴포넌트의 data-slot은 스타일/조합의 연결점이고, className 및 나머지 props는 원래 요소로 전달된다.
 */

import * as React from "react"
import { Label as LabelPrimitive } from "radix-ui"

import { cn } from "@/lib/utils"

function Label({
  className,
  ...props
}: React.ComponentProps<typeof LabelPrimitive.Root>) {
  return (
    <LabelPrimitive.Root
      data-slot="label"
      className={cn(
        "flex items-center gap-2 text-sm leading-none font-medium select-none group-data-[disabled=true]:pointer-events-none group-data-[disabled=true]:opacity-50 peer-disabled:cursor-not-allowed peer-disabled:opacity-50",
        className
      )}
      {...props}
    />
  )
}

export { Label }
