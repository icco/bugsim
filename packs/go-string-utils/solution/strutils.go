package strutils

import (
	"unicode"
	"unicode/utf8"
)

// ReverseString reverses s by Unicode code point.
func ReverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// IsPalindrome reports whether s is a palindrome under rune-level
// comparison after lower-casing and stripping anything that isn't a
// Unicode letter or digit.
func IsPalindrome(s string) bool {
	cleaned := make([]rune, 0, utf8.RuneCountInString(s))
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cleaned = append(cleaned, unicode.ToLower(r))
		}
	}
	for i, j := 0, len(cleaned)-1; i < j; i, j = i+1, j-1 {
		if cleaned[i] != cleaned[j] {
			return false
		}
	}
	return true
}

// RuneCount returns the number of runes in s. Equivalent to
// utf8.RuneCountInString, but written explicitly so the reference
// implementation doesn't simply delegate.
func RuneCount(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}
