import { test } from "node:test";
import assert from "node:assert/strict";

import { isValidEmail, normalizePhone } from "../src/validate.ts";

test("isValidEmail accepts a typical address", () => {
  assert.equal(isValidEmail("nat@example.com"), true);
});

test("isValidEmail rejects a missing @", () => {
  assert.equal(isValidEmail("nat.example.com"), false);
});

test("isValidEmail rejects whitespace inside the address", () => {
  assert.equal(isValidEmail("nat @example.com"), false);
  assert.equal(isValidEmail("nat@ example.com"), false);
});

test("isValidEmail rejects domains without a dot", () => {
  assert.equal(isValidEmail("nat@localhost"), false);
});

test("normalizePhone strips formatting from a valid US number", () => {
  assert.equal(normalizePhone("(415) 555-1212"), "4155551212");
});

test("normalizePhone accepts dot/space separated digits", () => {
  assert.equal(normalizePhone("415.555.1212"), "4155551212");
  assert.equal(normalizePhone("415 555 1212"), "4155551212");
});

test("normalizePhone returns null for the wrong digit count", () => {
  assert.equal(normalizePhone("555-1212"), null);
  assert.equal(normalizePhone("1-415-555-1212"), null);
  assert.equal(normalizePhone(""), null);
});
