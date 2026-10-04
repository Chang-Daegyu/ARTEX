"use client";

/* 한국어 해설: Simple Icons의 SVG 래퍼
 * SimpleIcon 데이터의 경로·제목과 호출자가 전달한 SVG 속성을 실제 svg 요소로 옮긴다.
 * 브랜드 아이콘을 공통 크기와 접근성 속성으로 표시하기 위한 어댑터다.
 * 아이콘은 외부 파일을 다운로드하는 대신 전달받은 경로 문자열로 그린다.
 * 자세한 안내: docs/ko/modules/web-pages.md
 */

import type * as React from "react";

import type { SimpleIcon as SimpleIconType } from "simple-icons";

import { cn } from "@/lib/utils";

type SimpleIconProps = {
  icon: SimpleIconType;
  className?: string;
} & React.SVGProps<SVGSVGElement>;

export function SimpleIcon({ icon, className, ...props }: SimpleIconProps) {
  const { title, path } = icon;

  return (
    <svg
      viewBox="0 0 24 24"
      aria-label={title}
      aria-hidden="false"
      focusable="false"
      className={cn("size-5 fill-foreground", className)}
      {...props}
    >
      <title>{title}</title>
      <path d={path} />
    </svg>
  );
}
