package encoder

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestRecordNotesToEncode verifies the encoder emits shared-note XRef pointers
// first, then inline note text.
func TestRecordNotesToEncode(t *testing.T) {
	tests := []struct {
		name      string
		noteXRefs []string
		inline    []string
		want      []string
	}{
		{
			name:      "xrefs precede inline text",
			noteXRefs: []string{"@N1@", "@N2@"},
			inline:    []string{"first inline", "second inline"},
			want:      []string{"@N1@", "@N2@", "first inline", "second inline"},
		},
		{
			name:      "only xrefs populated",
			noteXRefs: []string{"@N1@"},
			want:      []string{"@N1@"},
		},
		{
			name:   "only inline populated",
			inline: []string{"just inline"},
			want:   []string{"just inline"},
		},
		{
			name: "all empty returns an empty slice",
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := recordNotesToEncode(tt.noteXRefs, tt.inline)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("recordNotesToEncode() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// TestIndividualToTagsNoteSplit confirms an individual built with only the split
// note fields still emits NOTE tags for both the XRef pointer and inline text.
func TestIndividualToTagsNoteSplit(t *testing.T) {
	indi := &gedcom.Individual{
		NoteXRefs:   []string{"@N1@"},
		InlineNotes: []string{"An inline note"},
	}

	tags := individualToTags(indi, nil)

	var noteValues []string
	for _, tag := range tags {
		if tag.Tag == "NOTE" {
			noteValues = append(noteValues, tag.Value)
		}
	}

	want := []string{"@N1@", "An inline note"}
	if !reflect.DeepEqual(noteValues, want) {
		t.Errorf("individualToTags() NOTE values = %#v, want %#v", noteValues, want)
	}
}

// noteTagValues returns the Value of each top-level NOTE tag in order.
func noteTagValues(tags []*gedcom.Tag) []string {
	var out []string
	for _, tag := range tags {
		if tag.Tag == "NOTE" {
			out = append(out, tag.Value)
		}
	}
	return out
}

// TestRecordTypesNoteSplit confirms every note-bearing record type built with
// only the split note fields still emits NOTE tags for both the XRef pointer and
// inline text. It covers all six record types whose NOTE emission was changed.
func TestRecordTypesNoteSplit(t *testing.T) {
	want := []string{"@N1@", "An inline note"}

	indi := &gedcom.Individual{NoteXRefs: []string{"@N1@"}, InlineNotes: []string{"An inline note"}}
	if got := noteTagValues(individualToTags(indi, nil)); !reflect.DeepEqual(got, want) {
		t.Errorf("individualToTags() NOTE values = %#v, want %#v", got, want)
	}

	fam := &gedcom.Family{NoteXRefs: []string{"@N1@"}, InlineNotes: []string{"An inline note"}}
	if got := noteTagValues(familyToTags(fam, nil)); !reflect.DeepEqual(got, want) {
		t.Errorf("familyToTags() NOTE values = %#v, want %#v", got, want)
	}

	src := &gedcom.Source{NoteXRefs: []string{"@N1@"}, InlineNotes: []string{"An inline note"}}
	if got := noteTagValues(sourceToTags(src, nil)); !reflect.DeepEqual(got, want) {
		t.Errorf("sourceToTags() NOTE values = %#v, want %#v", got, want)
	}

	subm := &gedcom.Submitter{NoteXRefs: []string{"@N1@"}, InlineNotes: []string{"An inline note"}}
	if got := noteTagValues(submitterToTags(subm, nil)); !reflect.DeepEqual(got, want) {
		t.Errorf("submitterToTags() NOTE values = %#v, want %#v", got, want)
	}

	repo := &gedcom.Repository{NoteXRefs: []string{"@N1@"}, InlineNotes: []string{"An inline note"}}
	if got := noteTagValues(repositoryToTags(repo, nil)); !reflect.DeepEqual(got, want) {
		t.Errorf("repositoryToTags() NOTE values = %#v, want %#v", got, want)
	}

	media := &gedcom.MediaObject{NoteXRefs: []string{"@N1@"}, InlineNotes: []string{"An inline note"}}
	if got := noteTagValues(mediaObjectToTags(media, nil)); !reflect.DeepEqual(got, want) {
		t.Errorf("mediaObjectToTags() NOTE values = %#v, want %#v", got, want)
	}
}

// TestSubstructureTypesNoteSplit confirms the five substructures split in
// issue #472 emit NOTE tags for both the XRef pointer and inline text. The
// FamilyLink case is checked under both tag names, because #534 routed FAMS
// through the same writer as FAMC.
func TestSubstructureTypesNoteSplit(t *testing.T) {
	want := []string{"@N1@", "An inline note"}

	ml := &gedcom.MediaLink{MediaXRef: "@O1@", NoteXRefs: []string{"@N1@"},
		InlineNotes: []string{"An inline note"}}
	if got := noteTagValues(mediaLinkToTags(ml, 1, nil)); !reflect.DeepEqual(got, want) {
		t.Errorf("mediaLinkToTags() NOTE values = %#v, want %#v", got, want)
	}

	fl := &gedcom.FamilyLink{FamilyXRef: "@F1@", NoteXRefs: []string{"@N1@"},
		InlineNotes: []string{"An inline note"}}
	if got := noteTagValues(familyLinkToTags(fl, "FAMC", 1, nil)); !reflect.DeepEqual(got, want) {
		t.Errorf("familyLinkToTags(FAMC) NOTE values = %#v, want %#v", got, want)
	}
	// A spouse link is the same struct written under a different tag name
	// (#534), so both tag names must emit both note forms.
	if got := noteTagValues(familyLinkToTags(fl, "FAMS", 1, nil)); !reflect.DeepEqual(got, want) {
		t.Errorf("familyLinkToTags(FAMS) NOTE values = %#v, want %#v", got, want)
	}

	pd := &gedcom.PlaceDetail{Name: "Springfield", NoteXRefs: []string{"@N1@"},
		InlineNotes: []string{"An inline note"}}
	if got := noteTagValues(placeToTags(pd, 2, nil)); !reflect.DeepEqual(got, want) {
		t.Errorf("placeToTags() NOTE values = %#v, want %#v", got, want)
	}

	as := &gedcom.Association{IndividualXRef: "@I2@", NoteXRefs: []string{"@N1@"},
		InlineNotes: []string{"An inline note"}}
	if got := noteTagValues(associationToTags(as, 1, nil)); !reflect.DeepEqual(got, want) {
		t.Errorf("associationToTags() NOTE values = %#v, want %#v", got, want)
	}

	rl := &gedcom.SourceRepositoryLink{XRef: "@R1@", NoteXRefs: []string{"@N1@"},
		InlineNotes: []string{"An inline note"}}
	if got := noteTagValues(sourceRepositoryLinkToTags(rl, nil)); !reflect.DeepEqual(got, want) {
		t.Errorf("sourceRepositoryLinkToTags() NOTE values = %#v, want %#v", got, want)
	}
}

// TestSourceRepositoryLinkSplitNotesKeepREPO guards the degenerate-link skip in
// sourceRepositoryLinkToTags: a link whose only content is a split note must
// still emit its REPO line, or the note has nowhere to hang.
func TestSourceRepositoryLinkSplitNotesKeepREPO(t *testing.T) {
	link := &gedcom.SourceRepositoryLink{InlineNotes: []string{"An inline note"}}

	tags := sourceRepositoryLinkToTags(link, nil)
	if len(tags) == 0 || tags[0].Tag != "REPO" {
		t.Fatalf("sourceRepositoryLinkToTags() = %#v, want a leading REPO tag", tags)
	}
	if got := noteTagValues(tags); !reflect.DeepEqual(got, []string{"An inline note"}) {
		t.Errorf("NOTE values = %#v, want %#v", got, []string{"An inline note"})
	}
}

// TestMediaObjectToTagsWritesSharedNotePointers pins that partitioning
// MediaObject's two pointer slices (#499) did not cost the writer a note. SNOTE
// pointers used to reach mediaObjectToTags via NoteXRefs, which the decoder
// also filled; now they arrive only in SharedNoteXRefs, and the writer reads
// both. The writer emits every pointer as NOTE; choosing SNOTE for a 7.0
// pointer to an SNOTE record is writeRecord's job (#471), pinned by
// TestEncodeSharedNotePointerTag.
func TestMediaObjectToTagsWritesSharedNotePointers(t *testing.T) {
	media := &gedcom.MediaObject{
		NoteXRefs:       []string{"@N1@"},
		SharedNoteXRefs: []string{"@S1@"},
		InlineNotes:     []string{"An inline note"},
	}

	want := []string{"@N1@", "@S1@", "An inline note"}
	if got := noteTagValues(mediaObjectToTags(media, nil)); !reflect.DeepEqual(got, want) {
		t.Errorf("mediaObjectToTags() NOTE values = %#v, want %#v", got, want)
	}
}

// sharedNotePointerDoc builds a document in Go whose individual points at an
// SNOTE record (@S1@), a NOTE record (@N1@) and an XRef that resolves to
// nothing (@X1@), both on the record and on an event substructure.
func sharedNotePointerDoc(headerVersion gedcom.Version) *gedcom.Document {
	xrefs := []string{"@S1@", "@N1@", "@X1@"}
	return &gedcom.Document{
		Header: &gedcom.Header{Version: headerVersion},
		Records: []*gedcom.Record{
			{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{
				XRef:      "@I1@",
				NoteXRefs: xrefs,
				Events: []*gedcom.Event{{
					Type:      gedcom.EventBirth,
					Date:      "1900",
					NoteXRefs: xrefs,
				}},
			}},
			{XRef: "@S1@", Type: gedcom.RecordTypeSharedNote, Entity: &gedcom.SharedNote{XRef: "@S1@", Text: "Shared"}},
			{XRef: "@N1@", Type: gedcom.RecordTypeNote, Entity: &gedcom.Note{XRef: "@N1@", Text: "Note"}},
		},
	}
}

// TestEncodeSharedNotePointerTag pins which tag a note pointer is written
// with on the entity path (#471): SNOTE only when the output is GEDCOM 7.0 and
// the pointer resolves to an SNOTE record; NOTE otherwise.
func TestEncodeSharedNotePointerTag(t *testing.T) {
	sevenO := []string{"1 SNOTE @S1@", "1 NOTE @N1@", "1 NOTE @X1@", "2 SNOTE @S1@", "2 NOTE @N1@", "2 NOTE @X1@"}
	allNOTE := []string{"1 NOTE @S1@", "1 NOTE @N1@", "1 NOTE @X1@", "2 NOTE @S1@", "2 NOTE @N1@", "2 NOTE @X1@"}

	tests := []struct {
		name   string
		header gedcom.Version
		target gedcom.Version
		want   []string
	}{
		{"target 7.0", "", gedcom.Version70, sevenO},
		{"header 7.0, no target", gedcom.Version70, "", sevenO},
		{"target 7.0 overrides 5.5.1 header", gedcom.Version551, gedcom.Version70, sevenO},
		{"target 5.5.1", "", gedcom.Version551, allNOTE},
		{"target 5.5", "", gedcom.Version55, allNOTE},
		{"target 5.5.1 overrides 7.0 header", gedcom.Version70, gedcom.Version551, allNOTE},
		{"no version", "", "", allNOTE},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := DefaultOptions()
			opts.TargetVersion = tt.target
			encoders := map[string]func(*bytes.Buffer, *gedcom.Document) error{
				"EncodeWithOptions": func(b *bytes.Buffer, d *gedcom.Document) error { return EncodeWithOptions(b, d, opts) },
				"EncodeStreamingWithOptions": func(b *bytes.Buffer, d *gedcom.Document) error {
					return EncodeStreamingWithOptions(b, d, opts)
				},
			}
			for name, encode := range encoders {
				var buf bytes.Buffer
				if err := encode(&buf, sharedNotePointerDoc(tt.header)); err != nil {
					t.Fatalf("%s() error = %v", name, err)
				}
				out := buf.String()
				for _, line := range tt.want {
					if !strings.Contains(out, line+"\n") {
						t.Errorf("%s output missing %q:\n%s", name, line, out)
					}
				}
			}
		})
	}
}

// TestStreamEncoderKeepsNOTEPointers documents the streaming limit: a record
// passed to WriteRecord is written without the rest of the document, so a
// pointer cannot be resolved and stays NOTE even for 7.0. EncodeStreaming has
// the document and resolves as Encode does (TestEncodeSharedNotePointerTag).
func TestStreamEncoderKeepsNOTEPointers(t *testing.T) {
	doc := sharedNotePointerDoc(gedcom.Version70)
	var buf bytes.Buffer
	enc := NewStreamEncoder(&buf)
	if err := enc.WriteHeader(doc.Header); err != nil {
		t.Fatal(err)
	}
	for _, r := range doc.Records {
		if err := enc.WriteRecord(r); err != nil {
			t.Fatal(err)
		}
	}
	if err := enc.WriteTrailer(); err != nil {
		t.Fatal(err)
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "1 NOTE @S1@\n") {
		t.Errorf("streamed output = %q, want 1 NOTE @S1@", buf.String())
	}
}

// sharedNoteRoundTripFixture puts a pointer to an SNOTE record and one to a
// NOTE record on every note-bearing structure the entity writers emit.
const sharedNoteRoundTripFixture = `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME John /Doe/
2 SNOTE @S1@
2 NOTE @N1@
1 BIRT
2 DATE 1900
2 PLAC Springfield
3 SNOTE @S1@
3 NOTE @N1@
2 OBJE @M1@
3 SNOTE @S1@
3 NOTE @N1@
2 SNOTE @S1@
2 NOTE @N1@
1 OCCU Blacksmith
2 SNOTE @S1@
2 NOTE @N1@
1 BAPL
2 DATE 1 JAN 1990
2 SNOTE @S1@
2 NOTE @N1@
1 FAMC @F1@
2 SNOTE @S1@
2 NOTE @N1@
1 FAMS @F1@
2 SNOTE @S1@
2 NOTE @N1@
1 ASSO @I2@
2 ROLE WITN
2 SNOTE @S1@
2 NOTE @N1@
1 SOUR @SR1@
2 PAGE 12
2 SNOTE @S1@
2 NOTE @N1@
1 SNOTE @S1@
1 NOTE @N1@
1 CHAN
2 DATE 1 JAN 2020
2 SNOTE @S1@
2 NOTE @N1@
0 @I2@ INDI
1 NAME Jane /Roe/
0 @F1@ FAM
1 HUSB @I1@
1 SNOTE @S1@
1 NOTE @N1@
0 @SR1@ SOUR
1 TITL A Source
1 REPO @R1@
2 SNOTE @S1@
2 NOTE @N1@
1 SNOTE @S1@
1 NOTE @N1@
0 @R1@ REPO
1 NAME A Repository
1 SNOTE @S1@
1 NOTE @N1@
0 @M1@ OBJE
1 FILE photo.jpg
1 SNOTE @S1@
1 NOTE @N1@
0 @S1@ SNOTE Shared note
0 @N1@ NOTE Plain note
0 TRLR
`

// noteXRefsBySite collects the note pointers of every note-bearing structure
// in doc, keyed by a label naming the structure.
func noteXRefsBySite(doc *gedcom.Document) map[string][]string {
	sites := map[string][]string{}
	add := func(site string, xrefs []string) { sites[site] = xrefs }

	indi := doc.GetIndividual("@I1@")
	add("INDI", indi.NoteXRefs)
	add("INDI.CHAN", indi.ChangeDate.NoteXRefs)
	add("INDI.NAME", indi.Names[0].NoteXRefs)
	add("INDI.BIRT", indi.Events[0].NoteXRefs)
	add("INDI.BIRT.PLAC", indi.Events[0].PlaceDetail.NoteXRefs)
	add("INDI.BIRT.OBJE", indi.Events[0].Media[0].NoteXRefs)
	add("INDI.OCCU", indi.Attributes[0].NoteXRefs)
	add("INDI.BAPL", indi.LDSOrdinances[0].NoteXRefs)
	add("INDI.FAMC", indi.ChildInFamilies[0].NoteXRefs)
	add("INDI.FAMS", indi.SpouseInFamilies[0].NoteXRefs)
	add("INDI.ASSO", indi.Associations[0].NoteXRefs)
	add("INDI.SOUR", indi.SourceCitations[0].NoteXRefs)
	add("FAM", doc.GetFamily("@F1@").NoteXRefs)
	src := doc.GetSource("@SR1@")
	add("SOUR", src.NoteXRefs)
	add("SOUR.REPO", src.RepositoryLinks[0].NoteXRefs)
	add("REPO", doc.GetRepository("@R1@").NoteXRefs)
	media := doc.GetMediaObject("@M1@")
	add("OBJE", media.NoteXRefs)
	add("OBJE shared", media.SharedNoteXRefs)
	return sites
}

// TestSharedNotePointersRoundTrip7 asserts a 7.0 document encoded from its
// entities re-decodes with the same note pointers on every structure (#471),
// and that the SNOTE-record pointer is written as SNOTE wherever it occurs.
func TestSharedNotePointersRoundTrip7(t *testing.T) {
	doc, err := decoder.Decode(strings.NewReader(sharedNoteRoundTripFixture))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	want := noteXRefsBySite(doc)
	for site, xrefs := range want {
		if len(xrefs) == 0 {
			t.Fatalf("fixture decoded no note pointers on %s", site)
		}
	}

	out := encodeFromEntities(t, sharedNoteRoundTripFixture)
	if strings.Contains(out, " NOTE @S1@\n") {
		t.Errorf("SNOTE-record pointer written as NOTE:\n%s", out)
	}

	redoc, err := decoder.Decode(strings.NewReader(out))
	if err != nil {
		t.Fatalf("re-Decode() error = %v", err)
	}
	got := noteXRefsBySite(redoc)
	for site, xrefs := range want {
		if !reflect.DeepEqual(got[site], xrefs) {
			t.Errorf("%s NoteXRefs = %v after round trip, want %v", site, got[site], xrefs)
		}
	}
}
