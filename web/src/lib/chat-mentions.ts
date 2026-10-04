/**
 * 한국어 해설 — src/lib/chat-mentions.ts
 * 대화 입력의 @ 참조 문법을 해석하고 자산/발견 선택 토큰으로 직렬화한다.
 * activeMention은 커서 앞의 미완성 참조만 찾고 이메일 내부 @는 제외한다.
 * mentionSearch는 종류 별칭과 검색어를 분리하며, selectedMentions는 완성된 토큰의 위치를 돌려준다.
 * 중국어 종류 이름은 현재 토큰 문법과 정규식의 일부이므로 주석 번역과 별개로 원문을 유지한다.
 */

export const mentionKinds = [
  { kind: "finding", label: "漏洞", alias: "finding" },
  { kind: "asset", label: "资产", alias: "asset" },
  { kind: "company", label: "企业", alias: "company" },
  { kind: "endpoint", label: "接口", alias: "api" },
  { kind: "ip", label: "IP", alias: "ip" },
  { kind: "app", label: "应用", alias: "app" },
  { kind: "root_domain", label: "域名", alias: "domain" },
  { kind: "subdomain", label: "子域名", alias: "subdomain" },
  { kind: "service", label: "服务", alias: "service" },
] as const;

export type MentionKind = (typeof mentionKinds)[number]["kind"];
export interface ChatMention {
  kind: MentionKind;
  id: number;
  label: string;
  description: string;
}

// 커서 이전 마지막 @부터 검사한다. 앞 문자가 이메일/경로의 일부처럼 보이면 자동완성 대상으로 취급하지 않는다.
export function activeMention(value: string, caret: number) {
  const before = value.slice(0, caret);
  const start = before.lastIndexOf("@");
  if (start < 0 || (start > 0 && /[\w.+/-]/.test(before[start - 1]))) return null;
  const query = before.slice(start + 1);
  if (/[[\]\r\n@]/.test(query) || query.length > 220) return null;
  return { start, end: caret, query };
}

// 정확한 종류 별칭이 있으면 종류와 나머지 검색어를 돌려주고, 그렇지 않으면 종류 후보 또는 일반 검색어를 만든다.
export function mentionSearch(query: string) {
  const text = query.trimStart().toLowerCase();
  for (const item of mentionKinds) {
    for (const alias of [item.label.toLowerCase(), item.alias]) {
      if (text === alias || text.startsWith(`${alias} `) || (/[^a-z]/.test(alias) && text.startsWith(alias))) {
        return { kind: item.kind, query: query.trimStart().slice(alias.length).trim(), categories: [] };
      }
    }
  }
  const categories = mentionKinds.filter(
    (item) => item.label.toLowerCase().startsWith(text) || item.alias.startsWith(text),
  );
  return { kind: "" as const, query: query.trim(), categories };
}

// 대괄호·개행을 정리하고 레이블 길이를 제한해 @[종류#ID 표시명] 문법이 깨지지 않도록 직렬화한다.
export function mentionToken(item: ChatMention) {
  const kind = mentionKinds.find((entry) => entry.kind === item.kind)?.label ?? "资产";
  const label = item.label
    .replace(/[[\]]/g, (char) => (char === "[" ? "（" : "）"))
    .replace(/\s+/g, " ")
    .slice(0, 100);
  return `@[${kind}#${item.id} ${label}]`;
}

// 완성된 참조의 원문 토큰·표시 레이블·문자 시작 위치를 찾는다. 위치는 선택한 참조만 제거할 때 사용한다.
export function selectedMentions(value: string) {
  return [...value.matchAll(/@\[(漏洞|资产|企业|接口|IP|应用|域名|子域名|服务)#([0-9]+)(?: ([^\]\r\n]*))?\]/g)].map(
    (match) => ({
      token: match[0],
      label: `${match[1]} #${match[2]}${match[3] ? ` · ${match[3]}` : ""}`,
      start: match.index,
    }),
  );
}
