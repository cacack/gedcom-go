package encoder

import (
	"bytes"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// FuzzEncodeRoundTrip holds the encoder to the bar issue #726 set: whatever it
// writes without error, the decoder reads back with the same record count. The
// fuzzed record sits between two fixed ones, so a line that ends the file
// early, opens a record of its own or swallows the next one changes the count.
func FuzzEncodeRoundTrip(f *testing.F) {
	f.Add("@I1@", "INDI", "", 1, "", "NAME", "John /Doe/", "5.5.1", "")
	f.Add("@I1@", "INDI", "", 0, "@X@", "INDI", "", "5.5.1", "")
	f.Add("I1", "INDI", "", 1, "", "NAME X", "A /B/", "", "")
	f.Add("@N1@", "NOTE", "line one\nline two", 99, "", "_X", "a\r\nb", "7.0", "")
	f.Add("", "TRLR", "", 100, "", "", "Ren\xe9", "5.5", "ANSEL")
	f.Add("@I1", "INDI", "", 2, "@I 1@", "NOTE", "@@ escaped", "5.5.1", "UTF-8")

	f.Fuzz(func(t *testing.T, recXRef, recType, recValue string, level int, xref, tag, value, version, encoding string) {
		doc := &gedcom.Document{
			Header: &gedcom.Header{Version: gedcom.Version(version), Encoding: gedcom.Encoding(encoding)},
			Records: []*gedcom.Record{
				{XRef: "@A@", Type: gedcom.RecordTypeIndividual},
				{XRef: recXRef, Type: gedcom.RecordType(recType), Value: recValue,
					Tags: []*gedcom.Tag{{Level: level, XRef: xref, Tag: tag, Value: value}}},
				{XRef: "@Z@", Type: gedcom.RecordTypeIndividual},
			},
		}

		var buf bytes.Buffer
		if err := Encode(&buf, doc); err != nil {
			return
		}
		out := buf.String()

		got, err := decoder.Decode(&buf)
		if err != nil {
			t.Fatalf("Decode() of encoder output error = %v\n%q", err, out)
		}
		if len(got.Records) != len(doc.Records) {
			t.Fatalf("decoded %d records, encoded %d\n%q", len(got.Records), len(doc.Records), out)
		}
	})
}
