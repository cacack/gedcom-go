// Package gedcom defines the core data types for representing GEDCOM genealogy data.
//
// This package provides the fundamental structures for working with GEDCOM files,
// including individuals, families, sources, events, and other genealogical records.
// It supports GEDCOM versions 5.5, 5.5.1, and 7.0.
//
// The main entry point is the Document type, which contains a parsed GEDCOM file
// with all its records. Individual records can be accessed through helper methods
// or by using the XRefMap for cross-reference lookup.
//
// Example usage:
//
//	// After decoding a GEDCOM file
//	doc, _ := decoder.Decode(reader)
//
//	// Access individuals
//	for _, individual := range doc.Individuals() {
//	    fmt.Printf("Name: %s\n", individual.Name)
//	}
//
//	// Lookup by cross-reference
//	person := doc.GetIndividual("@I1@")
//	if person != nil {
//	    fmt.Printf("Found: %s\n", person.Name)
//	}
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
