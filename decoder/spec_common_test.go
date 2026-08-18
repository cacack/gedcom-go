package decoder

// spec_common_test.go is the machinery both specification coverage harnesses
// run on: GEDCOM 7.0's in spec7_*.go, and 5.5/5.5.1's in spec55_*.go.
//
// The two differ only in where their inventory comes from and what a minimal
// document of that version looks like. Everything between -- finding a path to
// a structure, building the pair of documents that isolates one line, decoding
// both, and reading the difference -- is the same measurement, and lives here
// so that a change to how coverage is derived cannot mean one thing for 7.0 and
// another for 5.5.

import (
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// updateSpecCoverage regenerates the checked-in coverage documents instead of
// comparing against them. Run via `make spec-coverage`. Both harnesses share
// it, so the two reports are always regenerated together and cannot drift into
// describing different revisions of the decoder.
var updateSpecCoverage = flag.Bool("update-spec-coverage", false,
	"rewrite the docs/reference/gedcom-*-coverage.md reports from the decoder's actual behaviour")

// specPublish writes a regenerated coverage document, or compares the derived
// one against what is checked in.
//
// It refuses to write after the measurement has reported an error. A pair whose
// superstructure is unreachable is reported and skipped, which leaves the
// document short of a row -- and writing it then would turn a loud failure into
// a quiet, committable omission.
func specPublish(t *testing.T, path, got string) {
	t.Helper()

	if *updateSpecCoverage {
		if t.Failed() {
			t.Fatalf("not writing %s: the measurement reported errors above, so the "+
				"document would be published incomplete", path)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		t.Logf("wrote %s", path)
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s (regenerate with `make spec-coverage`): %v", path, err)
	}
	if string(want) != got {
		t.Errorf("%s is out of date: the decoder's coverage has changed.\n"+
			"Regenerate with `make spec-coverage` and review the diff.\n%s",
			path, specDiff(string(want), got))
	}
}

// specStep is one line of a GEDCOM structure path: the tag as it appears in the
// file, and the identity of the structure that tag denotes in that position.
type specStep struct {
	tag       string
	structure string
}

// specPair is one (superstructure, tag) entry of a specification inventory,
// which is the unit coverage is reported for. superstructure is empty for the
// structures that appear at level 0.
type specPair struct {
	superstructure string
	tag            string
	structure      string
}

// specStatus is how the decoder handles one (superstructure, tag) pair.
type specStatus string

const (
	// specTyped means the structure reaches the typed model.
	specTyped specStatus = "typed"

	// specRawFlagged means the structure stays in raw tags only, and the
	// decoder reports it as an unknown tag. These are the sharpest gaps: the
	// decoder reads the context and does not understand the tag in it.
	specRawFlagged specStatus = "raw (flagged)"

	// specRawAccepted means the structure stays in raw tags only, but the
	// context does report unknown tags -- the sentinel is flagged there -- and
	// this tag is deliberately not among them. A known structure with no typed
	// field yet.
	specRawAccepted specStatus = "raw (accepted)"

	// specRawUndiagnosed means no unknown-tag diagnostic is emitted in this
	// context at all: not even the sentinel is reported. Silence about a
	// standard tag here therefore says nothing about whether the decoder knows
	// it.
	//
	// This does not mean the context is unparsed, and the harness cannot tell
	// the two causes apart. buildHeader reads every header line and types four
	// of them, yet buildDocument hands it no diagnosticCollector, so the whole
	// header is undiagnosed (issue #449). A context no parser visits looks
	// identical from out here (issue #448).
	specRawUndiagnosed specStatus = "raw (undiagnosed)"

	// specPartial means the structure reaches the typed model and is still
	// reported as unknown -- a tag handled in part, or reported in error.
	specPartial specStatus = "partial"
)

// specStatusOrder fixes the order statuses are reported in, best first.
var specStatusOrder = []specStatus{
	specTyped, specPartial, specRawAccepted, specRawFlagged, specRawUndiagnosed,
}

// specStatusMeaning explains each status where it is first tabulated, so a
// summary is readable without the methodology section.
var specStatusMeaning = map[specStatus]string{
	specTyped:          "Decoded into the typed model; reachable without walking `Record.Tags`.",
	specPartial:        "Reaches the typed model but is still reported as an unknown tag.",
	specRawAccepted:    "Raw tags only. The decoder reads this context and knows the tag, but has no typed field for it.",
	specRawFlagged:     "Raw tags only. The decoder reads this context and reports the tag as unknown.",
	specRawUndiagnosed: "Raw tags only, and no unknown-tag diagnostic is emitted anywhere in this context, so the silence says nothing.",
}

// specSentinelTag is a tag no version of GEDCOM defines and no vendor uses. A
// decoder that inspects a context at all must report it as unknown there, so it
// answers a question no standard tag can: whether silence about a standard tag
// means the decoder accepted it deliberately or never looked.
//
// It carries no underscore on purpose. Underscore-prefixed tags are vendor
// extensions, which ADR 0003 routes to raw storage without complaint, so one
// would be silent everywhere and measure nothing.
const specSentinelTag = "ZZZZ"

// specGraph is a specification's structures and how a document reaches them.
type specGraph struct {
	// pairs is every (superstructure, tag) entry, in specification file order.
	pairs []specPair

	// paths maps a structure to its shortest path from a top-level structure,
	// inclusive of the structure itself.
	paths map[string][]specStep
}

// newSpecGraph finds each structure's shortest path from a top-level structure.
//
// Breadth-first from the roots, so a structure reachable several ways gets its
// shallowest nesting -- the fewest lines a probe document has to spend before
// reaching the structure under test. Ties break on tag then structure, so the
// chosen path does not depend on map iteration order.
func newSpecGraph(pairs []specPair) *specGraph {
	children := map[string][]specPair{}
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

	paths := map[string][]specStep{}
	var queue []string
	for _, root := range children[""] {
		if _, seen := paths[root.structure]; seen {
			continue
		}
		paths[root.structure] = []specStep{{tag: root.tag, structure: root.structure}}
		queue = append(queue, root.structure)
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, child := range children[current] {
			if _, seen := paths[child.structure]; seen {
				continue
			}
			path := make([]specStep, len(paths[current]), len(paths[current])+1)
			copy(path, paths[current])
			paths[child.structure] = append(path, specStep{tag: child.tag, structure: child.structure})
			queue = append(queue, child.structure)
		}
	}
	return &specGraph{pairs: pairs, paths: paths}
}

// pathFor returns the nesting of lines that reaches pair: the path to its
// superstructure, extended by the pair's own tag. Top-level pairs are their own
// path. The second result is false when the superstructure is unreachable from
// any top-level structure, which would mean the specification data does not
// describe a connected document.
func (g *specGraph) pathFor(pair specPair) ([]specStep, bool) {
	step := specStep{tag: pair.tag, structure: pair.structure}
	if pair.superstructure == "" {
		return []specStep{step}, true
	}
	parent, ok := g.paths[pair.superstructure]
	if !ok {
		return nil, false
	}
	path := make([]specStep, len(parent), len(parent)+1)
	copy(path, parent)
	return append(path, step), true
}

// specFiller is a record a probe document carries so that pointer payloads
// point at something real.
type specFiller struct {
	// target is what a pointer payload names when it points at this record.
	target string

	// xref is the identifier the filler is given, deliberately distinct from
	// the probe record's own.
	xref string

	// tag and structure are how the filler's own line is written.
	tag       string
	structure string
}

// specForm is what building a probe document needs that differs between GEDCOM
// versions: which structures the version statement lives in, what it says, and
// which records exist to be pointed at.
type specForm struct {
	// version is the payload of the version statement in this form's documents.
	version string

	// head, gedc, vers and trlr are the structures the harness injects around a
	// path, identified so a path that already supplies one is not given a
	// second.
	head string
	gedc string
	vers string
	trlr string

	// fillers are the records available to be pointed at, in the order they are
	// written, so probe documents are byte-for-byte reproducible.
	fillers []specFiller

	// value returns a payload for a structure to carry, of the type the
	// specification declares for it.
	value func(structure string) string

	// pointer reports the record a structure's payload points at, and whether
	// it points at one at all. It has to agree with value about what a pointer
	// is: a path whose payload value is a cross-reference but whose pointer
	// test says otherwise would be written into a document with nothing to
	// point at.
	pointer func(structure string) (string, bool)
}

// specProbeXref is the identifier given to the record a probe path is rooted
// in, when that root takes one.
const specProbeXref = "@PX@"

// specPointerTarget reports what a pointer payload points at, and whether the
// payload is a pointer at all.
func specPointerTarget(payload string) (string, bool) {
	if !strings.HasPrefix(payload, "@<") {
		return "", false
	}
	return strings.TrimSuffix(strings.TrimPrefix(payload, "@<"), ">@"), true
}

// specProbe is a pair of documents that differ in exactly one line.
type specProbe struct {
	// with is a document in which the structure under test is present.
	with string

	// without is the same document with that one line left out. It is the
	// control: any difference in the decoded typed model between the two is
	// attributable to the structure under test and nothing else.
	without string

	// line is the 1-based line number of the structure under test within with.
	line int
}

// probeFor renders the pair of documents that isolate path's deepest line.
//
// The document is the smallest thing that decodes: a header stating the
// version, the path under test nested from its top-level structure, a filler
// record for each pointer payload the path actually uses, and a trailer. Levels
// come straight from the path index, since a path always starts at level 0.
//
// A probe document needs three things a path may not supply: a header, a
// version statement, and a trailer. What is missing is injected -- but never as
// a duplicate of a line the path itself contributes. A removed line that a
// retained line reproduces cannot change anything, so such a probe measures
// nothing and publishes a confident, meaningless status.
//
// Filler records are demand-driven rather than always present because they are
// not inert: a record type is itself evidence of a GEDCOM version, and unused
// fillers would let version detection succeed from the record list alone. Both
// documents get the same fillers -- those the full path needs -- so the pair
// still differs in exactly one line.
func (f *specForm) probeFor(path []specStep) specProbe {
	headRooted := path[0].structure == f.head

	var suppliesGedc, suppliesVersion, suppliesTrailer bool
	needed := map[string]bool{}
	for _, step := range path {
		switch step.structure {
		case f.gedc:
			suppliesGedc = true
		case f.vers:
			suppliesVersion = true
		case f.trlr:
			suppliesTrailer = true
		}
		if target, ok := f.pointer(step.structure); ok {
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
			if i == 0 && !headRooted && step.structure != f.trlr {
				line = fmt.Sprintf("0 %s %s", specProbeXref, step.tag)
			}
			if payload := f.value(step.structure); payload != "" {
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
		// between a path line and its parent. What is injected depends on how
		// much of it the path already supplies.
		switch {
		case suppliesVersion:
			// The path is the version statement. A second one would leave the
			// control declaring the version with the line removed.
		case suppliesGedc:
			// The path is the container. Nesting the version statement inside
			// it rather than in a second container is what makes the probe mean
			// something: with the line, the version is stated; without it, the
			// statement is orphaned and the document has no version.
			head = append(head, "2 VERS "+f.version)
		default:
			head = append(head, "1 GEDC", "2 VERS "+f.version)
		}

		lines = append(lines, head...)
		lines = append(lines, body...)
		if headRooted {
			probeLine = len(steps)
		} else {
			probeLine = len(head) + len(body)
		}

		for _, filler := range f.fillers {
			if !needed[filler.target] {
				continue
			}
			line := fmt.Sprintf("0 %s %s", filler.xref, filler.tag)
			if payload := f.value(filler.structure); payload != "" {
				line += " " + payload
			}
			lines = append(lines, line)
		}
		if !suppliesTrailer {
			lines = append(lines, "0 TRLR")
		}
		return lines, probeLine
	}

	with, probeLine := render(false)
	without, _ := render(true)
	return specProbe{
		with:    strings.Join(with, "\n") + "\n",
		without: strings.Join(without, "\n") + "\n",
		line:    probeLine,
	}
}

// sentinelFor renders a document with the sentinel tag in place of a
// substructure of superstructure. The second result is false when the
// superstructure is unreachable.
func (f *specForm) sentinelFor(graph *specGraph, superstructure string) (specProbe, bool) {
	sentinel := specStep{tag: specSentinelTag}
	if superstructure == "" {
		return f.probeFor([]specStep{sentinel}), true
	}
	parent, ok := graph.paths[superstructure]
	if !ok {
		return specProbe{}, false
	}
	path := make([]specStep, len(parent), len(parent)+1)
	copy(path, parent)
	return f.probeFor(append(path, sentinel)), true
}

// specProber measures pairs against the decoder, caching the one measurement
// that is shared between them.
type specProber struct {
	graph *specGraph
	form  *specForm

	// watched records whether the decoder reads a superstructure's contents at
	// all. That is a property of the superstructure rather than of each tag
	// under it, so it is measured once per superstructure rather than once per
	// pair.
	watched map[string]bool
}

// newSpecProber returns a prober for one specification's graph and form.
func newSpecProber(graph *specGraph, form *specForm) *specProber {
	return &specProber{graph: graph, form: form, watched: map[string]bool{}}
}

// measure classifies one pair. The second result is false when the pair's
// superstructure cannot be reached from any top-level structure, which would
// mean the specification data does not describe a connected document.
func (p *specProber) measure(t *testing.T, pair specPair) ([]specStep, specStatus, bool) {
	t.Helper()

	path, ok := p.graph.pathFor(pair)
	if !ok {
		return nil, "", false
	}
	if _, measured := p.watched[pair.superstructure]; !measured {
		sentinel, reachable := p.form.sentinelFor(p.graph, pair.superstructure)
		if !reachable {
			return nil, "", false
		}
		p.watched[pair.superstructure] = specFlagged(t, sentinel)
	}
	status := specClassify(t, p.form.probeFor(path), p.watched[pair.superstructure])
	return path, status, true
}

// specClassify decodes both halves of a probe and reports how the decoder
// handled the one line that differs between them. watched says whether the
// decoder reads the enclosing context at all, measured with the sentinel tag.
func specClassify(t *testing.T, probe specProbe, watched bool) specStatus {
	t.Helper()

	with, withDiagnostics := specDecode(t, probe.with)
	without, _ := specDecode(t, probe.without)

	typed := specTypedModel(with) != specTypedModel(without)
	flagged := specFlaggedIn(withDiagnostics, probe.line)

	if flagged && !watched {
		t.Fatalf("a tag was reported unknown in a context the sentinel tag is not "+
			"reported in; the sentinel is no longer a reliable probe:\n%s", probe.with)
	}

	switch {
	case typed && flagged:
		return specPartial
	case typed:
		return specTyped
	case flagged:
		return specRawFlagged
	case watched:
		return specRawAccepted
	default:
		return specRawUndiagnosed
	}
}

// specFlagged reports whether the decoder emitted UNKNOWN_TAG for the line a
// probe isolates.
func specFlagged(t *testing.T, probe specProbe) bool {
	t.Helper()

	_, diagnostics := specDecode(t, probe.with)
	return specFlaggedIn(diagnostics, probe.line)
}

// specFlaggedIn reports whether diagnostics contain an UNKNOWN_TAG for line.
func specFlaggedIn(diagnostics Diagnostics, line int) bool {
	for _, d := range diagnostics {
		if d.Code == CodeUnknownTag && d.Line == line {
			return true
		}
	}
	return false
}

// specDecode decodes a probe document in the lenient mode that collects
// diagnostics, which is the only mode that reports unknown tags.
func specDecode(t *testing.T, doc string) (*gedcom.Document, Diagnostics) {
	t.Helper()

	result, err := DecodeWithDiagnostics(bytes.NewReader([]byte(doc)), nil)
	if err != nil || result == nil {
		t.Fatalf("probe document failed to decode: %v\n%s", err, doc)
	}
	return result.Document, result.Diagnostics
}

// specMaxDepth bounds the typed-model rendering. The typed entities link by
// cross-reference string rather than by pointer, so there is nothing to cycle
// through; this only stops a future pointer field from hanging the test.
const specMaxDepth = 40

// specTypedModel renders everything a caller can reach without walking raw
// tags, as a string two documents can be compared by. Records with no parsed
// entity contribute nothing: at level 0 every line becomes a Record whether or
// not the decoder understands it, so including the wrapper would report every
// top-level structure as supported.
func specTypedModel(doc *gedcom.Document) string {
	var sb strings.Builder
	specRender(&sb, reflect.ValueOf(doc.Header), 0)
	specRender(&sb, reflect.ValueOf(doc.Schema), 0)
	sb.WriteString(string(doc.Vendor))
	for _, record := range doc.Records {
		if record.Entity == nil {
			continue
		}
		specRender(&sb, reflect.ValueOf(record.Entity), 0)
	}
	return sb.String()
}

// specRawField reports whether a struct field belongs to the raw half of the
// dual storage, or is otherwise not a statement about coverage.
func specRawField(name string) bool {
	// Tags is ADR 0003's raw storage: it holds every line regardless of
	// support. LineNumber shifts when the probe line is removed.
	return name == "Tags" || name == "LineNumber"
}

// specHasExportedField reports whether a struct type exposes any field at all.
func specHasExportedField(t reflect.Type) bool {
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).IsExported() {
			return true
		}
	}
	return false
}

// specRender writes a deterministic rendering of v, skipping raw-storage fields
// at every level.
func specRender(sb *strings.Builder, v reflect.Value, depth int) {
	if depth > specMaxDepth {
		sb.WriteString("...")
		return
	}

	switch v.Kind() {
	case reflect.Invalid:
		sb.WriteString("~")
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			sb.WriteString("~")
			return
		}
		specRender(sb, v.Elem(), depth+1)
	case reflect.Struct:
		// A struct with no exported fields at all -- time.Time is the one in
		// this model, behind Header.Date -- would render as an empty {} for
		// every value it can hold, making a typed field permanently invisible.
		// Formatting it is the only way to see inside. Structs that merely have
		// all their fields skipped are not this case and must keep rendering
		// empty, or the skip would not skip.
		if !specHasExportedField(v.Type()) {
			fmt.Fprintf(sb, "%v", v.Interface())
			return
		}
		sb.WriteString("{")
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() || specRawField(field.Name) {
				continue
			}
			sb.WriteString(field.Name)
			sb.WriteString("=")
			specRender(sb, v.Field(i), depth+1)
			sb.WriteString(";")
		}
		sb.WriteString("}")
	case reflect.Slice, reflect.Array:
		sb.WriteString("[")
		for i := 0; i < v.Len(); i++ {
			specRender(sb, v.Index(i), depth+1)
			sb.WriteString(",")
		}
		sb.WriteString("]")
	case reflect.Map:
		keys := make([]string, 0, v.Len())
		rendered := map[string]string{}
		iter := v.MapRange()
		for iter.Next() {
			var value strings.Builder
			specRender(&value, iter.Value(), depth+1)
			key := fmt.Sprint(iter.Key().Interface())
			keys = append(keys, key)
			rendered[key] = value.String()
		}
		sort.Strings(keys)
		sb.WriteString("<")
		for _, key := range keys {
			sb.WriteString(key)
			sb.WriteString("=")
			sb.WriteString(rendered[key])
			sb.WriteString(";")
		}
		sb.WriteString(">")
	default:
		fmt.Fprintf(sb, "%v", v.Interface())
	}
}

// specTSV reads a tab-separated specification file and returns its data rows.
//
// The header row is checked rather than assumed. Dropping row 1 unconditionally
// would eat a structure silently -- and the resulting failure says the decoder's
// coverage changed, which would send whoever re-vendored the files looking in
// entirely the wrong place.
func specTSV(t *testing.T, dir, name string, columns ...string) [][]string {
	t.Helper()

	f, err := os.Open(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	defer func() { _ = f.Close() }()

	r := csv.NewReader(f)
	r.Comma = '\t'
	r.FieldsPerRecord = len(columns)
	// A double quote inside a field is just a character here -- the transcribed
	// 5.5 files use them, quoting the specification text a defect appears in --
	// and LazyQuotes is what stops those from being an error.
	//
	// It does not make quoting meaningless: a field that *begins* with a quote
	// is still read as quoted, and would swallow the tabs after it into one
	// column. Nothing written by the transcription starts with one, and
	// specTSVFieldsAreUnquoted keeps it that way.
	r.LazyQuotes = true
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	if len(rows) == 0 {
		t.Fatalf("%s is empty", name)
	}
	if !slices.Equal(rows[0], columns) {
		t.Fatalf("%s: first row is %v, want the header %v -- regenerate it, or this "+
			"loses a structure", name, rows[0], columns)
	}
	specTSVFieldsAreUnquoted(t, name, rows)
	return rows[1:]
}

// specTSVFieldsAreUnquoted fails on a field that begins with a double quote,
// which csv.Reader reads as a quoted field even under LazyQuotes -- swallowing
// every tab until the closing quote and silently merging columns.
func specTSVFieldsAreUnquoted(t *testing.T, name string, rows [][]string) {
	t.Helper()

	for i, row := range rows {
		for _, field := range row {
			if strings.HasPrefix(field, `"`) {
				t.Fatalf("%s row %d: field %q begins with a double quote, which is read "+
					"as a quoted field and merges the columns after it", name, i+1, field)
			}
		}
	}
}

// specSourceFile parses a vendored SOURCE file into its "key: value" entries.
// Every vendored specification directory carries one, and it is the single
// place that directory's provenance is recorded.
func specSourceFile(t *testing.T, path string) map[string]string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	source := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			t.Fatalf("%s: line is not \"key: value\": %q", path, line)
		}
		source[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return source
}

// specDiff summarises the first few differing lines of two documents, which is
// enough to see what moved without printing a thousand-line table.
func specDiff(want, got string) string {
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")

	var sb strings.Builder
	shown := 0
	for i := 0; i < len(wantLines) || i < len(gotLines); i++ {
		var w, g string
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if w == g {
			continue
		}
		if shown == 10 {
			sb.WriteString("  ...\n")
			break
		}
		fmt.Fprintf(&sb, "  line %d:\n    -%s\n    +%s\n", i+1, w, g)
		shown++
	}
	return sb.String()
}

// specShare formats n out of total as a percentage.
func specShare(n, total int) string {
	if total == 0 {
		return "0.0%"
	}
	return fmt.Sprintf("%.1f%%", 100*float64(n)/float64(total))
}

// specProbesAreWellFormed checks that each probe document differs from its
// control in exactly one line, and that the line the classifier reads
// diagnostics for is that line. Every status depends on this: a probe that
// differs in two lines attributes both to one structure.
//
// name renders a pair for error messages, which each version does its own way.
func specProbesAreWellFormed(t *testing.T, graph *specGraph, form *specForm,
	name func(specPair) string) {
	t.Helper()

	for _, pair := range graph.pairs {
		path, ok := graph.pathFor(pair)
		if !ok {
			continue // reported by the coverage test
		}
		probe := form.probeFor(path)

		with := strings.Split(strings.TrimSuffix(probe.with, "\n"), "\n")
		without := strings.Split(strings.TrimSuffix(probe.without, "\n"), "\n")

		label := name(pair)
		if len(with) != len(without)+1 {
			t.Errorf("%s: probe has %d lines, control has %d; expected exactly one more",
				label, len(with), len(without))
			continue
		}
		if probe.line < 1 || probe.line > len(with) {
			t.Errorf("%s: probe line %d is outside the document's %d lines",
				label, probe.line, len(with))
			continue
		}

		removed := with[probe.line-1]
		rest := append(append([]string{}, with[:probe.line-1]...), with[probe.line:]...)
		if strings.Join(rest, "\n") != strings.Join(without, "\n") {
			t.Errorf("%s: control is not the probe minus line %d (%q)\nprobe:\n%s\ncontrol:\n%s",
				label, probe.line, removed, probe.with, probe.without)
			continue
		}
		// The tag follows the level, except on a record line where a
		// cross-reference identifier comes between them.
		if fields := strings.Fields(removed); len(fields) < 2 ||
			(fields[1] != pair.tag && (len(fields) < 3 || fields[2] != pair.tag)) {
			t.Errorf("%s: line %d is %q, which does not carry the tag under test",
				label, probe.line, removed)
			continue
		}
		// A removed line the control still contains verbatim cannot change
		// anything, so the probe would report a confident status having
		// measured nothing. This is how HEAD.GEDC and the top-level TRLR were
		// wrong: the harness injected boilerplate identical to the line under
		// test. It is also how the next such bug will announce itself.
		if slices.Contains(without, removed) {
			t.Errorf("%s: line %d (%q) is still present in the control, so removing "+
				"it measures nothing\nprobe:\n%s", label, probe.line, removed, probe.with)
		}
	}
}
