// Package converter provides GEDCOM version conversion functionality.
//
// This package converts GEDCOM documents between versions 5.5, 5.5.1, and 7.0.
// It handles all necessary transformations including encoding changes, tag
// modifications, and XRef normalization.
//
// Basic usage:
//
//	doc, _ := decoder.Decode(reader)
//	converted, report, err := converter.Convert(doc, gedcom.Version70)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(report)
//
// The converter returns a ConversionReport detailing all transformations
// and any data loss that occurred during conversion.
package converter
