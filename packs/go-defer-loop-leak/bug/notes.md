# Notes

- `ProcessFiles` is invoked once per batch.
- Each batch is on the order of 5000 files.
- `process()` itself reads the file, parses JSON, and inserts a row in
  Postgres. It does not retain `*os.File` beyond its return.
- The host has the default `RLIMIT_NOFILE = 1024`. Raising the limit
  is treated as a workaround, not a fix.
- Switching to streaming reads (e.g. bufio.Scanner) doesn't help —
  the report is `open: too many open files`, i.e. the failure is on
  `os.Open`, not on `Read`.
