package converter

import (
	"strings"

	"github.com/cacack/gedcom-go/gedcom"
)

// normalizeXRefsToUppercase converts all XRefs to uppercase for GEDCOM 7.0.
// GEDCOM 7.0 requires XRefs to be uppercase only (@[A-Z0-9_]+@).
func normalizeXRefsToUppercase(doc *gedcom.Document, report *gedcom.ConversionReport) {
	mapping := buildXRefMapping(doc)
	if len(mapping) == 0 {
		return
	}

	for _, record := range doc.Records {
		if newXRef, ok := mapping[record.XRef]; ok {
			path := BuildRecordPath(string(record.Type), record.XRef)
			report.AddNormalized(gedcom.ConversionNote{
				Path:     path,
				Original: record.XRef,
				Result:   newXRef,
				Reason:   "GEDCOM 7.0 requires XRefs to be uppercase only (@[A-Z0-9_]+@)",
			})
		}
	}

	gedcom.Apply(doc, mapping)

	report.AddTransformation(gedcom.Transformation{
		Type:        "XREF_UPPERCASE",
		Description: "Normalized XRefs to uppercase (required for GEDCOM 7.0)",
		Count:       len(mapping),
	})
}

// buildXRefMapping creates a map of original XRefs to their uppercase versions.
// Only XRefs that differ from their uppercase form are included.
func buildXRefMapping(doc *gedcom.Document) map[string]string {
	mapping := make(map[string]string)

	for _, record := range doc.Records {
		if record.XRef == "" {
			continue
		}
		upper := strings.ToUpper(record.XRef)
		if record.XRef != upper {
			mapping[record.XRef] = upper
		}
	}

	return mapping
}
