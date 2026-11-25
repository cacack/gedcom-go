package merge

import (
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestFullValue pins fullValue to Record.FullValue, a CONT or CONC deeper
// than one level below included, and checks a value with no continuation
// allocates nothing.
func TestFullValue(t *testing.T) {
	tests := [][]*gedcom.Tag{
		{{Level: 2, Tag: "PLAC", Value: "Ohio"}},
		{{Level: 2, Tag: "PLAC", Value: "Spring"}, {Level: 3, Tag: "CONC", Value: "field"}, {Level: 3, Tag: "MAP"}},
		{{Level: 1, Tag: "NOTE", Value: "a"}, {Level: 2, Tag: "CONT", Value: "b"}, {Level: 2, Tag: "SOUR", Value: "@S1@"}, {Level: 3, Tag: "CONC", Value: "x"}, {Level: 2, Tag: "CONC", Value: "c"}},
		{{Level: 2, Tag: "PLAC", Value: "Ohio"}, {Level: 4, Tag: "CONC", Value: " extra"}},
	}
	for _, tags := range tests {
		n := buildTree(tags)[0]
		if got, want := fullValue(n), (&gedcom.Record{Tags: tags}).FullValue(tags[0]); got != want {
			t.Errorf("fullValue(%s) = %q, want %q", tags[0].Value, got, want)
		}
	}
	n := buildTree(tests[0])[0]
	if allocs := testing.AllocsPerRun(10, func() { _ = fullValue(n) }); allocs != 0 {
		t.Errorf("fullValue with no continuation: %v allocs, want 0", allocs)
	}
}
