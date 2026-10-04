/**
 * 한국어 해설 — src/lib/task-assets.ts
 * 자산 종류와 작업 연결 출처를 화면 표시 문자열로 바꾼다.
 * agent/anchor/manual/company 등 출처는 자산이 이 작업과 연결된 경로를 설명하며 자산 자체의 유형과 다르다.
 * 알 수 없는 출처는 원래 문자열을 그대로 반환하여 새 서버 값을 숨기지 않는다.
 */

import type { NewAssetType } from "@/lib/types";

const ASSET_TYPE_LABELS: Record<NewAssetType, string> = {
  app: "应用",
  endpoint: "接口",
  ip: "IP",
  root_domain: "根域名",
  service: "服务",
  subdomain: "子域名",
};

const TASK_ASSET_SOURCE_LABELS: Record<string, string> = {
  agent: "Agent 发现",
  anchor: "黑板锚点",
  api: "资产 API",
  company: "企业关联",
  legacy: "历史关联",
  manual: "人工加入",
  system: "系统关联",
  task: "任务初始化",
};

// 현재 통합 자산 모델의 6가지 종류에 대한 표시명을 반환한다.
export function taskAssetTypeLabel(type: NewAssetType): string {
  return ASSET_TYPE_LABELS[type];
}

// 작업 연결 출처의 알려진 키를 설명 이름으로 바꾸고 미등록 값은 그대로 보존한다.
export function taskAssetSourceLabel(source: string): string {
  return TASK_ASSET_SOURCE_LABELS[source] ?? source;
}
