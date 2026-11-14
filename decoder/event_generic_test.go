package decoder

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestGenericEventType guards issue #775: a generic EVEN on an individual and
// on a family decodes into an event whose Type is gedcom.EventGeneric.
func TestGenericEventType(t *testing.T) {
	const input = `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 EVEN Won the county fair
2 TYPE Award
1 FAMS @F1@
0 @F1@ FAM
1 HUSB @I1@
1 EVEN Moved the household
2 TYPE Relocation
0 TRLR
`
	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("individual @I1@ not decoded")
	}
	fam := doc.GetFamily("@F1@")
	if fam == nil {
		t.Fatal("family @F1@ not decoded")
	}

	tests := []struct {
		name       string
		events     []*gedcom.Event
		wantDetail string
	}{
		{"INDI", indi.Events, "Award"},
		{"FAM", fam.Events, "Relocation"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.events) != 1 {
				t.Fatalf("len(Events) = %d, want 1", len(tt.events))
			}
			ev := tt.events[0]
			if ev.Type != gedcom.EventGeneric {
				t.Errorf("Type = %q, want gedcom.EventGeneric (%q)", ev.Type, gedcom.EventGeneric)
			}
			if ev.EventTypeDetail != tt.wantDetail {
				t.Errorf("EventTypeDetail = %q, want %q", ev.EventTypeDetail, tt.wantDetail)
			}
		})
	}
}
