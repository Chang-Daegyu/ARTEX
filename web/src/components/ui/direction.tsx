"use client"

/**
 * 한국어 해설 — src/components/ui/direction.tsx
 * Radix 컴포넌트에 LTR/RTL 문서 방향을 공유한다.
 * direction 속성이 있으면 dir보다 우선하며 useDirection을 그대로 노출한다. 문구 번역 기능과는 별개다.
 * 값은 Context로 전달되며 이 Provider 자체에 화면 레이아웃이나 클래스 병합을 추가하지 않는다.
 */

import * as React from "react"
import { Direction } from "radix-ui"

function DirectionProvider({
  dir,
  direction,
  children,
}: React.ComponentProps<typeof Direction.DirectionProvider> & {
  direction?: React.ComponentProps<typeof Direction.DirectionProvider>["dir"]
}) {
  return (
    <Direction.DirectionProvider dir={direction ?? dir}>
      {children}
    </Direction.DirectionProvider>
  )
}

const useDirection = Direction.useDirection

export { DirectionProvider, useDirection }
