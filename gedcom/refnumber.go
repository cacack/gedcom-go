package gedcom

// RefNumber represents a user reference number (REFN tag) with its optional
// TYPE subordinate. REFN is repeatable on INDI, FAM, SOUR, OBJE and REPO
// records in both GEDCOM 5.5.1 and 7.0, on the 5.5.1 NOTE record, and on the
// 7.0 SUBM and SNOTE records, so records carry a []RefNumber in file order.
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
