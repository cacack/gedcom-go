package decoder

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestEventAttributeLineNumber checks that each decoded event and attribute
// carries the line number of its own tag, so a caller can order a record's
// Events and Attributes against each other in file order: the RESI place
// before the BIRT place, and the OCCU and NCHI places before the event that
// follows them.
// Ref: Issue #964
func TestEventAttributeLineNumber(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 OCCU Farmer
2 PLAC Ohio
1 RESI
2 PLAC Boston
1 BIRT
2 PLAC Salem
1 FACT Tall
2 TYPE Height
0 @F1@ FAM
1 NCHI 2
2 PLAC Dayton
1 MARR
2 PLAC Akron
1 NO DIV
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
	if got := len(indi.Events); got != 2 {
		t.Fatalf("len(indi.Events) = %d, want 2", got)
	}
	if got := len(indi.Attributes); got != 2 {
		t.Fatalf("len(indi.Attributes) = %d, want 2", got)
	}
	occu, fact := indi.Attributes[0], indi.Attributes[1]
	resi, birt := indi.Events[0], indi.Events[1]
	for _, c := range []struct {
		name string
		got  int
		want int
	}{
		{"OCCU", occu.LineNumber, 6},
		{"RESI", resi.LineNumber, 8},
		{"BIRT", birt.LineNumber, 10},
		{"FACT", fact.LineNumber, 12},
	} {
		if c.got != c.want {
			t.Errorf("%s LineNumber = %d, want %d", c.name, c.got, c.want)
		}
	}
	if resi.LineNumber >= birt.LineNumber {
		t.Errorf("RESI place (line %d) should come before BIRT place (line %d)", resi.LineNumber, birt.LineNumber)
	}
	if occu.LineNumber >= resi.LineNumber {
		t.Errorf("OCCU attribute (line %d) should come before RESI event (line %d)", occu.LineNumber, resi.LineNumber)
	}

	fam := doc.GetFamily("@F1@")
	if fam == nil {
		t.Fatal("family @F1@ not decoded")
	}
	if len(fam.Attributes) != 1 || len(fam.Events) != 2 {
		t.Fatalf("family has %d attributes and %d events, want 1 and 2", len(fam.Attributes), len(fam.Events))
	}
	nchi, marr, div := fam.Attributes[0], fam.Events[0], fam.Events[1]
	if nchi.LineNumber != 15 || marr.LineNumber != 17 || div.LineNumber != 19 {
		t.Errorf("family LineNumbers = NCHI %d, MARR %d, NO DIV %d; want 15, 17, 19",
			nchi.LineNumber, marr.LineNumber, div.LineNumber)
	}
	if nchi.LineNumber >= marr.LineNumber {
		t.Errorf("NCHI place (line %d) should come before MARR place (line %d)", nchi.LineNumber, marr.LineNumber)
	}
}

// TestDecodeRecordsClearsEventLineNumbers checks that DecodeRecords gives the
// events and attributes of its records LineNumber 0, as it does their tags:
// fragment line numbers would otherwise read as lines of the target's source.
// Ref: Issue #964
func TestDecodeRecordsClearsEventLineNumbers(t *testing.T) {
	fragment := "0 @I1@ INDI\n1 OCCU Farmer\n1 BIRT\n0 @F1@ FAM\n1 NCHI 2\n1 MARR\n"
	target, err := Decode(strings.NewReader(wholeFile("5.5.1", "UTF-8", "")))
	if err != nil {
		t.Fatal(err)
	}
	records, _, err := DecodeRecords(strings.NewReader(fragment), target, nil)
	if err != nil {
		t.Fatalf("DecodeRecords: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	indi, ok := records[0].Entity.(*gedcom.Individual)
	if !ok || len(indi.Events) != 1 || len(indi.Attributes) != 1 {
		t.Fatalf("individual not decoded with one event and one attribute: %#v", records[0].Entity)
	}
	fam, ok := records[1].Entity.(*gedcom.Family)
	if !ok || len(fam.Events) != 1 || len(fam.Attributes) != 1 {
		t.Fatalf("family not decoded with one event and one attribute: %#v", records[1].Entity)
	}
	for name, n := range map[string]int{
		"INDI event":     indi.Events[0].LineNumber,
		"INDI attribute": indi.Attributes[0].LineNumber,
		"FAM event":      fam.Events[0].LineNumber,
		"FAM attribute":  fam.Attributes[0].LineNumber,
	} {
		if n != 0 {
			t.Errorf("%s LineNumber = %d, want 0", name, n)
		}
	}
}
