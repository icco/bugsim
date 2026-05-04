# Negative numbers sort weirdly with Intl.Collator

A pricing dashboard sorts string-encoded prices using `Intl.Collator`'s
`numeric: true` mode so that `"10"` ends up after `"2"` instead of before
it (the classic alphanumeric-sort fix).

It works for positive prices. It does **not** work for negatives — and
nobody noticed until a bug report came in about the "Top losers" panel
showing the wrong leaderboard.

Read `bug/notes.md` and `bug/sortPrices.ts`, then pick the explanation
that best matches the observed behaviour.
