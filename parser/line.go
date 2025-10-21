package parser

// Line represents a single parsed line from a GEDCOM file.
// GEDCOM files use a line-based format with hierarchical levels.
// Each line format: LEVEL [XREF] TAG [VALUE]
type Line struct {
	// Level indicates the hierarchical depth (0, 1, 2, etc.)
	Level int

	// Tag is the GEDCOM tag (e.g., HEAD, INDI, NAME, BIRT)
	Tag string

	// Value is the optional value associated with the tag
	Value string

	// XRef is the optional cross-reference identifier (e.g., @I1@)
	XRef string

	// LineNumber is the line number in the source file (1-based)
	// Used for error reporting
	LineNumber int
}
