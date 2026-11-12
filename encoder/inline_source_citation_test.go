package encoder

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestInlineSourceCitationEntityRoundTrip: an inline citation's description
// (SourceCitation.InlineText, issue #738) is written back as the SOUR value,
// with CONT for a line break, when a record is encoded from its entity, and
// reads back the same.
func TestInlineSourceCitationEntityRoundTrip(t *testing.T) {
	input := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n" +
		"0 @I1@ INDI\n1 SOUR Family bible\n2 CONT of John @@ Smith\n2 QUAY 2\n0 TRLR\n"
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	doc.Records[0].Tags = nil // encode from the entity

	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if want := "1 SOUR Family bible\n2 CONT of John @@ Smith\n2 QUAY 2\n"; !strings.Contains(buf.String(), want) {
		t.Errorf("output lacks %q:\n%s", want, buf.String())
	}

	again, err := decoder.Decode(&buf)
	if err != nil {
		t.Fatalf("re-Decode: %v", err)
	}
	cite := again.GetIndividual("@I1@").SourceCitations[0]
	if cite.SourceXRef != "" || cite.InlineText != "Family bible\nof John @ Smith" {
		t.Errorf("SourceXRef = %q, InlineText = %q after round trip", cite.SourceXRef, cite.InlineText)
	}
}

// TestSourceCitationPointerWinsOverInlineText: a hand-built citation with both
// fields set is written as the pointer.
func TestSourceCitationPointerWinsOverInlineText(t *testing.T) {
	tags := sourceCitationToTags(&gedcom.SourceCitation{SourceXRef: "@S1@", InlineText: "ignored"}, 1, DefaultOptions())
	if len(tags) != 1 || tags[0].Tag != "SOUR" || tags[0].Value != "@S1@" {
		t.Errorf("tags = %+v, want one SOUR @S1@", tags[0])
	}
}
