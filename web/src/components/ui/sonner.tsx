"use client"

/**
 * 한국어 해설 — src/components/ui/sonner.tsx
 * Sonner 토스트 알림의 테마·아이콘·색상 통일.
 * next-themes에서 받은 모드를 전달하고 성공/정보/주의/오류/로딩 아이콘을 설정한다. 알림 발행은 toast를 호출하는 쪽에서 수행한다.
 * 컴포넌트의 data-slot은 스타일/조합의 연결점이고, className 및 나머지 props는 원래 요소로 전달된다.
 */

import { useTheme } from "next-themes"
import { Toaster as Sonner, type ToasterProps } from "sonner"
import { CircleCheckIcon, InfoIcon, TriangleAlertIcon, OctagonXIcon, Loader2Icon } from "lucide-react"

// 전역 토스트의 테마와 종류별 아이콘을 지정한다. props는 뒤에서 펼치므로 호출자가 기본 설정을 덮어쓸 수 있다.
const Toaster = ({ ...props }: ToasterProps) => {
  const { theme = "system" } = useTheme()

  return (
    <Sonner
      theme={theme as ToasterProps["theme"]}
      className="toaster group"
      icons={{
        success: (
          <CircleCheckIcon className="size-4" />
        ),
        info: (
          <InfoIcon className="size-4" />
        ),
        warning: (
          <TriangleAlertIcon className="size-4" />
        ),
        error: (
          <OctagonXIcon className="size-4" />
        ),
        loading: (
          <Loader2Icon className="size-4 animate-spin" />
        ),
      }}
      style={
        {
          "--normal-bg": "var(--popover)",
          "--normal-text": "var(--popover-foreground)",
          "--normal-border": "var(--border)",
          "--border-radius": "var(--radius)",
        } as React.CSSProperties
      }
      toastOptions={{
        classNames: {
          toast: "cn-toast",
        },
      }}
      {...props}
    />
  )
}

export { Toaster }
