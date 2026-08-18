package decoder

// spec7_coverage_test.go derives what this library supports of GEDCOM 7.0, and
// publishes it as docs/reference/gedcom-7-coverage.md.
//
// CONSTITUTION.md names specification coverage as evidence for scoping work, on
// the grounds that a library calling itself the reference must be able to state
// its own coverage. Stating it is only worth anything if the statement is
// derived rather than estimated, so nothing here is hand-annotated: every entry
// comes from decoding a document built for it.
//
// The measurement, for each (superstructure, tag) pair the specification
// defines (see spec7_source_test.go for how the documents are built):
//
//   - decode a minimal document containing the structure, and the same document
//     with that one line removed;
//   - if the typed model differs, the structure has typed access -- the decoder
//     turned it into something reachable without walking raw tags;
//   - independently, record whether the decoder reported UNKNOWN_TAG for the
//     line, which distinguishes a gap it knows about from one it never looks at.
//
// "Typed model" means the header, the schema, the detected vendor, and each
// record's parsed entity. It excludes Record.Tags and every other Tags field,
// which is the raw half of ADR 0003's dual storage and holds every line
// regardless of support -- counting it would make everything look supported.
// Line numbers are excluded too, since removing a line shifts the ones after it.
//
// The generated document is checked in and compared on every run, so a change
// in what the decoder supports fails this test until the document is
// regenerated with `make spec-coverage`.

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// updateSpec7Coverage regenerates the checked-in coverage document instead of
// comparing against it. Run via `make spec-coverage`.
var updateSpec7Coverage = flag.Bool("update-spec-coverage", false,
	"rewrite docs/reference/gedcom-7-coverage.md from the decoder's actual behaviour")

// spec7DocPath is the published coverage document, relative to this package.
const spec7DocPath = "../docs/reference/gedcom-7-coverage.md"

// spec7Status is how the decoder handles one (superstructure, tag) pair.
type spec7Status string

const (
	// spec7Typed means the structure reaches the typed model.
	spec7Typed spec7Status = "typed"

	// spec7RawFlagged means the structure stays in raw tags only, and the
	// decoder reports it as an unknown tag. These are the sharpest gaps: the
	// decoder reads the context and does not understand the tag in it.
	spec7RawFlagged spec7Status = "raw (flagged)"

	// spec7RawAccepted means the structure stays in raw tags only, but the
	// context does report unknown tags -- the sentinel is flagged there -- and
	// this tag is deliberately not among them. A known structure with no typed
	// field yet.
	spec7RawAccepted spec7Status = "raw (accepted)"

	// spec7RawUndiagnosed means no unknown-tag diagnostic is emitted in this
	// context at all: not even the sentinel is reported. Silence about a
	// standard tag here therefore says nothing about whether the decoder knows
	// it.
	//
	// This does not mean the context is unparsed, and the harness cannot tell
	// the two causes apart. buildHeader reads every header line and types four
	// of them, yet buildDocument hands it no diagnosticCollector, so the whole
	// header is undiagnosed (issue #449). A context no parser visits looks
	// identical from out here (issue #448).
	spec7RawUndiagnosed spec7Status = "raw (undiagnosed)"

	// spec7Partial means the structure reaches the typed model and is still
	// reported as unknown -- a tag handled in part, or reported in error.
	spec7Partial spec7Status = "partial"
)

// spec7Entry is one row of the published inventory.
type spec7Entry struct {
	pair   spec7Pair
	path   []spec7Step
	status spec7Status
}

// context returns the dotted tag path of the entry's superstructure, which is
// where in a document the tag has to appear for this entry to apply.
func (e *spec7Entry) context() string {
	tags := make([]string, 0, len(e.path)-1)
	for _, step := range e.path[:len(e.path)-1] {
		tags = append(tags, step.tag)
	}
	if len(tags) == 0 {
		return "(record)"
	}
	return strings.Join(tags, ".")
}

// TestSpec7Coverage derives the GEDCOM 7.0 coverage inventory and checks the
// published document still matches it.
func TestSpec7Coverage(t *testing.T) {
	spec := loadSpec7(t)
	prober := newSpec7Prober(spec)

	entries := make([]spec7Entry, 0, len(spec.pairs))
	for _, pair := range spec.pairs {
		entry, ok := prober.measure(t, pair)
		if !ok {
			t.Errorf("superstructure %s is unreachable from any top-level structure",
				spec7Term(pair.superstructure))
			continue
		}
		entries = append(entries, entry)
	}

	got := spec7Document(spec, entries)

	if *updateSpec7Coverage {
		if err := os.WriteFile(spec7DocPath, []byte(got), 0o644); err != nil {
			t.Fatalf("write %s: %v", spec7DocPath, err)
		}
		t.Logf("wrote %s (%d entries)", spec7DocPath, len(entries))
		return
	}

	want, err := os.ReadFile(spec7DocPath)
	if err != nil {
		t.Fatalf("read %s (regenerate with `make spec-coverage`): %v", spec7DocPath, err)
	}
	if string(want) != got {
		t.Errorf("%s is out of date: the decoder's GEDCOM 7.0 coverage has changed.\n"+
			"Regenerate with `make spec-coverage` and review the diff.\n%s",
			spec7DocPath, spec7Diff(string(want), got))
	}
}

// spec7Prober measures pairs against the decoder, caching the one measurement
// that is shared between them.
type spec7Prober struct {
	spec *spec7Spec

	// watched records whether the decoder reads a superstructure's contents at
	// all. That is a property of the superstructure rather than of each tag
	// under it, so it is measured once per superstructure rather than once per
	// pair -- 122 sentinel probes instead of 1,389.
	watched map[string]bool
}

// newSpec7Prober returns a prober for spec.
func newSpec7Prober(spec *spec7Spec) *spec7Prober {
	return &spec7Prober{spec: spec, watched: map[string]bool{}}
}

// measure classifies one pair. The second result is false when the pair's
// superstructure cannot be reached from any top-level structure, which would
// mean the specification data does not describe a connected document.
func (p *spec7Prober) measure(t *testing.T, pair spec7Pair) (spec7Entry, bool) {
	t.Helper()

	path, ok := p.spec.pathFor(pair)
	if !ok {
		return spec7Entry{}, false
	}
	if _, measured := p.watched[pair.superstructure]; !measured {
		sentinel, reachable := p.spec.sentinelFor(pair.superstructure)
		if !reachable {
			return spec7Entry{}, false
		}
		p.watched[pair.superstructure] = spec7Flagged(t, sentinel)
	}
	return spec7Entry{
		pair:   pair,
		path:   path,
		status: spec7Classify(t, p.spec.probeFor(path), p.watched[pair.superstructure]),
	}, true
}

// spec7Diff summarises the first few differing lines of two documents, which is
// enough to see what moved without printing a thousand-line table.
func spec7Diff(want, got string) string {
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

// spec7Classify decodes both halves of a probe and reports how the decoder
// handled the one line that differs between them. watched says whether the
// decoder reads the enclosing context at all, measured with the sentinel tag.
func spec7Classify(t *testing.T, probe spec7Probe, watched bool) spec7Status {
	t.Helper()

	with, withDiagnostics := spec7Decode(t, probe.with)
	without, _ := spec7Decode(t, probe.without)

	typed := spec7TypedModel(with) != spec7TypedModel(without)
	flagged := spec7FlaggedIn(withDiagnostics, probe.line)

	if flagged && !watched {
		t.Fatalf("a tag was reported unknown in a context the sentinel tag is not "+
			"reported in; the sentinel is no longer a reliable probe:\n%s", probe.with)
	}

	switch {
	case typed && flagged:
		return spec7Partial
	case typed:
		return spec7Typed
	case flagged:
		return spec7RawFlagged
	case watched:
		return spec7RawAccepted
	default:
		return spec7RawUndiagnosed
	}
}

// spec7Flagged reports whether the decoder emitted UNKNOWN_TAG for the line a
// probe isolates.
func spec7Flagged(t *testing.T, probe spec7Probe) bool {
	t.Helper()

	_, diagnostics := spec7Decode(t, probe.with)
	return spec7FlaggedIn(diagnostics, probe.line)
}

// spec7FlaggedIn reports whether diagnostics contain an UNKNOWN_TAG for line.
func spec7FlaggedIn(diagnostics Diagnostics, line int) bool {
	for _, d := range diagnostics {
		if d.Code == CodeUnknownTag && d.Line == line {
			return true
		}
	}
	return false
}

// spec7Decode decodes a probe document in the lenient mode that collects
// diagnostics, which is the only mode that reports unknown tags.
func spec7Decode(t *testing.T, doc string) (*gedcom.Document, Diagnostics) {
	t.Helper()

	result, err := DecodeWithDiagnostics(bytes.NewReader([]byte(doc)), nil)
	if err != nil || result == nil {
		t.Fatalf("probe document failed to decode: %v\n%s", err, doc)
	}
	return result.Document, result.Diagnostics
}

// spec7MaxDepth bounds the typed-model rendering. The typed entities link by
// cross-reference string rather than by pointer, so there is nothing to cycle
// through; this only stops a future pointer field from hanging the test.
const spec7MaxDepth = 40

// spec7TypedModel renders everything a caller can reach without walking raw
// tags, as a string two documents can be compared by. Records with no parsed
// entity contribute nothing: at level 0 every line becomes a Record whether or
// not the decoder understands it, so including the wrapper would report every
// top-level structure as supported.
func spec7TypedModel(doc *gedcom.Document) string {
	var sb strings.Builder
	spec7Render(&sb, reflect.ValueOf(doc.Header), 0)
	spec7Render(&sb, reflect.ValueOf(doc.Schema), 0)
	sb.WriteString(string(doc.Vendor))
	for _, record := range doc.Records {
		if record.Entity == nil {
			continue
		}
		spec7Render(&sb, reflect.ValueOf(record.Entity), 0)
	}
	return sb.String()
}

// spec7RawField reports whether a struct field belongs to the raw half of the
// dual storage, or is otherwise not a statement about coverage.
func spec7RawField(name string) bool {
	// Tags is ADR 0003's raw storage: it holds every line regardless of
	// support. LineNumber shifts when the probe line is removed.
	return name == "Tags" || name == "LineNumber"
}

// spec7HasExportedField reports whether a struct type exposes any field at all.
func spec7HasExportedField(t reflect.Type) bool {
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).IsExported() {
			return true
		}
	}
	return false
}

// spec7Render writes a deterministic rendering of v, skipping raw-storage
// fields at every level.
func spec7Render(sb *strings.Builder, v reflect.Value, depth int) {
	if depth > spec7MaxDepth {
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
		spec7Render(sb, v.Elem(), depth+1)
	case reflect.Struct:
		// A struct with no exported fields at all -- time.Time is the one in
		// this model, behind Header.Date -- would render as an empty {} for
		// every value it can hold, making a typed field permanently invisible.
		// Formatting it is the only way to see inside. Structs that merely have
		// all their fields skipped are not this case and must keep rendering
		// empty, or the skip would not skip.
		if !spec7HasExportedField(v.Type()) {
			fmt.Fprintf(sb, "%v", v.Interface())
			return
		}
		sb.WriteString("{")
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() || spec7RawField(field.Name) {
				continue
			}
			sb.WriteString(field.Name)
			sb.WriteString("=")
			spec7Render(sb, v.Field(i), depth+1)
			sb.WriteString(";")
		}
		sb.WriteString("}")
	case reflect.Slice, reflect.Array:
		sb.WriteString("[")
		for i := 0; i < v.Len(); i++ {
			spec7Render(sb, v.Index(i), depth+1)
			sb.WriteString(",")
		}
		sb.WriteString("]")
	case reflect.Map:
		keys := make([]string, 0, v.Len())
		rendered := map[string]string{}
		iter := v.MapRange()
		for iter.Next() {
			var value strings.Builder
			spec7Render(&value, iter.Value(), depth+1)
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
