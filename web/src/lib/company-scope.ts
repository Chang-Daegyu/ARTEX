/**
 * 한국어 해설 — src/lib/company-scope.ts
 * 기업 범위 입력을 줄 단위로 분류·검사·정규화하는 순수 함수 모음.
 * 도메인/URL, IPv4·IPv6, CIDR, ICP, 키워드를 구분하고 오류에는 원래 줄 번호를 남긴다.
 * 기존 규칙과 같은 값은 큐로 보관해 사용자가 선택했던 종류를 최대한 유지한다.
 * 이 검사는 UI 입력 보조이며, 서버가 실제 범위 적용과 데이터 검증을 다시 수행해야 한다.
 */

import type { CompanyScopeKind, CompanyScopeRule } from "@/lib/types";

export const MAX_COMPANY_SCOPE_VALUE_LENGTH = 1024;

export type CompanyScopeTextIssue = {
  line: number;
  rule?: CompanyScopeRule;
  error?: string;
};

export type ParsedCompanyScopeText = {
  rules: CompanyScopeRule[];
  errors: Array<{ line: number; error: string }>;
};

export const COMPANY_SCOPE_KINDS = new Set<CompanyScopeKind>(["domain", "ip", "cidr", "icp", "keyword"]);

// 외부 값이 문자열이며 허용된 범위 종류인지 확인하는 TypeScript 타입 가드다.
export function isCompanyScopeKind(value: unknown): value is CompanyScopeKind {
  return typeof value === "string" && COMPANY_SCOPE_KINDS.has(value as CompanyScopeKind);
}

// 4개 숫자 조각, 0~255 범위, 불필요한 선행 0 금지를 검사한다.
function isIPv4(value: string): boolean {
  const parts = value.split(".");
  return (
    parts.length === 4 && parts.every((part) => (part === "0" || /^[1-9]\d{0,2}$/.test(part)) && Number(part) <= 255)
  );
}

// :: 축약과 IPv4 꼬리를 고려해 세그먼트를 세며 잘못된 16진수 조각과 중복 축약을 거른다.
function isIPv6(value: string): boolean {
  if (!value.includes(":") || /[^0-9a-f:.]/i.test(value) || value.includes(":::")) return false;
  const halves = value.split("::");
  if (halves.length > 2) return false;
  const countSegments = (half: string): number | null => {
    if (!half) return 0;
    const segments = half.split(":");
    let count = 0;
    for (const [index, segment] of segments.entries()) {
      if (segment.includes(".")) {
        if (index !== segments.length - 1 || !isIPv4(segment)) return null;
        count += 2;
      } else {
        if (!/^[0-9a-f]{1,4}$/i.test(segment)) return null;
        count++;
      }
    }
    return count;
  };
  const left = countSegments(halves[0]);
  const right = countSegments(halves[1] ?? "");
  if (left === null || right === null) return false;
  return halves.length === 2 ? left + right < 8 : left === 8;
}

// 엄격한 검사를 통과한 주소만 4 또는 6으로 분류하고 나머지는 null로 돌려준다.
function ipVersion(value: string): 4 | 6 | null {
  if (isIPv4(value)) return 4;
  if (isIPv6(value)) return 6;
  return null;
}

// 엄격한 IP 검증에 실패하더라도 주소처럼 생겼는지 추정한다.
// 잘못 쓴 IP를 키워드나 도메인으로 조용히 받아들이지 않기 위한 보조 판정이다.
function looksLikeIPAddress(value: string): boolean {
  const trimmed = value.trim();
  if (trimmed.includes("://") || /\s/.test(trimmed)) return false;
  if ((trimmed.match(/:/g) ?? []).length >= 2) {
    const parts = trimmed.split(":");
    let validSegments = 0;
    for (const part of parts) {
      if (!part) {
        if (validSegments > 0 || trimmed.startsWith("::")) return true;
        continue;
      }
      if (part.length > 4 || !/^[0-9a-f]+$/i.test(part)) return false;
      validSegments++;
      if (validSegments >= 2) return true;
    }
    return false;
  }
  return trimmed.includes(".") && /^[0-9.]+$/.test(trimmed);
}

// 스킴 없는 입력에 임시 http 스킴을 붙여 URL 파서로 host를 추출한다.
// IPv6 대괄호와 마지막 점을 제거하고 소문자로 맞추며 파싱 실패는 null로 돌려준다.
function domainHostname(value: string): string | null {
  try {
    let candidate = value;
    if (value.startsWith("//")) candidate = `http:${value}`;
    else if (!value.includes("://")) candidate = `http://${value}`;
    const url = new URL(candidate);
    return (
      url.hostname
        .replace(/^\[|\]$/g, "")
        .replace(/\.$/, "")
        .toLowerCase() || null
    );
  } catch {
    return null;
  }
}

// 종류별 입력 형식을 검사하고 정상은 빈 문자열, 오류는 표시 메시지를 반환한다.
// CIDR 입력은 현재 UI 정책상 IPv4 /16~32, IPv6 /32~128 범위로 제한한다.
export function companyScopeRuleError(rule: CompanyScopeRule): string {
  const value = rule.value.trim();
  if (!value) return "请填写范围值";
  if (Array.from(value).length > MAX_COMPANY_SCOPE_VALUE_LENGTH) {
    return `最多 ${MAX_COMPANY_SCOPE_VALUE_LENGTH} 个字符`;
  }
  if (rule.kind === "domain") {
    if (/\s/.test(value)) return "请输入不含空格的有效域名或 URL";
    const hostname = domainHostname(value);
    if (!hostname || ipVersion(hostname) !== null) return "请输入有效域名或 URL";
    const labels = hostname.split(".");
    if (
      hostname.length > 253 ||
      labels.length < 2 ||
      labels.some(
        (label) =>
          !label || label.length > 63 || label.startsWith("-") || label.endsWith("-") || !/^[a-z0-9-]+$/i.test(label),
      )
    ) {
      return "请输入有效域名或 URL";
    }
  }
  if (rule.kind === "ip" && ipVersion(value) === null) return "请输入有效 IP";
  if (rule.kind === "cidr") {
    const separator = value.lastIndexOf("/");
    if (separator <= 0) return "请输入 CIDR 网段";
    const address = value.slice(0, separator);
    const prefixText = value.slice(separator + 1);
    const version = ipVersion(address);
    if (version === null || !/^\d+$/.test(prefixText)) return "请输入 CIDR 网段";
    const prefix = Number(prefixText);
    if (version === 4 && (prefix < 16 || prefix > 32)) return "IPv4 网段前缀需为 /16 至 /32";
    if (version === 6 && (prefix < 32 || prefix > 128)) return "IPv6 网段前缀需为 /32 至 /128";
  }
  return "";
}

// 기존 종류 보존을 우선한 뒤 CIDR → IP → 도메인/URL → ICP → 키워드 순서로 한 줄을 분류한다.
// 오류가 있으면 원래 줄 번호와 메시지를 함께 반환해 편집기에서 위치를 안내할 수 있다.
export function classifyCompanyScopeLine(
  raw: string,
  line: number,
  preservedRule?: CompanyScopeRule,
): CompanyScopeTextIssue {
  const value = raw.trim();
  if (Array.from(value).length > MAX_COMPANY_SCOPE_VALUE_LENGTH) {
    return { line, error: `最多 ${MAX_COMPANY_SCOPE_VALUE_LENGTH} 个字符` };
  }
  if (preservedRule) {
    const rule = { kind: preservedRule.kind, value };
    const error = companyScopeRuleError(rule);
    return error ? { line, error } : { line, rule };
  }

  const separator = value.lastIndexOf("/");
  if (
    separator > 0 &&
    (ipVersion(value.slice(0, separator)) !== null || looksLikeIPAddress(value.slice(0, separator)))
  ) {
    const rule: CompanyScopeRule = { kind: "cidr", value };
    const error = companyScopeRuleError(rule);
    return error ? { line, error } : { line, rule };
  }
  if (ipVersion(value) !== null) return { line, rule: { kind: "ip", value } };
  if (looksLikeIPAddress(value)) return { line, error: "请输入有效 IP" };

  const looksLikeDomain = value.includes("://") || (!/\s/.test(value) && value.includes("."));
  if (looksLikeDomain) {
    const hostname = domainHostname(value);
    if (hostname && ipVersion(hostname) !== null) return { line, rule: { kind: "ip", value: hostname } };
    const rule: CompanyScopeRule = { kind: "domain", value };
    const error = companyScopeRuleError(rule);
    return error ? { line, error } : { line, rule };
  }
  if (/icp|备案/i.test(value)) return { line, rule: { kind: "icp", value } };
  return { line, rule: { kind: "keyword", value } };
}

// 빈 줄은 제외하되 원본 줄 번호는 유지한다. 같은 저장값의 기존 규칙은 큐에서 하나씩 꺼내 종류를 보존한다.
// 정상 규칙과 오류를 분리하여 일부 줄이 잘못되어도 전체 입력 상태를 설명할 수 있게 한다.
export function parseCompanyScopeText(
  value: string,
  options: { preservedRules?: CompanyScopeRule[] } = {},
): ParsedCompanyScopeText {
  const nonEmpty = value
    .split(/\r?\n/)
    .map((raw, index) => ({ raw, line: index + 1 }))
    .filter(({ raw }) => raw.trim());
  const preserved = new Map<string, CompanyScopeRule[]>();
  for (const rule of options.preservedRules ?? []) {
    const key = rule.value.trim();
    preserved.set(key, [...(preserved.get(key) ?? []), rule]);
  }
  const issues = nonEmpty.map(({ raw, line }) => {
    const key = raw.trim();
    const queue = preserved.get(key);
    const preservedRule = queue?.shift();
    return classifyCompanyScopeLine(raw, line, preservedRule);
  });
  return {
    rules: issues.flatMap((item) => (item.rule ? [item.rule] : [])),
    errors: issues.flatMap((item) => (item.error ? [{ line: item.line, error: item.error }] : [])),
  };
}

// 중복 비교에 사용할 정규형: 도메인은 host, ICP는 공백 제거, 키워드는 연속 공백 축약, 나머지는 소문자다.
export function normalizeCompanyScopeValue(rule: CompanyScopeRule): string {
  const value = rule.value.trim();
  if (rule.kind === "domain") return domainHostname(value) ?? value.toLowerCase();
  if (rule.kind === "icp") return value.toLowerCase().replace(/\s/gu, "");
  if (rule.kind === "keyword") return value.toLowerCase().split(/\s+/u).filter(Boolean).join(" ");
  return value.toLowerCase();
}
