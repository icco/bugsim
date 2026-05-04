package sumpositive

import "testing"

func TestSumPositive(t *testing.T) {
	cases := []struct {
		name string
		in   []int
		want int
	}{
		{"empty", nil, 0},
		{"all positive", []int{1, 2, 3}, 6},
		{"mixed", []int{-1, 2, -3, 4}, 6},
		{"zeros", []int{0, 0, 0}, 0},
		{"all negative", []int{-2, -7}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SumPositive(tc.in); got != tc.want {
				t.Fatalf("SumPositive(%v) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}
