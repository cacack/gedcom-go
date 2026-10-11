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

	var uppercase *gedcom.Transformation
	for i := range report.Transformations {
		if report.Transformations[i].Type == "XREF_UPPERCASE" {
			uppercase = &report.Transformations[i]
		}
	}
	if uppercase == nil || uppercase.Count != 1 {
		t.Errorf("report.Transformations = %+v, want XREF_UPPERCASE with Count 1", report.Transformations)
	}

	if doc.GetIndividual("@i 2@") == nil || doc.GetFamily("@F1@").Husband != "@i 2@" {
		t.Error("Convert mutated the caller's document")
	}
}

// TestConvert70To70DanglingPointers pins the documented handling of pointers
// no record defines on the 7.0 -> 7.0 path: one with a space is not a 7.0
// pointer and is left as-is, even when a record rename forces the copy; one
// the 7.0 grammar reads as a pointer but rejects as an XRef (@x9@) is renamed
// on its own, so the copy decision covers pointers and not only records.
func TestConvert70To70DanglingPointers(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantOut    []string
		wantNotes  map[string]string // Original -> Result
		notWantOut []string
	}{
		{
			name:      "spaced dangling pointer kept beside a renamed record",
			body:      "0 @i1@ INDI\n1 NOTE @X 9@\n",
			wantOut:   []string{"0 @I1@ INDI", "1 NOTE @X 9@"},
			wantNotes: map[string]string{"@i1@": "@I1@"},
		},
		{
			name:       "spaced dangling pointer alone",
			body:       "0 @I1@ INDI\n1 NOTE @X 9@\n",
			wantOut:    []string{"1 NOTE @X 9@"},
			wantNotes:  map[string]string{},
			notWantOut: []string{"@X_9@"},
		},
		{
			name:      "invalid dangling pointer alone",
			body:      "0 @I1@ INDI\n1 ASSO @x9@\n2 ROLE FRIEND\n",
			wantOut:   []string{"1 ASSO @X9@"},
			wantNotes: map[string]string{"@x9@": "@X9@"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := decodeString(t, "0 HEAD\n1 GEDC\n2 VERS 7.0\n"+tt.body+"0 TRLR\n")
			before := encodeString(t, doc)

			got, report, err := converter.Convert(doc, gedcom.Version70)
			if err != nil {
				t.Fatalf("Convert: %v", err)
			}
			out := encodeString(t, got)
			for _, want := range tt.wantOut {
				if !strings.Contains(out, want+"\n") && !strings.Contains(out, want+"\r\n") {
					t.Errorf("output missing %q:\n%s", want, out)
				}
			}
			for _, bad := range tt.notWantOut {
				if strings.Contains(out, bad) {
					t.Errorf("output contains %q:\n%s", bad, out)
				}
			}
			notes := make(map[string]string, len(report.Normalized))
			for _, n := range report.Normalized {
				notes[n.Original] = n.Result
			}
			if len(notes) != len(tt.wantNotes) {
				t.Errorf("report.Normalized = %+v, want %v", report.Normalized, tt.wantNotes)
			}
			for orig, res := range tt.wantNotes {
				if notes[orig] != res {
					t.Errorf("note for %s = %q, want %q", orig, notes[orig], res)
				}
			}
			if len(tt.wantNotes) == 0 && got != doc {
				t.Error("Convert returned a copy, want the original document")
			}
			if after := encodeString(t, doc); after != before {
				t.Errorf("Convert mutated the caller's document:\n%s", after)
			}
		})
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
