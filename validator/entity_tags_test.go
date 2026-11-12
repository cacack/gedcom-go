package validator

import (
	"os"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Issue #725: a typed edit the encoder would ignore is reported before writing.

func decodeFixture(t testing.TB, path string) *gedcom.Document {
	t.Helper()
	f, err := os.Open(path) // #nosec G304 -- path is a constant in the repo's own testdata
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	doc, err := decoder.Decode(f)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	return doc
}

// checking is a validator with the opt-in ENTITY_TAGS_DIVERGED check on.
func checking() *Validator {
	return NewWithOptions(&ValidateOptions{CheckEntityTags: true})
}

func divergedIssues(doc *gedcom.Document) []Issue {
	var out []Issue
	for _, issue := range checking().ValidateAll(doc) {
		if issue.Code == CodeEntityTagsDiverged {
			out = append(out, issue)
		}
	}
	return out
}

func TestEntityTagsDiverged(t *testing.T) {
	doc := decodeFixture(t, "../testdata/gedcom-5.5/pres2020.ged")
	if got := divergedIssues(doc); len(got) != 0 {
		t.Fatalf("unmodified document reported %d ENTITY_TAGS_DIVERGED, first %v", len(got), got[0])
	}

	rec := doc.Records[0]
	for _, r := range doc.Records {
		if r.Type == gedcom.RecordTypeIndividual {
			rec = r
			break
		}
	}
	indi, _ := rec.GetIndividual()
	indi.Names[0].Full = "Zzz /Edited/"
	doc.Header.SourceSystem = "EDITED-APP"
	tagsBefore := len(rec.Tags)

	got := divergedIssues(doc)
	if len(got) != 2 {
		t.Fatalf("got %d ENTITY_TAGS_DIVERGED, want 2 (header and %s): %v", len(got), rec.XRef, got)
	}
	if got[0].Details["record"] != "HEAD" || got[0].RecordXRef != "" {
		t.Errorf("first issue = %+v, want the header", got[0])
	}
	if got[1].RecordXRef != rec.XRef || got[1].LineNumber != rec.LineNumber || got[1].Severity != SeverityWarning {
		t.Errorf("second issue = %+v, want a warning on %s at line %d", got[1], rec.XRef, rec.LineNumber)
	}
	if len(rec.Tags) != tagsBefore || rec.Tags[0].Value == "Zzz /Edited/" {
		t.Error("validation modified the record's Tags")
	}

	if err := encoder.SyncTags(doc); err != nil {
		t.Fatalf("SyncTags: %v", err)
	}
	if got := divergedIssues(doc); len(got) != 0 {
		t.Errorf("after SyncTags still reported %d ENTITY_TAGS_DIVERGED: %v", len(got), got)
	}
}

// A hand-built document has no raw tags to diverge from, and a document
// SyncTags cannot process gets one Info issue saying so.
func TestEntityTagsDivergedSkips(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version551, Submitter: "@S\n1@"},
		Records: []*gedcom.Record{nil, {XRef: "@I1@", Type: gedcom.RecordTypeIndividual,
			Entity: &gedcom.Individual{Names: []*gedcom.PersonalName{{Full: "A /B/"}}}}},
	}
	if got := validateEntityTags(doc); len(got) != 0 {
		t.Errorf("hand-built document reported %v", got)
	}
	doc.Header.Tags = []*gedcom.Tag{{Level: 1, Tag: "SUBM", Value: "@S1@"}}
	got := validateEntityTags(doc)
	if len(got) != 1 || got[0].Severity != SeverityInfo || got[0].Details["error"] == "" {
		t.Errorf("document SyncTags rejects reported %v, want one Info issue carrying the error", got)
	}
}

func TestEntityTagsDivergedSkipRule(t *testing.T) {
	doc := decodeFixture(t, "../testdata/gedcom-5.5/royal92.ged")
	doc.Header.SourceSystem = "EDITED-APP"
	if len(divergedIssues(doc)) != 1 {
		t.Fatal("edited header not reported")
	}
	v := NewWithOptions(&ValidateOptions{CheckEntityTags: true, SkipRules: []string{CodeEntityTagsDiverged}})
	for _, issue := range v.ValidateAll(doc) {
		if issue.Code == CodeEntityTagsDiverged {
			t.Errorf("reported despite SkipRules: %v", issue)
		}
	}
}

// TestEntityTagsDivergedOptIn asserts the check is off by default: the same
// edited header goes unreported by a default ValidateAll and QualityReport.
func TestEntityTagsDivergedOptIn(t *testing.T) {
	doc := decodeFixture(t, "../testdata/gedcom-5.5/royal92.ged")
	doc.Header.SourceSystem = "EDITED-APP"
	for _, issue := range New().ValidateAll(doc) {
		if issue.Code == CodeEntityTagsDiverged {
			t.Errorf("default ValidateAll reported %v", issue)
		}
	}
	for _, issue := range New().QualityReport(doc).ConformanceIssues {
		if issue.Code == CodeEntityTagsDiverged {
			t.Errorf("default QualityReport reported %v", issue)
		}
	}
}

func TestEntityTagsDivergedInQualityReport(t *testing.T) {
	doc := decodeFixture(t, "../testdata/gedcom-5.5/royal92.ged")
	doc.Header.SourceSystem = "EDITED-APP"
	found := false
	for _, issue := range checking().QualityReport(doc).ConformanceIssues {
		found = found || issue.Code == CodeEntityTagsDiverged
	}
	if !found {
		t.Error("QualityReport.ConformanceIssues lacks ENTITY_TAGS_DIVERGED")
	}
}

// TestWithEntityTagsCheck asserts the QualityAnalyzer opt-in adds the check
// and nothing else: the same edited document does not report it by default.
func TestWithEntityTagsCheck(t *testing.T) {
	doc := decodeFixture(t, "../testdata/gedcom-5.5/royal92.ged")
	doc.Header.SourceSystem = "EDITED-APP"
	count := func(r *QualityReport) (diverged, other int) {
		for _, issue := range r.ConformanceIssues {
			if issue.Code == CodeEntityTagsDiverged {
				diverged++
			} else {
				other++
			}
		}
		return diverged, other
	}
	withD, withOther := count(NewQualityAnalyzer(WithEntityTagsCheck()).Analyze(doc))
	withoutD, withoutOther := count(NewQualityAnalyzer().Analyze(doc))
	if withD == 0 {
		t.Fatal("WithEntityTagsCheck: no ENTITY_TAGS_DIVERGED for an edited header")
	}
	if withoutD != 0 {
		t.Errorf("default analyzer: %d ENTITY_TAGS_DIVERGED issues, want 0", withoutD)
	}
	if withoutOther != withOther {
		t.Errorf("WithEntityTagsCheck changed the other conformance issues: %d, want %d", withOther, withoutOther)
	}
}

// TestEntityTagsDivergedAgePhraseOnly pins issue #795 on age.ged's phrase-only
// "2 AGE" / "3 PHRASE Zero": the unedited file reports no
// ENTITY_TAGS_DIVERGED, and an edit to that phrase is reported and reaches
// Tags through SyncTags (the entity writer used to drop the line, so the edit
// went unseen).
func TestEntityTagsDivergedAgePhraseOnly(t *testing.T) {
	doc := decodeFixture(t, "../testdata/gedcom-7.0/familysearch-examples/age.ged")
	if got := divergedIssues(doc); len(got) != 0 {
		t.Fatalf("unmodified age.ged reported ENTITY_TAGS_DIVERGED: %v", got)
	}
	indi := doc.GetIndividual("@I1@")
	var edited *gedcom.Event
	for _, ev := range indi.Events {
		if ev.Age == "" && ev.AgePhrase == "Zero" {
			edited = ev
		}
	}
	if edited == nil {
		t.Fatal("age.ged has no phrase-only age")
	}
	edited.AgePhrase = "Newborn"
	if got := divergedIssues(doc); len(got) != 1 || got[0].RecordXRef != "@I1@" {
		t.Fatalf("edited phrase-only age reported %v, want one issue on @I1@", got)
	}
	if err := encoder.SyncTags(doc); err != nil {
		t.Fatalf("SyncTags: %v", err)
	}
	tags := doc.GetRecord("@I1@").Tags
	found := false
	for i, tag := range tags {
		if tag.Tag == "AGE" && tag.Value == "" && i+1 < len(tags) {
			next := tags[i+1]
			found = found || (next.Tag == "PHRASE" && next.Value == "Newborn" && next.Level == tag.Level+1)
		}
	}
	if !found {
		t.Error("SyncTags did not write the edited phrase beneath an empty AGE")
	}
}
