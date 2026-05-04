# Email + phone validation (TypeScript) — the realistic version

The hidden tests are taken from real RFC 5322 corner cases and the
[heybounce.io](https://www.heybounce.io/blog/stackoverflow-most-copied-email-regexes)
analysis of "the 10 most copied email regexes on Stack Overflow" — half
of which were measurably wrong on real inboxes.

Implement two helpers in `src/validate.ts`:

- `isValidEmail(s: string): boolean` — return true iff `s` is a
  syntactically valid email address under these (deliberately strict)
  rules:
  - exactly one `@`
  - the local part:
    - is between 1 and 64 characters,
    - contains only `[A-Za-z0-9._+-]`,
    - has no leading or trailing `.`,
    - has no consecutive `.` (so `a..b@x.com` is invalid),
  - the domain part:
    - is split by `.` into one or more labels,
    - every label is 1–63 characters of `[A-Za-z0-9-]` (no leading/trailing hyphen),
    - the top-level label is at least 2 characters of letters only.
  - **No catastrophic-backtracking regexes.** Validation must complete in
    well under 100 ms for inputs up to 1000 characters of pathological
    "almost-matching" content. (See the ReDoS articles linked in the
    research notes; nested `(a+)+`-style patterns are out.)

- `normalizePhone(s: string): string | null` — return the E.164 form for
  US numbers (`+1XXXXXXXXXX`) or `null` for anything that doesn't fit. Accept:
  - 10 digits with any combination of spaces, dots, dashes, or parens,
  - the same with a leading `1`, `1-`, or `+1` country code,
  - and only those. Reject foreign numbers, alphabetic input, lengths
    other than 10 / 11 (with `1` prefix), and the empty string.

Press `r` to run the hidden tests.
