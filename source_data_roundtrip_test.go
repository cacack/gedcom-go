package gedcomgo

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Issue #737: a source record's ABBR, DATA (EVEN with DATE and PLAC, AGNC,
// notes) and TEXT.MIME/LANG, and a citation's EVEN (with ROLE and PHRASEs) and
// OBJE, are typed and written back by the entity encoder.

// sourceDataInput is written exactly as the entity writer emits it, so the
// typed path can be held to a byte-identical round trip.
const sourceDataInput = "0 HEAD\n" +
	"1 GEDC\n" +
	"2 VERS 7.0\n" +
	"0 @I1@ INDI\n" +
	"1 SOUR @S1@\n" +
	"2 PAGE Entry 12\n" +
	"2 EVEN BIRT\n" +
	"3 PHRASE Baptism entry\n" +
	"3 ROLE OTHER\n" +
	"4 PHRASE Godmother\n" +
	"2 OBJE @O1@\n" +
	"3 TITL Register page\n" +
	"0 @S1@ SOUR\n" +
	"1 DATA\n" +
	"2 EVEN BIRT, CHR\n" +
	"3 DATE FROM 1820 TO 1825\n" +
	"4 PHRASE The early years\n" +
	"3 PLAC Boston, Massachusetts\n" +
	"2 EVEN DEAT\n" +
	"2 AGNC Parish of St. Mary\n" +
	"2 NOTE Transcribed in 1910.\n" +
	"1 TITL Parish register of St. Mary\n" +
	"1 ABBR St. Mary register\n" +
	"1 TEXT <p>Baptisms</p>\n" +
	"2 MIME text/html\n" +
	"2 LANG en\n" +
	"0 @O1@ OBJE\n" +
	"1 FILE register.jpg\n" +
	"2 FORM image/jpeg\n" +
	"0 TRLR\n"

func TestSourceDataAndCitationEventTyped(t *testing.T) {
	doc, err := Decode(strings.NewReader(sourceDataInput))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	src := doc.GetSource("@S1@")
	if src == nil {
		t.Fatal("source @S1@ missing")
	}
	if src.Abbreviation != "St. Mary register" {
		t.Errorf("Abbreviation = %q, want %q", src.Abbreviation, "St. Mary register")
	}
	if src.Text != "<p>Baptisms</p>" || src.TextMIME != "text/html" || src.TextLanguage != "en" {
		t.Errorf("Text = %q (MIME %q, LANG %q), want %q (text/html, en)",
			src.Text, src.TextMIME, src.TextLanguage, "<p>Baptisms</p>")
	}
	wantData := &gedcom.SourceData{
		Events: []*gedcom.SourceDataEvent{
			{
				Types: "BIRT, CHR", Date: "FROM 1820 TO 1825", DatePhrase: "The early years",
				PlaceDetail: &gedcom.PlaceDetail{Name: "Boston, Massachusetts"},
			},
			{Types: "DEAT"},
		},
		Agency:      "Parish of St. Mary",
		InlineNotes: []string{"Transcribed in 1910."},
	}
	if !reflect.DeepEqual(src.Data, wantData) {
		t.Errorf("Data =\n  %#v\nwant\n  %#v", src.Data, wantData)
	}

	ind := doc.GetIndividual("@I1@")
	if ind == nil || len(ind.SourceCitations) != 1 {
		t.Fatalf("expected @I1@ with one citation, got %+v", ind)
	}
	cite := ind.SourceCitations[0]
	wantEvent := &gedcom.SourceCitationEvent{
		Type: "BIRT", Phrase: "Baptism entry", Role: "OTHER", RolePhrase: "Godmother",
	}
	if !reflect.DeepEqual(cite.Event, wantEvent) {
		t.Errorf("Event = %+v, want %+v", cite.Event, wantEvent)
	}
	if len(cite.Media) != 1 || cite.Media[0].MediaXRef != "@O1@" || cite.Media[0].Title != "Register page" {
		t.Errorf("Media = %+v, want one link to @O1@ titled %q", cite.Media, "Register page")
	}
}

// TestSourceDataByteRoundTrip drops the raw tags so the encoder has to write
// every record from the typed model, then requires the exact input back.
func TestSourceDataByteRoundTrip(t *testing.T) {
	doc, err := Decode(strings.NewReader(sourceDataInput))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	for _, rec := range doc.Records {
		rec.Tags = nil
	}

	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if got := buf.String(); got != sourceDataInput {
		t.Fatalf("typed-path round trip is not byte-identical:\n--- got ---\n%s--- want ---\n%s", got, sourceDataInput)
	}
}

// TestSour1EntityRoundTrip is the acceptance check of #737: both halves of
// gedcom7code's sour-1 come back with the same typed entities after
// decode -> entity-encode -> decode.
func TestSour1EntityRoundTrip(t *testing.T) {
	for _, path := range []string{"testdata/gedcom7code/5/sour-1.ged", "testdata/gedcom7code/7/sour-1.ged"} {
		t.Run(path, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			first, err := Decode(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}

			// The fixture's typed source data and citation event, so the
			// comparison below cannot pass by both sides being empty.
			src := first.GetSource("@2@")
			if src == nil || src.Data == nil || len(src.Data.Events) != 1 ||
				src.Data.Events[0].Types != "BIRT, CHR" || src.Data.Events[0].Date != "FROM 1950 TO 1980" {
				t.Fatalf("@2@ Data = %+v, want one EVEN BIRT, CHR dated FROM 1950 TO 1980", src)
			}
			var cited *gedcom.SourceCitation
			for _, c := range first.GetIndividual("@1@").SourceCitations {
				if c != nil && c.SourceXRef == "@2@" {
					cited = c
				}
			}
			if cited == nil || cited.Event == nil || cited.Event.Type != "BIRT" || cited.Event.Role != "MOTH" {
				t.Fatalf("citation of @2@ = %+v, want EVEN BIRT with ROLE MOTH", cited)
			}

			// Clone keeps first intact, so the comparison is against what
			// was decoded, not against the document being re-encoded.
			second := first.Clone()
			for _, rec := range second.Records {
				rec.Tags = nil
			}
			var buf bytes.Buffer
			if err := Encode(&buf, second); err != nil {
				t.Fatalf("Encode: %v", err)
			}
			back, err := Decode(bytes.NewReader(buf.Bytes()))
			if err != nil {
				t.Fatalf("re-Decode: %v", err)
			}

			for _, rec := range first.Records {
				got := back.XRefMap[rec.XRef]
				if got == nil {
					t.Errorf("record %s missing after round trip", rec.XRef)
					continue
				}
				want, have := entityWithoutTags(rec.Entity), entityWithoutTags(got.Entity)
				if !reflect.DeepEqual(want, have) {
					t.Errorf("record %s entity changed across the round trip:\n got  %#v\n want %#v", rec.XRef, have, want)
				}
			}
		})
	}
}

// entityWithoutTags returns a shallow copy of entity with its raw Tags field
// cleared: the raw half of ADR 0003's dual storage is not what the entity
// path is asked to preserve.
func entityWithoutTags(entity interface{}) interface{} {
	v := reflect.ValueOf(entity)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return entity
	}
	copied := reflect.New(v.Elem().Type())
	copied.Elem().Set(v.Elem())
	if f := copied.Elem().FieldByName("Tags"); f.IsValid() && f.CanSet() {
		f.Set(reflect.Zero(f.Type()))
	}
	return copied.Interface()
}
