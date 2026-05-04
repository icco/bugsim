# String utilities (TypeScript)

Implement the three exported functions in `src/string-utils.ts`:

- `reverseString(s: string): string` — return `s` reversed character-by-character.
- `isPalindrome(s: string): boolean` — true when `s` reads the same forwards and backwards, **case-insensitive** and ignoring any character that is not an ASCII letter or digit (so `"A man, a plan, a canal: Panama"` is a palindrome).
- `countVowels(s: string): number` — count occurrences of `a, e, i, o, u` (case-insensitive).

Tests live under `tests/` and are executed with `node --test --experimental-strip-types tests/`. Press `r` to run them.
