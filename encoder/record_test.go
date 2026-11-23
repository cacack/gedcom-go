package encoder

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Issue #878: EncodeRecord writes one record with no HEAD and no TRLR.

// encodeEachRecord encodes doc's records one EncodeRecord call at a time and
// returns their outputs.
func encodeEachRecord(t *testing.T, doc *gedcom.Document, opts *EncodeOptions) []string {
	t.Helper()
	outs := make([]string, 0, len(doc.Records))
	for _, record := range doc.Records {
		var buf bytes.Buffer
		if err := EncodeRecord(&buf, doc, record, opts); err != nil {
			t.Fatalf("EncodeRecord(%s): %v", record.XRef, err)
		}
		outs = append(outs, buf.String())
	}
	return outs
}

// assertRecordsMatchEncode checks that the records encoded one at a time are,
// in order, exactly the bytes Encode writes between the header and TRLR.
func assertRecordsMatchEncode(t *testing.T, doc *gedcom.Document, opts *EncodeOptions) {
	t.Helper()
	var whole bytes.Buffer
	if err := EncodeWithOptions(&whole, doc, opts); err != nil {
		t.Fatalf("EncodeWithOptions: %v", err)
	}
	outs := encodeEachRecord(t, doc, opts)
	eol := opts.effectiveLineEnding()
	records := strings.Join(outs, "")
	if !strings.HasSuffix(whole.String(), eol+records+"0 TRLR"+eol) {
		t.Fatalf("records encoded one at a time are not the records Encode wrote\nEncode:\n%s\nEncodeRecord:\n%s", whole.String(), records)
	}
	for i, out := range outs {
		if strings.Contains(out, "0 HEAD") || strings.Contains(out, "0 TRLR") {
			t.Errorf("record %d output holds a header or trailer:\n%s", i, out)
		}
		if strings.Count(eol+out, eol+"0 ") != 1 {
			t.Errorf("record %d output is not exactly one record:\n%s", i, out)
		}
	}
}

func TestEncodeRecord_MatchesEncodeForDecodedFiles(t *testing.T) {
	var paths []string
	for _, pattern := range []string{"../testdata/gedcom-5.5/*.ged", "../testdata/gedcom-5.5.1/*.ged", "../testdata/gedcom-7.0/*.ged"} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		paths = append(paths, matches...)
	}
	if len(paths) == 0 {
		t.Fatal("no testdata files found")
	}

	optionSets := map[string]func() *EncodeOptions{
		"default": DefaultOptions,
		"crlf":    func() *EncodeOptions { o := DefaultOptions(); o.LineEnding = "\r\n"; return o },
		"short lines": func() *EncodeOptions {
			o := DefaultOptions()
			o.MaxLineLength = 20
			return o
		},
		"drop unknown": func() *EncodeOptions { o := DefaultOptions(); o.DropUnknownTags = true; return o },
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		doc, err := decoder.Decode(bytes.NewReader(data))
		if err != nil {
			continue // a file the decoder rejects has no records to compare
		}
		for name, opts := range optionSets {
			t.Run(filepath.Base(path)+"/"+name, func(t *testing.T) {
				assertRecordsMatchEncode(t, doc, opts())
			})
		}
	}
}

// longNoteDoc is a document whose one record is written from its entity, with
// a note longer than the default line length and a pointer to an SNOTE record.
func longNoteDoc(version gedcom.Version) (*gedcom.Document, *gedcom.Record) {
	indi := &gedcom.Record{
		XRef: "@I1@",
		Type: gedcom.RecordTypeIndividual,
		Entity: &gedcom.Individual{
			XRef:        "@I1@",
			Names:       []*gedcom.PersonalName{{Full: "Ann /Example/"}},
			NoteXRefs:   []string{"@N1@"},
			InlineNotes: []string{strings.Repeat("word ", 80)},
		},
	}
	snote := &gedcom.Record{
		XRef:   "@N1@",
		Type:   gedcom.RecordTypeSharedNote,
		Entity: &gedcom.SharedNote{XRef: "@N1@", Text: "Shared text"},
	}
	doc := &gedcom.Document{
		Header:  &gedcom.Header{Version: version},
		Records: []*gedcom.Record{indi, snote},
	}
	return doc, indi
}

func TestEncodeRecord_VersionFromDocument(t *testing.T) {
	doc, indi := longNoteDoc(gedcom.Version70)
	assertRecordsMatchEncode(t, doc, DefaultOptions())

	var withDoc, withoutDoc bytes.Buffer
	if err := EncodeRecord(&withDoc, doc, indi, nil); err != nil {
		t.Fatal(err)
	}
	if err := EncodeRecord(&withoutDoc, nil, indi, nil); err != nil {
		t.Fatal(err)
	}

	// 7.0 from the header: no CONC, and the pointer to the SNOTE record is SNOTE.
	if strings.Contains(withDoc.String(), "CONC") {
		t.Errorf("7.0 record wrapped with CONC:\n%s", withDoc.String())
	}
	if !strings.Contains(withDoc.String(), "1 SNOTE @N1@\n") {
		t.Errorf("7.0 record lacks SNOTE pointer:\n%s", withDoc.String())
	}

	// No document, no TargetVersion: unknown version, so lines wrap and the
	// pointer stays NOTE.
	if !strings.Contains(withoutDoc.String(), "CONC") {
		t.Errorf("record with no document did not wrap:\n%s", withoutDoc.String())
	}
	if !strings.Contains(withoutDoc.String(), "1 NOTE @N1@\n") {
		t.Errorf("record with no document lacks NOTE pointer:\n%s", withoutDoc.String())
	}
}

func TestEncodeRecord_TargetVersionWithoutDocument(t *testing.T) {
	_, indi := longNoteDoc("")
	var buf bytes.Buffer
	if err := EncodeRecord(&buf, nil, indi, &EncodeOptions{TargetVersion: gedcom.Version70}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "CONC") {
		t.Errorf("TargetVersion 7.0 wrapped with CONC:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "1 NOTE @N1@\n") {
		t.Errorf("pointer with no document should stay NOTE:\n%s", buf.String())
	}
}

func TestEncodeRecord_FiveFiveOneDocumentMatchesEncode(t *testing.T) {
	doc, _ := longNoteDoc(gedcom.Version551)
	assertRecordsMatchEncode(t, doc, DefaultOptions())
}

func TestEncodeRecord_InlineMediaStaysInline(t *testing.T) {
	record := &gedcom.Record{
		XRef: "@I1@",
		Type: gedcom.RecordTypeIndividual,
		Entity: &gedcom.Individual{
			XRef:  "@I1@",
			Media: []*gedcom.MediaLink{inlinePhotoLink()},
		},
	}
	doc := &gedcom.Document{Header: &gedcom.Header{Version: gedcom.Version70}, Records: []*gedcom.Record{record}}

	var buf bytes.Buffer
	if err := EncodeRecord(&buf, doc, record, nil); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if strings.Contains(got, "INLINE_OBJE") {
		t.Errorf("inline media promoted to a record:\n%s", got)
	}
	if !strings.Contains(got, "1 OBJE\n2 FILE ") {
		t.Errorf("inline media not written inline:\n%s", got)
	}
}

func TestEncodeRecord_NilArguments(t *testing.T) {
	var buf bytes.Buffer
	if err := EncodeRecord(&buf, nil, nil, nil); err != nil {
		t.Errorf("nil record: error = %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("nil record wrote %q", buf.String())
	}

	record := &gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Tags: []*gedcom.Tag{nil, {Level: 1, Tag: "SEX", Value: "F"}}}
	if err := EncodeRecord(&buf, &gedcom.Document{}, record, nil); err != nil {
		t.Fatal(err)
	}
	if got, want := buf.String(), "0 @I1@ INDI\n1 SEX F\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEncodeRecord_DropUnknownCustomRecord(t *testing.T) {
	record := &gedcom.Record{Type: "_EVDEF", Tags: []*gedcom.Tag{{Level: 1, Tag: "NAME", Value: "x"}}}
	var buf bytes.Buffer
	if err := EncodeRecord(&buf, nil, record, &EncodeOptions{DropUnknownTags: true}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("custom record written with DropUnknownTags: %q", buf.String())
	}
}

func TestEncodeRecord_Errors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	record := &gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual}
	var buf bytes.Buffer
	if err := EncodeRecord(&buf, nil, record, &EncodeOptions{Context: ctx}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled context: error = %v, want context.Canceled", err)
	}
	if buf.Len() != 0 {
		t.Errorf("cancelled context wrote %q", buf.String())
	}

	bad := &gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Tags: []*gedcom.Tag{{Level: 1, Tag: "NA\nME"}}}
	if err := EncodeRecord(&buf, nil, bad, nil); !errors.Is(err, ErrLineBreak) {
		t.Errorf("line break in tag: error = %v, want ErrLineBreak", err)
	}
}
