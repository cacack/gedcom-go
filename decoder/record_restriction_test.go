package decoder

import (
	"strings"
	"testing"
)

// TestRecordRestriction pins INDI.RESN and FAM.RESN as typed fields (#739):
// the record-level restriction notice decodes into Restriction, as the
// event-level one already did, in 5.5.1 spelling and as a 7.0 list, and is
// still not reported as UNKNOWN_TAG.
func TestRecordRestriction(t *testing.T) {
	tests := []struct {
		name, version, indi, fam string
	}{
		{"5.5.1", "5.5.1", "privacy", "confidential"},
		{"7.0 list", "7.0", "CONFIDENTIAL, LOCKED", "LOCKED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := "0 HEAD\n1 GEDC\n2 VERS " + tt.version + "\n" +
				"0 @I1@ INDI\n1 RESN " + tt.indi + "\n1 NAME John /Smith/\n" +
				"0 @F1@ FAM\n1 RESN " + tt.fam + "\n1 HUSB @I1@\n0 TRLR\n"
			result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
			if err != nil {
				t.Fatalf("DecodeWithDiagnostics() error = %v", err)
			}
			for _, diag := range result.Diagnostics {
				if diag.Code == CodeUnknownTag {
					t.Errorf("unexpected UNKNOWN_TAG: %s", diag.Message)
				}
			}
			if got := result.Document.GetIndividual("@I1@").Restriction; got != tt.indi {
				t.Errorf("Individual.Restriction = %q, want %q", got, tt.indi)
			}
			if got := result.Document.GetFamily("@F1@").Restriction; got != tt.fam {
				t.Errorf("Family.Restriction = %q, want %q", got, tt.fam)
			}
		})
	}
}
