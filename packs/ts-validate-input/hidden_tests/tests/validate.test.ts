import { test } from "node:test";
import assert from "node:assert/strict";

import { isValidEmail, normalizePhone } from "../src/validate.ts";

test("isValidEmail accepts plus-tags and multi-label domains", () => {
  // Plus-tag is RFC-allowed and used by Gmail; multi-label TLDs (.co.uk)
  // are common. Several of the most-copied SO regexes get these wrong.
  assert.equal(isValidEmail("name+tag@example.com"), true);
  assert.equal(isValidEmail("first.last@sub.example.co.uk"), true);
});

test("isValidEmail rejects consecutive dots in the local part", () => {
  // RFC allows quoted forms but unquoted "a..b" is invalid; many naive
  // regexes accept it.
  assert.equal(isValidEmail("a..b@example.com"), false);
});

test("isValidEmail rejects leading or trailing dots in the local part", () => {
  assert.equal(isValidEmail(".alice@example.com"), false);
  assert.equal(isValidEmail("alice.@example.com"), false);
});

test("isValidEmail rejects domains with empty or hyphen-bookended labels", () => {
  assert.equal(isValidEmail("alice@example..com"), false);
  assert.equal(isValidEmail("alice@.example.com"), false);
  assert.equal(isValidEmail("alice@example.com."), false);
  assert.equal(isValidEmail("alice@-example.com"), false);
  assert.equal(isValidEmail("alice@example-.com"), false);
});

test("isValidEmail does not catastrophically backtrack on adversarial input", () => {
  // The classic ReDoS pattern /^([a-z]+)+@x\.com$/ takes >1.5s on V8 for
  // a 28-character input that doesn't end in @x.com. A correct,
  // non-backtracking implementation finishes in microseconds. We give a
  // generous 500ms ceiling so the test isn't flaky on slow CI but still
  // catches the obvious vulnerable patterns at n=30.
  const adversarial = "a".repeat(30) + "!";
  const start = process.hrtime.bigint();
  isValidEmail(adversarial);
  const elapsedMs = Number(process.hrtime.bigint() - start) / 1_000_000;
  assert.ok(
    elapsedMs < 500,
    `validation took ${elapsedMs.toFixed(1)}ms — likely catastrophic backtracking`,
  );
});

test("normalizePhone returns E.164 form for typical US inputs", () => {
  assert.equal(normalizePhone("(415) 555-1212"), "+14155551212");
  assert.equal(normalizePhone("+1 415 555 1212"), "+14155551212");
  assert.equal(normalizePhone("1.415.555.1212"), "+14155551212");
  assert.equal(normalizePhone("415-555-1212"), "+14155551212");
});

test("normalizePhone returns null for non-US, malformed, or short inputs", () => {
  // Wrong length, non-digit content, foreign-looking country codes the
  // spec doesn't promise to support, the empty string.
  assert.equal(normalizePhone("415-555-12345"), null);
  assert.equal(normalizePhone("555-1212"), null);
  assert.equal(normalizePhone("+44 20 7946 0958"), null);
  assert.equal(normalizePhone("call me maybe"), null);
  assert.equal(normalizePhone(""), null);
});
