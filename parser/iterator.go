package parser

import (
	"bufio"
	"errors"
	"io"
	"iter"
)

// ErrLineTooLong is returned by [RecordIteratorWithOffset] / [RecordsWithOffset]
// when a single line exceeds [MaxLineBytes]. The streaming [RecordIterator] /
// [Records] surface the same condition via [bufio.ErrTooLong].
var ErrLineTooLong = errors.New("gedcom-go/parser: line exceeds MaxLineBytes")

// MaxLineBytes is the maximum length of a single GEDCOM line accepted by the
// streaming iterators. The GEDCOM 5.5.1 spec recommends a 255-byte limit;
// real-world files routinely exceed it (CONC/CONT chains, embedded BLOB
// data), so we use a generous 1 MiB ceiling. A line longer than this aborts
// the iterator with an error rather than allocating unboundedly — preventing
// hostile or corrupt input from exhausting memory.
const MaxLineBytes = 1 << 20 // 1 MiB

// recordLinesInitialCap is a hint for the initial capacity of RawRecord.Lines.
// Typical INDI records have 10-30 subordinate lines; this avoids 3-4 slice
// reallocations per record without over-allocating for simple ones.
const recordLinesInitialCap = 16

// RawRecord represents a complete GEDCOM record with all its subordinate lines.
// A record starts at level 0 and includes all following lines until the next level 0.
type RawRecord struct {
	// XRef is the optional cross-reference identifier (e.g., "@I1@")
	XRef string

	// Type is the tag at level 0 (e.g., "INDI", "FAM", "HEAD", "TRLR")
	Type string

	// Lines contains all parsed lines belonging to this record, including the level-0 line
	Lines []*Line

	// ByteOffset is the starting byte position of this record in the file
	ByteOffset int64

	// ByteLength is the total number of bytes for this record
	ByteLength int64
}

// RecordIterator provides streaming access to GEDCOM records.
// It groups lines into records (level-0 boundaries) without loading the entire file into memory.
type RecordIterator struct {
	scanner    *bufio.Scanner
	parser     *Parser
	current    *RawRecord
	pending    *Line // Buffered line that belongs to next record
	pendingLen int   // Byte length of pending line
	err        error
	byteOffset int64
	lineEnding int // Track typical line ending size (1 for LF/CR, 2 for CRLF)
}

// NewRecordIterator creates a new RecordIterator that reads from the given reader.
// The reader should already be wrapped with charset.NewReader() for encoding normalization.
//
// Lines longer than [MaxLineBytes] cause iteration to abort with an error
// rather than allocating unboundedly. Spec-compliant GEDCOM lines never
// approach this limit.
func NewRecordIterator(r io.Reader) *RecordIterator {
	scanner := bufio.NewScanner(r)
	scanner.Split(scanGEDCOMLines)
	// Explicit buffer with documented ceiling; default bufio.Scanner cap is
	// 64 KiB which can be too small for files containing embedded BLOBs.
	scanner.Buffer(make([]byte, 0, 4096), MaxLineBytes)

	return &RecordIterator{
		scanner:    scanner,
		parser:     NewParser(),
		lineEnding: 1, // Conservative default
	}
}

// Next advances the iterator to the next record.
// Returns true if a record is available, false when iteration is complete or on error.
func (it *RecordIterator) Next() bool {
	if it.err != nil {
		return false
	}

	record := &RawRecord{
		ByteOffset: it.byteOffset,
		Lines:      make([]*Line, 0, recordLinesInitialCap),
	}

	// Use pending line from previous iteration if available
	if it.pending != nil {
		record.XRef = it.pending.XRef
		record.Type = it.pending.Tag
		record.Lines = append(record.Lines, it.pending)
		record.ByteOffset = it.byteOffset - int64(it.pendingLen)
		it.pending = nil
		it.pendingLen = 0
	} else if !it.scanNextLine(record) {
		// Read first line of record
		return false
	}

	// Read subordinate lines until next level-0 tag or EOF
	for it.scanner.Scan() {
		text := it.scanner.Text()
		lineLen := len(it.scanner.Bytes()) + it.lineEnding

		line, err := it.parser.ParseLine(text)
		if err != nil {
			it.err = err
			return false
		}

		if line.Level == 0 {
			// This line belongs to next record - buffer it
			it.pending = line
			it.pendingLen = lineLen
			break
		}

		record.Lines = append(record.Lines, line)
		it.byteOffset += int64(lineLen)
	}

	if err := it.scanner.Err(); err != nil {
		it.err = err
		return false
	}

	// Calculate byte length
	record.ByteLength = it.byteOffset - record.ByteOffset

	// Empty record means we've reached EOF without any lines
	if len(record.Lines) == 0 {
		return false
	}

	it.current = record
	return true
}

// scanNextLine reads and parses the first line of a new record.
func (it *RecordIterator) scanNextLine(record *RawRecord) bool {
	if !it.scanner.Scan() {
		if err := it.scanner.Err(); err != nil {
			it.err = err
		}
		return false
	}

	text := it.scanner.Text()
	lineLen := len(it.scanner.Bytes()) + it.lineEnding

	line, err := it.parser.ParseLine(text)
	if err != nil {
		it.err = err
		return false
	}

	record.XRef = line.XRef
	record.Type = line.Tag
	record.Lines = append(record.Lines, line)
	it.byteOffset += int64(lineLen)

	return true
}

// Record returns the current record.
// Returns nil if Next() has not been called or returned false.
func (it *RecordIterator) Record() *RawRecord {
	return it.current
}

// Err returns any error encountered during iteration.
// Should be checked after Next() returns false.
func (it *RecordIterator) Err() error {
	return it.err
}

// RecordIteratorWithOffset creates a RecordIterator that tracks accurate byte offsets.
// This is used when building an index and needs precise offset tracking.
type RecordIteratorWithOffset struct {
	reader  *bufio.Reader
	parser  *Parser
	current *RawRecord
	pending *lineWithPos
	err     error
	bytePos int64 // Current byte position
}

// lineWithPos holds a parsed line with its byte position.
type lineWithPos struct {
	line    *Line
	pos     int64
	byteLen int64
}

// NewRecordIteratorWithOffset creates an iterator with accurate byte offset tracking.
func NewRecordIteratorWithOffset(r io.Reader) *RecordIteratorWithOffset {
	return &RecordIteratorWithOffset{
		reader: bufio.NewReader(r),
		parser: NewParser(),
	}
}

// readLine reads a single line with its byte position and length.
func (it *RecordIteratorWithOffset) readLine() (*lineWithPos, error) {
	startPos := it.bytePos

	// Read until line terminator
	lineBytes, err := readGEDCOMLine(it.reader)
	if err != nil {
		return nil, err
	}

	byteLen := int64(len(lineBytes))
	it.bytePos += byteLen

	// Parse the line (strip line endings for parsing)
	text := string(trimLineEnding(lineBytes))
	line, err := it.parser.ParseLine(text)
	if err != nil {
		return nil, err
	}

	return &lineWithPos{
		line:    line,
		pos:     startPos,
		byteLen: byteLen,
	}, nil
}

// trimLineEnding removes CR, LF, or CRLF from the end of a byte slice.
func trimLineEnding(b []byte) []byte {
	n := len(b)
	if n > 0 && b[n-1] == '\n' {
		n--
		if n > 0 && b[n-1] == '\r' {
			n--
		}
	} else if n > 0 && b[n-1] == '\r' {
		n--
	}
	return b[:n]
}

// readGEDCOMLine reads bytes until a line terminator (CR, LF, or CRLF).
// Returns the line including the terminator(s). Aborts with ErrLineTooLong
// if the line exceeds [MaxLineBytes] before a terminator is reached.
func readGEDCOMLine(r *bufio.Reader) ([]byte, error) {
	// Pre-size to cover typical GEDCOM lines (255-byte spec recommendation)
	// without intermediate reallocations.
	line := make([]byte, 0, 256)

	for {
		b, err := r.ReadByte()
		if err != nil {
			if err == io.EOF && len(line) > 0 {
				return line, nil
			}
			return nil, err
		}

		line = append(line, b)
		if len(line) > MaxLineBytes {
			return nil, ErrLineTooLong
		}

		if b == '\n' {
			return line, nil
		}
		if b == '\r' {
			// Check for CRLF
			next, err := r.Peek(1)
			if err == nil && len(next) > 0 && next[0] == '\n' {
				lf, _ := r.ReadByte()
				line = append(line, lf)
			}
			return line, nil
		}
	}
}

// Next advances the iterator to the next record.
func (it *RecordIteratorWithOffset) Next() bool {
	if it.err != nil {
		return false
	}

	record := &RawRecord{
		Lines: make([]*Line, 0, recordLinesInitialCap),
	}

	// Use pending line from previous iteration
	if it.pending != nil {
		record.XRef = it.pending.line.XRef
		record.Type = it.pending.line.Tag
		record.Lines = append(record.Lines, it.pending.line)
		record.ByteOffset = it.pending.pos
		it.pending = nil
	} else {
		// Read first line
		lp, err := it.readLine()
		if err != nil {
			if err != io.EOF {
				it.err = err
			}
			return false
		}

		record.XRef = lp.line.XRef
		record.Type = lp.line.Tag
		record.Lines = append(record.Lines, lp.line)
		record.ByteOffset = lp.pos
	}

	// Read subordinate lines
	for {
		lp, err := it.readLine()
		if err != nil {
			if err != io.EOF {
				it.err = err
				return false
			}
			break // EOF - finish current record
		}

		if lp.line.Level == 0 {
			// This line belongs to next record
			it.pending = lp
			break
		}

		record.Lines = append(record.Lines, lp.line)
	}

	if len(record.Lines) == 0 {
		return false
	}

	// Calculate byte length
	if it.pending != nil {
		record.ByteLength = it.pending.pos - record.ByteOffset
	} else {
		record.ByteLength = it.bytePos - record.ByteOffset
	}

	it.current = record
	return true
}

// Record returns the current record.
func (it *RecordIteratorWithOffset) Record() *RawRecord {
	return it.current
}

// Err returns any error encountered during iteration.
func (it *RecordIteratorWithOffset) Err() error {
	return it.err
}

// Records returns an iterator over GEDCOM records using Go 1.23 range-over-func.
// It yields (*RawRecord, nil) for each successfully parsed record.
// On parse error, it yields (nil, error) — exactly once — and stops iteration.
//
// IMPORTANT: when err is non-nil, record is nil. Always check err before
// dereferencing record:
//
//	for record, err := range parser.Records(reader) {
//	    if err != nil {
//	        return err  // record is nil here
//	    }
//	    // safe to use record
//	}
//
// This function provides a modern, idiomatic alternative to [RecordIterator]
// for streaming GEDCOM record processing. The reader should already be wrapped
// with charset.NewReader() for encoding normalization. Lines longer than
// [MaxLineBytes] cause iteration to abort with [bufio.ErrTooLong].
//
// Early termination is supported — breaking from the loop will stop iteration:
//
//	for record, err := range parser.Records(reader) {
//	    if err != nil {
//	        return err
//	    }
//	    if record.Type == "TRLR" {
//	        break // stop at trailer
//	    }
//	}
func Records(r io.Reader) iter.Seq2[*RawRecord, error] {
	return func(yield func(*RawRecord, error) bool) {
		it := NewRecordIterator(r)
		for it.Next() {
			if !yield(it.Record(), nil) {
				return // consumer broke out of loop
			}
		}
		if err := it.Err(); err != nil {
			yield(nil, err)
		}
	}
}

// RecordsWithOffset returns an iterator over GEDCOM records with accurate byte offset tracking.
// It yields (*RawRecord, nil) for each successfully parsed record.
// On parse error, it yields (nil, error) — exactly once — and stops iteration.
// When err is non-nil, record is nil; always check err before dereferencing record.
//
// This is the range-over-func equivalent of [RecordIteratorWithOffset], providing
// precise ByteOffset and ByteLength values suitable for building file indexes.
// The reader should already be wrapped with charset.NewReader() for encoding normalization.
// Lines longer than [MaxLineBytes] cause iteration to abort with [ErrLineTooLong].
//
// Usage:
//
//	for record, err := range parser.RecordsWithOffset(reader) {
//	    if err != nil {
//	        return err
//	    }
//	    fmt.Printf("Record %s at offset %d, length %d\n",
//	        record.Type, record.ByteOffset, record.ByteLength)
//	}
func RecordsWithOffset(r io.Reader) iter.Seq2[*RawRecord, error] {
	return func(yield func(*RawRecord, error) bool) {
		it := NewRecordIteratorWithOffset(r)
		for it.Next() {
			if !yield(it.Record(), nil) {
				return // consumer broke out of loop
			}
		}
		if err := it.Err(); err != nil {
			yield(nil, err)
		}
	}
}
