# Slice utilities (Go) — backing-array safety

Implement three exported generic helpers in `sliceutils.go`. The hidden
tests are derived from real Go bug reports about slice aliasing and
the append/backing-array trap (see e.g.
[golang/go#19982](https://github.com/golang/go/issues/19982) and the
[Go slices blog post](https://go.dev/blog/slices)).

```go
func Chunk[T any](items []T, size int) [][]T
func Unique[T comparable](items []T) []T
func RemoveAtIndex[T any](items []T, idx int) []T
```

- `Chunk` splits `items` into consecutive sub-slices of length `size`.
  The final chunk may be shorter. If `size <= 0`, return `nil`.

- `Unique` returns the elements of `items` in their first-seen order
  with duplicates removed. The empty/nil input must round-trip to a
  non-nil zero-length slice (`len == 0`, but not `nil`), because the
  hidden tests `json.Marshal` the result and assert `[]` rather than
  `null`.

- `RemoveAtIndex` returns a new slice equal to `items` with the element
  at position `idx` removed. **Critically, it must not mutate the
  caller's backing array.** The naive one-liner

  ```go
  return append(items[:idx], items[idx+1:]...)
  ```

  shifts elements in place and writes a stale value past the new tail
  of the slice header — which is fine if the caller immediately
  discards `items`, but corrupts the original if they hold any other
  slice pointing at the same array. The hidden tests deliberately keep
  references and check for mutation. Use a fresh allocation.

Out-of-range `idx` (including negative) must return `items` unchanged,
without panicking.
