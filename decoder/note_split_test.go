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
1 SNOTE @N1@
0 @SUB1@ SUBM
1 NAME A Submitter
1 NOTE Inline submitter note
0 @O1@ OBJE
1 FILE photo.jpg
2 FORM image/jpeg
1 NOTE Inline media note
1 NOTE @N1@
1 SNOTE @N1@
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
			name: "Repository routes NOTE and SNOTE xrefs through split path",
			got: func() recordNotes {
				r := doc.GetRepository("@R1@")
				return recordNotes{r.NoteXRefs, r.InlineNotes, r.Notes}
			}(),
			want: recordNotes{
				xrefs:  []string{"@N1@", "@N1@"},
				inline: nil,
				legacy: []string{"@N1@", "@N1@"},
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
			name: "MediaObject routes NOTE and SNOTE through split path",
			got: func() recordNotes {
				m := doc.GetMediaObject("@O1@")
				return recordNotes{m.NoteXRefs, m.InlineNotes, m.Notes}
			}(),
			want: recordNotes{
				xrefs:  []string{"@N1@", "@N1@"},
				inline: []string{"Inline media note"},
				legacy: []string{"Inline media note", "@N1@", "@N1@"},
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

	// A media SNOTE is also tracked in SharedNoteXRefs (the GEDCOM 7.0 form used
	// for version detection) in addition to the split-note path.
	if got, want := doc.GetMediaObject("@O1@").SharedNoteXRefs, []string{"@N1@"}; !reflect.DeepEqual(got, want) {
		t.Errorf("MediaObject SharedNoteXRefs = %#v, want %#v", got, want)
	}
}

// TestDecodeSubstructureNoteSplit covers the five substructures given note
// fields by issue #472. Three of them (MediaLink, FamilyLink, PlaceDetail) have
// no legacy combined Notes slice, so only the split fields are asserted.
//
// The MediaLink case uses a padded pointer ("NOTE  @N1@"), which is the shape
// #426 preserves and which a naive pointer test on the raw value would
// misclassify as inline text.
func TestDecodeSubstructureNoteSplit(t *testing.T) {
	const input = `0 HEAD
1 GEDC
2 VERS 7.0
1 CHAR UTF-8
0 @N1@ NOTE A shared note record
0 @I1@ INDI
1 FAMC @F1@
2 PEDI birth
2 NOTE Family link note
2 NOTE @N1@
1 ASSO @I2@
2 ROLE WITN
2 NOTE @N1@
2 NOTE Association note
1 OBJE @O1@
2 NOTE  @N1@
2 NOTE Media link note
3 CONT second line
1 BIRT
2 PLAC Springfield
3 NOTE Place note
3 SNOTE @N1@
0 @I2@ INDI
1 NAME Jane /Roe/
0 @F1@ FAM
1 HUSB @I1@
0 @S1@ SOUR
1 TITL A Source
1 REPO @R1@
2 CALN 123
2 NOTE @N1@
2 NOTE Repository link note
0 @R1@ REPO
1 NAME A Repository
0 @O1@ OBJE
1 FILE photo.jpg
0 TRLR`

	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("GetIndividual(@I1@) returned nil")
	}
	src := doc.GetSource("@S1@")
	if src == nil || src.RepositoryLink == nil {
		t.Fatal("GetSource(@S1@) returned no repository link")
	}

	tests := []struct {
		name string
		got  recordNotes
		want recordNotes
	}{
		{
			name: "FamilyLink splits inline and xref, no legacy slice",
			got: func() recordNotes {
				l := indi.ChildInFamilies[0]
				return recordNotes{l.NoteXRefs, l.InlineNotes, nil}
			}(),
			want: recordNotes{
				xrefs:  []string{"@N1@"},
				inline: []string{"Family link note"},
			},
		},
		{
			name: "Association keeps legacy order alongside the split",
			got: func() recordNotes {
				a := indi.Associations[0]
				return recordNotes{a.NoteXRefs, a.InlineNotes, a.Notes}
			}(),
			want: recordNotes{
				xrefs:  []string{"@N1@"},
				inline: []string{"Association note"},
				legacy: []string{"@N1@", "Association note"},
			},
		},
		{
			name: "MediaLink trims a padded pointer and folds CONT",
			got: func() recordNotes {
				m := indi.Media[0]
				return recordNotes{m.NoteXRefs, m.InlineNotes, nil}
			}(),
			want: recordNotes{
				xrefs:  []string{"@N1@"},
				inline: []string{"Media link note\nsecond line"},
			},
		},
		{
			name: "PlaceDetail routes NOTE and SNOTE through the split path",
			got: func() recordNotes {
				p := indi.Events[0].PlaceDetail
				return recordNotes{p.NoteXRefs, p.InlineNotes, nil}
			}(),
			want: recordNotes{
				xrefs:  []string{"@N1@"},
				inline: []string{"Place note"},
			},
		},
		{
			name: "SourceRepositoryLink keeps legacy order alongside the split",
			got: recordNotes{
				src.RepositoryLink.NoteXRefs,
				src.RepositoryLink.InlineNotes,
				src.RepositoryLink.Notes,
			},
			want: recordNotes{
				xrefs:  []string{"@N1@"},
				inline: []string{"Repository link note"},
				legacy: []string{"@N1@", "Repository link note"},
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

// TestMediaLinkNoteNotReportedUnknown pins issue #470: a NOTE under an inline
// OBJE used to fall through parseMediaLink's default branch and be reported as
// an unknown tag.
func TestMediaLinkNoteNotReportedUnknown(t *testing.T) {
	const input = `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 OBJE
2 TITL A photo
2 FILE photo.jpg
3 FORM jpg
2 NOTE
0 TRLR`

	res, err := DecodeWithDiagnostics(strings.NewReader(input), DefaultOptions())
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics failed: %v", err)
	}
	for _, d := range res.Diagnostics {
		if d.Code == CodeUnknownTag {
			t.Errorf("unexpected diagnostic: %s", d.String())
		}
	}
	// A valueless NOTE still decodes to an empty inline note rather than
	// vanishing -- the tag was present in the input.
	m := res.Document.GetIndividual("@I1@").Media[0]
	if want := []string{""}; !reflect.DeepEqual(m.InlineNotes, want) {
		t.Errorf("MediaLink InlineNotes = %#v, want %#v", m.InlineNotes, want)
	}
}
