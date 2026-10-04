/* 한국어 해설: 알림 통계 숫자의 표시 조각
 * 라벨·값·설명·색조를 받아 작은 통계 타일을 렌더링한다.
 * formatBacklog는 밀리초 단위의 대기량을 사람이 읽기 쉬운 시간 크기로 표시할 뿐 큐의 처리 시간을 예측하지 않는다.
 * 데이터 조회 없이 알림 페이지가 계산/수신한 props만 사용한다.
 * 자세한 안내: docs/ko/modules/web-pages.md
 */

import { Card, CardContent } from "@/components/ui/card";

export function StatTile({ label, value, hint, tone }: { label: string; value: string; hint?: string; tone?: string }) {
  return (
    <Card size="sm" className="gap-1">
      <CardContent>
        <p className="text-muted-foreground text-xs">{label}</p>
        <p className={`text-lg font-semibold ${tone === "red" ? "text-rose-600" : ""}`}>{value}</p>
        {hint && <p className={`text-xs ${tone === "red" ? "text-rose-600" : "text-muted-foreground"}`}>{hint}</p>}
      </CardContent>
    </Card>
  );
}

// formatBacklog 把积压毫秒数渲染成人看得懂的量级。
export function formatBacklog(ms: number): string {
  if (!ms) return "—";
  if (ms < 60_000) return `${Math.round(ms / 1000)} 秒`;
  if (ms < 3_600_000) return `${Math.round(ms / 60_000)} 分钟`;
  return `${(ms / 3_600_000).toFixed(1)} 小时`;
}
