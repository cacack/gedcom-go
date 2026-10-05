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

	// ByteOffset is the starting byte position of this record. [LazyParser]
	// iterators report it as an absolute offset in the file; [Records],
	// [NewRecordIterator], [RecordsWithOffset] and [NewRecordIteratorWithOffset]
	// report it relative to the start of the reader they are given. For a
	// reader wrapped with charset.NewReader that means bytes of the decoded
	// stream, which are not positions in the original file.
	ByteOffset int64

	// ByteLength is the total number of bytes for this record
	ByteLength int64
}

// RecordIterator provides streaming access to GEDCOM records.
// It groups lines into records (level-0 boundaries) without loading the entire file into memory.
type RecordIterator struct {
	scanner    *lineScanner
	parser     *Parser
	current    *RawRecord
	pending    *Line // Buffered line that belongs to next record
	pendingPos int64 // Byte offset where the pending line starts
	err        error
	byteOffset int64 // Bytes consumed so far, including any pending line
}

// NewRecordIterator creates a new RecordIterator that reads from the given reader.
// The reader should already be wrapped with charset.NewReader() for encoding normalization.
// Record ByteOffset values are relative to the start of r; use the
// [LazyParser] iterators for offsets that are absolute in a seekable file.
//
// Lines longer than [MaxLineBytes] cause iteration to abort with an error
// rather than allocating unboundedly. Spec-compliant GEDCOM lines never
// approach this limit.
//
// A reader failure outranks the truncated final line it leaves behind: the
// fragment is dropped and [RecordIterator.Err] reports the reader error. See
// [lineScanner.Truncated].
func NewRecordIterator(r io.Reader) *RecordIterator {
	return newRecordIteratorAt(r, 0)
}

// newRecordIteratorAt is [NewRecordIterator] for a reader positioned base
// bytes into the file, so ByteOffset values stay file-relative.
func newRecordIteratorAt(r io.Reader, base int64) *RecordIterator {
	scanner := newLineScanner(r)
	// Explicit buffer with documented ceiling; default bufio.Scanner cap is
	// 64 KiB which can be too small for files containing embedded BLOBs.
	scanner.Buffer(make([]byte, 0, 4096), MaxLineBytes)

	return &RecordIterator{
		scanner:    scanner,
		parser:     NewParser(),
		byteOffset: base,
	}
}

// Next advances the iterator to the next record.
// Returns true if a record is available, false when iteration is complete or on error.
func (it *RecordIterator) Next() bool {
	if it.err != nil {
		return false
	}

	// ByteOffset starts at the next unread line; a buffered level-0 line
	// overrides it below.
	record := &RawRecord{
		ByteOffset: it.byteOffset,
		Lines:      make([]*Line, 0, recordLinesInitialCap),
	}

	// Use pending line from previous iteration if available
	if it.pending != nil {
		record.XRef = it.pending.XRef
		record.Type = it.pending.Tag
		record.Lines = append(record.Lines, it.pending)
		record.ByteOffset = it.pendingPos
		it.pending = nil
	} else if !it.scanNextLine(record) {
		// Read first line of record
		return false
	}

	// Read subordinate lines until next level-0 tag or EOF
	for it.scanner.Scan() {
		if it.scanner.Truncated() {
			// Not a line — the head of one the reader cut short. Drop it and
			// let the reader error below be the failure that is reported.
			break
		}
		text := it.scanner.Text()
		lineStart := it.byteOffset
		it.byteOffset += int64(it.scanner.Consumed())

		line, err := it.parser.ParseLine(text)
		if err != nil {
			it.err = err
			return false
		}

		if line.Level == 0 {
			// This line belongs to next record - buffer it
			it.pending = line
			it.pendingPos = lineStart
			break
		}

		record.Lines = append(record.Lines, line)
	}

	if err := it.scanner.Err(); err != nil {
		it.err = err
		return false
	}

	// The record ends where the buffered level-0 line (which belongs to the
	// next record) starts, or at the last byte consumed.
	end := it.byteOffset
	if it.pending != nil {
		end = it.pendingPos
	}
	record.ByteLength = end - record.ByteOffset

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

	if it.scanner.Truncated() {
		// Not a line — the head of one the reader cut short. Report the reader
		// failure rather than whatever the fragment parses as.
		it.err = it.scanner.Err()
		return false
	}

	text := it.scanner.Text()
	it.byteOffset += int64(it.scanner.Consumed())

	line, err := it.parser.ParseLine(text)
	if err != nil {
		it.err = err
		return false
	}

	record.XRef = line.XRef
	record.Type = line.Tag
	record.Lines = append(record.Lines, line)

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

// RecordIteratorWithOffset iterates records like [RecordIterator], reporting
// the same ByteOffset and ByteLength values (relative to the start of the
// reader). [LazyParser.BuildIndex] uses it. It counts line terminators
// independently of [RecordIterator], so the two must agree;
// TestRecordIterators_ExactByteOffsets enforces that.
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

// NewRecordIteratorWithOffset creates a [RecordIteratorWithOffset]. Record
// ByteOffset values are relative to the start of r.
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
// Record ByteOffset values are relative to the start of r; use
// [LazyParser.Records] for offsets that are absolute in a seekable file.
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
	return recordsAt(r, 0)
}

// recordsAt is [Records] for a reader positioned base bytes into the file,
// so ByteOffset values stay file-relative.
func recordsAt(r io.Reader, base int64) iter.Seq2[*RawRecord, error] {
	return func(yield func(*RawRecord, error) bool) {
		it := newRecordIteratorAt(r, base)
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

// RecordsWithOffset returns an iterator over GEDCOM records with byte offsets.
// It yields (*RawRecord, nil) for each successfully parsed record.
// On parse error, it yields (nil, error) — exactly once — and stops iteration.
// When err is non-nil, record is nil; always check err before dereferencing record.
//
// This is the range-over-func equivalent of [RecordIteratorWithOffset]. Its
// ByteOffset and ByteLength values match [Records] and are relative to the
// start of r.
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
