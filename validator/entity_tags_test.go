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

func divergedIssues(doc *gedcom.Document) []Issue {
	var out []Issue
	for _, issue := range New().ValidateAll(doc) {
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
// SyncTags cannot process reports nothing here.
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
	if got := validateEntityTags(doc); len(got) != 0 {
		t.Errorf("document SyncTags rejects reported %v", got)
	}
}

func TestEntityTagsDivergedSkipRule(t *testing.T) {
	doc := decodeFixture(t, "../testdata/gedcom-5.5/royal92.ged")
	doc.Header.SourceSystem = "EDITED-APP"
	if len(divergedIssues(doc)) != 1 {
		t.Fatal("edited header not reported")
	}
	v := NewWithOptions(&ValidateOptions{SkipRules: []string{CodeEntityTagsDiverged}})
	for _, issue := range v.ValidateAll(doc) {
		if issue.Code == CodeEntityTagsDiverged {
			t.Errorf("reported despite SkipRules: %v", issue)
		}
	}
}

func TestEntityTagsDivergedInQualityReport(t *testing.T) {
	doc := decodeFixture(t, "../testdata/gedcom-5.5/royal92.ged")
	doc.Header.SourceSystem = "EDITED-APP"
	found := false
	for _, issue := range New().QualityReport(doc).ConformanceIssues {
		found = found || issue.Code == CodeEntityTagsDiverged
	}
	if !found {
		t.Error("QualityReport.ConformanceIssues lacks ENTITY_TAGS_DIVERGED")
	}
}
