package validator

import (
	"os"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// A GEDCOM 7.0 negative assertion (n NO <EVENT>) records that an event did NOT
// happen. It decodes into Individual.Events / Family.Events with IsNegative
// set, so every rule below would misread it as the event itself. Each test
// pairs the negated file with a control where the same event is asserted
// positively, proving the rule would have fired on the same data.

// negativeAssertion70 has @I1@ born 1920 and NOT dead between 1900 and 1910,
// @I2@ with only a NO BIRT (placed), and @F1@ NOT married between 1800 and
// 1850 -- all periods before the births, so a reader that ignores IsNegative
// sees impossible dates.
const negativeAssertion70 = `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME John /Doe/
1 SEX M
1 BIRT
2 DATE 1 JAN 1920
1 NO DEAT
2 DATE FROM 1900 TO 1910
1 FAMS @F1@
0 @I2@ INDI
1 NAME Jane /Doe/
1 SEX F
1 NO BIRT
2 DATE FROM 1800 TO 1810
2 PLAC Nowhere
1 FAMS @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 NO MARR
2 DATE FROM 1800 TO 1850
0 TRLR
`

func decodeText(t *testing.T, text string) *gedcom.Document {
	t.Helper()
	doc, err := decoder.Decode(strings.NewReader(text))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	return doc
}

// asserted turns every NO assertion in text into the positive event.
func asserted(text string) string {
	return strings.ReplaceAll(text, "1 NO ", "1 ")
}

func countCode(issues []Issue, code, xref string) int {
	n := 0
	for _, is := range issues {
		if is.Code == code && is.RecordXRef == xref {
			n++
		}
	}
	return n
}

func TestDateLogic_SkipsNegativeAssertions(t *testing.T) {
	v := NewDateLogicValidator(nil)

	negated := v.Validate(decodeText(t, negativeAssertion70))
	if n := countCode(negated, CodeDeathBeforeBirth, "@I1@"); n != 0 {
		t.Errorf("NO DEAT raised %d %s issue(s), want 0", n, CodeDeathBeforeBirth)
	}
	if n := countCode(negated, CodeMarriageBeforeBirth, "@I1@"); n != 0 {
		t.Errorf("NO MARR raised %d %s issue(s), want 0", n, CodeMarriageBeforeBirth)
	}

	// Control: the same dates asserted positively are impossible.
	positive := v.Validate(decodeText(t, asserted(negativeAssertion70)))
	if n := countCode(positive, CodeDeathBeforeBirth, "@I1@"); n != 1 {
		t.Errorf("control DEAT raised %d %s issue(s), want 1", n, CodeDeathBeforeBirth)
	}
	if n := countCode(positive, CodeMarriageBeforeBirth, "@I1@"); n != 1 {
		t.Errorf("control MARR raised %d %s issue(s), want 1", n, CodeMarriageBeforeBirth)
	}
}

// maximal70WithNegatedDeathAndMarriage returns maximal70.ged with a NO DEAT
// (dated before @I1@'s 2000 birth) placed ahead of @I1@'s real DEAT, and @F1@'s
// NO DIV retargeted to NO MARR (same 1700-1800 period). maximal70's own NO
// assertions (NATU, EMIG, DIV, ANUL) are not read by any date rule, so this
// is the smallest edit that makes the fixture exercise one.
func maximal70WithNegatedDeathAndMarriage(t *testing.T) (original, modified string) {
	t.Helper()
	data, err := os.ReadFile("../testdata/gedcom-7.0/maximal70.ged")
	if err != nil {
		t.Fatalf("read maximal70.ged: %v", err)
	}
	original = string(data)
	modified = strings.Replace(original, "0 @I1@ INDI\n", "0 @I1@ INDI\n1 NO DEAT\n2 DATE FROM 1700 TO 1800\n", 1)
	modified = strings.Replace(modified, "1 NO DIV\n", "1 NO MARR\n", 1)
	modified = strings.Replace(modified, "0 @I2@ INDI\n", "0 @I2@ INDI\n1 NO DEAT\n2 DATE FROM 1700 TO 1800\n", 1)
	if strings.Count(modified, "1 NO DEAT\n") != 2 || !strings.Contains(modified, "1 NO MARR\n") {
		t.Fatal("maximal70.ged no longer has the anchors this test edits")
	}
	return original, modified
}

func TestDateLogic_SkipsNegativeAssertions_Maximal70(t *testing.T) {
	original, modified := maximal70WithNegatedDeathAndMarriage(t)
	v := NewDateLogicValidator(nil)

	base := v.Validate(decodeText(t, original))
	got := v.Validate(decodeText(t, modified))
	for _, code := range []string{CodeDeathBeforeBirth, CodeMarriageBeforeBirth} {
		if b, g := countCode(base, code, "@I1@"), countCode(got, code, "@I1@"); g != b {
			t.Errorf("%s for @I1@: %d with negative assertions added, want %d (unchanged)", code, g, b)
		}
	}
}

func TestQuality_SkipsNegativeAssertions(t *testing.T) {
	report := NewQualityAnalyzer().Analyze(decodeText(t, negativeAssertion70))
	if report.IndividualsWithDeathDate != 0 {
		t.Errorf("IndividualsWithDeathDate = %d, want 0 (only a NO DEAT)", report.IndividualsWithDeathDate)
	}
	if report.IndividualsWithBirthDate != 1 {
		t.Errorf("IndividualsWithBirthDate = %d, want 1 (@I2@ has only a NO BIRT)", report.IndividualsWithBirthDate)
	}
	if report.IndividualsWithPlaces != 0 {
		t.Errorf("IndividualsWithPlaces = %d, want 0 (the only PLAC is under NO BIRT)", report.IndividualsWithPlaces)
	}
	if countCode(report.CompletenessIssues, CodeMissingBirthDate, "@I2@") != 1 {
		t.Errorf("want %s for @I2@, whose only BIRT is negated", CodeMissingBirthDate)
	}

	// Control: asserted positively, the same data counts.
	control := NewQualityAnalyzer().Analyze(decodeText(t, asserted(negativeAssertion70)))
	if control.IndividualsWithDeathDate != 1 || control.IndividualsWithBirthDate != 2 || control.IndividualsWithPlaces != 1 {
		t.Errorf("control counts = death %d, birth %d, places %d; want 1, 2, 1",
			control.IndividualsWithDeathDate, control.IndividualsWithBirthDate, control.IndividualsWithPlaces)
	}
}

func TestQuality_SkipsNegativeAssertions_Maximal70(t *testing.T) {
	original, modified := maximal70WithNegatedDeathAndMarriage(t)
	base := NewQualityAnalyzer().Analyze(decodeText(t, original))
	got := NewQualityAnalyzer().Analyze(decodeText(t, modified))
	if got.IndividualsWithDeathDate != base.IndividualsWithDeathDate {
		t.Errorf("IndividualsWithDeathDate = %d with a NO DEAT added to @I2@, want %d (unchanged)",
			got.IndividualsWithDeathDate, base.IndividualsWithDeathDate)
	}
}

// TestStreaming_SkipsNegativeAssertions pins the streaming path: a NO DEAT
// whose period precedes the birth is not a death, so StreamingValidator must
// not report DEATH_BEFORE_BIRTH for it. The control asserts the same period
// as a real death and expects the issue.
func TestStreaming_SkipsNegativeAssertions(t *testing.T) {
	run := func(t *testing.T, input string) []Issue {
		t.Helper()
		doc, err := decoder.Decode(strings.NewReader(input))
		if err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		sv := NewStreamingValidator(StreamingOptions{})
		var issues []Issue
		for _, rec := range doc.Records {
			issues = append(issues, sv.ValidateRecord(rec)...)
		}
		return append(issues, sv.Finalize()...)
	}
	count := func(issues []Issue) int {
		n := 0
		for _, iss := range issues {
			if iss.Code == CodeDeathBeforeBirth {
				n++
			}
		}
		return n
	}
	const negated = "0 HEAD\n1 GEDC\n2 VERS 7.0\n0 @I1@ INDI\n1 BIRT\n2 DATE 1 JAN 1920\n1 NO DEAT\n2 DATE FROM 1900 TO 1910\n0 TRLR\n"
	if n := count(run(t, negated)); n != 0 {
		t.Errorf("streaming DEATH_BEFORE_BIRTH for NO DEAT = %d, want 0", n)
	}
	control := strings.Replace(negated, "1 NO DEAT\n2 DATE FROM 1900 TO 1910", "1 DEAT\n2 DATE 1 JAN 1905", 1)
	if n := count(run(t, control)); n != 1 {
		t.Errorf("streaming DEATH_BEFORE_BIRTH for a real 1905 death = %d, want 1", n)
	}
}
