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

	// NumberOfChildren is the declared number of children (NCHI tag)
	NumberOfChildren string

	// Events contains family events (marriage, divorce, etc.)
	Events []*Event

	// SourceCitations are source citations with page/quality details
	SourceCitations []*SourceCitation

	// Notes are references to note records
	Notes []string

	// MediaRefs are references to media objects
	MediaRefs []string

	// LDSOrdinances are LDS (Latter-Day Saints) ordinances (SLGS - spouse sealing)
	LDSOrdinances []*LDSOrdinance

	// ChangeDate is when the record was last modified (CHAN tag)
	ChangeDate *ChangeDate

	// CreationDate is when the record was created (CREA tag, GEDCOM 7.0)
	CreationDate *ChangeDate

	// RefNumber is the user reference number (REFN tag)
	RefNumber string

	// UID is the unique identifier (UID tag)
	UID string

	// Tags contains all raw tags for this family (for unknown/custom tags)
	Tags []*Tag
}
