package gedcom

// Repository represents a physical or digital location where sources are stored.
type Repository struct {
	// XRef is the cross-reference identifier for this repository
	XRef string

	// Name is the repository name
	Name string

	// Address is the physical address
	Address *Address

	// NoteXRefs are XRef pointers to shared NOTE/SNOTE records (e.g. "@N1@").
	NoteXRefs []string

	// InlineNotes are note text values written directly on this record
	// (1 NOTE <text> form, including CONT/CONC continuations).
	InlineNotes []string

	// Notes is deprecated: use NoteXRefs and InlineNotes instead. It is kept
	// for backward compatibility and populated during decode with the inline
	// note text and shared-note XRefs interleaved in their original GEDCOM
	// order (not the NoteXRefs-then-InlineNotes order of the split fields).
	//
	// Deprecated: use NoteXRefs and InlineNotes.
	Notes []string

	// ExternalIDs are external identifiers (EXID tags, GEDCOM 7.0).
	// Links this record to external systems like FamilySearch, Ancestry, etc.
	ExternalIDs []*ExternalID

	// Tags contains all raw tags for this repository (for unknown/custom tags)
	Tags []*Tag
}

// AllNotes returns this repository's inline notes followed by the text of any
// shared notes referenced by NoteXRefs, resolved against doc. Shared notes that
// do not resolve are skipped. Returns nil when there are no notes.
func (r *Repository) AllNotes(doc *Document) []string {
	return allNotes(doc, r.InlineNotes, r.NoteXRefs)
}

// InlineRepository represents an inline repository definition within a Source.
// Used when a Source references a repository by name rather than by XRef.
type InlineRepository struct {
	// Name is the repository name
	Name string
}

// SourceRepositoryLink is a Source's reference to a Repository, with per-link
// metadata (call numbers, media type, notes). It models the REPO substructure
// of a SOUR record, which can carry CALN (call number), MEDI (media type), and
// NOTE subordinates in addition to the repository pointer itself.
type SourceRepositoryLink struct {
	// XRef is the repository pointer, e.g. "@R1@". XRef and Inline are
	// mutually exclusive: when XRef is non-empty, Inline is nil (the decoder
	// enforces this even for malformed input that carries both).
	XRef string

	// Inline is set when the source references the repository by name rather
	// than by XRef (i.e. the GEDCOM has `1 REPO` with a name value and no
	// separate repository record).
	Inline *InlineRepository

	// CallNumbers holds CALN values (multiple allowed per GEDCOM spec).
	CallNumbers []string

	// MediaType is the MEDI subordinate of the first CALN that carries one
	// (manuscript, photo, etc.). When multiple CALNs have differing MEDI
	// values, use CallNumberMedia to recover the per-CALN pairing.
	//
	// The encoder only round-trips MediaType faithfully for a single-CALN
	// link (it is emitted as that CALN's MEDI when CallNumberMedia has no
	// entry for it). For multi-CALN links the encoder relies on
	// CallNumberMedia; a MediaType set without a matching CallNumberMedia
	// entry is not written out.
	MediaType string

	// CallNumberMedia indexes MEDI values by their parent CALN, when CALN and
	// MEDI need to stay paired. Empty when no MEDI subordinates exist.
	//
	// Keyed by the CALN string value. If a record carries two CALN entries
	// with identical text but different MEDI subordinates, the later MEDI
	// wins (last-writer-wins); CallNumbers still retains both entries.
	CallNumberMedia map[string]string

	// Notes carries NOTE subordinates of the REPO link (not the source).
	Notes []string
}

// Address represents a physical or digital address.
type Address struct {
	// Line1 is the first address line
	Line1 string

	// Line2 is the second address line (optional)
	Line2 string

	// Line3 is the third address line (optional)
	Line3 string

	// City is the city name
	City string

	// State is the state/province
	State string

	// PostalCode is the postal/zip code
	PostalCode string

	// Country is the country name
	Country string

	// Phone is the phone number (optional)
	Phone string

	// Email is the email address (optional)
	Email string

	// Website is the website URL (optional)
	Website string
}
