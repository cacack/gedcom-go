package gedcom

// Individual represents a person in the GEDCOM file.
type Individual struct {
	// XRef is the cross-reference identifier for this individual
	XRef string

	// Restriction is the record's restriction notice (RESN): "confidential",
	// "locked" or "privacy" in GEDCOM 5.5/5.5.1, upper case and possibly a
	// comma-separated list ("CONFIDENTIAL, LOCKED") in 7.0. It is the flag an
	// application checks before publishing a person. Same type and spelling
	// rules as Event.Restriction.
	Restriction string

	// Names contains all name variants for this person
	Names []*PersonalName

	// Sex is the person's sex (M, F, X, U for unknown)
	Sex string

	// Events contains life events (birth, death, marriage, etc.) in file order.
	//
	// It also holds GEDCOM 7.0 negative assertions (n NO <EVENT>), decoded as
	// events with IsNegative set: a record that an event did NOT happen, not a
	// fact that it did. Use OccurredEvents for the events that happened and
	// NegativeAssertions for the NO assertions.
	Events []*Event

	// Attributes contains personal attributes (occupation, education, etc.)
	Attributes []*Attribute

	// ChildInFamilies are references to families where this person is a child
	ChildInFamilies []FamilyLink

	// SpouseInFamilies are references to families where this person is a spouse
	SpouseInFamilies []FamilyLink

	// Associations are links to associated individuals (godparents, witnesses, etc.)
	Associations []*Association

	// SourceCitations are source citations with page/quality details
	SourceCitations []*SourceCitation

	// NoteXRefs are XRef pointers to shared NOTE/SNOTE records (e.g. "@N1@").
	NoteXRefs []string

	// InlineNotes are note text values written directly on this record
	// (1 NOTE <text> form, including CONT/CONC continuations).
	InlineNotes []string

	// InlineNoteDetails holds the GEDCOM 7.0 substructures of the inline
	// notes (MIME, LANG and TRAN): InlineNoteDetails[i] describes
	// InlineNotes[i]. It is nil when no inline note has any; otherwise an
	// entry is nil, or absent past the end, for a note without them. An entry
	// past the end of InlineNotes is not written. Keep the two slices in step
	// when adding or removing a note.
	InlineNoteDetails []*InlineNoteDetail

	// Media are references to media objects with optional crop/title
	Media []*MediaLink

	// LDSOrdinances are LDS (Latter-Day Saints) ordinances (BAPL, CONL, ENDL, SLGC)
	LDSOrdinances []*LDSOrdinance

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

	// FamilySearchID is the FamilySearch Family Tree ID (_FSFTID tag).
	// This is a vendor extension from FamilySearch.org that uniquely identifies
	// an individual in their Family Tree database. Format: alphanumeric like "KWCJ-QN7".
	FamilySearchID string

	// Tags contains all raw tags for this individual (for unknown/custom tags),
	// a lossless read-side record per ADR 0003. While Record.Tags is non-empty
	// it does not drive encoding; a record written from its entity appends
	// the extension (_-prefixed) level-1 tags here after its typed fields.
	// See Record.Tags for the rule that governs what a decoded record writes.
	Tags []*Tag
}

// AllNotes returns this individual's inline notes followed by the text of any
// shared notes referenced by NoteXRefs, resolved against doc. Shared notes that
// do not resolve are skipped. Returns nil when there are no notes.
func (i *Individual) AllNotes(doc *Document) []string {
	return allNotes(doc, i.InlineNotes, i.NoteXRefs)
}

// PersonalName represents a person's name with optional components.
type PersonalName struct {
	// Full is the full name (e.g., "John /Doe/")
	Full string

	// Given is the given (first) name: the first GIVN value, else the text
	// before the surname slashes in Full. A repeated GIVN is in
	// AdditionalGiven.
	Given string

	// Surname is the family name
	Surname string

	// Prefix is the name prefix (e.g., "Dr.", "Sir")
	Prefix string

	// Suffix is the name suffix (e.g., "Jr.", "III"): the NSFX value, else
	// the text after the surname slashes in Full ("John /Smith/ Jr."), taken
	// as written whatever it holds (a title or nickname such as "(Jack)" too)
	Suffix string

	// Nickname is the person's nickname (e.g., "Bill" for William)
	Nickname string

	// SurnamePrefix is the surname prefix (e.g., "von", "de", "van der")
	SurnamePrefix string

	// AdditionalGiven, AdditionalSurnames, AdditionalPrefixes,
	// AdditionalSuffixes, AdditionalNicknames and AdditionalSurnamePrefixes
	// hold the second and later GIVN, SURN, NPFX, NSFX, NICK and SPFX values
	// of this name, in file order. GEDCOM 7.0 allows each name piece to
	// repeat; the first value is in Given, Surname, Prefix, Suffix, Nickname
	// or SurnamePrefix, and the encoder writes these after it. GEDCOM 5.5 and
	// 5.5.1 allow one of each, so they are nil for a conforming 5.x name.
	AdditionalGiven           []string
	AdditionalSurnames        []string
	AdditionalPrefixes        []string
	AdditionalSuffixes        []string
	AdditionalNicknames       []string
	AdditionalSurnamePrefixes []string

	// MarriedName is the married name from the vendor extension _MARNM
	// (MyHeritage), kept as written. MyHeritage writes the married surname
	// alone ("Gascoigne"); GEDCOM 7.0 instead records a married name as a
	// separate NAME with TYPE MARRIED, which converter.Convert adds from the
	// record's _MARNM tag (so only for a record that has Tags).
	MarriedName string

	// Type is the name type (e.g., "birth", "married", "aka")
	Type string

	// TypePhrase is the PHRASE under TYPE (GEDCOM 7.0), typically describing
	// a TYPE OTHER. Written only beneath a Type.
	TypePhrase string

	// Transliterations are alternative representations of the name in different
	// writing systems or scripts (GEDCOM 7.0 TRAN tag). Used to store the same
	// name in different languages, scripts, or romanization systems.
	Transliterations []*Transliteration

	// Phonetic are phonetic variations of the name (GEDCOM 5.5.1 FONE tag),
	// in file order. FONE is defined only in 5.5.1 (7.0 uses TRAN); the
	// encoder writes these only for 5.5.1 output, and converter.Convert to
	// 7.0 or 5.5 clears them.
	Phonetic []*NameVariation

	// Romanized are romanized variations of the name (GEDCOM 5.5.1 ROMN tag),
	// in file order. ROMN is defined only in 5.5.1 (7.0 uses TRAN); the
	// encoder writes these only for 5.5.1 output, and converter.Convert to
	// 7.0 or 5.5 clears them.
	Romanized []*NameVariation

	// SourceCitations are source citations documenting this name (NAME.SOUR).
	SourceCitations []*SourceCitation

	// NoteXRefs are XRef pointers to shared NOTE/SNOTE records (e.g. "@N1@").
	NoteXRefs []string

	// InlineNotes are note text values written directly on this name
	// (NOTE <text> form, including CONT/CONC continuations).
	InlineNotes []string
}

// Transliteration represents an alternative representation of a name in a different
// writing system, script, or language (GEDCOM 7.0 TRAN tag under NAME).
// Each transliteration can include the full transliterated name value plus
// individual name components in that writing system.
type Transliteration struct {
	// Value is the full transliterated name in GEDCOM format (e.g., "John /Doe/").
	// This is the value from the TRAN tag itself.
	Value string

	// Language is the BCP 47 language tag indicating the language/script of this
	// transliteration (GEDCOM 7.0 LANG tag). Examples: "en-GB", "ja-Latn", "zh-Hans".
	Language string

	// Given is the transliterated given (first) name (GIVN tag).
	Given string

	// Surname is the transliterated family name (SURN tag).
	Surname string

	// Prefix is the transliterated name prefix, e.g., "Dr.", "Sir" (NPFX tag).
	Prefix string

	// Suffix is the transliterated name suffix, e.g., "Jr.", "III" (NSFX tag).
	Suffix string

	// Nickname is the transliterated nickname (NICK tag).
	Nickname string

	// SurnamePrefix is the transliterated surname prefix, e.g., "von", "de" (SPFX tag).
	SurnamePrefix string
}

// NameVariation is a phonetic (FONE) or romanized (ROMN) variation of a name,
// as GEDCOM 5.5.1 defines under NAME. It is kept apart from Transliteration,
// GEDCOM 7.0's TRAN, because the two record their scheme differently: TRAN
// names a language with LANG, while FONE and ROMN name the method used with a
// required TYPE. NOTE and SOUR beneath a variation are kept only in the raw
// tags (ADR 0003).
type NameVariation struct {
	// Value is the full name variation in GEDCOM format (e.g., "Yamada /Taro/").
	// This is the value from the FONE or ROMN tag itself.
	Value string

	// Type is the method used to produce the variation (TYPE tag). GEDCOM
	// 5.5.1 defines "hangul" and "kana" for FONE and "pinyin", "romaji" and
	// "wadegiles" for ROMN, and allows any user-defined value.
	// TYPE is required {1:1}; the encoder omits it when empty.
	Type string

	// Given is the given (first) name in this variation (GIVN tag).
	Given string

	// Surname is the family name in this variation (SURN tag).
	Surname string

	// Prefix is the name prefix in this variation (NPFX tag).
	Prefix string

	// Suffix is the name suffix in this variation (NSFX tag).
	Suffix string

	// Nickname is the nickname in this variation (NICK tag).
	Nickname string

	// SurnamePrefix is the surname prefix in this variation (SPFX tag).
	SurnamePrefix string
}

// FamilyLink represents a link to a family with optional pedigree type.
type FamilyLink struct {
	// FamilyXRef is the cross-reference to the family record
	FamilyXRef string

	// Pedigree is the pedigree linkage type (e.g., "birth", "adopted", "foster", "sealing")
	// Empty string if not specified. Preserves original casing from GEDCOM.
	// Pedigree applies to child links (FAMC) only; neither spec defines it for a
	// spouse link (FAMS), where it is preserved solely so malformed input round-trips.
	Pedigree string

	// PedigreePhrase is the PHRASE under PEDI (GEDCOM 7.0), typically
	// describing a PEDI OTHER. Written only beneath a Pedigree.
	PedigreePhrase string

	// NoteXRefs are XRef pointers to shared NOTE/SNOTE records (e.g. "@N1@").
	NoteXRefs []string

	// InlineNotes are note text values written directly on this family link
	// (NOTE <text> form, including CONT/CONC continuations).
	InlineNotes []string
}

// Association represents a link to an associated individual with a role.
// Used for relationships like godparents (GODP), witnesses (WITN), etc.
type Association struct {
	// IndividualXRef is the cross-reference to the associated individual
	IndividualXRef string

	// Role is the relationship role (e.g., "GODP" for godparent, "WITN" for witness)
	// In GEDCOM 5.5 and 5.5.1 this comes from RELA tag, in GEDCOM 7.0 from ROLE
	// tag; the encoder writes RELA for 5.5.x output and ROLE otherwise.
	Role string

	// RolePhrase is the PHRASE under ROLE (GEDCOM 7.0), typically describing
	// a ROLE OTHER. Distinct from Phrase, which qualifies the ASSO itself.
	// Written only beneath a Role.
	RolePhrase string

	// RecordType is the type of the associated record (GEDCOM 5.5 TYPE
	// subordinate, e.g. "INDI" or "FAM"). Only 5.5 defines it, where it is
	// required: the encoder writes it only for 5.5 output, as "INDI" when
	// empty, and drops it for 5.5.1 and 7.0 output.
	RecordType string

	// Phrase is a human-readable description of the association (GEDCOM 7.0 PHRASE tag).
	// Used when the structured data cannot fully express the relationship.
	// Example: "Mr Stockdale" as the associated person's name when @XREF@ is unavailable.
	Phrase string

	// SourceCitations are source citations documenting this association (GEDCOM 7.0).
	// Allows citing sources for the association relationship itself.
	SourceCitations []*SourceCitation

	// NoteXRefs are XRef pointers to shared NOTE/SNOTE records (e.g. "@N1@").
	NoteXRefs []string

	// InlineNotes are note text values written directly on this association
	// (NOTE <text> form, including CONT/CONC continuations).
	InlineNotes []string
}

// AttributeType represents the type of a personal or family attribute.
//
// These are the tags the decoder routes to Individual.Attributes (and, for
// NCHI and FACT, to Family.Attributes). OCCU is an attribute, not an event:
// an occupation is found in Attributes, never in Events.
type AttributeType string

const (
	// AttributeOccupation represents an occupation (OCCU) attribute.
	AttributeOccupation AttributeType = "OCCU"
	// AttributeCaste represents a caste (CAST) attribute.
	AttributeCaste AttributeType = "CAST"
	// AttributePhysicalDescription represents a physical description (DSCR) attribute.
	AttributePhysicalDescription AttributeType = "DSCR"
	// AttributeEducation represents an education (EDUC) attribute.
	AttributeEducation AttributeType = "EDUC"
	// AttributeIDNumber represents an identification number (IDNO) attribute.
	AttributeIDNumber AttributeType = "IDNO"
	// AttributeNationality represents a nationality (NATI) attribute.
	AttributeNationality AttributeType = "NATI"
	// AttributeSSN represents a social security number (SSN) attribute.
	AttributeSSN AttributeType = "SSN"
	// AttributeTitle represents a nobility or other title (TITL) attribute.
	AttributeTitle AttributeType = "TITL"
	// AttributeReligion represents a religious affiliation (RELI) attribute.
	AttributeReligion AttributeType = "RELI"
	// AttributeNumberOfChildren represents a number of children (NCHI) attribute.
	AttributeNumberOfChildren AttributeType = "NCHI"
	// AttributeNumberOfMarriages represents a number of marriages (NMR) attribute.
	AttributeNumberOfMarriages AttributeType = "NMR"
	// AttributeProperty represents a property or possessions (PROP) attribute.
	AttributeProperty AttributeType = "PROP"
	// AttributeFact represents a generic fact (FACT) attribute.
	AttributeFact AttributeType = "FACT"
)

// Attribute represents a personal attribute.
type Attribute struct {
	// Type is the attribute type (e.g., AttributeOccupation, AttributeEducation).
	// See the AttributeType constants.
	Type AttributeType

	// Value is the attribute value, with its CONT/CONC continuation lines
	// folded in (CONT as a newline).
	Value string

	// Date when the attribute was applicable (optional)
	Date string

	// ParsedDate is the parsed representation of Date.
	// This is nil if the date string could not be parsed.
	ParsedDate *Date

	// PlaceDetail is where the attribute was applicable (optional), with
	// optional coordinates. It is the sole place carrier;
	// [Attribute.PlaceName] is the nil-safe read path for just the name.
	PlaceDetail *PlaceDetail

	// TypeDetail is the user-supplied classification of this attribute
	// (TYPE subordinate). It is the Attribute counterpart of
	// Event.EventTypeDetail; Type already holds the tag, so the tag's own
	// TYPE line lands here. Mandatory under IDNO and FACT in 5.5.1, where
	// the attribute has no meaning without it.
	// Per ADR 0003 the raw TYPE tag also remains on Individual.Tags, so
	// callers that hand-walk the raw subtree now have two sources for the
	// same fact — the same dual storage Event.EventTypeDetail already has.
	TypeDetail string

	// Cause is the cause of the attribute (CAUS subordinate)
	Cause string

	// Age is the age at the time the attribute applied (AGE subordinate)
	Age string

	// AgePhrase is the PHRASE under AGE (GEDCOM 7.0), the age in prose.
	// Written only beneath an Age.
	AgePhrase string

	// HusbandAge is the husband's age at the time of the attribute (AGE
	// beneath the HUSB subordinate, FAMILY_EVENT_DETAIL). Only a family's
	// attributes and events carry one: the decoder fills it there alone, and
	// the encoder writes it only beneath a family's.
	HusbandAge string

	// HusbandAgePhrase is the PHRASE under that AGE (GEDCOM 7.0). Written
	// only beneath a HusbandAge.
	HusbandAgePhrase string

	// WifeAge is the wife's age at the time of the attribute (AGE beneath the
	// WIFE subordinate, FAMILY_EVENT_DETAIL), under the same rules as
	// HusbandAge.
	WifeAge string

	// WifeAgePhrase is the PHRASE under that AGE (GEDCOM 7.0). Written only
	// beneath a WifeAge.
	WifeAgePhrase string

	// Agency is the responsible agency (AGNC subordinate)
	Agency string

	// ReligiousAffiliation is the religious denomination associated with the
	// attribute (RELI subordinate)
	ReligiousAffiliation string

	// Address is the attribute address structure (ADDR subordinate)
	Address *Address

	// Phone numbers associated with the attribute (PHON subordinate, can repeat)
	Phone []string

	// Email addresses associated with the attribute (EMAIL subordinate, can repeat)
	Email []string

	// Fax numbers associated with the attribute (FAX subordinate, can repeat)
	Fax []string

	// Websites associated with the attribute (WWW subordinate, can repeat)
	Website []string

	// Restriction notice for privacy controls (RESN subordinate)
	// Common values: "confidential", "locked", "privacy" (or combinations)
	Restriction string

	// UIDs are unique identifiers for the attribute (UID subordinates,
	// repeatable in GEDCOM 7.0), in file order.
	UIDs []string

	// SortDate is the date used for sorting attributes (SDATE subordinate, GEDCOM 7.0)
	// Typically in ISO 8601 format (e.g., "1900-01-01")
	SortDate string

	// SortDatePhrase is the PHRASE under SDATE (GEDCOM 7.0). Written only
	// beneath a SortDate.
	SortDatePhrase string

	// SourceCitations are source citations with page/quality details
	SourceCitations []*SourceCitation

	// Media are references to media objects with optional crop/title
	Media []*MediaLink

	// Associations are links to individuals associated with this attribute
	// (ASSO subordinate), e.g. witnesses, officiants or godparents
	Associations []*Association

	// NoteXRefs are XRef pointers to shared NOTE/SNOTE records (e.g. "@N1@").
	NoteXRefs []string

	// InlineNotes are note text values written directly on this attribute
	// (NOTE <text> form, including CONT/CONC continuations).
	InlineNotes []string
}

// PlaceName returns PlaceDetail.Name, or "" when no place is recorded.
// Safe on a nil receiver and a nil PlaceDetail.
func (a *Attribute) PlaceName() string {
	if a == nil || a.PlaceDetail == nil {
		return ""
	}
	return a.PlaceDetail.Name
}

// SetPlaceName records name as the attribute's place, allocating PlaceDetail if
// the attribute does not have one yet and leaving any existing Form and
// Coordinates intact. Safe on a nil receiver, where it does nothing.
//
// This is the counterpart of [Attribute.PlaceName] and the supported write
// path: assigning through PlaceDetail directly panics on an attribute built
// without one, and leaving PlaceDetail nil means the encoder writes no PLAC
// line at all.
//
// An empty name on an attribute that has no PlaceDetail is a no-op, so threading
// an optional place through this method cannot conjure a placeless PLAC line. To
// clear a place that is already recorded, set PlaceDetail to nil -- calling this
// with an empty name only blanks the name, and a PlaceDetail whose Name is empty
// still encodes as a valueless PLAC line.
func (a *Attribute) SetPlaceName(name string) {
	if a == nil {
		return
	}
	if a.PlaceDetail == nil {
		if name == "" {
			return
		}
		a.PlaceDetail = &PlaceDetail{}
	}
	a.PlaceDetail.Name = name
}

// AllNotes returns this attribute's inline notes followed by the text of any
// shared notes referenced by NoteXRefs, resolved against doc. Shared notes that
// do not resolve are skipped. Returns nil when there are no notes.
func (a *Attribute) AllNotes(doc *Document) []string {
	return allNotes(doc, a.InlineNotes, a.NoteXRefs)
}

// OccurredEvents returns the events recorded as having happened: Events in file
// order, without nil entries and without GEDCOM 7.0 negative assertions
// (IsNegative). It returns nil when there are none, and is safe on a nil
// receiver. The returned slice is new; the events are shared with Events.
func (i *Individual) OccurredEvents() []*Event {
	if i == nil {
		return nil
	}
	return filterEvents(i.Events, false)
}

// NegativeAssertions returns the GEDCOM 7.0 negative assertions (n NO <EVENT>)
// recorded on this individual -- the events with IsNegative set, in file order.
// Each one states that the event did not happen, optionally within its DATE
// period. It returns nil when there are none, and is safe on a nil receiver.
// The returned slice is new; the events are shared with Events.
func (i *Individual) NegativeAssertions() []*Event {
	if i == nil {
		return nil
	}
	return filterEvents(i.Events, true)
}

// BirthEvent returns the first birth event for this individual, or nil if none found.
// A negative assertion (NO BIRT, IsNegative) is not a birth event and is skipped.
func (i *Individual) BirthEvent() *Event {
	if i == nil {
		return nil
	}
	return firstOccurredEvent(i.Events, EventBirth)
}

// DeathEvent returns the first death event for this individual, or nil if none found.
// A negative assertion (NO DEAT, IsNegative) is not a death event and is skipped.
func (i *Individual) DeathEvent() *Event {
	if i == nil {
		return nil
	}
	return firstOccurredEvent(i.Events, EventDeath)
}

// BirthDate returns the parsed birth date for this individual, or nil if no birth event
// or no parsed date is available. It can be a phrase-only date (IsPhrase set, no
// components), which callers such as the validator count as a recorded date.
func (i *Individual) BirthDate() *Date {
	event := i.BirthEvent()
	if event == nil {
		return nil
	}
	return event.ParsedDate
}

// DeathDate returns the parsed death date for this individual, or nil if no death event
// or no parsed date is available. Like BirthDate, it can be a phrase-only date.
func (i *Individual) DeathDate() *Date {
	event := i.DeathEvent()
	if event == nil {
		return nil
	}
	return event.ParsedDate
}

// Name returns the individual's first recorded name, skipping nil entries, or
// nil when there is none. Safe on a nil receiver. Use it instead of indexing
// Names[0], which panics on a nameless individual.
func (i *Individual) Name() *PersonalName {
	if i == nil {
		return nil
	}
	for _, name := range i.Names {
		if name != nil {
			return name
		}
	}
	return nil
}

// FullName returns the Full value of [Individual.Name], or "" when the
// individual has no name. Safe on a nil receiver.
func (i *Individual) FullName() string {
	name := i.Name()
	if name == nil {
		return ""
	}
	return name.Full
}

// BirthPlace returns the place name of [Individual.BirthEvent], or "" when there
// is no birth event or it records no place. Safe on a nil receiver.
func (i *Individual) BirthPlace() string {
	return i.BirthEvent().PlaceName()
}

// DeathPlace returns the place name of [Individual.DeathEvent], or "" when there
// is no death event or it records no place. Safe on a nil receiver.
func (i *Individual) DeathPlace() string {
	return i.DeathEvent().PlaceName()
}

// EventOf returns the first event of the given type, or nil if none found. Like
// BirthEvent, it skips nil entries and negative assertions (IsNegative). Safe on
// a nil receiver.
func (i *Individual) EventOf(eventType EventType) *Event {
	if i == nil {
		return nil
	}
	return firstOccurredEvent(i.Events, eventType)
}

// AttributeOf returns the first attribute of the given type, skipping nil
// entries, or nil if none found. Safe on a nil receiver.
func (i *Individual) AttributeOf(attributeType AttributeType) *Attribute {
	if i == nil {
		return nil
	}
	for _, attr := range i.Attributes {
		if attr != nil && attr.Type == attributeType {
			return attr
		}
	}
	return nil
}

// FamilySearchURL returns the FamilySearch.org URL for this individual's record.
// Returns an empty string if FamilySearchID is not set.
func (i *Individual) FamilySearchURL() string {
	if i.FamilySearchID == "" {
		return ""
	}
	return "https://www.familysearch.org/tree/person/details/" + i.FamilySearchID
}

// Parents returns all parents (biological) of this individual by looking up
// the families where this individual is a child and collecting the husband
// and wife from each family.
//
// The doc parameter is required for O(1) cross-reference lookups.
// Returns an empty slice if doc is nil, no parents are found, or if
// family/individual xrefs are invalid. Invalid xrefs are silently skipped.
// Order is preserved from the GEDCOM file. ParentLinks returns the same
// parents with the family and pedigree that link them.
func (i *Individual) Parents(doc *Document) []*Individual {
	links := i.ParentLinks(doc)
	if links == nil {
		return nil
	}
	parents := make([]*Individual, len(links))
	for n, link := range links {
		parents[n] = link.Individual
	}
	return parents
}

// Spouses returns all spouses of this individual by looking up the families
// where this individual is a spouse and returning the other spouse from each
// family.
//
// The doc parameter is required for O(1) cross-reference lookups.
// Returns an empty slice if doc is nil, no spouses are found, or if
// family/individual xrefs are invalid. Invalid xrefs are silently skipped.
// Handles multiple marriages (multiple FAMS entries).
// Order is preserved from the GEDCOM file.
func (i *Individual) Spouses(doc *Document) []*Individual {
	if doc == nil {
		return nil
	}

	var spouses []*Individual
	for _, link := range i.SpouseInFamilies {
		fam := doc.GetFamily(link.FamilyXRef)
		if fam == nil {
			continue
		}
		// Determine which spouse is "the other one"
		if fam.Husband == i.XRef {
			// This individual is the husband, spouse is wife
			if fam.Wife != "" {
				if wife := doc.GetIndividual(fam.Wife); wife != nil {
					spouses = append(spouses, wife)
				}
			}
		} else {
			// This individual is the wife, spouse is husband
			if fam.Husband != "" {
				if husband := doc.GetIndividual(fam.Husband); husband != nil {
					spouses = append(spouses, husband)
				}
			}
		}
	}
	return spouses
}

// Children returns all children of this individual by looking up the families
// where this individual is a spouse and collecting all children from each family.
//
// The doc parameter is required for O(1) cross-reference lookups.
// Returns an empty slice if doc is nil, no children are found, or if
// family/individual xrefs are invalid. Invalid xrefs are silently skipped.
// Order is preserved from the GEDCOM file.
func (i *Individual) Children(doc *Document) []*Individual {
	if doc == nil {
		return nil
	}

	var children []*Individual
	for _, link := range i.SpouseInFamilies {
		fam := doc.GetFamily(link.FamilyXRef)
		if fam == nil {
			continue
		}
		for _, childXRef := range fam.Children {
			if child := doc.GetIndividual(childXRef); child != nil {
				children = append(children, child)
			}
		}
	}
	return children
}

// FamiliesAsChild returns the families in which this individual is a child --
// the family records containing this individual's parents.
//
// The doc parameter is required for O(1) cross-reference lookups.
// Returns an empty slice if doc is nil, no parental families are found, or if
// family xrefs are invalid. Invalid xrefs are silently skipped.
// Order is preserved from the GEDCOM file.
func (i *Individual) FamiliesAsChild(doc *Document) []*Family {
	if doc == nil {
		return nil
	}

	var families []*Family
	for _, link := range i.ChildInFamilies {
		if fam := doc.GetFamily(link.FamilyXRef); fam != nil {
			families = append(families, fam)
		}
	}
	return families
}

// FamiliesAsSpouse returns the families in which this individual is a spouse --
// the family records where this individual is either the husband or the wife.
//
// The doc parameter is required for O(1) cross-reference lookups.
// Returns an empty slice if doc is nil, no spouse families are found, or if
// family xrefs are invalid. Invalid xrefs are silently skipped.
// Order is preserved from the GEDCOM file.
func (i *Individual) FamiliesAsSpouse(doc *Document) []*Family {
	if doc == nil {
		return nil
	}

	var families []*Family
	for _, link := range i.SpouseInFamilies {
		if fam := doc.GetFamily(link.FamilyXRef); fam != nil {
			families = append(families, fam)
		}
	}
	return families
}

// ParentLink is one parent of an individual together with the family that
// links them and the child's pedigree in that family.
type ParentLink struct {
	// Individual is the parent: the family's husband or wife.
	Individual *Individual

	// Family is the family in which the child is a child (the FAMC target).
	Family *Family

	// Pedigree is the child's FamilyLink.Pedigree for Family ("birth",
	// "adopted", "foster", "sealing", ...), kept as written; empty when the
	// FAMC line carries no PEDI.
	Pedigree string
}

// ParentLinks returns the same parents as Parents, in the same order, each
// with the family that links them and the pedigree of that link, so birth and
// adoptive parents can be told apart.
//
// The doc parameter is required for O(1) cross-reference lookups.
// Returns an empty slice if doc is nil or no parents are found. Invalid
// family or individual xrefs are silently skipped.
func (i *Individual) ParentLinks(doc *Document) []ParentLink {
	if doc == nil {
		return nil
	}

	var links []ParentLink
	for _, link := range i.ChildInFamilies {
		fam := doc.GetFamily(link.FamilyXRef)
		if fam == nil {
			continue
		}
		for _, xref := range []string{fam.Husband, fam.Wife} {
			if xref == "" {
				continue
			}
			if parent := doc.GetIndividual(xref); parent != nil {
				links = append(links, ParentLink{Individual: parent, Family: fam, Pedigree: link.Pedigree})
			}
		}
	}
	return links
}

// Sibling is one sibling of an individual.
type Sibling struct {
	// Individual is the sibling.
	Individual *Individual

	// Family is the family through which they are siblings: for a full
	// sibling, a family both are children of; for a half sibling, the other
	// family of the shared parent, which the sibling is a child of.
	Family *Family

	// Half is true when the sibling is a child of one of the individual's
	// parents in a family the individual is not a child of, such as a child
	// of a parent's other marriage.
	Half bool
}

// Siblings returns this individual's siblings: first the other children of
// each family the individual is a child of (full siblings, Half false), then
// the children of each parent's other families (half siblings, Half true).
// Each sibling appears once, as full if it qualifies as both; the individual
// itself is never included.
//
// Full and half are read from the family records, not from pedigree: children
// of the same family are full siblings even when one is adopted. Use
// ParentLinks on each to compare pedigrees.
//
// The doc parameter is required for O(1) cross-reference lookups.
// Returns an empty slice if doc is nil or no siblings are found. Invalid
// family or individual xrefs are silently skipped. Order follows the GEDCOM
// file: families in FAMC order, parents as Parents returns them, then each
// parent's families in FAMS order.
func (i *Individual) Siblings(doc *Document) []Sibling {
	if doc == nil {
		return nil
	}

	seen := map[string]bool{i.XRef: true}
	var siblings []Sibling
	addChildren := func(fam *Family, half bool) {
		for _, xref := range fam.Children {
			if seen[xref] {
				continue
			}
			if child := doc.GetIndividual(xref); child != nil {
				seen[xref] = true
				siblings = append(siblings, Sibling{Individual: child, Family: fam, Half: half})
			}
		}
	}

	ownFamilies := i.FamiliesAsChild(doc)
	isOwn := make(map[*Family]bool, len(ownFamilies))
	for _, fam := range ownFamilies {
		isOwn[fam] = true
		addChildren(fam, false)
	}
	for _, parent := range i.Parents(doc) {
		for _, fam := range parent.FamiliesAsSpouse(doc) {
			if !isOwn[fam] {
				addChildren(fam, true)
			}
		}
	}
	return siblings
}
