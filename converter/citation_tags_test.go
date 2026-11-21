package converter_test

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/converter"
	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestConvertKeepsCitationExtensionTags checks that a 7.0 to 5.5.1
// conversion keeps a citation's extension subtag in both Record.Tags and the
// re-derived SourceCitation.Tags.
// Ref: Issue #864
func TestConvertKeepsCitationExtensionTags(t *testing.T) {
	in := "0 HEAD\n1 GEDC\n2 VERS 7.0\n0 @I1@ INDI\n1 BIRT\n2 SOUR @S1@\n3 _RESN confidential\n0 @S1@ SOUR\n1 TITL Census\n0 TRLR\n"
	doc, err := decoder.Decode(strings.NewReader(in))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	got, _, err := converter.Convert(doc, gedcom.Version551)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	indi := got.GetIndividual("@I1@")
	if indi == nil || len(indi.Events) != 1 || len(indi.Events[0].SourceCitations) != 1 {
		t.Fatalf("converted @I1@ = %+v", indi)
	}
	tags := indi.Events[0].SourceCitations[0].Tags
	if len(tags) != 1 || tags[0].Tag != "_RESN" || tags[0].Value != "confidential" || tags[0].Level != 1 {
		t.Errorf("converted citation Tags = %+v", tags)
	}
	raw := false
	for _, rec := range got.Records {
		for _, tag := range rec.Tags {
			if tag.Tag == "_RESN" && tag.Level == 3 {
				raw = true
			}
		}
	}
	if !raw {
		t.Error("converted Record.Tags lacks 3 _RESN")
	}
}
