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
// "_"; if that identifier is already taken by a record or by a pointer no
// record defines (or is the 7.0 null pointer @VOID@), a numeric suffix ("_2",
// "_3", ...) keeps it unique. A pointer to an undefined XRef that differs
// only in ASCII case from a record's XRef (@test@ for @TEST@) is joined to
// that record; any other undefined pointer gets a
// fresh identifier the same way a record does, so it stays dangling rather
// than resolving to an unrelated record. Each rewrite is recorded as a
// Normalized note.
func normalizeXRefsForV7(doc *gedcom.Document, report *gedcom.ConversionReport) {
	mapping, dangling := buildXRefMapping(doc)
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
	for _, ref := range dangling {
		report.AddNormalized(gedcom.ConversionNote{
			Path:     "Pointer " + ref,
			Original: ref,
			Result:   mapping[ref],
			Reason:   v7XRefReason + "; no record defines this XRef",
		})
	}

	gedcom.Apply(doc, mapping)

	// The type name predates rewriting characters other than case; it is
	// kept so existing report consumers keep matching it. Count covers
	// every rewritten XRef, dangling pointers included.
	report.AddTransformation(gedcom.Transformation{
		Type:        "XREF_UPPERCASE",
		Description: "Normalized XRefs to uppercase letters, digits and underscore (required for GEDCOM 7.0)",
		Count:       len(mapping),
	})
}

// buildXRefMapping maps each XRef that is not a valid GEDCOM 7.0 Xref to a
// valid one, as described on normalizeXRefsForV7, and also returns the
// mapped pointers that no record defines. Valid XRefs are not included.
// Records and pointers are visited in document order, so the result is
// deterministic.
func buildXRefMapping(doc *gedcom.Document) (mapping map[string]string, dangling []string) {
	mapping = make(map[string]string)

	taken, byUpper := indexRecordXRefs(doc)

	// A valid pointer that no record defines also keeps its name, so it is
	// reserved too; otherwise a rewritten record could take it over. One that
	// differs only in case from a record is left free, so that record's
	// rewrite joins it.
	undefined := undefinedPointers(doc)
	for _, ref := range undefined {
		if !isV7XRef(ref) {
			dangling = append(dangling, ref)
			continue
		}
		if _, ok := byUpper[ref]; !ok {
			taken[ref] = true
		}
	}

	nextSuffix := make(map[string]int) // base -> next suffix to try
	assign := func(xref string) string {
		base := toV7XRefBody(xref)
		candidate := "@" + base + "@"
		for taken[candidate] {
			n := max(nextSuffix[base], 2)
			nextSuffix[base] = n + 1
			candidate = "@" + base + "_" + strconv.Itoa(n) + "@"
		}
		taken[candidate] = true
		return candidate
	}

	for _, record := range doc.Records {
		if record == nil || record.XRef == "" || isV7XRef(record.XRef) {
			continue
		}
		if _, done := mapping[record.XRef]; done {
			continue
		}
		mapping[record.XRef] = assign(record.XRef)
	}

	for _, ref := range dangling {
		if def, ok := byUpper[asciiUpper(ref)]; ok {
			if newDef, renamed := mapping[def]; renamed {
				def = newDef
			}
			mapping[ref] = def
			continue
		}
		mapping[ref] = assign(ref)
	}

	return mapping, dangling
}

// indexRecordXRefs returns the identifiers a rewrite must avoid (valid record
// XRefs, which keep their name, and the 7.0 null pointer @VOID@) and, for the
// case-only join, each record XRef keyed by its ASCII-uppercased form (the
// first record wins).
func indexRecordXRefs(doc *gedcom.Document) (taken map[string]bool, byUpper map[string]string) {
	taken = map[string]bool{"@VOID@": true}
	byUpper = make(map[string]string)
	for _, record := range doc.Records {
		if record == nil || record.XRef == "" {
			continue
		}
		if isV7XRef(record.XRef) {
			taken[record.XRef] = true
		}
		upper := asciiUpper(record.XRef)
		if _, ok := byUpper[upper]; !ok {
			byUpper[upper] = record.XRef
		}
	}
	return taken, byUpper
}

// undefinedPointers returns, in document order and without duplicates, each
// pointer that no record defines. Pointers are found with
// gedcom.VisitForVersion and in the header's submitter and tags, under the
// grammar of the source's declared version (Document.IsPointerXRef), so a
// spaced 5.5/5.5.1 pointer is found too.
func undefinedPointers(doc *gedcom.Document) []string {
	// Only a header-declared 5.5/5.5.1 version relaxes the grammar, never a
	// guessed one (Header.Version), matching the decoder and
	// Document.IsPointerXRef: guessing would turn text such as "@home office@"
	// into a pointer. So without a declared version a dangling spaced pointer
	// is not found here and is left unchanged; a defined spaced record XRef is
	// still renamed, since its record is known to be one.
	var declared gedcom.Version
	if doc.Header != nil {
		declared = doc.Header.DeclaredVersion
	}
	var refs []string
	seen := make(map[string]bool) // defined XRefs and refs already added
	for _, record := range doc.Records {
		if record != nil {
			seen[record.XRef] = true
		}
	}
	add := func(ref string) {
		if ref == "" || seen[ref] {
			return
		}
		seen[ref] = true
		refs = append(refs, ref)
	}
	if doc.Header != nil {
		add(doc.Header.Submitter)
		for _, tag := range doc.Header.Tags {
			if tag == nil {
				continue
			}
			for _, v := range []string{tag.XRef, tag.Value} {
				if doc.IsPointerXRef(v) {
					add(v)
				}
			}
		}
	}
	for _, record := range doc.Records {
		gedcom.VisitForVersion(record, declared, add)
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
	body := asciiUpper(strings.TrimSuffix(strings.TrimPrefix(xref, "@"), "@"))
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

// asciiUpper uppercases only ASCII letters. strings.ToUpper would also map
// characters such as dotless i (U+0131) onto ASCII letters, joining XRefs
// that differ by more than case.
func asciiUpper(s string) string {
	return strings.Map(func(c rune) rune {
		if c >= 'a' && c <= 'z' {
			return c - 'a' + 'A'
		}
		return c
	}, s)
}
