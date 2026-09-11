package encoder

import (
	"reflect"
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
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
// issue #472 emit NOTE tags for both the XRef pointer and inline text.
func TestSubstructureTypesNoteSplit(t *testing.T) {
	want := []string{"@N1@", "An inline note"}

	ml := &gedcom.MediaLink{MediaXRef: "@O1@", NoteXRefs: []string{"@N1@"},
		InlineNotes: []string{"An inline note"}}
	if got := noteTagValues(mediaLinkToTags(ml, 1, nil)); !reflect.DeepEqual(got, want) {
		t.Errorf("mediaLinkToTags() NOTE values = %#v, want %#v", got, want)
	}

	fl := &gedcom.FamilyLink{FamilyXRef: "@F1@", NoteXRefs: []string{"@N1@"},
		InlineNotes: []string{"An inline note"}}
	if got := noteTagValues(familyLinkToTags(fl, 1, nil)); !reflect.DeepEqual(got, want) {
		t.Errorf("familyLinkToTags() NOTE values = %#v, want %#v", got, want)
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
// both. They are still written as NOTE, which is what came out before the
// partition -- the 7.0 SNOTE form is issue #471.
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
