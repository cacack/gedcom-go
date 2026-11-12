package charset

import (
	"errors"
	"fmt"
	"testing"
)

// TestInvalidByteErrorSentinels checks each invalid-byte error type matches its
// own sentinel through wrapping, and not the other's.
func TestInvalidByteErrorSentinels(t *testing.T) {
	utf8Err := fmt.Errorf("reading: %w", &InvalidUTF8Error{Line: 1, Column: 2, Byte: 0xE9})
	anselErr := fmt.Errorf("reading: %w", &InvalidANSELError{Line: 3, Column: 4, Byte: 0x80})

	if !errors.Is(utf8Err, ErrInvalidUTF8Sequence) {
		t.Errorf("errors.Is(%v, ErrInvalidUTF8Sequence) = false", utf8Err)
	}
	if errors.Is(utf8Err, ErrInvalidANSELByte) {
		t.Errorf("errors.Is(%v, ErrInvalidANSELByte) = true", utf8Err)
	}
	if !errors.Is(anselErr, ErrInvalidANSELByte) {
		t.Errorf("errors.Is(%v, ErrInvalidANSELByte) = false", anselErr)
	}
	if errors.Is(anselErr, ErrInvalidUTF8Sequence) {
		t.Errorf("errors.Is(%v, ErrInvalidUTF8Sequence) = true", anselErr)
	}
}

// TestDeprecatedErrorTypeAliases checks the former type names still name the
// same types, so errors.As with either finds the error.
func TestDeprecatedErrorTypeAliases(t *testing.T) {
	err := fmt.Errorf("reading: %w", &InvalidUTF8Error{Line: 1, Column: 2, Byte: 0xE9})
	var old *ErrInvalidUTF8 //nolint:staticcheck // the deprecated alias is under test
	if !errors.As(err, &old) || old.Byte != 0xE9 {
		t.Errorf("errors.As(%v, *ErrInvalidUTF8) = %v, want the InvalidUTF8Error", err, old)
	}

	err = fmt.Errorf("reading: %w", &InvalidANSELError{Line: 3, Column: 4, Byte: 0x80})
	var oldANSEL *ErrInvalidANSEL //nolint:staticcheck // the deprecated alias is under test
	if !errors.As(err, &oldANSEL) || oldANSEL.Byte != 0x80 {
		t.Errorf("errors.As(%v, *ErrInvalidANSEL) = %v, want the InvalidANSELError", err, oldANSEL)
	}
}
