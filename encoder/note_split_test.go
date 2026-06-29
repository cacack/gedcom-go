package encoder

import (
	"reflect"
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// TestRecordNotesToEncode verifies the encoder prefers the split
// NoteXRefs/InlineNotes fields (emitting XRefs first, then inline text) and
// falls back to the deprecated combined Notes slice only when both split fields
// are empty.
func TestRecordNotesToEncode(t *testing.T) {
	tests := []struct {
		name      string
		noteXRefs []string
		inline    []string
		notes     []string
		want      []string
	}{
		{
			name:  "falls back to legacy Notes when split fields empty",
			notes: []string{"@N1@", "inline text"},
			want:  []string{"@N1@", "inline text"},
		},
		{
			name:      "split fields preferred, xrefs before inline",
			noteXRefs: []string{"@N1@", "@N2@"},
			inline:    []string{"first inline", "second inline"},
			notes:     []string{"ignored legacy value"},
			want:      []string{"@N1@", "@N2@", "first inline", "second inline"},
		},
		{
			name:      "only xrefs populated",
			noteXRefs: []string{"@N1@"},
			notes:     []string{"ignored"},
			want:      []string{"@N1@"},
		},
		{
			name:   "only inline populated",
			inline: []string{"just inline"},
			notes:  []string{"ignored"},
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
