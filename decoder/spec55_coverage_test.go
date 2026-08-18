package decoder

// spec55_coverage_test.go derives what this library supports of GEDCOM 5.5 and
// 5.5.1, and publishes it as docs/reference/gedcom-5.5-coverage.md.
//
// It is the sibling of spec7_coverage_test.go and makes the same measurement,
// described in spec_common_test.go: for each (superstructure, tag) pair the
// grammar defines, decode a minimal document containing the structure and the
// same document with that one line removed, compare the typed models, and
// record whether the decoder reported UNKNOWN_TAG for the line.
//
// The two versions are measured separately and reported together. They share
// most of their grammar, so a table that listed them apart would be nine parts
// repetition; where they agree, one row says so, and where they differ -- 5.5.1
// adding EMAIL, FAX, WWW and the phonetic and romanized name variations, or
// dropping 5.5's BLOB -- the row shows the difference.
//
// The generated document is checked in and compared on every run, so a change
// in what the decoder supports fails this test until the document is
// regenerated with `make spec-coverage`.

import (
	"sort"
	"testing"
)

// spec55DocPath is the published coverage document, relative to this package.
const spec55DocPath = "../docs/reference/gedcom-5.5-coverage.md"

// spec55Measurement is what one version's probe found for one pair.
type spec55Measurement struct {
	status      specStatus
	cardinality string
	path        []specStep
}

// spec55Row is one row of the published inventory: a (superstructure, tag,
// structure) triple, with what each version that defines it does.
type spec55Row struct {
	superstructure string
	tag            string
	structure      string
	measured       map[string]spec55Measurement
}

// context returns the dotted tag path the row's structure is reached at, for
// the given version, which is where in a document the tag has to appear for
// the row to apply.
func (r *spec55Row) context(version string) string {
	m, ok := r.measured[version]
	if !ok || len(m.path) == 0 {
		return ""
	}
	tags := make([]string, 0, len(m.path)-1)
	for _, step := range m.path[:len(m.path)-1] {
		tags = append(tags, step.tag)
	}
	if len(tags) == 0 {
		return "(record)"
	}
	return joinTags(tags)
}

// root returns the level-0 tag the row was measured under, for the given
// version, or the empty string if that version does not define it.
func (r *spec55Row) root(version string) string {
	m, ok := r.measured[version]
	if !ok || len(m.path) == 0 {
		return ""
	}
	return m.path[0].tag
}

// TestSpec55Coverage derives the GEDCOM 5.5 and 5.5.1 coverage inventory and
// checks the published document still matches it.
func TestSpec55Coverage(t *testing.T) {
	specs := map[string]*spec55Spec{}
	rows := map[[3]string]*spec55Row{}
	var order [][3]string

	for _, edition := range spec55Editions {
		spec := loadSpec55(t, edition)
		specs[edition.version] = spec
		prober := newSpecProber(spec.specGraph, spec.specForm)

		for _, pair := range spec.pairs {
			path, status, ok := prober.measure(t, pair)
			if !ok {
				t.Errorf("%s: superstructure %s is unreachable from any top-level structure",
					edition.version, pair.superstructure)
				continue
			}
			key := [3]string{pair.superstructure, pair.tag, pair.structure}
			row, seen := rows[key]
			if !seen {
				row = &spec55Row{
					superstructure: pair.superstructure,
					tag:            pair.tag,
					structure:      pair.structure,
					measured:       map[string]spec55Measurement{},
				}
				rows[key] = row
				order = append(order, key)
			}
			row.measured[edition.version] = spec55Measurement{
				status:      status,
				cardinality: spec.cardinality(pair),
				path:        path,
			}
		}
	}

	sort.Slice(order, func(i, j int) bool {
		// Level 0 first; it is the entry point to everything else and its
		// superstructure is not a structure at all.
		if (order[i][0] == "") != (order[j][0] == "") {
			return order[i][0] == ""
		}
		for k := range order[i] {
			if order[i][k] != order[j][k] {
				return order[i][k] < order[j][k]
			}
		}
		return false
	})

	ordered := make([]*spec55Row, 0, len(order))
	for _, key := range order {
		ordered = append(ordered, rows[key])
	}

	specPublish(t, spec55DocPath, spec55Document(specs, ordered))
}
