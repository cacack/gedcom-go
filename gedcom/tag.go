package gedcom

// Tag represents a GEDCOM tag-value pair with hierarchical level information.
// Tags are the fundamental building blocks of GEDCOM files, representing
// structured data in a hierarchical format.
type Tag struct {
	// Level is the hierarchical depth (0 for top-level records, 1+ for nested data)
	Level int

	// Tag is the GEDCOM tag name (e.g., "INDI", "NAME", "BIRT")
	Tag string

	// Value is the optional value associated with the tag
	Value string

	// XRef is the optional cross-reference identifier (e.g., "@I1@")
	// Only present for records that can be referenced by other records
	XRef string

	// LineNumber is the line number in the source file where this tag appears
	// Used for error reporting and debugging
	LineNumber int
}

// HasValue returns true if the tag has a non-empty value.
func (t *Tag) HasValue() bool {
	return t.Value != ""
}

// HasXRef returns true if the tag has a cross-reference identifier.
func (t *Tag) HasXRef() bool {
	return t.XRef != ""
}
