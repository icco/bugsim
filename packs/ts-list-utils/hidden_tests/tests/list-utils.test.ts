import { test } from "node:test";
import assert from "node:assert/strict";

import { chunk, flatten, unique } from "../src/list-utils.ts";

test("chunk groups items into evenly sized chunks", () => {
  assert.deepEqual(chunk([1, 2, 3, 4], 2), [
    [1, 2],
    [3, 4],
  ]);
});

test("chunk emits a smaller trailing chunk when needed", () => {
  assert.deepEqual(chunk([1, 2, 3, 4, 5], 2), [[1, 2], [3, 4], [5]]);
});

test("chunk returns an empty array when size is 0 or negative", () => {
  assert.deepEqual(chunk([1, 2, 3], 0), []);
  assert.deepEqual(chunk([1, 2, 3], -1), []);
});

test("unique removes duplicates and preserves order", () => {
  assert.deepEqual(unique([1, 2, 2, 3, 1, 4]), [1, 2, 3, 4]);
});

test("unique returns an empty array unchanged", () => {
  assert.deepEqual(unique<number>([]), []);
});

test("unique works on strings", () => {
  assert.deepEqual(unique(["a", "b", "a", "c", "b"]), ["a", "b", "c"]);
});

test("flatten concatenates one level of arrays", () => {
  assert.deepEqual(flatten([[1, 2], [3, 4]]), [1, 2, 3, 4]);
});

test("flatten ignores empty inner arrays", () => {
  assert.deepEqual(flatten([[1], [], [2, 3], []]), [1, 2, 3]);
});

test("flatten returns an empty array when input is empty", () => {
  assert.deepEqual(flatten<number>([]), []);
});
