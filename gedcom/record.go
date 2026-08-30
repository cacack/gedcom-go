package gedcom

// RecordType represents the type of GEDCOM record.
type RecordType string

const (
	// RecordTypeIndividual represents a person (INDI)
	RecordTypeIndividual RecordType = "INDI"

	// RecordTypeFamily represents a family unit (FAM)
	RecordTypeFamily RecordType = "FAM"

	// RecordTypeSource represents a source of information (SOUR)
	RecordTypeSource RecordType = "SOUR"

	// RecordTypeRepository represents a repository (REPO)
	RecordTypeRepository RecordType = "REPO"

	// RecordTypeNote represents a note (NOTE)
	RecordTypeNote RecordType = "NOTE"

	// RecordTypeMedia represents a multimedia object (OBJE)
	RecordTypeMedia RecordType = "OBJE"

	// RecordTypeSubmitter represents a submitter (SUBM)
	RecordTypeSubmitter RecordType = "SUBM"

	// RecordTypeSharedNote represents a shared note (SNOTE, GEDCOM 7.0)
	RecordTypeSharedNote RecordType = "SNOTE"
)

// Record represents a top-level GEDCOM record with a cross-reference identifier.
// Records are the main entities in a GEDCOM file (individuals, families, sources, etc.).
type Record struct {
	// XRef is the cross-reference identifier (e.g., "@I1@", "@F1@")
	XRef string

	// Type is the record type (INDI, FAM, SOUR, etc.)
	Type RecordType

	// Value is the value from the level 0 line (used for NOTE records, etc.)
	Value string

	// Tags contains all the tags that make up this record, in document order.
	//
	// Tags is authoritative when encoding. The encoder writes Tags verbatim
	// whenever it is non-empty and builds tags from Entity only for a record
	// that has none. Editing a typed field on a decoded record therefore does
	// not change the encoded output: the write succeeds in memory, survives a
	// round of reads, and is silently absent from the file. There is no error
	// and no diagnostic.
	//
	// Two supported ways to change what a decoded record encodes to:
	//
	//   - Edit the matching entry in Tags.
	//   - Set Tags to nil, so the encoder re-derives the whole record from
	//     Entity. This drops any raw tag the typed model does not represent,
	//     so it trades losslessness for the typed model's view.
	//
	// The converter keeps both in step and is the safe option when converting
	// between GEDCOM versions.
	Tags []*Tag

	// LineNumber is the line number where the record starts
	LineNumber int

	// Parsed entity (one of: Individual, Family, Source, Repository, Note, MediaObject)
	// Will be populated during decoding based on the Type
	Entity interface{}
}

// The Is* and Get* accessors below never panic. A nil *Record receiver, and an
// Entity holding a typed nil pointer, both report false rather than returning a
// value the caller cannot use. See docs/decisions/0007-error-transparency.md.

// IsIndividual returns true if this record is an individual record.
func (r *Record) IsIndividual() bool {
	if r == nil {
		return false
	}
	return r.Type == RecordTypeIndividual
}

// IsFamily returns true if this record is a family record.
func (r *Record) IsFamily() bool {
	if r == nil {
		return false
	}
	return r.Type == RecordTypeFamily
}

// IsSource returns true if this record is a source record.
func (r *Record) IsSource() bool {
	if r == nil {
		return false
	}
	return r.Type == RecordTypeSource
}

// GetIndividual returns the record as an Individual if it's the correct type.
func (r *Record) GetIndividual() (*Individual, bool) {
	if r == nil {
		return nil, false
	}
	if ind, ok := r.Entity.(*Individual); ok && ind != nil {
		return ind, true
	}
	return nil, false
}

// GetFamily returns the record as a Family if it's the correct type.
func (r *Record) GetFamily() (*Family, bool) {
	if r == nil {
		return nil, false
	}
	if fam, ok := r.Entity.(*Family); ok && fam != nil {
		return fam, true
	}
	return nil, false
}

// GetSource returns the record as a Source if it's the correct type.
func (r *Record) GetSource() (*Source, bool) {
	if r == nil {
		return nil, false
	}
	if src, ok := r.Entity.(*Source); ok && src != nil {
		return src, true
	}
	return nil, false
}

// GetSubmitter returns the record as a Submitter if it's the correct type.
func (r *Record) GetSubmitter() (*Submitter, bool) {
	if r == nil {
		return nil, false
	}
	if subm, ok := r.Entity.(*Submitter); ok && subm != nil {
		return subm, true
	}
	return nil, false
}

// GetRepository returns the record as a Repository if it's the correct type.
func (r *Record) GetRepository() (*Repository, bool) {
	if r == nil {
		return nil, false
	}
	if repo, ok := r.Entity.(*Repository); ok && repo != nil {
		return repo, true
	}
	return nil, false
}

// GetNote returns the record as a Note if it's the correct type.
func (r *Record) GetNote() (*Note, bool) {
	if r == nil {
		return nil, false
	}
	if note, ok := r.Entity.(*Note); ok && note != nil {
		return note, true
	}
	return nil, false
}

// GetMediaObject returns the record as a MediaObject if it's the correct type.
func (r *Record) GetMediaObject() (*MediaObject, bool) {
	if r == nil {
		return nil, false
	}
	if media, ok := r.Entity.(*MediaObject); ok && media != nil {
		return media, true
	}
	return nil, false
}

// IsSharedNote returns true if this record is a shared note record (GEDCOM 7.0).
func (r *Record) IsSharedNote() bool {
	if r == nil {
		return false
	}
	return r.Type == RecordTypeSharedNote
}

// GetSharedNote returns the record as a SharedNote if it's the correct type.
func (r *Record) GetSharedNote() (*SharedNote, bool) {
	if r == nil {
		return nil, false
	}
	if snote, ok := r.Entity.(*SharedNote); ok && snote != nil {
		return snote, true
	}
	return nil, false
}
