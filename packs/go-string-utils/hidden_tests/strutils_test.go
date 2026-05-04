package strutils

import (
	"testing"
	"unicode/utf8"
)

func TestReverseStringASCII(t *testing.T) {
	if got := ReverseString("hello"); got != "olleh" {
		t.Fatalf("ReverseString(\"hello\") = %q, want %q", got, "olleh")
	}
}

func TestReverseStringEmpty(t *testing.T) {
	if got := ReverseString(""); got != "" {
		t.Fatalf("ReverseString(\"\") = %q, want \"\"", got)
	}
}

// TestReverseStringMultiByteUTF8 catches the classic
// `[]byte(s)` byte-reverse mistake. "héllo" is 6 bytes (h é=0xC3 0xA9 l l o)
// but only 5 runes, so a byte-level reverse produces invalid UTF-8.
func TestReverseStringMultiByteUTF8(t *testing.T) {
	in := "héllo"
	got := ReverseString(in)
	if got != "olléh" {
		t.Fatalf("ReverseString(%q) = %q, want %q", in, got, "olléh")
	}
	if !utf8.ValidString(got) {
		t.Fatalf("ReverseString(%q) returned invalid UTF-8: % x", in, []byte(got))
	}
}

func TestReverseStringPreservesSingleCodepointEmoji(t *testing.T) {
	// U+1F600 (😀) is a single code point but 4 bytes in UTF-8. A
	// byte-level reverse would split the surrogate-equivalent bytes.
	in := "a😀b"
	want := "b😀a"
	got := ReverseString(in)
	if got != want {
		t.Fatalf("ReverseString(%q) = %q, want %q", in, got, want)
	}
	if !utf8.ValidString(got) {
		t.Fatalf("output is invalid UTF-8: % x", []byte(got))
	}
}

func TestIsPalindromeASCII(t *testing.T) {
	if !IsPalindrome("racecar") {
		t.Fatal("racecar should be a palindrome")
	}
	if IsPalindrome("hello") {
		t.Fatal("hello should not be a palindrome")
	}
}

// TestIsPalindromeIgnoresPunctuationAndCase asserts the canonical
// "A man, a plan, a canal: Panama" example holds.
func TestIsPalindromeIgnoresPunctuationAndCase(t *testing.T) {
	cases := []string{
		"A man, a plan, a canal: Panama",
		"Was it a car or a cat I saw?",
		"No 'x' in Nixon",
	}
	for _, s := range cases {
		if !IsPalindrome(s) {
			t.Errorf("expected palindrome: %q", s)
		}
	}
}

// TestIsPalindromeUnicode checks rune-level reversal on real
// Hungarian palindromes. A naive byte-level check fails on these
// because `é`, `ö` etc. encode as two bytes (0xC3 0xA9, 0xC3 0xB6),
// so the symmetric continuation bytes don't line up: position 0 is
// the leading byte 0xC3 but position N-1 is the continuation byte
// 0xA9.
func TestIsPalindromeUnicode(t *testing.T) {
	cases := []string{
		"Géza, kék az ég",      // "Géza, the sky is blue"
		"Indul a görög aludni", // "the Greek goes to sleep"
		"Égé",                  // minimal 3-rune case
	}
	for _, s := range cases {
		if !IsPalindrome(s) {
			t.Errorf("expected palindrome under rune-level reversal: %q", s)
		}
	}
}

// TestRuneCountMatchesUTF8 exercises golang/go#22127 directly: len() is
// the byte count, RuneCount must agree with utf8.RuneCountInString.
func TestRuneCountMatchesUTF8(t *testing.T) {
	cases := []struct {
		s         string
		wantRunes int
		wantBytes int
	}{
		{"", 0, 0},
		{"hello", 5, 5},
		{"héllo", 5, 6},
		{"é um cãozinho", 13, 15},
		{"😀", 1, 4},
		{"a😀b", 3, 6},
	}
	for _, tc := range cases {
		t.Run(tc.s, func(t *testing.T) {
			if got := RuneCount(tc.s); got != tc.wantRunes {
				t.Errorf("RuneCount(%q) = %d, want %d (len = %d)", tc.s, got, tc.wantRunes, tc.wantBytes)
			}
			if got, want := len(tc.s), tc.wantBytes; got != want {
				// Sanity check our test data — Go's len() must agree with
				// our byte expectation, otherwise the test is wrong.
				t.Fatalf("test data error: len(%q) = %d, expected %d", tc.s, got, want)
			}
		})
	}
}
