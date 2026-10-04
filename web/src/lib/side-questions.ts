/**
 * 한국어 해설 — src/lib/side-questions.ts
 * 보조질문 API 계약과 /btw 명령 인식.
 * 질문은 부모 대화 경로에 연결되고 독립 id·진행 상태·답변 sequence·문맥 요약 정보를 가진다.
 * history 응답은 배열/null/커서를 먼저 정규화해 React 상태 갱신 중의 예외를 줄인다.
 * 요청 ID는 재시도 시 서버 중복 방지에 사용할 수 있도록 ask에 함께 전달한다.
 */

import { http } from "@/lib/api";

export interface SideModel {
  model: string;
  name: string;
  format: string;
  profile_id: number;
}
export interface SideExchange {
  id: string;
  ordinal: number;
  client_request_id: string;
  question: string;
  answer: string;
  status: "running" | "completed" | "failed" | "cancelled" | "interrupted";
  error?: string;
  model: SideModel;
  snapshot_at: string;
  created_at: string;
  sequence: number;
  context?: {
    phase?: "preparing" | "summarizing_history" | "compressing_snapshot" | "retrying" | "answering";
    recent_exchanges: number;
    history_summarized: boolean;
    snapshot_summarized: boolean;
    estimated_input_tokens?: number;
    input_budget?: number;
    output_tokens?: number;
    overflow_retried?: boolean;
  };
}
export interface SideHistory {
  items: SideExchange[];
  current: SideExchange | null;
  next_cursor: number;
  snapshot: { captured_at: string; model: SideModel; available: boolean; reason: string } | null;
}

// 부모 경로에 이미 /api가 포함되어도 공통 http가 중복 접두사를 붙이지 않도록 한 번 제거한다.
async function request<T>(path: string, method = "GET", body?: unknown): Promise<T> {
  return http<T>(path.replace(/^\/api/, ""), { method, body: body === undefined ? undefined : JSON.stringify(body) });
}

export const sideAPI = {
  // 历史必须归一化后再交给调用方:items 一旦不是数组,消费端的 setState updater 会抛,
  // 而 React 会把 updater 的异常推迟到 render 阶段重抛 —— 那时调用方的 catch 已经够不着,
  // 整页直接被错误边界接管。
  // 응답 모양을 React setState 전에 정규화한다. 렌더 단계로 넘어간 updater 오류는 호출부 catch로 잡기 어렵기 때문이다.
  history: async (parent: string, before = 0) => {
    const data = await request<Partial<SideHistory>>(`${parent}/side-questions?before=${before}`);
    return {
      items: Array.isArray(data?.items) ? data.items : [],
      current: data?.current ?? null,
      next_cursor: Number(data?.next_cursor) || 0,
      snapshot: data?.snapshot ?? null,
    } satisfies SideHistory;
  },
  ask: (parent: string, question: string, client_request_id: string) =>
    request<SideExchange>(`${parent}/side-questions`, "POST", { question, client_request_id }),
  clear: (parent: string) => request(`${parent}/side-questions`, "DELETE"),
  cancel: (id: string) => request(`/api/side-questions/${id}/cancel`, "POST"),
};

// 앞뒤 공백을 무시하되 /btw 다음은 공백이나 끝이어야 한다. /btwXYZ 같은 일반 문자열은 명령이 아니다.
export function isBtwCommand(text: string) {
  return /^\/btw(?:\s|$)/.test(text.trim());
}
