package encoder

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Issue #732: Family.ChildLinks and PersonalName.MarriedName are written as
// the vendor tags they are read from.

// TestEntityWritesChildLinksAndMarriedName checks a hand-built family and
// name: each link goes beneath its child's CHIL, a link naming no child is
// not written, and _MARNM goes beneath its NAME.
func TestEntityWritesChildLinksAndMarriedName(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version551},
		Records: []*gedcom.Record{
			{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{
				XRef:  "@I1@",
				Names: []*gedcom.PersonalName{{Full: "Ann /Smith/", MarriedName: "Jones"}},
			}},
			{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: &gedcom.Family{
				XRef:     "@F1@",
				Children: []string{"@I2@", "@I3@"},
				ChildLinks: []gedcom.ChildLink{
					{XRef: "@I3@", FatherRelation: "Adopted", MotherRelation: "Natural"},
					{XRef: "@I2@", MotherRelation: "Step"},
					{XRef: "@I9@", FatherRelation: "Foster"},
				},
			}},
		},
	}
	out := encodeToString(t, doc)
	for _, want := range []string{
		"1 NAME Ann /Smith/\n2 _MARNM Jones\n",
		"1 CHIL @I2@\n2 _MREL Step\n1 CHIL @I3@\n2 _FREL Adopted\n2 _MREL Natural\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Foster") {
		t.Errorf("a ChildLink naming no child was written:\n%s", out)
	}
}

// TestChildLinksRoundTripFromEntity checks that a decoded FTM family and a
// MyHeritage name written from their entities (Tags cleared) keep their
// relationships and married names, which were dropped before #732.
func TestChildLinksRoundTripFromEntity(t *testing.T) {
	for _, tc := range []struct{ path, xref string }{
		{"../testdata/edge-cases/relationships-complex.ged", "@F00003@"},
		{"../testdata/edge-cases/mhftb8-export.ged", "@I589@"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			doc := decodeFile(t, tc.path)
			before := doc.XRefMap[tc.xref].Entity
			clearRecordTags(doc)
			redoc := decodeGEDCOM(t, encodeToString(t, doc))
			after := redoc.XRefMap[tc.xref].Entity
			switch b := before.(type) {
			case *gedcom.Family:
				if got := after.(*gedcom.Family).ChildLinks; len(b.ChildLinks) == 0 || !reflect.DeepEqual(got, b.ChildLinks) {
					t.Errorf("ChildLinks after an entity round trip = %+v, want %+v", got, b.ChildLinks)
				}
			case *gedcom.Individual:
				if got := after.(*gedcom.Individual).Names[0].MarriedName; got == "" || got != b.Names[0].MarriedName {
					t.Errorf("MarriedName after an entity round trip = %q, want %q", got, b.Names[0].MarriedName)
				}
			}
		})
	}
}

// TestSyncTagsCarriesChildLinkEdit checks that editing a relationship and
// calling SyncTags reaches the output, keeping the CHIL's other lines.
func TestSyncTagsCarriesChildLinkEdit(t *testing.T) {
	doc := decodeFile(t, "../testdata/edge-cases/relationships-complex.ged")
	before := encodeToString(t, doc)
	doc.GetFamily("@F00003@").LinkForChild("@I00015@").MotherRelation = "Adopted"
	syncTags(t, doc)
	removed, added := lineDiff(before, encodeToString(t, doc))
	if !sameLineSet(removed, []string{"2 _MREL Step"}) || !sameLineSet(added, []string{"2 _MREL Adopted"}) {
		t.Errorf("SyncTags removed %q and added %q, want only _MREL Step -> Adopted", removed, added)
	}
}
