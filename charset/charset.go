package charset

import (
	"bytes"
	"errors"
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
	// EncodingLATIN1 indicates a single-byte Latin encoding, declared as
	// ANSI, LATIN1 or ISO-8859-1. It is decoded as Windows-1252, which matches
	// ISO-8859-1 on every printable byte and is what ANSI files contain.
	// WINDOWS-1252, CP1252 and WIN-1252 declare it too.
	EncodingLATIN1
	// EncodingCP437 indicates IBM code page 437 (the original IBM PC set),
	// declared as IBMPC, IBM PC, CP437 or IBM-437.
	EncodingCP437
	// EncodingCP850 indicates IBM code page 850 (DOS Latin-1), declared as
	// CP850 or IBM-850.
	EncodingCP850
	// EncodingMacRoman indicates Mac OS Roman, declared as MACINTOSH, MAC or
	// MACROMAN.
	EncodingMacRoman
	// EncodingWindows1250 indicates Windows-1250 (Central European). No CHAR
	// value declares it: Central European producers write CHAR ANSI over
	// Windows-1250 bytes, so a caller names it with [ReaderOptions.ANSI], or
	// passes it to [NewReaderWithEncoding].
	EncodingWindows1250
)

// ErrInvalidUTF8Sequence is matched, with errors.Is, by every
// *InvalidUTF8Error. Use errors.As with *InvalidUTF8Error to recover the
// offending byte and its position.
//
// It is distinct from encoder.ErrInvalidUTF8, which the encoder returns for a
// document value that is not valid UTF-8; neither matches the other.
var ErrInvalidUTF8Sequence = errors.New("invalid UTF-8 sequence")

// ErrInvalidUTF8 is the former name of InvalidUTF8Error.
//
// Deprecated: Use InvalidUTF8Error with errors.As, or ErrInvalidUTF8Sequence
// with errors.Is. This alias will be removed in v4.0.0.
type ErrInvalidUTF8 = InvalidUTF8Error

// InvalidUTF8Error is returned when invalid UTF-8 sequences are encountered.
// It is also what [ReaderOptions.OnInvalidUTF8] receives for each byte it
// replaces. It matches ErrInvalidUTF8Sequence with errors.Is.
type InvalidUTF8Error struct {
	Line   int
	Column int
	// Byte is the offending byte: the first byte that is not part of a valid
	// UTF-8 sequence.
	Byte byte
}

func (e *InvalidUTF8Error) Error() string {
	return fmt.Sprintf("invalid UTF-8 sequence at line %d, column %d", e.Line, e.Column)
}

// Is reports whether target is ErrInvalidUTF8Sequence.
func (e *InvalidUTF8Error) Is(target error) bool {
	return target == ErrInvalidUTF8Sequence
}

// ErrorLine reports the physical line of the offending byte. Line-oriented
// readers above this layer (the parser) only know how far their own input
// reached, which lags the failure when a read is rejected mid-chunk; this
// method lets them adopt the accurate location instead.
func (e *InvalidUTF8Error) ErrorLine() int {
	return e.Line
}

// The parser matches on this method, not on the concrete type, so pin it here.
var _ interface{ ErrorLine() int } = (*InvalidUTF8Error)(nil)

// NewReader wraps an io.Reader to provide encoding detection and UTF-8 validation.
// It first checks for a BOM (Byte Order Mark), then looks for a CHAR tag in the
// GEDCOM header to determine the encoding. The input is converted to UTF-8 and validated.
//
// Detection reads only the header before returning (at most 64 KiB; see
// DetectEncodingFromHeader) and replays those bytes, so the rest of the input
// is streamed as the returned reader is read. When the encoding is already
// known, NewReaderWithEncoding skips detection entirely.
//
// A read error during detection is not swallowed: the returned reader delivers
// the bytes read before it and then returns the error.
//
// Supported encodings:
//   - UTF-16 LE (BOM: 0xFF 0xFE) - Converted to UTF-8
//   - UTF-16 BE (BOM: 0xFE 0xFF) - Converted to UTF-8
//   - UTF-16 without a BOM, recognised by a first line starting "0 " in
//     UTF-16 (0x30 0x00 0x20 0x00 or 0x00 0x30 0x00 0x20) - Converted to UTF-8
//   - UTF-8 (BOM: 0xEF 0xBB 0xBF) - BOM removed, validated
//   - Any other encoding named by the header's CHAR tag (see
//     DetectEncodingFromHeader for the recognised values) - Converted to
//     UTF-8, validated. CHAR ANSI is read as Windows-1252;
//     [NewReaderWithOptions] can name its code page ([ReaderOptions.ANSI])
//   - No BOM or recognised CHAR tag - Assumed UTF-8, validated
func NewReader(r io.Reader) io.Reader {
	return NewReaderWithOptions(r, ReaderOptions{})
}

// ReaderOptions configures [NewReaderWithOptions]. The zero value gives the
// behaviour of [NewReader].
type ReaderOptions struct {
	// OnInvalidUTF8, when non-nil, makes the reader recover from invalid
	// UTF-8 instead of failing: each byte that is not part of a valid UTF-8
	// sequence is replaced with U+FFFD (the Unicode replacement character)
	// and reported to OnInvalidUTF8 with its line, column and value, once
	// per byte, and reading continues. When nil, the first such byte fails
	// the read with *InvalidUTF8Error.
	//
	// This applies to the UTF-8 validation every input passes through, so in
	// practice to input read as UTF-8: declared UTF-8 or ASCII, or undeclared.
	// It does not change ANSEL decoding, which reports an invalid ANSEL byte
	// with *InvalidANSELError either way.
	OnInvalidUTF8 func(*InvalidUTF8Error)

	// ANSI is the encoding a header declaring CHAR ANSI is decoded as. ANSI
	// names no code page of its own: it is whatever Windows code page the
	// producing system used, Windows-1252 in Western Europe and the Americas
	// but Windows-1250 in Central Europe. EncodingUnknown, the zero value,
	// decodes it as Windows-1252 (EncodingLATIN1). Any other value is used
	// as [NewReaderWithEncoding] would use it, typically EncodingLATIN1 or
	// EncodingWindows1250. It does not affect any other CHAR value, nor
	// LATIN1, ISO-8859-1 or WINDOWS-1252, which name their code page.
	ANSI Encoding

	// OnAmbiguousANSI, when non-nil and ANSI is EncodingUnknown, is called
	// for the first byte of a CHAR ANSI input whose Windows-1252 reading
	// differs from its Windows-1250 one, such as 0xE8 ("è" or "č"), and
	// at most once per input. Decoding is unchanged: the byte is still read
	// as Windows-1252. The report says that the file's letters may be wrong
	// and that ANSI can name the code page. It is not called when ANSI is
	// set, or for input whose bytes read the same in both code pages.
	OnAmbiguousANSI func(*AmbiguousANSI)
}

// AmbiguousANSI describes the first byte of a CHAR ANSI input that reads
// differently in Windows-1252 and Windows-1250; see
// [ReaderOptions.OnAmbiguousANSI].
type AmbiguousANSI struct {
	// Line and Column locate the byte, counting lines as [InvalidUTF8Error]
	// does and columns in bytes.
	Line   int
	Column int
	// Byte is the byte itself.
	Byte byte
	// Windows1252 is the byte's Windows-1252 reading, which the reader
	// delivers; Windows1250 is its Windows-1250 reading.
	Windows1252 rune
	Windows1250 rune
}

// NewReaderWithOptions is [NewReader] configured by opts.
func NewReaderWithOptions(r io.Reader, opts ReaderOptions) io.Reader {
	// First check for BOM (UTF-16, UTF-8 BOM)
	detectedReader, bomEnc, err := DetectBOM(r)
	if err != nil {
		// If BOM detection fails, fall back to UTF-8 validation of the
		// replaying reader, which delivers the bytes already read and then
		// reports the read error in place.
		return newUTF8Reader(detectedReader, opts, false)
	}

	// Without a BOM, UTF-16 shows itself in the first line's "0 ": the
	// header's CHAR tag cannot be read until the byte layout is known.
	if bomEnc == EncodingUnknown {
		detectedReader, bomEnc, err = sniffUTF16(detectedReader)
		if err != nil {
			return newUTF8Reader(detectedReader, opts, false)
		}
	}

	// If BOM indicates UTF-16, use that encoding (takes precedence)
	if bomEnc == EncodingUTF16LE || bomEnc == EncodingUTF16BE {
		return newReaderWithEncoding(detectedReader, bomEnc, opts)
	}

	// Check header for CHAR declaration
	probe, headerLen, err := probeHeader(detectedReader)
	if err != nil {
		// If header detection fails, fall back to UTF-8 validation of the
		// bytes already probed, then report the read error in place so the
		// consumed prefix is neither lost nor silently skipped.
		return newUTF8Reader(io.MultiReader(bytes.NewReader(probe), errReader{err: err}), opts, false)
	}

	// Use detected encoding (or UTF-8 if unknown)
	headerReader := io.MultiReader(bytes.NewReader(probe), detectedReader)
	value := charValue(probe[:headerLen])
	enc := charValues[value]
	if value == "ANSI" {
		switch {
		case opts.ANSI != EncodingUnknown:
			enc = opts.ANSI
		case opts.OnAmbiguousANSI != nil:
			headerReader = &ansiScanner{reader: headerReader, cursor: newCursor(), report: opts.OnAmbiguousANSI}
		}
	}
	return newReaderWithEncoding(headerReader, enc, opts)
}

// ansiScanner passes CHAR ANSI input through unchanged, reporting the first
// byte whose Windows-1252 and Windows-1250 readings differ. Once it has
// reported, it stops looking.
type ansiScanner struct {
	reader io.Reader
	cursor
	report func(*AmbiguousANSI) // nil once reported
}

func (s *ansiScanner) Read(p []byte) (int, error) {
	n, err := s.reader.Read(p)
	if s.report == nil {
		return n, err
	}
	for _, b := range p[:n] {
		if windows1252Table[b] != windows1250Table[b] {
			s.report(&AmbiguousANSI{
				Line:        s.line,
				Column:      s.column,
				Byte:        b,
				Windows1252: windows1252Table[b],
				Windows1250: windows1250Table[b],
			})
			s.report = nil
			break
		}
		s.advance(b, 1)
	}
	return n, err
}

// newUTF8Reader returns a utf8Reader over r. bomSkipped reports whether a
// UTF-8 BOM has already been handled, so the reader need not look for one.
func newUTF8Reader(r io.Reader, opts ReaderOptions, bomSkipped bool) *utf8Reader {
	return &utf8Reader{
		reader:     r,
		cursor:     newCursor(),
		bomSkipped: bomSkipped,
		onInvalid:  opts.OnInvalidUTF8,
	}
}

// readPrefix reads from r until buf is full or a read returns an error. Unlike
// io.ReadFull, it keeps an error returned alongside the bytes that fill buf, so
// a failure on that read is never dropped. A clean end of input is io.EOF.
func readPrefix(r io.Reader, buf []byte) (int, error) {
	n := 0
	for n < len(buf) {
		m, err := r.Read(buf[n:])
		n += m
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

// errReader is an io.Reader that always returns err.
type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }

// cursor tracks a 1-based line and byte column across reads.
type cursor struct {
	line      int
	column    int
	lastWasCR bool // previous byte was CR, so a following LF completes one CRLF break
}

// newCursor returns a cursor at line 1, column 1.
func newCursor() cursor { return cursor{line: 1, column: 1} }

type utf8Reader struct {
	reader io.Reader
	cursor
	bomSkipped bool
	pending    []byte                  // Incomplete UTF-8 sequence from previous read
	complete   []byte                  // Complete UTF-8 bytes ready to return
	err        error                   // Sticky failure, surfaced once complete has drained
	onInvalid  func(*InvalidUTF8Error) // When non-nil, replace invalid bytes and report them here
}

func (u *utf8Reader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
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
		u.handleBOM()
	}

	n, err = u.readAndProcess(p)

	if err == io.EOF && len(u.pending) > 0 {
		if u.onInvalid != nil {
			// The input ended inside a sequence: it can never complete, so
			// replace its bytes like any other invalid ones. End of input
			// is reported on the next read, once they have drained.
			return u.flushPending(p, n), nil
		}
		if n > 0 || len(u.complete) > 0 {
			// Return valid bytes first; error will surface on next read
			// when pending bytes are re-evaluated
			return n, nil
		}
		return 0, &InvalidUTF8Error{Line: u.line, Column: u.column, Byte: u.pending[0]}
	}

	return n, err
}

// flushPending delivers the incomplete sequence held at end of input, with its
// bytes replaced, after the n bytes already written to p. It returns the new
// total written to p; anything that does not fit joins u.complete.
func (u *utf8Reader) flushPending(p []byte, n int) int {
	out := u.replaceInvalid(u.pending)
	u.pending = nil
	m := copy(p[n:], out)
	u.complete = append(u.complete, out[m:]...)
	return n + m
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

	if u.onInvalid != nil {
		out := u.replaceInvalid(workBuf[:workN])
		n := copy(p, out)
		if n < len(out) {
			u.complete = append([]byte(nil), out[n:]...)
		}
		return n, nil
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

// handleBOM strips a leading UTF-8 BOM. Any other bytes read while looking for
// one are replayed ahead of the rest of the source, so readAndProcess validates
// and tracks them like any other input. A read error (or io.EOF) is replayed in
// place of the source, so the failure is reported after those bytes rather
// than resumed past.
func (u *utf8Reader) handleBOM() {
	u.bomSkipped = true
	bom := make([]byte, 3)
	n, err := readPrefix(u.reader, bom)
	if n == 3 && bytes.Equal(bom, []byte{0xEF, 0xBB, 0xBF}) {
		n = 0 // BOM found and skipped
	}
	rest := u.reader
	if err != nil {
		rest = errReader{err: err}
	}
	u.reader = io.MultiReader(bytes.NewReader(bom[:n]), rest)
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
			return i, &InvalidUTF8Error{Line: u.line, Column: u.column, Byte: p[i]}
		}
		u.advance(p[i], size)
		i += size
	}
	return len(p), nil
}

// replaceInvalid advances the cursor over p and returns it with each byte that
// is not part of a valid UTF-8 sequence replaced by U+FFFD, reporting each one
// to u.onInvalid. It returns p itself when p is valid.
func (u *utf8Reader) replaceInvalid(p []byte) []byte {
	if utf8.Valid(p) {
		u.updatePosition(p)
		return p
	}
	out := make([]byte, 0, len(p)+2*utf8.UTFMax)
	for i := 0; i < len(p); {
		r, size := utf8.DecodeRune(p[i:])
		if r == utf8.RuneError && size == 1 {
			u.onInvalid(&InvalidUTF8Error{Line: u.line, Column: u.column, Byte: p[i]})
			out = utf8.AppendRune(out, utf8.RuneError)
		} else {
			out = append(out, p[i:i+size]...)
		}
		// The cursor counts input bytes, so a replaced byte advances it by
		// one column whatever its replacement's width.
		u.advance(p[i], size)
		i += size
	}
	return out
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
func (c *cursor) advance(b byte, width int) {
	wasCR := c.lastWasCR
	c.lastWasCR = b == '\r'

	switch b {
	case '\r':
		c.line++
		c.column = 1
	case '\n':
		if !wasCR {
			c.line++
		}
		c.column = 1
	default:
		c.column += width
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
//
// On a read error other than io.EOF, DetectBOM returns EncodingUnknown and the
// error, along with a non-nil reader that replays the bytes read before the
// error and then returns that error, so no consumed byte is lost.
func DetectBOM(r io.Reader) (io.Reader, Encoding, error) {
	// Read up to 3 bytes to check for BOM (max BOM length)
	buf := make([]byte, 3)
	n, err := readPrefix(r, buf)

	// Handle read errors (but not EOF which is expected for small files)
	if err != nil && err != io.EOF {
		return io.MultiReader(bytes.NewReader(buf[:n]), errReader{err: err}), EncodingUnknown, err
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

// sniffUTF16 recognises UTF-16 without a BOM by its first four bytes, which in
// a GEDCOM file are the level "0" and the space after it: 0x30 0x00 0x20 0x00
// in little-endian and 0x00 0x30 0x00 0x20 in big-endian. Neither can open a
// GEDCOM file in any 8-bit encoding. It returns a reader replaying every byte
// of r, and EncodingUnknown when the bytes do not match.
//
// On a read error other than io.EOF, the returned reader replays the bytes
// read before it and then returns the error, as DetectBOM's does.
func sniffUTF16(r io.Reader) (io.Reader, Encoding, error) {
	buf := make([]byte, 4)
	n, err := readPrefix(r, buf)
	if err != nil && err != io.EOF {
		return io.MultiReader(bytes.NewReader(buf[:n]), errReader{err: err}), EncodingUnknown, err
	}
	replay := io.MultiReader(bytes.NewReader(buf[:n]), r)
	switch {
	case n == 4 && bytes.Equal(buf, []byte{'0', 0x00, ' ', 0x00}):
		return replay, EncodingUTF16LE, nil
	case n == 4 && bytes.Equal(buf, []byte{0x00, '0', 0x00, ' '}):
		return replay, EncodingUTF16BE, nil
	}
	return replay, EncodingUnknown, nil
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

// headerProbeLimit caps how many bytes DetectEncodingFromHeader reads while
// looking for the end of the GEDCOM header. Real headers are a few KiB at most;
// the cap keeps an input with no record after HEAD from being read whole.
const headerProbeLimit = 64 << 10

// headerProbeChunk is the size of each read DetectEncodingFromHeader makes
// while probing, so a short header costs one small read rather than the cap.
const headerProbeChunk = 4 << 10

// charTagPattern matches the GEDCOM CHAR tag that declares the character encoding
// and captures its whole value, up to the end of the line, so a value with a
// space in it ("IBM PC") is read whole rather than as its first word.
// Pattern handles both CR and LF line endings, and is case-insensitive.
// Matches: "1 CHAR ANSEL", "1 CHAR UTF-8", "1 CHAR IBM PC", etc.
var charTagPattern = regexp.MustCompile(`(?i)[\r\n]1[ \t]+CHAR[ \t]+([^\r\n]+)`)

// charValues maps each recognised CHAR value, upper-cased and with its blanks
// collapsed to single spaces, to the encoding it declares.
var charValues = map[string]Encoding{
	"ANSEL": EncodingANSEL,
	"ASCII": EncodingASCII,
	"UTF-8": EncodingUTF8,
	"UTF8":  EncodingUTF8,
	// UNICODE in a header readable byte by byte is UTF-8; UTF-16 without a
	// BOM is recognised before the header is read (see sniffUTF16).
	"UNICODE": EncodingUTF8,
	// UTF-16 input is recognised before the header is read (DetectBOM,
	// sniffUTF16); a header readable byte by byte that says UTF-16 maps to LE
	"UTF-16":   EncodingUTF16LE,
	"UTF-16LE": EncodingUTF16LE,
	"UTF-16BE": EncodingUTF16BE,
	// Windows-1252, which also serves ISO-8859-1 (#564)
	"ANSI":         EncodingLATIN1,
	"LATIN1":       EncodingLATIN1,
	"ISO-8859-1":   EncodingLATIN1,
	"WINDOWS-1252": EncodingLATIN1,
	"CP1252":       EncodingLATIN1,
	"WIN-1252":     EncodingLATIN1,
	// DOS and Mac code pages (#735)
	"IBMPC":     EncodingCP437,
	"IBM PC":    EncodingCP437,
	"CP437":     EncodingCP437,
	"IBM-437":   EncodingCP437,
	"CP850":     EncodingCP850,
	"IBM-850":   EncodingCP850,
	"MACINTOSH": EncodingMacRoman,
	"MAC":       EncodingMacRoman,
	"MACROMAN":  EncodingMacRoman,
}

// DetectEncodingFromHeader peeks at the GEDCOM header to find the CHAR tag.
// It returns a reader that replays every byte of r, the detected encoding,
// and any error encountered while probing.
//
// Only the header is read: probing stops at the first level-0 line after
// HEAD, at end of input, or after 64 KiB, whichever comes first. The CHAR tag
// is searched for within that header; if it is not found, or its value is not
// recognised, EncodingUnknown is returned and the caller should assume UTF-8.
//
// Recognised values, matched whole and without regard to case: ANSEL; ASCII;
// UTF-8, UTF8 and UNICODE (UTF-8); UTF-16, UTF-16LE and UTF-16BE; ANSI,
// LATIN1, ISO-8859-1, WINDOWS-1252, CP1252 and WIN-1252 (EncodingLATIN1,
// decoded as Windows-1252); IBMPC, IBM PC, CP437 and IBM-437 (EncodingCP437);
// CP850 and IBM-850 (EncodingCP850); MACINTOSH, MAC and MACROMAN
// (EncodingMacRoman). A bare UTF-16 value carries no byte order and is
// reported as EncodingUTF16LE; the BOM, when present, decides instead.
//
// The rest of r is not read until the returned reader is, so a read error
// past the probe surfaces from it.
func DetectEncodingFromHeader(r io.Reader) (io.Reader, Encoding, error) {
	probe, headerLen, err := probeHeader(r)
	if err != nil {
		return nil, EncodingUnknown, err
	}

	// Replay the probed bytes ahead of the unread remainder
	return io.MultiReader(bytes.NewReader(probe), r), charEncoding(probe[:headerLen]), nil
}

// charEncoding returns the encoding declared by the first CHAR tag in header,
// or EncodingUnknown if there is none or its value is not recognized.
func charEncoding(header []byte) Encoding {
	return charValues[charValue(header)] // a miss is EncodingUnknown
}

// charValue returns the value of the first CHAR tag in header, upper-cased and
// with its blanks collapsed as charValues keys are, or "" if there is none.
func charValue(header []byte) string {
	matches := charTagPattern.FindSubmatch(header)
	if len(matches) < 2 {
		return ""
	}
	return strings.Join(strings.Fields(strings.ToUpper(string(matches[1]))), " ")
}

// probeHeader reads r until the header is complete (see
// DetectEncodingFromHeader) and returns the bytes read along with the length
// of the header within them. On a read error it returns the bytes read before
// the error, so a caller can still replay them.
func probeHeader(r io.Reader) (probe []byte, headerLen int, err error) {
	chunk := make([]byte, headerProbeChunk)
	var scan headerScanner
	for len(probe) < headerProbeLimit {
		n, readErr := r.Read(chunk[:min(len(chunk), headerProbeLimit-len(probe))])
		probe = append(probe, chunk[:n]...)
		if readErr != nil && readErr != io.EOF {
			return probe, 0, readErr
		}
		if end := scan.headerEnd(probe); end >= 0 {
			return probe, end, nil
		}
		if readErr == io.EOF {
			break
		}
	}
	return probe, len(probe), nil
}

// headerScanState is where a headerScanner is within the current line.
type headerScanState int

const (
	scanLeadingBlank headerScanState = iota // before the HEAD line
	scanInLine                              // past the start of a line that is not level 0
	scanLineStart                           // after a line break, in optional leading blanks
	scanLevelZero                           // after a line break, blanks and a '0'
)

// headerScanner finds the end of the header in a probe that grows across
// reads. It keeps its place between calls and examines only the bytes appended
// since the last one, so a probe filled one byte per Read costs linear work.
type headerScanner struct {
	pos       int             // offset of the next byte to examine
	state     headerScanState // position within the current line
	lineBreak int             // offset of the line break that began the current line
}

// headerEnd returns the offset of the line break that begins the first level-0
// line after the HEAD line, or -1 if data does not contain one yet. data must
// extend the data given to earlier calls. Leading blank lines are skipped so
// the HEAD line itself is never mistaken for the end of the header.
func (s *headerScanner) headerEnd(data []byte) int {
	for ; s.pos < len(data); s.pos++ {
		if s.step(data[s.pos]) {
			return s.lineBreak
		}
	}
	return -1
}

// step advances the scan past c, which must be the byte at s.pos (s.pos is
// recorded as lineBreak), and reports whether c completes the start of a
// level-0 line. Case order matters: line breaks are handled first, and the
// blank cases rely on the earlier state-specific cases having been ruled out.
func (s *headerScanner) step(c byte) bool {
	blank := c == ' ' || c == '\t'
	switch {
	case c == '\r' || c == '\n':
		if s.state != scanLeadingBlank {
			s.state, s.lineBreak = scanLineStart, s.pos
		}
	case s.state == scanLeadingBlank:
		if !blank {
			s.state = scanInLine
		}
	case s.state == scanLevelZero && blank:
		return true
	case s.state == scanLineStart && c == '0':
		s.state = scanLevelZero
	case s.state == scanLineStart && blank:
		// Blanks before the level number: stay in scanLineStart.
	default:
		s.state = scanInLine
	}
	return false
}

// NewReaderWithEncoding wraps a reader with the specified encoding converter.
// It converts the input from the given encoding to UTF-8 and validates the result.
//
// Supported encodings:
//   - EncodingANSEL: ANSEL to UTF-8 conversion, then validation
//   - EncodingLATIN1: Windows-1252 to UTF-8 conversion, then validation; the
//     five bytes Windows-1252 leaves undefined keep their ISO-8859-1 code points
//   - EncodingWindows1250: Windows-1250 to UTF-8 conversion, then validation;
//     the bytes Windows-1250 leaves undefined keep their ISO-8859-1 code points
//   - EncodingCP437, EncodingCP850, EncodingMacRoman: code page to UTF-8
//     conversion, then validation; every byte has a character, so none is lost
//   - EncodingUTF16LE: UTF-16 LE to UTF-8 conversion, then validation
//   - EncodingUTF16BE: UTF-16 BE to UTF-8 conversion, then validation
//   - EncodingUTF8, EncodingASCII, EncodingUnknown: UTF-8 validation only
func NewReaderWithEncoding(r io.Reader, enc Encoding) io.Reader {
	return newReaderWithEncoding(r, enc, ReaderOptions{})
}

// newReaderWithEncoding is NewReaderWithEncoding configured by opts.
func newReaderWithEncoding(r io.Reader, enc Encoding, opts ReaderOptions) io.Reader {
	var convertedReader io.Reader

	switch enc {
	case EncodingANSEL:
		// ANSEL needs conversion to UTF-8
		convertedReader = newAnselReader(r)
	case EncodingLATIN1:
		// ANSI, LATIN1 and ISO-8859-1 files are decoded as Windows-1252 (#564)
		convertedReader = transform.NewReader(r, windows1252Decoder())
	case EncodingWindows1250:
		convertedReader = transform.NewReader(r, windows1250Decoder())
	case EncodingCP437:
		convertedReader = transform.NewReader(r, charmap.CodePage437.NewDecoder())
	case EncodingCP850:
		convertedReader = transform.NewReader(r, charmap.CodePage850.NewDecoder())
	case EncodingMacRoman:
		convertedReader = transform.NewReader(r, charmap.Macintosh.NewDecoder())
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

	// Wrap with UTF-8 validator; the BOM has already been handled
	return newUTF8Reader(convertedReader, opts, true)
}
