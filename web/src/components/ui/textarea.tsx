/**
 * 한국어 해설 — src/components/ui/textarea.tsx
 * 여러 줄 입력의 자동 높이와 공통 폼 스타일.
 * 내용에 따라 자라되 기본 최대 45vh 이후 스크롤하도록 하여 긴 붙여넣기가 다이얼로그 높이를 넘지 않게 한다.
 * 컴포넌트의 data-slot은 스타일/조합의 연결점이고, className 및 나머지 props는 원래 요소로 전달된다.
 */

import * as React from "react"

import { cn } from "@/lib/utils"

function Textarea({ className, ...props }: React.ComponentProps<"textarea">) {
  return (
    <textarea
      data-slot="textarea"
      className={cn(
        // field-sizing-content auto-grows the textarea to its content; cap the
        // growth (and scroll past it) so a large paste can't blow a dialog past the
        // viewport. Call sites can override via a max-h-* class (twMerge wins).
        "flex field-sizing-content max-h-[45vh] min-h-16 w-full overflow-auto rounded-lg border border-input bg-transparent px-2.5 py-2 text-base transition-colors outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:bg-input/50 disabled:opacity-50 aria-invalid:border-destructive aria-invalid:ring-3 aria-invalid:ring-destructive/20 md:text-sm dark:bg-input/30 dark:disabled:bg-input/80 dark:aria-invalid:border-destructive/50 dark:aria-invalid:ring-destructive/40",
        className
      )}
      {...props}
    />
  )
}

export { Textarea }
