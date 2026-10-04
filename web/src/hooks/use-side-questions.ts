"use client";

/**
 * 한국어 해설 — src/hooks/use-side-questions.ts
 * 주 대화에 붙은 /btw 보조질문 패널의 비동기 상태 관리.
 * history 조회·2초 폴링·실행 중 질문의 SSE 스냅샷을 같은 목록으로 합친다.
 * 질문 id로 중복을 없애고 sequence로 오래된 응답을 거른다. epoch는 부모 대화 변경/삭제 뒤 늦게 온 콜백을 무효화한다.
 * 제출 실패 시 같은 질문의 요청 ID를 재사용하며, 서버가 수락한 질문이 실패하면 입력 초안을 복구한다.
 */

import { useCallback, useEffect, useRef, useState } from "react";

import { toast } from "sonner";

import { sseUrl } from "@/lib/api";
import { isBtwCommand, type SideExchange, type SideHistory, sideAPI } from "@/lib/side-questions";

// crypto.randomUUID 仅在安全上下文可用(https/localhost);经 IP+http 访问时降级。
// 보안 컨텍스트에서 randomUUID를 사용하고 지원되지 않는 HTTP 접속은 시간/난수 문자열로 대체한다.
function newSideRequestID(): string {
  return (
    globalThis.crypto?.randomUUID?.() ?? `btw-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 12)}`
  );
}

// 같은 질문 id는 sequence가 뒤처지지 않는 응답만 채택하고 질문 순서 ordinal로 정렬한다.
function merge(old: SideExchange[], incoming: SideExchange[]) {
  const byID = new Map(old.map((item) => [item.id, item]));
  for (const item of incoming) {
    if ((byID.get(item.id)?.sequence ?? -1) <= item.sequence) byID.set(item.id, item);
  }
  return [...byID.values()].sort((a, b) => a.ordinal - b.ordinal);
}

// parent가 가리키는 대화/작업의 보조질문 상태를 관리한다.
// current/epoch는 다른 부모의 결과가 섞이는 것을 막고, 질문 상태의 진실은 서버 응답에서 받는다.
export function useSideQuestions(parent: string | null) {
  const [stateParent, setStateParent] = useState(parent);
  const current = stateParent === parent;
  const [open, setOpen] = useState(false);
  const [items, setItems] = useState<SideExchange[]>([]);
  const [snapshot, setSnapshot] = useState<SideHistory["snapshot"]>(null);
  const [draft, setDraft] = useState("");
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(false);
  const [nextCursor, setNextCursor] = useState(0);
  const [error, setError] = useState("");
  const [loadError, setLoadError] = useState("");
  // 부모 변경/삭제마다 증가시키는 세대 번호. 네트워크 요청을 취소하지 못해도 오래된 결과는 적용하지 않는다.
  const epoch = useRef(0);
  const [streamEpoch, setStreamEpoch] = useState(0);
  const cursor = useRef(0);
  const submitting = useRef(false);
  const retry = useRef<{ question: string; id: string } | null>(null);
  const accepted = useRef<{ id: string; question: string } | null>(null);
  const running = current ? items.find((item) => item.status === "running") : undefined;
  const runningID = running?.id;

  // 서버 수락 후 질문이 실패/중단되면 사용자가 다시 입력할 수 있도록 원래 질문을 복원한다.
  const restoreFailedDraft = useCallback((incoming: SideExchange[]) => {
    const pending = accepted.current;
    if (!pending) return;
    const item = incoming.find((entry) => entry.id === pending.id);
    if (!item || item.status === "running") return;
    accepted.current = null;
    if (item.status === "failed" || item.status === "interrupted") {
      setDraft((old) => old || pending.question);
    }
  }, []);

  // 히스토리와 다음 페이지 커서를 받되, 요청 중 부모가 바뀌면 응답을 버린다.
  const load = useCallback(
    async (before = 0) => {
      if (!parent) return;
      const version = epoch.current;
      try {
        const data = await sideAPI.history(parent, before);
        if (version !== epoch.current) return;
        setItems((old) => merge(old, data.items));
        restoreFailedDraft(data.items);
        setSnapshot(data.snapshot);
        setLoadError("");
        if (before || !cursor.current) {
          cursor.current = data.next_cursor;
          setNextCursor(data.next_cursor);
        }
      } catch (err) {
        if (version === epoch.current) setLoadError((err as Error).message);
      }
    },
    [parent, restoreFailedDraft],
  );

  useEffect(() => {
    epoch.current++;
    setStateParent(parent);
    setItems([]);
    setSnapshot(null);
    setNextCursor(0);
    setDraft("");
    setError("");
    setLoadError("");
    setBusy(false);
    submitting.current = false;
    retry.current = null;
    accepted.current = null;
    cursor.current = 0;
    if (!parent) {
      setOpen(false);
      setLoading(false);
      return;
    }
    const version = epoch.current;
    setLoading(true);
    void sideAPI
      .history(parent)
      .then((data) => {
        if (version !== epoch.current) return;
        setItems((old) => merge(old, data.items));
        setSnapshot(data.snapshot);
        setNextCursor(data.next_cursor);
        cursor.current = data.next_cursor;
      })
      .catch((err: Error) => {
        if (version === epoch.current) setError(err.message);
      })
      .finally(() => {
        if (version === epoch.current) setLoading(false);
      });
    return () => {
      epoch.current++;
    };
  }, [parent]);

  useEffect(() => {
    if (!open || !parent) return;
    void load();
    const timer = setInterval(() => void load(), 2000);
    return () => clearInterval(timer);
  }, [open, parent, load]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: Reconnect after a clear attempt invalidates older callbacks.
  useEffect(() => {
    if (!runningID) return;
    const version = epoch.current;
    const stream = new EventSource(sseUrl(`/api/side-questions/${runningID}/events`));
    stream.addEventListener("snapshot", (event) => {
      if (version !== epoch.current) return;
      try {
        const item = JSON.parse((event as MessageEvent).data) as SideExchange;
        if (item.id !== runningID) return;
        setItems((old) => merge(old, [item]));
        restoreFailedDraft([item]);
        if (item.status !== "running") stream.close();
      } catch {
        setError("旁路数据解析失败，请重新打开面板");
      }
    });
    stream.addEventListener("cleared", () => {
      stream.close();
      if (version === epoch.current) setItems([]);
    });
    return () => stream.close();
  }, [runningID, streamEpoch, restoreFailedDraft]);

  // 중복 클릭/이미 실행 중인 질문을 막는다. 서버 수락 전 재시도에는 같은 client_request_id를 유지한다.
  const ask = async (input: string) => {
    const question = input.trim();
    if (!parent || !question || submitting.current || running) return;
    const version = epoch.current;
    submitting.current = true;
    setBusy(true);
    setError("");
    setDraft(question);
    setOpen(true);
    if (retry.current?.question !== question) retry.current = { question, id: newSideRequestID() };
    try {
      const item = await sideAPI.ask(parent, question, retry.current.id);
      if (version !== epoch.current) return;
      setItems((old) => merge(old, [item]));
      accepted.current = { id: item.id, question };
      setDraft("");
      setError("");
      retry.current = null;
      void load();
    } catch (err) {
      if (version === epoch.current) {
        setError((err as Error).message);
      }
    } finally {
      if (version === epoch.current) {
        submitting.current = false;
        setBusy(false);
      }
    }
  };

  // 입력창에서 /btw를 가로채 패널을 열고 명령 본문의 질문만 별도 ask로 보낸다.
  const handleCommand = (text: string, clear: () => void) => {
    if (!parent || !isBtwCommand(text)) return false;
    const question = text.trim().slice(4).trim();
    setOpen(true);
    clear();
    if (question) {
      setDraft(question);
      void ask(question);
    }
    return true;
  };

  // 서버 기록 삭제 전에 epoch를 올려 이전 콜백을 무효화한다.
  // 성공/실패 뒤 다시 조회하고 스트림 세대를 갱신하여 남은 실행 상태와 재동기화한다.
  const clear = async () => {
    if (!parent || submitting.current) return;
    submitting.current = true;
    setBusy(true);
    const version = ++epoch.current;
    try {
      await sideAPI.clear(parent);
      if (version !== epoch.current) return;
      setItems([]);
      setNextCursor(0);
      cursor.current = 0;
      setError("");
      retry.current = null;
      accepted.current = null;
    } catch (err) {
      if (version === epoch.current) toast.error((err as Error).message);
    } finally {
      if (version === epoch.current) {
        submitting.current = false;
        setBusy(false);
        setStreamEpoch((value) => value + 1);
        void load();
      }
    }
  };

  // 실행 중 질문에 취소 요청을 보내며 최종 상태는 이후 조회/SSE 응답으로 확인한다.
  const stop = async () => {
    if (running) {
      try {
        await sideAPI.cancel(running.id);
      } catch (err) {
        toast.error((err as Error).message);
      }
    }
  };
  return {
    open,
    setOpen,
    items: current ? items : [],
    snapshot: current ? snapshot : null,
    draft: current ? draft : "",
    setDraft,
    busy: current && busy,
    loading: !current || loading,
    error: current ? error || loadError : "",
    running,
    nextCursor: current ? nextCursor : 0,
    load,
    ask,
    handleCommand,
    clear,
    stop,
    enabled: !!parent,
  };
}

export type SideQuestions = ReturnType<typeof useSideQuestions>;
