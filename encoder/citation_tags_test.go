package encoder

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestSourceCitationTagsRoundTrip checks that SourceCitation.Tags is written
// after the citation's typed lines, re-levelled under its SOUR line, at 5.5.1
// and 7.0, and decodes back; and that a standard tag or _APID in Tags is not
// written from there.
// Ref: Issue #864
func TestSourceCitationTagsRoundTrip(t *testing.T) {
	for _, version := range []gedcom.Version{gedcom.Version551, gedcom.Version70} {
		t.Run(string(version), func(t *testing.T) {
			cite := func() *gedcom.SourceCitation {
				return &gedcom.SourceCitation{
					SourceXRef: "@S1@",
					Page:       "12",
					Tags: []*gedcom.Tag{
						{Level: 1, Tag: "PAGE", Value: "ignored"},
						{Level: 1, Tag: "_APID", Value: "1,1::1"},
						{Level: 1, Tag: "_RESN", Value: "confidential"},
						{Level: 2, Tag: "_WHY", Value: "family request"},
					},
				}
			}
			doc := &gedcom.Document{
				Header: &gedcom.Header{Version: version},
				Records: []*gedcom.Record{
					{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{
						XRef:            "@I1@",
						SourceCitations: []*gedcom.SourceCitation{cite()},
						Events:          []*gedcom.Event{{Type: gedcom.EventBirth, SourceCitations: []*gedcom.SourceCitation{cite()}}},
					}},
					{XRef: "@S1@", Type: gedcom.RecordTypeSource, Entity: &gedcom.Source{XRef: "@S1@", Title: "Census"}},
				},
			}

			var buf bytes.Buffer
			if err := Encode(&buf, doc); err != nil {
				t.Fatalf("Encode() error = %v", err)
			}
			out := buf.String()
			for _, block := range []string{
				"\n1 SOUR @S1@\n2 PAGE 12\n2 _RESN confidential\n3 _WHY family request\n",
				"\n2 SOUR @S1@\n3 PAGE 12\n3 _RESN confidential\n4 _WHY family request\n",
			} {
				if !strings.Contains(out, block) {
					t.Errorf("output lacks %q\n%s", block, out)
				}
			}
			for _, absent := range []string{"PAGE ignored", "_APID"} {
				if strings.Contains(out, absent) {
					t.Errorf("output writes %q from Tags\n%s", absent, out)
				}
			}

			back, err := decoder.Decode(strings.NewReader(out))
			if err != nil {
				t.Fatalf("re-decode: %v\n%s", err, out)
			}
			indi := back.GetIndividual("@I1@")
			if indi == nil || len(indi.Events) != 1 {
				t.Fatalf("re-decoded @I1@ = %+v", indi)
			}
			for _, c := range []*gedcom.SourceCitation{indi.SourceCitations[0], indi.Events[0].SourceCitations[0]} {
				if len(c.Tags) != 2 || c.Tags[0].Tag != "_RESN" || c.Tags[0].Level != 1 || c.Tags[1].Level != 2 {
					t.Errorf("re-decoded citation Tags = %+v", c.Tags)
				}
			}
		})
	}
}

// TestSyncTagsKeepsCitationExtension checks that editing a citation's typed
// field and running SyncTags keeps its extension subtag exactly once.
// Ref: Issue #864
func TestSyncTagsKeepsCitationExtension(t *testing.T) {
	in := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n0 @I1@ INDI\n1 BIRT\n2 SOUR @S1@\n3 _RESN confidential\n3 PAGE 12\n0 @S1@ SOUR\n1 TITL Census\n0 TRLR\n"
	doc, err := decoder.Decode(strings.NewReader(in))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	doc.GetIndividual("@I1@").Events[0].SourceCitations[0].Page = "13"
	if err := SyncTags(doc); err != nil {
		t.Fatalf("SyncTags() error = %v", err)
	}
	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	out := buf.String()
	if n := strings.Count(out, "3 _RESN confidential\n"); n != 1 {
		t.Errorf("_RESN written %d times, want 1\n%s", n, out)
	}
	if !strings.Contains(out, "3 PAGE 13\n") {
		t.Errorf("PAGE edit not written\n%s", out)
	}
}

// TestSyncTagsCarriesCitationTagsEdit checks that an edit made through
// SourceCitation.Tags on a decoded document reaches the output after
// SyncTags, as the field documents: a changed value, and a removed subtag.
// Ref: Issue #864
func TestSyncTagsCarriesCitationTagsEdit(t *testing.T) {
	in := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n0 @I1@ INDI\n1 SOUR @S1@\n2 _RESN confidential\n1 BIRT\n2 SOUR @S1@\n3 _RESN locked\n0 @S1@ SOUR\n1 TITL Census\n0 TRLR\n"
	doc, err := decoder.Decode(strings.NewReader(in))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	indi := doc.GetIndividual("@I1@")
	indi.SourceCitations[0].Tags[0].Value = "privacy"
	indi.Events[0].SourceCitations[0].Tags = nil
	if err := SyncTags(doc); err != nil {
		t.Fatalf("SyncTags() error = %v", err)
	}
	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "\n1 SOUR @S1@\n2 _RESN privacy\n") {
		t.Errorf("edited _RESN not written\n%s", out)
	}
	if strings.Contains(out, "confidential") || strings.Contains(out, "locked") {
		t.Errorf("stale or removed _RESN still written\n%s", out)
	}
}
