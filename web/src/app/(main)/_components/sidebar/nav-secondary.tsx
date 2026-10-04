"use client";

/* 한국어 해설: 간단한 보조 링크 목록
 * 부모가 전달한 항목을 아이콘과 제목이 있는 SidebarMenu로 매핑하는 표시 컴포넌트다.
 * 자체 비동기 요청이나 저장 상태가 없으며 링크와 나머지 속성은 호출자가 제공한다.
 * 더 복잡한 활성 경로·하위 메뉴 처리는 nav-main.tsx를 읽는다.
 * 자세한 안내: docs/ko/modules/web-pages.md
 */

import type * as React from "react";

import type { LucideIcon } from "lucide-react";

import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";

export function NavSecondary({
  items,
  ...props
}: {
  items: {
    title: string;
    url: string;
    icon: LucideIcon;
  }[];
} & React.ComponentPropsWithoutRef<typeof SidebarGroup>) {
  return (
    <SidebarGroup {...props}>
      <SidebarGroupContent>
        <SidebarMenu>
          {items.map((item) => (
            <SidebarMenuItem key={item.title}>
              <SidebarMenuButton asChild>
                <a href={item.url}>
                  <item.icon />
                  <span>{item.title}</span>
                </a>
              </SidebarMenuButton>
            </SidebarMenuItem>
          ))}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  );
}
