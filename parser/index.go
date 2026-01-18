package parser

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io"
)

// IndexEntry represents a single record's location in a GEDCOM file.
type IndexEntry struct {
	// XRef is the cross-reference identifier (e.g., "@I1@")
	// Empty for records without XRefs (like HEAD, TRLR)
	XRef string

	// Type is the record type tag (e.g., "INDI", "FAM", "HEAD")
	Type string

	// ByteOffset is the starting byte position of this record in the file
	ByteOffset int64

	// ByteLength is the total number of bytes for this record
	ByteLength int64
}

// IndexVersion is the current version of the index format.
// Increment when making incompatible changes to the index structure.
const IndexVersion byte = 1

// RecordIndex provides O(1) lookup of records by XRef after an O(n) build phase.
// The index maps XRef strings to their byte offsets in the original file.
type RecordIndex struct {
	// entries maps XRef to IndexEntry for records with XRefs
	entries map[string]IndexEntry

	// typeEntries maps record types to entries for records without XRefs
	// Key format: "TYPE" (e.g., "HEAD", "TRLR")
	typeEntries map[string]IndexEntry

	// encoding is the detected character encoding of the indexed file
	encoding string

	// version is the index format version
	version byte
}

// NewRecordIndex creates an empty RecordIndex.
func NewRecordIndex() *RecordIndex {
	return &RecordIndex{
		entries:     make(map[string]IndexEntry),
		typeEntries: make(map[string]IndexEntry),
		version:     IndexVersion,
	}
}

// BuildIndex builds an index by scanning the entire file once.
// The reader should be positioned at the start of the file.
// After building, the reader position is at EOF.
func BuildIndex(r io.Reader) (*RecordIndex, error) {
	idx := NewRecordIndex()

	it := NewRecordIteratorWithOffset(r)
	for it.Next() {
		rec := it.Record()
		entry := IndexEntry{
			XRef:       rec.XRef,
			Type:       rec.Type,
			ByteOffset: rec.ByteOffset,
			ByteLength: rec.ByteLength,
		}

		if rec.XRef != "" {
			idx.entries[rec.XRef] = entry
		} else {
			// For records without XRef (HEAD, TRLR), index by type
			idx.typeEntries[rec.Type] = entry
		}
	}

	if err := it.Err(); err != nil {
		return nil, fmt.Errorf("building index: %w", err)
	}

	return idx, nil
}

// Lookup returns the index entry for a given XRef.
// Returns the entry and true if found, or zero entry and false if not found.
func (idx *RecordIndex) Lookup(xref string) (IndexEntry, bool) {
	entry, ok := idx.entries[xref]
	return entry, ok
}

// LookupByType returns the index entry for a record type without XRef.
// This is useful for finding HEAD or TRLR records.
// Returns the entry and true if found, or zero entry and false if not found.
func (idx *RecordIndex) LookupByType(recordType string) (IndexEntry, bool) {
	entry, ok := idx.typeEntries[recordType]
	return entry, ok
}

// XRefs returns all XRefs in the index.
func (idx *RecordIndex) XRefs() []string {
	xrefs := make([]string, 0, len(idx.entries))
	for xref := range idx.entries {
		xrefs = append(xrefs, xref)
	}
	return xrefs
}

// Types returns all record types without XRefs in the index.
func (idx *RecordIndex) Types() []string {
	types := make([]string, 0, len(idx.typeEntries))
	for t := range idx.typeEntries {
		types = append(types, t)
	}
	return types
}

// Len returns the total number of indexed entries (both XRef and type-based).
func (idx *RecordIndex) Len() int {
	return len(idx.entries) + len(idx.typeEntries)
}

// SetEncoding sets the encoding string for this index.
// This should be set during index building to record the detected encoding.
func (idx *RecordIndex) SetEncoding(enc string) {
	idx.encoding = enc
}

// Encoding returns the encoding that was detected when the index was built.
func (idx *RecordIndex) Encoding() string {
	return idx.encoding
}

// indexData is the serializable form of RecordIndex for gob encoding.
type indexData struct {
	Version     byte
	Encoding    string
	Entries     map[string]IndexEntry
	TypeEntries map[string]IndexEntry
}

// Save writes the index to the given writer in gob format.
// The format includes a version byte for future compatibility.
func (idx *RecordIndex) Save(w io.Writer) error {
	data := indexData{
		Version:     idx.version,
		Encoding:    idx.encoding,
		Entries:     idx.entries,
		TypeEntries: idx.typeEntries,
	}

	enc := gob.NewEncoder(w)
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encoding index: %w", err)
	}

	return nil
}

// ErrIndexVersionMismatch is returned when loading an index with an incompatible version.
var ErrIndexVersionMismatch = errors.New("index version mismatch")

// LoadIndex reads an index from the given reader.
// Returns an error if the index version is incompatible.
func LoadIndex(r io.Reader) (*RecordIndex, error) {
	var data indexData

	dec := gob.NewDecoder(r)
	if err := dec.Decode(&data); err != nil {
		return nil, fmt.Errorf("decoding index: %w", err)
	}

	if data.Version != IndexVersion {
		return nil, fmt.Errorf("%w: got %d, want %d", ErrIndexVersionMismatch, data.Version, IndexVersion)
	}

	idx := &RecordIndex{
		version:     data.Version,
		encoding:    data.Encoding,
		entries:     data.Entries,
		typeEntries: data.TypeEntries,
	}

	// Ensure maps are initialized even if empty
	if idx.entries == nil {
		idx.entries = make(map[string]IndexEntry)
	}
	if idx.typeEntries == nil {
		idx.typeEntries = make(map[string]IndexEntry)
	}

	return idx, nil
}
