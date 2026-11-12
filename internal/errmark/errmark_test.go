package errmark

import (
	"errors"
	"fmt"
	"testing"
)

func TestMark(t *testing.T) {
	sentinel := errors.New("sentinel")
	cause := errors.New("cause")
	err := Mark(sentinel, fmt.Errorf("wrapped: %w", cause))
	if !errors.Is(err, sentinel) {
		t.Errorf("errors.Is(%v, sentinel) = false", err)
	}
	if !errors.Is(err, cause) {
		t.Errorf("errors.Is(%v, cause) = false", err)
	}
	if got := errors.Unwrap(err); got == nil || got.Error() != "wrapped: cause" {
		t.Errorf("Unwrap() = %v, want the marked error", got)
	}
	if got := err.Error(); got != "wrapped: cause" {
		t.Errorf("Error() = %q, want %q", got, "wrapped: cause")
	}
}

func TestMarkNil(t *testing.T) {
	if err := Mark(errors.New("sentinel"), nil); err != nil {
		t.Errorf("Mark(_, nil) = %v, want nil", err)
	}
}
