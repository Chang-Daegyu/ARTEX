/**
 * 한국어 해설 — src/lib/activity-merge.ts
 * 히스토리 조회와 실시간 응답에서 겹친 활동을 seq 기준으로 병합한다.
 * 기존 항목을 Map에 넣고 새 응답으로 같은 seq를 덮어쓴 다음 오름차순으로 반환한다.
 * 입력 배열은 수정하지 않는다. 재전송된 result를 한 번만 보관하므로 화면과 토큰 합계의 중복도 막을 수 있다.
 */

import type { Activity } from "./types";

// History and live responses can overlap or arrive out of order. A persisted
// activity's seq is its identity; replay must not duplicate rows or token totals.
// 같은 seq가 다시 오면 최신 수신 객체로 교체하되, 다른 seq는 모두 유지한다.
// 이는 전달 순서와 중복을 정리하는 함수이며 누락된 서버 이벤트를 스스로 재조회하지는 않는다.
export function mergeActivities(current: Activity[], incoming: Activity[]): Activity[] {
  const bySeq = new Map<number, Activity>();
  for (const activity of current) bySeq.set(activity.seq, activity);
  for (const activity of incoming) bySeq.set(activity.seq, activity);
  return [...bySeq.values()].sort((left, right) => left.seq - right.seq);
}
