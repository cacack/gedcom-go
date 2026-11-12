package decoder

import (
	"strings"
	"testing"
)

// TestInlineSourceCitation covers the GEDCOM 5.5/5.5.1 citation without a
// source record ("1 SOUR <description>", issue #738): the description, with
// its CONT/CONC lines folded in, goes to InlineText, SourceXRef stays empty,
// and the continuation lines are not reported as unknown tags.
func TestInlineSourceCitation(t *testing.T) {
	const input = `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 SOUR Family bible in poss
2 CONC ession of
2 CONT John @@ Smith
2 TEXT Born 1850
2 QUAY 1
1 SOUR @S1@
2 PAGE 42
1 SOUR @S 2@
1 SOUR @S1
0 @S1@ SOUR
1 TITL Census
0 @S 2@ SOUR
1 TITL Spaced
0 TRLR`

	res, err := DecodeWithDiagnostics(strings.NewReader(input), DefaultOptions())
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics failed: %v", err)
	}
	for _, d := range res.Diagnostics {
		if d.Code == CodeUnknownTag {
			t.Errorf("unexpected diagnostic: %s", d.String())
		}
	}
	cites := res.Document.GetIndividual("@I1@").SourceCitations
	if len(cites) != 4 {
		t.Fatalf("got %d citations, want 4", len(cites))
	}
	tests := []struct {
		xref, text string
	}{
		{"", "Family bible in possession of\nJohn @ Smith"},
		{"@S1@", ""},
		{"@S 2@", ""}, // a 5.5.1 pointer may hold a space (#579)
		{"", "@S1"},   // not a pointer: text, as for NOTE
	}
	for i, tt := range tests {
		if cites[i].SourceXRef != tt.xref || cites[i].InlineText != tt.text {
			t.Errorf("citation %d: SourceXRef = %q, InlineText = %q; want %q, %q",
				i, cites[i].SourceXRef, cites[i].InlineText, tt.xref, tt.text)
		}
	}
	if q := cites[0].Quality; q == nil || *q != 1 {
		t.Errorf("inline citation QUAY = %v, want 1", q)
	}
}

// TestPointerSourceCitationContinuationReported: a pointer takes no
// continuation, so a CONT under one is still reported as an unknown tag.
func TestPointerSourceCitationContinuationReported(t *testing.T) {
	const input = `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 SOUR @S1@
2 CONT stray
0 @S1@ SOUR
0 TRLR`

	res, err := DecodeWithDiagnostics(strings.NewReader(input), DefaultOptions())
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics failed: %v", err)
	}
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == CodeUnknownTag && strings.Contains(d.String(), "CONT") {
			found = true
		}
	}
	if !found {
		t.Errorf("CONT under a SOUR pointer not reported; diagnostics: %v", res.Diagnostics)
	}
	if cite := res.Document.GetIndividual("@I1@").SourceCitations[0]; cite.SourceXRef != "@S1@" || cite.InlineText != "" {
		t.Errorf("SourceXRef = %q, InlineText = %q; want @S1@ and empty", cite.SourceXRef, cite.InlineText)
	}
}

// TestVoidSourceCitationIsPointer: the GEDCOM 7.0 null pointer stays a
// pointer, not inline text.
func TestVoidSourceCitationIsPointer(t *testing.T) {
	const input = `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 SOUR @VOID@
2 PAGE 7
0 TRLR`

	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if cite := doc.GetIndividual("@I1@").SourceCitations[0]; cite.SourceXRef != "@VOID@" || cite.InlineText != "" {
		t.Errorf("SourceXRef = %q, InlineText = %q; want @VOID@ and empty", cite.SourceXRef, cite.InlineText)
	}
}
