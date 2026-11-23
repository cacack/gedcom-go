package gedcomgo

import (
	"bytes"
	"os"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
)

// Layout-preserving round-trip: with PreserveLayout on both sides, decode
// then encode of an unmodified document must give back the input byte for
// byte, with none of normalizeGEDCOM's accepted differences applied (#876).
//
// layoutKnownBad lists the fixtures that cannot, with why. It is self-cleaning
// like the lists in byte_roundtrip_test.go: a listed fixture that starts
// passing fails asking to be removed.
var layoutKnownBad = map[string]string{
	// The encoder writes UTF-8 and says so (#425), so a source in another
	// charset keeps its layout but not its bytes. Out of scope for #876.
	"testdata/encoding/ansel-lf.ged":                    "non-UTF-8 source; payload transcoded",
	"testdata/encoding/ansel-trailing-combining.ged":    "non-UTF-8 source; payload transcoded",
	"testdata/encoding/ansi-cp1252-c1-range.ged":        "non-UTF-8 source; payload transcoded",
	"testdata/encoding/ansi-cp1252-ftm17.ged":           "non-UTF-8 source; payload transcoded",
	"testdata/encoding/brotherskeeper-5.5.1-w1250.ged":  "non-UTF-8 source; payload transcoded",
	"testdata/encoding/familyhistorian-5.5.1-w1250.ged": "non-UTF-8 source; payload transcoded",
	"testdata/encoding/geneanet-5.5.1-w1250.ged":        "non-UTF-8 source; payload transcoded",
	"testdata/encoding/ibmpc-cp437-broskeep.ged":        "non-UTF-8 source; payload transcoded",
	"testdata/encoding/utf16be.ged":                     "non-UTF-8 source; payload transcoded",
	"testdata/encoding/utf16le.ged":                     "non-UTF-8 source; payload transcoded",
	"testdata/gedcom-5.5/torture-test/TGC551.ged":       "non-UTF-8 source; payload transcoded",
	"testdata/gedcom-5.5/torture-test/TGC551LF.ged":     "non-UTF-8 source; payload transcoded",
	"testdata/gedcom-5.5/torture-test/TGC55C.ged":       "non-UTF-8 source; payload transcoded",
	"testdata/gedcom-5.5/torture-test/TGC55CLF.ged":     "non-UTF-8 source; payload transcoded",
	"testdata/gedcom7code/5/char_utf16be-1.ged":         "non-UTF-8 source; payload transcoded",
	"testdata/gedcom7code/5/char_utf16be-2.ged":         "non-UTF-8 source; payload transcoded",
	"testdata/gedcom7code/5/char_utf16le-1.ged":         "non-UTF-8 source; payload transcoded",
	"testdata/gedcom7code/5/char_utf16le-2.ged":         "non-UTF-8 source; payload transcoded",

	// ASCII bytes under a declaration other than UTF-8. Every byte is kept
	// but the CHAR line, which the encoder always writes as UTF-8 (#425):
	// it cannot know, when it writes the header, that the rest will be ASCII.
	"testdata/edge-cases/relationships-complex.ged":     "declares CHAR ANSI; written as CHAR UTF-8",
	"testdata/edge-cases/vendor-customtags-torture.ged": "declares CHAR ANSEL; written as CHAR UTF-8",
	"testdata/edge-cases/vendor-familyorigins5.ged":     "declares CHAR ANSI; written as CHAR UTF-8",
	"testdata/edge-cases/vendor-myroots-palmos.ged":     "declares CHAR ANSEL; written as CHAR UTF-8",
	"testdata/edge-cases/vendor-tmg12.ged":              "declares CHAR IBMPC; written as CHAR UTF-8",
	"testdata/edge-cases/xref-case.ged":                 "declares CHAR ASCII; written as CHAR UTF-8",
	"testdata/encoding/ibm-windows-easytree.ged":        "declares CHAR IBM WINDOWS; written as CHAR UTF-8",
	"testdata/gedcom-5.5/minimal.ged":                   "declares CHAR ASCII; written as CHAR UTF-8",
	"testdata/gedcom-5.5/royal92.ged":                   "declares CHAR ANSEL; written as CHAR UTF-8",
	"testdata/gedcom7code/5/char_ascii_1.ged":           "declares CHAR ASCII; written as CHAR UTF-8",
	"testdata/gedcom7code/5/char_ascii_2.ged":           "declares CHAR LATIN1; written as CHAR UTF-8",
	"testdata/gedcom7code/5/char_utf8-2.ged":            "declares CHAR UNICODE; written as CHAR UTF-8",
	"testdata/gedcom7code/5/enum-ext.ged":               "declares CHAR ASCII; written as CHAR UTF-8",
	"testdata/gedcom7code/5/filename-1.ged":             "declares CHAR ASCII; written as CHAR UTF-8",
	"testdata/gedcom7code/5/obsolete-1.ged":             "declares CHAR ASCII; written as CHAR UTF-8",
	"testdata/gedcom7code/5/rela_1.ged":                 "declares CHAR ASCII; written as CHAR UTF-8",
	"testdata/gedcom7code/5/xref-case.ged":              "declares CHAR ASCII; written as CHAR UTF-8",

	// The encoder supplies what a header must have when the source has none,
	// or an empty one: lines the source never had.
	"testdata/malformed/missing-header.ged":          "encoder writes 0 HEAD for a headerless file",
	"testdata/edge-cases/bare-header-geo-coords.ged": "encoder writes GEDC.VERS for an empty header",
	"testdata/gedcom7code/5/tiny-1.ged":              "encoder writes GEDC.VERS for an empty header",

	// Lenient recovery clamps an over-jumped level. The clamped line no
	// longer reads as its source text, so it is written as changed.
	"testdata/malformed/invalid-level.ged":          "over-jumped level is clamped by recovery",
	"testdata/malformed/level-jump-skip.ged":        "over-jumped level is clamped by recovery",
	"testdata/malformed/level-jump-subordinate.ged": "over-jumped level is clamped by recovery",
}

func TestLayoutByteRoundTrip(t *testing.T) {
	for _, path := range gedcomFixtures(t) {
		t.Run(path, func(t *testing.T) {
			orig, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			doc, err := decoder.DecodeWithOptions(bytes.NewReader(orig), &decoder.DecodeOptions{PreserveLayout: true})
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			var buf bytes.Buffer
			if err := encoder.EncodeWithOptions(&buf, doc, &encoder.EncodeOptions{PreserveLayout: true}); err != nil {
				t.Fatalf("encode: %v", err)
			}

			match := bytes.Equal(orig, buf.Bytes())
			reason, known := layoutKnownBad[path]
			switch {
			case match && known:
				t.Errorf("now round-trips byte for byte; remove %q from layoutKnownBad (was: %s)", path, reason)
			case !match && !known:
				t.Errorf("does not round-trip byte for byte:\n%s", firstDiff(string(orig), buf.String()))
			}
		})
	}
}
