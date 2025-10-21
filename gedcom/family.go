package gedcom

// Family represents a family unit (husband, wife, and children).
type Family struct {
	// XRef is the cross-reference identifier for this family
	XRef string

	// Husband is the XRef to the husband individual
	Husband string

	// Wife is the XRef to the wife individual
	Wife string

	// Children are XRefs to child individuals
	Children []string

	// Events contains family events (marriage, divorce, etc.)
	Events []*Event

	// Sources are references to source citations
	Sources []string

	// Notes are references to note records
	Notes []string

	// MediaRefs are references to media objects
	MediaRefs []string

	// Tags contains all raw tags for this family (for unknown/custom tags)
	Tags []*Tag
}
