package sliceutils

// Chunk splits items into consecutive sub-slices of length size. The
// final chunk may be shorter. If size <= 0, it returns nil.
func Chunk[T any](items []T, size int) [][]T {
	if size <= 0 {
		return nil
	}
	out := make([][]T, 0, (len(items)+size-1)/size)
	for i := 0; i < len(items); i += size {
		end := i + size
		if end > len(items) {
			end = len(items)
		}
		// Allocate a fresh slice per chunk so callers can't mutate
		// items via the chunk view.
		chunk := make([]T, end-i)
		copy(chunk, items[i:end])
		out = append(out, chunk)
	}
	return out
}

// Unique returns the first-seen unique elements of items. The empty
// case returns a non-nil zero-length slice so JSON marshals it as [].
func Unique[T comparable](items []T) []T {
	out := make([]T, 0, len(items))
	seen := make(map[T]struct{}, len(items))
	for _, v := range items {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// RemoveAtIndex returns a new slice equal to items with the element at
// idx removed. It does not mutate the caller's backing array. If idx
// is out of range, items is returned unchanged.
func RemoveAtIndex[T any](items []T, idx int) []T {
	if idx < 0 || idx >= len(items) {
		return items
	}
	out := make([]T, 0, len(items)-1)
	out = append(out, items[:idx]...)
	out = append(out, items[idx+1:]...)
	return out
}
