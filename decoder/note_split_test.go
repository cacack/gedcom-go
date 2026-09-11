package decoder

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// recordNotes is the common shape of the split note fields populated by
// appendRecordNote, used to assert decode results across record types.
type recordNotes struct {
	xrefs  []string
	inline []string
}

// TestDecodeRecordNoteSplit verifies that record-level NOTE tags are split into
// NoteXRefs (pointer-shaped values) and InlineNotes (text values, with CONT/CONC
// folded). It covers every note-bearing record type.
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
				return recordNotes{i.NoteXRefs, i.InlineNotes}
			}(),
			want: recordNotes{
				xrefs:  []string{"@N1@"},
				inline: []string{"Inline individual note", "Multi line note\nsecond linecontinued"},
			},
		},
		{
			name: "Family xref before inline preserves order",
			got: func() recordNotes {
				f := doc.GetFamily("@F1@")
				return recordNotes{f.NoteXRefs, f.InlineNotes}
			}(),
			want: recordNotes{
				xrefs:  []string{"@N1@"},
				inline: []string{"Inline family note"},
			},
		},
		{
			name: "Source inline before xref",
			got: func() recordNotes {
				s := doc.GetSource("@S1@")
				return recordNotes{s.NoteXRefs, s.InlineNotes}
			}(),
			want: recordNotes{
				xrefs:  []string{"@N1@"},
				inline: []string{"Inline source note"},
			},
		},
		{
			name: "Repository routes NOTE and SNOTE xrefs through split path",
			got: func() recordNotes {
				r := doc.GetRepository("@R1@")
				return recordNotes{r.NoteXRefs, r.InlineNotes}
			}(),
			want: recordNotes{
				xrefs:  []string{"@N1@", "@N1@"},
				inline: nil,
			},
		},
		{
			name: "Submitter inline only",
			got: func() recordNotes {
				s := doc.GetSubmitter("@SUB1@")
				return recordNotes{s.NoteXRefs, s.InlineNotes}
			}(),
			want: recordNotes{
				xrefs:  nil,
				inline: []string{"Inline submitter note"},
			},
		},
		{
			// MediaObject is the one record type that keeps SNOTE pointers out
			// of NoteXRefs: they go to SharedNoteXRefs instead (#499), so only
			// the NOTE pointer shows up here.
			name: "MediaObject keeps SNOTE out of NoteXRefs",
			got: func() recordNotes {
				m := doc.GetMediaObject("@O1@")
				return recordNotes{m.NoteXRefs, m.InlineNotes}
			}(),
			want: recordNotes{
				xrefs:  []string{"@N1@"},
				inline: []string{"Inline media note"},
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
		})
	}

	// A media SNOTE lands in SharedNoteXRefs (the GEDCOM 7.0 form used for
	// version detection) and nowhere else.
	if got, want := doc.GetMediaObject("@O1@").SharedNoteXRefs, []string{"@N1@"}; !reflect.DeepEqual(got, want) {
		t.Errorf("MediaObject SharedNoteXRefs = %#v, want %#v", got, want)
	}
}

// TestDecodeSubstructureNoteSplit covers the five substructures given note
// fields by issue #472, and both FamilyLink sites -- FAMC and, since #534,
// FAMS.
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
1 FAMS @F1@
2 NOTE Spouse link note
3 CONT second line
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
			name: "FamilyLink under FAMC splits inline and xref",
			got: func() recordNotes {
				l := indi.ChildInFamilies[0]
				return recordNotes{l.NoteXRefs, l.InlineNotes}
			}(),
			want: recordNotes{
				xrefs:  []string{"@N1@"},
				inline: []string{"Family link note"},
			},
		},
		{
			// The FAMS half of #534: a spouse link takes the same parse path
			// as a child link, so its notes split identically. The CONT also
			// pins that appendRecordNote folds continuations under a FAMS --
			// parity with FAMC, but FAMS is the newer caller and no file in
			// testdata/ carries a continued note there.
			name: "FamilyLink under FAMS splits inline and xref",
			got: func() recordNotes {
				l := indi.SpouseInFamilies[0]
				return recordNotes{l.NoteXRefs, l.InlineNotes}
			}(),
			want: recordNotes{
				xrefs:  []string{"@N1@"},
				inline: []string{"Spouse link note\nsecond line"},
			},
		},
		{
			name: "Association splits inline and xref",
			got: func() recordNotes {
				a := indi.Associations[0]
				return recordNotes{a.NoteXRefs, a.InlineNotes}
			}(),
			want: recordNotes{
				xrefs:  []string{"@N1@"},
				inline: []string{"Association note"},
			},
		},
		{
			name: "MediaLink trims a padded pointer and folds CONT",
			got: func() recordNotes {
				m := indi.Media[0]
				return recordNotes{m.NoteXRefs, m.InlineNotes}
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
				return recordNotes{p.NoteXRefs, p.InlineNotes}
			}(),
			want: recordNotes{
				xrefs:  []string{"@N1@"},
				inline: []string{"Place note"},
			},
		},
		{
			name: "SourceRepositoryLink splits inline and xref",
			got: recordNotes{
				src.RepositoryLink.NoteXRefs,
				src.RepositoryLink.InlineNotes,
			},
			want: recordNotes{
				xrefs:  []string{"@N1@"},
				inline: []string{"Repository link note"},
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

// TestMediaObjectNotePointersPartition guards the invariant issue #499
// established: a MediaObject's NoteXRefs holds NOTE pointers only and its
// SharedNoteXRefs holds SNOTE pointers only, so the two partition. Before the
// change the decoder put every SNOTE pointer in both, and AllNotes ran an
// O(n^2) dedup to undo it; a caller who trusted the doc comment and
// concatenated the slices saw every shared note twice.
func TestMediaObjectNotePointersPartition(t *testing.T) {
	const input = `0 HEAD
1 GEDC
2 VERS 7.0
1 CHAR UTF-8
0 @N1@ NOTE A NOTE record
0 @S1@ SNOTE An SNOTE record
0 @O1@ OBJE
1 FILE photo.jpg
2 FORM image/jpeg
1 NOTE Inline media note
1 NOTE @N1@
1 SNOTE @S1@
0 TRLR`

	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	media := doc.GetMediaObject("@O1@")
	if media == nil {
		t.Fatal("GetMediaObject(@O1@) returned nil")
	}

	if want := []string{"@N1@"}; !reflect.DeepEqual(media.NoteXRefs, want) {
		t.Errorf("NoteXRefs = %#v, want %#v (no SNOTE pointer)", media.NoteXRefs, want)
	}
	if want := []string{"@S1@"}; !reflect.DeepEqual(media.SharedNoteXRefs, want) {
		t.Errorf("SharedNoteXRefs = %#v, want %#v", media.SharedNoteXRefs, want)
	}

	// Every note, once: the inline text, then each pointer resolved in field
	// order. AllNotes no longer dedups, so a re-introduced overlap fails here.
	want := []string{"Inline media note", "A NOTE record", "An SNOTE record"}
	if got := media.AllNotes(doc); !reflect.DeepEqual(got, want) {
		t.Errorf("AllNotes() = %#v, want %#v", got, want)
	}
}

// TestMediaSharedNoteStillRequiresGEDCOM7 guards the reason #499 partitioned
// the two slices rather than merging them into one. Document.RequiresGEDCOM7
// reads len(MediaObject.SharedNoteXRefs) > 0 as its SNOTE-on-media signal; a
// merged slice would leave no way to tell a NOTE pointer from an SNOTE pointer
// and the signal would have no substitute.
//
// The media record is lifted into a document of its own because an SNOTE
// *record* is itself a 7.0-only signal -- leaving it in would prove nothing
// about the pointer.
func TestMediaSharedNoteStillRequiresGEDCOM7(t *testing.T) {
	const input = `0 HEAD
1 GEDC
2 VERS 7.0
1 CHAR UTF-8
0 @S1@ SNOTE An SNOTE record
0 @O1@ OBJE
1 FILE photo.jpg
2 FORM image/jpeg
1 SNOTE @S1@
0 TRLR`

	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	mediaOnly := &gedcom.Document{Records: []*gedcom.Record{doc.XRefMap["@O1@"]}}
	if !mediaOnly.RequiresGEDCOM7() {
		t.Error("RequiresGEDCOM7() = false, want true for a media object carrying only an SNOTE pointer")
	}
	if got := mediaOnly.MinimumVersion(); got != gedcom.Version70 {
		t.Errorf("MinimumVersion() = %v, want %v", got, gedcom.Version70)
	}
}

// TestMediaObjectNonPointerSNOTEKeepsItsText pins the classification the #499
// partition depends on. SNOTE is pointer-only in the 7.0 spec, but a lenient
// decoder still meets "1 SNOTE some text" in the wild, and an entry in
// SharedNoteXRefs that resolves to no record is dropped without trace by
// allNotes (gedcom/notes.go). Filing an unclassified value there therefore
// loses it from the typed model entirely -- notes silently disappearing is the
// opposite of the Lossless Representation principle, so the pointer test in
// parseMediaObject is load-bearing rather than cosmetic.
func TestMediaObjectNonPointerSNOTEKeepsItsText(t *testing.T) {
	const input = `0 HEAD
1 GEDC
2 VERS 7.0
1 CHAR UTF-8
0 @O1@ OBJE
1 FILE photo.jpg
2 FORM image/jpeg
1 SNOTE this is not a pointer
2 CONT second line
0 TRLR`

	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	media := doc.GetMediaObject("@O1@")
	if media == nil {
		t.Fatal("GetMediaObject(@O1@) = nil")
	}

	// The value is not a pointer, so it must not sit in a pointer-only slice.
	if len(media.SharedNoteXRefs) != 0 {
		t.Errorf("SharedNoteXRefs = %q, want empty: a non-pointer value belongs in InlineNotes", media.SharedNoteXRefs)
	}
	want := "this is not a pointer\nsecond line"
	if len(media.InlineNotes) != 1 || media.InlineNotes[0] != want {
		t.Errorf("InlineNotes = %q, want [%q] with the CONT folded in", media.InlineNotes, want)
	}

	// The whole point: the text still reaches a caller.
	all := media.AllNotes(doc)
	if len(all) != 1 || all[0] != want {
		t.Errorf("AllNotes() = %q, want [%q] -- the note text was lost", all, want)
	}
}
