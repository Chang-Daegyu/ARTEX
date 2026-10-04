"use client";

/* 한국어 해설: 특정 발견의 도출 경로
 * api.findingLineage로 해당 발견에 연결되는 탐색 노드·관계를 가져와 공통 ExplorationGraph에 전달한다.
 * 전체 작업 그래프가 아닌 서버가 선택한 계보 부분 그래프를 보여 주므로 관련 사실과 의도에서 발견으로 이어진 경로를 읽을 수 있다.
 * 데이터 조회 종료 여부와 빈 그래프 안내를 분리한다.
 * 자세한 안내: docs/ko/modules/web-pages.md
 */

import * as React from "react";

import { ExplorationGraph } from "@/components/exploration-graph";
import { api } from "@/lib/api";
import type { Edge, TaskNode } from "@/lib/types";

// FindingLineageView renders the exploration sub-graph from the task's initial
// node down to this finding's node — the same 攻击链路图 canvas as the task graph,
// scoped to just this finding's lineage.
export function FindingLineageView({ findingId }: { findingId: string }) {
  const [nodes, setNodes] = React.useState<TaskNode[]>([]);
  const [edges, setEdges] = React.useState<Edge[]>([]);
  const [loaded, setLoaded] = React.useState(false);

  React.useEffect(() => {
    let alive = true;
    api
      .findingLineage(findingId)
      .then((g) => {
        if (!alive) return;
        setNodes(g.nodes ?? []);
        setEdges(g.edges ?? []);
      })
      .catch(() => {})
      .finally(() => {
        if (alive) setLoaded(true);
      });
    return () => {
      alive = false;
    };
  }, [findingId]);

  if (loaded && nodes.length === 0) {
    return (
      <p className="text-muted-foreground p-6 text-sm">
        无链路可展示（该漏洞未关联探索节点，或所属任务已删除）。
      </p>
    );
  }

  return (
    <ExplorationGraph
      nodes={nodes}
      edges={edges}
      className="h-[68vh]"
      emptyHint={loaded ? "无链路" : "加载中…"}
    />
  );
}
