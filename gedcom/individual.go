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

	// Place where the attribute was applicable (optional)
	Place string

	// SourceCitations are source citations with page/quality details
	SourceCitations []*SourceCitation
}
