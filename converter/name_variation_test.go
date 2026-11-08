package converter_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/cacack/gedcom-go/v3/converter"
	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// nameVariationInput is a 5.5.1 file with FONE and ROMN under INDI.NAME
// (with TYPE and name pieces beneath them) and under a PLAC.
const nameVariationInput = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
	"0 @I1@ INDI\n" +
	"1 NAME 山田 /太郎/\n" +
	"2 GIVN 太郎\n" +
	"2 FONE やまだ /たろう/\n3 TYPE kana\n3 SURN やまだ\n" +
	"2 ROMN Yamada /Taro/\n3 TYPE romaji\n3 SURN Yamada\n" +
	"1 BIRT\n2 PLAC 東京\n3 ROMN Tokyo\n4 TYPE romaji\n" +
	"0 @I2@ INDI\n1 NAME John /Doe/\n" +
	"0 TRLR\n"

// convertAndEncode converts doc to target and encodes the result with
// default options, so a decoded record is written from its Record.Tags.
func convertAndEncode(t *testing.T, doc *gedcom.Document, target gedcom.Version) (*gedcom.Document, *gedcom.ConversionReport, string) {
	t.Helper()
	converted, report, err := converter.Convert(doc, target)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	var buf bytes.Buffer
	if err := encoder.Encode(&buf, converted); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return converted, report, buf.String()
}

// TestConvertDropsNameVariations pins issue #694: a 5.5.1 -> 7.0 or
// 5.5.1 -> 5.5 conversion removes FONE and ROMN, which neither target
// defines, from Record.Tags and the typed names, and reports each as data
// loss, while the rest of the name is kept.
func TestConvertDropsNameVariations(t *testing.T) {
	for _, target := range []gedcom.Version{gedcom.Version70, gedcom.Version55} {
		t.Run("decoded/"+target.String(), func(t *testing.T) {
			doc, err := decoder.Decode(bytes.NewReader([]byte(nameVariationInput)))
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if name := doc.GetIndividual("@I1@").Names[0]; len(name.Phonetic) != 1 || len(name.Romanized) != 1 {
				t.Fatalf("decoded name variations = %d FONE, %d ROMN, want 1 each", len(name.Phonetic), len(name.Romanized))
			}

			converted, report, out := convertAndEncode(t, doc, target)
			written := recordLineTags(out)
			for _, tag := range []string{"FONE", "ROMN"} {
				if written[tag] {
					t.Errorf("output writes %s:\n%s", tag, out)
				}
				if !hasDataLoss(report, tag+" tags", "@I1@") {
					t.Errorf("%s tags not reported for @I1@", tag)
				}
				if hasDataLoss(report, tag+" tags", "@I2@") {
					t.Errorf("%s tags reported for @I2@, which has none", tag)
				}
			}
			if !hasDropped(report, "FONE") || !hasDropped(report, "ROMN") {
				t.Errorf("Dropped notes = %+v, want FONE and ROMN", report.Dropped)
			}
			for _, tag := range []string{"NAME", "GIVN", "PLAC"} {
				if !written[tag] {
					t.Errorf("output lost %s:\n%s", tag, out)
				}
			}
			name := converted.GetIndividual("@I1@").Names[0]
			if len(name.Phonetic) != 0 || len(name.Romanized) != 0 {
				t.Errorf("converted name keeps Phonetic %v, Romanized %v; want both empty", name.Phonetic, name.Romanized)
			}
			if src := doc.GetIndividual("@I1@").Names[0]; len(src.Phonetic) != 1 || len(src.Romanized) != 1 {
				t.Error("Convert changed the source document's entity")
			}
		})
	}

	for _, target := range []gedcom.Version{gedcom.Version70, gedcom.Version55} {
		t.Run("entity-only/"+target.String(), func(t *testing.T) {
			name := &gedcom.PersonalName{
				Full:      "Yamada /Taro/",
				Phonetic:  []*gedcom.NameVariation{{Value: "やまだ /たろう/", Type: "kana"}},
				Romanized: []*gedcom.NameVariation{{Value: "Yamada /Taro/", Type: "romaji"}},
			}
			indi := &gedcom.Individual{XRef: "@I1@", Names: []*gedcom.PersonalName{name}}
			record := &gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: indi}
			doc := &gedcom.Document{
				Header:  &gedcom.Header{Version: gedcom.Version551},
				Records: []*gedcom.Record{record},
				XRefMap: map[string]*gedcom.Record{"@I1@": record},
			}

			converted, report, out := convertAndEncode(t, doc, target)
			written := recordLineTags(out)
			for _, tag := range []string{"FONE", "ROMN"} {
				if written[tag] {
					t.Errorf("output writes %s:\n%s", tag, out)
				}
				if !hasDataLoss(report, tag+" tags", "@I1@") {
					t.Errorf("%s tags not reported for @I1@", tag)
				}
			}
			got := converted.GetIndividual("@I1@").Names[0]
			if len(got.Phonetic) != 0 || len(got.Romanized) != 0 {
				t.Errorf("converted name keeps Phonetic %v, Romanized %v; want both empty", got.Phonetic, got.Romanized)
			}
			if got.Full != "Yamada /Taro/" {
				t.Errorf("converted name Full = %q, want it kept", got.Full)
			}
			if len(name.Phonetic) != 1 || len(name.Romanized) != 1 {
				t.Error("Convert changed the source document's entity")
			}
		})
	}
}

// TestConvertKeepsNameVariationsFor551 checks that a conversion to 5.5.1,
// which defines FONE and ROMN, keeps them.
func TestConvertKeepsNameVariationsFor551(t *testing.T) {
	input := "0 HEAD\n1 GEDC\n2 VERS 5.5\n0 @I1@ INDI\n1 NAME Yamada /Taro/\n2 ROMN Yamada Taro\n3 TYPE romaji\n0 TRLR\n"
	doc, err := decoder.Decode(bytes.NewReader([]byte(input)))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	_, report, out := convertAndEncode(t, doc, gedcom.Version551)
	if !recordLineTags(out)["ROMN"] {
		t.Errorf("5.5 -> 5.5.1 output lost ROMN:\n%s", out)
	}
	if hasDataLoss(report, "ROMN tags", "@I1@") {
		t.Error("ROMN reported as data loss on a conversion to 5.5.1")
	}
}

// TestConvertDropsNameVariationsFromOtherSources checks the paths whose
// source does not define FONE/ROMN either, for a nonconforming file that
// carries them: 5.5 -> 7.0 and 7.0 -> 5.5 drop and report them too.
func TestConvertDropsNameVariationsFromOtherSources(t *testing.T) {
	tests := []struct {
		vers   string
		target gedcom.Version
	}{
		{"5.5", gedcom.Version70},
		{"7.0", gedcom.Version55},
	}
	for _, tt := range tests {
		t.Run(tt.vers+"->"+tt.target.String(), func(t *testing.T) {
			input := "0 HEAD\n1 GEDC\n2 VERS " + tt.vers + "\n0 @I1@ INDI\n1 NAME Yamada /Taro/\n2 ROMN Yamada Taro\n3 TYPE romaji\n0 TRLR\n"
			doc, err := decoder.Decode(bytes.NewReader([]byte(input)))
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			_, report, out := convertAndEncode(t, doc, tt.target)
			if recordLineTags(out)["ROMN"] {
				t.Errorf("output writes ROMN:\n%s", out)
			}
			if !hasDataLoss(report, "ROMN tags", "@I1@") {
				t.Error("ROMN tags not reported for @I1@")
			}
		})
	}

	t.Run("entity-only/7.0->5.5", func(t *testing.T) {
		name := &gedcom.PersonalName{Full: "Yamada /Taro/", Romanized: []*gedcom.NameVariation{{Value: "Yamada Taro"}}}
		indi := &gedcom.Individual{XRef: "@I1@", Names: []*gedcom.PersonalName{name}}
		record := &gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: indi}
		doc := &gedcom.Document{
			Header:  &gedcom.Header{Version: gedcom.Version70},
			Records: []*gedcom.Record{record},
			XRefMap: map[string]*gedcom.Record{"@I1@": record},
		}
		converted, report, _ := convertAndEncode(t, doc, gedcom.Version55)
		if got := converted.GetIndividual("@I1@").Names[0].Romanized; len(got) != 0 {
			t.Errorf("converted name keeps Romanized %v; want empty", got)
		}
		if !hasDataLoss(report, "ROMN tags", "@I1@") {
			t.Error("ROMN tags not reported for @I1@")
		}
	})
}

// TestConvertNameVariationReportOrder checks that on a conversion to 7.0 no
// transform reports a change inside a FONE/ROMN block that is then removed,
// and that the drop names the record by its normalized XRef.
func TestConvertNameVariationReportOrder(t *testing.T) {
	input := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
		"0 @N1@ NOTE shared\n" +
		"0 @i-1@ INDI\n1 NAME Yamada /Taro/\n" +
		"2 ROMN Yamada /Taro/\n3 TYPE romaji\n3 NOTE @N1@\n3 NOTE some long\n4 CONC  text\n" +
		"0 TRLR\n"
	doc, err := decoder.Decode(bytes.NewReader([]byte(input)))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	_, report, out := convertAndEncode(t, doc, gedcom.Version70)
	if recordLineTags(out)["ROMN"] {
		t.Errorf("output writes ROMN:\n%s", out)
	}
	for _, typ := range []string{"CONC_REMOVED", "NOTE_POINTER_TO_SNOTE"} {
		if got := transformationCount(report, typ); got != 0 {
			t.Errorf("%s count = %d, want 0 (the only one is inside the dropped ROMN)", typ, got)
		}
	}
	for _, note := range report.Normalized {
		if note.Original == "some long" {
			t.Errorf("Normalized note %+v describes text inside the dropped ROMN", note)
		}
	}
	if !hasDataLoss(report, "ROMN tags", "@I_1@") {
		t.Errorf("ROMN tags not reported under the normalized XRef @I_1@; DataLoss = %+v", report.DataLoss)
	}
}

// TestConvertDropsCorpusNameVariations runs the conversion on the corpus file
// that carries ROMN and FONE under a second NAME.
func TestConvertDropsCorpusNameVariations(t *testing.T) {
	input, err := os.ReadFile("../testdata/edge-cases/structural-edge-cases.ged")
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []gedcom.Version{gedcom.Version70, gedcom.Version55} {
		t.Run(target.String(), func(t *testing.T) {
			doc, err := decoder.Decode(bytes.NewReader(input))
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			_, report, out := convertAndEncode(t, doc, target)
			written := recordLineTags(out)
			for _, tag := range []string{"FONE", "ROMN"} {
				if written[tag] {
					t.Errorf("output writes %s", tag)
				}
				if !hasDataLoss(report, tag+" tags", "@I8@") {
					t.Errorf("%s tags not reported for @I8@", tag)
				}
			}
		})
	}
}

func hasDropped(report *gedcom.ConversionReport, original string) bool {
	for _, note := range report.Dropped {
		if note.Original == original {
			return true
		}
	}
	return false
}
