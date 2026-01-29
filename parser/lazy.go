package parser

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"iter"
)

// ErrNoIndex is returned when attempting to use indexed operations without an index.
var ErrNoIndex = errors.New("no index available: call BuildIndex or LoadIndex first")

// ErrRecordNotFound is returned when a record cannot be found.
var ErrRecordNotFound = errors.New("record not found")

// LazyParser provides lazy/incremental parsing of GEDCOM files.
// It combines streaming iteration with indexed random access for efficient
// partial file processing.
type LazyParser struct {
	rs    io.ReadSeeker
	index *RecordIndex
}

// NewLazyParser creates a new LazyParser from an io.ReadSeeker.
// The reader must support seeking for indexed access operations.
func NewLazyParser(rs io.ReadSeeker) *LazyParser {
	return &LazyParser{
		rs: rs,
	}
}

// BuildIndex scans the entire file to build an index for O(1) record lookup.
// After calling BuildIndex, use FindRecord for efficient random access.
// The reader is rewound to the beginning after building the index.
func (p *LazyParser) BuildIndex() error {
	// Seek to beginning
	if _, err := p.rs.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seeking to start: %w", err)
	}

	// Build index
	idx, err := BuildIndex(p.rs)
	if err != nil {
		return err
	}

	p.index = idx

	// Rewind for subsequent operations
	if _, err := p.rs.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("rewinding after index build: %w", err)
	}

	return nil
}

// LoadIndex loads a pre-built index from the given reader.
// Using a pre-built index avoids the O(n) scan of BuildIndex.
func (p *LazyParser) LoadIndex(r io.Reader) error {
	idx, err := LoadIndex(r)
	if err != nil {
		return err
	}

	p.index = idx
	return nil
}

// SaveIndex writes the current index to the given writer.
// Returns ErrNoIndex if no index has been built or loaded.
func (p *LazyParser) SaveIndex(w io.Writer) error {
	if p.index == nil {
		return ErrNoIndex
	}

	return p.index.Save(w)
}

// HasIndex returns true if an index is available.
func (p *LazyParser) HasIndex() bool {
	return p.index != nil
}

// Index returns the current index, or nil if none is loaded.
func (p *LazyParser) Index() *RecordIndex {
	return p.index
}

// FindRecord locates and parses a specific record by XRef.
// Requires an index to be built or loaded first.
// Returns ErrNoIndex if no index is available.
// Returns ErrRecordNotFound if the XRef is not in the index.
func (p *LazyParser) FindRecord(xref string) (*RawRecord, error) {
	if p.index == nil {
		return nil, ErrNoIndex
	}

	entry, ok := p.index.Lookup(xref)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrRecordNotFound, xref)
	}

	return p.readRecordAt(entry)
}

// FindRecordByType locates and parses a record by type (for records without XRef).
// This is useful for finding HEAD or TRLR records.
// Requires an index to be built or loaded first.
func (p *LazyParser) FindRecordByType(recordType string) (*RawRecord, error) {
	if p.index == nil {
		return nil, ErrNoIndex
	}

	entry, ok := p.index.LookupByType(recordType)
	if !ok {
		return nil, fmt.Errorf("%w: type %s", ErrRecordNotFound, recordType)
	}

	return p.readRecordAt(entry)
}

// readRecordAt seeks to the given entry and parses the record.
func (p *LazyParser) readRecordAt(entry IndexEntry) (*RawRecord, error) {
	// Seek to record position
	if _, err := p.rs.Seek(entry.ByteOffset, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seeking to record: %w", err)
	}

	// Read exactly the bytes for this record
	limitedReader := io.LimitReader(p.rs, entry.ByteLength)

	// Parse the record
	record := &RawRecord{
		ByteOffset: entry.ByteOffset,
		ByteLength: entry.ByteLength,
	}

	parser := NewParser()
	scanner := bufio.NewScanner(limitedReader)
	scanner.Split(scanGEDCOMLines)

	for scanner.Scan() {
		line, err := parser.ParseLine(scanner.Text())
		if err != nil {
			return nil, fmt.Errorf("parsing line: %w", err)
		}

		if len(record.Lines) == 0 {
			record.XRef = line.XRef
			record.Type = line.Tag
		}

		record.Lines = append(record.Lines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning record: %w", err)
	}

	return record, nil
}

// Iterate returns a RecordIterator for streaming through records.
// The iterator starts from the current position of the reader.
// For full file iteration, seek to the beginning first.
func (p *LazyParser) Iterate() *RecordIterator {
	return NewRecordIterator(p.rs)
}

// IterateFrom seeks to the given byte offset and returns an iterator.
// This allows resuming iteration from a known position.
func (p *LazyParser) IterateFrom(offset int64) (*RecordIterator, error) {
	if _, err := p.rs.Seek(offset, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seeking to offset: %w", err)
	}

	return NewRecordIterator(p.rs), nil
}

// IterateAll seeks to the beginning and returns an iterator for all records.
func (p *LazyParser) IterateAll() (*RecordIterator, error) {
	return p.IterateFrom(0)
}

// Records returns an iterator over records from the current position using
// Go 1.23 range-over-func. This is the range-over-func equivalent of [Iterate].
//
// Usage:
//
//	for record, err := range lp.Records() {
//	    if err != nil {
//	        return err
//	    }
//	    // process record
//	}
func (p *LazyParser) Records() iter.Seq2[*RawRecord, error] {
	return Records(p.rs)
}

// RecordsFrom seeks to the given byte offset and returns an iterator over
// records using Go 1.23 range-over-func. This is the range-over-func
// equivalent of [IterateFrom].
//
// If seeking fails, the error is yielded as the first iteration result.
//
// Usage:
//
//	for record, err := range lp.RecordsFrom(offset) {
//	    if err != nil {
//	        return err
//	    }
//	    // process record
//	}
func (p *LazyParser) RecordsFrom(offset int64) iter.Seq2[*RawRecord, error] {
	return func(yield func(*RawRecord, error) bool) {
		if _, err := p.rs.Seek(offset, io.SeekStart); err != nil {
			yield(nil, fmt.Errorf("seeking to offset: %w", err))
			return
		}
		for record, err := range Records(p.rs) {
			if !yield(record, err) {
				return
			}
			if err != nil {
				return
			}
		}
	}
}

// AllRecords seeks to the beginning and returns an iterator over all records
// using Go 1.23 range-over-func. This is the range-over-func equivalent
// of [IterateAll].
//
// Usage:
//
//	for record, err := range lp.AllRecords() {
//	    if err != nil {
//	        return err
//	    }
//	    // process record
//	}
func (p *LazyParser) AllRecords() iter.Seq2[*RawRecord, error] {
	return p.RecordsFrom(0)
}

// XRefs returns all XRefs in the index.
// Returns nil if no index is available.
func (p *LazyParser) XRefs() []string {
	if p.index == nil {
		return nil
	}
	return p.index.XRefs()
}

// RecordCount returns the total number of indexed records.
// Returns 0 if no index is available.
func (p *LazyParser) RecordCount() int {
	if p.index == nil {
		return 0
	}
	return p.index.Len()
}
