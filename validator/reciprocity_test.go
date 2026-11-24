package validator

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
)

// unreciprocatedIssues decodes input and returns the Unreciprocated* issues
// ReferenceValidator reports for it.
func unreciprocatedIssues(t *testing.T, input string) []Issue {
	t.Helper()
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	var out []Issue
	for _, issue := range NewReferenceValidator().Validate(doc) {
		if strings.HasPrefix(issue.Code, "UNRECIPROCATED_") {
			out = append(out, issue)
		}
	}
	return out
}

func TestReferenceValidator_Unreciprocated(t *testing.T) {
	const head = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n"
	tests := []struct {
		name  string
		input string
		code  string
		line  int
	}{
		{
			// The GRAMPS case from #929: a FAMS to a family naming other people.
			name: "FAMS the family does not reciprocate",
			input: head + `0 @I1@ INDI
1 FAMS @F1@
0 @I2@ INDI
1 FAMS @F1@
0 @F1@ FAM
1 HUSB @I2@
0 TRLR`,
			code: CodeUnreciprocatedFAMS,
			line: 6,
		},
		{
			name: "FAMC the family does not reciprocate",
			input: head + `0 @I1@ INDI
1 FAMC @F1@
0 @F1@ FAM
0 TRLR`,
			code: CodeUnreciprocatedFAMC,
			line: 6,
		},
		{
			name: "HUSB without a matching FAMS",
			input: head + `0 @I1@ INDI
0 @F1@ FAM
1 HUSB @I1@
0 TRLR`,
			code: CodeUnreciprocatedHUSB,
			line: 7,
		},
		{
			name: "WIFE without a matching FAMS",
			input: head + `0 @I1@ INDI
1 FAMC @F1@
0 @F1@ FAM
1 WIFE @I1@
1 CHIL @I1@
0 TRLR`,
			code: CodeUnreciprocatedWIFE,
			line: 8,
		},
		{
			name: "CHIL without a matching FAMC",
			input: head + `0 @I1@ INDI
0 @F1@ FAM
1 CHIL @I1@
0 TRLR`,
			code: CodeUnreciprocatedCHIL,
			line: 7,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := unreciprocatedIssues(t, tt.input)
			// Reported once, from the side that records the link.
			if len(issues) != 1 {
				t.Fatalf("issues = %v, want one %s", issues, tt.code)
			}
			got := issues[0]
			if got.Code != tt.code || got.Severity != SeverityWarning {
				t.Errorf("Code, Severity = %s, %v; want %s, warning", got.Code, got.Severity, tt.code)
			}
			if got.RecordXRef != "@I1@" || got.RelatedXRef != "@F1@" {
				t.Errorf("RecordXRef, RelatedXRef = %s, %s; want @I1@, @F1@", got.RecordXRef, got.RelatedXRef)
			}
			if got.LineNumber != tt.line {
				t.Errorf("LineNumber = %d, want %d", got.LineNumber, tt.line)
			}
		})
	}
}

func TestReferenceValidator_ReciprocatedReportsNone(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 FAMS @F1@
0 @I2@ INDI
1 FAMS @F1@
0 @I3@ INDI
1 FAMC @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
0 TRLR`
	if issues := unreciprocatedIssues(t, input); len(issues) != 0 {
		t.Errorf("issues = %v, want none", issues)
	}
}

// TestReferenceValidator_UnreciprocatedSkipsOrphans pins that a pointer to a
// record that does not exist is reported only as ORPHANED_*, and a null
// pointer not at all.
func TestReferenceValidator_UnreciprocatedSkipsOrphans(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 FAMS @F9@
1 FAMC @VOID@
0 @F1@ FAM
1 HUSB @I9@
1 CHIL @VOID@
0 TRLR`
	if issues := unreciprocatedIssues(t, input); len(issues) != 0 {
		t.Errorf("issues = %v, want none", issues)
	}
}
