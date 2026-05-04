import { test } from "node:test";
import assert from "node:assert/strict";

import {
  countVowels,
  isPalindrome,
  reverseString,
} from "../src/string-utils.ts";

test("reverseString reverses plain ASCII", () => {
  assert.equal(reverseString("hello"), "olleh");
});

test("reverseString returns the empty string for an empty input", () => {
  assert.equal(reverseString(""), "");
});

test("reverseString preserves a ZWJ family emoji as a single grapheme", () => {
  // U+1F468 U+200D U+1F469 U+200D U+1F467 U+200D U+1F466 — one user-perceived
  // character that spans 7 UTF-16 code units. A naive [...s].reverse() will
  // tear the family apart and reorder the joiners.
  const family = "\u{1F468}\u200D\u{1F469}\u200D\u{1F467}\u200D\u{1F466}";
  assert.equal(reverseString(`a${family}b`), `b${family}a`);
});

test("reverseString preserves a base+combining-mark grapheme as one unit", () => {
  // "café" with the e+acute as base + combining mark (NFD form) must reverse
  // as a single character, yielding "éfac" not "́efac" with a dangling mark.
  const cafe = "cafe\u0301";
  assert.equal(reverseString(cafe), "e\u0301fac");
});

test("reverseString does not split astral-plane characters", () => {
  // U+1F600 (grinning face) is a single surrogate pair. A naive split
  // produces broken halves; a grapheme-aware reverse keeps it intact.
  const reversed = reverseString("a\u{1F600}b");
  assert.equal(reversed, "b\u{1F600}a");
  assert.equal([...reversed][1], "\u{1F600}");
});

test("isPalindrome accepts the canonical ASCII palindrome", () => {
  assert.equal(isPalindrome("A man, a plan, a canal: Panama"), true);
});

test("isPalindrome rejects an obvious non-palindrome", () => {
  assert.equal(isPalindrome("hello"), false);
});

test("isPalindrome accepts a Unicode palindrome with diacritics", () => {
  // After NFC + lowercase + alphanumeric-only, this collapses to "àbcbà",
  // which is the same forwards and backwards by grapheme cluster.
  assert.equal(isPalindrome("ÀbC, bà!"), true);
});

test("countVowels counts accented vowels by their base letter", () => {
  // r-é-s-u-m-é decomposes to base letters r,e,s,u,m,e → 3 vowels.
  // n-a-ï-v-e decomposes to n,a,i,v,e → 3 vowels.
  // A bare emoji has no Latin vowel base letter → 0.
  assert.equal(countVowels("résumé"), 3);
  assert.equal(countVowels("naïve"), 3);
  assert.equal(countVowels("😀😀"), 0);
});
