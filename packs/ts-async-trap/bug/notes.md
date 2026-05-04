# Reported behaviour

- Background job logs `"importBatch ok"` for every batch, but the
  destination database is missing rows.
- Re-running individual uploads against the same payloads reproduces the
  failure (HTTP 500 from the upstream service) — i.e. the upstream calls
  *are* failing, the importer just isn't propagating it.
- Errors do **not** appear in the importer's stderr, only in the upstream
  service's logs.
