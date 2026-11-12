package decoder

import (
	"strings"
	"testing"
)

// TestSpouseAgesAndAttributeAge pins issue #736: FAMILY_EVENT_DETAIL's
// HUSB.AGE and WIFE.AGE on a family's events and attributes, and AGE on an
// attribute, decode into typed fields with their 7.0 PHRASE.
func TestSpouseAgesAndAttributeAge(t *testing.T) {
	const input = "0 HEAD\n1 GEDC\n2 VERS 7.0\n" +
		"0 @I1@ INDI\n" +
		"1 OCCU Smith\n2 AGE 30y\n3 PHRASE About thirty\n" +
		"0 @F1@ FAM\n" +
		"1 MARR\n2 DATE 1 JUN 1900\n2 HUSB\n3 AGE 25y\n4 PHRASE Twenty-five\n2 WIFE\n3 AGE 22y\n" +
		"1 FACT Farmers\n2 TYPE Livelihood\n2 HUSB\n3 AGE 40y\n2 WIFE\n3 AGE 38y\n4 PHRASE Late thirties\n" +
		"0 TRLR\n"

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	if len(result.Diagnostics) != 0 {
		t.Errorf("unexpected diagnostics: %v", result.Diagnostics)
	}
	occu := result.Document.GetIndividual("@I1@").Attributes[0]
	fam := result.Document.GetFamily("@F1@")
	marr, fact := fam.Events[0], fam.Attributes[0]

	tests := []struct {
		name, got, want string
	}{
		{"OCCU.AGE", occu.Age, "30y"},
		{"OCCU.AGE.PHRASE", occu.AgePhrase, "About thirty"},
		{"MARR.HUSB.AGE", marr.HusbandAge, "25y"},
		{"MARR.HUSB.AGE.PHRASE", marr.HusbandAgePhrase, "Twenty-five"},
		{"MARR.WIFE.AGE", marr.WifeAge, "22y"},
		{"MARR.WIFE.AGE.PHRASE", marr.WifeAgePhrase, ""},
		{"MARR.AGE", marr.Age, ""},
		{"FACT.HUSB.AGE", fact.HusbandAge, "40y"},
		{"FACT.WIFE.AGE", fact.WifeAge, "38y"},
		{"FACT.WIFE.AGE.PHRASE", fact.WifeAgePhrase, "Late thirties"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
		}
	}
}

// TestSpouseAgesNotOnIndividualEvents pins that HUSB/WIFE under an
// individual's event stay nonstandard (issue #475): reported, and not typed.
func TestSpouseAgesNotOnIndividualEvents(t *testing.T) {
	const input = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n" +
		"0 @I1@ INDI\n1 BIRT\n2 HUSB\n3 AGE 25y\n" +
		"0 TRLR\n"

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	birt := result.Document.GetIndividual("@I1@").Events[0]
	if birt.HusbandAge != "" {
		t.Errorf("BIRT HusbandAge = %q, want empty", birt.HusbandAge)
	}
	const husbLine = 6
	if !specFlaggedIn(result.Diagnostics, husbLine) {
		t.Errorf("HUSB under BIRT (line %d) not reported UNKNOWN_TAG: %v", husbLine, result.Diagnostics)
	}
}
