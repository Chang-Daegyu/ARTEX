"use client";

/* 한국어 해설: 작업 범위의 승인 기록 연결
 * 공통 ApprovalRecords에 taskId를 전달하여 현재 작업의 도구 승인 기록만 보이도록 하는 얇은 어댑터다.
 * 승인 목록 조회·주기 갱신·허용 또는 거부 처리는 공통 컴포넌트에서 수행한다.
 * 전역 승인 페이지와 작업 탭이 같은 검토 동작을 공유하도록 분리했다.
 * 자세한 안내: docs/ko/modules/web-pages.md
 */

import { ApprovalRecords } from "@/components/approval-records";

export function InterceptTab({ taskId }: { taskId: string }) {
  return <ApprovalRecords key={taskId} taskId={taskId} />;
}
