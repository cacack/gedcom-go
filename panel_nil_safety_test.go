package gedcomgo

import (
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// nil_safety_test.go drives every nil shape through the facade on a fixture
// built around one individual. Duplicate detection needs two before it runs at
// all, so that fixture cannot reach the readers that take a name positionally
// -- which is where a nil element in Names lands. The per-package suites pin
// the guards themselves; this pins the fact that ValidateAll is a way in.
func TestFacade_ValidateAll_NilPersonalNameElement(t *testing.T) {
	build := func(names []*gedcom.PersonalName) *gedcom.Document {
		individual := func(xref string, names []*gedcom.PersonalName) *gedcom.Record {
			return &gedcom.Record{
				XRef:   xref,
				Type:   gedcom.RecordTypeIndividual,
				Tags:   []*gedcom.Tag{{Level: 1, Tag: "NAME", Value: "Ada /Lovelace/"}},
				Entity: &gedcom.Individual{XRef: xref, Names: names},
			}
		}
		twin := []*gedcom.PersonalName{{Full: "Ada /Lovelace/"}}
		records := []*gedcom.Record{individual("@I1@", names), individual("@I2@", twin)}

		doc := &gedcom.Document{
			Header:  &gedcom.Header{Version: gedcom.Version551},
			Records: records,
			XRefMap: make(map[string]*gedcom.Record, len(records)),
		}
		for _, record := range records {
			doc.XRefMap[record.XRef] = record
		}
		return doc
	}

	got := ValidateAll(build([]*gedcom.PersonalName{nil, {Full: "Ada /Lovelace/"}}))
	want := ValidateAll(build([]*gedcom.PersonalName{{Full: "Ada /Lovelace/"}}))

	if len(want) == 0 {
		t.Fatal("nil-free document reported no issues; the comparison would pass vacuously")
	}
	if len(got) != len(want) {
		t.Errorf("ValidateAll issues = %d, want %d (must match the nil-free document)", len(got), len(want))
	}
}
