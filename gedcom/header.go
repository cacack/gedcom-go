package gedcom

import "time"

// Header contains metadata about the GEDCOM file.
type Header struct {
	// Version is the GEDCOM specification version
	Version Version

	// Encoding is the character encoding used in the file
	Encoding Encoding

	// SourceSystem identifies the software that created the file
	SourceSystem string

	// Date is when the file was created
	Date time.Time

	// Language is the primary language used in the file (optional)
	Language string

	// Copyright notice (optional)
	Copyright string

	// Submitter reference (optional)
	Submitter string

	// Raw tags from the header for preserving unknown/custom tags
	Tags []*Tag
}
