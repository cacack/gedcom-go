package validator

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// substructureNotesGEDCOM puts a dangling note pointer under each of the five
// substructures that carry typed note pointers (#535): FAMC/FAMS links, ASSO,
// OBJE links, PLAC and source REPO links -- at record level and inside an event
// or attribute where the structure can appear there. @N1@ exists; every @NX..@
// pointer does not.
const substructureNotesGEDCOM = `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 FAMC @F1@
2 NOTE @NX1@
1 FAMS @F1@
2 NOTE @N1@
2 NOTE @NX2@
1 ASSO @I2@
2 RELA Godfather
2 NOTE @NX3@
1 OBJE @M1@
2 NOTE @NX4@
1 BIRT
2 PLAC Springfield
3 NOTE @NX5@
2 OBJE @M1@
3 NOTE @NX6@
2 ASSO @I2@
3 NOTE @NX7@
1 OCCU Farmer
2 PLAC Shelbyville
3 NOTE @NX8@
0 @I2@ INDI
1 NAME Jim /Roe/
0 @F1@ FAM
1 HUSB @I1@
1 OBJE @M1@
2 NOTE @NX9@
1 MARR
2 PLAC Capital City
3 NOTE @NX10@
0 @S1@ SOUR
1 TITL Register
1 REPO @R1@
2 NOTE @NX11@
1 OBJE @M1@
2 NOTE @NX12@
0 @R1@ REPO
1 NAME Archive
0 @M1@ OBJE
1 FILE photo.jpg
0 @N1@ NOTE A shared note
0 TRLR
`

// TestStreamingValidator_SubstructureNotePointers guards #535: a dangling note
// pointer is reported wherever a typed note pointer can appear, not only on a
// record's own NoteXRefs, and the issue's field locates it below the record.
func TestStreamingValidator_SubstructureNotePointers(t *testing.T) {
	doc, err := decoder.Decode(strings.NewReader(substructureNotesGEDCOM))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	sv := NewStreamingValidator(StreamingOptions{})
	for _, record := range doc.Records {
		if issues := sv.ValidateRecord(record); len(issues) != 0 {
			t.Fatalf("ValidateRecord(%s) = %v, want no immediate issues", record.XRef, issues)
		}
	}

	type loc struct{ record, field string }
	got := map[string]loc{}
	for _, iss := range sv.Finalize() {
		if iss.Code != "ORPHANED_NOTE" {
			t.Errorf("unexpected issue %s", iss.String())
			continue
		}
		if prev, dup := got[iss.RelatedXRef]; dup {
			t.Errorf("%s reported twice: %v and %s", iss.RelatedXRef, prev, iss.Details["field"])
		}
		got[iss.RelatedXRef] = loc{iss.RecordXRef, iss.Details["field"]}
	}

	want := map[string]loc{
		"@NX1@":  {"@I1@", "ChildInFamilies[0].NoteXRefs[0]"},
		"@NX2@":  {"@I1@", "SpouseInFamilies[0].NoteXRefs[1]"},
		"@NX3@":  {"@I1@", "Associations[0].NoteXRefs[0]"},
		"@NX4@":  {"@I1@", "Media[0].NoteXRefs[0]"},
		"@NX5@":  {"@I1@", "Events[0].PlaceDetail.NoteXRefs[0]"},
		"@NX6@":  {"@I1@", "Events[0].Media[0].NoteXRefs[0]"},
		"@NX7@":  {"@I1@", "Events[0].Associations[0].NoteXRefs[0]"},
		"@NX8@":  {"@I1@", "Attributes[0].PlaceDetail.NoteXRefs[0]"},
		"@NX9@":  {"@F1@", "Media[0].NoteXRefs[0]"},
		"@NX10@": {"@F1@", "Events[0].PlaceDetail.NoteXRefs[0]"},
		"@NX11@": {"@S1@", "RepositoryLinks[0].NoteXRefs[0]"},
		"@NX12@": {"@S1@", "Media[0].NoteXRefs[0]"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("orphaned NOTE issues =\n  %v\nwant\n  %v", got, want)
	}
}

// TestStreamingValidator_SubstructureNotePointers_NilElements checks that the
// substructure collectors skip nil elements in hand-built records rather than
// panicking (ADR 0007).
func TestStreamingValidator_SubstructureNotePointers_NilElements(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	ind := &gedcom.Individual{
		XRef:         "@I1@",
		Associations: []*gedcom.Association{nil},
		Media:        []*gedcom.MediaLink{nil},
		Events:       []*gedcom.Event{nil, {Associations: []*gedcom.Association{nil}}},
		Attributes:   []*gedcom.Attribute{nil, {Media: []*gedcom.MediaLink{nil}}},
	}
	src := &gedcom.Source{
		XRef:            "@S1@",
		RepositoryLinks: []*gedcom.SourceRepositoryLink{nil},
		Media:           []*gedcom.MediaLink{nil},
	}
	sv.ValidateRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: ind})
	sv.ValidateRecord(&gedcom.Record{XRef: "@S1@", Type: gedcom.RecordTypeSource, Entity: src})

	if issues := sv.Finalize(); len(issues) != 0 {
		t.Errorf("nil substructures produced %d issue(s), want 0: %v", len(issues), issues)
	}
}
