/* 한국어 해설: 전역 승인 기록 페이지
 * taskId 없이 ApprovalRecords를 렌더링하여 작업 전체의 도구 승인 이력을 보여 준다.
 * 공통 컴포넌트가 필터·페이지·대기 요청·결정 API를 관리하므로 이 파일은 경로 진입점 역할만 한다.
 * 작업 한정 기록은 상세 화면의 intercept-tab에서 같은 컴포넌트를 사용한다.
 * 자세한 안내: docs/ko/modules/web-pages.md
 */

import { ApprovalRecords } from "@/components/approval-records";

export default function ApprovalsPage() {
  return (
    <div className="flex min-w-0 flex-1 flex-col p-4 sm:p-6">
      <ApprovalRecords />
    </div>
  );
}
