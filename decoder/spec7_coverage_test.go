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
// The measurement is spec_common_test.go's, and is described there. In outline,
// for each (superstructure, tag) pair the specification defines: decode a
// minimal document containing the structure and the same document with that one
// line removed, compare the typed models, and record whether the decoder
// reported UNKNOWN_TAG for the line.
//
// The generated document is checked in and compared on every run, so a change
// in what the decoder supports fails this test until the document is
// regenerated with `make spec-coverage`.

import (
	"flag"
	"os"
	"strings"
	"testing"
)

// updateSpec7Coverage regenerates the checked-in coverage document instead of
// comparing against it. Run via `make spec-coverage`.
var updateSpec7Coverage = flag.Bool("update-spec-coverage", false,
	"rewrite docs/reference/gedcom-7-coverage.md from the decoder's actual behaviour")

// spec7DocPath is the published coverage document, relative to this package.
const spec7DocPath = "../docs/reference/gedcom-7-coverage.md"

// spec7Entry is one row of the published inventory.
type spec7Entry struct {
	pair   specPair
	path   []specStep
	status specStatus
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
	prober := newSpecProber(spec.specGraph, spec.specForm)

	entries := make([]spec7Entry, 0, len(spec.pairs))
	for _, pair := range spec.pairs {
		path, status, ok := prober.measure(t, pair)
		if !ok {
			t.Errorf("superstructure %s is unreachable from any top-level structure",
				spec7Term(pair.superstructure))
			continue
		}
		entries = append(entries, spec7Entry{pair: pair, path: path, status: status})
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
			spec7DocPath, specDiff(string(want), got))
	}
}
