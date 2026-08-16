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
	// Normally present only on level-0 records, the tags other records can
	// point at. It is also populated on a subordinate tag when the source
	// line carried a malformed identifier the decoder recovered (e.g.
	// "1 @I 1@ NOTE"), so that identifier is preserved verbatim.
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
