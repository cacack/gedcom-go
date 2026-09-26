package gedcomgo

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// Issue #497: SOURCE_CITATION.SOUR.DATA.TEXT is {0:M}. Every TEXT must reach
// SourceCitationData.Text, in document order, with its CONT/CONC continuation
// folded in (#442) and its GEDCOM 7.0 MIME/LANG typed; and the entity writer
// must emit them all back, in the same order.

// citationTextInput is written exactly as the entity writer emits it, so the
// typed path can be held to a byte-identical round trip.
const citationTextInput = "0 HEAD\n" +
	"1 GEDC\n" +
	"2 VERS 7.0\n" +
	"0 @I1@ INDI\n" +
	"1 BIRT\n" +
	"2 SOUR @S1@\n" +
	"3 DATA\n" +
	"4 DATE 6 AUG 1901\n" +
	"4 TEXT First passage.\n" +
	"5 CONT Its second line.\n" +
	"4 TEXT <p>Second passage.</p>\n" +
	"5 MIME text/html\n" +
	"5 LANG en\n" +
	"4 TEXT Dritte Stelle.\n" +
	"5 LANG de\n" +
	"0 @S1@ SOUR\n" +
	"1 TITL Parish register\n" +
	"0 TRLR\n"

func birthCitationData(t *testing.T, doc *gedcom.Document) *gedcom.SourceCitationData {
	t.Helper()
	ind := doc.GetIndividual("@I1@")
	if ind == nil || len(ind.Events) != 1 || len(ind.Events[0].SourceCitations) != 1 {
		t.Fatalf("expected @I1@ with one BIRT carrying one citation, got %+v", ind)
	}
	data := ind.Events[0].SourceCitations[0].Data
	if data == nil {
		t.Fatal("citation Data is nil")
	}
	return data
}

func TestSourceCitationDataTextRepeats(t *testing.T) {
	doc, err := Decode(strings.NewReader(citationTextInput))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	data := birthCitationData(t, doc)

	want := []*gedcom.SourceText{
		{Value: "First passage.\nIts second line."},
		{Value: "<p>Second passage.</p>", MIME: "text/html", Language: "en"},
		{Value: "Dritte Stelle.", Language: "de"},
	}
	if !reflect.DeepEqual(data.Text, want) {
		t.Fatalf("Data.Text mismatch:\n got  %s\n want %s", dumpSourceTexts(data.Text), dumpSourceTexts(want))
	}
	if data.Date != "6 AUG 1901" {
		t.Errorf("Data.Date = %q, want %q", data.Date, "6 AUG 1901")
	}
}

// TestSourceCitationDataTextByteRoundTrip drops the raw tags so the encoder has
// to write the citation from the typed model, then requires the exact input
// back and the same typed value on re-decode.
func TestSourceCitationDataTextByteRoundTrip(t *testing.T) {
	doc, err := Decode(strings.NewReader(citationTextInput))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	first := birthCitationData(t, doc)
	for _, rec := range doc.Records {
		rec.Tags = nil
	}

	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if got := buf.String(); got != citationTextInput {
		t.Fatalf("typed-path round trip is not byte-identical:\n--- got ---\n%s--- want ---\n%s", got, citationTextInput)
	}

	back, err := Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("re-Decode: %v", err)
	}
	if second := birthCitationData(t, back); !reflect.DeepEqual(first, second) {
		t.Errorf("typed value changed across decode -> encode -> decode:\n got  %s\n want %s",
			dumpSourceTexts(second.Text), dumpSourceTexts(first.Text))
	}
}

// TestSourceCitationDataTextCONC covers the CONC half of the fold and the
// GEDCOM 5.5.1 path, where TEXT has no MIME/LANG.
func TestSourceCitationDataTextCONC(t *testing.T) {
	input := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
		"0 @I1@ INDI\n1 BIRT\n2 SOUR @S1@\n3 DATA\n" +
		"4 TEXT Split mid-\n5 CONC word here.\n5 CONT Next line.\n" +
		"4 TEXT Second.\n" +
		"0 @S1@ SOUR\n0 TRLR\n"
	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	data := birthCitationData(t, doc)
	want := []*gedcom.SourceText{
		{Value: "Split mid-word here.\nNext line."},
		{Value: "Second."},
	}
	if !reflect.DeepEqual(data.Text, want) {
		t.Fatalf("Data.Text mismatch:\n got  %s\n want %s", dumpSourceTexts(data.Text), dumpSourceTexts(want))
	}
}

func dumpSourceTexts(texts []*gedcom.SourceText) string {
	parts := make([]string, len(texts))
	for i, st := range texts {
		if st == nil {
			parts[i] = "nil"
			continue
		}
		parts[i] = "{Value:" + st.Value + " MIME:" + st.MIME + " Language:" + st.Language + "}"
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
