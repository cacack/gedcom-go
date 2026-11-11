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
//
// The conversion rewrites each record's Tags, the authoritative form, and then
// re-derives every tagged record's typed Entity from them by the target version
// (decoder.SyncEntities), so the converted document's two views agree. That
// re-parse costs about as much as decoding the records. A record with no Tags
// (a hand-built one) keeps the Entity the conversion leaves it.
package converter
