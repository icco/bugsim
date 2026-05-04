# String utilities (Go) — runes vs bytes

Implement three exported functions in `strutils.go`. The hidden tests
are taken from the canonical
[strings, bytes, runes and characters in Go](https://go.dev/blog/strings)
post and from
[golang/go#22127](https://github.com/golang/go/issues/22127) — the famous
"`len()` returns 459 for my 255-character string" report.

In Go, a `string` is an immutable sequence of bytes. UTF-8 sequences for
characters outside ASCII span 2–4 bytes. Iterating with `for i := 0; i <
len(s); i++` walks bytes; iterating with `for i, r := range s` walks
runes (Unicode code points). The hidden tests assume **rune-level**
semantics for everything below.

- `ReverseString(s string) string` — reverse `s` by **rune**, not by
  byte. Reversing `"héllo"` must yield `"olléh"`, not a string with
  corrupted UTF-8 bytes.

- `IsPalindrome(s string) bool` — true when `s` reads the same forwards
  and backwards by rune, after lower-casing and stripping anything that
  isn't a Unicode letter or digit. So `"Was it a car or a cat I saw?"`
  is a palindrome.

- `RuneCount(s string) int` — return the number of runes in `s`.
  `RuneCount("héllo") == 5` (not `6`, which is `len(s)`'s byte count).

Use the `unicode/utf8`, `unicode`, and `strings` packages from the
standard library. No third-party dependencies are needed.
