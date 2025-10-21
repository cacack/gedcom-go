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
