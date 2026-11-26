package concsplit

import (
	"slices"
	"strings"
	"testing"
)

// TestSplitPoint tests where a long line is cut for CONC: between two
// non-space characters and never inside a UTF-8 character (issue #565).
func TestSplitPoint(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		maxLen   int
		expected int
	}{
		{
			name:     "line shorter than max",
			line:     "Short",
			maxLen:   10,
			expected: 5,
		},
		{
			name:     "cut between two letters at max",
			line:     "Hello world this is a test",
			maxLen:   15,
			expected: 15, // "Hello world thi" | "s is a test"
		},
		{
			name:     "no space found - split at max",
			line:     "Supercalifragilisticexpialidocious",
			maxLen:   10,
			expected: 10,
		},
		{
			name:     "space just before max moves the cut back past it",
			line:     "12345678 x",
			maxLen:   9,
			expected: 7, // "1234567" | "8 x"
		},
		{
			name:     "space at max moves the cut back past it",
			line:     "one two three four",
			maxLen:   14,
			expected: 12, // "one two thre" | "e four"
		},
		{
			name:     "never inside a multi-byte character",
			line:     "ab日本語",
			maxLen:   4, // byte 4 is inside 日 (bytes 2-4)
			expected: 2,
		},
		{
			name:     "run of spaces longer than max falls back to max",
			line:     "a      b",
			maxLen:   4,
			expected: 4,
		},
		{
			name:     "never inside an @@ pair",
			line:     "ab@@cd",
			maxLen:   3, // "ab@" | "@cd" would leave a lone "@" on each line
			expected: 2,
		},
		{
			name:     "between two @@ pairs",
			line:     "a@@@@b",
			maxLen:   4, // the run starts at 1, so offset 3 pairs off and 4 does not
			expected: 3,
		},
		{
			name:     "first character longer than max is kept whole",
			line:     "日本",
			maxLen:   2,
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitPoint(tt.line, tt.maxLen)
			if result != tt.expected {
				t.Errorf("splitPoint() = %d, want %d", result, tt.expected)
			}
		})
	}
}

// TestSplit checks that the pieces rebuild the line, fit maxLen and never
// start or end with a space where a cut was made.
func TestSplit(t *testing.T) {
	if got := Split("short", 10); len(got) != 1 || got[0] != "short" {
		t.Errorf("Split(short) = %q, want one unchanged piece", got)
	}
	// A non-positive limit acts as 1 rather than looping forever.
	for _, maxLen := range []int{0, -5} {
		if got := Split("", maxLen); len(got) != 1 || got[0] != "" {
			t.Errorf("Split(\"\", %d) = %q, want one empty piece", maxLen, got)
		}
		if got := Split("abc", maxLen); strings.Join(got, "|") != "a|b|c" {
			t.Errorf("Split(abc, %d) = %q, want a|b|c", maxLen, got)
		}
	}
	line := strings.Repeat("word ", 60)
	pieces := Split(line, 20)
	if strings.Join(pieces, "") != line {
		t.Fatalf("pieces do not rebuild the line")
	}
	for i, p := range pieces {
		if len(p) > 20 {
			t.Errorf("piece %d is %d bytes, want at most 20", i, len(p))
		}
		if i < len(pieces)-1 && (strings.HasSuffix(p, " ") || strings.HasPrefix(pieces[i+1], " ")) {
			t.Errorf("cut after piece %d is beside a space: %q | %q", i, p, pieces[i+1])
		}
	}
}

// TestFold checks the CONT and CONC lines a value is written as, with and
// without a length limit and an escape.
func TestFold(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		maxLen int
		escape func(string) string
		want   []Segment
	}{
		{"empty", "", 10, nil, []Segment{{"", ""}}},
		{"one short line", "abc", 10, nil, []Segment{{"", "abc"}}},
		{"line breaks of every form", "a\nb\r\nc\rd", 10, nil,
			[]Segment{{"", "a"}, {"CONT", "b"}, {"CONT", "c"}, {"CONT", "d"}}},
		{"long lines cut with CONC", "abcdef\nghij", 4, nil,
			[]Segment{{"", "abcd"}, {"CONC", "ef"}, {"CONT", "ghij"}}},
		{"no limit cuts no CONC", "abcdef", 0, nil, []Segment{{"", "abcdef"}}},
		{"escape before the cut", "ab\ncd", 3, func(s string) string { return s + s },
			[]Segment{{"", "aba"}, {"CONC", "b"}, {"CONT", "cdc"}, {"CONC", "d"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Fold(tt.value, tt.maxLen, tt.escape)
			if !slices.Equal(got, tt.want) {
				t.Errorf("Fold(%q, %d) = %q, want %q", tt.value, tt.maxLen, got, tt.want)
			}
		})
	}
}

// TestValueLength checks the whole-line bound: a value fills what 255 bytes
// leave after the longer of its own line's prefix and its CONC line's, capped
// at limit and never below 1 (#967).
func TestValueLength(t *testing.T) {
	long := "_" + strings.Repeat("T", 300)
	tests := []struct {
		name      string
		limit     int
		level     int
		xref, tag string
		want      int
	}{
		{"short line keeps the limit", MaxValueLength, 1, "", "NOTE", 248},
		{"record xref lowers it", MaxValueLength, 0, "@N1@", "NOTE", 243},
		{"long tag lowers it", MaxValueLength, 1, "", "_" + strings.Repeat("T", 60), 255 - 64},
		{"limit above the room is cut to the room", 1000, 0, "@N1@", "NOTE", 243},
		{"limit below the room stays", 100, 0, "@N1@", "NOTE", 100},
		{"deep CONC line bounds it", MaxValueLength, 99, "", "N", 255 - len("100 CONC ")},
		{"prefix filling the line leaves 1", MaxValueLength, 1, "", long, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValueLength(tt.limit, tt.level, tt.xref, tt.tag, tt.level+1); got != tt.want {
				t.Errorf("ValueLength = %d, want %d", got, tt.want)
			}
		})
	}
}
