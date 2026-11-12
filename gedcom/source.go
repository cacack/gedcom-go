package gedcom

// Source represents a source of genealogical information.
type Source struct {
	// XRef is the cross-reference identifier for this source
	XRef string

	// Title is the source title.
	//
	// Title, Author, Publication and Text are free text: CONT continuation
	// lines are folded in on decode, joined with "\n", and CONC continuations
	// are concatenated with no separator, so each can hold several lines.
	Title string

	// Author is the source author/originator.
	Author string

	// Publication is publication information.
	Publication string

	// Abbreviation is a short title used for sorting, filing and retrieving
	// the source (ABBR tag).
	Abbreviation string

	// Text is the actual text from the source.
	Text string

	// TextMIME is the media type of Text (TEXT.MIME, GEDCOM 7.0), e.g.
	// "text/plain" or "text/html". Empty when absent. Written only beneath
	// a non-empty Text.
	TextMIME string

	// TextLanguage is the BCP 47 language tag of Text (TEXT.LANG, GEDCOM
	// 7.0). Empty when absent. Written only beneath a non-empty Text.
	TextLanguage string

	// Data describes what the source records (DATA tag): the events it
	// covers, the agency responsible for it, and notes about it. nil when the
	// record has no DATA substructure.
	Data *SourceData

	// RepositoryLinks are the source's repository links, one per REPO
	// substructure, in source order. Each carries the repository pointer
	// (XRef) or an inline repository definition (Inline), plus the call
	// numbers (each with its media type) and per-link notes that the REPO
	// substructure can hold. GEDCOM 5.5.1 and 7.0 allow any number of REPO
	// links on a source record.
	RepositoryLinks []*SourceRepositoryLink

	// Media are references to media objects with optional crop/title
	Media []*MediaLink

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

	// UIDs are unique identifiers (UID tags, repeatable in GEDCOM 7.0), in
	// file order.
	UIDs []string

	// ExternalIDs are external identifiers (EXID tags, GEDCOM 7.0).
	// Links this record to external systems like FamilySearch, Ancestry, etc.
	ExternalIDs []*ExternalID

	// Tags contains all raw tags for this source (for unknown/custom tags),
	// a lossless read-side record per ADR 0003. While Record.Tags is non-empty
	// it does not drive encoding; a record written from its entity appends
	// the extension (_-prefixed) level-1 tags here after its typed fields.
	// See Record.Tags for the rule that governs what a decoded record writes.
	Tags []*Tag
}

// AllNotes returns this source's inline notes followed by the text of any
// shared notes referenced by NoteXRefs, resolved against doc. Shared notes that
// do not resolve are skipped. Returns nil when there are no notes.
func (s *Source) AllNotes(doc *Document) []string {
	return allNotes(doc, s.InlineNotes, s.NoteXRefs)
}

// SourceData is the DATA substructure of a source record: the events the
// source records, the agency responsible for the original data, and notes
// about the data. It has the same shape in GEDCOM 5.5, 5.5.1 and 7.0 (7.0 adds
// SNOTE, which lands in NoteXRefs like a NOTE pointer).
type SourceData struct {
	// Events are the kinds of events the source records, one per DATA.EVEN
	// substructure, in source order.
	Events []*SourceDataEvent

	// Agency is the person, agency or entity responsible for the original
	// data (DATA.AGNC).
	Agency string

	// NoteXRefs are XRef pointers to shared NOTE/SNOTE records (e.g. "@N1@").
	NoteXRefs []string

	// InlineNotes are note text values written directly under DATA.
	InlineNotes []string
}

// SourceDataEvent is one DATA.EVEN substructure of a source record: a set of
// event types the source records, with the period and place it covers.
type SourceDataEvent struct {
	// Types is the EVEN payload as written: a comma-separated list of event
	// tags such as "BIRT, DEAT, MARR".
	Types string

	// Date is the period the recorded events cover (DATE, a date period such
	// as "FROM 1820 TO 1825"). Empty when absent.
	Date string

	// DatePhrase is the PHRASE under DATE (GEDCOM 7.0). Written only beneath
	// a Date.
	DatePhrase string

	// PlaceDetail is the jurisdiction the recorded events cover (PLAC). nil
	// when absent.
	PlaceDetail *PlaceDetail
}

// SourceCitationEvent is the EVEN substructure of a source citation: the type
// of event the cited source records, and the role the citing record's subject
// played in it.
type SourceCitationEvent struct {
	// Type is the EVEN payload: the event type cited from, e.g. "BIRT".
	Type string

	// Phrase is the PHRASE under EVEN (GEDCOM 7.0).
	Phrase string

	// Role is the subject's role in the event (ROLE), e.g. "CHIL" or
	// "WITN". Empty when absent.
	Role string

	// RolePhrase is the PHRASE under ROLE (GEDCOM 7.0), typically describing
	// a ROLE OTHER. Written only beneath a Role.
	RolePhrase string
}

// SourceCitationData represents extracted text and date from a source citation.
type SourceCitationData struct {
	// Date is the date extracted from the source
	Date string

	// Text holds the passages quoted from the source, one entry per DATA.TEXT
	// line in document order. TEXT is repeatable ({0:M}) under a citation's
	// DATA in every GEDCOM version, so a citation quoting two passages carries
	// two entries. nil means the DATA block had no TEXT.
	Text []*SourceText
}

// SourceText is one TEXT substructure of a source citation's DATA block: a
// verbatim passage copied from the source, with the optional GEDCOM 7.0 MIME
// and LANG qualifiers that describe it.
type SourceText struct {
	// Value is the passage (the TEXT tag payload). Multi-line text carried by
	// CONT continuation lines is folded in on decode, joined with "\n"; CONC
	// continuations are concatenated with no separator.
	Value string

	// MIME is the media type of Value (TEXT.MIME, GEDCOM 7.0), e.g.
	// "text/plain" or "text/html". Empty when absent.
	MIME string

	// Language is the BCP 47 language tag of Value (TEXT.LANG, GEDCOM 7.0),
	// e.g. "en" or "de". Empty when absent.
	Language string
}

// SourceCitation represents a citation of a source with location and quality information.
type SourceCitation struct {
	// SourceXRef is the cross-reference to the source record (e.g., "@S1@").
	// It is empty for an inline citation (see InlineText).
	SourceXRef string

	// InlineText is the source description of a GEDCOM 5.5/5.5.1 citation
	// written without a source record ("1 SOUR Family bible ..."), with its
	// CONT continuations joined by "\n" and CONC continuations concatenated.
	// It is set only when the SOUR payload is not a pointer, in which case
	// SourceXRef is empty. Both are empty for a citation with no payload
	// ("1 SOUR", as Ancestry writes to carry an _APID). The encoder writes
	// InlineText as the SOUR value only when SourceXRef is empty: if both are
	// set, the pointer is written and InlineText is not. The citation's TEXT
	// lines have no typed home and stay in Record.Tags only. GEDCOM 7.0 has
	// no inline citation; converter.Convert promotes one to a SOUR record on
	// upgrade, from Record.Tags, so a record with no Tags keeps its inline
	// citation.
	InlineText string

	// Page is the page or location within the source (e.g., "Page 42, Entry 103").
	// CONT continuation lines are folded in on decode, joined with "\n"; CONC
	// continuations are concatenated with no separator.
	Page string

	// Quality is the evidence quality assessment (0-3 scale per GEDCOM spec).
	// nil means the citation carried no QUAY tag; every value in range is a
	// real assertion, including 0 -- which is why this is a pointer rather
	// than a bare int whose zero value would be indistinguishable from absent.
	//
	// 0 = unreliable evidence or estimated data
	// 1 = questionable reliability of evidence
	// 2 = secondary evidence, data officially recorded sometime after event
	// 3 = direct and primary evidence used, or by dominance of the evidence
	Quality *int

	// Data contains optional extracted text and date from the source
	Data *SourceCitationData

	// Event is the event the cited source records and the subject's role in
	// it (EVEN, with ROLE). nil when the citation has no EVEN.
	Event *SourceCitationEvent

	// Media are references to media objects attached to the citation (OBJE).
	Media []*MediaLink

	// AncestryAPID is the Ancestry Permanent Identifier from the _APID tag.
	// This is an Ancestry.com vendor extension that links the citation to a
	// specific record in an Ancestry database. Use AncestryAPID.URL() to
	// reconstruct the original Ancestry.com record URL.
	AncestryAPID *AncestryAPID

	// NoteXRefs are XRef pointers to shared NOTE/SNOTE records (e.g. "@N1@").
	NoteXRefs []string

	// InlineNotes are note text values written directly on this citation
	// (NOTE <text> form, including CONT/CONC continuations).
	InlineNotes []string
}
