package converter

import (
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/internal/atsign"
)

// atSignReescapeReason is the Normalized note reason for a raw value whose
// "@" escaping was rewritten for the target version.
const atSignReescapeReason = "GEDCOM 5.x doubles every \"@\" in text and GEDCOM 7.0 only a leading one; the value is escaped for the target version"

// reescapeRawText rewrites the "@" escaping of every raw text value -- each
// header and record Tag value and each record's level-0 value -- from the
// source version's rule to the target's (atsign.Reescape, issue #773), so the
// value holds the same text when read as the target version. A 5.5.1 "a@@b"
// becomes the 7.0 "a@b" and the reverse; a leading "@@" stays doubled.
//
// Pointer values are not text and are left alone, under either version's
// identifier grammar, as are tag names and XRefs. Each rewritten value is
// reported as a Normalized note and counted in the AT_SIGN_REESCAPED
// transformation.
//
// It must run once, on values still in source wire form: on an upgrade to 7.0
// after the CONC/CONT fold (7.0 escapes an "@" after a line break), the date
// phrase split (a 5.x phrase moved to a PHRASE is then escaped as one) and
// every pass that cuts or splices a value (a 7.0 "@" changes meaning when it
// moves to or from the start of a line); on a downgrade before both (the fold of a PHRASE into its DATE takes its 5.x
// form, and an "@@" is never split across CONC lines).
func reescapeRawText(doc *gedcom.Document, from, to gedcom.Version, report *gedcom.ConversionReport) {
	count := 0
	reescape := func(value *string, path string) {
		if isPointerValue(*value) {
			return
		}
		result := atsign.Reescape(*value, from, to)
		if result == *value {
			return
		}
		report.AddNormalized(gedcom.ConversionNote{
			Path:     path,
			Original: *value,
			Result:   result,
			Reason:   atSignReescapeReason,
		})
		*value = result
		count++
	}
	reescapeTags := func(recordType, xref string, tags []*gedcom.Tag) {
		for _, tag := range tags {
			if tag != nil {
				reescape(&tag.Value, BuildNestedPath(recordType, xref, tag.Tag))
			}
		}
	}

	if doc.Header != nil {
		reescapeTags("HEAD", "", doc.Header.Tags)
	}
	for _, record := range doc.Records {
		if record == nil {
			continue
		}
		reescape(&record.Value, BuildRecordPath(string(record.Type), record.XRef))
		reescapeTags(string(record.Type), record.XRef, record.Tags)
	}

	if count > 0 {
		report.AddTransformation(gedcom.Transformation{
			Type:        "AT_SIGN_REESCAPED",
			Description: "Rewrote \"@\" escaping of text values for the target version",
			Count:       count,
		})
	}
}

// isPointerValue reports whether value is a pointer under either version's
// identifier grammar (the 5.5.1 one also allows a space), or the 7.0 null
// pointer "@VOID@". A value 7.0 does not read as a pointer, such as a dangling
// "@I 1@" the XRef normalization leaves unchanged, is still left alone here.
func isPointerValue(value string) bool {
	return value == "@VOID@" || gedcom.IsPointerXRefForVersion(value, gedcom.Version551)
}
