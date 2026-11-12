package atsign

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

var versions = []gedcom.Version{gedcom.Version55, gedcom.Version551, gedcom.Version70, ""}

func TestEscape(t *testing.T) {
	tests := []struct {
		name string
		in   string
		v    gedcom.Version
		want string
	}{
		{"7.0 no at", "plain", gedcom.Version70, "plain"},
		{"7.0 leading", "@I1@", gedcom.Version70, "@@I1@"},
		{"7.0 interior kept", "a@b.com", gedcom.Version70, "a@b.com"},
		{"7.0 leading after LF", "x\n@y", gedcom.Version70, "x\n@@y"},
		{"7.0 leading after CRLF", "x\r\n@y", gedcom.Version70, "x\r\n@@y"},
		{"7.0 leading after CR", "x\r@y", gedcom.Version70, "x\r@@y"},
		{"unknown version is 7.0", "@x", "", "@@x"},
		{"5.5.1 every", "a@b @c@", gedcom.Version551, "a@@b @@c@@"},
		{"5.5 every", "@I1@", gedcom.Version55, "@@I1@@"},
		{"5.5.1 date escape kept", "@#DJULIAN@ 1700", gedcom.Version551, "@#DJULIAN@ 1700"},
		{"5.5.1 escape inside text kept", "on @#DJULIAN@ 1 JAN", gedcom.Version551, "on @#DJULIAN@ 1 JAN"},
		{"5.5.1 whole-value escape doubled", "@#X@", gedcom.Version551, "@@#X@@"},
		{"5.5.1 escape followed by at doubled", "@#X@@", gedcom.Version551, "@@#X@@@@"},
		{"5.5.1 empty escape text doubled", "@#@ x", gedcom.Version551, "@@#@@ x"},
		{"5.5.1 unterminated escape doubled", "@#DJULIAN", gedcom.Version551, "@@#DJULIAN"},
		{"5.5.1 escape across line break doubled", "@#D\nX@ y", gedcom.Version551, "@@#D\nX@@ y"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Escape(tt.in, tt.v); got != tt.want {
				t.Errorf("Escape(%q, %q) = %q, want %q", tt.in, tt.v, got, tt.want)
			}
		})
	}
}

func TestUnescape(t *testing.T) {
	tests := []struct {
		name string
		in   string
		v    gedcom.Version
		want string
	}{
		// gedcom7code 7/atsign.ged N01-N04 hold "@@start", "@@@@double" and
		// interior "@" cases; only a leading pair collapses in 7.0.
		{"7.0 leading", "@@start", gedcom.Version70, "@start"},
		{"7.0 leading quad", "@@@@double", gedcom.Version70, "@@@double"},
		{"7.0 interior kept", "cost @@ 5", gedcom.Version70, "cost @@ 5"},
		{"7.0 after LF", "x\n@@y", gedcom.Version70, "x\n@y"},
		{"7.0 lone at kept", "@x", gedcom.Version70, "@x"},
		{"5.5.1 every", "cost @@ 5 a@@b.com", gedcom.Version551, "cost @ 5 a@b.com"},
		{"5.5.1 lone at kept", "a@b.com", gedcom.Version551, "a@b.com"},
		{"5.5.1 escape kept", "@#DJULIAN@ 1700", gedcom.Version551, "@#DJULIAN@ 1700"},
		{"5.5 triple", "@@@", gedcom.Version55, "@@"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Unescape(tt.in, tt.v); got != tt.want {
				t.Errorf("Unescape(%q, %q) = %q, want %q", tt.in, tt.v, got, tt.want)
			}
		})
	}
}

func TestReescape(t *testing.T) {
	v55, v551, v70 := gedcom.Version55, gedcom.Version551, gedcom.Version70
	tests := []struct {
		name     string
		in       string
		from, to gedcom.Version
		want     string
	}{
		{"5.5.1 to 7.0 interior", "a@@b", v551, v70, "a@b"},
		{"5.5.1 to 7.0 leading", "@@lead", v551, v70, "@@lead"},
		{"5.5 to 7.0 leading after LF", "x\n@@y z@@w", v55, v70, "x\n@@y z@w"},
		{"5.5.1 to 7.0 lone at is text", "@x a@b", v551, v70, "@@x a@b"},
		{"5.5.1 to 7.0 calendar escape kept", "@#DROMAN@ 1700", v551, v70, "@#DROMAN@ 1700"},
		{"5.5.1 to 7.0 unterminated escape kept", "@#DJULIAN 1700", v551, v70, "@#DJULIAN 1700"},
		{"5.5.1 to 7.0 escaped escape is text", "@@#DX@@ 1", v551, v70, "@@#DX@ 1"},
		{"5.5.1 to 7.0 pair before escape", "@@@#DX@ 1", v551, v70, "@@@#DX@ 1"},
		{"7.0 to 5.5.1 interior", "a@b", v70, v551, "a@@b"},
		{"7.0 to 5.5 leading", "@@lead", v70, v55, "@@lead"},
		{"7.0 to 5.5.1 interior pair is two", "a@@b", v70, v551, "a@@@@b"},
		{"7.0 to 5.5.1 leading after LF", "x\n@@y", v70, v551, "x\n@@y"},
		{"5.5 to 5.5.1 unchanged", "a@@b", v55, v551, "a@@b"},
		{"7.0 to 7.0 unchanged", "a@b", v70, v70, "a@b"},
		{"no at", "plain", v551, v70, "plain"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Reescape(tt.in, tt.from, tt.to); got != tt.want {
				t.Errorf("Reescape(%q, %q, %q) = %q, want %q", tt.in, tt.from, tt.to, got, tt.want)
			}
		})
	}
}

// FuzzReescape asserts that Reescape carries text between versions: the value
// one version writes for s becomes the value the other writes. A text holding
// "@#" is exempt from 5.x to 7.0, where an escape sequence is kept as written.
func FuzzReescape(f *testing.F) {
	for _, seed := range []string{"", "@", "@@", "a@b", "@lead", "x\n@y", "@#DJULIAN@ 1700", "@@#X@ y"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		for _, from := range versions {
			for _, to := range versions {
				if doublesEvery(from) && !doublesEvery(to) && strings.Contains(s, "@#") {
					continue
				}
				if got, want := Reescape(Escape(s, from), from, to), Escape(s, to); got != want {
					t.Fatalf("Reescape(Escape(%q, %q), %q, %q) = %q, want %q", s, from, from, to, got, want)
				}
			}
		}
	})
}

// FuzzEscape asserts the two properties the encoder and decoder rely on, for
// every version: Unescape inverts Escape, and an escaped value is never a
// pointer, so typed text cannot re-decode as a link to a record (#708).
func FuzzEscape(f *testing.F) {
	for _, seed := range []string{
		"", "@", "@@", "@I1@", "@N 1@", "a@b.com", "@#DJULIAN@ 1700", "@#X@",
		"@#X@@", "x\n@y", "x\r\n@@y", "@@#X@", "@VOID@", "@#@ x",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		for _, v := range versions {
			escaped := Escape(s, v)
			if got := Unescape(escaped, v); got != s {
				t.Fatalf("Unescape(Escape(%q, %q)) = %q", s, v, got)
			}
			for _, line := range strings.FieldsFunc(escaped, func(r rune) bool { return r == '\n' || r == '\r' }) {
				for _, grammar := range versions {
					if gedcom.IsPointerXRefForVersion(line, grammar) {
						t.Fatalf("Escape(%q, %q) line %q is a pointer under %q", s, v, line, grammar)
					}
				}
			}
		}
	})
}
