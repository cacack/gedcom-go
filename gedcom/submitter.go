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

	// NoteXRefs are XRef pointers to shared NOTE/SNOTE records (e.g. "@N1@").
	NoteXRefs []string

	// InlineNotes are note text values written directly on this record
	// (1 NOTE <text> form, including CONT/CONC continuations).
	InlineNotes []string

	// Notes is deprecated: use NoteXRefs and InlineNotes instead. It is kept
	// for backward compatibility and populated during decode as the
	// concatenation NoteXRefs + InlineNotes.
	//
	// Deprecated: use NoteXRefs and InlineNotes.
	Notes []string

	// ExternalIDs are external identifiers (EXID tags, GEDCOM 7.0).
	// Links this record to external systems like FamilySearch, Ancestry, etc.
	ExternalIDs []*ExternalID

	// Tags contains all raw tags for this submitter (for unknown/custom tags)
	Tags []*Tag
}

// AllNotes returns this submitter's inline notes followed by the text of any
// shared notes referenced by NoteXRefs, resolved against doc. Shared notes that
// do not resolve are skipped. Returns nil when there are no notes.
func (s *Submitter) AllNotes(doc *Document) []string {
	return allNotes(doc, s.InlineNotes, s.NoteXRefs)
}
