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

	// UIDs are unique identifiers, in file order. 5.5.1 defines no UID; the
	// vendor _UID (or a stray UID) is read here, and the encoder writes them
	// as _UID for 5.5/5.5.1 output, so a NOTE record converted to and from a
	// 7.0 SNOTE keeps them (SharedNote.UIDs).
	UIDs []string

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

// InlineNoteDetail holds the GEDCOM 7.0 substructures of one inline note
// (NOTE <text>) beyond its text, which stays in the owner's InlineNotes:
// the MIME type and language of the text and its translations. It mirrors
// the same fields of SharedNote. Citations beneath an inline note are not
// typed; they stay in Record.Tags (ADR 0003).
//
// Only Individual carries it (Individual.InlineNoteDetails) so far; the same
// substructures beneath any other owner's inline note stay in Record.Tags.
type InlineNoteDetail struct {
	// MIME is the media type of the note text (NOTE.MIME), e.g. "text/html".
	MIME string

	// Language is the BCP 47 language tag of the note text (NOTE.LANG).
	Language string

	// Translations are the note's translations (NOTE.TRAN), in file order.
	Translations []*SharedNoteTranslation
}
