package gedcom

// CropRegion defines a subregion of an image to display (GEDCOM 7.0 CROP).
// Used to specify which portion of an image should be displayed when referenced.
type CropRegion struct {
	// Height is the region height in pixels (default: image height - Top)
	Height int `json:",omitempty"`

	// Left is the horizontal position of top-left corner (default 0)
	Left int `json:",omitempty"`

	// Top is the vertical position of top-left corner (default 0)
	Top int `json:",omitempty"`

	// Width is the region width in pixels (default: image width - Left)
	Width int `json:",omitempty"`
}

// MediaFile represents a single file within a multimedia record (GEDCOM 7.0 FILE structure).
// A multimedia object can have multiple files (e.g., high-res and thumbnail versions).
type MediaFile struct {
	// FileRef is the path or URL to the file
	FileRef string `json:",omitempty"`

	// Form is the MIME type (required in GEDCOM 7.0, e.g., "image/jpeg", "video/mp4")
	Form string `json:",omitempty"`

	// MediaType is the category (MEDI tag; FORM.TYPE in a 5.5.1 multimedia
	// record): AUDIO, BOOK, CARD, ELECTRONIC, PHOTO, VIDEO, etc.
	MediaType string `json:",omitempty"`

	// MediaPhrase is the PHRASE under MEDI (GEDCOM 7.0), typically
	// describing a MEDI OTHER. Written only beneath a MediaType.
	MediaPhrase string `json:",omitempty"`

	// Title is a descriptive title for this file
	Title string `json:",omitempty"`

	// Translations contains alternate versions (transcripts, thumbnails, different formats)
	Translations []*MediaTranslation `json:",omitempty"`
}

// MediaLink represents a reference to a multimedia object (GEDCOM 7.0 MULTIMEDIA_LINK).
// Used when entities (individuals, families, events) reference media objects.
//
// A link is either a pointer to an OBJE record (MediaXRef set) or, in GEDCOM
// 5.5 and 5.5.1, an inline link that names its files itself (MediaXRef empty,
// Files set). GEDCOM 7.0 has no inline link; see the encoder package for how
// one is written for 7.0 output.
type MediaLink struct {
	// Crop is an optional crop region for images
	Crop *CropRegion `json:",omitempty"`

	// Files are the files of an inline link (GEDCOM 5.5/5.5.1 "OBJE" with no
	// pointer): FILE with its FORM and MEDI (or the record-style TYPE). In the
	// 5.5 layout, where FORM is written beside FILE rather than under it, that
	// FORM fills the Form of each file that has none, or with no FILE is kept
	// in a file with an empty FileRef. Empty on a pointer link; the encoder
	// ignores Files on a link that has a MediaXRef.
	//
	// 5.5 output is lossy: it writes one FORM, the first non-empty Form, and
	// each non-empty FileRef, dropping MediaType, Title and Translations.
	// 5.5.1 and 7.0 output write a file with an empty FileRef as a FILE with
	// no value, under which its FORM is written.
	Files []*MediaFile `json:",omitempty"`

	// MediaXRef is the pointer to the OBJE record (e.g., "@O1@")
	MediaXRef string `json:",omitempty"`

	// NoteXRefs are XRef pointers to shared NOTE/SNOTE records (e.g. "@N1@").
	NoteXRefs []string `json:",omitempty"`

	// InlineNotes are note text values written directly on this media link
	// (NOTE <text> form, including CONT/CONC continuations).
	InlineNotes []string `json:",omitempty"`

	// Title is an optional title that overrides the FILE's TITL
	Title string `json:",omitempty"`
}

// MediaObject represents a multimedia record (GEDCOM 7.0 MULTIMEDIA_RECORD).
// This is a top-level record (0 @Xn@ OBJE) that stores the actual media files.
type MediaObject struct {
	// ChangeDate is when the record was last modified (CHAN tag)
	ChangeDate *ChangeDate `json:",omitempty"`

	// CreationDate is when the record was created (CREA tag, GEDCOM 7.0)
	CreationDate *ChangeDate `json:",omitempty"`

	// Files contains 1:M file references (required, at least one)
	Files []*MediaFile `json:",omitempty"`

	// NoteXRefs are XRef pointers to shared note records (e.g. "@N1@") carried
	// by NOTE tags. SNOTE pointers are tracked separately in SharedNoteXRefs;
	// the two slices partition this record's note pointers and never overlap.
	NoteXRefs []string `json:",omitempty"`

	// InlineNotes are note text values written directly on this record
	// (1 NOTE <text> form, including CONT/CONC continuations).
	InlineNotes []string `json:",omitempty"`

	// RefNumbers are user reference numbers (REFN tags, repeatable), each
	// with its optional TYPE, in file order.
	RefNumbers []RefNumber `json:",omitempty"`

	// SharedNoteXRefs are cross-references to shared note records (SNOTE tags,
	// GEDCOM 7.0). Disjoint from NoteXRefs, which holds the NOTE-tag pointers.
	SharedNoteXRefs []string `json:",omitempty"`

	// Restriction is the access restriction level (RESN tag)
	Restriction string `json:",omitempty"`

	// SourceCitations are source citations with page/quality details
	SourceCitations []*SourceCitation `json:",omitempty"`

	// Tags contains all raw tags for this media object (for unknown/custom tags),
	// a lossless read-side record per ADR 0003. While Record.Tags is non-empty
	// it does not drive encoding; a record written from its entity appends
	// the extension (_-prefixed) level-1 tags here after its typed fields.
	// See Record.Tags for the rule that governs what a decoded record writes.
	Tags []*Tag `json:",omitempty"`

	// UIDs are unique identifiers (UID tag, can have multiple in GEDCOM 7.0)
	UIDs []string `json:",omitempty"`

	// ExternalIDs are external identifiers (EXID tags, GEDCOM 7.0).
	// Links this record to external systems like FamilySearch, Ancestry, etc.
	ExternalIDs []*ExternalID `json:",omitempty"`

	// XRef is the cross-reference identifier for this media object
	XRef string `json:",omitempty"`
}

// AllNotes returns this media object's inline notes followed by the text of the
// shared notes it points at, resolved against doc. NoteXRefs (NOTE pointers) and
// SharedNoteXRefs (GEDCOM 7.0 SNOTE pointers) partition this record's pointers,
// so both are followed and neither can duplicate the other. Shared notes that do
// not resolve are skipped. Returns nil when there are no notes.
func (m *MediaObject) AllNotes(doc *Document) []string {
	xrefs := make([]string, 0, len(m.NoteXRefs)+len(m.SharedNoteXRefs))
	xrefs = append(xrefs, m.NoteXRefs...)
	xrefs = append(xrefs, m.SharedNoteXRefs...)
	return allNotes(doc, m.InlineNotes, xrefs)
}

// MediaTranslation represents an alternate version of a file (GEDCOM 7.0 FILE-TRAN).
// Examples: transcripts for audio, thumbnails for images, different format conversions.
type MediaTranslation struct {
	// FileRef is the path or URL to the translation/alternate file
	FileRef string `json:",omitempty"`

	// Form is the MIME type of the translation file
	Form string `json:",omitempty"`
}
