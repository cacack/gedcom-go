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

	// ChangeDate is when the record was last modified (CHAN tag)
	ChangeDate *ChangeDate

	// CreationDate is when the record was created (CREA tag, GEDCOM 7.0)
	CreationDate *ChangeDate

	// RefNumbers are user reference numbers (REFN tags, repeatable), each
	// with its optional TYPE, in file order.
	RefNumbers []RefNumber

	// UIDs are unique identifiers (UID tags, repeatable in GEDCOM 7.0, or
	// the _UID vendor extension in 5.5/5.5.1), in file order.
	UIDs []string

	// ExternalIDs are external identifiers (EXID tags, GEDCOM 7.0).
	// Links this record to external systems like FamilySearch, Ancestry, etc.
	ExternalIDs []*ExternalID

	// Tags contains all raw tags for this submitter (for unknown/custom tags),
	// a lossless read-side record per ADR 0003. While Record.Tags is non-empty
	// it does not drive encoding; a record written from its entity appends
	// the extension (_-prefixed) level-1 tags here after its typed fields.
	// See Record.Tags for the rule that governs what a decoded record writes.
	Tags []*Tag
}

// AllNotes returns this submitter's inline notes followed by the text of any
// shared notes referenced by NoteXRefs, resolved against doc. Shared notes that
// do not resolve are skipped. Returns nil when there are no notes.
func (s *Submitter) AllNotes(doc *Document) []string {
	return allNotes(doc, s.InlineNotes, s.NoteXRefs)
}
