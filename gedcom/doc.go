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
