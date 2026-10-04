/**
 * 한국어 해설 — src/lib/types.ts
 * 프런트엔드가 기대하는 API 데이터 계약을 모은 타입 사전.
 * Task는 실행 단위, Asset은 작업 사이에서 공유하는 자산, TaskNode/Edge는 탐색 과정, Finding은 별도 검토 상태를 갖는 발견이다.
 * Activity.seq는 활동 병합의 식별자이며, 토큰 통계는 작업/세션/모델 등 집계 단위를 구분해야 한다.
 * 원본에는 이전 자산 그래프 타입과 현재 통합 자산 타입이 함께 남아 있다. 새 API 연동은 실제 메서드의 반환 타입을 기준으로 읽는다.
 * 물음표는 필드 생략 가능성, 문자열 유니언은 서버와 주고받는 값의 집합이다. 타입 선언 자체로 외부 JSON을 검증하지 않는다.
 */

// ARTEX domain model — types used across the UI.
// Derived from the functional spec (section 7: 关键数据形状).

// 작업 전체의 생명주기 값. 개별 intent의 실행 상태 및 엔진의 exploring/idle 상태와 구분한다.
export type TaskStatus = "created" | "queued" | "running" | "paused" | "done" | "failed" | "timeout";
// planner/worker 엔진의 활동 모드. 작업 자체의 최종 완료 상태와 같은 축이 아니다.
export type EngineMode = "exploring" | "paused" | "stalled" | "idle";

// 작업 목록/상세의 기본 단위. 시간, 목표, 큐/실행 상태, 모델 체인, 직접 연결한 원본 작업을 담는다.
// source_task_ids는 직접 출처 관계이며 전체 연결망을 재귀적으로 상속한다는 뜻은 아니다.
export interface Task {
  id: string;
  name?: string; // 可选任务名称;空/缺省=未命名,展示时回退到描述
  category_id?: number;
  category_name?: string;
  pinned?: boolean;
  pinned_at?: string | null;
  description: string;
  goal: string;
  status: TaskStatus;
  created_at: string;
  created_unix?: number; // created_at as unix seconds (run-duration calc)
  completed_at?: string; // RFC3339 finish time (done/failed); "" if unfinished
  completed_unix?: number; // completed_at as unix seconds (0/undef if unfinished)
  last_activity_unix?: number; // unix seconds of the last activity (0/undef if none)
  paused?: boolean;
  queued?: boolean;
  active?: boolean;
  in_flight?: number;
  findings?: { critical: number; high: number; medium: number; low: number }; // 已登记漏洞数(按严重度分档)
  last_activity?: string;
  stalled?: boolean;
  goals_total?: number;
  goals_met?: number;
  engine_mode?: EngineMode;
  tokens?: TokenTotal; // whole-task token consumption
  llm_profile_id?: number; // LLM profile used; absent = default profile
  llm_profile_ids?: number[]; // ordered task-level failover chain
  active_llm_profile_id?: number; // profile used by the next LLM call
  llm_failover_state?: "default" | "ready" | "chain_exhausted" | string;
  llm_failover_reason?: string;
  source_task_ids?: string[]; // directly related tasks inherited as read-only context
  archive_blocked_by_task_id?: string; // live direct dependent that must be archived first
  company_ids?: number[]; // associated company scopes; current company assets join the task at creation
  coverage_enabled?: boolean; // 资产覆盖度功能开关(创建时定,默认开)；false=不计算/不展示覆盖度
}

// 사용자가 작업을 묶어 관리하는 분류와 그 분류에 속한 작업 수.
export interface TaskCategory {
  id: number;
  name: string;
  task_count: number;
  created_at: string;
  updated_at: string;
}

// 새 작업에 복사할 이름·설명·목표·분류·자산 규칙의 템플릿. 실행 중인 Task와는 별도 리소스다.
export interface TaskTemplate {
  id: number;
  name: string;
  description: string;
  goal: string;
  category_id?: number | null; // 预设分类；null/缺省=无
  intercept_rules?: AssetInterceptRuleInput[]; // 预设的任务级拦截/允许规则
  created_at: string;
  updated_at: string;
}

// 작업 삭제 시 자산·원본 트래픽·파일·발견·LLM 기록을 각각 삭제할지 명시하는 선택값.
export interface DeleteTaskOptions {
  delete_assets: boolean;
  delete_traffic: boolean;
  delete_files: boolean;
  delete_findings: boolean;
  delete_llm_records: boolean;
}

// 삭제 결과의 실제 처리 수와 정리 경고. 자산 삭제와 작업 연결만 해제한 수를 구분한다.
export interface DeleteTaskResult {
  deleted: string;
  assets_deleted: number;
  assets_detached: number;
  traffic_deleted: number;
  files_deleted: boolean;
  findings_deleted: number;
  llm_records_deleted: number;
  cleanup_warning?: string;
}

// 작업 보관·복원·삭제의 대기/실행/실패 상태. 보관 준비 완료는 ready다.
export type TaskArchiveState =
  | "archive_queued"
  | "archiving"
  | "archive_failed"
  | "ready"
  | "restore_queued"
  | "restoring"
  | "restore_failed"
  | "delete_queued"
  | "deleting"
  | "delete_failed";

// 보관된 작업의 호출/토큰 집계. 옛 자료에서는 일부 항목이 없을 수 있어 선택 필드로 선언한다.
export interface TaskArchiveTokenStats {
  calls?: number;
  input_tokens?: number;
  output_tokens?: number;
  cache_read_tokens?: number;
  cache_write_tokens?: number;
}

// 작업 보관 처리 상태와 원래 작업의 설명·분류·출처·남은 시간·검증 해시/크기/집계 메타데이터.
export interface TaskArchive {
  id: number;
  task_id: number;
  state: TaskArchiveState;
  phase: string;
  progress: number;
  error?: string;
  warnings?: string[];
  format_version: number;
  sha256?: string;
  original_size: number;
  compressed_size: number;
  task_name: string;
  task_description: string;
  task_goal: string;
  original_status: TaskStatus;
  category_id?: number;
  category_name?: string;
  source_task_ids: number[];
  remaining_timeout_seconds: number;
  data_counts: Record<string, number>;
  aggregate_stats: {
    tokens?: TaskArchiveTokenStats;
    skills?: Record<string, number>;
    tools?: Record<string, number>;
    findings?: Record<string, number>;
  };
  archived_at?: string;
  requested_at: string;
  created_at: string;
  updated_at: string;
}

// 보관 목록의 페이지 데이터 및 전체 수. items 길이와 전체 total을 혼동하지 않는다.
export interface TaskArchivePage {
  items: TaskArchive[];
  total: number;
  page: number;
  size: number;
}

// 보관/복원/삭제 일괄 요청에서 항목별 성공·큐 등록·오류를 전달한다.
export interface ArchiveBatchItem {
  id: string;
  archive_id?: number;
  ok: boolean;
  queued: boolean;
  error?: string;
}

// ---- Asset graph (global, shared across tasks) ----
// 기존 자산 그래프 표현에 남은 종류 목록. 현재 통합 자산 API의 6가지 종류는 NewAssetType에 있다.
export type AssetType =
  | "company"
  | "domain"
  | "ip"
  | "port"
  | "service"
  | "site"
  | "endpoint"
  | "parameter"
  | "tech"
  | "credential"
  | "data";

// 기존 자산 노드의 관찰/확인/폐기 상태. 발견의 처리 상태 FindingStatus와 별개다.
export type NodeState = "observed" | "confirmed" | "tombstoned";

// 기존 그래프 형태 자산의 이름·정규 키·신뢰도·속성·관찰 시각. 현재 Asset과 구조가 다르다.
export interface AssetNode {
  id: string;
  type: AssetType;
  name: string;
  key: string; // nkey
  value?: string;
  company_id?: string; // 归属公司资产 id；空=未归属
  state: NodeState;
  confidence: number; // 0..1
  attrs?: Record<string, unknown>;
  first_seen: string;
  last_seen: string;
}

// 기존 자산 그래프의 소유·해석·노출·서비스·접점 등 연결 관계 종류.
export type AssetRel =
  | "owns"
  | "resolves"
  | "exposes"
  | "runs"
  | "serves"
  | "has_endpoint"
  | "has_param"
  | "fingerprinted"
  | "authenticates_as"
  | "reachable"
  | "has_subdomain";

// 시작 ID와 끝 ID, 관계 이름으로 구성한 간선. rel에는 자산 관계 또는 탐색 관계가 올 수 있다.
export interface Edge {
  src: string;
  dst: string;
  rel: AssetRel | ExploreRel;
}

// Task asset view — server-side enriched, paginated.
// 기존 작업 자산 화면에서 기술/인증/매개변수 등 연결 객체를 가볍게 참조하는 타입.
export interface TaskAssetRef {
  id: string;
  name?: string;
  key: string;
  attrs?: Record<string, unknown>;
}

// 기존 AssetNode에 기술·인증·매개변수 참조를 덧붙인 화면용 행.
export interface TaskAssetItem extends AssetNode {
  techs?: TaskAssetRef[];
  auth?: TaskAssetRef[];
  params?: TaskAssetRef[];
}

// 기존 자산 화면용 종류별 수·전체 수·현재 페이지 목록.
export interface TaskAssetView {
  counts: Record<string, number>;
  total: number;
  items: TaskAssetItem[];
}

// ---- New unified asset model (new backend) ----
// 현재 통합 모델의 루트 도메인·IP·하위 도메인·앱·서비스·엔드포인트 6종.
export type NewAssetType = "root_domain" | "ip" | "subdomain" | "app" | "service" | "endpoint";

// 현재 통합 자산 레코드. type에 따라 domain/ip/port/url/method 등의 필드가 채워진다.
// task_ids는 공유 자산과 여러 작업의 연결이며 task_source 계열은 특정 작업에서 연결된 경위를 설명한다.
export interface Asset {
  id: number;
  type: NewAssetType;
  company_id?: number;
  task_ids: number[];
  domain?: string;
  root_domain?: string;
  ip?: string;
  c_segment?: string;
  port?: number;
  icp?: string;
  bound_domains?: string[];
  open_ports?: { port: number; service?: string }[];
  record_type?: string;
  record_value?: string[] | string;
  bundle_id?: string;
  app_name?: string;
  category?: string;
  app_description?: string;
  app_icp?: string;
  url?: string;
  service_type?: string;
  service_name?: string;
  favicon_mmh3?: string;
  status_code?: number;
  content_length?: number;
  page_title?: string;
  technologies?: string[];
  auth?: Record<string, unknown>[];
  method?: string;
  params?: Record<string, unknown>[];
  extra?: Record<string, unknown>;
  last_seen: string;
  task_source?: string;
  task_source_summary?: string;
  task_source_node_id?: number;
}

// 의도에 연결된 자산의 표시 정보 및 출처. inherited와 source_task_id로 현재 작업/출처 작업을 구분한다.
export interface IntentAsset {
  intent_id: number | string;
  asset_id: number;
  type: NewAssetType;
  label: string;
  source: string;
  source_summary: string;
  source_node_id?: number;
  source_task_id: number;
  inherited: boolean;
}

// 작업에 자산을 연결한 결과. 요청 수, 새 연결 수, 이미 연결된 수를 나눠 반환한다.
export interface TaskAssetMutation {
  requested: number;
  attached: number;
  existing: number;
}

// 자산 연결과 범위 추가를 함께 처리한 결과. 두 종류의 신규/기존 항목 수를 따로 갖는다.
export interface TaskAssetScopeMutation {
  requested: number;
  assets_linked: number;
  assets_existing: number;
  scopes_added: number;
  scopes_existing: number;
}

// ---- Asset coverage graph (per task) ----
// 力导向「资产覆盖图」的一个节点。key 唯一：资产="a:<id>"、公司="c:<id>"、
// 无资产行的根域名="r:<domain>"。in_scope=false 的是仅用于连线的灰色上下文节点。
// 자산 범위 그림의 노드. key는 a:/c:/r: 네임스페이스로 구분하며 in_scope=false는 연결 설명용 배경 노드다.
// tested는 탐색 기록과 연결되었음을 나타내는 표시값이므로 모든 검사가 완료됐다는 증명은 아니다.
export interface CoverageGraphNode {
  key: string;
  kind: "company" | "root_domain" | "subdomain" | "ip" | "service" | "app" | "endpoint";
  label: string;
  tested: boolean;
  in_scope: boolean;
  asset_id?: number;
  company_id?: number;
  domain?: string;
  root_domain?: string;
  ip?: string;
  url?: string;
  port?: number;
  service_type?: string;
  app_name?: string;
  page_title?: string;
  status_code?: number;
}

// 자산 범위 그림에서 두 key를 잇는 간선. 탐색 원인의 증거 관계와 별도로 표시한다.
export interface CoverageGraphEdge {
  src: string;
  dst: string;
}

// 범위 그래프를 한 번에 렌더링하기 위한 노드/간선 응답.
export interface CoverageGraphData {
  nodes: CoverageGraphNode[];
  edges: CoverageGraphEdge[];
}

// 某资产在本任务探索图里关联到的意图/事实/发现（覆盖图节点抽屉用）。
// 선택한 자산과 연결된 탐색 노드의 간단한 요약·상태·출처.
export interface CoverageAssetRef {
  id: number;
  kind: string;
  state: string;
  summary: string;
  source_task_id?: string;
  inherited?: boolean;
}
// 선택 자산의 연관 기록을 의도/사실/발견으로 나눈 상세 패널 응답.
export interface CoverageAssetRefs {
  intents: CoverageAssetRef[];
  facts: CoverageAssetRef[];
  findings: CoverageAssetRef[];
}

// ---- Workspace file manager (workDir) ----
// 작업 디렉터리 내 파일/폴더 한 항목. path는 상대 경로이며 mtime은 Unix 밀리초다.
export interface WorkspaceEntry {
  name: string;
  path: string; // workspace-relative, forward slashes
  dir: boolean;
  size: number;
  mtime: number; // unix millis
}
// 현재 작업 공간 경로와 그 아래 항목 목록.
export interface WorkspaceListing {
  path: string;
  entries: WorkspaceEntry[];
}
// 파일 읽기 결과. 바이너리 또는 크기 제한 때문에 content가 생략될 수 있다.
export interface WorkspaceFile {
  path: string;
  size: number;
  binary: boolean;
  too_large?: boolean;
  content?: string;
}

// 任务测试范围的一条（覆盖度分母 + 授权边界）。
// 작업의 평가 범위 한 항목. 종류별 실제 값과 자동/에이전트/수동 출처를 갖는다.
// 범위 레코드의 존재만으로 모든 도구의 네트워크 요청이 강제로 차단되는 것은 아니다.
export interface TaskScopeRow {
  id: number;
  task_id: number;
  kind: "company" | "root_domain" | "subdomain" | "ip" | "cidr" | "icp" | "keyword";
  company_id?: number;
  company_name?: string; // 后端 JOIN companies 解析，仅 kind=company 有值
  domain?: string;
  net?: string;
  value?: string;
  source: "auto" | "agent" | "manual";
  reason?: string;
}

// 기업 소유 범위를 표현하는 도메인·IP·네트워크·ICP·키워드 종류.
export type CompanyScopeKind = "domain" | "ip" | "cidr" | "icp" | "keyword";

// 新增企业时提交的结构化资产范围规则。
// 기업 범위를 등록할 때 보내는 최소 입력: 종류와 값.
export interface CompanyScopeRule {
  kind: CompanyScopeKind;
  value: string;
}

// 资产范围写入的结果。errors 是本次提交里不合法的行；warnings 是与本次提交无关、
// 但会让归属结果不符合预期的既有数据问题（如 ip 字段存了主机名的资产）。
// 범위 저장의 추가/건너뜀/오류 수. 제출 오류와 기존 데이터 문제 경고를 분리해 표시한다.
export interface CompanyScopeMutation {
  added: number;
  skipped: number;
  invalid: number;
  errors?: string[];
  warnings?: string[];
}

// 公司资产范围规则的一条（归属唯一真值来源）。
// 서버에 저장된 기업 소유 범위 규칙. 정규 필드와 사용자 원문 raw를 함께 보관해 수정 시 복원한다.
export interface ScopeRow {
  id: number;
  company_id: number;
  kind: CompanyScopeKind;
  domain?: string; // kind=domain 时有值
  net?: string; // kind=ip|cidr 时有值
  value?: string; // kind=icp|keyword 时可能由后端直接返回
  raw: string; // 原始用户输入，用于显示和回填
  reason?: string;
}

// 企业：type=company 的资产节点 + 图标 + 资产计数 + 资产范围规则。
// 기업 메타데이터, 자산 수, 선택적으로 포함한 소유 범위 목록.
export interface Company {
  id: number;
  name: string;
  logo?: string; // 远程图标 URL；空则前端用名称首字母
  asset_count: number;
  scope?: ScopeRow[];
}

// ---- Exploration graph (per task) ----
// 탐색 그래프의 노드 종류. 목표·의도·사실·발견·힌트·요약은 역할이 다르다.
export type ExploreKind = "task" | "begin" | "goal" | "intent" | "fact" | "finding" | "hint" | "digest";
// 목표의 진행 중/달성/포기 상태.
export type GoalState = "open" | "met" | "abandoned";
// 기본 의도 상태 목록. 실제 TaskNode.state는 삭제 등 확장을 수용하도록 string이다.
export type IntentState = "open" | "running" | "paused" | "done" | "blocked" | "exhausted" | "stopped";
// 탐색 그래프 내부 발견 노드 상태. 독립 발견 레코드의 처리 워크플로와 혼동하지 않는다.
export type FindingState = "confirmed" | "dismissed";
// 힌트가 아직 유효한지 소비되었는지 표현한다.
export type HintState = "active" | "consumed";
// 생성·출처·결과·목표 입증·요약 포함 관계. 그래프는 이 이름으로 과정의 연결 의미를 기록한다.
export type ExploreRel = "spawns" | "derived_from" | "yields" | "proves" | "covers";

// 탐색 과정의 공통 노드. payload는 종류별 JSON 문자열일 수 있어 사용하는 화면에서 해석한다.
// state는 여러 노드 종류의 상태를 담고 inherited/source_task_id는 읽기 전용 출처 표시를 돕는다.
export interface TaskNode {
  id: string;
  type: ExploreKind;
  payload?: string;
  priority: number; // 0..10
  state: string; // GoalState | IntentState | FindingState | HintState
  origin: string;
  ts: string;
  source_task_id?: string;
  inherited?: boolean;
  delete_reason?: string; // 意图假删除(state='deleted')时的删除原因
}

// 播报板一页:按创建顺序分页的节点 + 这一页涉及的边 + 边另一端的节点(refs,按 id 索引),
// 这样每条播报都能说清「从哪来、产出了什么」,而不用把整张图拉下来。
// 방송/이력 목록 한 페이지와 관련 간선·이웃 노드 refs·자산 참조. 전체 그래프를 받지 않고도 주변 문맥을 표시한다.
export interface ExplorationNodePage {
  items: TaskNode[];
  total: number;
  page: number;
  size: number;
  edges: Edge[];
  refs: Record<string, TaskNode>;
  // 节点 id → 该节点锚定的资产(播报板展开时顺带展示,含本页节点与其邻居)。
  assets: Record<string, FindingAsset[]>;
}

// 탐색 노드 페이지의 종류/상태/검색어/정렬 조건.
export interface ExplorationNodeQuery {
  page?: number;
  size?: number;
  kinds?: ExploreKind[];
  states?: string[];
  q?: string;
  order?: "asc" | "desc";
}

// 目标管理卡片用的目标(后端已把 payload 拆成 text/vulnclass)。
// 서버가 목표 payload에서 text/vulnclass를 풀어 제공하는 목표 편집 카드용 타입.
export interface TaskGoal {
  id: string;
  text: string;
  vulnclass?: string;
  state: string; // GoalState
  origin?: string;
  ts: string;
}

// 约束管理卡片用的操作约束(allow=允许 / deny=禁止)。
// 사용자 지시 문구가 허용인지 금지인지 구분하는 값.
export type ConstraintKind = "allow" | "deny";
// 작업의 운영 제약 문구와 작성 출처. 실행 권한의 강제 정책 구현과 구분해 읽는다.
export interface TaskConstraint {
  id: string;
  kind: ConstraintKind;
  text: string;
  origin?: string;
  ts?: string;
}

// ---- Findings ----
// 발견의 심각도 4단계. 정렬은 문자열 사전순이 아니라 별도 의미 순서가 필요하다.
export type Severity = "critical" | "high" | "medium" | "low";

// 漏洞处置状态:待处理 / 处理中 / 已确认 / 已处理 / 已修复 / 误报 / 忽略 / 重复 / 风险接受。
// 발견 검토/조치 상태. pending·confirmed·fixed·false_positive 등은 작업 완료나 그래프 노드 상태와 독립적이다.
export type FindingStatus =
  | "pending"
  | "in_progress"
  | "confirmed"
  | "resolved"
  | "fixed"
  | "false_positive"
  | "ignored"
  | "duplicate"
  | "risk_accepted";

// FindingAsset 是一个漏洞绑定的资产(已在后端预渲染 label)。
// 발견과 연결한 자산의 가벼운 ID/종류/표시 레이블 참조.
export interface FindingAsset {
  id: string;
  type: string;
  label: string;
}

// 발견의 유형·심각도·처리 상태·증거·보고서·출처. finding_id는 별도 발견 테이블의 수정 핸들일 수 있다.
// evidence_version과 report_evidence_version은 증거 변경 후 보고서가 오래되었는지 판단하는 데 사용한다.
export interface Finding {
  traffic_count?: number;
  evidence_version?: number;
  report_evidence_version?: number;
  report_stale?: boolean;
  id: string;
  finding_id?: string; // 独立 findings 表的行 id,状态更新的句柄(任务内旧节点可能缺失)
  vulnclass: string;
  name?: string; // 漏洞名称;为空时展示回退到 vulnclass
  severity: Severity;
  status: FindingStatus;
  summary: string;
  evidence: string;
  report?: string; // 详细报告(Markdown);仅详情接口返回,列表为空
  intent_id?: string;
  param_id?: string;
  task_id?: string;
  task_description?: string;
  source_task_id?: string;
  inherited?: boolean;
  assets?: FindingAsset[];
  ts: string;
}

// FindingsPage 是发现列表的服务端分页响应。
// 서버에서 필터링/정렬한 발견 목록 한 페이지 및 전체 수.
export interface FindingsPage {
  items: Finding[];
  total: number;
  page: number;
  page_size: number;
}

// 발견을 작업별로 묶은 집계. 작업 정보와 심각도별 수, 마지막 발견 시각을 제공한다.
export interface FindingGroup {
  task_id: string | number | null;
  task_name?: string; // 可选任务名称;空/缺省=未命名
  task_description: string;
  task_status: string;
  count: number;
  critical: number;
  high: number;
  medium: number;
  low: number;
  last_found_at: string;
}

// 작업 그룹의 페이지 수와 발견 총수를 구분하는 응답.
export interface FindingGroupsPage {
  items: FindingGroup[];
  total: number;
  finding_total: number;
  page: number;
  page_size: number;
}

// 발견에서 추가 탐색을 요청한 결과. 생성된 작업/의도 ID와 대기 여부를 알려준다.
export interface FindingDeepenResponse {
  task_id: string;
  intent_id: string;
  state: IntentState;
  queued: boolean;
}

// FindingStats 是发现全表聚合(统计卡 + 漏洞类型下拉),服务端计算,不受分页影响。
// 발견 전체 집계와 필터 후보. 현재 테이블 페이지의 행만 세어 만든 통계가 아니다.
export interface FindingStats {
  total: number;
  pending: number;
  critical: number;
  high: number;
  medium: number;
  low: number;
  vulnclasses: string[];
  tasks: FindingTaskOption[];
}

// FindingTaskOption 是发现页「按任务」筛选下拉的一项:有漏洞的任务(描述为空表示任务已删除,
// 前端回退展示 id)及其漏洞条数。
// 발견 필터에 표시할 작업과 발견 수. 삭제된 작업은 설명이 없을 수 있어 ID 표시가 필요하다.
export interface FindingTaskOption {
  id: string | number;
  name?: string; // 可选任务名称;空/缺省=未命名
  description: string;
  count: number;
}

// FindingQuery 是发现列表分页/筛选/排序参数。
// 발견 화면의 페이지·심각도·처리 상태·유형·작업·검색·자산 하위 트리 조건.
export interface FindingQuery {
  page: number;
  pageSize: number;
  severity?: "all" | Severity;
  status?: "all" | FindingStatus;
  vulnclass?: string;
  task?: string; // 任务 id;"all"/空 = 不按任务筛选
  query?: string;
  sort?: "severity" | "time";
  // 资产树节点 key;选中一个节点 = 选中它的整棵子树。空 = 不按资产筛选。
  assetScope?: string;
}

// ---- Findings by asset (资产视图) ----
// 발견을 자산별로 묶을 때 쓰는 계층 종류. none은 연결 자산 없는 발견의 묶음이다.
export type FindingAssetKind = "company" | "root_domain" | "subdomain" | "ip" | "service" | "app" | "endpoint" | "none";

// FindingAssetNode 是资产树的一个节点。key 形如 a:<id>(资产)、c:<id>(企业)、
// r:<domain>(库里没有资产行的根域名)、__none__(未关联资产)。
// 발견 자산 트리 노드. self는 직접 연결 수, total은 자손까지 포함해 발견 ID를 중복 제거한 수다.
export interface FindingAssetNode {
  key: string;
  parent?: string;
  kind: FindingAssetKind;
  label: string;
  asset_id?: number;
  company_id?: number;
  self: number; // 直接挂在该资产上的发现数
  total: number; // 含子孙、按发现去重
  critical: number;
  high: number;
  medium: number;
  low: number;
  last_found_at: string;
}

// 발견 자산 트리와 잘림 여부. truncated/dropped_kinds를 확인해야 제한된 목록을 전체로 오해하지 않는다.
export interface FindingAssetTree {
  nodes: FindingAssetNode[];
  finding_total: number;
  truncated: boolean;
  dropped_kinds?: string[];
}

// FINDING_UNASSIGNED_ASSET 与后端 db.FindingUnassignedAsset 对应。
export const FINDING_UNASSIGNED_ASSET = "__none__";

// ---- Activity / sessions ----
// 도구 요청/결과·문자 출력·사용자 입력·라운드·사용량·모델 전환·승인 요청 등 활동 종류.
export type ActivityKind =
  | "tool_use"
  | "tool_result"
  | "text"
  | "thinking"
  | "result"
  | "user"
  | "intent" // LLM-generated exploration objective leading a worker session (UI-synthesized)
  | "round" // planner round boundary marker (engine-emitted)
  | "usage" // live cumulative token usage (per model turn); not rendered
  | "llm_switch" // automatic/manual task-level LLM switch
  | "llm_failover" // task-level provider switch / chain exhaustion audit event
  | "intercept_request"; // user-approval request from the intercept layer

// ChatAttachment 是一次上传的文件:path 相对该会话/任务工作目录(即 agent 的 CWD)。
// 업로드한 파일의 이름/크기/작업 디렉터리 상대 경로. staging에서는 절대 경로가 추가될 수 있다.
export interface ChatAttachment {
  name: string;
  path: string;
  size: number;
  abs?: string; // 绝对路径(scope=staging 暂存上传时返回;建任务前把它写进描述)
}

// 시간순 실행 이력의 한 레코드. seq로 중복 제거하고 tool_use_id로 도구 요청과 결과를 연결한다.
// worker 이름은 실행자, intent_id/main_seg는 실제 UI 세션 구분에 사용되며 두 개념이 같지는 않다.
export interface Activity {
  seq: number;
  intent_id?: string;
  worker: string; // session owner: planner | mainagent | work#1 ...
  ts: string;
  kind: ActivityKind;
  tool?: string;
  tool_use_id?: string;
  is_error?: boolean;
  summary: string;
  detail?: string;
  metadata?: {
    llm_transition?: LLMTransition;
  };
  source_task_id?: string;
  inherited?: boolean;
  main_seg?: number; // main-agent conversation segment (present only on worker="mainagent" rows)
  // token usage (present only on kind='result')
  input_tokens?: number;
  output_tokens?: number;
  cache_read_tokens?: number;
  cache_write_tokens?: number;
}

// 모델 전환 감사 기록에 남길 프로파일의 ID·이름·호환 형식·모델명.
export interface LLMAuditProfile {
  id: number;
  name: string;
  format: string;
  model: string;
}

// 자동/수동 전환 또는 체인 소진의 이유와 이전/다음 모델 프로파일.
export interface LLMTransition {
  mode: "automatic" | "manual" | "exhausted";
  reason: string;
  previous?: LLMAuditProfile;
  next?: LLMAuditProfile;
}

// 한 역할에 실제 적용될 모델과 선택 출처. 설정 값이 있어도 available=false와 사유가 올 수 있다.
export interface TaskLLMResolution {
  profile_id?: number;
  name: string;
  format: string;
  model: string;
  source: "task_chain" | "agent_binding" | "global_profile" | "environment" | "global";
  available: boolean;
  reason?: string;
}

// mainagent/planner/worker 각각에 해석된 모델 설정을 함께 반환한다.
export interface TaskLLMResolutions {
  mainagent: TaskLLMResolution;
  planner: TaskLLMResolution;
  worker: TaskLLMResolution;
}

// ---- Agent triggers (P3 调度，仅自定义 agent) ----
// 사용자 정의 에이전트의 시간/발견/목표/도구/작업 이벤트 조건과 각 조건의 입력 메시지.
export interface AgentTrigger {
  id: number;
  agent_key: string;
  enabled: boolean;
  interval_sec: number; // 定时:每 N 秒(0=不定时)
  on_finding: boolean; // 任意任务发现 finding 时触发
  on_goal_met: boolean; // 任意任务达成目标时触发
  on_task_timeout: boolean; // 任意任务超时时触发
  on_tool_call: boolean; // 选中工具被调用(执行完成)时触发
  on_task_create: boolean; // 任意任务被创建时触发
  interval_message: string; // 各触发条件的独立用户消息
  finding_message: string;
  goal_message: string;
  task_timeout_message: string;
  tool_call_message: string;
  task_create_message: string;
  tool_names: string[]; // on_tool_call 选中的工具 key(至少一个)
  last_fire?: string;
}

// ---- Conversations (chat page) ----
// 현재 대기 또는 실행 중인 재검사와 연결 대화의 최소 참조.
export interface ActiveFindingRetest {
  id: number;
  finding_id: string;
  conversation_id: number;
  status: "pending" | "running";
}

// 재검사 실행 상태와 판정. completed와 fixed/reproduced/inconclusive는 각각 실행 완료와 판정 내용의 다른 축이다.
export interface FindingRetest {
  id: number;
  finding_id: number;
  conversation_id: number | null;
  status: "pending" | "running" | "completed" | "failed" | "stopped";
  verdict: "" | "reproduced" | "fixed" | "inconclusive";
  notes: string;
  summary: string;
  evidence: string;
  error: string;
  created_at: string;
  started_at: string | null;
  finished_at: string | null;
}

// 독립 대화의 에이전트·제목·모델 연결·핀·시각. running은 서버가 계산한 실시간 상태다.
export interface Conversation {
  id: number;
  running?: boolean; // live server state, returned with the conversation list
  agent_key: string;
  title: string;
  llm_profile_id?: number;
  pinned?: boolean;
  pinned_at?: string | null;
  created_at: string;
  updated_at: string;
}

// ---- Backend logs (/logs page) ----
// 서버 로그 한 줄. seq는 표시/스트림 순서용이고 db_id는 영속 로그 행에만 존재할 수 있다.
export interface LogLine {
  seq: number;
  db_id?: number; // server_logs.id; present for DB-persisted lines
  ts: string;
  level: "info" | "warn" | "error";
  tag: string;
  text: string;
}

// 화면에서 세션을 주 대화·계획·작업·시스템으로 분류하는 역할 값.
export type SessionRole = "mainagent" | "planner" | "worker" | "system";
// 화면 세션의 상태 표현. intent 상태와 대화 실행 여부 등을 조합해 사용할 수 있다.
export type SessionStatus = "running" | "paused" | "done" | "blocked" | "exhausted" | "pending" | "stopped" | "deleted";

// Daily token aggregate bucket (GET /api/tokens/daily).
// 날짜별 토큰 집계. 입력/출력과 캐시 읽기/쓰기를 따로 담는다.
export interface DailyTokenBucket {
  date: string; // "YYYY-MM-DD"
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_write_tokens: number;
}

// Per-worker token usage (GET /api/exploration/tokens).
// worker 실행자 이름 기준 토큰 집계. 같은 실행자가 여러 의도를 수행할 수 있다.
export interface TokenUsage {
  worker: string;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_write_tokens: number;
}

// 안정된 세션 키 기준 토큰 집계. 실행 슬롯 이름 대신 main/plan/intent:<id> 같은 단위를 사용한다.
export interface SessionTokenUsage {
  session: string;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_write_tokens: number;
}

// 일괄 작업 제어의 항목별 결과와 최종 상태/대기 여부/오류.
export interface BatchControlItem {
  id: string;
  ok: boolean;
  status?: string;
  queued?: boolean;
  error?: string;
}

// 批量改分类的逐任务结果。失败只可能是任务已被删除，分类本身的写入是原子的。
// 일괄 분류 변경의 작업별 성공/오류 결과.
export interface BatchCategoryItem {
  id: string;
  ok: boolean;
  error?: string;
}

// Whole-task (all agents) token aggregate.
// 작업에 속한 여러 에이전트의 전체 토큰 합계.
export interface TokenTotal {
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_write_tokens: number;
}

// Global per-profile token spend from the llm_usage ledger (GET /api/tokens/usage).
// 전역 llm_usage 원장 기반 프로파일별 사용량 및 호출/작업 수.
export interface ProfileUsage {
  profile_name: string;
  calls: number;
  tasks: number;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_write_tokens: number;
}

// One (profile, UTC day) token bucket for the dashboard's daily chart (new source).
// 한 프로파일과 UTC 날짜 조합의 토큰 버킷.
export interface ProfileDayUsage {
  profile_name: string;
  date: string; // YYYY-MM-DD
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
}

// Response of GET /api/tokens/usage — the dashboard's "new" (llm_usage) token view.
// 대시보드의 프로파일별 합계와 일별 추이를 한 응답으로 전달한다.
export interface UsageStats {
  by_profile: ProfileUsage[];
  daily: ProfileDayUsage[];
}

// Per-model token usage for one task (GET /api/llm/records/by-model), from the
// always-on llm_usage metering ledger. calls = number of LLM calls on this model.
// 작업의 모델별 호출/토큰 집계. 원본 LLM 녹화와 별도의 사용량 원장에서 나오는 통계다.
export interface ModelTokenStat {
  model: string;
  calls: number;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_write_tokens: number;
}

// UI 세션 카드의 제목/역할/상태와 의도·출처·대화 구간 참조.
export interface Session {
  id: string;
  role: SessionRole;
  title: string;
  status: SessionStatus;
  live: boolean;
  last_activity: string;
  intent_id?: string;
  source_task_id?: string;
  inherited?: boolean;
  seg?: number; // main-agent session: which conversation segment (0 = original)
}

// ---- Security ----
// 도구 허용/차단의 간단한 감사 기록. 승인 모델의 상세 입력은 별도 InterceptAudit에 있다.
export interface AuditEntry {
  ts: string;
  tool: string;
  action: "allow" | "block";
  reason?: string;
  command?: string;
}

// 간단 감사 목록 및 출처별 집계.
export interface Audit {
  entries?: AuditEntry[];
  attributions?: Record<string, number>;
}

// ---- Traffic ----
// HTTP 교환 목록의 메타데이터. 요청/응답 본문은 이 타입에 넣지 않아 행 선택 뒤 별도 읽는다.
export interface TrafficExchange {
  id: string;
  ts: string;
  host: string;
  method: string;
  url: string;
  status: number;
  content_type: string;
  resp_len: number;
}

// 트래픽 기능 활성 여부와 필터 페이지. count는 전체, total은 현재 필터 결과 수일 수 있다.
export interface TrafficResp {
  enabled: boolean;
  proxy?: string;
  count?: number; // global total (unfiltered)
  total?: number; // rows matching the current filter (for pagination)
  page?: number;
  size?: number;
  exchanges?: TrafficExchange[];
}

// Full raw request/response of one exchange (lazy-loaded on row select).
// 선택한 HTTP 교환의 요청/응답 원문 문자열.
export interface TrafficDetail {
  req: string;
  resp: string;
}

// One distinct recorded host with its exchange count (target picker).
// 트래픽 대상 선택기에 쓸 host별 기록 수.
export interface TrafficHost {
  host: string;
  count: number;
}

// ---- App settings (runtime toggles) ----
// 서버 실행 옵션의 읽기/쓰기 계약. 일부 키는 설정 여부만 읽고 실제 비밀값은 쓰기 전용으로 보낸다.
// 동시성·원본 기록·프록시·모델 풀·요약·알림 설정의 적용 시점은 각각 서버 구현을 확인한다.
export interface Settings {
  traffic_capture: boolean;
  agent_traffic_binding: boolean; // Agent 自动绑定流量证据，默认关闭；不影响人工绑定
  llm_record: boolean; // LLM 录制开关（默认关）；关闭时不记录任何 LLM 调用
  // Web search. brave_key_set / tavily_key_set reflect whether a key is stored
  // (the values are never returned). On PUT, send the corresponding field to set/clear.
  web_search_enabled: boolean;
  web_search_backend: string; // "ddgs" | "brave-free" | "tavily" | "deepseek"
  brave_key_set: boolean;
  tavily_key_set: boolean;
  // write-only: only sent on PUT to store/clear the key.
  brave_search_api_key?: string;
  tavily_search_api_key?: string;
  // 独立出口代理(http/https/socks5)，用于访问搜索端点；与记录流量的 MITM 代理无关。空=直连。
  web_search_proxy?: string;
  // 全局出口代理(http/https/socks5，可带 user:pass)，所有目标流量走它。开启流量捕获时作为
  // MITM 上游；关闭捕获时直接注入 agent 的 bash/WebFetch。空=直连。
  global_proxy?: string;
  python_interpreter?: string; // 自定义脚本工具的 python 解释器路径(空=运行时检测)
  workers?: number; // 并发工作 agent 数(默认3)；对之后启动的任务生效
  // 任务并发上限:同时「运行中」的任务数上限。关闭=不限;开启后新建任务超限则排队,有空位自动启动。
  task_concurrency_enabled?: boolean; // 默认 false
  task_concurrency_limit?: number; // 开启后默认 5
  // LLM 轮询(故障转移)。默认关；开启后「未指定模型」的 agent 在当前配置不可用
  // （余额不足/key 失效/限流/服务异常）时自动切到下一个配置。
  llm_pool_enabled?: boolean; // 默认 false
  // 绑定了指定配置的 agent/任务失败时是否也回落到轮询链。默认 false = 绑定即独占。
  llm_pool_bind_fallback?: boolean;
  // 操作约束注入范围(默认都开):把任务的 allow/deny 约束拼进对应 agent 的系统提示。
  constraints_inject_planner?: boolean;
  constraints_inject_worker?: boolean;
  // 实验功能:noa 模型驱动上下文压缩(默认关)。开启后平台接入的四类 agent(planner/
  // worker/主 agent/对话)由 noa 接管上下文压缩,取代内置 compaction;每 run 读一次,对
  // 之后启动的 run 生效。
  noa_compaction?: boolean;
  // ---- 漏洞 IM 推送（渠道本身是独立资源，见 /api/notify/*，这里只有三项全局配置）----
  notify_enabled?: boolean; // 推送总开关，默认开；用于维护期一键止血
  notify_public_base_url?: string; // 漏洞详情回链的外部访问地址；空=消息不带回链
  notify_digest_interval_min?: number; // 汇总模式周期（分钟），默认 30
}

// ---- 漏洞 IM 推送 ----

// NotificationFilter 是渠道的过滤条件，字段全部可选，缺省即不过滤。
// 后端对所有字段都不加校验：配置畸形时按「命中」处理（宁可多推不可漏推）。
// 알림 채널의 심각도·작업·자산·유형·상태 변경 조건. 빈 조건은 해당 차원의 제한 없음이다.
export interface NotificationFilter {
  min_severity?: string; // "" | low | medium | high | critical
  task_ids?: number[]; // 空=不限；非空则要求与漏洞所属任务有交集
  asset_ids?: number[]; // 空=不限；非空则要求与漏洞锚定资产有交集
  vulnclass_include?: string[]; // 空=全收；非空则要求漏洞类型命中任一关键词（大小写不敏感子串）
  vulnclass_exclude?: string[]; // 命中任一关键词即排除（排除优先于包含）
  on_status_change?: boolean; // 是否也接收漏洞处置状态变更事件
}

// NotificationChannel 是一个渠道实例。config 的字段随 kind 而异，
// 且凭据字段在读取时被替换成 "__masked__" 开头的掩码值——原样回传即表示「不改」。
// 알림 채널 하나의 종류·전송 방식·구성·필터·속도 제한. 마스킹된 비밀값을 유지한 채 되돌려 보내면 변경하지 않는 계약이다.
export interface NotificationChannel {
  id: number;
  name: string;
  kind: string;
  enabled: boolean;
  mode: "realtime" | "digest";
  config: Record<string, unknown>;
  filter: NotificationFilter;
  rate_per_min: number;
  created_at: string;
  updated_at: string;
  // secret_keys 由后端按渠道类型给出，前端据此渲染密码框与「留空即不改」提示，
  // 不硬编码任何渠道知识。
  secret_keys: string[];
}

// NotificationKind 是 /api/notify/meta 返回的渠道类型元数据。
// 서버가 알려주는 알림 종류별 기본 속도와 비밀 필드 이름.
export interface NotificationKind {
  kind: string;
  default_rate_per_min: number;
  secret_keys: string[];
}

// 알림 설정 화면의 채널 메타데이터·전역 기본값·대기열 통계.
export interface NotificationMeta {
  kinds: NotificationKind[];
  enabled: boolean;
  public_base_url: string;
  digest_interval_min: string;
  defaults: { digest_interval_min: number };
  stats: {
    channels: number;
    channels_on: number;
    pending: number;
    failed: number;
    sent_today: number;
    backlog_age_ms: number;
  };
}

// NotificationDelivery 是一条投递记录，用于投递历史与失败重发。
// 발견 이벤트 한 건의 채널 전송 이력. 시도 횟수·다음 시각·실패 이유는 재시도 UI에 사용한다.
export interface NotificationDelivery {
  id: number;
  finding_id: string;
  event_kind: string; // finding_created | finding_status_changed
  channel_id: number;
  channel_name: string;
  channel_kind: string;
  state: "pending" | "sending" | "sent" | "failed" | "skipped";
  attempts: number;
  last_error: string;
  batch_id?: number;
  created_at: string;
  sent_at?: string;
  next_attempt_at: string;
  title: string;
  severity: string;
}

// ---- LLM config ----
// 모델 엔드포인트의 형식·모델명·속도 제한·출력 한도·컨텍스트 한도·재시도·풀 우선순위.
// max_tokens는 응답 길이, context_window_k는 문맥 용량 힌트라 서로 대체할 수 없다.
export interface LLMProfile {
  id: string;
  name: string;
  format: "openai" | "anthropic" | "openai-responses";
  base_url?: string;
  proxy?: string;
  model: string;
  api_key_hint?: string;
  rate_per_second: number;
  rate_per_minute: number;
  context_window_k?: number;
  // 思考开关(thinking.type): ""=不发送(默认) | "disabled"=关闭 | "enabled"=开启
  thinking_type?: string;
  // 思考强度: ""=不发送(默认) | "low"/"medium"/"high"/"xhigh"/"max"
  reasoning_effort?: string;
  is_default: boolean;
  // 轮询顺位：越大越先被选中。激活配置恒为链首，与本值无关。
  priority?: number;
  // true = 不作为故障转移目标（仍可被 agent/任务显式绑定使用）。
  pool_exclude?: boolean;
  // true（默认）= 流式(SSE) | false = 真·非流式(stream:false，一次性返回)。
  streaming?: boolean;
  // 单次回复的输出上限(token)。0 = 不发送该字段，由服务端默认值决定。
  // 注意与 context_window_k 区分：后者是模型总容量，只在本地用于压缩阈值。
  max_tokens?: number;
  // 上限用哪个请求字段名，仅 format="openai" 有意义：
  // ""=max_tokens(默认) | "max_completion_tokens"(OpenAI 推理模型只认它)
  max_tokens_field?: string;
  // 自定义会话头名：非空时每次请求带该 HTTP 头，头值=当前会话/意图的 session id。
  // ""=不发送。用于按 session-id 头做提示缓存/粘性路由的网关。
  session_header_key?: string;
  // 本配置对重试的覆盖（建连/空响应/同 provider 安全窗口）。留空/全 0 = 跟随全局策略。
  retry?: LLMRetryOverride;
}

// ---- LLM 重试策略 ----
// 一层重试的两个旋钮。两者都是「0 = 未配置」：
//   attempts    0=用默认次数 | -1=关闭该层重试 | >0=重试次数
//   interval_ms 0=用默认的指数退避 | >0=改用这个固定毫秒间隔
// 한 재시도 계층의 횟수와 간격. attempts=0은 기본값, -1은 비활성, 양수는 지정 횟수다.
export interface LLMRetryRule {
  attempts: number;
  interval_ms: number;
}

// 单个 LLM 配置能覆盖的三层（都是「跟着端点走」的重试）。
// 연결·빈 응답·출력 전달 전 스트림 실패 재시도의 프로파일별 덮어쓰기.
export interface LLMRetryOverride {
  connect: LLMRetryRule; // 建连重试：连接重置/超时/429/5xx，流开始前
  empty: LLMRetryRule; // 空响应重试：完成但没有任何内容（仅 openai 格式）
  stream: LLMRetryRule; // 同 provider 安全窗口重试：未交付输出前的断流重放
}

// 全局策略 = 上面三层的默认值 + 两层只有全局的：
//   breaker 轮询熔断（attempts=连续几次瞬时失败熔断，interval_ms=固定冷却时长）
//   intent  意图重跑（worker 以 model_error 收场后整条意图重跑）
// 공통 재시도 3종에 전역 회로 차단기와 전체 intent 재실행 정책을 더한다.
export interface LLMRetryPolicy extends LLMRetryOverride {
  breaker: LLMRetryRule;
  intent: LLMRetryRule;
}

// ---- LLM 轮询（故障转移）----
// 一个配置在轮询链中的位置与健康状态。state:
//   ok       正常
//   degraded 有连续失败但未达熔断阈值
//   tripped  已熔断，冷却期内被跳过（cooldown_secs 为剩余秒数）
// 모델 풀의 프로파일 우선순위·현재 선택 여부·연속 실패/차단 상태·남은 냉각 시간.
export interface LLMPoolMember {
  profile_id: string;
  name: string;
  model: string;
  format: string;
  priority: number;
  active: boolean; // 是否为当前激活配置（恒为链首）
  excluded: boolean; // pool_exclude：不参与轮询
  state: "ok" | "degraded" | "tripped";
  fails: number;
  trips: number;
  cooldown_secs: number;
  last_error?: string;
  last_at?: string;
}

// 모델 풀 활성/명시적 연결의 fallback 여부와 현재 체인 목록.
export interface LLMPoolStatus {
  enabled: boolean;
  bind_fallback: boolean;
  chain: LLMPoolMember[];
}

// ---- Agents ----
// 에이전트 역할 구성. 모델·턴/시간 한도·도구 기능·트리거 병렬성·연결 수를 담는다.
export interface Agent {
  id: string;
  key: string; // 内置为 goals/planner/mainagent/worker；自定义为用户自定 key
  name: string;
  description?: string;
  role: string;
  builtin: boolean;
  enabled: boolean;
  llm_profile_id?: number | null; // 绑定的 LLM 配置；null/absent = 跟随任务/会话/全局
  max_turns?: number; // 0 = 无限制
  run_seconds?: number; // worker 单次运行墙钟上限(秒)；0 = 无限制
  web_search?: boolean; // 是否启用网络搜索(受系统全局开关门控)
  interactive_shell?: boolean; // 是否启用交互式 shell(持久 PTY 会话工具族)
  // P3 触发后处理策略(仅自定义 agent 有意义)
  trigger_run_mode?: "serial" | "parallel"; // 串行排队 / 每次触发各自并发一个会话
  trigger_merge_mode?: "by_task" | "all" | "none"; // 仅 serial：同任务合并 / 全部合并 / 不合并
  trigger_max_parallel?: number; // 仅 parallel：每 agent 并发上限；0=不限
  // 绑定数量(仅列表接口返回)：可见 MCP / 可见 Skill / 绑定工具
  mcp_count?: number;
  skill_count?: number;
  tool_count?: number;
}

// 프롬프트 템플릿에서 쓸 변수의 이름·설명·예시·값 공급 출처.
export interface PromptVar {
  name: string;
  description: string;
  example: string;
  source: "exploration" | "runtime" | "distilled";
}

// 프롬프트 수정 이력의 버전·시각·메모·실제 템플릿 본문.
export interface PromptVersion {
  version: number;
  ts: string;
  note: string;
  template_text: string;
}

// 에이전트 설정과 프롬프트/변수/버전/가시성/마무리 정책을 모은 편집 화면 응답.
export interface AgentDetail {
  agent: Agent;
  prompt: string;
  variables: PromptVar[];
  versions: PromptVersion[];
  visibility: { mcp: number[]; skill: string[] };
  // 可绑定的 LLM 配置候选(供「默认模型」下拉)；当前绑定见 agent.llm_profile_id
  llm_profiles?: { id: number; name: string; model: string; is_default: boolean }[];
  wrapup_prompt?: string; // 已保存的收尾提示词(空=用内置默认)
  wrapup_default?: string; // 内置默认收尾提示词(占位/恢复默认)
  wrapup_max_turns?: number; // 已保存的收尾轮数(0=用内置默认)
  wrapup_max_turns_default?: number; // 内置默认收尾轮数(供 "0=默认N" 提示)
  // 任务级超时收尾词(仅 worker/planner，task_timeout_wrapup_supported=true 时才显示该分区)
  task_timeout_wrapup_supported?: boolean;
  task_timeout_wrapup_prompt?: string;
  task_timeout_wrapup_default?: string;
  task_timeout_wrapup_max_turns?: number;
  task_timeout_wrapup_max_turns_default?: number;
}

// ---- MCP ----
// 서버가 관리할 MCP 연결의 전송 방식·프로세스/URL·환경·TLS 옵션. 브라우저 실행 프로세스 설정이 아니다.
export interface MCPServer {
  id: number;
  name: string;
  transport: "stdio" | "http" | "sse";
  command?: string;
  args: string[];
  env: Record<string, string>;
  url?: string;
  enabled: boolean;
  insecure?: boolean; // http: skip TLS cert verification (self-signed servers)
  tools?: string[]; // mcp_tools_cache (names only, for the count)
}

// MCP가 공개한 도구 이름과 설명. 실제 전체 입력 스키마와 실행은 서버 연결 계층에서 처리한다.
export interface MCPTool {
  name: string;
  description: string;
}

// ---- Skills ----
// Fields align with the agentskills.io open specification.
// description covers both "what the skill does" and "when to use it".
// Skill 디렉터리 메타데이터·파일 목록·MCP 해제 대상·사용 통계. name은 디렉터리 이름과 연결되는 키다.
export interface SkillItem {
  name: string; // unique key = directory name
  description?: string; // required per spec; covers what + when to use
  license?: string; // optional: SPDX identifier or free text
  compatibility?: string; // optional: environment requirements
  mcps?: string[]; // MCP server names this skill unlocks on load
  files: string[]; // files in the skill directory
  // 调用统计（skill_usage 账本）。从未被调用过的 skill：calls=0、last_used 缺省。
  calls: number;
  tasks: number; // 加载过它的任务数（chat 会话不计入）
  usage_agents: string[]; // 加载过它的 agent key
  last_used?: string;
}

// SkillCall 是一次 Skill() 调用（单个 skill 的最近调用列表）。
// Skill 호출 1회의 시각·에이전트·작업/세션·인자 길이. task_id=0은 작업 밖 대화일 수 있다.
export interface SkillCall {
  ts: string;
  agent_key: string;
  task_id: number; // 0 = 非任务场景（对话会话）
  session_id: string;
  args_len: number;
}

// MissingSkill 是被点名但不存在的 skill —— "想用但没有"的缺口。
// 에이전트가 호출하려 했으나 존재하지 않은 Skill의 수요 통계.
export interface MissingSkill {
  skill: string;
  calls: number;
  agents: string[];
  last_used?: string;
}

// ---- Tools (内置工具目录) ----
// key + handler live in Go; only these fields are page-editable. system tools lock
// the key and the parameter *structure* (name/type/required) — the per-param
// description/default and the agent binding are what move.
// 내장/사용자 도구의 설명·스키마·에이전트 연결·실행 명세. deferred는 추가 도구 검색/실행 경로를 통한 스키마 지연 공개다.
export interface Tool {
  key: string;
  system: boolean;
  description: string;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  schema: Record<string, any>; // full JSON-Schema (object with properties)
  agents: string[]; // bound agent keys
  enabled: boolean;
  kind?: "builtin" | "shell" | "command" | "script" | "http"; // 自定义工具类型
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  exec?: Record<string, any>; // 自定义工具执行规格(kind!=builtin)
  deferred?: boolean; // schema 延迟(SearchExtraTools/ExecuteExtraTool)
  calls?: number; // persistent runtime invocation count (older APIs may omit it)
}

// ---- Stats ----
// 대시보드의 기본 자산/발견/엔진/모델 구성 상태와 활성 작업 요약.
export interface Stats {
  assets: number;
  engine_mode: EngineMode;
  llm_configured: boolean;
  roe_enabled: boolean;
  findings_confirmed: number;
  active_task?: Partial<Task>;
}

// ---- Intercept Rules ----
// 도구 규칙의 즉시 허용/거부/사용자 승인 요청 결과.
export type InterceptAction = "allow" | "deny" | "ask";
// 규칙을 도구 이름에 적용할지 입력 JSON에 적용할지 구분한다.
export type InterceptMatchTarget = "tool_name" | "tool_input";
// 규칙 패턴의 단순 문자열 또는 정규식 해석 방식.
export type InterceptMatchType = "string" | "regex";

// 도구 실행 규칙의 순서·대상·일치 방식·패턴·결정·대기 시간 설정.
export interface InterceptRule {
  id: number;
  name: string;
  enabled: boolean;
  priority: number;
  match_target: InterceptMatchTarget;
  match_type: InterceptMatchType;
  pattern: string;
  action: InterceptAction;
  message: string;
  timeout_enabled: boolean;
  timeout_seconds: number;
  timeout_action: "deny" | "allow";
  created_at: string;
  updated_at: string;
}

// ---- Asset Intercept Rules（资产拦截：全局黑名单） ----
// IP·도메인·URL·CIDR 등 자산 패턴의 해석 방식.
export type AssetInterceptKind =
  | "exact_domain"
  | "exact_ip"
  | "exact_url"
  | "fuzzy_domain"
  | "fuzzy_ip"
  | "fuzzy_url"
  | "cidr";

// action 仅用于任务级规则：block=拦截(禁止测试) allow=允许(白名单)。
// 작업별 자산 규칙의 block/allow. 도구 인터셉터의 deny/ask와 다른 값이다.
export type AssetInterceptAction = "block" | "allow";

// 任务级资产拦截/允许规则的录入项（创建任务、任务详情编辑使用）。
// 작업 생성/수정에서 보내는 자산 규칙의 동작·종류·패턴·설명·활성 값.
export interface AssetInterceptRuleInput {
  action: AssetInterceptAction;
  kind: AssetInterceptKind;
  pattern: string;
  note: string;
  enabled: boolean;
}

// 저장된 자산 규칙. 전역 규칙은 차단용이며 작업별 규칙에 action이 추가될 수 있다.
export interface AssetInterceptRule {
  id: number;
  enabled: boolean;
  action?: AssetInterceptAction; // 全局规则不带此字段（恒为拦截）；任务级规则区分 block/allow
  kind: AssetInterceptKind;
  pattern: string;
  note: string;
  builtin: boolean;
  created_at: string;
  updated_at: string;
}

// 승인 요청의 도구 입력·원인·연결 작업/대화·대기/허용/거부/시간 초과 상태.
export interface InterceptPending {
  decision_source?: "rule" | "model" | "unknown" | "";
  id: number;
  rule_id?: number;
  conversation_id?: number;
  task_id?: string;
  agent_name: string;
  tool_name: string;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  tool_input: Record<string, any>;
  status: "pending" | "allowed" | "denied" | "timeout";
  reason: string; // 规则 message 或模型判定理由(模型判定带 [模型] 前缀)
  decided_at?: string;
  created_at: string;
}

// JudgeConfig: 模型兜底审批(仅当没有任何拦截规则命中时由模型判断)的全局配置。
// 도구 규칙에 이어 사용할 선택적 모델 판단의 설정. 모델 실패 동작과 사람 승인 시간 초과 동작을 별도로 지정한다.
export interface JudgeConfig {
  enabled: boolean;
  profile_id: number; // 0 = 跟随激活/默认配置
  prompt: string; // 判定提示词;GET 未设置时后端回填内置模板全文
  timeout_seconds: number; // 模型调用超时
  fail_action: "allow" | "ask" | "deny"; // 模型出错/超时/不可解析时的回退
  ask_timeout_seconds: number; // 模型判 ask 转人工后的审批等待超时
  ask_timeout_action: "allow" | "deny"; // 审批超时后的默认动作
}

// JudgeUsage: 模型兜底审批(judge 通道)的累计 token 用量 + 近 N 天每日序列。
// Judge 호출의 UTC 일별 호출/입력/출력 토큰 수.
export interface JudgeDayUsage {
  date: string; // YYYY-MM-DD (UTC)
  calls: number;
  input_tokens: number;
  output_tokens: number;
}
// Judge 전체 호출/토큰 수 및 일별 목록. worker의 일반 실행 통계와 분리한 채널이다.
export interface JudgeUsage {
  calls: number;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_write_tokens: number;
  daily: JudgeDayUsage[];
}

// 승인 이력을 상태와 결정 출처로 좁히는 조건.
export interface InterceptApprovalFilter {
  status?: InterceptPending["status"];
  decision_source?: "rule" | "model" | "unknown";
}

// InterceptApprovalRow enriches InterceptPending with conversation/task and rule context.
// 승인 요청에 대화 제목·에이전트 키·규칙 이름을 더한 목록 행.
export interface InterceptApprovalRow extends InterceptPending {
  conv_title: string; // "" if no linked conversation
  conv_agent_key: string; // "" if no linked conversation
  rule_name: string; // "" if rule was deleted
}

// ── 资产同步 (ScopeSentry 数据源) ──────────────────────────────────────────────
// ScopeSentry 프로젝트 선택 정보. id와 원본 대소문자 필드 이름은 외부 데이터 계약을 유지한다.
export interface SSProject {
  id: string; // MongoDB ObjectID — used as filter.project
  name: string;
  logo?: string;
  AssetCount?: number;
  tag?: string;
}

// ScopeSentry 작업 선택 정보. name은 외부 task 필터에 쓰이며 원본 creatTime 필드도 보존한다.
export interface SSTask {
  id: string;
  name: string; // used as filter.task
  status?: number;
  progress?: number;
  creatTime?: string;
  endTime?: string;
}

// ConvTokenSummary — one conversation's token total (+ profile/date) for merging
// chat usage into the dashboard token stats. GET /api/tokens/conversations.
// 대화 한 건의 모델 프로파일/날짜/토큰 합계를 대시보드에 합칠 때 사용하는 자료.
export interface ConvTokenSummary {
  llm_profile_id: number | null;
  created_at: string;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_write_tokens: number;
}

// ---- Command recording (Bash execution history) ----
// 도구 실행 이력의 원본 입력 JSON·출력·실패 상태. command라는 이름이 Bash만 뜻하지는 않는다.
export interface CommandRecord {
  id: number;
  exploration_id: number;
  worker: string;
  tool: string;
  command: string; // raw tool input (JSON)
  output: string;
  is_error: boolean;
  created_at: string;
}

// 单个工具的调用统计（/commands/stats）；errors 为其中失败的次数。
// 도구별 전체 호출 수와 그중 오류 수.
export interface ToolStat {
  tool: string;
  total: number;
  errors: number;
}

// ---- LLM recording ----
// 모델 요청 한 건의 시각·지연·세션·작업·사용량·상태 메타데이터.
export interface LLMRecordItem {
  id: number;
  ts: string;
  model: string;
  profile_name: string;
  session_id: string;
  task_id: string;
  worker: string;
  latency_ms: number;
  input_tokens: number;
  output_tokens: number;
  cache_read: number;
  cache_write: number;
  status: string;
  error?: string;
}

// LLM 기록의 정규화된 본문과 실제 전송 원문. raw_request/raw_response에는 스키마와 SSE 프레임 등이 남을 수 있다.
export interface LLMRecordDetail extends LLMRecordItem {
  request_body: string;
  response_body: string;
  // provider 实际收发的 HTTP 原文：请求为 buildBody() 发出的完整 body（含工具
  // schema），响应为原始 SSE 帧。上面的 request_body/response_body 是归一化视图，
  // 丢弃了工具 schema 与 tool_use 块。旧记录为空。
  raw_request?: string;
  raw_response?: string;
}

// One distinct task with its LLM-record count (task picker on the records page).
// LLM 원본 기록을 가진 작업과 기록 수.
export interface LLMTask {
  task_id: string;
  count: number;
}

// The exact JSON sent to the review model, retained for all model verdicts.
// 판단 모델에 실제 보낸 입력 스냅샷의 버전별 구조. 옛 version의 history는 보존하되 v3는 실행 이력을 보내지 않는다.
export interface InterceptReviewInput {
  version: number;
  background?: {
    // worker_summary is retained only for immutable v2/v3 snapshots.
    source: "user_message" | "worker_summary";
    text: string;
    truncated?: boolean;
  };
  // Version 1 snapshots are immutable and remain readable in historical audits.
  task?: {
    task_id: number;
    description: string;
    goal: string;
    constraints: { id: number; kind: string; text: string; origin: string; created_at: number }[];
    truncated?: boolean;
  };
  working_directory?: string;
  worker_intent?: string;
  turn_input?: string;
  background_truncated?: boolean;
  // Legacy v1/v2 snapshots only; v3 never sends execution history.
  history?: {
    tool_use_id: string;
    tool: string;
    arguments_preview: string;
    result: string;
    status: "succeeded" | "failed";
    truncated?: boolean;
  }[];
  history_truncated?: boolean;
  correlation?: "exact" | "ambiguous" | "unavailable";
  tool_name: string;
  arguments: Record<string, unknown>;
}

// Immutable review snapshot plus separately recorded execution outcome.
// 판단 당시 입력·결정·설정 해시와 별도로 기록한 후속 실행 결과. 승인되었다는 것과 실제 실행 성공은 서로 다르다.
export interface InterceptAudit {
  model_input?: InterceptReviewInput;
  model_input_digest?: string;
  run_id?: string;
  tool_use_id?: string;
  correlation: "exact" | "ambiguous" | "unavailable";
  input_digest: string;
  user_message: string;
  user_truncated?: boolean;
  context:
    | { kind: string; tool?: string; tool_use_id?: string; text: string; is_error?: boolean; truncated?: boolean }[]
    | null;
  context_truncated?: boolean;
  captured_at: string;
  model_fallback?: boolean;
  initial_action: "allow" | "ask" | "deny";
  initial_reason: string;
  effective_action?: "allow" | "deny";
  decision_reason?: string;
  rule_name?: string;
  config_digest?: string;
  profile_id?: number;
  execution_status: "not_started" | "not_executed" | "awaiting_result" | "succeeded" | "failed" | "unknown";
  output?: string;
  output_truncated?: boolean;
  execution_ended_at?: string;
}
// 승인 목록 행에 상세 감사 스냅샷을 결합한다. 옛 기록은 audit가 없을 수 있다.
export interface InterceptDetail extends InterceptApprovalRow {
  audit: InterceptAudit | null;
}

// 연결 트래픽의 설명 역할: 정상 대조·입증·재확인·보조 자료.
export type TrafficEvidenceRole = "baseline" | "proof" | "verification" | "supporting";
// 원본 트래픽을 증거로 연결할 때 보내는 ID·역할·메모.
export interface TrafficEvidenceRef {
  traffic_id: string;
  role?: TrafficEvidenceRole;
  note?: string;
}
// 별도로 보관한 HTTP 증거의 메타데이터와 요청/응답 해시·길이. 원본 traffic 행의 수명과 구분한다.
export interface TrafficEvidenceSnapshot {
  id: string;
  source_traffic_id: string;
  captured_at: number;
  url: string;
  method: string;
  status: number;
  content_type: string;
  req_head?: string;
  resp_head?: string;
  req_hash: string;
  resp_hash: string;
  req_len: number;
  resp_len: number;
}
// 발견과 보관 증거 스냅샷을 잇는 관계. 역할·메모·순서와 스냅샷 요약을 함께 반환한다.
export interface FindingTrafficBinding {
  id: string;
  finding_id: string;
  snapshot_id: string;
  role: TrafficEvidenceRole;
  note: string;
  position: number;
  created_at: string;
  snapshot: TrafficEvidenceSnapshot;
}
// 발견에 붙은 증거 목록과 현재 증거/보고서 버전. 두 버전이 다르면 보고서 최신성 확인이 필요하다.
export interface FindingTraffic {
  finding_id: string;
  version: number;
  report_version: number;
  bindings: FindingTrafficBinding[];
}
// 큰 증거 본문을 나눠 읽기 위한 내용·현재/다음 오프셋·전체 길이·잘림/바이너리 표시.
export interface EvidenceBodyPreview {
  content: string;
  offset: number;
  total: number;
  next_offset: number;
  truncated: boolean;
  binary: boolean;
}
// 선택한 증거 연결과 요청/응답 미리보기를 모은 상세 응답.
export interface FindingTrafficDetail {
  binding: FindingTrafficBinding;
  request: EvidenceBodyPreview;
  response: EvidenceBodyPreview;
}

/** GET /api/update/check —— 当前版本与 GitHub 最新正式版的比较结果。 */
// 현재 실행 버전과 최신 릴리스의 비교·플랫폼 패키지 존재·복구 가능 여부. 오류/비교 불가도 정상적인 응답 상태다.
export interface UpdateCheck {
  /** 当前运行的版本；开发构建为 "dev" 或 git describe 的带后缀形式。 */
  current: string;
  /** 运行形态。docker 下换装只作用于容器可写层，重建容器会退回镜像版本。 */
  mode: "docker" | "binary";
  os: string;
  arch: string;
  repo: string;
  /** 是否存在可回滚的上一版本（artex.old）。 */
  has_backup: boolean;
  /** 本次启动时自更新自举的结论（换装失败 / 已回滚等），无事发生时为空。 */
  boot_notice?: string;
  rolled_back?: boolean;
  /** 查询 GitHub 失败时给出原因，此时下面的字段都不会有。 */
  error?: string;
  latest?: string;
  notes?: string;
  html_url?: string;
  published_at?: string;
  /** 当前平台对应的发布包名，以及该 Release 是否真的带了它。 */
  asset?: string;
  asset_available?: boolean;
  size?: number;
  has_update?: boolean;
  /** 双方版本号是否可比较；开发构建为 false，此时禁用一键更新。 */
  comparable?: boolean;
  /** comparable 为 false 时的说明。 */
  reason?: string;
}

/** /api/update/stream 推送的一条更新进度。 */
// 다운로드·검증·압축 해제·준비·실패 단계의 SSE 진행 메시지. percent는 다운로드 중에만 유효하다.
export interface UpdateProgress {
  phase: "idle" | "downloading" | "verifying" | "extracting" | "staged" | "failed";
  /** 仅下载阶段有意义（0-100）；其余阶段为 -1。 */
  percent: number;
  message: string;
  version?: string;
  error?: string;
}

// Original execution selected from an approval, never submitted to the reviewer.
// 승인 항목에서 선택한 원래 실행 세션/활동의 조회 결과. 판정 모델에 다시 보내는 입력을 뜻하지 않는다.
export interface InterceptExecution {
  conversation_id: number | null;
  task_id: string | null;
  session: string;
  seq: number;
  items: Activity[];
}
