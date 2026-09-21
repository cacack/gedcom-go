package converter_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/converter"
	"github.com/cacack/gedcom-go/v2/decoder"
	"github.com/cacack/gedcom-go/v2/encoder"
	"github.com/cacack/gedcom-go/v2/gedcom"
)

// The converter does not transform HEAD.SUBM -- the pointer means the same
// thing in 5.5, 5.5.1 and 7.0 -- so the field and its raw tag must simply
// survive a conversion unchanged. That was unobservable before issue #503,
// when Header.Submitter was always empty; these tests pin it now that the
// decoder populates it.

const submitterHeader551 = `0 HEAD
1 SOUR CONVTEST
1 SUBM @U1@
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @U1@ SUBM
1 NAME Conversion Submitter
0 @I1@ INDI
1 NAME Test /Person/
0 TRLR
`

// submTagValue returns the value of the level-1 SUBM header tag, or "" when
// the header does not carry one.
func submTagValue(h *gedcom.Header) string {
	for _, tag := range h.Tags {
		if tag != nil && tag.Level == 1 && tag.Tag == "SUBM" {
			return tag.Value
		}
	}
	return ""
}

// encodeHeader encodes doc and returns the text up to the first record, which
// is the whole HEAD structure.
func encodeHeader(t *testing.T, doc *gedcom.Document) string {
	t.Helper()

	var buf bytes.Buffer
	if err := encoder.Encode(&buf, doc); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return strings.SplitN(buf.String(), "0 @", 2)[0]
}

// TestConvertRoundTripsHeaderSubmitter takes a decoded 5.5.1 document up to
// 7.0, back through encode/decode, and down to 5.5.1 again. The typed field,
// the raw header tag and the encoded line have to agree at every step.
func TestConvertRoundTripsHeaderSubmitter(t *testing.T) {
	doc, err := decoder.Decode(strings.NewReader(submitterHeader551))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	up, _, err := converter.Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatalf("convert to 7.0: %v", err)
	}
	if up.Header.Submitter != "@U1@" {
		t.Errorf("7.0 Header.Submitter = %q, want @U1@", up.Header.Submitter)
	}
	if got := submTagValue(up.Header); got != "@U1@" {
		t.Errorf("7.0 raw header SUBM tag = %q, want @U1@", got)
	}
	if head := encodeHeader(t, up); !strings.Contains(head, "1 SUBM @U1@") {
		t.Errorf("7.0 output dropped the header submitter:\n%s", head)
	}

	// Re-decoding the 7.0 output proves the pointer survived the file, not
	// just the in-memory document.
	var buf bytes.Buffer
	if err := encoder.Encode(&buf, up); err != nil {
		t.Fatalf("encode 7.0: %v", err)
	}
	reread, err := decoder.Decode(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatalf("re-decode 7.0: %v", err)
	}
	if reread.Header.Submitter != "@U1@" {
		t.Fatalf("re-decoded 7.0 Header.Submitter = %q, want @U1@", reread.Header.Submitter)
	}

	down, _, err := converter.Convert(reread, gedcom.Version551)
	if err != nil {
		t.Fatalf("convert back to 5.5.1: %v", err)
	}
	if down.Header.Submitter != "@U1@" {
		t.Errorf("5.5.1 Header.Submitter = %q, want @U1@", down.Header.Submitter)
	}
	if got := submTagValue(down.Header); got != "@U1@" {
		t.Errorf("5.5.1 raw header SUBM tag = %q, want @U1@", got)
	}
	if down.GetSubmitter(down.Header.Submitter) == nil {
		t.Errorf("Header.Submitter %q names no record after the round trip", down.Header.Submitter)
	}
	if head := encodeHeader(t, down); !strings.Contains(head, "1 SUBM @U1@") {
		t.Errorf("5.5.1 output dropped the header submitter:\n%s", head)
	}
}

// TestConvertEmitsHeaderSubmitterFromTypedField covers the other encoder path:
// a hand-built header has no Tags, so writeHeaderFields writes SUBM from the
// typed field alone, in each version's grammar position.
func TestConvertEmitsHeaderSubmitterFromTypedField(t *testing.T) {
	for _, target := range []gedcom.Version{gedcom.Version55, gedcom.Version551, gedcom.Version70} {
		t.Run(target.String(), func(t *testing.T) {
			record := &gedcom.Record{
				XRef:   "@U1@",
				Type:   gedcom.RecordTypeSubmitter,
				Entity: &gedcom.Submitter{XRef: "@U1@", Name: "Conversion Submitter"},
			}
			doc := &gedcom.Document{
				Header: &gedcom.Header{
					Version:   gedcom.Version551,
					Encoding:  gedcom.EncodingUTF8,
					Submitter: "@U1@",
				},
				Records: []*gedcom.Record{record},
				Trailer: &gedcom.Trailer{},
				XRefMap: map[string]*gedcom.Record{"@U1@": record},
			}

			out, _, err := converter.Convert(doc, target)
			if err != nil {
				t.Fatalf("convert to %s: %v", target, err)
			}
			if out.Header.Submitter != "@U1@" {
				t.Errorf("%s: Header.Submitter = %q, want @U1@", target, out.Header.Submitter)
			}
			if head := encodeHeader(t, out); !strings.Contains(head, "1 SUBM @U1@") {
				t.Errorf("%s: encoded header has no submitter:\n%s", target, head)
			}
		})
	}
}
