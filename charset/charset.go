package charset

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
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
	// EncodingANSEL indicates ANSEL (ANSI Z39.47) encoding.
	EncodingANSEL
	// EncodingASCII indicates ASCII encoding.
	EncodingASCII
	// EncodingLATIN1 indicates ISO-8859-1 (Latin-1) encoding.
	EncodingLATIN1
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
// It first checks for a BOM (Byte Order Mark), then looks for a CHAR tag in the
// GEDCOM header to determine the encoding. The input is converted to UTF-8 and validated.
//
// Supported encodings:
//   - UTF-16 LE (BOM: 0xFF 0xFE) - Converted to UTF-8
//   - UTF-16 BE (BOM: 0xFE 0xFF) - Converted to UTF-8
//   - UTF-8 (BOM: 0xEF 0xBB 0xBF) - BOM removed, validated
//   - ANSEL (CHAR tag: ANSEL) - Converted to UTF-8, validated
//   - No BOM or CHAR tag - Assumed UTF-8, validated
func NewReader(r io.Reader) io.Reader {
	// First check for BOM (UTF-16, UTF-8 BOM)
	detectedReader, bomEnc, err := DetectBOM(r)
	if err != nil {
		// If BOM detection fails, fall back to UTF-8 validation
		return &utf8Reader{
			reader: r,
			line:   1,
			column: 1,
		}
	}

	// If BOM indicates UTF-16, use that encoding (takes precedence)
	if bomEnc == EncodingUTF16LE || bomEnc == EncodingUTF16BE {
		return NewReaderWithEncoding(detectedReader, bomEnc)
	}

	// Check header for CHAR declaration
	headerReader, headerEnc, err := DetectEncodingFromHeader(detectedReader)
	if err != nil {
		// If header detection fails, fall back to UTF-8 validation
		return &utf8Reader{
			reader: detectedReader,
			line:   1,
			column: 1,
		}
	}

	// Use detected encoding (or UTF-8 if unknown)
	return NewReaderWithEncoding(headerReader, headerEnc)
}

type utf8Reader struct {
	reader     io.Reader
	line       int
	column     int
	bomSkipped bool
	buffer     []byte // Buffer for BOM bytes that need to be returned
	bufPos     int    // Current position in buffer
	pending    []byte // Incomplete UTF-8 sequence from previous read
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

	pendingLen := len(u.pending)
	if pendingLen > 0 {
		copied := copy(p, u.pending)
		if copied < pendingLen {
			u.pending = u.pending[copied:]
			return copied, nil
		}
		u.pending = nil
		if copied == len(p) {
			return copied, nil
		}
	}

	n, err = u.reader.Read(p[pendingLen:])
	n += pendingLen

	if n > 0 {
		completeLen := findLastCompleteUTF8(p[:n])
		if completeLen < n {
			u.pending = make([]byte, n-completeLen)
			copy(u.pending, p[completeLen:n])
			n = completeLen
		}
		if n > 0 {
			if err := u.validateAndTrack(p[:n]); err != nil {
				return 0, err
			}
		}
	}

	if err == io.EOF && len(u.pending) > 0 {
		return 0, &ErrInvalidUTF8{Line: u.line, Column: u.column}
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

func findLastCompleteUTF8(p []byte) int {
	n := len(p)
	if n == 0 {
		return 0
	}

	for i := 1; i <= 3 && i <= n; i++ {
		b := p[n-i]

		if b&0x80 == 0 {
			return n
		} else if b&0xC0 == 0xC0 {
			var seqLen int
			if b&0xE0 == 0xC0 {
				seqLen = 2
			} else if b&0xF0 == 0xE0 {
				seqLen = 3
			} else if b&0xF8 == 0xF0 {
				seqLen = 4
			} else {
				return n
			}
			if i >= seqLen {
				return n
			}
			return n - i
		}
	}

	return n
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

// headerPeekSize is the number of bytes to read when looking for the CHAR tag.
// GEDCOM headers are typically small, so 1000 bytes should be sufficient.
const headerPeekSize = 1000

// charTagPattern matches the GEDCOM CHAR tag that declares the character encoding.
// Pattern handles both CR and LF line endings, and is case-insensitive.
// Matches: "1 CHAR ANSEL", "1 CHAR UTF-8", "1 CHAR ASCII", etc.
var charTagPattern = regexp.MustCompile(`(?i)[\r\n]1\s+CHAR\s+(\S+)`)

// DetectEncodingFromHeader peeks at GEDCOM header to find the CHAR tag.
// It returns a new reader with all bytes preserved, the detected encoding,
// and any error encountered.
//
// If the CHAR tag is not found within the first headerPeekSize bytes,
// EncodingUnknown is returned and the caller should assume UTF-8.
//
// Note: This function reads the entire remaining content to avoid issues with
// multi-byte UTF-8 sequences being split at arbitrary boundaries.
func DetectEncodingFromHeader(r io.Reader) (io.Reader, Encoding, error) {
	// Read all content to avoid splitting multi-byte UTF-8 sequences
	allContent, err := io.ReadAll(r)
	if err != nil {
		return nil, EncodingUnknown, err
	}

	// No data read
	if len(allContent) == 0 {
		return bytes.NewReader(nil), EncodingUnknown, nil
	}

	// Search for CHAR tag in the first headerPeekSize bytes (or less)
	searchLen := headerPeekSize
	if len(allContent) < searchLen {
		searchLen = len(allContent)
	}

	encoding := EncodingUnknown
	matches := charTagPattern.FindSubmatch(allContent[:searchLen])
	if len(matches) >= 2 {
		charValue := strings.ToUpper(string(matches[1]))
		switch charValue {
		case "ANSEL":
			encoding = EncodingANSEL
		case "ASCII":
			encoding = EncodingASCII
		case "UTF-8":
			encoding = EncodingUTF8
		case "UNICODE":
			// UNICODE typically means UTF-8 in GEDCOM context
			encoding = EncodingUTF8
		// UTF-16 is handled by BOM detection, but recognize the tag
		case "UTF-16", "UTF-16LE":
			encoding = EncodingUTF16LE
		case "UTF-16BE":
			encoding = EncodingUTF16BE
		case "LATIN1", "ISO-8859-1", "ANSI":
			encoding = EncodingLATIN1
		}
	}

	// Return reader with all content
	return bytes.NewReader(allContent), encoding, nil
}

// NewReaderWithEncoding wraps a reader with the specified encoding converter.
// It converts the input from the given encoding to UTF-8 and validates the result.
//
// Supported encodings:
//   - EncodingANSEL: ANSEL to UTF-8 conversion, then validation
//   - EncodingLATIN1: ISO-8859-1 to UTF-8 conversion, then validation
//   - EncodingUTF16LE: UTF-16 LE to UTF-8 conversion, then validation
//   - EncodingUTF16BE: UTF-16 BE to UTF-8 conversion, then validation
//   - EncodingUTF8, EncodingASCII, EncodingUnknown: UTF-8 validation only
func NewReaderWithEncoding(r io.Reader, enc Encoding) io.Reader {
	var convertedReader io.Reader

	switch enc {
	case EncodingANSEL:
		// ANSEL needs conversion to UTF-8
		convertedReader = newAnselReader(r)
	case EncodingLATIN1:
		// LATIN1 (ISO-8859-1) needs conversion to UTF-8
		decoder := charmap.ISO8859_1.NewDecoder()
		convertedReader = transform.NewReader(r, decoder)
	case EncodingUTF16LE:
		// UTF-16 LE needs conversion to UTF-8
		convertedReader = newUTF16Reader(r, false)
	case EncodingUTF16BE:
		// UTF-16 BE needs conversion to UTF-8
		convertedReader = newUTF16Reader(r, true)
	case EncodingUTF8, EncodingASCII, EncodingUnknown:
		// Already UTF-8 compatible, just validate
		convertedReader = r
	default:
		// Unknown encoding, assume UTF-8
		convertedReader = r
	}

	// Wrap with UTF-8 validator
	return &utf8Reader{
		reader:     convertedReader,
		line:       1,
		column:     1,
		bomSkipped: true, // Assume BOM already handled
	}
}
