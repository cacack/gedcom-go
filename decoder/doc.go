// Package decoder provides high-level GEDCOM file decoding functionality.
//
// The decoder package converts GEDCOM files into structured Go data types,
// building on the lower-level parser package. It handles character encoding,
// validates the GEDCOM structure, and constructs a complete Document with
// cross-reference resolution.
//
// Example usage:
//
//	f, err := os.Open("family.ged")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer f.Close()
//
//	doc, err := decoder.Decode(f)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Found %d individuals\n", len(doc.Individuals()))
//
// # Overlap with the validator
//
// [DecodeResult.Diagnostics] and the validator package's findings
// (validator.Validator.ValidateAll, QualityReport and StreamingValidator) are
// not disjoint: a finding can appear in both. The only code for which it can
// is DUPLICATE_XREF ([CodeDuplicateXRef] here, validator.CodeDuplicateXRef
// there), each reporting a redefined XRef as an error on the line of the
// redefinition. Every other code is reported by one package only. A caller
// that reports both lists counts a duplicated XRef twice unless it passes the
// validator's issues through gedcomgo.WithoutDiagnosed, which drops the ones
// the diagnostics already hold.
package decoder
