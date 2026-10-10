package parser

import (
	"bufio"
	"errors"
	"strings"
	"testing"
)

// longLineInput is a GEDCOM 7 file whose shared note is one line of n bytes
// of payload. GEDCOM 7 has no line-length limit, so this is valid input.
func longLineInput(n int) string {
	return "0 HEAD\n1 GEDC\n2 VERS 7.0\n0 @N1@ SNOTE " +
		strings.Repeat("x", n) + "\n0 TRLR\n"
}

// TestLineLimit_AboveScannerDefault verifies that every line-reading path
// accepts a line longer than bufio.Scanner's 64 KiB default, up to
// MaxLineBytes (issue #578).
func TestLineLimit_AboveScannerDefault(t *testing.T) {
	const n = 70 << 10
	input := longLineInput(n)

	t.Run("FindRecord", func(t *testing.T) {
		p := NewLazyParser(strings.NewReader(input))
		if err := p.BuildIndex(); err != nil {
			t.Fatalf("BuildIndex() error = %v", err)
		}
		rec, err := p.FindRecord("@N1@")
		if err != nil {
			t.Fatalf("FindRecord() error = %v", err)
		}
		if got := len(rec.Lines[0].Value); got != n {
			t.Errorf("SNOTE value length = %d, want %d", got, n)
		}
	})

	t.Run("Parse", func(t *testing.T) {
		lines, err := NewParser().Parse(strings.NewReader(input))
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}
		if got := len(lines[3].Value); got != n {
			t.Errorf("SNOTE value length = %d, want %d", got, n)
		}
	})

	t.Run("ParseWithOptions", func(t *testing.T) {
		lines, _, err := NewParser().ParseWithOptions(strings.NewReader(input), &ParseOptions{})
		if err != nil {
			t.Fatalf("ParseWithOptions() error = %v", err)
		}
		if got := len(lines[3].Value); got != n {
			t.Errorf("SNOTE value length = %d, want %d", got, n)
		}
	})
}

// TestLineLimit_AboveMaxLineBytes verifies that a line longer than
// MaxLineBytes fails with an inspectable error, not a panic, on every
// line-reading path. BuildIndex reads through the offset-aware reader and
// wraps ErrLineTooLong; the lineScanner paths wrap bufio.ErrTooLong. Both
// sentinels are part of the documented API, so they are not unified here.
func TestLineLimit_AboveMaxLineBytes(t *testing.T) {
	input := longLineInput(MaxLineBytes + 1)

	t.Run("BuildIndex", func(t *testing.T) {
		err := NewLazyParser(strings.NewReader(input)).BuildIndex()
		if !errors.Is(err, ErrLineTooLong) {
			t.Fatalf("BuildIndex() error = %v, want ErrLineTooLong", err)
		}
	})

	t.Run("readRecordAt", func(t *testing.T) {
		// BuildIndex rejects this input, so read through a hand-built entry
		// to exercise the indexed-read scanner itself.
		p := NewLazyParser(strings.NewReader(input))
		_, err := p.readRecordAt(IndexEntry{ByteLength: int64(len(input))})
		if !errors.Is(err, bufio.ErrTooLong) {
			t.Fatalf("readRecordAt() error = %v, want bufio.ErrTooLong", err)
		}
	})

	t.Run("Parse", func(t *testing.T) {
		_, err := NewParser().Parse(strings.NewReader(input))
		if !errors.Is(err, bufio.ErrTooLong) {
			t.Fatalf("Parse() error = %v, want bufio.ErrTooLong", err)
		}
	})

	t.Run("ParseWithOptions", func(t *testing.T) {
		_, _, err := NewParser().ParseWithOptions(strings.NewReader(input), &ParseOptions{})
		if !errors.Is(err, bufio.ErrTooLong) {
			t.Fatalf("ParseWithOptions() error = %v, want bufio.ErrTooLong", err)
		}
	})
}

// boundaryInput is a file whose shared-note line holds exactly n content
// bytes, with every line ended by term.
func boundaryInput(n int, term string) string {
	const prefix = "0 @N1@ SNOTE "
	return "0 HEAD" + term + "1 GEDC" + term + "2 VERS 7.0" + term +
		prefix + strings.Repeat("x", n-len(prefix)) + term + "0 TRLR" + term
}

// TestLineLimit_ExactBoundary verifies that BuildIndex and the scanner paths
// agree on MaxLineBytes as a content limit for every line ending, so an
// indexed record is always readable (issue #578).
func TestLineLimit_ExactBoundary(t *testing.T) {
	terms := map[string]string{"LF": "\n", "CR": "\r", "CRLF": "\r\n"}
	for name, term := range terms {
		t.Run(name+"/at limit", func(t *testing.T) {
			input := boundaryInput(MaxLineBytes, term)
			p := NewLazyParser(strings.NewReader(input))
			if err := p.BuildIndex(); err != nil {
				t.Fatalf("BuildIndex() error = %v", err)
			}
			if _, err := p.FindRecord("@N1@"); err != nil {
				t.Fatalf("FindRecord() error = %v", err)
			}
			if _, err := NewParser().Parse(strings.NewReader(input)); err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
		})
		t.Run(name+"/over limit", func(t *testing.T) {
			input := boundaryInput(MaxLineBytes+1, term)
			err := NewLazyParser(strings.NewReader(input)).BuildIndex()
			if !errors.Is(err, ErrLineTooLong) {
				t.Fatalf("BuildIndex() error = %v, want ErrLineTooLong", err)
			}
			_, err = NewParser().Parse(strings.NewReader(input))
			if !errors.Is(err, bufio.ErrTooLong) {
				t.Fatalf("Parse() error = %v, want bufio.ErrTooLong", err)
			}
		})
	}

	t.Run("unterminated final line", func(t *testing.T) {
		const prefix = "0 @N1@ SNOTE "
		at := "0 HEAD\n" + prefix + strings.Repeat("x", MaxLineBytes-len(prefix))
		if _, err := NewParser().Parse(strings.NewReader(at)); err != nil {
			t.Fatalf("Parse() at limit error = %v", err)
		}
		_, err := NewParser().Parse(strings.NewReader(at + "x"))
		if !errors.Is(err, bufio.ErrTooLong) {
			t.Fatalf("Parse() over limit error = %v, want bufio.ErrTooLong", err)
		}
	})
}
