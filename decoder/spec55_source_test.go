package decoder

// spec55_source_test.go loads the transcribed GEDCOM 5.5 and 5.5.1
// specification files and turns them into the graph and document form
// spec_common_test.go measures.
//
// 7.0 is published with machine-readable extracted files. 5.5 and 5.5.1 are
// not: their structure set exists only as the Lineage-Linked Grammar printed in
// Chapter 2 of each PDF, so `scripts/spec55/transcribe.py` reads those chapters
// and writes the same files 7.0 ships. What lands under testdata/spec is the
// facts -- tag, superstructure, cardinality, value type -- plus the grammar
// lines the recorded defects have to quote, and none of the surrounding prose.
//
// A structure here is identified by where the grammar defines it rather than by
// a URI: "EVENT_DETAIL.DATE" is the DATE line of the EVENT_DETAIL production,
// and stays one structure however many event tags splice EVENT_DETAIL in. That
// is the same identity 7.0's URIs carry, spelled the way these specifications
// name things.

import (
	"path/filepath"
	"strings"
	"testing"
)

// spec55Editions are the two versions this harness covers, in the order they
// are reported.
var spec55Editions = []spec55Edition{
	{version: "5.5", dir: "../testdata/spec/gedcom-5.5"},
	{version: "5.5.1", dir: "../testdata/spec/gedcom-5.5.1"},
}

// spec55Edition names one transcribed specification.
type spec55Edition struct {
	version string
	dir     string
}

// The structures a probe document needs to know by name, because the harness
// injects them when a path does not supply them. The identities are the
// grammar's own production names, so these read as the lines they stand for.
const (
	// spec55Head is the header record.
	spec55Head = "HEADER.HEAD"

	// spec55Gedc is the container the version statement lives in.
	spec55Gedc = "HEADER.HEAD.GEDC"

	// spec55Vers is the version statement itself.
	spec55Vers = "HEADER.HEAD.GEDC.VERS"

	// spec55Trlr is the trailer.
	spec55Trlr = "LINEAGE_LINKED_GEDCOM.TRLR"

	// spec55Char is the character set declaration, which decides how the bytes
	// of the document are read before any of this is parsed.
	spec55Char = "HEADER.HEAD.CHAR"

	// spec55Form is the GEDCOM form declaration.
	spec55Form = "HEADER.HEAD.GEDC.FORM"
)

// spec55Defect is one place the specification text does not say what it means,
// as recorded by the transcription rather than judged here.
type spec55Defect struct {
	block   string
	line    string
	problem string
	reading string
}

// spec55Spec is one version's transcribed grammar, ready to probe with.
type spec55Spec struct {
	*specGraph
	*specForm

	version string

	// payloads maps a structure to the value type the grammar states for it,
	// verbatim. A structure with no payload is present with an empty value.
	payloads map[string]string

	// cardinalities maps a (superstructure, structure) pair to the cardinality
	// the grammar states, which is stated per pair because the same structure
	// spliced twice can be allowed a different number of times in each place.
	cardinalities map[[2]string]string

	// enumValues maps a value type defined as a list of literals to the last of
	// them. Last rather than first for the reason 7.0's harness takes the last:
	// CERTAINTY_ASSESSMENT begins with "0", which is what its typed field holds
	// when the line is absent, so a probe carrying it would measure nothing.
	enumValues map[string]string

	// defects are the specification-text problems the transcription hit.
	defects []spec55Defect

	// source is the provenance recorded in the SOURCE file.
	source map[string]string
}

// loadSpec55 reads one transcribed specification and derives its structure
// graph. It fails the test rather than returning an error: every caller here
// treats missing or malformed spec data as unrecoverable.
func loadSpec55(t *testing.T, edition spec55Edition) *spec55Spec {
	t.Helper()

	s := &spec55Spec{
		version:       edition.version,
		payloads:      map[string]string{},
		cardinalities: map[[2]string]string{},
		enumValues:    map[string]string{},
		source:        specSourceFile(t, filepath.Join(edition.dir, "SOURCE")),
	}

	var pairs []specPair
	for _, row := range specTSV(t, edition.dir, "substructures.tsv",
		"superstructure", "tag", "structure") {
		pairs = append(pairs, specPair{superstructure: row[0], tag: row[1], structure: row[2]})
	}
	for _, row := range specTSV(t, edition.dir, "payloads.tsv", "structure", "payload") {
		s.payloads[row[0]] = row[1]
	}
	for _, row := range specTSV(t, edition.dir, "cardinalities.tsv",
		"superstructure", "structure", "cardinality") {
		s.cardinalities[[2]string{row[0], row[1]}] = row[2]
	}
	for _, row := range specTSV(t, edition.dir, "primitives.tsv", "primitive", "size", "values") {
		if row[2] != "" {
			values := strings.Split(row[2], "|")
			s.enumValues["<"+row[0]+">"] = values[len(values)-1]
		}
	}
	for _, row := range specTSV(t, edition.dir, "defects.tsv",
		"block", "line", "problem", "reading") {
		s.defects = append(s.defects, spec55Defect{
			block: row[0], line: row[1], problem: row[2], reading: row[3],
		})
	}

	s.specGraph = newSpecGraph(pairs)
	s.specForm = &specForm{
		version: edition.version,
		head:    spec55Head,
		gedc:    spec55Gedc,
		vers:    spec55Vers,
		trlr:    spec55Trlr,
		fillers: spec55Fillers,
		value:   s.payloadFor,
		pointer: func(structure string) (string, bool) {
			return specPointerTarget(spec55Declared(s.payloads[structure]))
		},
	}
	return s
}

// cardinality returns what the grammar states for one pair, or the empty string
// where it states none -- which happens, and is recorded as a defect rather
// than filled in.
func (s *spec55Spec) cardinality(pair specPair) string {
	return s.cardinalities[[2]string{pair.superstructure, pair.structure}]
}

// spec55Fillers are the records every probe document can point at, in the order
// they are written. A 5.5.x pointer payload names a record type rather than a
// structure, so the target and the structure differ.
var spec55Fillers = []specFiller{
	{target: "XREF:INDI", xref: "@PI@", tag: "INDI", structure: "INDIVIDUAL_RECORD.INDI"},
	{target: "XREF:FAM", xref: "@PF@", tag: "FAM", structure: "FAM_RECORD.FAM"},
	{target: "XREF:SOUR", xref: "@PS@", tag: "SOUR", structure: "SOURCE_RECORD.SOUR"},
	{target: "XREF:REPO", xref: "@PR@", tag: "REPO", structure: "REPOSITORY_RECORD.REPO"},
	{target: "XREF:OBJE", xref: "@PO@", tag: "OBJE", structure: "MULTIMEDIA_RECORD.OBJE"},
	{target: "XREF:NOTE", xref: "@PN@", tag: "NOTE", structure: "NOTE_RECORD.NOTE"},
	{target: "XREF:SUBM", xref: "@PU@", tag: "SUBM", structure: "SUBMITTER_RECORD.SUBM"},
	{target: "XREF:SUBN", xref: "@PB@", tag: "SUBN", structure: "SUBMISSION_RECORD.SUBN"},
}

// spec55Xrefs maps a pointer payload's target to the filler record's identifier.
var spec55Xrefs = func() map[string]string {
	xrefs := map[string]string{}
	for _, filler := range spec55Fillers {
		xrefs[filler.target] = filler.xref
	}
	return xrefs
}()

// payloadFor returns a value of the type the grammar declares for the
// structure, or the empty string when it declares none.
//
// Which value it is matters in one specific way, learned building the 7.0
// harness: a probe measures whether removing a line changes the typed model, so
// the payload must be a value the model could not already hold without that
// line. Free-text payloads are therefore made unique per structure rather than
// sharing one literal, and enumerations take their last member rather than
// their first, which is a zero often enough to matter.
//
// Three values are not free choices at all. The character set and form
// declarations decide how the document is read before any of this is parsed,
// and the version statement decides which specification it is read against;
// giving those synthetic values would measure the harness rather than the
// decoder.
func (s *spec55Spec) payloadFor(structure string) string {
	switch structure {
	case spec55Vers:
		return s.version
	case spec55Char:
		return "UTF-8"
	case spec55Form:
		return "LINEAGE-LINKED"
	}

	payload := spec55Declared(s.payloads[structure])
	if payload == "" {
		return ""
	}

	if target, ok := specPointerTarget(payload); ok {
		return spec55Xrefs[target]
	}
	if payload == "Y" {
		return "Y"
	}
	if value, ok := s.enumValues[payload]; ok {
		return value
	}
	return spec55Value(payload, structure)
}

// spec55Declared reduces a declared payload to the one alternative a probe
// measures.
//
// "[ <SUBMITTER_TEXT> | <NULL>]" and "[Y|<NULL>]" are the grammar's way of
// saying the payload is optional, and the alternative that is there to be
// measured is always the first. Everything downstream -- the value a probe
// carries and the test for whether it is a cross-reference -- reads the result
// of this rather than the raw text, so the two cannot disagree about whether a
// structure points at a record that has to exist in the document.
func spec55Declared(payload string) string {
	if !strings.HasPrefix(payload, "[") {
		return payload
	}
	first := strings.SplitN(strings.TrimPrefix(payload, "["), "|", 2)[0]
	return strings.TrimSuffix(strings.TrimSpace(first), "]")
}

// spec55Value builds a value of one of the grammar's declared types.
//
// The value types are named rather than formally defined -- the specification
// describes each in prose under "Primitive Elements of the Lineage-Linked
// Form" -- so this switch is where that prose is read. Anything not named here
// is free text, and gets a literal unique to its structure.
func spec55Value(payload, structure string) string {
	switch payload {
	case "<DATE_VALUE>", "<CHANGE_DATE>", "<DATE_LDS_ORD>", "<PUBLICATION_DATE>",
		"<TRANSMISSION_DATE>", "<ENTRY_RECORDING_DATE>", "<DATE_EXACT>":
		return "1 JAN 2000"
	case "<DATE_PERIOD>":
		return "FROM 1 JAN 2000 TO 31 DEC 2000"
	case "<TIME_VALUE>":
		return "12:00:00"
	case "<AGE_AT_EVENT>":
		return "30y"
	case "<NAME_PERSONAL>":
		return "John /Doe/"
	case "<PLACE_LATITUDE>":
		return "N18.150944"
	case "<PLACE_LONGITUDE>":
		return "E168.150944"
	case "<COUNT_OF_CHILDREN>", "<COUNT_OF_MARRIAGES>",
		"<GENERATIONS_OF_ANCESTORS>", "<GENERATIONS_OF_DESCENDANTS>":
		return "1"
	case "<LANGUAGE_OF_TEXT>", "<LANGUAGE_PREFERENCE>":
		return "English"
	case "<MULTIMEDIA_FILE_REFN>", "<MULTIMEDIA_FILE_REFERENCE>":
		return "media/example.jpg"
	case "<ADDRESS_EMAIL>":
		return "someone@example.com"
	case "<ADDRESS_WEB_PAGE>":
		return "https://example.com/"
	default:
		return "text-" + structure
	}
}
