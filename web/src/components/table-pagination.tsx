"use client";

/* 한국어 해설: 표의 페이지 이동 컨트롤
 * 총 페이지 수와 현재 페이지를 받아 앞/뒤 및 페이지 번호 버튼을 구성한다.
 * pageWindows는 현재 위치와 양끝을 남기고 중간은 생략 기호로 표시해 페이지가 많아도 컨트롤이 길어지지 않게 한다.
 * 실제 데이터 조회·페이지 크기 반영은 부모 콜백이 맡는다.
 * 자세한 안내: docs/ko/modules/web-pages.md
 */

import { useEffect, useRef } from "react";

import { ChevronLeftIcon, ChevronRightIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Pagination, PaginationContent, PaginationEllipsis, PaginationItem } from "@/components/ui/pagination";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

interface TablePaginationProps {
  page: number; // 1-based
  pageSize: number;
  total: number; // total (after filtering)
  onPageChange: (p: number) => void;
  onPageSizeChange: (s: number) => void;
  pageSizeOptions?: number[];
}

// pageWindows returns the sequence of page numbers / ellipsis to render.
function pageWindows(page: number, total: number): (number | "...")[] {
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1);
  const out: (number | "...")[] = [1];
  if (page > 3) out.push("...");
  const lo = Math.max(2, page - 1);
  const hi = Math.min(total - 1, page + 1);
  for (let i = lo; i <= hi; i++) out.push(i);
  if (page < total - 2) out.push("...");
  out.push(total);
  return out;
}

export function TablePagination({
  page,
  pageSize,
  total,
  onPageChange,
  onPageSizeChange,
  pageSizeOptions = [10, 20, 50],
}: TablePaginationProps) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const safePage = Math.min(Math.max(1, page), totalPages);
  const from = total === 0 ? 0 : (safePage - 1) * pageSize + 1;
  const to = Math.min(safePage * pageSize, total);
  const onPageChangeRef = useRef(onPageChange);
  onPageChangeRef.current = onPageChange;

  useEffect(() => {
    if (page !== safePage) onPageChangeRef.current(safePage);
  }, [page, safePage]);

  return (
    <div className="flex flex-wrap items-center justify-between gap-2 border-t px-4 py-2 lg:px-6">
      <div className="text-muted-foreground flex items-center gap-2 text-xs">
        <Select value={String(pageSize)} onValueChange={(v) => onPageSizeChange(Number(v))}>
          <SelectTrigger size="sm" className="h-7 w-16">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {pageSizeOptions.map((s) => (
              <SelectItem key={s} value={String(s)}>
                {s}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <span>条/页</span>
        {total > 0 ? (
          <span className="tabular-nums">
            {from}–{to} / 共 {total} 条
          </span>
        ) : (
          <span>共 0 条</span>
        )}
      </div>

      {totalPages > 1 && (
        <Pagination className="mx-0 w-auto justify-end">
          <PaginationContent>
            <PaginationItem>
              <Button
                variant="ghost"
                size="icon-sm"
                disabled={safePage === 1}
                onClick={() => onPageChange(safePage - 1)}
                aria-label="上一页"
              >
                <ChevronLeftIcon className="size-4" />
              </Button>
            </PaginationItem>
            {pageWindows(safePage, totalPages).map((p, i) =>
              p === "..." ? (
                <PaginationItem key={`el-${i}`}>
                  <PaginationEllipsis />
                </PaginationItem>
              ) : (
                <PaginationItem key={p}>
                  <Button
                    variant={p === safePage ? "outline" : "ghost"}
                    size="icon-sm"
                    onClick={() => onPageChange(p)}
                    aria-current={p === safePage ? "page" : undefined}
                  >
                    {p}
                  </Button>
                </PaginationItem>
              ),
            )}
            <PaginationItem>
              <Button
                variant="ghost"
                size="icon-sm"
                disabled={safePage === totalPages}
                onClick={() => onPageChange(safePage + 1)}
                aria-label="下一页"
              >
                <ChevronRightIcon className="size-4" />
              </Button>
            </PaginationItem>
          </PaginationContent>
        </Pagination>
      )}
    </div>
  );
}
