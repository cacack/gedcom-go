package converter_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/converter"
	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// entityDiffs appends to diffs the path of every exported field where a and b
// differ, skipping Tags fields: those hold raw lines whose line numbers change
// on a re-decode, and the comparison is of the typed view.
func entityDiffs(a, b reflect.Value, path string, diffs *[]string) {
	if !a.IsValid() || !b.IsValid() { // a nil Entity: a record type with none
		if a.IsValid() != b.IsValid() {
			*diffs = append(*diffs, path)
		}
		return
	}
	if a.Kind() == reflect.Pointer || a.Kind() == reflect.Interface {
		if a.IsNil() || b.IsNil() {
			if a.IsNil() != b.IsNil() {
				*diffs = append(*diffs, path)
			}
			return
		}
		a, b = a.Elem(), b.Elem()
	}
	if a.Kind() == reflect.Struct {
		for i := 0; i < a.NumField(); i++ {
			f := a.Type().Field(i)
			if f.IsExported() && f.Name != "Tags" {
				entityDiffs(a.Field(i), b.Field(i), path+"."+f.Name, diffs)
			}
		}
		return
	}
	if a.Kind() == reflect.Slice && a.Len() == b.Len() && a.Type().Elem().Kind() == reflect.Pointer {
		for i := 0; i < a.Len(); i++ {
			entityDiffs(a.Index(i), b.Index(i), path, diffs)
		}
		return
	}
	if !reflect.DeepEqual(a.Interface(), b.Interface()) {
		*diffs = append(*diffs, path)
	}
}

// TestConvertedEntitiesAgreeWithTags asserts that, for every fixture and every
// target version, each converted record's typed Entity is what decoding its
// converted Tags gives (#772): the output is encoded from its Tags and decoded
// again, and the two typed views compared field by field.
func TestConvertedEntitiesAgreeWithTags(t *testing.T) {
	files, err := filepath.Glob("../testdata/*/*.ged")
	if err != nil || len(files) == 0 {
		t.Fatalf("no fixtures: %v", err)
	}
	targets := []gedcom.Version{gedcom.Version55, gedcom.Version551, gedcom.Version70}
	for _, path := range files {
		if filepath.Base(path) == "longsword.ged" {
			continue // the scale fixture, as other corpus walks skip it
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, target := range targets {
			doc, err := decoder.Decode(bytes.NewReader(data))
			if err != nil || doc.Header == nil {
				continue // a malformed fixture, not a conversion input
			}
			if doc.Header.Version == target && target != gedcom.Version70 {
				continue // returned as is; only 7.0 -> 7.0 rewrites a copy
			}
			name := filepath.Base(path) + "/" + string(doc.Header.Version) + "->" + string(target)
			conv, _, err := converter.Convert(doc, target)
			if err != nil {
				continue // an unsupported or strict-mode refusal, not a divergence
			}
			var buf bytes.Buffer
			if err := encoder.Encode(&buf, conv); err != nil {
				t.Errorf("%s: Encode: %v", name, err)
				continue
			}
			again, err := decoder.Decode(&buf)
			if err != nil {
				t.Errorf("%s: re-Decode: %v", name, err)
				continue
			}
			if len(again.Records) != len(conv.Records) {
				t.Errorf("%s: %d records re-decoded, want %d", name, len(again.Records), len(conv.Records))
				continue
			}
			for i, r := range conv.Records {
				var diffs []string
				entityDiffs(reflect.ValueOf(r.Entity), reflect.ValueOf(again.Records[i].Entity), string(r.Type), &diffs)
				if len(diffs) > 0 {
					t.Errorf("%s: %s %s: Entity disagrees with Tags at %s", name, r.Type, r.XRef, strings.Join(diffs, ", "))
				}
			}
		}
	}
}

// TestConvertNoteRecordKeepsChangeDateAndSources pins the case the issue was
// filed on: a 5.5.1 NOTE record converted to a 7.0 SNOTE kept CHAN and SOUR in
// its Tags but lost them from the SharedNote entity.
func TestConvertNoteRecordKeepsChangeDateAndSources(t *testing.T) {
	const in = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
		"0 @S1@ SOUR\n1 TITL A source\n" +
		"0 @N1@ NOTE Some text\n1 SOUR @S1@\n2 PAGE p. 4\n1 CHAN\n2 DATE 1 JAN 2020\n" +
		"0 TRLR\n"
	doc, err := decoder.Decode(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	conv, _, err := converter.Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatal(err)
	}
	rec := conv.XRefMap["@N1@"]
	if rec == nil {
		for _, r := range conv.Records {
			if r.XRef == "@N1@" {
				rec = r
			}
		}
	}
	if rec == nil {
		t.Fatal("note record missing")
	}
	note, ok := rec.Entity.(*gedcom.SharedNote)
	if !ok {
		t.Fatalf("Entity = %T, want *gedcom.SharedNote", rec.Entity)
	}
	if note.Text != "Some text" {
		t.Errorf("Text = %q, want %q", note.Text, "Some text")
	}
	if note.ChangeDate == nil || note.ChangeDate.Date != "1 JAN 2020" {
		t.Errorf("ChangeDate = %+v, want date 1 JAN 2020", note.ChangeDate)
	}
	if len(note.SourceCitations) != 1 || note.SourceCitations[0].SourceXRef != "@S1@" || note.SourceCitations[0].Page != "p. 4" {
		t.Errorf("SourceCitations = %+v, want one citing @S1@ p. 4", note.SourceCitations)
	}
}

// TestConvertKeepsTaglessEntity asserts that a hand-built record with no Tags
// keeps the Entity the conversion left it: there are no Tags to derive one from.
func TestConvertKeepsTaglessEntity(t *testing.T) {
	indi := &gedcom.Individual{XRef: "@I1@", Sex: "F"}
	doc := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version551},
		Records: []*gedcom.Record{{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: indi}},
	}
	conv, _, err := converter.Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := conv.Records[0].Entity.(*gedcom.Individual)
	if !ok || got.Sex != "F" || got.XRef != "@I1@" {
		t.Errorf("Entity = %+v, want the hand-built individual", conv.Records[0].Entity)
	}
}
