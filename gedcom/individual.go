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
	ChildInFamilies []string // XRef to Family records

	// SpouseInFamilies are references to families where this person is a spouse
	SpouseInFamilies []string // XRef to Family records

	// Sources are references to source citations
	Sources []string // XRef to Source records

	// Notes are references to note records
	Notes []string // XRef to Note records

	// MediaRefs are references to media objects
	MediaRefs []string // XRef to MediaObject records

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

	// Type is the name type (e.g., "birth", "married", "aka")
	Type string
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

	// Sources are references to source citations
	Sources []string
}
