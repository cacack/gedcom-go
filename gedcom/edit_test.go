package gedcom

import (
	"errors"
	"reflect"
	"testing"
)

// editDocument returns a document of three individuals, a family linking
// them and a source, with raw Tags as the decoder gives them, plus a header
// naming a submitter.
func editDocument() *Document {
	doc := NewDocument(Version551)
	doc.Header.Tags = []*Tag{
		{Level: 1, Tag: "GEDC"},
		{Level: 2, Tag: "VERS", Value: "5.5.1"},
		{Level: 1, Tag: "SUBM", Value: "@U1@"},
	}
	records := []*Record{
		{XRef: "@I1@", Type: RecordTypeIndividual, Tags: []*Tag{
			{Level: 1, Tag: "NAME", Value: "Ann /Example/"},
			{Level: 1, Tag: "FAMS", Value: "@F1@"},
		}, Entity: &Individual{XRef: "@I1@"}},
		{XRef: "@I2@", Type: RecordTypeIndividual, Tags: []*Tag{
			{Level: 1, Tag: "NAME", Value: "Ben /Example/"},
			{Level: 1, Tag: "BIRT"},
			{Level: 2, Tag: "SOUR", Value: "@S1@"},
			{Level: 3, Tag: "PAGE", Value: "p. 1"},
			{Level: 1, Tag: "FAMC", Value: "@F1@"},
		}, Entity: &Individual{XRef: "@I2@"}},
		{XRef: "@F1@", Type: RecordTypeFamily, Tags: []*Tag{
			{Level: 1, Tag: "HUSB", Value: "@I1@"},
			{Level: 1, Tag: "CHIL", Value: "@I2@"},
			{Level: 1, Tag: "NOTE", Value: "see @I2@"},
		}, Entity: &Family{XRef: "@F1@", Husband: "@I1@", Children: []string{"@I2@"}}},
		{XRef: "@S1@", Type: RecordTypeSource, Tags: []*Tag{
			{Level: 1, Tag: "TITL", Value: "Example register"},
		}, Entity: &Source{XRef: "@S1@"}},
		{XRef: "@U1@", Type: RecordTypeSubmitter},
	}
	for _, r := range records {
		if err := doc.AddRecord(r); err != nil {
			panic(err)
		}
	}
	return doc
}

func recordXRefs(doc *Document) []string {
	xrefs := make([]string, 0, len(doc.Records))
	for _, r := range doc.Records {
		xrefs = append(xrefs, r.XRef)
	}
	return xrefs
}

func TestDocument_ReplaceRecord(t *testing.T) {
	doc := editDocument()
	replacement := &Record{XRef: "@I2@", Type: RecordTypeIndividual, Tags: []*Tag{
		{Level: 1, Tag: "NAME", Value: "Benjamin /Example/"},
	}, Entity: &Individual{XRef: "@I2@", Sex: "M"}}

	if err := doc.ReplaceRecord(replacement); err != nil {
		t.Fatalf("ReplaceRecord: %v", err)
	}
	if got, want := recordXRefs(doc), []string{"@I1@", "@I2@", "@F1@", "@S1@", "@U1@"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Records = %v, want %v", got, want)
	}
	if doc.Records[1] != replacement {
		t.Error("Records[1] is not the replacement")
	}
	if doc.GetRecord("@I2@") != replacement {
		t.Error("GetRecord does not return the replacement")
	}
	if ind := doc.GetIndividual("@I2@"); ind == nil || ind.Sex != "M" {
		t.Errorf("GetIndividual = %+v, want the replacement's entity", ind)
	}
	if len(doc.XRefMap) != 5 {
		t.Errorf("XRefMap has %d entries, want 5", len(doc.XRefMap))
	}
}

func TestDocument_ReplaceRecord_ChangesType(t *testing.T) {
	doc := editDocument()
	note := &Record{XRef: "@S1@", Type: RecordTypeNote, Value: "now a note", Entity: &Note{XRef: "@S1@", Text: "now a note"}}
	if err := doc.ReplaceRecord(note); err != nil {
		t.Fatalf("ReplaceRecord: %v", err)
	}
	if doc.GetSource("@S1@") != nil {
		t.Error("GetSource still finds the replaced source")
	}
	if doc.GetNote("@S1@") == nil {
		t.Error("GetNote does not find the replacement")
	}
}

func TestDocument_ReplaceRecord_UnknownXRef(t *testing.T) {
	for _, xref := range []string{"@I9@", ""} {
		doc := editDocument()
		before := recordXRefs(doc)
		err := doc.ReplaceRecord(&Record{XRef: xref, Type: RecordTypeIndividual})
		if !errors.Is(err, ErrUnknownXRef) {
			t.Errorf("ReplaceRecord(%q) error = %v, want ErrUnknownXRef", xref, err)
		}
		if got := recordXRefs(doc); !reflect.DeepEqual(got, before) {
			t.Errorf("ReplaceRecord(%q) changed Records to %v", xref, got)
		}
	}
}

func TestDocument_ReplaceRecord_NilArguments(t *testing.T) {
	var nilDoc *Document
	if err := nilDoc.ReplaceRecord(&Record{XRef: "@I1@"}); !errors.Is(err, ErrNilDocument) {
		t.Errorf("nil document: error = %v, want ErrNilDocument", err)
	}
	if err := editDocument().ReplaceRecord(nil); !errors.Is(err, ErrNilRecord) {
		t.Errorf("nil record: error = %v, want ErrNilRecord", err)
	}
}

// TestDocument_ReplaceRecord_DuplicateXRef checks that the indexed record,
// the first of two sharing an XRef, is the one replaced.
func TestDocument_ReplaceRecord_DuplicateXRef(t *testing.T) {
	first := &Record{XRef: "@I1@", Type: RecordTypeIndividual}
	second := &Record{XRef: "@I1@", Type: RecordTypeIndividual}
	doc := &Document{Records: []*Record{first, second}}
	replacement := &Record{XRef: "@I1@", Type: RecordTypeIndividual}
	if err := doc.ReplaceRecord(replacement); err != nil {
		t.Fatalf("ReplaceRecord: %v", err)
	}
	if doc.Records[0] != replacement || doc.Records[1] != second {
		t.Error("the first record was not the one replaced")
	}
}

func TestDocument_RemoveRecord(t *testing.T) {
	doc := editDocument()
	refs, err := doc.RemoveRecord("@I2@")
	if err != nil {
		t.Fatalf("RemoveRecord: %v", err)
	}
	if got, want := recordXRefs(doc), []string{"@I1@", "@F1@", "@S1@", "@U1@"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Records = %v, want %v", got, want)
	}
	if doc.GetRecord("@I2@") != nil {
		t.Error("GetRecord still finds the removed record")
	}
	if len(doc.XRefMap) != 4 {
		t.Errorf("XRefMap has %d entries, want 4", len(doc.XRefMap))
	}
	// The NOTE text naming @I2@ is not a pointer; the removed record's own
	// pointers are gone with it.
	want := []Reference{{XRef: "@F1@", Path: []string{"FAM", "CHIL"}}}
	if !reflect.DeepEqual(refs, want) {
		t.Errorf("references = %+v, want %+v", refs, want)
	}
	// Pointers are reported, not changed.
	if v := doc.Records[1].Tags[1].Value; v != "@I2@" {
		t.Errorf("CHIL value = %q, want it unchanged", v)
	}
}

func TestDocument_RemoveRecord_ReferencePaths(t *testing.T) {
	tests := []struct {
		xref string
		want []Reference
	}{
		{"@S1@", []Reference{{XRef: "@I2@", Path: []string{"INDI", "BIRT", "SOUR"}}}},
		{"@F1@", []Reference{
			{XRef: "@I1@", Path: []string{"INDI", "FAMS"}},
			{XRef: "@I2@", Path: []string{"INDI", "FAMC"}},
		}},
		{"@U1@", []Reference{{Path: []string{"HEAD", "SUBM"}}}},
		{"@I1@", []Reference{{XRef: "@F1@", Path: []string{"FAM", "HUSB"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.xref, func(t *testing.T) {
			refs, err := editDocument().RemoveRecord(tt.xref)
			if err != nil {
				t.Fatalf("RemoveRecord: %v", err)
			}
			if !reflect.DeepEqual(refs, tt.want) {
				t.Errorf("references = %+v, want %+v", refs, tt.want)
			}
		})
	}
}

// TestDocument_RemoveRecord_EntityOnly checks that pointers held only in the
// typed entities of records built in code, and in the typed header, are
// reported.
func TestDocument_RemoveRecord_EntityOnly(t *testing.T) {
	doc := NewDocument(Version70)
	doc.Header.Submitter = "@U1@"
	for _, r := range []*Record{
		{XRef: "@I1@", Type: RecordTypeIndividual, Entity: &Individual{XRef: "@I1@"}},
		{XRef: "@F1@", Type: RecordTypeFamily, Entity: &Family{XRef: "@F1@", Husband: "@I1@", Children: []string{"@I1@"}}},
		{XRef: "@U1@", Type: RecordTypeSubmitter, Entity: &Submitter{XRef: "@U1@"}},
	} {
		if err := doc.AddRecord(r); err != nil {
			t.Fatal(err)
		}
	}

	refs, err := doc.RemoveRecord("@I1@")
	if err != nil {
		t.Fatalf("RemoveRecord: %v", err)
	}
	want := []Reference{{XRef: "@F1@", Path: []string{"FAM"}}, {XRef: "@F1@", Path: []string{"FAM"}}}
	if !reflect.DeepEqual(refs, want) {
		t.Errorf("references = %+v, want %+v", refs, want)
	}

	refs, err = doc.RemoveRecord("@U1@")
	if err != nil {
		t.Fatalf("RemoveRecord: %v", err)
	}
	if want := []Reference{{Path: []string{"HEAD", "SUBM"}}}; !reflect.DeepEqual(refs, want) {
		t.Errorf("header references = %+v, want %+v", refs, want)
	}
}

func TestDocument_RemoveRecord_UnknownXRef(t *testing.T) {
	doc := editDocument()
	for _, xref := range []string{"@I9@", ""} {
		refs, err := doc.RemoveRecord(xref)
		if !errors.Is(err, ErrUnknownXRef) {
			t.Errorf("RemoveRecord(%q) error = %v, want ErrUnknownXRef", xref, err)
		}
		if refs != nil {
			t.Errorf("RemoveRecord(%q) references = %v, want nil", xref, refs)
		}
	}
	if len(doc.Records) != 5 {
		t.Errorf("Records has %d entries, want 5", len(doc.Records))
	}
	var nilDoc *Document
	if _, err := nilDoc.RemoveRecord("@I1@"); !errors.Is(err, ErrNilDocument) {
		t.Errorf("nil document: error = %v, want ErrNilDocument", err)
	}
}

// TestDocument_RemoveRecord_DuplicateXRef checks that removing the indexed
// record of two sharing an XRef indexes the other in its place.
func TestDocument_RemoveRecord_DuplicateXRef(t *testing.T) {
	first := &Record{XRef: "@I1@", Type: RecordTypeIndividual}
	second := &Record{XRef: "@I1@", Type: RecordTypeIndividual}
	doc := &Document{Records: []*Record{first, second}}
	if _, err := doc.RemoveRecord("@I1@"); err != nil {
		t.Fatalf("RemoveRecord: %v", err)
	}
	if len(doc.Records) != 1 || doc.Records[0] != second {
		t.Fatalf("Records = %v, want only the second record", doc.Records)
	}
	if doc.GetRecord("@I1@") != second {
		t.Error("the remaining record is not indexed")
	}
}

// TestDocument_RemoveRecord_NullPointer checks that a GEDCOM 7.0 @VOID@
// pointer is not reported as naming a removed record called @VOID@.
func TestDocument_RemoveRecord_NullPointer(t *testing.T) {
	doc := NewDocument(Version70)
	for _, r := range []*Record{
		{XRef: "@VOID@", Type: RecordTypeIndividual},
		{XRef: "@F1@", Type: RecordTypeFamily, Tags: []*Tag{{Level: 1, Tag: "HUSB", Value: "@VOID@"}}},
	} {
		if err := doc.AddRecord(r); err != nil {
			t.Fatal(err)
		}
	}
	refs, err := doc.RemoveRecord("@VOID@")
	if err != nil {
		t.Fatalf("RemoveRecord: %v", err)
	}
	if refs != nil {
		t.Errorf("references = %+v, want none", refs)
	}
}

// TestDocument_RemoveRecord_SharedBacking checks that removal does not write
// through to a slice sharing Records' backing array.
func TestDocument_RemoveRecord_SharedBacking(t *testing.T) {
	doc := editDocument()
	alias := doc.Records
	if _, err := doc.RemoveRecord("@I1@"); err != nil {
		t.Fatal(err)
	}
	if alias[0].XRef != "@I1@" || alias[1].XRef != "@I2@" {
		t.Errorf("alias changed to %v", []string{alias[0].XRef, alias[1].XRef})
	}
}
