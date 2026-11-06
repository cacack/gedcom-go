package decoder

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestPhraseTypedAccess pins GEDCOM 7.0 PHRASE typed access (issue #450): a
// PHRASE qualifying a typed value lands in the <Value>Phrase field beside it.
func TestPhraseTypedAccess(t *testing.T) {
	const input = "0 HEAD\n1 GEDC\n2 VERS 7.0\n" +
		"0 @I1@ INDI\n" +
		"1 NAME Alex /Example/\n2 TYPE OTHER\n3 PHRASE Religious name\n" +
		"1 FAMC @F1@\n2 PEDI OTHER\n3 PHRASE Raised by an aunt\n" +
		"1 ADOP\n2 AGE 3y\n3 PHRASE About three\n" +
		"2 SDATE 1900\n3 PHRASE Sorted by registration\n" +
		"1 ASSO @I2@\n2 ROLE OTHER\n3 PHRASE Village elder\n" +
		"1 OCCU Smith\n2 SDATE 1910\n3 PHRASE Sorted by census\n" +
		"0 @F1@ FAM\n" +
		"1 HUSB @I3@\n2 PHRASE Stepfather\n" +
		"1 WIFE @VOID@\n2 PHRASE Unknown mother\n" +
		"0 @O1@ OBJE\n1 FILE photo.jpg\n2 FORM image/jpeg\n3 MEDI OTHER\n4 PHRASE Tintype\n" +
		"0 TRLR\n"

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	if len(result.Diagnostics) != 0 {
		t.Errorf("unexpected diagnostics: %v", result.Diagnostics)
	}
	doc := result.Document
	indi := doc.GetIndividual("@I1@")
	fam := doc.GetFamily("@F1@")
	media := doc.GetMediaObject("@O1@")
	adop := indi.Events[0]

	tests := []struct {
		name, got, want string
	}{
		{"INDI.NAME.TYPE", indi.Names[0].TypePhrase, "Religious name"},
		{"INDI.FAMC.PEDI", indi.ChildInFamilies[0].PedigreePhrase, "Raised by an aunt"},
		{"INDI.ADOP.AGE", adop.AgePhrase, "About three"},
		{"INDI.ADOP.SDATE", adop.SortDatePhrase, "Sorted by registration"},
		{"INDI.ASSO.ROLE", indi.Associations[0].RolePhrase, "Village elder"},
		{"INDI.OCCU.SDATE", indi.Attributes[0].SortDatePhrase, "Sorted by census"},
		{"FAM.HUSB", fam.HusbandPhrase, "Stepfather"},
		{"FAM.WIFE", fam.WifePhrase, "Unknown mother"},
		{"OBJE.FILE.FORM.MEDI", media.Files[0].MediaPhrase, "Tintype"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s PHRASE = %q, want %q", tt.name, tt.got, tt.want)
		}
	}
	// ASSO's own PHRASE and ROLE's PHRASE are distinct structures.
	if got := indi.Associations[0].Phrase; got != "" {
		t.Errorf("ROLE.PHRASE leaked into Association.Phrase = %q", got)
	}
}

// TestPhraseOnlyDate pins that GEDCOM 7.0's empty DateValue with a PHRASE
// child decodes to a phrase date without a diagnostic (issue #450).
func TestPhraseOnlyDate(t *testing.T) {
	const input = "0 HEAD\n1 GEDC\n2 VERS 7.0\n" +
		"0 @I1@ INDI\n1 BIRT\n2 DATE\n3 PHRASE 5 January (year unknown)\n" +
		"1 DEAT\n2 DATE\n0 TRLR\n"

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	indi := result.Document.GetIndividual("@I1@")
	want := &gedcom.Date{Phrase: "5 January (year unknown)", IsPhrase: true}
	got := indi.Events[0].ParsedDate
	if got == nil || got.Phrase != want.Phrase || !got.IsPhrase || got.Year != 0 || got.Original != "" {
		t.Errorf("BIRT ParsedDate = %+v, want %+v", got, want)
	}

	// An empty DATE with no PHRASE still has nothing to say and is reported.
	if indi.Events[1].ParsedDate != nil {
		t.Errorf("DEAT ParsedDate = %+v, want nil", indi.Events[1].ParsedDate)
	}
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != CodeInvalidValue {
		t.Errorf("diagnostics = %v, want one INVALID_VALUE for the bare DEAT DATE", result.Diagnostics)
	}
}
