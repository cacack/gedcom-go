package gedcom

// Source represents a source of genealogical information.
type Source struct {
	// XRef is the cross-reference identifier for this source
	XRef string

	// Title is the source title
	Title string

	// Author is the source author/originator
	Author string

	// Publication is publication information
	Publication string

	// Text is the actual text from the source
	Text string

	// RepositoryRef is the XRef to the repository where this source is stored
	RepositoryRef string

	// MediaRefs are references to media objects
	MediaRefs []string

	// Notes are references to note records
	Notes []string

	// Tags contains all raw tags for this source (for unknown/custom tags)
	Tags []*Tag
}
