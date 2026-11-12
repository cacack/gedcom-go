package decoder

import (
	"os"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestEventPayloadDescription checks that a named event's own payload, with
// its CONT/CONC lines folded in, reaches Event.Description and that the
// continuation lines are not reported as unknown tags. The "Y" occurrence
// flag is not a description, and a 7.0 NO assertion has none.
// Ref: Issue #734
func TestEventPayloadDescription(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 RESI Age: 57; Occupation: Farmer
2 CONC ; Birthplace: Virginia
2 CONT second line
2 DATE 1860
1 DEAT Y
1 BIRT
1 EVEN Race
2 CONC horse
2 TYPE Hobby
0 @F1@ FAM
1 MARR Y
1 CENS Household of two
0 TRLR
`
	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	for _, d := range result.Diagnostics {
		if d.Code == CodeUnknownTag {
			t.Errorf("unexpected diagnostic on line %d: %s", d.Line, d.Message)
		}
	}

	indi := result.Document.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("individual @I1@ not decoded")
	}
	want := map[gedcom.EventType]string{
		gedcom.EventResidence: "Age: 57; Occupation: Farmer; Birthplace: Virginia\nsecond line",
		gedcom.EventDeath:     "",
		gedcom.EventBirth:     "",
		"EVEN":                "Racehorse",
	}
	if len(indi.Events) != len(want) {
		t.Fatalf("len(Events) = %d, want %d", len(indi.Events), len(want))
	}
	for _, ev := range indi.Events {
		if ev.Description != want[ev.Type] {
			t.Errorf("%s Description = %q, want %q", ev.Type, ev.Description, want[ev.Type])
		}
	}
	if ev := indi.Events[0]; ev.Date != "1860" {
		t.Errorf("RESI Date = %q, want %q (subordinates after continuations still parse)", ev.Date, "1860")
	}

	fam := result.Document.GetFamily("@F1@")
	if fam == nil {
		t.Fatal("family @F1@ not decoded")
	}
	famWant := map[gedcom.EventType]string{
		gedcom.EventMarriage: "",
		gedcom.EventCensus:   "Household of two",
	}
	for _, ev := range fam.Events {
		if ev.Description != famWant[ev.Type] {
			t.Errorf("FAM %s Description = %q, want %q", ev.Type, ev.Description, famWant[ev.Type])
		}
	}
}

// TestNegativeEventContinuationStillReported checks that a 7.0 NO assertion,
// whose payload is an event type, neither takes a Description nor silently
// accepts a continuation line.
// Ref: Issue #734
func TestNegativeEventContinuationStillReported(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NO MARR
2 CONT stray
0 TRLR
`
	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	indi := result.Document.GetIndividual("@I1@")
	if indi == nil || len(indi.Events) != 1 {
		t.Fatalf("want one event, got %+v", indi)
	}
	if ev := indi.Events[0]; !ev.IsNegative || ev.Description != "" {
		t.Errorf("NO MARR = {IsNegative:%v Description:%q}, want negative with no description", ev.IsNegative, ev.Description)
	}
	found := false
	for _, d := range result.Diagnostics {
		if d.Code == CodeUnknownTag && d.Line == 6 {
			found = true
		}
	}
	if !found {
		t.Errorf("CONT under NO not reported as %s; diagnostics: %v", CodeUnknownTag, result.Diagnostics)
	}
}

// TestAttributeValueContinuation checks that an attribute's CONT/CONC lines
// fold into Attribute.Value and are not reported as unknown tags.
// Ref: Issue #734
func TestAttributeValueContinuation(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 OCCU Farmer
2 CONT and carpenter
2 CONC s' mate
2 DATE 1870
0 TRLR
`
	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	for _, d := range result.Diagnostics {
		if d.Code == CodeUnknownTag {
			t.Errorf("unexpected diagnostic on line %d: %s", d.Line, d.Message)
		}
	}
	indi := result.Document.GetIndividual("@I1@")
	if indi == nil || len(indi.Attributes) != 1 {
		t.Fatalf("want one attribute, got %+v", indi)
	}
	attr := indi.Attributes[0]
	if want := "Farmer\nand carpenters' mate"; attr.Value != want {
		t.Errorf("OCCU Value = %q, want %q", attr.Value, want)
	}
	if attr.Date != "1870" {
		t.Errorf("OCCU Date = %q, want %q", attr.Date, "1870")
	}
}

// TestPres2020EventPayloads checks the real FTM export the issue was found
// on: RESI payloads with CONC continuations reach Description, and no CONC
// or CONT under an individual's event is reported as an unknown tag.
// Ref: Issue #734
func TestPres2020EventPayloads(t *testing.T) {
	f, err := os.Open("../testdata/gedcom-5.5/pres2020.ged")
	if err != nil {
		t.Skipf("fixture not available: %v", err)
	}
	defer f.Close()

	result, err := DecodeWithDiagnostics(f, nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	for _, d := range result.Diagnostics {
		if d.Code == CodeUnknownTag && (strings.Contains(d.Message, "CONC") || strings.Contains(d.Message, "CONT")) {
			t.Errorf("continuation still reported on line %d: %s", d.Line, d.Message)
		}
	}

	// The first RESI in the file splits "EnumerationDistrict" across its
	// CONC line; finding the whole word proves the continuation was folded.
	const want = "Age: 57; Occupation: Farmer; CannotRead: Yes; CannotWrite: Yes; " +
		"EnumerationDistrict: 193; MaritalStatus: Married; RelationToHead: Self"
	for _, rec := range result.Document.Records {
		indi, ok := rec.Entity.(*gedcom.Individual)
		if !ok {
			continue
		}
		for _, ev := range indi.Events {
			if ev.Type == gedcom.EventResidence && ev.Description == want {
				return
			}
		}
	}
	t.Errorf("no RESI event has Description %q", want)
}
