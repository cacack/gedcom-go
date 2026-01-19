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
package decoder
