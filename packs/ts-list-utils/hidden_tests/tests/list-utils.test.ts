import { test } from "node:test";
import assert from "node:assert/strict";

import { chunk, flatten, unique } from "../src/list-utils.ts";

test("chunk groups items into evenly sized chunks", () => {
  assert.deepEqual(chunk([1, 2, 3, 4], 2), [
    [1, 2],
    [3, 4],
  ]);
});

test("chunk emits a smaller trailing chunk when the array doesn't divide evenly", () => {
  assert.deepEqual(chunk([1, 2, 3, 4, 5], 2), [[1, 2], [3, 4], [5]]);
});

test("chunk returns [] for size <= 0", () => {
  assert.deepEqual(chunk([1, 2, 3], 0), []);
  assert.deepEqual(chunk([1, 2, 3], -7), []);
});

test("chunk returns [items] when size is larger than the array (lodash#896)", () => {
  // Buggy implementations using Math.floor/ceil with division returned []
  // here because they computed 0 chunks. The correct answer is one chunk
  // containing everything.
  assert.deepEqual(chunk([1, 2, 3], 10), [[1, 2, 3]]);
});

test("unique deduplicates NaN to a single entry (SameValueZero)", () => {
  // NaN !== NaN under === / Object.is(NaN, NaN) is true, and Set treats
  // NaN as equal to itself. A naive indexOf-based dedupe leaves duplicate
  // NaNs in the output.
  const out = unique([NaN, 1, NaN, 2, NaN]);
  assert.equal(out.length, 3);
  assert.ok(Number.isNaN(out[0]));
  assert.deepEqual(out.slice(1), [1, 2]);
});

test("unique uses reference identity for objects", () => {
  const a = { x: 1 };
  const b = { x: 1 };
  const out = unique([a, b, a, b]);
  assert.equal(out.length, 2);
  assert.equal(out[0], a);
  assert.equal(out[1], b);
});

test("unique does not mutate the input array", () => {
  const input = [3, 1, 2, 1, 3];
  const snapshot = [...input];
  unique(input);
  assert.deepEqual(input, snapshot);
});

test("flatten concatenates one level deep, preserving deeper nesting", () => {
  // Only flatten *one* level — inner arrays of arrays must remain.
  assert.deepEqual(flatten<unknown>([[1, [2, 3]], [4, [5]]]), [1, [2, 3], 4, [5]]);
});

test("flatten survives 200,000 single-element groups without RangeError (lodash#349)", () => {
  // [].concat(...big) and Array.prototype.flat-via-apply both die on V8
  // with "RangeError: Maximum call stack size exceeded" / argument-count
  // limits around 125k. A loop-based implementation is fine.
  const big = Array.from({ length: 200_000 }, (_, i) => [i]);
  const out = flatten(big);
  assert.equal(out.length, 200_000);
  assert.equal(out[0], 0);
  assert.equal(out[199_999], 199_999);
});
