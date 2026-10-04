"use client";

/* 한국어 해설: 공통 클립보드 복사 버튼
 * copyText 유틸리티를 호출하여 HTTPS 클립보드와 가능한 대체 경로를 공유한다.
 * 성공 후 1.5초 동안 체크 표시를 보여 주고 실패 알림을 처리하며 타이머는 해제 시 정리한다.
 * 복사할 실제 문자열과 버튼 모양은 호출자가 props로 제공한다.
 * 자세한 안내: docs/ko/modules/web-pages.md
 */

import * as React from "react";

import { CheckIcon, CopyIcon } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { cn, copyText } from "@/lib/utils";

type CopyButtonProps = {
  // 要复制的文本;为空则按钮禁用。
  text: string | null | undefined;
  // 复制成功后的 toast 文案,默认「已复制」。
  successMessage?: string;
  label?: React.ReactNode;
  size?: React.ComponentProps<typeof Button>["size"];
  variant?: React.ComponentProps<typeof Button>["variant"];
  className?: string;
};

// CopyButton 统一的「复制到剪贴板」按钮:内置成功/失败反馈,并在 HTTP 非安全上下文
// 下自动降级(见 copyText)。
export function CopyButton({
  text,
  successMessage = "已复制",
  label = "复制",
  size = "sm",
  variant = "outline",
  className,
}: CopyButtonProps) {
  const [copied, setCopied] = React.useState(false);
  const timer = React.useRef<ReturnType<typeof setTimeout> | null>(null);

  React.useEffect(() => {
    return () => {
      if (timer.current) clearTimeout(timer.current);
    };
  }, []);

  async function handleCopy() {
    if (!text) return;
    const ok = await copyText(text);
    if (ok) {
      setCopied(true);
      toast.success(successMessage);
      if (timer.current) clearTimeout(timer.current);
      timer.current = setTimeout(() => setCopied(false), 1500);
    } else {
      toast.error("复制失败，请手动选择文本复制");
    }
  }

  return (
    <Button
      type="button"
      size={size}
      variant={variant}
      className={cn(className)}
      disabled={!text}
      onClick={handleCopy}
    >
      {copied ? <CheckIcon /> : <CopyIcon />}
      {label}
    </Button>
  );
}
