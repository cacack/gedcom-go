package clip

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestString(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		maxBytes int
		want     string
	}{
		{"shorter than max", "abc", 5, "abc"},
		{"exactly max", "abcde", 5, "abcde"},
		{"longer than max", "abcdef", 5, "abcde…"},
		{"cut never splits a character", "abcdé", 5, "abcd…"},
		{"cut before a multi-byte character", "ab日本", 4, "ab…"},
		{"zero max", "abc", 0, "…"},
		{"negative max", "abc", -1, "…"},
		{"empty", "", 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := String(tt.s, tt.maxBytes)
			if got != tt.want {
				t.Errorf("String(%q, %d) = %q, want %q", tt.s, tt.maxBytes, got, tt.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("String(%q, %d) = %q is not valid UTF-8", tt.s, tt.maxBytes, got)
			}
		})
	}
}

// TestStringLongInput checks a long input is cut to the bound plus the
// ellipsis, keeping its head.
func TestStringLongInput(t *testing.T) {
	s := strings.Repeat("x", 1<<20)
	got := String(s, 256)
	if want := strings.Repeat("x", 256) + Ellipsis; got != want {
		t.Errorf("String(1 MiB, 256) has length %d, want %d", len(got), len(want))
	}
}
