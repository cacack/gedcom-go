package decoder_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// addTestdataSeeds seeds f with the repository's GEDCOM fixtures plus a few
// synthetic shapes. Seeds must stay small: every seed runs on each plain
// `go test`, and the mutator works over seed-sized inputs.
func addTestdataSeeds(f *testing.F) {
	const maxSeedSize = 64 << 10
	for _, dir := range []string{"gedcom-5.5", "gedcom-5.5.1", "gedcom-7.0", "malformed", "edge-cases", "encoding"} {
		files, _ := filepath.Glob(filepath.Join("..", "testdata", dir, "*.ged"))
		for _, file := range files {
			if info, err := os.Stat(file); err != nil || info.Size() > maxSeedSize {
				continue
			}
			data, err := os.ReadFile(file) // #nosec G304 -- path comes from the repo's own testdata
			if err != nil {
				continue
			}
			f.Add(data)
		}
	}

	f.Add([]byte("0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n0 @I1@ INDI\n1 NAME John /Smith/\n1 FAMS @F1@\n0 @F1@ FAM\n1 HUSB @I1@\n0 TRLR\n"))
	f.Add([]byte("0 HEAD\n1 GEDC\n2 VERS 7.0\n0 @N1@ SNOTE one\n1 CONT two\n0 @I1@ INDI\n1 SNOTE @N1@\n1 BIRT\n2 DATE 1 JAN 1900\n0 TRLR\n"))
	f.Add([]byte("0 HEAD\r\n1 CHAR ANSEL\r\n0 @I1 INDI\r\n1 BIRT\r\n4 DATE 1900\r\n0 @I1@ INDI\r\n0 @I1@ INDI\r\n"))
	f.Add([]byte(""))
}

// FuzzDecode holds every decode entry point to ADR 0007 on arbitrary bytes:
// errors and diagnostics are expected, panics are not. A strict decode gives a
// document or an error, never both and never neither; a lenient one honours
// MaxDiagnostics; and whatever document comes back, with an error or without,
// can be read through its accessors and cloned.
func FuzzDecode(f *testing.F) {
	addTestdataSeeds(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		strict, err := decoder.DecodeWithOptions(bytes.NewReader(data), &decoder.DecodeOptions{StrictMode: true})
		if (strict == nil) == (err == nil) {
			t.Fatalf("strict decode returned document %v with error %v", strict != nil, err)
		}

		const maxDiagnostics = 8
		res, err := decoder.DecodeWithDiagnostics(bytes.NewReader(data), &decoder.DecodeOptions{MaxDiagnostics: maxDiagnostics})
		if res == nil {
			// Only a read that fails before any line was read gives no result.
			if err == nil {
				t.Fatal("lenient decode returned neither a result nor an error")
			}
			return
		}
		if len(res.Diagnostics) > maxDiagnostics+1 {
			t.Fatalf("%d diagnostics, over MaxDiagnostics+1", len(res.Diagnostics))
		}

		doc := res.Document
		if doc == nil {
			t.Fatalf("lenient decode returned a result with no document (error %v)", err)
		}
		walkDocument(doc)

		clone := doc.Clone()
		if len(clone.Records) != len(doc.Records) {
			t.Fatalf("Clone() has %d records, the document %d", len(clone.Records), len(doc.Records))
		}
		walkDocument(clone)
	})
}

// walkDocument calls the document's accessors; none may panic, whatever the
// decoder recovered.
func walkDocument(doc *gedcom.Document) {
	_ = doc.Individuals()
	_ = doc.Families()
	_ = doc.Sources()
	_ = doc.Repositories()
	_ = doc.Notes()
	_ = doc.SharedNotes()
	_ = doc.MediaObjects()
	_ = doc.Submitters()
	_ = doc.Submissions()
	_ = doc.MinimumVersion()
	_ = doc.RequiresGEDCOM7()
	for _, rec := range doc.Records {
		_ = doc.GetRecord(rec.XRef)
		_ = doc.GetIndividual(rec.XRef)
		_ = doc.GetFamily(rec.XRef)
		_ = doc.IsPointerXRef(rec.XRef)
	}
}

// numericTagLine matches an encoded line whose tag is all digits (#851).
var numericTagLine = regexp.MustCompile(`(?m)^\d+ \d+( |$)`)

// FuzzDecodeRoundTrip holds the decoder and encoder to Lossless
// Representation on arbitrary bytes: whatever a lenient decode recovers, the
// encoder writes without error, and decoding that output gives the same
// records in the same order. Encoding the second document must then reproduce
// the first output byte for byte, so nothing drifts over repeated round trips
// (the growth #427 described).
func FuzzDecodeRoundTrip(f *testing.F) {
	addTestdataSeeds(f)
	f.Add([]byte("0 HEAD\n0 @NOTE@ NOTE some text\n1 CONC @NOTE@ more\n0 TRLR\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		first, err := decoder.Decode(bytes.NewReader(data))
		if err != nil {
			return
		}

		var out1 bytes.Buffer
		if err := encoder.Encode(&out1, first); err != nil {
			// Known soft-invariant class (#851): below level 0 the parser
			// reads a lone "@" before a tag ("1 @ NOTE x") as an identifier,
			// which the encoder refuses to write.
			if errors.Is(err, encoder.ErrMalformedLine) && strings.Contains(err.Error(), `xref "@"`) {
				return
			}
			t.Fatalf("Encode() of a decoded document error = %v", err)
		}

		// Known soft-invariant class (#851, the residue of #427): the parser
		// looks for the tag from the start of a line with no XRef, so an
		// all-digit tag is found inside the level ("1 1 x" reads as value
		// "1 x") and the line grows on every round trip.
		if numericTagLine.Match(out1.Bytes()) {
			return
		}

		second, err := decoder.Decode(bytes.NewReader(out1.Bytes()))
		if err != nil {
			t.Fatalf("Decode() of encoder output error = %v\n%q", err, out1.Bytes())
		}
		if len(second.Records) != len(first.Records) {
			t.Fatalf("decoded %d records, encoded %d\n%q", len(second.Records), len(first.Records), out1.Bytes())
		}
		for i, rec := range first.Records {
			if got := second.Records[i]; got.XRef != rec.XRef || got.Type != rec.Type {
				t.Fatalf("record %d read back as %q %q, encoded as %q %q\n%q",
					i, got.XRef, got.Type, rec.XRef, rec.Type, out1.Bytes())
			}
		}

		var out2 bytes.Buffer
		if err := encoder.Encode(&out2, second); err != nil {
			t.Fatalf("second Encode() error = %v\n%q", err, out1.Bytes())
		}
		if !bytes.Equal(out1.Bytes(), out2.Bytes()) {
			t.Fatalf("round trip is not a fixpoint\nfirst:  %q\nsecond: %q", out1.Bytes(), out2.Bytes())
		}
	})
}
