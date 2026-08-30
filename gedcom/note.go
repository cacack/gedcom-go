package gedcom

// Note represents a textual note or annotation.
type Note struct {
	// XRef is the cross-reference identifier for this note
	XRef string

	// Text is the whole note content. Multi-line bodies carried by CONT/CONC
	// continuation lines are folded in on decode (joined with "\n"), and the
	// encoder splits Text back out into CONT/CONC on write. This mirrors
	// SharedNote.Text: one field, one representation.
	Text string

	// ExternalIDs are external identifiers (EXID tags, GEDCOM 7.0).
	// Links this record to external systems like FamilySearch, Ancestry, etc.
	ExternalIDs []*ExternalID

	// Tags contains all raw tags for this note (for unknown/custom tags)
	Tags []*Tag
}
