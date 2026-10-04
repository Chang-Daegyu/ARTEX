"use client"

/**
 * 한국어 해설 — src/components/ui/aspect-ratio.tsx
 * 컨텐츠의 가로세로 비율을 유지하는 Radix 컨테이너.
 * ratio 등 원래 props를 그대로 전달한다. 이미지 로드나 리사이즈 작업 자체를 수행하는 컴포넌트는 아니다.
 * 컴포넌트의 data-slot은 스타일/조합의 연결점이고, className 및 나머지 props는 원래 요소로 전달된다.
 */

import { AspectRatio as AspectRatioPrimitive } from "radix-ui"

function AspectRatio({
  ...props
}: React.ComponentProps<typeof AspectRatioPrimitive.Root>) {
  return <AspectRatioPrimitive.Root data-slot="aspect-ratio" {...props} />
}

export { AspectRatio }
