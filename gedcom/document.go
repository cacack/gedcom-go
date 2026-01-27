package gedcom

// Document represents a complete GEDCOM file with all its records.
type Document struct {
	// Header contains file metadata
	Header *Header

	// Records contains all top-level records (individuals, families, sources, etc.)
	Records []*Record

	// Trailer marks the end of the file
	Trailer *Trailer

	// XRefMap provides fast lookup of records by cross-reference ID
	// Map key is the XRef (e.g., "@I1@"), value is the Record
	XRefMap map[string]*Record

	// Vendor identifies the software that created this GEDCOM file.
	// Detected from the HEAD.SOUR tag during decoding.
	Vendor Vendor

	// Schema contains GEDCOM 7.0 schema definitions that map custom tags to URIs.
	// Extracted from the HEAD.SCHMA structure during decoding.
	Schema *SchemaDefinition
}

// GetRecord returns the record with the given cross-reference ID.
// Returns nil if the record is not found.
func (d *Document) GetRecord(xref string) *Record {
	if d.XRefMap == nil {
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
	var sources []*Source
	for _, record := range d.Records {
		if src, ok := record.GetSource(); ok {
			sources = append(sources, src)
		}
	}
	return sources
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
	var notes []*SharedNote
	for _, record := range d.Records {
		if snote, ok := record.GetSharedNote(); ok {
			notes = append(notes, snote)
		}
	}
	return notes
}
