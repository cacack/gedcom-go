package gedcom

// SharedNote represents a GEDCOM 7.0 shared note record (SNOTE).
// Unlike regular NOTE records, shared notes support MIME types, language tags,
// and translations for internationalization.
type SharedNote struct {
	// XRef is the cross-reference identifier for this shared note
	XRef string `json:",omitempty"`

	// Text is the full note content. Multi-line bodies carried by CONT
	// continuation lines are folded in on decode (joined with "\n"), the same
	// as Note.Text.
	Text string `json:",omitempty"`

	// MIME is the media type of the text content (e.g., "text/plain", "text/html")
	MIME string `json:",omitempty"`

	// Language is the BCP 47 language tag for the text (e.g., "en", "de", "zh-Hans")
	Language string `json:",omitempty"`

	// Translations contains alternative representations of this note in different languages
	Translations []*SharedNoteTranslation `json:",omitempty"`

	// SourceCitations are source citations with page/quality details
	SourceCitations []*SourceCitation `json:",omitempty"`

	// ExternalIDs are external identifiers (EXID tags, GEDCOM 7.0).
	// Links this record to external systems like FamilySearch, Ancestry, etc.
	ExternalIDs []*ExternalID `json:",omitempty"`

	// ChangeDate is when the record was last modified (CHAN tag)
	ChangeDate *ChangeDate `json:",omitempty"`

	// CreationDate is when the record was created (CREA tag)
	CreationDate *ChangeDate `json:",omitempty"`

	// RefNumbers are user reference numbers (REFN tags, repeatable), each
	// with its optional TYPE, in file order.
	RefNumbers []RefNumber `json:",omitempty"`

	// UIDs are unique identifiers (UID tags, repeatable), in file order.
	UIDs []string `json:",omitempty"`

	// Tags contains all raw tags for this shared note (for unknown/custom tags),
	// a lossless read-side record per ADR 0003. While Record.Tags is non-empty
	// it does not drive encoding; a record written from its entity appends
	// the extension (_-prefixed) level-1 tags here after its typed fields.
	// See Record.Tags for the rule that governs what a decoded record writes.
	Tags []*Tag `json:",omitempty"`
}

// SharedNoteTranslation represents a translation of a shared note into another language.
// This mirrors the Transliteration pattern used for PersonalName.
type SharedNoteTranslation struct {
	// Value is the translated text content
	Value string `json:",omitempty"`

	// MIME is the media type of the translated text (e.g., "text/plain", "text/html")
	MIME string `json:",omitempty"`

	// Language is the BCP 47 language tag for this translation (e.g., "en", "de", "zh-Hans")
	Language string `json:",omitempty"`
}
