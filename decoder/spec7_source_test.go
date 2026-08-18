package decoder

// spec7_source_test.go loads the vendored GEDCOM 7.0 specification files and
// turns them into probe documents.
//
// The specification's own extracted files (testdata/spec/gedcom-7.0) list every
// structure the standard defines, keyed by the superstructure it appears under.
// That key is the whole point: GEDCOM tag meaning depends on context, so a flat
// per-tag table would overstate coverage. What this file does with the data:
//
//  1. read substructures.tsv into a directed graph of (superstructure -> tag ->
//     substructure) edges, with the ten top-level structures as roots;
//  2. find each structure's shortest path from a root, so any (superstructure,
//     tag) pair can be expressed as a concrete nesting of GEDCOM lines;
//  3. render that path as a minimal, decodable GEDCOM 7.0 document, giving every
//     line a payload of the type payloads.tsv declares for it.
//
// spec7_coverage_test.go decodes those documents to derive what the library
// supports. Nothing here reads or asserts anything about the decoder.

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// spec7Dir is the vendored copy of the specification's extracted files.
const spec7Dir = "../testdata/spec/gedcom-7.0"

// spec7TermPrefix is the URI namespace every standard 7.0 structure lives under.
const spec7TermPrefix = "https://gedcom.io/terms/v7/"

// spec7Step is one line of a GEDCOM structure path: the tag as it appears in
// the file, and the URI of the structure that tag denotes in that position.
type spec7Step struct {
	tag       string
	structure string
}

// spec7Pair is one (superstructure, tag) entry of the specification inventory,
// which is the unit this whole exercise reports coverage for. superstructure is
// empty for the ten structures that appear at level 0.
type spec7Pair struct {
	superstructure string
	tag            string
	structure      string
}

// spec7Spec is the specification data needed to build probe documents.
type spec7Spec struct {
	// pairs is every (superstructure, tag) entry, in specification file order.
	pairs []spec7Pair

	// payloads maps a structure URI to its declared payload type. A structure
	// with no payload is present with an empty value.
	payloads map[string]string

	// enumValues maps a structure URI whose payload is an enumeration to a
	// valid value for it.
	enumValues map[string]string

	// paths maps a structure URI to its shortest path from a top-level
	// structure, inclusive of the structure itself.
	paths map[string][]spec7Step

	// source is the provenance recorded in the vendored SOURCE file.
	source map[string]string
}

// spec7Term shortens a structure URI to the term name the specification uses
// for it, e.g. "https://gedcom.io/terms/v7/INDI-RESI" to "INDI-RESI".
func spec7Term(uri string) string {
	return strings.TrimPrefix(uri, spec7TermPrefix)
}

// loadSpec7 reads the vendored specification files and derives the structure
// paths. It fails the test rather than returning an error: every caller here
// treats missing or malformed spec data as unrecoverable.
func loadSpec7(t *testing.T) *spec7Spec {
	t.Helper()

	s := &spec7Spec{
		payloads:   map[string]string{},
		enumValues: map[string]string{},
		source:     spec7Source(t),
	}

	for _, row := range spec7TSV(t, "substructures.tsv", 3) {
		s.pairs = append(s.pairs, spec7Pair{superstructure: row[0], tag: row[1], structure: row[2]})
	}
	for _, row := range spec7TSV(t, "payloads.tsv", 2) {
		s.payloads[row[0]] = row[1]
	}

	// enumerations.tsv names the enumeration set a structure draws from;
	// enumerationsets.tsv lists that set's members. Collapse the two into one
	// representative value per structure, taking the first member in file order
	// so the choice is stable across runs.
	setMember := map[string]string{}
	for _, row := range spec7TSV(t, "enumerationsets.tsv", 2) {
		if _, seen := setMember[row[0]]; !seen {
			setMember[row[0]] = row[1]
		}
	}
	for _, row := range spec7TSV(t, "enumerations.tsv", 2) {
		if member, ok := setMember[row[1]]; ok {
			s.enumValues[row[0]] = spec7EnumTag(member)
		}
	}

	s.paths = spec7Paths(t, s.pairs)
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

// spec7TSV reads a tab-separated file from the vendored directory, skipping its
// header row and requiring exactly fields columns.
func spec7TSV(t *testing.T, name string, fields int) [][]string {
	t.Helper()

	f, err := os.Open(filepath.Join(spec7Dir, name))
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	defer func() { _ = f.Close() }()

	r := csv.NewReader(f)
	r.Comma = '\t'
	r.FieldsPerRecord = fields
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	if len(rows) == 0 {
		t.Fatalf("%s is empty", name)
	}
	return rows[1:]
}

// spec7Source parses the vendored SOURCE file into its "key: value" entries.
func spec7Source(t *testing.T) map[string]string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(spec7Dir, "SOURCE"))
	if err != nil {
		t.Fatalf("read SOURCE: %v", err)
	}

	source := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			t.Fatalf("SOURCE line is not \"key: value\": %q", line)
		}
		source[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return source
}

// spec7Paths finds each structure's shortest path from a top-level structure.
//
// Breadth-first from the roots, so a structure reachable several ways gets its
// shallowest nesting -- the fewest lines a probe document has to spend before
// reaching the structure under test. Ties break on tag then structure URI, so
// the chosen path does not depend on map iteration order.
func spec7Paths(t *testing.T, pairs []spec7Pair) map[string][]spec7Step {
	t.Helper()

	children := map[string][]spec7Pair{}
	for _, p := range pairs {
		children[p.superstructure] = append(children[p.superstructure], p)
	}
	for _, group := range children {
		sort.Slice(group, func(i, j int) bool {
			if group[i].tag != group[j].tag {
				return group[i].tag < group[j].tag
			}
			return group[i].structure < group[j].structure
		})
	}

	paths := map[string][]spec7Step{}
	var queue []string
	for _, root := range children[""] {
		if _, seen := paths[root.structure]; seen {
			continue
		}
		paths[root.structure] = []spec7Step{{tag: root.tag, structure: root.structure}}
		queue = append(queue, root.structure)
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, child := range children[current] {
			if _, seen := paths[child.structure]; seen {
				continue
			}
			path := make([]spec7Step, len(paths[current]), len(paths[current])+1)
			copy(path, paths[current])
			paths[child.structure] = append(path, spec7Step{tag: child.tag, structure: child.structure})
			queue = append(queue, child.structure)
		}
	}
	return paths
}

// pathFor returns the nesting of lines that reaches pair: the path to its
// superstructure, extended by the pair's own tag. Top-level pairs are their own
// path. The second result is false when the superstructure is unreachable from
// any top-level structure, which would mean the specification data does not
// describe a connected document.
func (s *spec7Spec) pathFor(pair spec7Pair) ([]spec7Step, bool) {
	step := spec7Step{tag: pair.tag, structure: pair.structure}
	if pair.superstructure == "" {
		return []spec7Step{step}, true
	}
	parent, ok := s.paths[pair.superstructure]
	if !ok {
		return nil, false
	}
	path := make([]spec7Step, len(parent), len(parent)+1)
	copy(path, parent)
	return append(path, step), true
}

// spec7Xrefs are the cross-reference identifiers of the filler records every
// probe document carries, so that pointer payloads point at something real.
// They are deliberately distinct from the probe record's own identifier.
var spec7Xrefs = map[string]string{
	spec7TermPrefix + "record-INDI":  "@PI@",
	spec7TermPrefix + "record-FAM":   "@PF@",
	spec7TermPrefix + "record-SOUR":  "@PS@",
	spec7TermPrefix + "record-REPO":  "@PR@",
	spec7TermPrefix + "record-OBJE":  "@PO@",
	spec7TermPrefix + "record-SNOTE": "@PN@",
	spec7TermPrefix + "record-SUBM":  "@PU@",
}

// spec7ProbeXref is the identifier given to the record a probe path is rooted
// in, when that root takes one.
const spec7ProbeXref = "@PX@"

// spec7PointerTarget reports the record type a pointer payload points at, and
// whether the payload is a pointer at all.
func spec7PointerTarget(payload string) (string, bool) {
	if !strings.HasPrefix(payload, "@<") {
		return "", false
	}
	return strings.TrimSuffix(strings.TrimPrefix(payload, "@<"), ">@"), true
}

// payloadFor returns a value of the type the specification declares for the
// structure, or the empty string when it declares none. The value only has to
// be well formed enough that the decoder has something real to store; nothing
// here depends on which value it is.
func (s *spec7Spec) payloadFor(structure string) string {
	// The specification types the version statement as a plain string, but a
	// probe document has to declare which GEDCOM it is before any of this means
	// anything. Stating 7.0 here is part of making the document 7.0, not an
	// annotation about coverage.
	if structure == gedcVersURI {
		return "7.0"
	}

	payload := s.payloads[structure]

	if target, ok := spec7PointerTarget(payload); ok {
		return spec7Xrefs[target]
	}

	switch payload {
	case "":
		return ""
	case "Y|<NULL>":
		return "Y"
	case "http://www.w3.org/2001/XMLSchema#string":
		return "text"
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
		return "text"
	}
}

// spec7SentinelTag is a tag no version of GEDCOM defines and no vendor uses. A
// decoder that inspects a context at all must report it as unknown there, so it
// answers a question no standard tag can: whether silence about a standard tag
// means the decoder accepted it deliberately or never looked.
//
// It carries no underscore on purpose. Underscore-prefixed tags are vendor
// extensions, which ADR 0003 routes to raw storage without complaint, so one
// would be silent everywhere and measure nothing.
const spec7SentinelTag = "ZZZZ"

// sentinelFor renders a document with the sentinel tag in place of a
// substructure of superstructure. The second result is false when the
// superstructure is unreachable.
func (s *spec7Spec) sentinelFor(superstructure string) (spec7Probe, bool) {
	sentinel := spec7Step{tag: spec7SentinelTag}
	if superstructure == "" {
		return s.probeFor([]spec7Step{sentinel}), true
	}
	parent, ok := s.paths[superstructure]
	if !ok {
		return spec7Probe{}, false
	}
	path := make([]spec7Step, len(parent), len(parent)+1)
	copy(path, parent)
	return s.probeFor(append(path, sentinel)), true
}

// spec7Probe is a pair of documents that differ in exactly one line.
type spec7Probe struct {
	// with is a document in which the structure under test is present.
	with string

	// without is the same document with that one line left out. It is the
	// control: any difference in the decoded typed model between the two is
	// attributable to the structure under test and nothing else.
	without string

	// line is the 1-based line number of the structure under test within with.
	line int
}

// gedcVersURI identifies the VERS structure under GEDC, which is what states a
// document's version. A probe path that already contains it supplies the
// version itself and must not be given a second one -- otherwise the control
// document would still declare 7.0 with the line removed, and the probe would
// measure nothing.
const gedcVersURI = spec7TermPrefix + "GEDC-VERS"

// probeFor renders the pair of documents that isolate path's deepest line.
//
// The document is the smallest thing that decodes: a header stating GEDCOM 7.0,
// the path under test nested from its top-level structure, a filler record for
// each pointer payload the path actually uses, and a trailer. Levels come
// straight from the path index, since a path always starts at level 0.
//
// Filler records are demand-driven rather than always present because they are
// not inert: a record type is itself evidence of a GEDCOM version, and unused
// fillers would let version detection succeed from the record list alone. Both
// documents get the same fillers -- those the full path needs -- so the pair
// still differs in exactly one line.
func (s *spec7Spec) probeFor(path []spec7Step) spec7Probe {
	headRooted := path[0].structure == spec7TermPrefix+"HEAD"

	suppliesVersion := false
	needed := map[string]bool{}
	for _, step := range path {
		if step.structure == gedcVersURI {
			suppliesVersion = true
		}
		if target, ok := spec7PointerTarget(s.payloads[step.structure]); ok {
			needed[target] = true
		}
	}

	render := func(omitLast bool) (lines []string, probeLine int) {
		steps := path
		if omitLast {
			steps = path[:len(path)-1]
		}

		var head, body []string
		for i, step := range steps {
			line := fmt.Sprintf("%d %s", i, step.tag)
			if i == 0 && !headRooted && step.tag != "TRLR" {
				line = fmt.Sprintf("0 %s %s", spec7ProbeXref, step.tag)
			}
			if payload := s.payloadFor(step.structure); payload != "" {
				line += " " + payload
			}
			if headRooted {
				head = append(head, line)
			} else {
				body = append(body, line)
			}
		}

		if !headRooted {
			head = []string{"0 HEAD"}
		}
		// The version statement goes last in the header so it never sits
		// between a path line and its parent. It is omitted only when the path
		// is itself the version statement, which is the one case where adding
		// it would mask what the probe is measuring. It stays even when the
		// control document drops `0 HEAD` itself, leaving it orphaned: the
		// documents have to differ in exactly one line, and an orphaned header
		// body is the honest answer to what `0 HEAD` contributes.
		if !suppliesVersion {
			head = append(head, "1 GEDC", "2 VERS 7.0")
		}

		lines = append(lines, head...)
		lines = append(lines, body...)
		if headRooted {
			probeLine = len(steps)
		} else {
			probeLine = len(head) + len(body)
		}

		for _, uri := range spec7FillerOrder {
			if !needed[uri] {
				continue
			}
			line := fmt.Sprintf("0 %s %s", spec7Xrefs[uri], strings.TrimPrefix(spec7Term(uri), "record-"))
			if payload := s.payloadFor(uri); payload != "" {
				line += " " + payload
			}
			lines = append(lines, line)
		}
		lines = append(lines, "0 TRLR")
		return lines, probeLine
	}

	with, probeLine := render(false)
	without, _ := render(true)
	return spec7Probe{
		with:    strings.Join(with, "\n") + "\n",
		without: strings.Join(without, "\n") + "\n",
		line:    probeLine,
	}
}

// spec7FillerOrder fixes the order of the filler records so probe documents are
// byte-for-byte reproducible.
var spec7FillerOrder = []string{
	spec7TermPrefix + "record-INDI",
	spec7TermPrefix + "record-FAM",
	spec7TermPrefix + "record-SOUR",
	spec7TermPrefix + "record-REPO",
	spec7TermPrefix + "record-OBJE",
	spec7TermPrefix + "record-SNOTE",
	spec7TermPrefix + "record-SUBM",
}
