package decoder

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestSubmissionParsing pins #452: a SUBN record decodes to a typed
// gedcom.Submission carrying every substructure GEDCOM 5.5.1 defines.
func TestSubmissionParsing(t *testing.T) {
	input := `0 HEAD
1 SOUR Test
1 SUBM @U1@
1 SUBN @SN1@
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @U1@ SUBM
1 NAME Submitter
0 @N1@ NOTE Shared
0 @SN1@ SUBN
1 SUBM @U1@
1 FAMF family.ged
1 TEMP SLAKE
1 ANCE 3
1 DESC 2
1 ORDI yes
1 RIN 42
1 NOTE Inline submission note
2 CONT second line
1 NOTE @N1@
1 CHAN
2 DATE 1 JAN 2000
3 TIME 12:00:00
1 _CUSTOM kept raw
0 TRLR
`
	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	doc := result.Document
	if len(result.Diagnostics) != 0 {
		t.Errorf("unexpected diagnostics: %v", result.Diagnostics)
	}

	subn := doc.GetSubmission("@SN1@")
	if subn == nil {
		t.Fatal("GetSubmission(@SN1@) = nil, want a typed Submission")
	}
	want := gedcom.Submission{
		XRef:                  "@SN1@",
		SubmitterXRef:         "@U1@",
		FamilyFileName:        "family.ged",
		TempleCode:            "SLAKE",
		AncestorGenerations:   "3",
		DescendantGenerations: "2",
		OrdinanceProcessFlag:  "yes",
		AutomatedRecordID:     "42",
	}
	got := *subn
	if got.XRef != want.XRef || got.SubmitterXRef != want.SubmitterXRef ||
		got.FamilyFileName != want.FamilyFileName || got.TempleCode != want.TempleCode ||
		got.AncestorGenerations != want.AncestorGenerations ||
		got.DescendantGenerations != want.DescendantGenerations ||
		got.OrdinanceProcessFlag != want.OrdinanceProcessFlag ||
		got.AutomatedRecordID != want.AutomatedRecordID {
		t.Errorf("Submission scalars = %+v, want %+v", got, want)
	}
	if len(subn.InlineNotes) != 1 || subn.InlineNotes[0] != "Inline submission note\nsecond line" {
		t.Errorf("InlineNotes = %q", subn.InlineNotes)
	}
	if len(subn.NoteXRefs) != 1 || subn.NoteXRefs[0] != "@N1@" {
		t.Errorf("NoteXRefs = %q", subn.NoteXRefs)
	}
	if subn.ChangeDate == nil || subn.ChangeDate.Date != "1 JAN 2000" || subn.ChangeDate.Time != "12:00:00" {
		t.Errorf("ChangeDate = %+v", subn.ChangeDate)
	}
	if got := subn.AllNotes(doc); len(got) != 2 || got[1] != "Shared" {
		t.Errorf("AllNotes = %q", got)
	}
	if len(subn.Tags) == 0 || subn.Tags[len(subn.Tags)-1].Tag != "_CUSTOM" {
		t.Errorf("Tags do not hold the raw record: %v", subn.Tags)
	}
	if subs := doc.Submissions(); len(subs) != 1 || subs[0] != subn {
		t.Errorf("Submissions() = %v, want [subn]", subs)
	}
}

// TestSubmissionUnknownTag pins that a nonstandard tag inside SUBN is now
// diagnosed, as it is in every other record.
func TestSubmissionUnknownTag(t *testing.T) {
	input := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @SN1@ SUBN\n1 BOGUS x\n0 TRLR\n"
	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	var found bool
	for _, d := range result.Diagnostics {
		if d.Code == CodeUnknownTag && strings.Contains(d.Message, "BOGUS") {
			found = true
		}
	}
	if !found {
		t.Errorf("no unknown-tag diagnostic for BOGUS under SUBN: %v", result.Diagnostics)
	}
}
