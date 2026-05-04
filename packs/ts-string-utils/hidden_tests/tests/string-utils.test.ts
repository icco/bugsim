import { test } from "node:test";
import assert from "node:assert/strict";

import {
  countVowels,
  isPalindrome,
  reverseString,
} from "../src/string-utils.ts";

test("reverseString reverses an ASCII string", () => {
  assert.equal(reverseString("hello"), "olleh");
});

test("reverseString handles the empty string", () => {
  assert.equal(reverseString(""), "");
});

test("reverseString reverses a single character to itself", () => {
  assert.equal(reverseString("z"), "z");
});

test("isPalindrome accepts a simple palindrome", () => {
  assert.equal(isPalindrome("racecar"), true);
});

test("isPalindrome ignores case and punctuation", () => {
  assert.equal(isPalindrome("A man, a plan, a canal: Panama"), true);
});

test("isPalindrome rejects a non-palindrome", () => {
  assert.equal(isPalindrome("hello"), false);
});

test("countVowels counts lowercase vowels", () => {
  assert.equal(countVowels("education"), 5);
});

test("countVowels is case-insensitive", () => {
  assert.equal(countVowels("AEIOUaeiou"), 10);
});

test("countVowels returns 0 for a string with no vowels", () => {
  assert.equal(countVowels("rhythm"), 0);
});
