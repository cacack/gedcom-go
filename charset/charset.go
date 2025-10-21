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
	if len(u.buffer) > 0 && u.bufPos < len(u.buffer) {
		n = copy(p, u.buffer[u.bufPos:])
		u.bufPos += n
		if u.bufPos >= len(u.buffer) {
			u.buffer = nil
			u.bufPos = 0
		}
		return n, nil
	}

	// Skip BOM on first read
	if !u.bomSkipped {
		u.bomSkipped = true
		bom := make([]byte, 3)
		n, err := io.ReadFull(u.reader, bom)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return 0, err
		}
		// Check for UTF-8 BOM (0xEF 0xBB 0xBF)
		if n == 3 && bytes.Equal(bom, []byte{0xEF, 0xBB, 0xBF}) {
			// BOM found and skipped, continue to read actual content
		} else {
			// No BOM, buffer these bytes to return
			if n > 0 {
				u.buffer = bom[:n]
				u.bufPos = 0
				// Return the buffered bytes
				copied := copy(p, u.buffer)
				u.bufPos = copied
				if u.bufPos >= len(u.buffer) {
					u.buffer = nil
					u.bufPos = 0
				}
				return copied, nil
			}
		}
	}

	n, err = u.reader.Read(p)
	if n > 0 {
		// Validate UTF-8
		if !utf8.Valid(p[:n]) {
			// Find the position of the invalid sequence
			for i := 0; i < n; {
				r, size := utf8.DecodeRune(p[i:])
				if r == utf8.RuneError && size == 1 {
					return 0, &ErrInvalidUTF8{Line: u.line, Column: u.column + i}
				}
				if p[i] == '\n' {
					u.line++
					u.column = 1
				} else {
					u.column += size
				}
				i += size
			}
		} else {
			// Update line and column tracking
			for i := 0; i < n; i++ {
				if p[i] == '\n' {
					u.line++
					u.column = 1
				} else {
					u.column++
				}
			}
		}
	}

	return n, err
}

// ValidateString checks if a string is valid UTF-8.
func ValidateString(s string) bool {
	return utf8.ValidString(s)
}

// ValidateBytes checks if a byte slice is valid UTF-8.
func ValidateBytes(b []byte) bool {
	return utf8.Valid(b)
}
