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

	// Tags contains all the tags that make up this record
	Tags []*Tag

	// LineNumber is the line number where the record starts
	LineNumber int

	// Parsed entity (one of: Individual, Family, Source, Repository, Note, MediaObject)
	// Will be populated during decoding based on the Type
	Entity interface{}
}

// IsIndividual returns true if this record is an individual record.
func (r *Record) IsIndividual() bool {
	return r.Type == RecordTypeIndividual
}

// IsFamily returns true if this record is a family record.
func (r *Record) IsFamily() bool {
	return r.Type == RecordTypeFamily
}

// IsSource returns true if this record is a source record.
func (r *Record) IsSource() bool {
	return r.Type == RecordTypeSource
}

// GetIndividual returns the record as an Individual if it's the correct type.
func (r *Record) GetIndividual() (*Individual, bool) {
	if ind, ok := r.Entity.(*Individual); ok {
		return ind, true
	}
	return nil, false
}

// GetFamily returns the record as a Family if it's the correct type.
func (r *Record) GetFamily() (*Family, bool) {
	if fam, ok := r.Entity.(*Family); ok {
		return fam, true
	}
	return nil, false
}

// GetSource returns the record as a Source if it's the correct type.
func (r *Record) GetSource() (*Source, bool) {
	if src, ok := r.Entity.(*Source); ok {
		return src, true
	}
	return nil, false
}

// GetSubmitter returns the record as a Submitter if it's the correct type.
func (r *Record) GetSubmitter() (*Submitter, bool) {
	if subm, ok := r.Entity.(*Submitter); ok {
		return subm, true
	}
	return nil, false
}

// GetRepository returns the record as a Repository if it's the correct type.
func (r *Record) GetRepository() (*Repository, bool) {
	if repo, ok := r.Entity.(*Repository); ok {
		return repo, true
	}
	return nil, false
}

// GetNote returns the record as a Note if it's the correct type.
func (r *Record) GetNote() (*Note, bool) {
	if note, ok := r.Entity.(*Note); ok {
		return note, true
	}
	return nil, false
}

// GetMediaObject returns the record as a MediaObject if it's the correct type.
func (r *Record) GetMediaObject() (*MediaObject, bool) {
	if media, ok := r.Entity.(*MediaObject); ok {
		return media, true
	}
	return nil, false
}

// IsSharedNote returns true if this record is a shared note record (GEDCOM 7.0).
func (r *Record) IsSharedNote() bool {
	return r.Type == RecordTypeSharedNote
}

// GetSharedNote returns the record as a SharedNote if it's the correct type.
func (r *Record) GetSharedNote() (*SharedNote, bool) {
	if snote, ok := r.Entity.(*SharedNote); ok {
		return snote, true
	}
	return nil, false
}
