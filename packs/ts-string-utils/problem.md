# String utilities (Unicode-aware, TypeScript)

Implement three exported functions in `src/string-utils.ts`. The catch:
input strings can contain emoji (including ZWJ family/profession sequences),
combining marks, and other multi-code-unit characters. Treat "characters"
as **user-perceived characters** (grapheme clusters), not as UTF-16 code
units.

- `reverseString(s: string): string` — reverse `s` by grapheme cluster.
  Reversing `"a👨‍👩‍👧‍👦b"` must yield `"b👨‍👩‍👧‍👦a"`, not a string with broken
  surrogate pairs or a torn-apart family emoji.

- `isPalindrome(s: string): boolean` — true when `s` reads the same
  forwards and backwards after:
    - lowercasing,
    - stripping anything that isn't a Unicode letter or digit,
    - normalising to NFC,
  - and comparing by grapheme cluster.

- `countVowels(s: string): number` — count grapheme clusters whose first
  base code point (after NFD decomposition) is one of `a, e, i, o, u`,
  case-insensitive. So `"résumé"` counts as **3**, `"naïve"` as **3**, and
  `"😀😀"` as **0**.

Hint: `Intl.Segmenter` and `String.prototype.normalize` exist in Node 22+
and are exactly the right tools for this job.
