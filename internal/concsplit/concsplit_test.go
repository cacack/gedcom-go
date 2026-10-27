package concsplit

import (
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
