package gedcom

// SharedNote represents a GEDCOM 7.0 shared note record (SNOTE).
// Unlike regular NOTE records, shared notes support MIME types, language tags,
// and translations for internationalization.
type SharedNote struct {
	// XRef is the cross-reference identifier for this shared note
	XRef string

	// Text is the note content
	Text string

	// MIME is the media type of the text content (e.g., "text/plain", "text/html")
	MIME string

	// Language is the BCP 47 language tag for the text (e.g., "en", "de", "zh-Hans")
	Language string

	// Translations contains alternative representations of this note in different languages
	Translations []*SharedNoteTranslation

	// SourceCitations are source citations with page/quality details
	SourceCitations []*SourceCitation

	// ExternalIDs are external identifiers (EXID tags, GEDCOM 7.0).
	// Links this record to external systems like FamilySearch, Ancestry, etc.
	ExternalIDs []*ExternalID

	// ChangeDate is when the record was last modified (CHAN tag)
	ChangeDate *ChangeDate

	// Tags contains all raw tags for this shared note (for unknown/custom tags)
	Tags []*Tag
}

// SharedNoteTranslation represents a translation of a shared note into another language.
// This mirrors the Transliteration pattern used for PersonalName.
type SharedNoteTranslation struct {
	// Value is the translated text content
	Value string

	// MIME is the media type of the translated text (e.g., "text/plain", "text/html")
	MIME string

	// Language is the BCP 47 language tag for this translation (e.g., "en", "de", "zh-Hans")
	Language string
}
