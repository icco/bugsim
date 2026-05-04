# Reported behaviour

- Page size is 3, sorted ascending by `createdAt`.
- A burst of 7 events all commit with `createdAt = 1700000000000`
  (single transaction → same wall-clock millisecond).
- Page 1 returns events `[A, B, C]` — the first three of the seven.
- Page 2's cursor is `C.createdAt = 1700000000000`.
- Page 2 returns the *next* batch of unrelated events. Events `D`, `E`,
  `F`, `G` are never delivered to the listener.

The DB column is the right type (an integer ms epoch); no rounding or
encoding step is involved. Page-2 SQL is roughly:

```sql
SELECT * FROM events
WHERE created_at > $cursor
ORDER BY created_at
LIMIT 3
```

Re-running with the same cursor reproduces the missed rows every time.
