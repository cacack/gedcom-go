package decoder

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Issue #787: four GEDCOM 7.0 values the oxidged fixture (full70.ged) carries
// that never reached the typed model.

const head70 = "0 HEAD\n1 GEDC\n2 VERS 7.0\n"

func decodeIndividual(t *testing.T, input string) *gedcom.Individual {
	t.Helper()
	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("no @I1@")
	}
	return indi
}

func TestCitationDataDateTimeAndPhrase(t *testing.T) {
	indi := decodeIndividual(t, head70+
		"0 @I1@ INDI\n1 BIRT\n2 SOUR @S1@\n3 DATA\n4 DATE 2 JAN 1900\n5 TIME 08:00\n5 PHRASE the day after\n"+
		"4 TEXT Transcribed entry\n"+
		"0 @S1@ SOUR\n0 TRLR\n")
	data := indi.Events[0].SourceCitations[0].Data
	if data.Date != "2 JAN 1900" || data.Time != "08:00" || data.DatePhrase != "the day after" {
		t.Errorf("Data = {Date %q, Time %q, DatePhrase %q}, want {2 JAN 1900, 08:00, the day after}",
			data.Date, data.Time, data.DatePhrase)
	}
	if len(data.Text) != 1 || data.Text[0].Value != "Transcribed entry" {
		t.Errorf("Data.Text = %v, want the one TEXT", data.Text)
	}
}

func TestCitationDataPhraseOnlyDate(t *testing.T) {
	indi := decodeIndividual(t, head70+
		"0 @I1@ INDI\n1 SOUR @S1@\n2 DATA\n3 DATE\n4 PHRASE Undated entry\n0 @S1@ SOUR\n0 TRLR\n")
	data := indi.SourceCitations[0].Data
	if data.Date != "" || data.DatePhrase != "Undated entry" {
		t.Errorf("Data = {Date %q, DatePhrase %q}, want {\"\", Undated entry}", data.Date, data.DatePhrase)
	}
}

func TestPlaceExternalIDs(t *testing.T) {
	indi := decodeIndividual(t, head70+
		"0 @I1@ INDI\n1 BIRT\n2 PLAC Sampletown\n3 MAP\n4 LATI N10.5\n4 LONG W20.25\n"+
		"3 EXID 999\n4 TYPE http://places.example.invalid/id\n3 EXID 1000\n0 TRLR\n")
	got := indi.Events[0].PlaceDetail.ExternalIDs
	want := []*gedcom.ExternalID{
		{Value: "999", Type: "http://places.example.invalid/id"},
		{Value: "1000"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("PlaceDetail.ExternalIDs = %+v, want %+v", got, want)
	}
}

func TestRepeatedNamePieces(t *testing.T) {
	indi := decodeIndividual(t, head70+
		"0 @I1@ INDI\n1 NAME Alex /Sample/ Jr.\n"+
		"2 NPFX Dr.\n2 NPFX Prof.\n2 GIVN Alex\n2 GIVN Alexander\n2 GIVN Al\n"+
		"2 NICK Lex\n2 NICK Xander\n2 SPFX van\n2 SPFX der\n"+
		"2 SURN Sample\n2 SURN Example\n2 NSFX Jr.\n2 NSFX PhD\n0 TRLR\n")
	name := indi.Names[0]
	scalars := []string{name.Given, name.Surname, name.Prefix, name.Suffix, name.Nickname, name.SurnamePrefix}
	if want := []string{"Alex", "Sample", "Dr.", "Jr.", "Lex", "van"}; !reflect.DeepEqual(scalars, want) {
		t.Errorf("first pieces = %q, want %q", scalars, want)
	}
	more := [][]string{
		name.AdditionalGiven, name.AdditionalSurnames, name.AdditionalPrefixes,
		name.AdditionalSuffixes, name.AdditionalNicknames, name.AdditionalSurnamePrefixes,
	}
	want := [][]string{{"Alexander", "Al"}, {"Example"}, {"Prof."}, {"PhD"}, {"Xander"}, {"der"}}
	if !reflect.DeepEqual(more, want) {
		t.Errorf("additional pieces = %q, want %q", more, want)
	}
}

func TestSingleNamePiecesHaveNoAdditional(t *testing.T) {
	indi := decodeIndividual(t, "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n"+
		"0 @I1@ INDI\n1 NAME John /Smith/\n2 GIVN Johann\n2 SURN Smith\n0 TRLR\n")
	name := indi.Names[0]
	if name.Given != "Johann" || name.Surname != "Smith" {
		t.Errorf("Given, Surname = %q, %q, want Johann, Smith", name.Given, name.Surname)
	}
	if name.AdditionalGiven != nil || name.AdditionalSurnames != nil {
		t.Errorf("Additional = %q, %q, want nil", name.AdditionalGiven, name.AdditionalSurnames)
	}
}

func TestInlineNoteDetails(t *testing.T) {
	indi := decodeIndividual(t, head70+
		"0 @I1@ INDI\n1 NOTE Plain note\n1 SNOTE @N1@\n"+
		"1 NOTE Line one\n2 CONT second line\n2 MIME text/plain\n2 LANG en\n"+
		"2 TRAN Ligne une\n3 LANG fr\n2 TRAN <p>Zeile eins</p>\n3 MIME text/html\n3 LANG de\n2 SOUR @S1@\n"+
		"0 @N1@ SNOTE Shared\n0 @S1@ SOUR\n0 TRLR\n")
	if want := []string{"Plain note", "Line one\nsecond line"}; !reflect.DeepEqual(indi.InlineNotes, want) {
		t.Fatalf("InlineNotes = %q, want %q", indi.InlineNotes, want)
	}
	want := []*gedcom.InlineNoteDetail{
		nil,
		{
			MIME:     "text/plain",
			Language: "en",
			Translations: []*gedcom.SharedNoteTranslation{
				{Value: "Ligne une", Language: "fr"},
				{Value: "<p>Zeile eins</p>", MIME: "text/html", Language: "de"},
			},
		},
	}
	if !reflect.DeepEqual(indi.InlineNoteDetails, want) {
		t.Errorf("InlineNoteDetails = %+v, want %+v", indi.InlineNoteDetails, want)
	}
}

func TestInlineNoteDetailsNilWithoutSubstructures(t *testing.T) {
	indi := decodeIndividual(t, head70+"0 @I1@ INDI\n1 NOTE One\n1 NOTE Two\n2 CONT more\n0 TRLR\n")
	if indi.InlineNoteDetails != nil {
		t.Errorf("InlineNoteDetails = %+v, want nil", indi.InlineNoteDetails)
	}
}
