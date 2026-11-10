package encoder

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// assertRejected checks that err wraps want and names the record it was found
// in, and that nothing written before the error contains forged.
func assertRejected(t *testing.T, err, want error, record, out, forged string) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
	if !strings.Contains(err.Error(), record) {
		t.Errorf("error %q does not name record %q", err, record)
	}
	if forged != "" && strings.Contains(out, forged) {
		t.Errorf("output contains %q:\n%s", forged, out)
	}
}

// TestEncodeRejectsMalformedLine pins issue #726: a level, tag name or record
// identifier the decoder cannot read back as written is an error, not a line.
func TestEncodeRejectsMalformedLine(t *testing.T) {
	indi := func(tags ...*gedcom.Tag) *gedcom.Record {
		return &gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Tags: tags}
	}
	cases := map[string]struct {
		record *gedcom.Record
		name   string // the record the error must name
		forged string // text that must not reach the output
	}{
		// Level 0 is Tag's zero value; mid-record it opens a new record.
		"level 0 forges a record": {indi(
			&gedcom.Tag{Level: 1, Tag: "NAME", Value: "John /Doe/"},
			&gedcom.Tag{Level: 0, XRef: "@X@", Tag: "INDI"}), "@I1@ INDI", "0 @X@ INDI"},
		"zero-value level":      {indi(&gedcom.Tag{Tag: "NAME", Value: "John /Doe/"}), "@I1@ INDI", "0 NAME"},
		"negative level":        {indi(&gedcom.Tag{Level: -1, Tag: "NAME"}), "@I1@ INDI", "-1 NAME"},
		"level past 99":         {indi(&gedcom.Tag{Level: 100, Tag: "NAME"}), "@I1@ INDI", "100 NAME"},
		"empty tag":             {indi(&gedcom.Tag{Level: 1, Value: "x"}), "@I1@ INDI", "1  x"},
		"space in tag":          {indi(&gedcom.Tag{Level: 1, Tag: "NAME X", Value: "A /B/"}), "@I1@ INDI", "NAME X"},
		"tab in tag":            {indi(&gedcom.Tag{Level: 1, Tag: "NAME\tX"}), "@I1@ INDI", "NAME\t"},
		"unicode space in tag":  {indi(&gedcom.Tag{Level: 1, Tag: "NAME\u00a0X"}), "@I1@ INDI", "NAME\u00a0"},
		"space in record type":  {&gedcom.Record{XRef: "@I1@", Type: "IN DI"}, "@I1@ IN DI", "IN DI"},
		"empty record type":     {&gedcom.Record{XRef: "@I1@"}, "@I1@", "0 @I1@"},
		"xref without @":        {&gedcom.Record{XRef: "I1", Type: gedcom.RecordTypeIndividual}, "I1 INDI", "0 I1"},
		"unterminated with @":   {&gedcom.Record{XRef: "@I1@x", Type: gedcom.RecordTypeIndividual}, "@I1@x INDI", "0 @I1@x"},
		"unterminated spaced":   {&gedcom.Record{XRef: "@I 1", Type: gedcom.RecordTypeIndividual}, "@I 1 INDI", "0 @I 1"},
		"entity with bad xref":  {&gedcom.Record{XRef: "I1", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: "I1"}}, "I1 INDI", "0 I1"},
		"structural TRLR":       {&gedcom.Record{Type: "TRLR"}, "TRLR", "0 TRLR\n0 TRLR"},
		"structural HEAD":       {&gedcom.Record{Type: "HEAD"}, "HEAD", "0 HEAD\n0 TRLR"},
		"entity event type tab": {&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: "@I1@", Events: []*gedcom.Event{{Type: "BI RT"}}}}, "@I1@ INDI", "BI RT"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			err := Encode(&buf, docWithRecord(tc.record))
			assertRejected(t, err, ErrMalformedLine, tc.name, buf.String(), tc.forged)
		})
	}
}

// TestEncodeAcceptsReadableRecordXRef pins the identifiers that stay legal:
// the ordinary form, a 5.5 spaced one, and the unterminated shape a lenient
// decode recovers, which must re-encode byte for byte (issue #385).
func TestEncodeAcceptsReadableRecordXRef(t *testing.T) {
	for _, xref := range []string{"@I1@", "@I 1@", "@I1", "@", "@VOID@"} {
		t.Run(xref, func(t *testing.T) {
			var buf bytes.Buffer
			rec := &gedcom.Record{XRef: xref, Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{{Level: 1, Tag: "NAME", Value: "A /B/"}, {Level: 99, Tag: "_X"}}}
			if err := Encode(&buf, docWithRecord(rec)); err != nil {
				t.Fatalf("Encode() error = %v", err)
			}
			if want := "0 " + xref + " INDI\n"; !strings.Contains(buf.String(), want) {
				t.Errorf("output does not contain %q:\n%s", want, buf.String())
			}
		})
	}
	// A structural tag with an identifier is an ordinary record to the decoder.
	var buf bytes.Buffer
	if err := Encode(&buf, docWithRecord(&gedcom.Record{XRef: "@X1@", Type: "HEAD"})); err != nil {
		t.Errorf("Encode(0 @X1@ HEAD) error = %v", err)
	}
}

// TestEncodeRejectsHeaderMalformedLine covers the header, which is named HEAD.
func TestEncodeRejectsHeaderMalformedLine(t *testing.T) {
	var buf bytes.Buffer
	err := Encode(&buf, &gedcom.Document{Header: &gedcom.Header{Tags: []*gedcom.Tag{{Tag: "SOUR", Value: "App"}}}})
	assertRejected(t, err, ErrMalformedLine, "HEAD", buf.String(), "0 SOUR")
}

// TestEncodeRejectsInvalidUTF8 pins issue #726: the output declares UTF-8, so
// a string still in a legacy charset is an error rather than a file the
// decoder rejects.
func TestEncodeRejectsInvalidUTF8(t *testing.T) {
	const bad = "Ren\xe9 /Dupont/"
	cases := map[string]struct {
		doc  *gedcom.Document
		name string
	}{
		"tag value": {docWithRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual,
			Tags: []*gedcom.Tag{{Level: 1, Tag: "NAME", Value: bad}}}), "@I1@ INDI"},
		"CONT line": {docWithRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual,
			Tags: []*gedcom.Tag{{Level: 1, Tag: "NOTE", Value: "ok\n" + bad}}}), "@I1@ INDI"},
		"tag name": {docWithRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual,
			Tags: []*gedcom.Tag{{Level: 1, Tag: "_\xe9"}}}), "@I1@ INDI"},
		"entity name": {docWithRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual,
			Entity: &gedcom.Individual{XRef: "@I1@", Names: []*gedcom.PersonalName{{Full: bad}}}}), "@I1@ INDI"},
		"record value": {docWithRecord(&gedcom.Record{XRef: "@N1@", Type: gedcom.RecordTypeNote, Value: bad}), "@N1@ NOTE"},
		"header field": {&gedcom.Document{Header: &gedcom.Header{SourceSystem: bad}}, "HEAD"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			err := Encode(&buf, tc.doc)
			assertRejected(t, err, ErrInvalidUTF8, tc.name, buf.String(), "\xe9")
		})
	}
}

// TestStreamEncoderRejectsMalformedLine covers the streaming encoder, which
// shares writeRecord.
func TestStreamEncoderRejectsMalformedLine(t *testing.T) {
	var buf bytes.Buffer
	enc := NewStreamEncoder(&buf)
	if err := enc.WriteHeader(&gedcom.Header{}); err != nil {
		t.Fatalf("WriteHeader() error = %v", err)
	}
	err := enc.WriteRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual,
		Tags: []*gedcom.Tag{{Tag: "NAME", Value: "A /B/"}}})
	_ = enc.Flush()
	assertRejected(t, err, ErrMalformedLine, "@I1@ INDI", buf.String(), "0 NAME")
}

// TestEncodeHeaderFieldsDeclareCHAR pins issue #726: a hand-built header with
// no Encoding still declares the UTF-8 it is written in, except for 7.0, which
// removed CHAR (see TestHeaderFieldOrder).
func TestEncodeHeaderFieldsDeclareCHAR(t *testing.T) {
	for name, header := range map[string]*gedcom.Header{
		"5.5.1":      {Version: gedcom.Version551},
		"5.5":        {Version: gedcom.Version55},
		"no version": {},
		"nil header": nil,
	} {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := Encode(&buf, &gedcom.Document{Header: header}); err != nil {
				t.Fatalf("Encode() error = %v", err)
			}
			if !strings.Contains(buf.String(), "\n1 CHAR UTF-8\n") {
				t.Errorf("header declares no CHAR:\n%s", buf.String())
			}
		})
	}
}

// TestEncodeUTF8RoundTripsWithoutEncoding is the end-to-end case from #726: a
// hand-built 5.5.1 document with non-ASCII text and no Encoding decodes back
// strictly with every record.
func TestEncodeUTF8RoundTripsWithoutEncoding(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version551},
		Records: []*gedcom.Record{
			{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Tags: []*gedcom.Tag{{Level: 1, Tag: "NAME", Value: "René /Dupont/"}}},
			{XRef: "@I2@", Type: gedcom.RecordTypeIndividual, Tags: []*gedcom.Tag{{Level: 1, Tag: "NAME", Value: "Zoë /Dupont/"}}},
		},
	}
	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	got, err := decoder.DecodeWithOptions(&buf, &decoder.DecodeOptions{StrictMode: true})
	if err != nil {
		t.Fatalf("strict Decode() error = %v", err)
	}
	if len(got.Records) != 2 || got.Header.Encoding != gedcom.EncodingUTF8 {
		t.Errorf("decoded %d records, encoding %q; want 2, UTF-8", len(got.Records), got.Header.Encoding)
	}
}
