/**
 * 한국어 해설 — src/lib/chat-mentions.test.mjs
 * @ 참조의 커서 처리·이메일 제외·중국어/영어 별칭·토큰 삭제 위치를 검사한다.
 * 테스트의 중국어 문자열은 현재 직렬화 문법을 검증하는 입력/기대값이므로 번역 대상 설명과 구분한다.
 * 한 참조를 제거해도 이웃 토큰이 보존되는지까지 확인하는 순수 함수 테스트다.
 */

import assert from "node:assert/strict";
import test from "node:test";
import { activeMention, mentionSearch, mentionToken, selectedMentions } from "./chat-mentions.ts";

// 이메일·이미 완성된 참조·개행은 제외하고 커서 앞의 유효한 미완성 참조만 찾는지 검사한다.
test("mention trigger supports Chinese and cursor placement without hijacking email", () => {
  assert.equal(activeMention("user@example.com", 16), null);
  assert.equal(activeMention("已选 @[漏洞#1 X]", 12), null);
  assert.deepEqual(activeMention("查看@漏洞 后面的文字", 5), { start: 2, end: 5, query: "漏洞" });
  assert.equal(activeMention("@漏洞\n下一行", 8), null);
});

// 중국어/영문 별칭과 일반 검색어를 올바른 종류/검색어로 나누는지 검사한다.
test("categories, Chinese aliases, IP and keyword search", () => {
  assert.equal(mentionSearch("").categories.length, 9);
  assert.equal(mentionSearch("漏").categories[0].kind, "finding");
  assert.equal(mentionSearch("漏洞").kind, "finding");
  assert.equal(mentionSearch("漏洞SQL注入").query, "SQL注入");
  assert.equal(mentionSearch("ip 192.0.2.1").kind, "ip");
  assert.equal(mentionSearch("接口 GET /api").query, "GET /api");
  assert.equal(mentionSearch("acme.com").kind, "");
});

// 레이블의 괄호/공백 정규화와 선택 토큰 삭제 후 이웃 토큰 보존을 검사한다.
test("tokens roundtrip labels and removing one reference preserves its neighbors", () => {
  const first = mentionToken({ kind: "finding", id: 12, label: "标题[1]\n描述" });
  const second = mentionToken({ kind: "ip", id: 13, label: "192.0.2.1" });
  const value = `分析 ${first} 和 ${second}`;
  const selected = selectedMentions(value);
  assert.equal(selected.length, 2);
  assert.equal(selected[0].label, "漏洞 #12 · 标题（1） 描述");
  const next = value.slice(0, selected[0].start) + value.slice(selected[0].start + selected[0].token.length);
  assert.equal(selectedMentions(next)[0].token, second);
});
