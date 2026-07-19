package converter

import (
	"strconv"
	"strings"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// familySearchArkType is the substring that identifies a FamilySearch ARK in an
// EXID's TYPE URI (e.g. "http://www.familysearch.org/ark"). Matching is
// case-insensitive; the "ark" token must be a whole path segment (end of
// string, or followed by ":" or "/") so an unrelated URI such as
// ".../arkansas-records" does not match.
const familySearchArkType = "familysearch.org/ark"

// transformEXIDToVendorTags rewrites GEDCOM 7.0 EXID structures that have a
// faithful vendor-tag equivalent in the older target version, so the identifier
// survives a downgrade instead of being dropped by record70DataLoss.
//
// EXID is a GEDCOM 7.0-only structure. On a 7.0 -> 5.5/5.5.1 downgrade it is
// otherwise dropped and reported as data loss. The only vendor tag with a
// well-established, semantically-correct meaning for an EXID is FamilySearch's
// _FSFTID (a FamilySearch Family Tree person ID), so this maps a FamilySearch
// ARK EXID on an individual to _FSFTID:
//
//	1 EXID KWCJ-QN7
//	  2 TYPE https://www.familysearch.org/ark   ->   1 _FSFTID KWCJ-QN7
//
// Scope and caveats:
//   - Restricted to individual records: _FSFTID has no defined meaning outside
//     INDI, so a FamilySearch ARK EXID on a FAM/SOUR/REPO record is left for the
//     data-loss sweep.
//   - Only a clean EXID whose sole subordinate is the matching TYPE is
//     converted. An EXID carrying other subordinates (a non-ARK TYPE, NOTE,
//     SOUR, ...) is left untouched so those are not silently discarded — they
//     continue to be reported as data loss.
//   - One-way: this is a downgrade-only mapping. There is no inverse transform
//     on the 5.5.x -> 7.0 upgrade path, so a round trip does not restore the
//     original EXID structure (the identifier persists as _FSFTID).
//   - Leading-@ escaping: an identifier beginning with "@" (e.g. the
//     pointer-shaped "@I2@") is escaped to "@@..." in the emitted _FSFTID so it
//     is not mistaken for an XRef pointer. The decoder unescapes it back into
//     Individual.FamilySearchID (see gedcom.EscapeLeadingAt/UnescapeLeadingAt),
//     but the raw tag value on disk retains the escaped "@@" form.
//   - Gated by the caller: convert70To55/convert70To551 only invoke this when
//     ConvertOptions.PreserveUnknownTags is set, since _FSFTID is itself a
//     vendor extension; a caller opting out of vendor tags keeps the plain
//     EXID-is-data-loss behavior.
//
// It must run before record70DataLoss so a converted EXID is not also counted
// as dropped. Each conversion is recorded as a normalized note, plus a single
// aggregate transformation entry.
func transformEXIDToVendorTags(doc *gedcom.Document, report *gedcom.ConversionReport, targetVersion gedcom.Version) {
	total := 0
	for _, record := range doc.Records {
		if record.Type != gedcom.RecordTypeIndividual {
			continue
		}
		rewritten, n := rewriteEXIDTags(record, report, targetVersion)
		if n > 0 {
			record.Tags = rewritten
			total += n
		}
	}
	if total > 0 {
		report.AddTransformation(gedcom.Transformation{
			Type:        "EXID_TO_VENDOR_TAG",
			Description: "Mapped FamilySearch ARK EXID to _FSFTID for GEDCOM " + targetVersion.String(),
			Count:       total,
		})
	}
}

// rewriteEXIDTags returns record.Tags with each convertible FamilySearch-ARK
// EXID (together with its subordinate TYPE) replaced by a single level-1
// _FSFTID tag, and the number of conversions made. It leaves record.Tags
// untouched (and allocates nothing) when the record carries no level-1 EXID.
func rewriteEXIDTags(record *gedcom.Record, report *gedcom.ConversionReport, targetVersion gedcom.Version) (rewritten []*gedcom.Tag, converted int) {
	tags := record.Tags
	if !hasLevel1EXID(tags) {
		return tags, 0
	}

	out := make([]*gedcom.Tag, 0, len(tags))
	count := 0
	for i := 0; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level != 1 || tag.Tag != "EXID" {
			out = append(out, tag)
			continue
		}

		// Delimit this EXID's subordinate block (everything deeper than it,
		// until the next tag at the same or a shallower level).
		j := i + 1
		for j < len(tags) && tags[j].Level > tag.Level {
			j++
		}

		if isConvertibleFamilySearchEXID(tags[i:j]) {
			// A leading "@" in the identifier (e.g. the pointer-shaped "@I2@")
			// copied verbatim would be mistaken for an XRef pointer by the XRef
			// walk and by strict parsers. Escape it per the GEDCOM spec (@@...)
			// so it reads as literal data; the decoder unescapes it back into
			// Individual.FamilySearchID.
			val := gedcom.EscapeLeadingAt(tag.Value)
			out = append(out, &gedcom.Tag{Level: 1, Tag: "_FSFTID", Value: val})
			report.AddNormalized(gedcom.ConversionNote{
				Path:     BuildNestedPath(string(record.Type), record.XRef, exidPathSegment(count)),
				Original: "EXID",
				Result:   "_FSFTID",
				Reason:   "FamilySearch ARK external identifier mapped to _FSFTID for GEDCOM " + targetVersion.String(),
			})
			count++
		} else {
			out = append(out, tags[i:j]...)
		}
		i = j - 1
	}
	return out, count
}

// hasLevel1EXID reports whether the tags include a top-level EXID, used as a
// cheap presence check so rewriteEXIDTags allocates only when there is work.
func hasLevel1EXID(tags []*gedcom.Tag) bool {
	for _, t := range tags {
		if t.Level == 1 && t.Tag == "EXID" {
			return true
		}
	}
	return false
}

// isConvertibleFamilySearchEXID reports whether an EXID block (block[0] is the
// EXID; block[1:] its subordinates) is a clean FamilySearch ARK identifier: its
// only direct subordinate is a TYPE naming a FamilySearch ARK. If the EXID has
// any other direct subordinate, converting it to a single _FSFTID would drop
// that content silently, so it is not convertible.
func isConvertibleFamilySearchEXID(block []*gedcom.Tag) bool {
	exidLevel := block[0].Level
	hasArkType := false
	for _, t := range block[1:] {
		// Only a plain EXID whose sole subordinate is the matching TYPE is
		// convertible. Any other direct subordinate (a non-ARK TYPE, NOTE,
		// SOUR, ...) or any deeper tag (e.g. a NOTE nested under the TYPE)
		// would be lost when the block collapses to a single _FSFTID, so leave
		// the EXID for the data-loss sweep instead.
		if t.Level == exidLevel+1 && t.Tag == "TYPE" && matchesFamilySearchArk(t.Value) {
			hasArkType = true
			continue
		}
		return false
	}
	return hasArkType
}

// matchesFamilySearchArk reports whether a TYPE URI names a FamilySearch ARK,
// requiring "ark" to be a whole path segment so ".../arkansas" does not match.
func matchesFamilySearchArk(typeURI string) bool {
	lower := strings.ToLower(typeURI)
	idx := strings.Index(lower, familySearchArkType)
	if idx < 0 {
		return false
	}
	rest := lower[idx+len(familySearchArkType):]
	return rest == "" || rest[0] == ':' || rest[0] == '/'
}

// exidPathSegment names the EXID element in a conversion-note path, indexing it
// when a record has more than one converted EXID so the notes stay distinct.
func exidPathSegment(index int) string {
	if index == 0 {
		return "EXID"
	}
	return "EXID[" + strconv.Itoa(index) + "]"
}
