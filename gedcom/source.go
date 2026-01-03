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

	// Repository is an inline repository definition (alternative to RepositoryRef)
	Repository *InlineRepository

	// Media are references to media objects with optional crop/title
	Media []*MediaLink

	// Notes are references to note records
	Notes []string

	// ChangeDate is when the record was last modified (CHAN tag)
	ChangeDate *ChangeDate

	// CreationDate is when the record was created (CREA tag, GEDCOM 7.0)
	CreationDate *ChangeDate

	// RefNumber is the user reference number (REFN tag)
	RefNumber string

	// UID is the unique identifier (UID tag)
	UID string

	// Tags contains all raw tags for this source (for unknown/custom tags)
	Tags []*Tag
}

// SourceCitationData represents extracted text and date from a source citation.
type SourceCitationData struct {
	// Date is the date extracted from the source
	Date string

	// Text is the quoted text from the source
	Text string
}

// SourceCitation represents a citation of a source with location and quality information.
type SourceCitation struct {
	// SourceXRef is the cross-reference to the source record (e.g., "@S1@")
	SourceXRef string

	// Page is the page or location within the source (e.g., "Page 42, Entry 103")
	Page string

	// Quality is the evidence quality assessment (0-3 scale per GEDCOM spec)
	// 0 = unreliable evidence or estimated data
	// 1 = questionable reliability of evidence
	// 2 = secondary evidence, data officially recorded sometime after event
	// 3 = direct and primary evidence used, or by dominance of the evidence
	Quality int

	// Data contains optional extracted text and date from the source
	Data *SourceCitationData

	// AncestryAPID is the Ancestry Permanent Identifier from the _APID tag.
	// This is an Ancestry.com vendor extension that links the citation to a
	// specific record in an Ancestry database. Use AncestryAPID.URL() to
	// reconstruct the original Ancestry.com record URL.
	AncestryAPID *AncestryAPID
}
