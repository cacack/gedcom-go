package decoder

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestParseSourceTextSubstructures pins the DATA.TEXT subtree walk added for
// issue #497: MIME, LANG and CONT/CONC are typed and silent, deeper levels are
// skipped, a vendor tag is kept raw without a diagnostic, and any other
// standard-looking tag is reported as UNKNOWN_TAG.
func TestParseSourceTextSubstructures(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 BIRT
2 SOUR @S1@
3 DATA
4 TEXT Passage
5 MIME text/plain
5 LANG en
5 _VEND vendor
6 ZZZY deeper
5 ZZZZ sentinel
4 TEXT Second
0 @S1@ SOUR
0 TRLR
`
	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}

	var unknown []string
	for _, diag := range result.Diagnostics {
		if diag.Code == CodeUnknownTag {
			unknown = append(unknown, diag.Message)
		}
	}
	if len(unknown) != 1 || !strings.Contains(unknown[0], "ZZZZ") {
		t.Errorf("UNKNOWN_TAG diagnostics = %q, want exactly one for ZZZZ", unknown)
	}

	ind := result.Document.GetIndividual("@I1@")
	if ind == nil || len(ind.Events) != 1 || len(ind.Events[0].SourceCitations) != 1 {
		t.Fatalf("expected one BIRT with one citation, got %+v", ind)
	}
	texts := ind.Events[0].SourceCitations[0].Data.Text
	if len(texts) != 2 {
		t.Fatalf("len(Data.Text) = %d, want 2", len(texts))
	}
	if got := *texts[0]; got != (gedcom.SourceText{Value: "Passage", MIME: "text/plain", Language: "en"}) {
		t.Errorf("Data.Text[0] = %+v", got)
	}
	if got := *texts[1]; got != (gedcom.SourceText{Value: "Second"}) {
		t.Errorf("Data.Text[1] = %+v", got)
	}
}

// TestParseSourceTextNoContinuationReusesValue guards the allocation-saving
// fast path: a TEXT with no CONT/CONC keeps the tag's own string.
func TestParseSourceTextNoContinuationReusesValue(t *testing.T) {
	tags := []*gedcom.Tag{
		{Level: 4, Tag: "TEXT", Value: "Only line"},
		{Level: 4, Tag: "TEXT", Value: "Next"},
	}
	got := parseSourceText(tags, 0, &diagnosticCollector{})
	if got.Value != "Only line" || got.MIME != "" || got.Language != "" {
		t.Errorf("parseSourceText = %+v, want {Value: Only line}", *got)
	}
}
