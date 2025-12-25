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

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// Encoding represents the character encoding of a GEDCOM file.
type Encoding int

const (
	// EncodingUnknown indicates no BOM was detected.
	EncodingUnknown Encoding = iota
	// EncodingUTF8 indicates UTF-8 encoding (BOM: 0xEF 0xBB 0xBF).
	EncodingUTF8
	// EncodingUTF16LE indicates UTF-16 Little Endian (BOM: 0xFF 0xFE).
	EncodingUTF16LE
	// EncodingUTF16BE indicates UTF-16 Big Endian (BOM: 0xFE 0xFF).
	EncodingUTF16BE
)

// ErrInvalidUTF8 is returned when invalid UTF-8 sequences are encountered.
type ErrInvalidUTF8 struct {
	Line   int
	Column int
}

func (e *ErrInvalidUTF8) Error() string {
	return fmt.Sprintf("invalid UTF-8 sequence at line %d, column %d", e.Line, e.Column)
}

// NewReader wraps an io.Reader to provide encoding detection and UTF-8 validation.
// It detects the encoding from the BOM (Byte Order Mark), converts non-UTF-8 encodings
// to UTF-8, and validates the resulting UTF-8 stream.
//
// Supported encodings:
//   - UTF-16 LE (BOM: 0xFF 0xFE) - Converted to UTF-8
//   - UTF-16 BE (BOM: 0xFE 0xFF) - Converted to UTF-8
//   - UTF-8 (BOM: 0xEF 0xBB 0xBF) - BOM removed, validated
//   - No BOM - Assumed UTF-8, validated
func NewReader(r io.Reader) io.Reader {
	// Detect encoding from BOM
	detectedReader, encoding, err := DetectBOM(r)
	if err != nil {
		// If BOM detection fails, fall back to UTF-8 validation
		return &utf8Reader{
			reader: r,
			line:   1,
			column: 1,
		}
	}

	// Handle UTF-16 encodings by converting to UTF-8
	var finalReader io.Reader
	switch encoding {
	case EncodingUTF16LE:
		// Convert UTF-16 LE to UTF-8
		utf16Reader := newUTF16Reader(detectedReader, false)
		finalReader = utf16Reader
	case EncodingUTF16BE:
		// Convert UTF-16 BE to UTF-8
		utf16Reader := newUTF16Reader(detectedReader, true)
		finalReader = utf16Reader
	case EncodingUTF8, EncodingUnknown:
		// Already UTF-8 or unknown (assume UTF-8)
		finalReader = detectedReader
	}

	// Wrap with UTF-8 validator
	return &utf8Reader{
		reader:     finalReader,
		line:       1,
		column:     1,
		bomSkipped: true, // BOM already handled by DetectBOM
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

// DetectBOM reads the first few bytes from r to detect a Byte Order Mark (BOM).
// It returns a new reader containing all the original data (with BOM consumed if present),
// the detected encoding, and any error encountered.
//
// BOM detection:
//   - UTF-16 LE: 0xFF 0xFE
//   - UTF-16 BE: 0xFE 0xFF
//   - UTF-8: 0xEF 0xBB 0xBF
//
// If no BOM is detected, the encoding is EncodingUnknown and all bytes are preserved.
func DetectBOM(r io.Reader) (io.Reader, Encoding, error) {
	// Read up to 3 bytes to check for BOM (max BOM length)
	// Use ReadFull to ensure we get complete bytes or handle short reads properly
	buf := make([]byte, 3)
	n, err := io.ReadFull(r, buf)

	// Handle read errors (but not EOF which is expected for small files)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, EncodingUnknown, err
	}

	// No data read
	if n == 0 {
		return bytes.NewReader(nil), EncodingUnknown, nil
	}

	var encoding Encoding
	var skipBytes int

	// Detect encoding from BOM
	// Use explicit slice comparisons to satisfy static analysis
	switch {
	case n >= 2 && bytes.Equal(buf[:2], []byte{0xFF, 0xFE}):
		// UTF-16 LE BOM
		encoding = EncodingUTF16LE
		skipBytes = 2
	case n >= 2 && bytes.Equal(buf[:2], []byte{0xFE, 0xFF}):
		// UTF-16 BE BOM
		encoding = EncodingUTF16BE
		skipBytes = 2
	case n >= 3 && bytes.Equal(buf[:3], []byte{0xEF, 0xBB, 0xBF}):
		// UTF-8 BOM
		encoding = EncodingUTF8
		skipBytes = 3
	default:
		// No BOM detected
		encoding = EncodingUnknown
		skipBytes = 0
	}

	// Create new reader with remaining bytes after BOM
	remaining := buf[skipBytes:n]
	newReader := io.MultiReader(bytes.NewReader(remaining), r)

	return newReader, encoding, nil
}

// newUTF16Reader creates a reader that converts UTF-16 encoded data to UTF-8.
// The bigEndian parameter specifies whether the input is UTF-16 BE (true) or LE (false).
// The returned reader automatically handles the conversion.
func newUTF16Reader(r io.Reader, bigEndian bool) io.Reader {
	var endian unicode.Endianness
	if bigEndian {
		endian = unicode.BigEndian
	} else {
		endian = unicode.LittleEndian
	}

	// Use IgnoreBOM since we've already consumed the BOM in DetectBOM
	decoder := unicode.UTF16(endian, unicode.IgnoreBOM).NewDecoder()
	return transform.NewReader(r, decoder)
}
