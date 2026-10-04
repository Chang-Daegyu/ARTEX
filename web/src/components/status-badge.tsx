/* 한국어 해설: 도메인별 상태 표시
 * status 라이브러리에서 domain과 value에 해당하는 라벨·색조를 찾아 공통 배지로 렌더링한다.
 * 같은 문자열이라도 작업·엔진·발견 상태의 의미가 달라질 수 있으므로 호출부의 domain이 중요하다.
 * ToneDot은 같은 색조 규칙을 작은 점으로 재사용한다.
 * 자세한 안내: docs/ko/modules/web-pages.md
 */

import { cn } from "@/lib/utils";
import {
  statusMeta,
  toneClasses,
  toneDot,
  type StatusDomain,
  type Tone,
} from "@/lib/status";

export function StatusBadge({
  domain,
  value,
  dot = false,
  className,
}: {
  domain: StatusDomain;
  value: string;
  dot?: boolean;
  className?: string;
}) {
  const meta = statusMeta(domain, value);
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-md border px-2 py-0.5 text-xs font-medium whitespace-nowrap",
        toneClasses[meta.tone],
        className,
      )}
    >
      {dot && (
        <span className={cn("size-1.5 rounded-full", toneDot[meta.tone])} />
      )}
      {meta.label}
    </span>
  );
}

export function ToneDot({ tone }: { tone: Tone }) {
  return <span className={cn("size-2 rounded-full", toneDot[tone])} />;
}
