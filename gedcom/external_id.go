package gedcom

// ExternalID represents an external identifier (GEDCOM 7.0 EXID tag).
// Links a record to an external system like FamilySearch, Ancestry, etc.
//
// GEDCOM 7.0 structure:
//
//	n EXID <Text>
//	  +1 TYPE <URI>
//
// Example:
//
//	1 EXID 9876543210
//	  2 TYPE http://www.familysearch.org/ark
type ExternalID struct {
	// Value is the external identifier string
	Value string

	// Type is the URI identifying the external system (from TYPE subordinate)
	Type string
}
