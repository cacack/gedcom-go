package converter

import (
	"slices"
	"strings"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// promoteInlineSourceCitations turns every inline source citation -- a SOUR
// substructure whose payload is a source description rather than a pointer
// ("1 SOUR Family bible", GEDCOM 5.5/5.5.1) -- into a pointer to a new SOUR
// record on a 5.5/5.5.1 -> 7.0 upgrade (issues #738, #793). GEDCOM 7.0 has
// only the pointer form; the gedcom7code test suite (sour-1.ged) gives this
// conversion:
//
//	1 SOUR Family bible        1 SOUR @S1@
//	2 TEXT Born 1850     ->    2 DATA
//	2 QUAY 1                   3 TEXT Born 1850
//	                           2 QUAY 1
//	                           0 @S1@ SOUR
//	                           1 NOTE Family bible
//
// The description becomes the new record's NOTE, and each TEXT of the
// citation moves, in order, under the citation's DATA (added where the first
// TEXT was, or the existing one); everything else stays on the citation. The
// new record's XRef is the first free @S<n>@, and it goes just before the
// record that cites it, where the suite puts it. A citation with an empty payload
// ("1 SOUR", as Ancestry writes to carry an _APID) cites no source at all, so
// it points to @VOID@, the 7.0 null pointer, instead of a new empty record.
// Each rewrite is a Normalized note, with one aggregate transformation for
// each kind.
//
// It runs after transformTextForVersion, so the description's CONT/CONC lines
// are already folded into its value, and after normalizeXRefsForV7, so the
// XRefs it checks for collisions are final and every pointer passes
// Document.IsPointerXRef.
func promoteInlineSourceCitations(doc *gedcom.Document, report *gedcom.ConversionReport) {
	used := xrefsInUse(doc)
	next, added, voided := 1, 0, 0
	records := make([]*gedcom.Record, 0, len(doc.Records))
	for _, record := range doc.Records {
		if record == nil {
			records = append(records, record)
			continue
		}
		for i := 0; i < len(record.Tags); i++ {
			tag := record.Tags[i]
			if tag == nil || tag.Tag != "SOUR" || tag.Level < 1 || isSourcePointer(doc, tag.Value) {
				continue
			}
			end := blockEnd(record.Tags, i)
			block := citationTextToData(record.Tags[i+1:end], tag.Level)
			original, reason := "SOUR (inline source citation)",
				"GEDCOM 7.0 has no inline source citation; its description moved to a new SOUR record"
			xref := v7NullXRef
			if strings.TrimSpace(tag.Value) == "" {
				original, reason = "SOUR (empty)",
					"GEDCOM 7.0 requires a pointer; a citation of no source record points to @VOID@"
				voided++
			} else {
				xref, next = freeXRef(used, "S", next)
				used[xref] = true
				promoted := &gedcom.Record{
					XRef: xref,
					Type: gedcom.RecordTypeSource,
					Tags: []*gedcom.Tag{{Level: 1, Tag: "NOTE", Value: tag.Value}},
				}
				records = append(records, promoted)
				added++
				if doc.XRefMap == nil {
					doc.XRefMap = map[string]*gedcom.Record{}
				}
				doc.XRefMap[xref] = promoted
			}
			tag.Value = xref
			record.Tags = slices.Concat(record.Tags[:i+1], block, record.Tags[end:])
			report.AddNormalized(gedcom.ConversionNote{
				Path:     BuildNestedPath(string(record.Type), record.XRef, tagPath(record.Tags, i)...),
				Original: original,
				Result:   "SOUR " + xref,
				Reason:   reason,
			})
		}
		records = append(records, record)
	}
	if voided > 0 {
		report.AddTransformation(gedcom.Transformation{
			Type:        "EMPTY_SOUR_TO_VOID",
			Description: "Pointed empty source citations at @VOID@",
			Count:       voided,
		})
	}
	if added == 0 {
		return
	}
	doc.Records = records
	report.AddTransformation(gedcom.Transformation{
		Type:        "INLINE_SOUR_TO_RECORD",
		Description: "Converted inline source citations to SOUR records",
		Count:       added,
	})
}

// isSourcePointer reports whether a SOUR citation's value is a pointer: one
// Document.IsPointerXRef accepts, or the GEDCOM 7.0 null pointer @VOID@.
func isSourcePointer(doc *gedcom.Document, value string) bool {
	v := strings.TrimSpace(value)
	return v == v7NullXRef || doc.IsPointerXRef(v)
}

// citationTextToData returns the subordinate block of an inline citation at
// citeLevel with each direct TEXT (and its subordinates) moved one level down
// under DATA: the citation's own DATA when it has one, else a new DATA where
// the first TEXT was. A block with no TEXT is returned as it is; otherwise
// nil tags are dropped.
func citationTextToData(block []*gedcom.Tag, citeLevel int) []*gedcom.Tag {
	var texts []*gedcom.Tag
	rest := make([]*gedcom.Tag, 0, len(block))
	firstText, dataEnd := -1, -1
	for i := 0; i < len(block); i++ {
		tag := block[i]
		if tag == nil {
			continue
		}
		end := blockEnd(block, i)
		switch {
		case tag.Level != citeLevel+1:
			rest = append(rest, block[i:end]...)
		case tag.Tag == "TEXT":
			if firstText < 0 {
				firstText = len(rest)
			}
			texts = append(texts, relevel(block[i:end], -1)...)
		case tag.Tag == "DATA" && dataEnd < 0:
			rest = append(rest, block[i:end]...)
			dataEnd = len(rest)
		default:
			rest = append(rest, block[i:end]...)
		}
		i = end - 1
	}
	if len(texts) == 0 {
		return block
	}
	if dataEnd >= 0 {
		return slices.Concat(rest[:dataEnd], texts, rest[dataEnd:])
	}
	data := &gedcom.Tag{Level: citeLevel + 1, Tag: "DATA"}
	return slices.Concat(rest[:firstText], []*gedcom.Tag{data}, texts, rest[firstText:])
}
