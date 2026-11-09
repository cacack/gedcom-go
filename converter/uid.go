package converter

import (
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// transformUIDTags rewrites unique identifiers between their GEDCOM 7.0
// spelling, UID, and the vendor spelling GEDCOM 5.5/5.5.1 producers use for
// the same identifier, _UID: on an upgrade to 7.0 every _UID becomes UID, and
// on a downgrade every UID becomes _UID (#715). _UID is the most common
// extension tag in real 5.5.x exports (MyHeritage, PAF, RootsMagic, Legacy,
// Family Historian all write it) and is exactly what 7.0 standardised as UID,
// so the identifier survives conversion in both directions instead of being
// dropped on a downgrade or left as an extension on an upgrade.
//
// The rename applies at every level: a 7.0 UID under an event becomes a
// 5.5.x _UID under the event, where the vendors that write it put it. The
// typed UIDs fields are left as they are, since the decoder fills them from
// either spelling and the encoder writes them under the target's spelling.
// Each rename is recorded as a normalized note, plus one aggregate
// transformation entry.
func transformUIDTags(doc *gedcom.Document, report *gedcom.ConversionReport, targetVersion gedcom.Version) {
	from, to := "UID", "_UID"
	if targetVersion == gedcom.Version70 {
		from, to = "_UID", "UID"
	}
	total := 0
	for _, record := range doc.Records {
		if record == nil {
			continue
		}
		n := 0
		for _, tag := range record.Tags {
			if tag == nil || tag.Tag != from {
				continue
			}
			tag.Tag = to
			report.AddNormalized(gedcom.ConversionNote{
				Path:     BuildNestedPath(string(record.Type), record.XRef, to),
				Original: from,
				Result:   to,
				Reason:   "Unique identifier written as " + to + " for GEDCOM " + targetVersion.String(),
			})
			n++
		}
		total += n
	}
	if total > 0 {
		typ := "UID_TO_VENDOR_TAG"
		if to == "UID" {
			typ = "UID_FROM_VENDOR_TAG"
		}
		report.AddTransformation(gedcom.Transformation{
			Type:        typ,
			Description: "Rewrote " + from + " as " + to + " for GEDCOM " + targetVersion.String(),
			Count:       total,
		})
	}
}
