/**
 * 한국어 해설 — src/lib/activity-merge.test.mjs
 * 활동 병합의 회귀 테스트: 겹치는 응답, 늦게 도착한 페이지, 토큰 중복, 입력 불변성을 검증한다.
 * Node 내장 test/assert를 사용하며 네트워크·Go 서버·LLM 없이 순수 병합 함수만 실행한다.
 * 실행 환경은 .ts 모듈을 직접 읽을 수 있는 Node 또는 동등한 TypeScript 로더가 필요하다.
 */

import { mergeActivities } from "./activity-merge.ts";
import assert from "node:assert/strict";
import test from "node:test";

const activity = (seq, kind = "tool_use", extra = {}) => ({
  seq,
  kind,
  worker: "auto",
  ts: "2026-09-10T07:00:00Z",
  summary: String(seq),
  ...extra,
});

// 겹친 페이지의 동일 seq가 한 행만 남고 같은 페이지를 다시 병합해도 결과가 바뀌지 않는지 확인한다.
test("overlapping poll responses render tool 198 only once", () => {
  const first = [activity(197), activity(198)];
  const second = [activity(198), activity(199, "tool_result")];
  const merged = mergeActivities(mergeActivities([], first), second);
  assert.deepEqual(
    merged.map((item) => item.seq),
    [197, 198, 199],
  );
  assert.deepEqual(mergeActivities(merged, second), merged);
});

// 늦게 온 과거 응답 때문에 먼저 받은 새 활동이 사라지지 않는지 확인한다.
test("a late response and overlapping older page preserve newer messages", () => {
  let rows = mergeActivities([activity(198)], [activity(199), activity(200)]);
  rows = mergeActivities(rows, [activity(198), activity(199)]);
  rows = mergeActivities([activity(196), activity(197), activity(198)], rows);
  assert.deepEqual(
    rows.map((item) => item.seq),
    [196, 197, 198, 199, 200],
  );
});

// 중복 result를 한 번만 집계해 입력 토큰이 두 배가 되지 않는지 확인한다.
test("replayed results and duplicate entries do not double-count usage", () => {
  const result = activity(201, "result", { input_tokens: 120, output_tokens: 30 });
  const rows = mergeActivities([result, result], [result, result]);
  assert.equal(rows.length, 1);
  assert.equal(
    rows.reduce((sum, item) => sum + item.input_tokens, 0),
    120,
  );
});

// 동결한 입력을 병합해 원본 배열/객체를 변경하지 않고 별도 결과를 만드는지 확인한다.
test("merging does not mutate either response and keeps distinct IDs", () => {
  const first = Object.freeze([Object.freeze(activity(199))]);
  const second = Object.freeze([Object.freeze(activity(198)), Object.freeze(activity(200))]);
  assert.deepEqual(
    mergeActivities(first, second).map((item) => item.seq),
    [198, 199, 200],
  );
  assert.equal(first.length, 1);
  assert.equal(second.length, 2);
});
