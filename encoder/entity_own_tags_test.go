package encoder

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Issues #707 and #711: a record written from its entity (Record.Tags empty)
// takes a note's text from the entity whatever Record.Value holds, and carries
// the entity's own raw Tags that the typed fields did not produce.

func decodeGEDCOM(t *testing.T, text string) *gedcom.Document {
	t.Helper()
	doc, err := decoder.Decode(strings.NewReader(text))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	return doc
}

func encodeToString(t *testing.T, doc *gedcom.Document) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return buf.String()
}

// clearRecordTags makes every record write from its entity, the documented
// way to make typed edits reach the output.
func clearRecordTags(doc *gedcom.Document) {
	for _, rec := range doc.Records {
		rec.Tags = nil
	}
}

// TestNoteRecordDecodedThenClearedKeepsAllLines pins the decoded case: the
// record's Value holds the first line, and before #707 that alone was
// written. #330 covered only the hand-built case with an empty Value.
func TestNoteRecordDecodedThenClearedKeepsAllLines(t *testing.T) {
	const in = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
		"0 @N1@ NOTE first line\n1 CONT second line\n1 CONC  continued\n1 CONT third line\n" +
		"0 TRLR\n"
	doc := decodeGEDCOM(t, in)
	note, _ := doc.Records[0].GetNote()
	if note.Text != "first line\nsecond line continued\nthird line" {
		t.Fatalf("decoded Note.Text = %q", note.Text)
	}
	if doc.Records[0].Value != "first line" {
		t.Fatalf("decoded Record.Value = %q, want first line (the case this test pins)", doc.Records[0].Value)
	}

	clearRecordTags(doc)
	out := encodeToString(t, doc)
	if !strings.Contains(out, "0 @N1@ NOTE first line\n1 CONT second line continued\n1 CONT third line\n") {
		t.Errorf("multi-line NOTE lost its continuation lines:\n%s", out)
	}

	redoc := decodeGEDCOM(t, out)
	renote, _ := redoc.Records[0].GetNote()
	if renote.Text != note.Text {
		t.Errorf("Note.Text after decode -> clear Tags -> encode -> decode = %q, want %q", renote.Text, note.Text)
	}
}

func TestSharedNoteRecordDecodedThenClearedKeepsAllLines(t *testing.T) {
	const in = "0 HEAD\n1 GEDC\n2 VERS 7.0\n" +
		"0 @N1@ SNOTE first line\n1 CONT second line\n1 MIME text/plain\n1 LANG en\n" +
		"0 TRLR\n"
	doc := decodeGEDCOM(t, in)
	clearRecordTags(doc)
	out := encodeToString(t, doc)
	if !strings.Contains(out, "0 @N1@ SNOTE first line\n1 CONT second line\n") {
		t.Errorf("multi-line SNOTE lost its continuation:\n%s", out)
	}
	if strings.Count(out, "1 MIME text/plain") != 1 || strings.Count(out, "1 LANG en") != 1 {
		t.Errorf("SNOTE substructures written other than once (#711 SharedNote.Tags pass-through):\n%s", out)
	}
	redoc := decodeGEDCOM(t, out)
	snote, _ := redoc.Records[0].GetSharedNote()
	if snote.Text != "first line\nsecond line" {
		t.Errorf("SharedNote.Text after round trip = %q", snote.Text)
	}
}

// TestNoteTextSurvivesClearedTagsInCorpus runs decode -> clear Tags -> encode
// -> decode over the fixtures named in #707 and compares every NOTE and
// SNOTE record's text.
func TestNoteTextSurvivesClearedTagsInCorpus(t *testing.T) {
	files := []string{
		"../testdata/gedcom-5.5.1/comprehensive.ged",
		"../testdata/gedcom-5.5/torture-test/TGC55C.ged",
		"../testdata/gedcom-7.0/maximal70.ged",
	}
	for _, path := range files {
		t.Run(path, func(t *testing.T) {
			f, err := os.Open(path) // #nosec G304 -- path is a constant in the repo's own testdata
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			doc, err := decoder.Decode(f)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			want := noteTextsByXRef(doc)
			if len(want) == 0 {
				t.Fatalf("%s has no NOTE/SNOTE records; the fixture no longer exercises this", path)
			}

			clearRecordTags(doc)
			redoc := decodeGEDCOM(t, encodeToString(t, doc))
			got := noteTextsByXRef(redoc)
			for xref, text := range want {
				if got[xref] != text {
					t.Errorf("%s: text after clear-Tags round trip = %q, want %q", xref, got[xref], text)
				}
			}
		})
	}
}

func noteTextsByXRef(doc *gedcom.Document) map[string]string {
	out := map[string]string{}
	for _, rec := range doc.Records {
		if n, ok := rec.GetNote(); ok {
			out[rec.XRef] = n.Text
		}
		if n, ok := rec.GetSharedNote(); ok {
			out[rec.XRef] = n.Text
		}
	}
	return out
}

// TestHandBuiltRecordCarriesOwnTags is the example from #711.
func TestHandBuiltRecordCarriesOwnTags(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version551},
		Records: []*gedcom.Record{{
			Type: gedcom.RecordTypeIndividual, XRef: "@I1@",
			Entity: &gedcom.Individual{
				XRef:  "@I1@",
				Names: []*gedcom.PersonalName{{Full: "A /B/"}},
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "_MYAPP_ID", Value: "abc123"},
					nil, // a nil element is skipped, not written and not a panic (ADR 0007)
					{Level: 2, Tag: "_SOURCE", Value: "sync"},
				},
			},
		}},
	}
	out := encodeToString(t, doc)
	if !strings.Contains(out, "1 NAME A /B/\n") || !strings.Contains(out, "1 _MYAPP_ID abc123\n2 _SOURCE sync\n") {
		t.Fatalf("hand-built record missing NAME or its own tag block:\n%s", out)
	}

	redoc := decodeGEDCOM(t, out)
	indi := redoc.GetIndividual("@I1@")
	if indi == nil || len(indi.Names) != 1 || indi.Names[0].Full != "A /B/" {
		t.Fatalf("re-decoded individual lost its name: %+v", indi)
	}
	found := false
	for _, tag := range indi.Tags {
		if tag.Level == 1 && tag.Tag == "_MYAPP_ID" && tag.Value == "abc123" {
			found = true
		}
	}
	if !found {
		t.Errorf("re-decoded individual has no _MYAPP_ID in Tags: %+v", indi.Tags)
	}
}

// TestOwnTagsAppliesToEveryEntityType checks the rule for each record type
// that has a Tags field.
func TestOwnTagsAppliesToEveryEntityType(t *testing.T) {
	own := []*gedcom.Tag{{Level: 1, Tag: "_X", Value: "1"}}
	entities := map[gedcom.RecordType]any{
		gedcom.RecordTypeIndividual: &gedcom.Individual{Tags: own},
		gedcom.RecordTypeFamily:     &gedcom.Family{Tags: own},
		gedcom.RecordTypeSource:     &gedcom.Source{Tags: own},
		gedcom.RecordTypeSubmitter:  &gedcom.Submitter{Tags: own},
		gedcom.RecordTypeSubmission: &gedcom.Submission{Tags: own},
		gedcom.RecordTypeRepository: &gedcom.Repository{Tags: own},
		gedcom.RecordTypeNote:       &gedcom.Note{Text: "n", Tags: own},
		gedcom.RecordTypeMedia:      &gedcom.MediaObject{Tags: own},
		gedcom.RecordTypeSharedNote: &gedcom.SharedNote{Text: "s", Tags: own},
	}
	for typ, entity := range entities {
		doc := &gedcom.Document{Records: []*gedcom.Record{{Type: typ, XRef: "@R1@", Entity: entity}}}
		if out := encodeToString(t, doc); !strings.Contains(out, "1 _X 1\n") {
			t.Errorf("%s: own tag not written:\n%s", typ, out)
		}
	}
}

// TestOwnTagsExtensionOnly checks the filter: only _-prefixed level-1 tags
// are taken from an entity's Tags, with their blocks; standard tags the
// typed model has no field for are dropped on this path, as before #711,
// and CONT/CONC are never written from Tags.
func TestOwnTagsExtensionOnly(t *testing.T) {
	const in = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
		"0 @I1@ INDI\n1 NAME A /B/\n2 GIVN A\n1 SEX M\n1 AFN 123-ABC\n1 _MYAPP x\n2 _SUB y\n1 NOTE line one\n2 CONT line two\n" +
		"0 TRLR\n"
	doc := decodeGEDCOM(t, in)
	clearRecordTags(doc)
	out := encodeToString(t, doc)

	for _, want := range []string{"1 NAME A /B/\n", "1 SEX M\n", "1 _MYAPP x\n2 _SUB y\n", "1 NOTE line one\n2 CONT line two\n"} {
		if strings.Count(out, want) != 1 {
			t.Errorf("%q written %d times, want once:\n%s", want, strings.Count(out, want), out)
		}
	}
	if strings.Contains(out, "AFN") {
		t.Errorf("standard tag AFN written from entity Tags:\n%s", out)
	}

	// A hand-built NOTE whose Tags carry stale CONT lines: the text wins, and
	// the standard REFN is not taken from Tags.
	note := &gedcom.Document{Records: []*gedcom.Record{{
		Type: gedcom.RecordTypeNote, XRef: "@N1@", Value: "stale value",
		Entity: &gedcom.Note{Text: "fresh\ntext", Tags: []*gedcom.Tag{
			{Level: 1, Tag: "CONT", Value: "stale"},
			{Level: 1, Tag: "CONC", Value: "stale"},
			{Level: 1, Tag: "REFN", Value: "r1"},
			{Level: 1, Tag: "_REF", Value: "r2"},
		}},
	}}}
	out = encodeToString(t, note)
	if !strings.Contains(out, "0 @N1@ NOTE fresh\n1 CONT text\n1 _REF r2\n") || strings.Contains(out, "stale") || strings.Contains(out, "REFN") {
		t.Errorf("note written from entity should take text from Note.Text and only extension tags from Tags:\n%s", out)
	}
}

// TestClearedTagsHonourDeletion is the panel-review case for #711: a decoded
// record's entity Tags hold the whole raw record, so a structure the caller
// deleted from the typed model must not come back from there.
func TestClearedTagsHonourDeletion(t *testing.T) {
	const in = "0 HEAD\n1 GEDC\n2 VERS 7.0\n" +
		"0 @I1@ INDI\n1 NAME A /B/\n1 SEX M\n1 BIRT\n2 DATE 1 JAN 1900\n1 DEAT\n2 DATE 1 JAN 1980\n" +
		"1 OCCU Smith\n1 UID abc\n1 _FSFTID KWCJ-QN7\n1 EXID 123\n2 TYPE http://x.example\n1 _MYAPP keep\n" +
		"0 TRLR\n"
	doc := decodeGEDCOM(t, in)
	indi := doc.GetIndividual("@I1@")
	indi.Names = nil
	indi.Events = nil
	indi.Attributes = nil
	indi.UIDs = nil
	indi.FamilySearchID = ""
	indi.ExternalIDs = nil
	clearRecordTags(doc)

	out := encodeToString(t, doc)
	for _, gone := range []string{"NAME", "BIRT", "DEAT", "OCCU", "UID", "_FSFTID", "EXID"} {
		if strings.Contains(out, " "+gone) {
			t.Errorf("deleted %s came back from entity Tags:\n%s", gone, out)
		}
	}
	if !strings.Contains(out, "1 SEX M\n") || !strings.Contains(out, "1 _MYAPP keep\n") {
		t.Errorf("kept structures missing:\n%s", out)
	}
}
