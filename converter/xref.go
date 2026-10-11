package converter

import (
	"strconv"
	"strings"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// v7XRefReason explains every XRef rewrite in the conversion report.
const v7XRefReason = "GEDCOM 7.0 requires XRefs to use only uppercase letters, digits and underscore (@[A-Z0-9_]+@)"

// normalizeXRefsForV7 rewrites every XRef that is not a valid GEDCOM 7.0
// Xref (@[A-Z0-9_]+@) and every pointer to it. GEDCOM 5.5 and 5.5.1 allow
// lowercase letters, spaces and other characters that 7.0 rejects.
//
// A record's XRef is uppercased with each other invalid character replaced by
// "_"; if that identifier is already taken, a numeric suffix ("_2", "_3", ...)
// keeps it unique. A pointer to an undefined XRef gets the same uppercase and
// underscore form without a suffix, so a reference differing only in case
// from a record's XRef (@test@ for @TEST@) resolves to that record. Each
// rewrite is recorded as a Normalized note.
func normalizeXRefsForV7(doc *gedcom.Document, report *gedcom.ConversionReport) {
	mapping := buildXRefMapping(doc)
	if len(mapping) == 0 {
		return
	}

	for _, record := range doc.Records {
		if record == nil || record.XRef == "" {
			continue
		}
		if newXRef, ok := mapping[record.XRef]; ok {
			path := BuildRecordPath(string(record.Type), record.XRef)
			report.AddNormalized(gedcom.ConversionNote{
				Path:     path,
				Original: record.XRef,
				Result:   newXRef,
				Reason:   v7XRefReason,
			})
		}
	}
	for _, ref := range danglingXRefs(doc) {
		if newRef, ok := mapping[ref]; ok {
			report.AddNormalized(gedcom.ConversionNote{
				Path:     "Pointer " + ref,
				Original: ref,
				Result:   newRef,
				Reason:   v7XRefReason + "; no record defines this XRef",
			})
		}
	}

	gedcom.Apply(doc, mapping)

	report.AddTransformation(gedcom.Transformation{
		Type:        "XREF_UPPERCASE",
		Description: "Normalized XRefs to uppercase letters, digits and underscore (required for GEDCOM 7.0)",
		Count:       len(mapping),
	})
}

// buildXRefMapping maps each XRef that is not a valid GEDCOM 7.0 Xref to a
// valid one, as described on normalizeXRefsForV7. Valid XRefs are not
// included. Records are visited in document order, so the result is
// deterministic.
func buildXRefMapping(doc *gedcom.Document) map[string]string {
	mapping := make(map[string]string)

	// Valid identifiers keep their name, so a rewritten one must avoid them.
	taken := make(map[string]bool)
	for _, record := range doc.Records {
		if record != nil && isV7XRef(record.XRef) {
			taken[record.XRef] = true
		}
	}

	for _, record := range doc.Records {
		if record == nil || record.XRef == "" || taken[record.XRef] {
			continue
		}
		if _, done := mapping[record.XRef]; done {
			continue
		}
		base := toV7XRefBody(record.XRef)
		candidate := "@" + base + "@"
		for n := 2; taken[candidate]; n++ {
			candidate = "@" + base + "_" + strconv.Itoa(n) + "@"
		}
		taken[candidate] = true
		mapping[record.XRef] = candidate
	}

	for _, ref := range danglingXRefs(doc) {
		mapping[ref] = "@" + toV7XRefBody(ref) + "@"
	}

	return mapping
}

// danglingXRefs returns, in document order and without duplicates, each
// pointer that is not a valid GEDCOM 7.0 Xref and that no record defines.
// Pointers are found with gedcom.Visit, which does not yet follow spaced
// pointers (issue #591), plus the header's submitter.
func danglingXRefs(doc *gedcom.Document) []string {
	var refs []string
	seen := make(map[string]bool) // defined XRefs and refs already added
	for _, record := range doc.Records {
		if record != nil {
			seen[record.XRef] = true
		}
	}
	add := func(ref string) {
		if ref == "" || seen[ref] || isV7XRef(ref) {
			return
		}
		seen[ref] = true
		refs = append(refs, ref)
	}
	if doc.Header != nil {
		add(doc.Header.Submitter)
	}
	for _, record := range doc.Records {
		gedcom.Visit(record, add)
	}
	return refs
}

// isV7XRef reports whether xref matches the GEDCOM 7.0 Xref grammar
// @[A-Z0-9_]+@.
func isV7XRef(xref string) bool {
	if len(xref) < 3 || xref[0] != '@' || xref[len(xref)-1] != '@' {
		return false
	}
	for _, c := range xref[1 : len(xref)-1] {
		if !isV7XRefChar(c) {
			return false
		}
	}
	return true
}

func isV7XRefChar(c rune) bool {
	return (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// toV7XRefBody uppercases the identifier between xref's @ delimiters and
// replaces each character GEDCOM 7.0 does not allow with "_".
func toV7XRefBody(xref string) string {
	body := strings.ToUpper(strings.TrimSuffix(strings.TrimPrefix(xref, "@"), "@"))
	if body == "" {
		return "_"
	}
	return strings.Map(func(c rune) rune {
		if isV7XRefChar(c) {
			return c
		}
		return '_'
	}, body)
}
