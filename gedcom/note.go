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

	// SourceCitations are source citations with page/quality details (SOUR).
	SourceCitations []*SourceCitation

	// ChangeDate is when the record was last modified (CHAN tag)
	ChangeDate *ChangeDate

	// RefNumbers are user reference numbers (REFN tags, repeatable), each
	// with its optional TYPE, in file order.
	RefNumbers []RefNumber

	// ExternalIDs are external identifiers (EXID tags, GEDCOM 7.0).
	// Links this record to external systems like FamilySearch, Ancestry, etc.
	ExternalIDs []*ExternalID

	// Tags contains all raw tags for this note (for unknown/custom tags),
	// a lossless read-side record per ADR 0003. While Record.Tags is non-empty
	// it does not drive encoding; a record written from its entity appends
	// the extension (_-prefixed) level-1 tags here after its typed fields.
	// See Record.Tags for the rule that governs what a decoded record writes.
	Tags []*Tag
}
