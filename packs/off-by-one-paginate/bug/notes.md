# Reported behavior

- Endpoint backed by `Paginate` panics with a slice-out-of-bounds error.
- Reproduces only when the request asks for the last page **and** total items
  is not a multiple of `pageSize` (e.g., 7 items, pageSize=5, page=2).
