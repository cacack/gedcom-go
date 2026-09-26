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

// SourceRepositoryLink is one of a Source's references to a Repository, with
// per-link metadata (call numbers with their media types, notes). It models a
// REPO substructure of a SOUR record, which can carry CALN (call number, each
// with an optional MEDI media type) and NOTE/SNOTE subordinates in addition to
// the repository pointer itself.
type SourceRepositoryLink struct {
	// XRef is the repository pointer, e.g. "@R1@". XRef and Inline are
	// mutually exclusive: when XRef is non-empty, Inline is nil (the decoder
	// enforces this even for malformed input that carries both).
	XRef string

	// Inline is set when the source references the repository by name rather
	// than by XRef (i.e. the GEDCOM has `1 REPO` with a name value and no
	// separate repository record).
	Inline *InlineRepository

	// CallNumbers holds the link's CALN entries in source order, each with
	// its own MEDI media type (multiple CALNs are allowed per GEDCOM spec,
	// and each CALN carries at most one MEDI). Two CALNs with the same text
	// but different MEDI values stay separate entries.
	CallNumbers []*CallNumber

	// NoteXRefs are XRef pointers to shared NOTE/SNOTE records (e.g. "@N1@")
	// carried by NOTE subordinates of the REPO link (not the source).
	NoteXRefs []string

	// InlineNotes are note text values written directly on the REPO link
	// (NOTE <text> form, including CONT/CONC continuations).
	InlineNotes []string
}

// CallNumber is one CALN entry of a source's repository link, together with
// the MEDI media type subordinate to it. It models
//
//	n CALN <Text>
//	  +1 MEDI <Enum>      (0:1)
//	     +2 PHRASE <Text> (0:1, GEDCOM 7.0)
//
// MEDI and PHRASE are the only substructures GEDCOM 5.5, 5.5.1 and 7.0 define
// under CALN.
type CallNumber struct {
	// Value is the call number text (the CALN line value). It may be empty:
	// a CALN line with no value can still carry a MEDI.
	Value string

	// MediaType is the MEDI value (book, manuscript, VIDEO, etc.), stored as
	// written. Empty when the CALN has no MEDI subordinate.
	MediaType string

	// MediaPhrase is the free-text PHRASE under MEDI (GEDCOM 7.0), typically
	// used with MEDI OTHER to describe the medium. Empty when absent.
	MediaPhrase string
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
