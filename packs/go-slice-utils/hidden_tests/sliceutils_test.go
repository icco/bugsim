package sliceutils

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestChunkEvenSplit(t *testing.T) {
	got := Chunk([]int{1, 2, 3, 4, 5, 6}, 2)
	want := [][]int{{1, 2}, {3, 4}, {5, 6}}
	if !equal2D(got, want) {
		t.Fatalf("Chunk = %v, want %v", got, want)
	}
}

func TestChunkRemainderTrailing(t *testing.T) {
	got := Chunk([]int{1, 2, 3, 4, 5}, 2)
	want := [][]int{{1, 2}, {3, 4}, {5}}
	if !equal2D(got, want) {
		t.Fatalf("Chunk = %v, want %v", got, want)
	}
}

func TestChunkSizeZeroReturnsNil(t *testing.T) {
	if got := Chunk([]int{1, 2, 3}, 0); got != nil {
		t.Fatalf("Chunk(_, 0) = %v, want nil", got)
	}
	if got := Chunk([]int{1, 2, 3}, -2); got != nil {
		t.Fatalf("Chunk(_, -2) = %v, want nil", got)
	}
}

func TestChunkLargerThanLen(t *testing.T) {
	got := Chunk([]int{1, 2, 3}, 10)
	want := [][]int{{1, 2, 3}}
	if !equal2D(got, want) {
		t.Fatalf("Chunk = %v, want %v", got, want)
	}
}

func TestUniquePreservesFirstSeenOrder(t *testing.T) {
	got := Unique([]int{3, 1, 2, 1, 3, 4, 2})
	want := []int{3, 1, 2, 4}
	if !slices.Equal(got, want) {
		t.Fatalf("Unique = %v, want %v", got, want)
	}
}

// TestUniqueEmptyMarshalsAsArray catches the classic
// `var out []T` mistake — a nil slice marshals to `null` in JSON,
// which breaks downstream API consumers expecting `[]`.
func TestUniqueEmptyMarshalsAsArray(t *testing.T) {
	got := Unique([]string{})
	if got == nil {
		t.Fatal("Unique on empty input must return a non-nil zero-length slice")
	}
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if string(b) != "[]" {
		t.Fatalf("Unique([]) marshals to %q, want %q", string(b), "[]")
	}
}

// TestRemoveAtIndexDoesNotMutateInput is the headline test: the
// classic `append(items[:idx], items[idx+1:]...)` one-liner mutates
// the backing array, breaking any caller that retains a reference.
func TestRemoveAtIndexDoesNotMutateInput(t *testing.T) {
	original := []int{10, 20, 30, 40, 50}
	originalCopy := slices.Clone(original)

	got := RemoveAtIndex(original, 2)
	want := []int{10, 20, 40, 50}

	if !slices.Equal(got, want) {
		t.Errorf("RemoveAtIndex = %v, want %v", got, want)
	}
	if !slices.Equal(original, originalCopy) {
		t.Errorf("RemoveAtIndex mutated input: got %v, want unchanged %v", original, originalCopy)
	}
}

// TestRemoveAtIndexBackingArrayIsolation demonstrates the bug at one
// remove of indirection — a separate slice header pointing at the
// same backing array sees the mutation, even when the caller "did the
// right thing" of reassigning the result.
func TestRemoveAtIndexBackingArrayIsolation(t *testing.T) {
	backing := []int{1, 2, 3, 4, 5, 6, 7, 8}
	view := backing[:6]
	viewCopy := slices.Clone(view)

	_ = RemoveAtIndex(view, 1)

	if !slices.Equal(view, viewCopy) {
		t.Fatalf("RemoveAtIndex mutated the caller's backing array via shared view: got %v, want %v", view, viewCopy)
	}
	if !slices.Equal(backing, []int{1, 2, 3, 4, 5, 6, 7, 8}) {
		t.Fatalf("RemoveAtIndex leaked a write into the wider backing array: %v", backing)
	}
}

func TestRemoveAtIndexOutOfRangeIsSafe(t *testing.T) {
	in := []int{1, 2, 3}
	if got := RemoveAtIndex(in, -1); !slices.Equal(got, in) {
		t.Errorf("RemoveAtIndex(_, -1) = %v, want unchanged %v", got, in)
	}
	if got := RemoveAtIndex(in, 99); !slices.Equal(got, in) {
		t.Errorf("RemoveAtIndex(_, 99) = %v, want unchanged %v", got, in)
	}
}

func TestRemoveAtIndexFirstAndLast(t *testing.T) {
	if got, want := RemoveAtIndex([]string{"a", "b", "c"}, 0), []string{"b", "c"}; !slices.Equal(got, want) {
		t.Errorf("RemoveAtIndex first = %v, want %v", got, want)
	}
	if got, want := RemoveAtIndex([]string{"a", "b", "c"}, 2), []string{"a", "b"}; !slices.Equal(got, want) {
		t.Errorf("RemoveAtIndex last = %v, want %v", got, want)
	}
}

func equal2D[T comparable](a, b [][]T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !slices.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}
