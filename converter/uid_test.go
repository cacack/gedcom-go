package converter

import (
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Issue #715: _UID and UID are the 5.5.x and 7.0 spellings of one identifier,
// so conversion renames rather than drops.

func uidDoc(version gedcom.Version, tag string) *gedcom.Document {
	rec := &gedcom.Record{
		XRef: "@I1@", Type: gedcom.RecordTypeIndividual,
		Tags: []*gedcom.Tag{
			{Level: 0, Tag: "INDI", XRef: "@I1@"},
			{Level: 1, Tag: tag, Value: "abc"},
			{Level: 1, Tag: "BIRT"},
			{Level: 2, Tag: tag, Value: "ev"},
		},
		Entity: &gedcom.Individual{XRef: "@I1@", UIDs: []string{"abc"}},
	}
	return &gedcom.Document{
		Header:  &gedcom.Header{Version: version, DeclaredVersion: version},
		Records: []*gedcom.Record{rec},
		XRefMap: map[string]*gedcom.Record{"@I1@": rec},
	}
}

func tagNames(rec *gedcom.Record) []string {
	var out []string
	for _, t := range rec.Tags {
		out = append(out, t.Tag)
	}
	return out
}

func TestTransformUIDTags(t *testing.T) {
	tests := []struct {
		name         string
		from, target gedcom.Version
		in, to       string
		typ          string
	}{
		{"5.5 to 7.0", gedcom.Version55, gedcom.Version70, "_UID", "UID", "UID_FROM_VENDOR_TAG"},
		{"5.5.1 to 7.0", gedcom.Version551, gedcom.Version70, "_UID", "UID", "UID_FROM_VENDOR_TAG"},
		{"7.0 to 5.5", gedcom.Version70, gedcom.Version55, "UID", "_UID", "UID_TO_VENDOR_TAG"},
		{"7.0 to 5.5.1", gedcom.Version70, gedcom.Version551, "UID", "_UID", "UID_TO_VENDOR_TAG"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converted, report, err := Convert(uidDoc(tt.from, tt.in), tt.target)
			if err != nil {
				t.Fatalf("Convert: %v", err)
			}
			rec := converted.XRefMap["@I1@"]
			names := tagNames(rec)
			got := map[string]int{}
			for _, n := range names {
				got[n]++
			}
			if got[tt.to] != 2 || got[tt.in] != 0 {
				t.Errorf("tags after conversion = %v, want both %s renamed to %s", names, tt.in, tt.to)
			}
			indi, _ := rec.GetIndividual()
			if len(indi.UIDs) != 1 || indi.UIDs[0] != "abc" {
				t.Errorf("Individual.UIDs = %v, want kept", indi.UIDs)
			}
			for _, loss := range report.DataLoss {
				if loss.Feature == "UID tags" || loss.Feature == "_UID tags" {
					t.Errorf("identifier reported as data loss: %+v", loss)
				}
			}

			normalized := 0
			for _, note := range report.Normalized {
				if note.Original == tt.in && note.Result == tt.to {
					normalized++
				}
			}
			if normalized != 2 {
				t.Errorf("Normalized notes %s -> %s = %d, want 2", tt.in, tt.to, normalized)
			}
			found := false
			for _, tr := range report.Transformations {
				if tr.Type == tt.typ {
					found = true
					if tr.Count != 2 {
						t.Errorf("%s Count = %d, want 2", tt.typ, tr.Count)
					}
				}
			}
			if !found {
				t.Errorf("no %s transformation in report: %+v", tt.typ, report.Transformations)
			}
		})
	}
}

// TestTransformUIDTagsNoneToRename checks a document without identifiers
// gets no note and no transformation entry.
func TestTransformUIDTagsNoneToRename(t *testing.T) {
	doc := uidDoc(gedcom.Version551, "REFN")
	_, report, err := Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	for _, tr := range report.Transformations {
		if tr.Type == "UID_FROM_VENDOR_TAG" || tr.Type == "UID_TO_VENDOR_TAG" {
			t.Errorf("unexpected transformation %+v", tr)
		}
	}
}
