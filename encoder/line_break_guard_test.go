package encoder

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// forgedRecord is what a line break in a tag name or identifier would smuggle
// onto its own line: text starting with a digit re-decodes as a record the
// caller never wrote (issue #466).
const forgedRecord = "0 @F9@ FAM"

// lineBreaks are the three terminators the parser recognises.
var lineBreaks = map[string]string{"LF": "\n", "CRLF": "\r\n", "CR": "\r"}

// assertLineBreakRejected checks that err is ErrLineBreak and that nothing
// written before the error contains the forged record.
func assertLineBreakRejected(t *testing.T, err error, out string) {
	t.Helper()
	if !errors.Is(err, ErrLineBreak) {
		t.Fatalf("error = %v, want ErrLineBreak", err)
	}
	if strings.Contains(out, "@F9@") {
		t.Errorf("forged record reached the output:\n%s", out)
	}
}

// TestEncodeRejectsLineBreakInTagOrXRef pins that a line break in Tag.Tag,
// Tag.XRef, Record.Type or Record.XRef is an error rather than a new line
// (issue #466). Unlike a value, a tag name or identifier has no CONT form, so
// there is no way to write one losslessly.
func TestEncodeRejectsLineBreakInTagOrXRef(t *testing.T) {
	cases := map[string]func(brk string) *gedcom.Document{
		"Tag.Tag": func(brk string) *gedcom.Document {
			return docWithRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{{Level: 1, Tag: "NAME" + brk + forgedRecord, Value: "John /Doe/"}}})
		},
		"Tag.XRef": func(brk string) *gedcom.Document {
			return docWithRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{{Level: 1, XRef: "@X@" + brk + forgedRecord, Tag: "NOTE"}}})
		},
		"Record.Type": func(brk string) *gedcom.Document {
			return docWithRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordType("INDI" + brk + forgedRecord)})
		},
		"Record.XRef": func(brk string) *gedcom.Document {
			return docWithRecord(&gedcom.Record{XRef: "@I1@" + brk + forgedRecord, Type: gedcom.RecordTypeIndividual})
		},
		"Header.Tags": func(brk string) *gedcom.Document {
			return &gedcom.Document{Header: &gedcom.Header{
				Tags: []*gedcom.Tag{{Level: 1, Tag: "SOUR" + brk + forgedRecord, Value: "x"}}}}
		},
		// The entity path: a typed field that becomes a tag name, and the
		// record identifier of a record with no raw tags.
		"entity event type": func(brk string) *gedcom.Document {
			return docWithRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual,
				Entity: &gedcom.Individual{XRef: "@I1@",
					Events: []*gedcom.Event{{Type: gedcom.EventType("BIRT" + brk + forgedRecord)}}}})
		},
		"entity Record.XRef": func(brk string) *gedcom.Document {
			return docWithRecord(&gedcom.Record{XRef: "@I1@" + brk + forgedRecord, Type: gedcom.RecordTypeIndividual,
				Entity: &gedcom.Individual{XRef: "@I1@"}})
		},
	}

	for name, build := range cases {
		for brkName, brk := range lineBreaks {
			t.Run(name+"/"+brkName, func(t *testing.T) {
				var buf bytes.Buffer
				err := Encode(&buf, build(brk))
				assertLineBreakRejected(t, err, buf.String())
			})
		}
	}
}

// TestStreamEncoderRejectsLineBreakInXRef covers the streaming encoder, which
// shares writeRecord but reports through its own error state.
func TestStreamEncoderRejectsLineBreakInXRef(t *testing.T) {
	var buf bytes.Buffer
	enc := NewStreamEncoder(&buf)
	if err := enc.WriteHeader(&gedcom.Header{}); err != nil {
		t.Fatalf("WriteHeader() error = %v", err)
	}
	err := enc.WriteRecord(&gedcom.Record{XRef: "@I1@\n" + forgedRecord, Type: gedcom.RecordTypeIndividual})
	_ = enc.Flush()
	assertLineBreakRejected(t, err, buf.String())
}

func docWithRecord(record *gedcom.Record) *gedcom.Document {
	return &gedcom.Document{Header: &gedcom.Header{}, Records: []*gedcom.Record{record}}
}

// TestEncodeHeaderFieldsLineBreak covers the hand-built header path, which
// writes typed fields rather than raw tags: a line break in a value field
// folds into CONT, and one in the Submitter pointer is rejected (issue #466).
func TestEncodeHeaderFieldsLineBreak(t *testing.T) {
	folded := map[string]struct {
		header *gedcom.Header
		opts   *EncodeOptions
		want   string
	}{
		"Version": {&gedcom.Header{Version: gedcom.Version("5.5.1\n" + forgedRecord)}, nil,
			"2 VERS 5.5.1\n3 CONT " + forgedRecord + "\n"},
		"TargetVersion": {&gedcom.Header{}, &EncodeOptions{TargetVersion: gedcom.Version("5.5.1\n" + forgedRecord)},
			"2 VERS 5.5.1\n3 CONT " + forgedRecord + "\n"},
		"SourceSystem": {&gedcom.Header{SourceSystem: "App\n" + forgedRecord}, nil,
			"1 SOUR App\n2 CONT " + forgedRecord + "\n"},
		"Language": {&gedcom.Header{Language: "English\n" + forgedRecord}, nil,
			"1 LANG English\n2 CONT " + forgedRecord + "\n"},
	}
	for name, tc := range folded {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := EncodeWithOptions(&buf, &gedcom.Document{Header: tc.header}, tc.opts); err != nil {
				t.Fatalf("Encode() error = %v", err)
			}
			if !strings.Contains(buf.String(), tc.want) {
				t.Errorf("output does not contain %q:\n%s", tc.want, buf.String())
			}
			if strings.Contains(buf.String(), "\n"+forgedRecord) {
				t.Errorf("forged record reached the output:\n%s", buf.String())
			}
		})
	}

	for brkName, brk := range lineBreaks {
		t.Run("Submitter/"+brkName, func(t *testing.T) {
			var buf bytes.Buffer
			err := Encode(&buf, &gedcom.Document{Header: &gedcom.Header{Submitter: "@U1@" + brk + forgedRecord}})
			assertLineBreakRejected(t, err, buf.String())
		})
	}
}

// TestEncodeHeaderFoldedValueRoundTrip pins what the package doc promises for
// a header value folded where the grammar allows no CONT: the re-decoded typed
// field keeps only the first line, and the CONT survives in Header.Tags, so
// no record is forged and nothing is lost.
func TestEncodeHeaderFoldedValueRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	doc := &gedcom.Document{Header: &gedcom.Header{Version: gedcom.Version551, SourceSystem: "App\n" + forgedRecord}}
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	got, err := decoder.Decode(&buf)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Header.SourceSystem != "App" {
		t.Errorf("SourceSystem = %q, want %q", got.Header.SourceSystem, "App")
	}
	if len(got.Families()) != 0 {
		t.Errorf("decoded %d families, want 0 (forged record)", len(got.Families()))
	}
	var cont bool
	for _, tag := range got.Header.Tags {
		if tag.Tag == "CONT" && tag.Value == forgedRecord {
			cont = true
		}
	}
	if !cont {
		t.Errorf("Header.Tags lost the CONT line: %+v", got.Header.Tags)
	}
}

// TestStreamEncoderRejectsLineBreakInHeader covers WriteHeader, and that the
// error sticks: later calls return it rather than writing past the bad line.
func TestStreamEncoderRejectsLineBreakInHeader(t *testing.T) {
	var buf bytes.Buffer
	enc := NewStreamEncoder(&buf)
	err := enc.WriteHeader(&gedcom.Header{Tags: []*gedcom.Tag{{Level: 1, Tag: "SOUR\n" + forgedRecord, Value: "x"}}})
	assertLineBreakRejected(t, err, buf.String())

	if err := enc.WriteRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual}); !errors.Is(err, ErrLineBreak) {
		t.Errorf("WriteRecord() after failed header error = %v, want ErrLineBreak", err)
	}
	if err := enc.Close(); !errors.Is(err, ErrLineBreak) {
		t.Errorf("Close() error = %v, want ErrLineBreak", err)
	}
	if strings.Contains(buf.String(), "@I1@") || strings.Contains(buf.String(), "TRLR") {
		t.Errorf("stream wrote past the failed header:\n%s", buf.String())
	}
}
