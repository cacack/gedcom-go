package decoder

import (
	"reflect"
	"strings"
	"testing"
)

// recordNotes is the common shape of the split note fields populated by
// appendRecordNote, used to assert decode results across record types.
type recordNotes struct {
	xrefs  []string
	inline []string
	legacy []string
}

// TestDecodeRecordNoteSplit verifies that record-level NOTE tags are split into
// NoteXRefs (pointer-shaped values) and InlineNotes (text values, with CONT/CONC
// folded), while the deprecated combined Notes slice preserves the original
// order for backward compatibility. It covers every note-bearing record type.
func TestDecodeRecordNoteSplit(t *testing.T) {
	const input = `0 HEAD
1 GEDC
2 VERS 7.0
1 CHAR UTF-8
0 @N1@ NOTE A shared note record
0 @I1@ INDI
1 NOTE Inline individual note
1 NOTE @N1@
1 NOTE Multi line note
2 CONT second line
2 CONC continued
0 @F1@ FAM
1 NOTE @N1@
1 NOTE Inline family note
0 @S1@ SOUR
1 TITL A Source
1 NOTE Inline source note
1 NOTE @N1@
0 @R1@ REPO
1 NAME A Repository
1 NOTE @N1@
0 @SUB1@ SUBM
1 NAME A Submitter
1 NOTE Inline submitter note
0 @O1@ OBJE
1 FILE photo.jpg
2 FORM image/jpeg
1 NOTE Inline media note
1 NOTE @N1@
0 TRLR`

	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	tests := []struct {
		name string
		got  recordNotes
		want recordNotes
	}{
		{
			name: "Individual splits inline, xref, and CONT/CONC",
			got: func() recordNotes {
				i := doc.GetIndividual("@I1@")
				return recordNotes{i.NoteXRefs, i.InlineNotes, i.Notes}
			}(),
			want: recordNotes{
				xrefs:  []string{"@N1@"},
				inline: []string{"Inline individual note", "Multi line note\nsecond linecontinued"},
				legacy: []string{"Inline individual note", "@N1@", "Multi line note\nsecond linecontinued"},
			},
		},
		{
			name: "Family xref before inline preserves order",
			got: func() recordNotes {
				f := doc.GetFamily("@F1@")
				return recordNotes{f.NoteXRefs, f.InlineNotes, f.Notes}
			}(),
			want: recordNotes{
				xrefs:  []string{"@N1@"},
				inline: []string{"Inline family note"},
				legacy: []string{"@N1@", "Inline family note"},
			},
		},
		{
			name: "Source inline before xref",
			got: func() recordNotes {
				s := doc.GetSource("@S1@")
				return recordNotes{s.NoteXRefs, s.InlineNotes, s.Notes}
			}(),
			want: recordNotes{
				xrefs:  []string{"@N1@"},
				inline: []string{"Inline source note"},
				legacy: []string{"Inline source note", "@N1@"},
			},
		},
		{
			name: "Repository xref only",
			got: func() recordNotes {
				r := doc.GetRepository("@R1@")
				return recordNotes{r.NoteXRefs, r.InlineNotes, r.Notes}
			}(),
			want: recordNotes{
				xrefs:  []string{"@N1@"},
				inline: nil,
				legacy: []string{"@N1@"},
			},
		},
		{
			name: "Submitter inline only",
			got: func() recordNotes {
				s := doc.GetSubmitter("@SUB1@")
				return recordNotes{s.NoteXRefs, s.InlineNotes, s.Notes}
			}(),
			want: recordNotes{
				xrefs:  nil,
				inline: []string{"Inline submitter note"},
				legacy: []string{"Inline submitter note"},
			},
		},
		{
			name: "MediaObject inline before xref",
			got: func() recordNotes {
				m := doc.GetMediaObject("@O1@")
				return recordNotes{m.NoteXRefs, m.InlineNotes, m.Notes}
			}(),
			want: recordNotes{
				xrefs:  []string{"@N1@"},
				inline: []string{"Inline media note"},
				legacy: []string{"Inline media note", "@N1@"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(tt.got.xrefs, tt.want.xrefs) {
				t.Errorf("NoteXRefs = %#v, want %#v", tt.got.xrefs, tt.want.xrefs)
			}
			if !reflect.DeepEqual(tt.got.inline, tt.want.inline) {
				t.Errorf("InlineNotes = %#v, want %#v", tt.got.inline, tt.want.inline)
			}
			if !reflect.DeepEqual(tt.got.legacy, tt.want.legacy) {
				t.Errorf("Notes = %#v, want %#v", tt.got.legacy, tt.want.legacy)
			}
		})
	}
}
