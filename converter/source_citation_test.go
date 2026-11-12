package converter

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestUpgradePromotesInlineSourceCitation: on an upgrade to 7.0 an inline
// citation becomes a pointer to a new SOUR record holding its description,
// its TEXT moves under DATA, and an empty citation points to @VOID@ (issues
// #738, #793).
func TestUpgradePromotesInlineSourceCitation(t *testing.T) {
	const doc551 = `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @S1@ SUBM
1 NAME Submitter
0 @I1@ INDI
1 SOUR Family bible in poss
2 CONC ession of
2 CONT John Smith
2 TEXT Born 1850
2 QUAY 1
2 TEXT in Ohio
1 BIRT
2 SOUR Parish register
3 DATA
4 DATE 1850
3 TEXT Baptised
1 SOUR
2 _APID 1,7545::1
1 SOUR @S2@
0 @S2@ SOUR
1 TITL Census
0 TRLR
`
	for _, from := range []string{"5.5.1", "5.5"} {
		t.Run(from, func(t *testing.T) {
			input := strings.Replace(doc551, "2 VERS 5.5.1", "2 VERS "+from, 1)
			converted, report, out := convertText(t, input, gedcom.Version70)

			for _, want := range []string{
				"0 @S3@ SOUR\n1 NOTE Family bible in possession of\n2 CONT John Smith\n" +
					"0 @S4@ SOUR\n1 NOTE Parish register\n0 @I1@ INDI\n",
				"1 SOUR @S3@\n2 DATA\n3 TEXT Born 1850\n3 TEXT in Ohio\n2 QUAY 1\n",
				"2 SOUR @S4@\n3 DATA\n4 DATE 1850\n4 TEXT Baptised\n",
				"1 SOUR @VOID@\n2 _APID 1,7545::1\n",
				"1 SOUR @S2@\n",
			} {
				if !strings.Contains(out, want) {
					t.Errorf("output lacks %q:\n%s", want, out)
				}
			}

			notes := map[string]string{}
			for _, n := range report.Normalized {
				notes[n.Path+" "+n.Original] = n.Result
			}
			for key, result := range map[string]string{
				"Individual @I1@ > SOUR SOUR (inline source citation)":        "SOUR @S3@",
				"Individual @I1@ > BIRT > SOUR SOUR (inline source citation)": "SOUR @S4@",
				"Individual @I1@ > SOUR SOUR (empty)":                         "SOUR @VOID@",
			} {
				if notes[key] != result {
					t.Errorf("Normalized %q = %q, want %q (all: %v)", key, notes[key], result, notes)
				}
			}
			counts := map[string]int{}
			for _, tr := range report.Transformations {
				counts[tr.Type] = tr.Count
			}
			if counts["INLINE_SOUR_TO_RECORD"] != 2 || counts["EMPTY_SOUR_TO_VOID"] != 1 {
				t.Errorf("transformations = %v", counts)
			}

			// The typed model follows: the citation points at a record whose
			// entity is a Source.
			cite := converted.GetIndividual("@I1@").SourceCitations[0]
			if cite.SourceXRef != "@S3@" || cite.InlineText != "" {
				t.Errorf("citation = %q / %q", cite.SourceXRef, cite.InlineText)
			}
			if src := converted.GetSource("@S3@"); src == nil {
				t.Error("promoted record has no Source entity")
			}
		})
	}
}

// TestUpgradeWithoutInlineSourceCitationAddsNothing: pointer citations are
// left alone and no transformation is reported.
func TestUpgradeWithoutInlineSourceCitationAddsNothing(t *testing.T) {
	const doc = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @I1@ INDI\n1 SOUR @S 1@\n0 @S 1@ SOUR\n0 TRLR\n"
	converted, report, _ := convertText(t, doc, gedcom.Version70)
	if len(converted.Records) != 2 {
		t.Errorf("%d records, want 2", len(converted.Records))
	}
	for _, tr := range report.Transformations {
		if tr.Type == "INLINE_SOUR_TO_RECORD" || tr.Type == "EMPTY_SOUR_TO_VOID" {
			t.Errorf("unexpected transformation %+v", tr)
		}
	}
}
