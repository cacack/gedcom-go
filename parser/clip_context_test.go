package parser

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// TestParseErrorClipsContext checks a ParseError does not echo a long line
// whole (#744): a 900 KB malformed line used to give a 900 KB error string.
func TestParseErrorClipsContext(t *testing.T) {
	long := strings.Repeat("x", 900_000)
	clipped := long[:MaxContextBytes] + "…"

	t.Run("invalid level", func(t *testing.T) {
		_, err := NewParser().ParseLine(long + " TAG")
		var pe *ParseError
		if !errors.As(err, &pe) {
			t.Fatalf("ParseLine() error = %v, want a *ParseError", err)
		}
		if pe.Context != clipped {
			t.Errorf("Context has length %d, want the first %d bytes and an ellipsis", len(pe.Context), MaxContextBytes)
		}
		if n := len(pe.Error()); n > 2*MaxContextBytes {
			t.Errorf("Error() is %d bytes, want it bounded by the clipped context", n)
		}
		// The underlying *strconv.NumError quotes the field too.
		var numErr *strconv.NumError
		if !errors.As(err, &numErr) || numErr.Num != clipped {
			t.Errorf("NumError = %v, want its Num clipped to %d bytes", numErr, MaxContextBytes)
		}
	})

	t.Run("xref with a space quoted in the message", func(t *testing.T) {
		_, err := NewParser().ParseLine("0 @I " + long + "@ INDI")
		var pe *ParseError
		if !errors.As(err, &pe) || !errors.Is(err, ErrXRefContainsSpace) {
			t.Fatalf("ParseLine() error = %v, want an ErrXRefContainsSpace *ParseError", err)
		}
		if len(pe.Message) > MaxContextBytes+len("…") || !strings.HasPrefix(pe.Message, "xref contains a space: @I x") {
			t.Errorf("Message = %.60q (length %d), want it clipped to %d bytes", pe.Message, len(pe.Message), MaxContextBytes)
		}
		if len(pe.Context) > MaxContextBytes+len("…") {
			t.Errorf("Context has length %d, want at most %d bytes and an ellipsis", len(pe.Context), MaxContextBytes)
		}
	})

	t.Run("short line kept whole", func(t *testing.T) {
		_, err := NewParser().ParseLine("abc TAG")
		var pe *ParseError
		if !errors.As(err, &pe) || pe.Context != "abc TAG" {
			t.Errorf("ParseLine() error = %v, want context %q", err, "abc TAG")
		}
	})
}

// keepFunc adapts a func to ErrorFilter for tests.
type keepFunc struct{ f func(*ParseError) bool }

func (k *keepFunc) KeepError(pe *ParseError) bool { return k.f(pe) }

// TestParseWithOptionsFilter checks the Filter sees every lenient error in
// input order, that only those it keeps are collected, and that the error
// stopping a TolerateOnly parse is collected without consulting it.
func TestParseWithOptionsFilter(t *testing.T) {
	input := "0 HEAD\nbad a\nbad b\nbad c\n0 TRLR\n"

	var seen []int
	lines, errs, fatal := NewParser().ParseWithOptions(strings.NewReader(input), &ParseOptions{
		Lenient: true,
		Filter: &keepFunc{func(pe *ParseError) bool {
			seen = append(seen, pe.Line)
			return pe.Line != 3
		}},
	})
	if fatal != nil {
		t.Fatalf("ParseWithOptions() fatal = %v", fatal)
	}
	if want := []int{2, 3, 4}; !reflect.DeepEqual(seen, want) {
		t.Errorf("Filter saw lines %v, want %v", seen, want)
	}
	var kept []int
	for _, pe := range errs {
		kept = append(kept, pe.Line)
	}
	if want := []int{2, 4}; !reflect.DeepEqual(kept, want) {
		t.Errorf("collected errors on lines %v, want %v", kept, want)
	}
	if len(lines) != 2 {
		t.Errorf("got %d lines, want HEAD and TRLR", len(lines))
	}

	called := false
	_, errs, _ = NewParser().ParseWithOptions(strings.NewReader(input), &ParseOptions{
		Lenient:      true,
		TolerateOnly: ErrXRefContainsSpace,
		Filter:       &keepFunc{func(*ParseError) bool { called = true; return false }},
	})
	if called || len(errs) != 1 || errs[0].Line != 2 {
		t.Errorf("TolerateOnly parse: Filter called = %v, errors = %v; want the stopping error on line 2, unconsulted", called, errs)
	}
}
