package encoder

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestTypedAccess787EntityRoundTrip pins issue #787: decode -> entity-encode
// -> decode keeps a citation's DATA.DATE TIME and PHRASE, a PLAC's EXIDs,
// repeated name pieces, and the MIME/LANG/TRAN of an individual's inline
// NOTE, each written where GEDCOM 7.0 places it.
func TestTypedAccess787EntityRoundTrip(t *testing.T) {
	input := "0 HEAD\n1 GEDC\n2 VERS 7.0\n" +
		"0 @I1@ INDI\n1 NAME Alex /Sample/\n2 GIVN Alex\n2 GIVN Alexander\n2 SURN Sample\n2 SURN Example\n" +
		"2 NPFX Dr.\n2 NPFX Prof.\n2 NSFX Jr.\n2 NSFX PhD\n2 NICK Lex\n2 NICK Xander\n2 SPFX van\n2 SPFX der\n" +
		"1 BIRT\n2 PLAC Sampletown\n3 EXID 999\n4 TYPE http://places.example.invalid/id\n3 EXID 1000\n" +
		"2 SOUR @S1@\n3 DATA\n4 DATE 2 JAN 1900\n5 TIME 08:00\n5 PHRASE the day after\n" +
		"1 NOTE Plain note\n" +
		"1 NOTE Line one\n2 MIME text/plain\n2 LANG en\n2 TRAN Ligne une\n3 LANG fr\n" +
		"0 @S1@ SOUR\n0 TRLR\n"
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	want := doc.GetIndividual("@I1@").Clone()
	for _, rec := range doc.Records {
		rec.Tags = nil // force the entity path
	}
	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	out := buf.String()
	for _, frag := range []string{
		"2 GIVN Alex\n2 GIVN Alexander\n2 SURN Sample\n2 SURN Example\n2 NPFX Dr.\n2 NPFX Prof.\n" +
			"2 NSFX Jr.\n2 NSFX PhD\n2 NICK Lex\n2 NICK Xander\n2 SPFX van\n2 SPFX der\n",
		"2 PLAC Sampletown\n3 EXID 999\n4 TYPE http://places.example.invalid/id\n3 EXID 1000\n",
		"3 DATA\n4 DATE 2 JAN 1900\n5 TIME 08:00\n5 PHRASE the day after\n",
		"1 NOTE Plain note\n1 NOTE Line one\n2 MIME text/plain\n2 LANG en\n2 TRAN Ligne une\n3 LANG fr\n",
	} {
		if !strings.Contains(out, frag) {
			t.Errorf("output missing %q:\n%s", frag, out)
		}
	}

	again, err := decoder.Decode(strings.NewReader(out))
	if err != nil {
		t.Fatalf("re-Decode() error = %v", err)
	}
	got := again.GetIndividual("@I1@")
	if !reflect.DeepEqual(got.Names[0], want.Names[0]) {
		t.Errorf("name = %+v, want %+v", got.Names[0], want.Names[0])
	}
	if !reflect.DeepEqual(got.Events[0].PlaceDetail, want.Events[0].PlaceDetail) {
		t.Errorf("place = %+v, want %+v", got.Events[0].PlaceDetail, want.Events[0].PlaceDetail)
	}
	if !reflect.DeepEqual(got.Events[0].SourceCitations[0].Data, want.Events[0].SourceCitations[0].Data) {
		t.Errorf("citation data = %+v, want %+v", got.Events[0].SourceCitations[0].Data, want.Events[0].SourceCitations[0].Data)
	}
	if !reflect.DeepEqual(got.InlineNoteDetails, want.InlineNoteDetails) || len(got.InlineNoteDetails) != 2 {
		t.Errorf("InlineNoteDetails = %+v, want %+v", got.InlineNoteDetails, want.InlineNoteDetails)
	}
}

// TestCitationDataPhraseOnlyDateByVersion: a citation date with only a TIME
// or PHRASE writes an empty DATE to carry it for 7.0 output, and nothing for
// 5.5.1 output, which has no empty DATE (as ageToTags does for AGE).
func TestCitationDataPhraseOnlyDateByVersion(t *testing.T) {
	data := &gedcom.SourceCitationData{DatePhrase: "Undated entry"}
	tests := []struct {
		version gedcom.Version
		want    []string
	}{
		{gedcom.Version70, []string{"1 DATA ", "2 DATE ", "3 PHRASE Undated entry"}},
		{gedcom.Version551, []string{"1 DATA "}},
	}
	for _, tt := range tests {
		t.Run(string(tt.version), func(t *testing.T) {
			tags := sourceCitationDataToTags(data, 1, &EncodeOptions{version: tt.version})
			var got []string
			for _, tag := range tags {
				got = append(got, fmt.Sprintf("%d %s %s", tag.Level, tag.Tag, tag.Value))
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("tags = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestInlineNoteDetailsOutOfStep: a detail past the end of InlineNotes has
// no note to qualify and is not written; a nil detail writes nothing.
func TestInlineNoteDetailsOutOfStep(t *testing.T) {
	tags := notesWithDetailsToTags(nil, []string{"Only note"},
		[]*gedcom.InlineNoteDetail{nil, {Language: "fr"}}, 1, nil)
	if len(tags) != 1 || tags[0].Tag != "NOTE" || tags[0].Value != "Only note" {
		t.Errorf("tags = %+v, want only the NOTE line", tags)
	}
}

// TestNamePieceEmptyFirstWithRepeats: an empty first value followed by
// repeats is written as an empty line, so a re-decode keeps the split
// between the first value and the repeats instead of promoting the first
// repeat.
func TestNamePieceEmptyFirstWithRepeats(t *testing.T) {
	tags := appendNamePiece(nil, "GIVN", "", []string{"Alexander"}, 2, nil)
	var got []string
	for _, tag := range tags {
		got = append(got, fmt.Sprintf("%d %s %q", tag.Level, tag.Tag, tag.Value))
	}
	want := []string{`2 GIVN ""`, `2 GIVN "Alexander"`}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("tags = %v, want %v", got, want)
	}
	if tags := appendNamePiece(nil, "GIVN", "", nil, 2, nil); len(tags) != 0 {
		t.Errorf("empty piece without repeats wrote %+v, want nothing", tags)
	}
}
