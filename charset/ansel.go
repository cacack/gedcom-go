// ANSEL decoder for GEDCOM character encoding support.
//
// ANSEL (ANSI Z39.47) is a legacy character encoding used in GEDCOM 5.5 files.
// This file implements an io.Reader that converts ANSEL-encoded bytes to UTF-8.
//
// IMPORTANT: ANSEL places combining diacritical marks BEFORE the base character,
// while Unicode places them AFTER. This decoder handles the reordering automatically.
// For example, ANSEL bytes [0xE2, 0x65] (acute + e) become UTF-8 "e\u0301" (e + combining acute).

package charset

import (
	"fmt"
	"io"
	"unicode/utf8"
)

// ErrInvalidANSEL is returned when an invalid ANSEL byte sequence is encountered.
type ErrInvalidANSEL struct {
	Line   int
	Column int
	Byte   byte
}

func (e *ErrInvalidANSEL) Error() string {
	return fmt.Sprintf("invalid ANSEL byte 0x%02X at line %d, column %d", e.Byte, e.Line, e.Column)
}

// anselReader implements io.Reader to convert ANSEL-encoded input to UTF-8.
type anselReader struct {
	reader  io.Reader
	pending []rune // Buffered combining diacriticals waiting for base character
	outBuf  []byte // UTF-8 output buffer
	outPos  int    // Current read position in output buffer
	line    int    // Current line number for error reporting
	column  int    // Current column number for error reporting
	eof     bool   // Whether underlying reader has reached EOF
	err     error  // Stored error from processing
}

// newAnselReader creates a new reader that converts ANSEL-encoded input to UTF-8.
// The returned reader handles:
//   - ASCII passthrough (0x00-0x7F)
//   - Extended Latin character mapping (0xA1-0xC8)
//   - Combining diacritical reordering (0xE0-0xFE placed after base character)
func newAnselReader(r io.Reader) io.Reader {
	return &anselReader{
		reader:  r,
		pending: make([]rune, 0, 4), // Pre-allocate for typical case of few combining marks
		outBuf:  make([]byte, 0, 256),
		line:    1,
		column:  1,
	}
}

// Read implements io.Reader. It reads ANSEL-encoded bytes from the underlying
// reader, converts them to UTF-8, and writes the result to p.
func (r *anselReader) Read(p []byte) (n int, err error) {
	// Return any buffered output first
	if n, done := r.returnBuffered(p); done {
		return n, nil
	}

	// Return stored error if any
	if r.err != nil {
		return 0, r.err
	}

	// Return EOF if we've finished
	if r.eof {
		return 0, io.EOF
	}

	// Reset output buffer for new data
	r.outBuf = r.outBuf[:0]
	r.outPos = 0

	// Read and process input
	n, err = r.readAndProcess(p)
	if n > 0 || err != nil {
		return n, err
	}

	// If we have no output but haven't hit EOF, try reading more
	if !r.eof {
		return r.Read(p)
	}

	return 0, io.EOF
}

// returnBuffered copies buffered output to p and returns (bytes copied, true) if there was data.
// Returns (0, false) if there's no buffered data to return.
func (r *anselReader) returnBuffered(p []byte) (int, bool) {
	if r.outPos >= len(r.outBuf) {
		return 0, false
	}
	n := copy(p, r.outBuf[r.outPos:])
	r.outPos += n
	if r.outPos >= len(r.outBuf) {
		r.outBuf = r.outBuf[:0]
		r.outPos = 0
	}
	return n, true
}

// readAndProcess reads from the underlying reader and processes bytes.
func (r *anselReader) readAndProcess(p []byte) (int, error) {
	inputBuf := make([]byte, 256)
	nRead, readErr := r.reader.Read(inputBuf)

	if readErr != nil && readErr != io.EOF {
		return 0, readErr
	}

	// Handle pure EOF with no data
	if nRead == 0 && readErr == io.EOF {
		return r.handleEOF(p)
	}

	// Process each input byte
	for i := 0; i < nRead; i++ {
		if err := r.processByte(inputBuf[i]); err != nil {
			r.err = err
			if n, done := r.returnBuffered(p); done {
				return n, nil
			}
			return 0, err
		}
	}

	// Mark EOF for next call if underlying reader is done
	if readErr == io.EOF {
		r.eof = true
		r.flushPendingCombining()
	}

	// Return processed output
	if n, done := r.returnBuffered(p); done {
		return n, nil
	}
	return 0, nil
}

// handleEOF processes end-of-file, flushing any pending combining marks.
func (r *anselReader) handleEOF(p []byte) (int, error) {
	r.eof = true
	r.flushPendingCombining()
	if n, done := r.returnBuffered(p); done {
		return n, nil
	}
	return 0, io.EOF
}

// processByte handles a single ANSEL input byte, converting it to UTF-8.
func (r *anselReader) processByte(b byte) error {
	// Check for combining diacritical (0xE0-0xFE)
	if IsCombiningDiacritical(b) {
		combining, ok := anselCombining[b]
		if !ok {
			// Undefined combining mark in range (0xFC, 0xFD)
			return &ErrInvalidANSEL{Line: r.line, Column: r.column, Byte: b}
		}
		r.pending = append(r.pending, combining)
		r.column++
		return nil
	}

	// Get the base character
	var baseRune rune

	if b < 0x80 {
		// ASCII passthrough (0x00-0x7F)
		baseRune = rune(b)
	} else if mapped, ok := anselToUnicode[b]; ok {
		// Extended Latin and LDS extensions (0xA1-0xCF range with defined mappings)
		baseRune = mapped
	} else {
		// Invalid ANSEL byte:
		// - 0x80-0xA0: C1 control characters and undefined (not used in ANSEL)
		// - 0xAF, 0xBB: Undefined in ANSEL extended Latin range
		// - 0xC9-0xCC, 0xD0-0xDF: Undefined in ANSEL
		// - 0xFF: Undefined in ANSEL
		return &ErrInvalidANSEL{Line: r.line, Column: r.column, Byte: b}
	}

	// Emit base character as UTF-8
	r.emitRune(baseRune)

	// Emit any pending combining marks (now AFTER the base character)
	for _, combining := range r.pending {
		r.emitRune(combining)
	}
	r.pending = r.pending[:0] // Clear the buffer

	// Update line/column tracking
	if b == '\n' {
		r.line++
		r.column = 1
	} else {
		r.column++
	}

	return nil
}

// emitRune appends a rune to the output buffer as UTF-8 bytes.
func (r *anselReader) emitRune(ru rune) {
	var buf [utf8.UTFMax]byte
	n := utf8.EncodeRune(buf[:], ru)
	r.outBuf = append(r.outBuf, buf[:n]...)
}

// flushPendingCombining emits any remaining combining marks as standalone characters.
// This handles the edge case where combining marks appear at the end of the stream
// without a following base character.
func (r *anselReader) flushPendingCombining() {
	for _, combining := range r.pending {
		r.emitRune(combining)
	}
	r.pending = r.pending[:0]
}
