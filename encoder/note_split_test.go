package encoder

import (
	"reflect"
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// TestRecordNotesToEncode verifies the encoder prefers the order-preserving
// legacy Notes slice whenever it is populated (keeping round-trips lossless),
// and only combines the split NoteXRefs/InlineNotes fields (XRefs first, then
// inline text) when Notes is empty.
func TestRecordNotesToEncode(t *testing.T) {
	tests := []struct {
		name      string
		noteXRefs []string
		inline    []string
		notes     []string
		want      []string
	}{
		{
			name:  "uses legacy Notes when split fields empty",
			notes: []string{"@N1@", "inline text"},
			want:  []string{"@N1@", "inline text"},
		},
		{
			name:      "legacy Notes preserves original interleaved order",
			noteXRefs: []string{"@N1@", "@N2@"},
			inline:    []string{"first inline", "second inline"},
			notes:     []string{"@N1@", "first inline", "@N2@", "second inline"},
			want:      []string{"@N1@", "first inline", "@N2@", "second inline"},
		},
		{
			name:      "combines split fields when legacy Notes empty",
			noteXRefs: []string{"@N1@", "@N2@"},
			inline:    []string{"first inline", "second inline"},
			want:      []string{"@N1@", "@N2@", "first inline", "second inline"},
		},
		{
			name:      "only xrefs populated, no legacy",
			noteXRefs: []string{"@N1@"},
			want:      []string{"@N1@"},
		},
		{
			name:   "only inline populated, no legacy",
			inline: []string{"just inline"},
			want:   []string{"just inline"},
		},
		{
			name: "all empty returns nil legacy",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := recordNotesToEncode(tt.noteXRefs, tt.inline, tt.notes)
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

// TestSubmitterRepositoryMediaNoteSplit confirms Submitter, Repository, and
// MediaObject records built with only the split note fields still emit NOTE
// tags for both the XRef pointer and inline text.
func TestSubmitterRepositoryMediaNoteSplit(t *testing.T) {
	want := []string{"@N1@", "An inline note"}

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

// TestNoteOrderPreservedFromLegacy confirms that when the legacy Notes slice is
// populated (as it always is for decoded documents), the encoder emits notes in
// that original interleaved order rather than reordering xrefs ahead of inline
// text. This guards the lossless round-trip of interleaved record notes.
func TestNoteOrderPreservedFromLegacy(t *testing.T) {
	indi := &gedcom.Individual{
		NoteXRefs:   []string{"@N1@", "@N2@"},
		InlineNotes: []string{"inline text"},
		Notes:       []string{"@N1@", "inline text", "@N2@"},
	}

	got := noteTagValues(individualToTags(indi, nil))
	want := []string{"@N1@", "inline text", "@N2@"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("individualToTags() NOTE order = %#v, want %#v", got, want)
	}
}
