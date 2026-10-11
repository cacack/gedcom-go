package encoder

import (
	"bytes"
	"errors"
	"strings"
	"testing"

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
