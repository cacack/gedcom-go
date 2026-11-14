package converter_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/cacack/gedcom-go/v3/converter"
	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/validator"
)

// FuzzValidateConvert runs whatever a lenient decode recovers from arbitrary
// bytes through the validator and through a conversion to each GEDCOM
// version. Issues, data loss and conversion errors are expected; panics are
// not. A conversion that succeeds must leave the caller's document alone and
// give one the encoder writes and the decoder reads back with the same number
// of records.
func FuzzValidateConvert(f *testing.F) {
	// Seeds must stay small: every seed runs on each plain `go test`, three
	// conversions apiece, and the mutator works over seed-sized inputs.
	const maxSeedSize = 16 << 10
	for _, dir := range []string{"gedcom-5.5", "gedcom-5.5.1", "gedcom-7.0", "malformed", "edge-cases"} {
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
	f.Add([]byte("0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n0 @I1@ INDI\n1 NAME John /Smith/\n1 BIRT\n2 DATE @#DJULIAN@ 1 JAN 1700\n1 NOTE a@b.com\n2 CONC  more\n0 TRLR\n"))
	f.Add([]byte("0 HEAD\n1 GEDC\n2 VERS 7.0\n1 SCHMA\n2 TAG _X http://example.com/x\n0 @N1@ SNOTE one\n0 @I1@ INDI\n1 SNOTE @N1@\n1 EXID 1\n2 TYPE http://example.com\n1 FAMC @VOID@\n0 TRLR\n"))
	f.Add([]byte(""))

	f.Fuzz(func(t *testing.T, data []byte) {
		doc, err := decoder.Decode(bytes.NewReader(data))
		if err != nil || doc == nil {
			return
		}
		_ = validator.New().ValidateAll(doc)

		// A decoded document the encoder refuses is FuzzDecodeRoundTrip's
		// finding (package decoder), not a conversion's.
		var before bytes.Buffer
		if err := encoder.Encode(&before, doc); err != nil {
			return
		}

		for _, target := range []gedcom.Version{gedcom.Version55, gedcom.Version551, gedcom.Version70} {
			converted, report, err := converter.Convert(doc, target)
			if err != nil {
				if converted != nil {
					t.Fatalf("Convert(%s) returned a document with error %v", target, err)
				}
				continue
			}
			if converted == nil || report == nil {
				t.Fatalf("Convert(%s) returned document %v, report %v and no error", target, converted != nil, report != nil)
			}
			_ = report.String()

			var after bytes.Buffer
			if err := encoder.Encode(&after, doc); err != nil || !bytes.Equal(before.Bytes(), after.Bytes()) {
				t.Fatalf("Convert(%s) modified the caller's document\nbefore: %q\nafter:  %q", target, before.Bytes(), after.Bytes())
			}

			var out bytes.Buffer
			if err := encoder.Encode(&out, converted); err != nil {
				t.Fatalf("Encode() of the %s conversion error = %v", target, err)
			}
			got, err := decoder.Decode(bytes.NewReader(out.Bytes()))
			if err != nil {
				t.Fatalf("Decode() of the encoded %s conversion error = %v\n%q", target, err, out.Bytes())
			}
			if len(got.Records) != len(converted.Records) {
				t.Fatalf("decoded %d records, the %s conversion has %d\n%q", len(got.Records), target, len(converted.Records), out.Bytes())
			}
		}
	})
}
