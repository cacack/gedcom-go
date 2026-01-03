package gedcom

// Individual represents a person in the GEDCOM file.
type Individual struct {
	// XRef is the cross-reference identifier for this individual
	XRef string

	// Names contains all name variants for this person
	Names []*PersonalName

	// Sex is the person's sex (M, F, X, U for unknown)
	Sex string

	// Events contains life events (birth, death, marriage, etc.)
	Events []*Event

	// Attributes contains personal attributes (occupation, education, etc.)
	Attributes []*Attribute

	// ChildInFamilies are references to families where this person is a child
	ChildInFamilies []FamilyLink

	// SpouseInFamilies are references to families where this person is a spouse
	SpouseInFamilies []string // XRef to Family records

	// Associations are links to associated individuals (godparents, witnesses, etc.)
	Associations []*Association

	// SourceCitations are source citations with page/quality details
	SourceCitations []*SourceCitation

	// Notes are references to note records
	Notes []string // XRef to Note records

	// Media are references to media objects with optional crop/title
	Media []*MediaLink

	// LDSOrdinances are LDS (Latter-Day Saints) ordinances (BAPL, CONL, ENDL, SLGC)
	LDSOrdinances []*LDSOrdinance

	// ChangeDate is when the record was last modified (CHAN tag)
	ChangeDate *ChangeDate

	// CreationDate is when the record was created (CREA tag, GEDCOM 7.0)
	CreationDate *ChangeDate

	// RefNumber is the user reference number (REFN tag)
	RefNumber string

	// UID is the unique identifier (UID tag)
	UID string

	// FamilySearchID is the FamilySearch Family Tree ID (_FSFTID tag).
	// This is a vendor extension from FamilySearch.org that uniquely identifies
	// an individual in their Family Tree database. Format: alphanumeric like "KWCJ-QN7".
	FamilySearchID string

	// Tags contains all raw tags for this individual (for unknown/custom tags)
	Tags []*Tag
}

// PersonalName represents a person's name with optional components.
type PersonalName struct {
	// Full is the full name (e.g., "John /Doe/")
	Full string

	// Given is the given (first) name
	Given string

	// Surname is the family name
	Surname string

	// Prefix is the name prefix (e.g., "Dr.", "Sir")
	Prefix string

	// Suffix is the name suffix (e.g., "Jr.", "III")
	Suffix string

	// Nickname is the person's nickname (e.g., "Bill" for William)
	Nickname string

	// SurnamePrefix is the surname prefix (e.g., "von", "de", "van der")
	SurnamePrefix string

	// Type is the name type (e.g., "birth", "married", "aka")
	Type string

	// Transliterations are alternative representations of the name in different
	// writing systems or scripts (GEDCOM 7.0 TRAN tag). Used to store the same
	// name in different languages, scripts, or romanization systems.
	Transliterations []*Transliteration
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

// FamilyLink represents a link to a family with optional pedigree type.
type FamilyLink struct {
	// FamilyXRef is the cross-reference to the family record
	FamilyXRef string

	// Pedigree is the pedigree linkage type (e.g., "birth", "adopted", "foster", "sealing")
	// Empty string if not specified. Preserves original casing from GEDCOM.
	Pedigree string
}

// Association represents a link to an associated individual with a role.
// Used for relationships like godparents (GODP), witnesses (WITN), etc.
type Association struct {
	// IndividualXRef is the cross-reference to the associated individual
	IndividualXRef string

	// Role is the relationship role (e.g., "GODP" for godparent, "WITN" for witness)
	// In GEDCOM 5.5.1 this comes from RELA tag, in GEDCOM 7.0 from ROLE tag
	Role string

	// Phrase is a human-readable description of the association (GEDCOM 7.0 PHRASE tag).
	// Used when the structured data cannot fully express the relationship.
	// Example: "Mr Stockdale" as the associated person's name when @XREF@ is unavailable.
	Phrase string

	// SourceCitations are source citations documenting this association (GEDCOM 7.0).
	// Allows citing sources for the association relationship itself.
	SourceCitations []*SourceCitation

	// Notes are note references for this association
	Notes []string
}

// Attribute represents a personal attribute.
type Attribute struct {
	// Type is the attribute type (e.g., "OCCU" for occupation, "EDUC" for education)
	Type string

	// Value is the attribute value
	Value string

	// Date when the attribute was applicable (optional)
	Date string

	// ParsedDate is the parsed representation of Date.
	// This is nil if the date string could not be parsed.
	ParsedDate *Date

	// Place where the attribute was applicable (optional)
	Place string

	// SourceCitations are source citations with page/quality details
	SourceCitations []*SourceCitation
}

// BirthEvent returns the first birth event for this individual, or nil if none found.
func (i *Individual) BirthEvent() *Event {
	for _, event := range i.Events {
		if event.Type == EventBirth {
			return event
		}
	}
	return nil
}

// DeathEvent returns the first death event for this individual, or nil if none found.
func (i *Individual) DeathEvent() *Event {
	for _, event := range i.Events {
		if event.Type == EventDeath {
			return event
		}
	}
	return nil
}

// BirthDate returns the parsed birth date for this individual, or nil if no birth event
// or no parsed date is available.
func (i *Individual) BirthDate() *Date {
	event := i.BirthEvent()
	if event == nil {
		return nil
	}
	return event.ParsedDate
}

// DeathDate returns the parsed death date for this individual, or nil if no death event
// or no parsed date is available.
func (i *Individual) DeathDate() *Date {
	event := i.DeathEvent()
	if event == nil {
		return nil
	}
	return event.ParsedDate
}

// FamilySearchURL returns the FamilySearch.org URL for this individual's record.
// Returns an empty string if FamilySearchID is not set.
func (i *Individual) FamilySearchURL() string {
	if i.FamilySearchID == "" {
		return ""
	}
	return "https://www.familysearch.org/tree/person/details/" + i.FamilySearchID
}
