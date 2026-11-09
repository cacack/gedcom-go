package decoder

import (
	"os"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Issue #715: _UID, the vendor spelling of GEDCOM 7.0's UID that every 5.5.x
// desktop producer writes, fills the same UIDs fields.

// TestVendorUIDFillsUIDs pins the counts from three real exports. The
// expected numbers are the level-1 _UID lines per record type in each file.
func TestVendorUIDFillsUIDs(t *testing.T) {
	tests := []struct {
		path         string
		producer     string
		indi, fam    int
		eventAndAttr int
	}{
		{"../testdata/edge-cases/mhftb8-export.ged", "MyHeritage Family Tree Builder", 4683, 2863, 11607},
		{"../testdata/edge-cases/vendor-paf5.ged", "PAF 5", 33, 14, 0},
		{"../testdata/edge-cases/rootsmagic-2026-export.ged", "RootsMagic", 14, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.producer, func(t *testing.T) {
			f, err := os.Open(tt.path) // #nosec G304 -- path is a constant in the repo's own testdata
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			doc, err := Decode(f)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}

			var indi, fam, nested int
			for _, rec := range doc.Records {
				switch e := rec.Entity.(type) {
				case *gedcom.Individual:
					indi += len(e.UIDs)
					for _, ev := range e.Events {
						nested += len(ev.UIDs)
					}
					for _, at := range e.Attributes {
						nested += len(at.UIDs)
					}
					if !hasLevel1(rec, "_UID") || len(e.UIDs) == 0 {
						continue
					}
					// The raw tag stays in Tags (ADR 0003).
					if countLevel1(rec, "_UID") != len(e.UIDs) {
						t.Errorf("%s: %d _UID tags but %d UIDs", rec.XRef, countLevel1(rec, "_UID"), len(e.UIDs))
					}
				case *gedcom.Family:
					fam += len(e.UIDs)
					for _, ev := range e.Events {
						nested += len(ev.UIDs)
					}
				}
			}
			if indi != tt.indi || fam != tt.fam || nested != tt.eventAndAttr {
				t.Errorf("UIDs filled: INDI %d FAM %d event/attribute %d, want %d, %d, %d",
					indi, fam, nested, tt.indi, tt.fam, tt.eventAndAttr)
			}
			if doc.RequiresGEDCOM7() {
				t.Error("RequiresGEDCOM7() = true for a 5.5.x export whose only 7.0-looking feature is _UID")
			}
		})
	}
}

func hasLevel1(rec *gedcom.Record, name string) bool { return countLevel1(rec, name) > 0 }

func countLevel1(rec *gedcom.Record, name string) int {
	n := 0
	for _, tag := range rec.Tags {
		if tag != nil && tag.Level == 1 && tag.Tag == name {
			n++
		}
	}
	return n
}

// TestVendorUIDOnEveryRecordType covers the record types that have a UIDs
// field, and that SUBM, which has none, keeps _UID in Tags only.
func TestVendorUIDOnEveryRecordType(t *testing.T) {
	doc, err := Decode(strings.NewReader(strings.Join([]string{
		"0 HEAD",
		"1 GEDC",
		"2 VERS 5.5.1",
		"2 FORM LINEAGE-LINKED",
		"1 CHAR UTF-8",
		"0 @I1@ INDI",
		"1 _UID I-1",
		"1 UID I-2",
		"1 BIRT",
		"2 _UID E-1",
		"0 @F1@ FAM",
		"1 _UID F-1",
		"0 @S1@ SOUR",
		"1 _UID S-1",
		"0 @M1@ OBJE",
		"1 FILE a.jpg",
		"2 FORM JPG",
		"1 _UID M-1",
		"0 @U1@ SUBM",
		"1 NAME Sub",
		"1 _UID U-1",
		"0 TRLR",
	}, "\n") + "\n"))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	indi := doc.GetIndividual("@I1@")
	if got := strings.Join(indi.UIDs, ","); got != "I-1,I-2" {
		t.Errorf("Individual.UIDs = %q, want both spellings in file order", got)
	}
	if len(indi.Events) != 1 || strings.Join(indi.Events[0].UIDs, ",") != "E-1" {
		t.Errorf("Event.UIDs = %v, want [E-1]", indi.Events)
	}
	if got := doc.GetFamily("@F1@").UIDs; len(got) != 1 || got[0] != "F-1" {
		t.Errorf("Family.UIDs = %v", got)
	}
	if got := doc.GetSource("@S1@").UIDs; len(got) != 1 || got[0] != "S-1" {
		t.Errorf("Source.UIDs = %v", got)
	}
	if got := doc.GetMediaObject("@M1@").UIDs; len(got) != 1 || got[0] != "M-1" {
		t.Errorf("MediaObject.UIDs = %v", got)
	}
	subm := doc.XRefMap["@U1@"]
	if subm == nil || countLevel1(subm, "_UID") != 1 {
		t.Errorf("SUBM _UID should stay in Tags: %+v", subm)
	}
}
