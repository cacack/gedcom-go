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

// ErrorLine reports the physical line of the offending byte. Line-oriented
// readers above this layer (the parser) only know how far their own input
// reached, which lags the failure when a read is rejected mid-chunk; this
// method lets them adopt the accurate location instead.
func (e *ErrInvalidUTF8) ErrorLine() int {
	return e.Line
}

// The parser matches on this method, not on the concrete type, so pin it here.
var _ interface{ ErrorLine() int } = (*ErrInvalidUTF8)(nil)

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
	lastWasCR  bool // previous byte was CR, so a following LF completes one CRLF break
	bomSkipped bool
	buffer     []byte // Buffer for BOM bytes that need to be returned
	bufPos     int    // Current position in buffer
	pending    []byte // Incomplete UTF-8 sequence from previous read
	complete   []byte // Complete UTF-8 bytes ready to return
	err        error  // Sticky read failure, surfaced once complete has drained
}

func (u *utf8Reader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	// Return buffered BOM bytes first if any
	if n, ok := u.readBuffered(p); ok {
		return n, nil
	}

	// Return any complete bytes first
	if n, ok := u.drainComplete(p); ok {
		return n, nil
	}

	// A validation failure is sticky. Its valid prefix has been delivered by
	// now, so report it: the bytes that follow the offending one were never
	// validated, and resuming past them would turn a rejected chunk into a
	// silent gap in the middle of the caller's document.
	if u.err != nil {
		return 0, u.err
	}

	// Skip BOM on first read
	if !u.bomSkipped {
		if n, err := u.handleBOM(p); err != nil || n > 0 {
			return n, err
		}
	}

	n, err = u.readAndProcess(p)

	if err == io.EOF && len(u.pending) > 0 {
		if n > 0 || len(u.complete) > 0 {
			// Return valid bytes first; error will surface on next read
			// when pending bytes are re-evaluated
			return n, nil
		}
		return 0, &ErrInvalidUTF8{Line: u.line, Column: u.column}
	}

	return n, err
}

func (u *utf8Reader) drainComplete(p []byte) (int, bool) {
	if len(u.complete) == 0 {
		return 0, false
	}
	n := copy(p, u.complete)
	if n < len(u.complete) {
		u.complete = u.complete[n:]
	} else {
		u.complete = nil
	}
	return n, true
}

func (u *utf8Reader) readAndProcess(p []byte) (int, error) {
	bufSize := len(u.pending) + len(p)
	if bufSize < 8 {
		bufSize = 8
	}
	workBuf := make([]byte, bufSize)

	workN := copy(workBuf, u.pending)
	u.pending = nil

	readN, err := u.reader.Read(workBuf[workN:])
	workN += readN

	n, verr := u.processWorkBuffer(p, workBuf, workN)
	if verr != nil {
		// The valid prefix has already been handed over — into p, and into
		// u.complete when the work buffer outran p, which it can by up to
		// len(u.pending) bytes (bufSize above) or by the 8-byte floor. Hold
		// the failure until those bytes have drained, as the EOF path in Read
		// does, so delivering the prefix never costs the caller a byte. A
		// prefix of zero bytes leaves nothing in u.complete either, so there
		// is nothing to wait for.
		u.err = verr
		if n > 0 {
			return n, nil
		}
		return 0, verr
	}

	return n, err
}

func (u *utf8Reader) processWorkBuffer(p, workBuf []byte, workN int) (int, error) {
	if workN == 0 {
		return 0, nil
	}

	completeLen := findLastCompleteUTF8(workBuf[:workN])
	if completeLen < workN {
		u.pending = make([]byte, workN-completeLen)
		copy(u.pending, workBuf[completeLen:workN])
		workN = completeLen
	}

	if workN == 0 {
		return 0, nil
	}

	goodLen, verr := u.validateAndTrack(workBuf[:workN])
	if verr != nil {
		// Everything ahead of the offending byte is valid UTF-8 and is the
		// caller's data, so deliver it: dropping it would make a recovered
		// partial document stop up to a whole chunk short of the line the
		// error names. Nothing from the offending byte onward is deliverable,
		// including the incomplete sequence pended above — it sits past the
		// failure point and will never be completed.
		u.pending = nil
		workN = goodLen
	}

	n := copy(p, workBuf[:workN])
	if n < workN {
		u.complete = make([]byte, workN-n)
		copy(u.complete, workBuf[n:workN])
	}

	return n, verr
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

// validateAndTrack validates p as UTF-8 and advances the line/column cursor
// over it. It returns how many leading bytes are valid: len(p) on success, and
// the offset of the offending byte alongside the error on failure.
func (u *utf8Reader) validateAndTrack(p []byte) (int, error) {
	if !utf8.Valid(p) {
		return u.findInvalidUTF8(p)
	}
	u.updatePosition(p)
	return len(p), nil
}

func (u *utf8Reader) findInvalidUTF8(p []byte) (int, error) {
	for i := 0; i < len(p); {
		r, size := utf8.DecodeRune(p[i:])
		if r == utf8.RuneError && size == 1 {
			// u.line/u.column already track the offending byte: the loop
			// advanced them over every rune preceding it.
			return i, &ErrInvalidUTF8{Line: u.line, Column: u.column}
		}
		u.advance(p[i], size)
		i += size
	}
	return len(p), nil
}

func (u *utf8Reader) updatePosition(p []byte) {
	for i := 0; i < len(p); i++ {
		u.advance(p[i], 1)
	}
}

// advance moves the cursor over one rune of width bytes starting with b.
// Line breaks follow the same rules as the parser's line splitter
// (parser.lineScanner): LF, CRLF and a bare CR each end one line. Tracking
// lastWasCR across calls keeps a CRLF pair split over two reads from counting
// twice. Columns are counted in bytes.
func (u *utf8Reader) advance(b byte, width int) {
	wasCR := u.lastWasCR
	u.lastWasCR = b == '\r'

	switch b {
	case '\r':
		u.line++
		u.column = 1
	case '\n':
		if !wasCR {
			u.line++
		}
		u.column = 1
	default:
		u.column += width
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
			switch {
			case b&0xE0 == 0xC0:
				seqLen = 2
			case b&0xF0 == 0xE0:
				seqLen = 3
			case b&0xF8 == 0xF0:
				seqLen = 4
			default:
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
