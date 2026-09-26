package gedcom

// RefNumber represents a user reference number (REFN tag) with its optional
// TYPE subordinate. REFN is repeatable on INDI, FAM, SOUR and OBJE records in
// both GEDCOM 5.5.1 and 7.0, so records carry a []RefNumber in file order.
//
// GEDCOM structure:
//
//	n REFN <Special>
//	  +1 TYPE <Text>
//
// Example:
//
//	1 REFN 1234
//	  2 TYPE Ancestral File
type RefNumber struct {
	// Value is the user reference number string.
	Value string

	// Type is the user-defined reference type (from the TYPE subordinate).
	// Empty when no TYPE was given.
	Type string
}
