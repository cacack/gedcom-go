package encoder

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/decoder"
	"github.com/cacack/gedcom-go/v2/gedcom"
)

// TestQualityZeroSurvivesEncode pins QUAY 0 as a real assertion rather than an
// absence. SourceCitation.Quality was an int, so its Go zero value collided
// with the meaningful GEDCOM value 0 ("unreliable evidence or estimated data")
// and the encoder resolved the ambiguity in favour of absent, emitting the tag
// only when Quality > 0. A caller could not express "unreliable" at all
// (issue #493).
func TestQualityZeroSurvivesEncode(t *testing.T) {
	t.Run("hand-built citation with Quality 0 encodes QUAY 0", func(t *testing.T) {
		zero := 0
		rec := &gedcom.Record{
			XRef: "@I1@",
			Type: gedcom.RecordTypeIndividual,
			Entity: &gedcom.Individual{
				XRef:            "@I1@",
				SourceCitations: []*gedcom.SourceCitation{{SourceXRef: "@S1@", Quality: &zero}},
			},
		}
		doc := &gedcom.Document{
			Header:  &gedcom.Header{Version: gedcom.Version551, Encoding: gedcom.EncodingUTF8},
			Records: []*gedcom.Record{rec},
			XRefMap: map[string]*gedcom.Record{"@I1@": rec},
		}

		var buf bytes.Buffer
		if err := Encode(&buf, doc); err != nil {
			t.Fatalf("Encode() error = %v", err)
		}
		if !strings.Contains(buf.String(), "QUAY 0") {
			t.Errorf("QUAY 0 was dropped:\n%s", buf.String())
		}
	})

	t.Run("nil Quality emits no QUAY", func(t *testing.T) {
		tags := sourceCitationToTags(&gedcom.SourceCitation{SourceXRef: "@S1@"}, 1, nil)
		for _, tag := range tags {
			if tag.Tag == "QUAY" {
				t.Errorf("nil Quality emitted %+v", tag)
			}
		}
	})

	// Record.Tags is authoritative on encode, so the loss only ever showed on a
	// document whose raw tags were cleared -- the converter's path, and anything
	// building from entities.
	t.Run("QUAY 0 survives decode, Tags clear, re-encode", func(t *testing.T) {
		input := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n" +
			"0 @I1@ INDI\n1 NAME John /Doe/\n1 SOUR @S1@\n2 QUAY 0\n" +
			"0 @S1@ SOUR\n1 TITL A source\n0 TRLR\n"

		doc, err := decoder.Decode(strings.NewReader(input))
		if err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		indi := doc.GetIndividual("@I1@")
		if indi == nil {
			t.Fatal("GetIndividual(@I1@) returned nil")
		}
		if len(indi.SourceCitations) != 1 {
			t.Fatalf("len(SourceCitations) = %d, want 1", len(indi.SourceCitations))
		}
		q := indi.SourceCitations[0].Quality
		if q == nil {
			t.Fatal("QUAY 0 decoded to a nil Quality; it is a value, not an absence")
		}
		if *q != 0 {
			t.Fatalf("*Quality = %d, want 0", *q)
		}

		for _, rec := range doc.Records {
			rec.Tags = nil
		}

		var buf bytes.Buffer
		if err := Encode(&buf, doc); err != nil {
			t.Fatalf("re-encode error = %v", err)
		}
		if !strings.Contains(buf.String(), "QUAY 0") {
			t.Errorf("QUAY 0 lost on the entity path:\n%s", buf.String())
		}
	})
}
