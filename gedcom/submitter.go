package gedcom

// Submitter represents a person or organization who submitted genealogical data.
// In GEDCOM files, submitters are identified by SUBM records and provide attribution
// and contact information for data sources.
type Submitter struct {
	// XRef is the cross-reference identifier for this submitter
	XRef string

	// Name is the submitter's name
	Name string

	// Address is the submitter's physical address
	Address *Address

	// Phone contains phone numbers (can have multiple)
	Phone []string

	// Email contains email addresses (can have multiple)
	Email []string

	// Language contains preferred languages (can have multiple)
	Language []string

	// Notes are references to note records
	Notes []string

	// ExternalIDs are external identifiers (EXID tags, GEDCOM 7.0).
	// Links this record to external systems like FamilySearch, Ancestry, etc.
	ExternalIDs []*ExternalID

	// Tags contains all raw tags for this submitter (for unknown/custom tags)
	Tags []*Tag
}
