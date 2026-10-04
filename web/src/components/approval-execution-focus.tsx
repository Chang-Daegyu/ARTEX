"use client";

/* 한국어 해설: 승인 기록에서 정확한 실행으로 이동
 * interceptExecution 응답으로 승인 대상 호출의 영속 위치를 찾고 현재 대화의 페이지 캐시를 그 위치까지 과거 방향으로 채운다.
 * useApprovalHistory는 중간 메시지를 생략한 단일 행 삽입 대신 연속 이력을 유지한다.
 * 화면 강조용 상태와 실제 허용/거부 결정은 별개이며 이 컴포넌트는 위치 찾기를 담당한다.
 * 자세한 안내: docs/ko/modules/web-pages.md
 */

import * as React from "react";
import { Button } from "@/components/ui/button";
import { api } from "@/lib/api";
import type { Activity, InterceptExecution } from "@/lib/types";

// Resolve one exact persisted call, independently of transcript pagination.
/* 한국어 흐름: useApprovalFocus
 * 승인 참조를 읽어 영속 실행 원본과 해당 세션을 조회한다. 화면의 현재 페이지에 호출이 없어도 서버가 가진 정확한 호출 위치를 기준으로 이동 상태를 구성한다.
 */
export function useApprovalFocus({ taskId, conversationId }: { taskId?: string; conversationId?: number }) {
  const [state, setState] = React.useState<{
    id: number;
    source?: InterceptExecution;
    error?: string;
    loading: boolean;
  } | null>(null);
  const [revision, setRevision] = React.useState(0);
  // biome-ignore lint/correctness/useExhaustiveDependencies: revision is the explicit retry trigger.
  React.useEffect(() => {
    const q = new URLSearchParams(window.location.search);
    const id = Number(q.get("approval"));
    const matches = conversationId ? q.get("c") === String(conversationId) : q.get("id") === taskId;
    if (!matches || !Number.isSafeInteger(id) || id <= 0) {
      setState(null);
      return;
    }
    let cancelled = false;
    setState({ id, loading: true });
    api
      .interceptExecution(id, conversationId)
      .then((source) => {
        if (cancelled) return;
        if (
          conversationId
            ? source.conversation_id !== conversationId
            : source.task_id !== taskId || source.conversation_id != null
        ) {
          throw new Error("审批来源与当前会话不一致");
        }
        setState({ id, source, loading: false });
      })
      .catch((e) => {
        if (!cancelled) setState({ id, error: (e as Error).message || "无法定位原始执行", loading: false });
      });
    return () => {
      cancelled = true;
    };
  }, [taskId, conversationId, revision]);
  const close = React.useCallback(() => {
    const url = new URL(window.location.href);
    url.searchParams.delete("approval");
    window.history.replaceState(null, "", url);
    setState(null);
  }, []);
  return { state, close, retry: () => setRevision((v) => v + 1) };
}

type HistoryPage = { items: Activity[]; hasMore: boolean };

// Fill the existing transcript continuously back to the call, preserving all
// intervening messages. Never splice an isolated call into a paginated history.
/* 한국어 흐름: useApprovalHistory
 * 대상 호출이 현재 캐시에 없으면 before 커서로 과거 페이지를 차례대로 읽어 병합한다. 중간 이력이 그대로 남으므로 승인 앞뒤 맥락을 검토할 수 있으며 더 이상
 * 진행되지 않는 응답은 반복을 종료한다.
 */
export function useApprovalHistory(
  source: InterceptExecution | undefined,
  loaded: boolean,
  items: Activity[],
  loadPage: (before: number) => Promise<HistoryPage>,
  mergePage: (page: HistoryPage) => void,
) {
  const itemsRef = React.useRef(items);
  itemsRef.current = items;
  const [result, setResult] = React.useState<{ source: InterceptExecution; error?: string }>();
  React.useEffect(() => {
    if (!source || !loaded) return;
    let cancelled = false;
    void (async () => {
      let current = itemsRef.current;
      let before = current[0]?.seq ?? 0;
      while (!current.some((a) => a.seq === source.seq && a.kind === "tool_use")) {
        const page = await loadPage(before);
        if (cancelled) return;
        if (!page.items.length || (before > 0 && page.items[0].seq >= before)) {
          throw new Error("会话中未找到对应工具调用，记录可能已删除");
        }
        mergePage(page);
        current = page.items;
        before = current[0].seq;
        if (!page.hasMore && !current.some((a) => a.seq === source.seq)) {
          throw new Error("会话中未找到对应工具调用");
        }
      }
      if (!cancelled) setResult({ source });
    })().catch((error) => {
      if (!cancelled) setResult({ source, error: (error as Error).message });
    });
    return () => {
      cancelled = true;
    };
  }, [source, loaded, loadPage, mergePage]);
  return {
    ready: !!source && result?.source === source && !result.error,
    error: result?.source === source ? result?.error : undefined,
  };
}

export function ApprovalExecutionFocus({
  focus,
  history,
}: {
  focus: ReturnType<typeof useApprovalFocus>;
  history: ReturnType<typeof useApprovalHistory>;
}) {
  const { state } = focus;
  if (!state) return null;
  const error = state.error || history.error;
  return (
    <div
      role="status"
      className="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b px-4 py-2 text-xs"
    >
      <span className={error ? "text-destructive" : "text-muted-foreground"}>
        {error
          ? `无法定位：${error}`
          : history.ready
            ? `已展开审批 #${state.id} 对应的工具调用`
            : `正在加载审批 #${state.id} 所在的对话位置…`}
      </span>
      <div className="flex gap-2">
        {error ? (
          <Button size="sm" variant="outline" onClick={focus.retry}>
            重试定位
          </Button>
        ) : null}
        <Button size="sm" variant="ghost" onClick={focus.close}>
          取消定位
        </Button>
      </div>
    </div>
  );
}
