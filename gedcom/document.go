package gedcom

// Document represents a complete GEDCOM file with all its records.
//
// The document model carries json struct tags so that encoding/json writes
// each record once and leaves out what is absent: XRefMap is not written, and
// a field is omitted when its zero value means absent. Member names are the Go
// field names. Tag.Level, Date.Calendar and Date.Modifier are always written,
// because their zero is a value (level 0, Gregorian, an exact date).
type Document struct {
	// Header contains file metadata
	Header *Header `json:",omitempty"`

	// Records contains all top-level records (individuals, families, sources, etc.)
	Records []*Record `json:",omitempty"`

	// Trailer marks the end of the file
	Trailer *Trailer `json:",omitempty"`

	// XRefMap provides fast lookup of records by cross-reference ID
	// Map key is the XRef (e.g., "@I1@"), value is the Record.
	//
	// When two records in Records share an XRef, which is malformed, the map
	// holds the first: the decoder keeps both records, maps the first and
	// reports the second as DUPLICATE_XREF, and Clone, Subset and merge.Combine
	// follow the same first-definition rule.
	//
	// It is an index over Records rather than data, so encoding/json does not
	// write it; call ReindexXRefs on a document read back from JSON.
	XRefMap map[string]*Record `json:"-"`

	// Vendor identifies the software that created this GEDCOM file.
	// Detected from the HEAD.SOUR tag during decoding.
	Vendor Vendor `json:",omitempty"`

	// Schema contains GEDCOM 7.0 schema definitions that map custom tags to URIs.
	// Extracted from the HEAD.SCHMA structure during decoding.
	Schema *SchemaDefinition `json:",omitempty"`
}

// GetRecord returns the record with the given cross-reference ID.
// Returns nil if the record is not found.
func (d *Document) GetRecord(xref string) *Record {
	if d == nil || d.XRefMap == nil {
		return nil
	}
	return d.XRefMap[xref]
}

// GetIndividual returns the individual record with the given XRef.
// Returns nil if not found or if the record is not an individual.
func (d *Document) GetIndividual(xref string) *Individual {
	record := d.GetRecord(xref)
	if record == nil {
		return nil
	}
	if ind, ok := record.GetIndividual(); ok {
		return ind
	}
	return nil
}

// GetFamily returns the family record with the given XRef.
// Returns nil if not found or if the record is not a family.
func (d *Document) GetFamily(xref string) *Family {
	record := d.GetRecord(xref)
	if record == nil {
		return nil
	}
	if fam, ok := record.GetFamily(); ok {
		return fam
	}
	return nil
}

// GetSource returns the source record with the given XRef.
// Returns nil if not found or if the record is not a source.
func (d *Document) GetSource(xref string) *Source {
	record := d.GetRecord(xref)
	if record == nil {
		return nil
	}
	if src, ok := record.GetSource(); ok {
		return src
	}
	return nil
}

// Individuals returns all individual records in the document.
func (d *Document) Individuals() []*Individual {
	if d == nil {
		return nil
	}
	var individuals []*Individual
	for _, record := range d.Records {
		if ind, ok := record.GetIndividual(); ok {
			individuals = append(individuals, ind)
		}
	}
	return individuals
}

// Families returns all family records in the document.
func (d *Document) Families() []*Family {
	if d == nil {
		return nil
	}
	var families []*Family
	for _, record := range d.Records {
		if fam, ok := record.GetFamily(); ok {
			families = append(families, fam)
		}
	}
	return families
}

// Sources returns all source records in the document.
func (d *Document) Sources() []*Source {
	if d == nil {
		return nil
	}
	var sources []*Source
	for _, record := range d.Records {
		if src, ok := record.GetSource(); ok {
			sources = append(sources, src)
		}
	}
	return sources
}

// GetSubmission returns the submission record with the given XRef.
// Returns nil if not found or if the record is not a submission.
func (d *Document) GetSubmission(xref string) *Submission {
	record := d.GetRecord(xref)
	if record == nil {
		return nil
	}
	if subn, ok := record.GetSubmission(); ok {
		return subn
	}
	return nil
}

// Submissions returns all submission records in the document.
func (d *Document) Submissions() []*Submission {
	if d == nil {
		return nil
	}
	var submissions []*Submission
	for _, record := range d.Records {
		if subn, ok := record.GetSubmission(); ok {
			submissions = append(submissions, subn)
		}
	}
	return submissions
}

// GetSubmitter returns the submitter record with the given XRef.
// Returns nil if not found or if the record is not a submitter.
func (d *Document) GetSubmitter(xref string) *Submitter {
	record := d.GetRecord(xref)
	if record == nil {
		return nil
	}
	if subm, ok := record.GetSubmitter(); ok {
		return subm
	}
	return nil
}

// Submitters returns all submitter records in the document.
func (d *Document) Submitters() []*Submitter {
	if d == nil {
		return nil
	}
	var submitters []*Submitter
	for _, record := range d.Records {
		if subm, ok := record.GetSubmitter(); ok {
			submitters = append(submitters, subm)
		}
	}
	return submitters
}

// GetRepository returns the repository record with the given XRef.
// Returns nil if not found or if the record is not a repository.
func (d *Document) GetRepository(xref string) *Repository {
	record := d.GetRecord(xref)
	if record == nil {
		return nil
	}
	if repo, ok := record.GetRepository(); ok {
		return repo
	}
	return nil
}

// Repositories returns all repository records in the document.
func (d *Document) Repositories() []*Repository {
	if d == nil {
		return nil
	}
	var repositories []*Repository
	for _, record := range d.Records {
		if repo, ok := record.GetRepository(); ok {
			repositories = append(repositories, repo)
		}
	}
	return repositories
}

// GetNote returns the note record with the given XRef.
// Returns nil if not found or if the record is not a note.
func (d *Document) GetNote(xref string) *Note {
	record := d.GetRecord(xref)
	if record == nil {
		return nil
	}
	if note, ok := record.GetNote(); ok {
		return note
	}
	return nil
}

// Notes returns all note records in the document.
func (d *Document) Notes() []*Note {
	if d == nil {
		return nil
	}
	var notes []*Note
	for _, record := range d.Records {
		if note, ok := record.GetNote(); ok {
			notes = append(notes, note)
		}
	}
	return notes
}

// GetMediaObject returns the media object with the given XRef.
// Returns nil if not found or if the record is not a media object.
func (d *Document) GetMediaObject(xref string) *MediaObject {
	record := d.GetRecord(xref)
	if record == nil {
		return nil
	}
	if media, ok := record.GetMediaObject(); ok {
		return media
	}
	return nil
}

// MediaObjects returns all media object records in the document.
func (d *Document) MediaObjects() []*MediaObject {
	if d == nil {
		return nil
	}
	var objects []*MediaObject
	for _, record := range d.Records {
		if media, ok := record.GetMediaObject(); ok {
			objects = append(objects, media)
		}
	}
	return objects
}

// GetSharedNote returns the shared note record with the given XRef.
// Returns nil if not found or if the record is not a shared note.
func (d *Document) GetSharedNote(xref string) *SharedNote {
	record := d.GetRecord(xref)
	if record == nil {
		return nil
	}
	if snote, ok := record.GetSharedNote(); ok {
		return snote
	}
	return nil
}

// SharedNotes returns all shared note records in the document (GEDCOM 7.0).
func (d *Document) SharedNotes() []*SharedNote {
	if d == nil {
		return nil
	}
	var notes []*SharedNote
	for _, record := range d.Records {
		if snote, ok := record.GetSharedNote(); ok {
			notes = append(notes, snote)
		}
	}
	return notes
}
