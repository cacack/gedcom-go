package gedcom

// Repository represents a physical or digital location where sources are stored.
type Repository struct {
	// XRef is the cross-reference identifier for this repository
	XRef string

	// Name is the repository name
	Name string

	// Address is the physical address (ADDR).
	Address *Address

	// Phone holds the repository's phone numbers (PHON, {0:3}), in document
	// order. PHON is a sibling of ADDR in ADDRESS_STRUCTURE, not part of it.
	Phone []string

	// Email holds the repository's email addresses (EMAIL, {0:3}, GEDCOM
	// 5.5.1+), in document order.
	Email []string

	// Fax holds the repository's fax numbers (FAX, {0:3}, GEDCOM 5.5.1+), in
	// document order.
	Fax []string

	// Website holds the repository's web addresses (WWW, {0:3}, GEDCOM
	// 5.5.1+), in document order.
	Website []string

	// NoteXRefs are XRef pointers to shared NOTE/SNOTE records (e.g. "@N1@").
	NoteXRefs []string

	// InlineNotes are note text values written directly on this record
	// (1 NOTE <text> form, including CONT/CONC continuations).
	InlineNotes []string

	// ExternalIDs are external identifiers (EXID tags, GEDCOM 7.0).
	// Links this record to external systems like FamilySearch, Ancestry, etc.
	ExternalIDs []*ExternalID

	// Tags contains all raw tags for this repository (for unknown/custom tags),
	// a lossless read-side record per ADR 0003. It does not drive encoding --
	// see Record.Tags for the rule that governs what a decoded record writes.
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

	// NoteXRefs are XRef pointers to shared NOTE/SNOTE records (e.g. "@N1@")
	// carried by NOTE subordinates of the REPO link (not the source).
	NoteXRefs []string

	// InlineNotes are note text values written directly on the REPO link
	// (NOTE <text> form, including CONT/CONC continuations).
	InlineNotes []string
}

// Address represents the ADDR structure: a postal address and its ADR1-3,
// CITY, STAE, POST and CTRY subordinates, each {0:1}.
//
// The contact tags PHON, EMAIL, FAX and WWW are siblings of ADDR in the
// GEDCOM grammar, not part of it, and are {0:3}. They are typed as []string
// fields on the structure that owns the address: Repository, Event and
// Attribute (all four), and Submitter (Phone and Email).
//
// Address is comparable with ==; a compile-time assertion in this package
// keeps it that way.
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
}

// Address must stay comparable: every subordinate of ADDR is {0:1}, so no
// field needs a slice or map. Adding one makes this line fail to compile.
var _ = map[Address]struct{}{}
