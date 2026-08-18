package decoder

// spec7_source_test.go loads the vendored GEDCOM 7.0 specification files and
// turns them into the graph and document form spec_common_test.go measures.
//
// The specification's own extracted files (testdata/spec/gedcom-7.0) list every
// structure the standard defines, keyed by the superstructure it appears under.
// That key is the whole point: GEDCOM tag meaning depends on context, so a flat
// per-tag table would overstate coverage. What this file does with the data:
//
//  1. read substructures.tsv into (superstructure, tag, structure) pairs, which
//     newSpecGraph turns into a directed graph with the ten top-level
//     structures as roots;
//  2. describe what a minimal 7.0 document looks like, so any pair can be
//     rendered as a concrete nesting of GEDCOM lines with a payload of the type
//     payloads.tsv declares for it.
//
// spec7_coverage_test.go decodes those documents to derive what the library
// supports. Nothing here reads or asserts anything about the decoder.

import (
	"path/filepath"
	"strings"
	"testing"
)

// spec7Dir is the vendored copy of the specification's extracted files.
const spec7Dir = "../testdata/spec/gedcom-7.0"

// spec7TermPrefix is the URI namespace every standard 7.0 structure lives under.
const spec7TermPrefix = "https://gedcom.io/terms/v7/"

// spec7Spec is the specification data needed to build probe documents.
type spec7Spec struct {
	*specGraph
	*specForm

	// payloads maps a structure URI to its declared payload type. A structure
	// with no payload is present with an empty value.
	payloads map[string]string

	// enumValues maps a structure URI whose payload is an enumeration to a
	// valid value for it.
	enumValues map[string]string

	// source is the provenance recorded in the vendored SOURCE file.
	source map[string]string
}

// spec7Term shortens a structure URI to the term name the specification uses
// for it, e.g. "https://gedcom.io/terms/v7/INDI-RESI" to "INDI-RESI".
func spec7Term(uri string) string {
	return strings.TrimPrefix(uri, spec7TermPrefix)
}

// loadSpec7 reads the vendored specification files and derives the structure
// graph. It fails the test rather than returning an error: every caller here
// treats missing or malformed spec data as unrecoverable.
func loadSpec7(t *testing.T) *spec7Spec {
	t.Helper()

	s := &spec7Spec{
		payloads:   map[string]string{},
		enumValues: map[string]string{},
		source:     spec7Source(t),
	}

	var pairs []specPair
	for _, row := range spec7TSV(t, "substructures.tsv", "superstructure", "tag", "structure") {
		pairs = append(pairs, specPair{superstructure: row[0], tag: row[1], structure: row[2]})
	}
	for _, row := range spec7TSV(t, "payloads.tsv", "structure", "payload") {
		s.payloads[row[0]] = row[1]
	}

	// enumerations.tsv names the enumeration set a structure draws from;
	// enumerationsets.tsv lists that set's members. Collapse the two into one
	// representative value per structure, taking the last member in file order:
	// stable across runs, and away from the first, which for QUAY is "0" -- the
	// value its typed field holds when the line is absent, so a probe carrying
	// it would measure nothing. See payloadFor.
	setMember := map[string]string{}
	for _, row := range spec7TSV(t, "enumerationsets.tsv", "set", "value") {
		setMember[row[0]] = row[1]
	}
	for _, row := range spec7TSV(t, "enumerations.tsv", "structure", "set") {
		if member, ok := setMember[row[1]]; ok {
			s.enumValues[row[0]] = spec7EnumTag(member)
		}
	}

	s.specGraph = newSpecGraph(pairs)
	s.specForm = &specForm{
		version: "7.0",
		head:    spec7TermPrefix + "HEAD",
		gedc:    gedcURI,
		vers:    gedcVersURI,
		trlr:    trlrURI,
		fillers: spec7Fillers,
		value:   s.payloadFor,
		pointer: func(structure string) (string, bool) {
			return specPointerTarget(s.payloads[structure])
		},
	}
	return s
}

// spec7EnumTag reduces an enumeration value URI to the tag written in the file.
// Values are named either for the tag alone ("enum-BOTH", "MARR") or qualified
// by the structure they belong to ("enum-ADOP-HUSB" is written "HUSB"), so the
// tag is the last hyphen-separated segment after the "enum-" prefix is dropped.
func spec7EnumTag(uri string) string {
	name := strings.TrimPrefix(spec7Term(uri), "enum-")
	if i := strings.LastIndex(name, "-"); i >= 0 {
		return name[i+1:]
	}
	return name
}

// spec7TSV reads one of the vendored files. Upstream has shipped them without a
// header row, which specTSV is what catches.
func spec7TSV(t *testing.T, name string, columns ...string) [][]string {
	t.Helper()

	return specTSV(t, spec7Dir, name, columns...)
}

// spec7Source parses the vendored SOURCE file into its "key: value" entries.
func spec7Source(t *testing.T) map[string]string {
	t.Helper()

	return specSourceFile(t, filepath.Join(spec7Dir, "SOURCE"))
}

// payloadFor returns a value of the type the specification declares for the
// structure, or the empty string when it declares none.
//
// Which value it is does matter, in one specific way: a probe measures whether
// removing a line changes the typed model, so the payload must be a value the
// model could not already hold without that line. Two ways it can collide, both
// of which produced a wrong published status before this was understood:
//
//   - a shared literal. Every xsd:string payload used to be "text", so an ADR1
//     carrying "text" under an ADDR carrying "text" left Address.Line1 unchanged
//     -- the ADDR line seeds it and ADR1 overwrote it with the same string.
//     Free-text payloads are therefore unique per structure.
//   - a zero value. QUAY decodes to an int, and its enumeration's first member
//     is "0", which is what the field holds anyway. Enumerations therefore take
//     their last member rather than their first.
func (s *spec7Spec) payloadFor(structure string) string {
	// The specification types the version statement as a plain string, but a
	// probe document has to declare which GEDCOM it is before any of this means
	// anything. Stating 7.0 here is part of making the document 7.0, not an
	// annotation about coverage.
	if structure == gedcVersURI {
		return "7.0"
	}

	payload := s.payloads[structure]

	if target, ok := specPointerTarget(payload); ok {
		return spec7Xrefs[target]
	}

	switch payload {
	case "":
		return ""
	case "Y|<NULL>":
		return "Y"
	case "http://www.w3.org/2001/XMLSchema#string":
		return "text-" + spec7Term(structure)
	case "http://www.w3.org/2001/XMLSchema#nonNegativeInteger":
		return "1"
	case "http://www.w3.org/2001/XMLSchema#Language":
		return "en"
	case "http://www.w3.org/2001/XMLSchema#anyURI":
		return "https://example.com/"
	case "http://www.w3.org/ns/dcat#mediaType":
		return "text/plain"
	case spec7TermPrefix + "type-Enum", spec7TermPrefix + "type-List#Enum":
		if value, ok := s.enumValues[structure]; ok {
			return value
		}
		return "OTHER"
	case spec7TermPrefix + "type-List#Text":
		return "one, two"
	case spec7TermPrefix + "type-Name":
		return "John /Doe/"
	case spec7TermPrefix + "type-FilePath":
		return "media/example.jpg"
	case spec7TermPrefix + "type-Date", spec7TermPrefix + "type-Date#exact":
		return "1 JAN 2000"
	case spec7TermPrefix + "type-Date#period":
		return "FROM 1 JAN 2000 TO 31 DEC 2000"
	case spec7TermPrefix + "type-Time":
		return "12:00:00"
	case spec7TermPrefix + "type-Age":
		return "30y"
	case spec7TermPrefix + "type-Latitude":
		return "N18.150944"
	case spec7TermPrefix + "type-Longitude":
		return "E168.150944"
	case spec7TermPrefix + "type-TagDef":
		return "_EXAMPLE https://example.com/ext"
	default:
		return "text-" + spec7Term(structure)
	}
}

// The structures a probe document needs to know by name, because the harness
// injects them when a path does not supply them.
const (
	// gedcURI is the container the version statement lives in.
	gedcURI = spec7TermPrefix + "GEDC"

	// gedcVersURI is the version statement itself.
	gedcVersURI = spec7TermPrefix + "GEDC-VERS"

	// trlrURI is the trailer.
	trlrURI = spec7TermPrefix + "TRLR"
)

// spec7Fillers are the records every probe document can point at, in the order
// they are written.
var spec7Fillers = []specFiller{
	spec7Filler("INDI", "@PI@"),
	spec7Filler("FAM", "@PF@"),
	spec7Filler("SOUR", "@PS@"),
	spec7Filler("REPO", "@PR@"),
	spec7Filler("OBJE", "@PO@"),
	spec7Filler("SNOTE", "@PN@"),
	spec7Filler("SUBM", "@PU@"),
}

// spec7Xrefs maps a pointer payload's target to the filler record's identifier.
var spec7Xrefs = func() map[string]string {
	xrefs := map[string]string{}
	for _, filler := range spec7Fillers {
		xrefs[filler.target] = filler.xref
	}
	return xrefs
}()

// spec7Filler describes one filler record. In 7.0 a pointer payload names the
// record structure's URI, so the target and the structure are the same thing.
func spec7Filler(tag, xref string) specFiller {
	uri := spec7TermPrefix + "record-" + tag
	return specFiller{target: uri, xref: xref, tag: tag, structure: uri}
}
