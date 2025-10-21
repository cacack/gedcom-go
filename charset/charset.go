// Package charset provides character encoding utilities for GEDCOM files.
//
// This package handles UTF-8 validation and Byte Order Mark (BOM) removal
// for GEDCOM file parsing. It ensures that GEDCOM data is properly encoded
// and provides detailed error reporting for encoding issues.
package charset

import (
	"bytes"
	"fmt"
	"io"
	"unicode/utf8"
)

// ErrInvalidUTF8 is returned when invalid UTF-8 sequences are encountered.
type ErrInvalidUTF8 struct {
	Line   int
	Column int
}

func (e *ErrInvalidUTF8) Error() string {
	return fmt.Sprintf("invalid UTF-8 sequence at line %d, column %d", e.Line, e.Column)
}

// NewReader wraps an io.Reader to provide UTF-8 validation.
// It removes BOM (Byte Order Mark) if present and validates UTF-8 encoding.
func NewReader(r io.Reader) io.Reader {
	return &utf8Reader{
		reader: r,
		line:   1,
		column: 1,
	}
}

type utf8Reader struct {
	reader     io.Reader
	line       int
	column     int
	bomSkipped bool
	buffer     []byte // Buffer for BOM bytes that need to be returned
	bufPos     int    // Current position in buffer
}

func (u *utf8Reader) Read(p []byte) (n int, err error) {
	// Return buffered BOM bytes first if any
	if n, ok := u.readBuffered(p); ok {
		return n, nil
	}

	// Skip BOM on first read
	if !u.bomSkipped {
		if n, err := u.handleBOM(p); err != nil || n > 0 {
			return n, err
		}
	}

	n, err = u.reader.Read(p)
	if n > 0 {
		if err := u.validateAndTrack(p[:n]); err != nil {
			return 0, err
		}
	}

	return n, err
}

func (u *utf8Reader) readBuffered(p []byte) (int, bool) {
	if len(u.buffer) > 0 && u.bufPos < len(u.buffer) {
		n := copy(p, u.buffer[u.bufPos:])
		u.bufPos += n
		if u.bufPos >= len(u.buffer) {
			u.buffer = nil
			u.bufPos = 0
		}
		return n, true
	}
	return 0, false
}

func (u *utf8Reader) handleBOM(p []byte) (int, error) {
	u.bomSkipped = true
	bom := make([]byte, 3)
	n, err := io.ReadFull(u.reader, bom)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return 0, err
	}

	// Check for UTF-8 BOM (0xEF 0xBB 0xBF)
	if n == 3 && bytes.Equal(bom, []byte{0xEF, 0xBB, 0xBF}) {
		return 0, nil // BOM found and skipped
	}

	// No BOM, buffer these bytes to return
	if n > 0 {
		u.buffer = bom[:n]
		u.bufPos = 0
		copied := copy(p, u.buffer)
		u.bufPos = copied
		if u.bufPos >= len(u.buffer) {
			u.buffer = nil
			u.bufPos = 0
		}
		return copied, nil
	}
	return 0, nil
}

func (u *utf8Reader) validateAndTrack(p []byte) error {
	if !utf8.Valid(p) {
		return u.findInvalidUTF8(p)
	}
	u.updatePosition(p)
	return nil
}

func (u *utf8Reader) findInvalidUTF8(p []byte) error {
	for i := 0; i < len(p); {
		r, size := utf8.DecodeRune(p[i:])
		if r == utf8.RuneError && size == 1 {
			return &ErrInvalidUTF8{Line: u.line, Column: u.column + i}
		}
		if p[i] == '\n' {
			u.line++
			u.column = 1
		} else {
			u.column += size
		}
		i += size
	}
	return nil
}

func (u *utf8Reader) updatePosition(p []byte) {
	for i := 0; i < len(p); i++ {
		if p[i] == '\n' {
			u.line++
			u.column = 1
		} else {
			u.column++
		}
	}
}

// ValidateString checks if a string is valid UTF-8.
func ValidateString(s string) bool {
	return utf8.ValidString(s)
}

// ValidateBytes checks if a byte slice is valid UTF-8.
func ValidateBytes(b []byte) bool {
	return utf8.Valid(b)
}
