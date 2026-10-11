package converter_test

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/converter"
	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestConvert70To70NormalizesXRefs pins issue #604: converting a document
// that already declares 7.0 to 7.0 must still rewrite XRefs the 7.0 grammar
// rejects, along with the pointers to them, and record each rename, while
// leaving the caller's document untouched.
func TestConvert70To70NormalizesXRefs(t *testing.T) {
	const input = `0 HEAD
1 GEDC
2 VERS 7.0
0 @i 2@ INDI
1 NAME John /Doe/
1 FAMS @F1@
0 @F1@ FAM
1 HUSB @i 2@
0 TRLR
`
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	got, report, err := converter.Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if !report.Success {
		t.Error("report.Success = false, want true")
	}

	if got.GetIndividual("@I_2@") == nil {
		t.Fatalf("individual @I_2@ missing; records = %v", recordXRefs(got))
	}
	fam := got.GetFamily("@F1@")
	if fam == nil {
		t.Fatal("family @F1@ missing")
	}
	if fam.Husband != "@I_2@" || got.GetIndividual(fam.Husband) == nil {
		t.Errorf("HUSB = %q, want resolving @I_2@", fam.Husband)
	}

	found := false
	for _, note := range report.Normalized {
		if note.Original == "@i 2@" && note.Result == "@I_2@" {
			found = true
		}
	}
	if !found {
		t.Errorf("report.Normalized = %+v, want a note for @i 2@ -> @I_2@", report.Normalized)
	}

	if doc.GetIndividual("@i 2@") == nil || doc.GetFamily("@F1@").Husband != "@i 2@" {
		t.Error("Convert mutated the caller's document")
	}
}

// TestConvert70To70ValidXRefsReturnsOriginal pins that a 7.0 document whose
// XRefs are already valid still comes back as-is, with nothing reported.
func TestConvert70To70ValidXRefsReturnsOriginal(t *testing.T) {
	doc := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version70, DeclaredVersion: gedcom.Version70},
		Records: []*gedcom.Record{{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: "@I1@"}}},
	}
	got, report, err := converter.Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if got != doc {
		t.Error("Convert returned a copy, want the original document")
	}
	if len(report.Normalized) != 0 || len(report.Transformations) != 0 {
		t.Errorf("report = %+v, want no notes or transformations", report)
	}
}

func recordXRefs(doc *gedcom.Document) []string {
	xrefs := make([]string, 0, len(doc.Records))
	for _, r := range doc.Records {
		xrefs = append(xrefs, r.XRef)
	}
	return xrefs
}
