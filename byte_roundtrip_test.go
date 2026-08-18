package gedcomgo

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Byte-level round-trip: compare the bytes we write against the bytes we read,
// rather than comparing two decoded documents.
//
// TestRoundTrip and gedcomtesting.AssertRoundTrip compare decode(input) against
// decode(encode(decode(input))). That is a fixpoint check: anything both decode
// passes lose identically is invisible to it. Header collapse (#429), CONC
// leading spaces (#426) and charset mismatch (#425) all survived that check for
// exactly this reason — the loss happens before the comparison, symmetrically.
//
// Comparing against the original bytes has no such blind spot. The cost is that
// it also sees legitimate normalization, so the small set of differences we
// accept is enumerated in normalizeGEDCOM below and nowhere else.
//
// Both lists here are self-cleaning: a fixture that starts passing fails the
// test asking to be promoted, and a fixture that starts failing fails the test
// as a regression. Fixing an issue deletes entries; nothing goes stale quietly.

// headerByteIdentical lists the fixtures whose HEAD block survives a round-trip
// byte for byte. It is an allow-list rather than a deny-list because the
// encoder rebuilds HEAD from four scalar fields and discards Header.Tags
// (#429), so almost nothing survives: 13 of 96 at the time of writing, and all
// 13 have headers small enough to fit in those four fields.
//
// When #429 lands this list becomes every decodable fixture, and the assertion
// below inverts into a plain "all must match".
var headerByteIdentical = map[string]bool{
	"testdata/gedcom-7.0/familysearch-examples/obje-1.ged":            true,
	"testdata/gedcom-7.0/familysearch-examples/remarriage2.ged":       true,
	"testdata/gedcom-7.0/familysearch-examples/same-sex-marriage.ged": true,
	"testdata/gedcom-7.0/familysearch-examples/voidptr.ged":           true,
	"testdata/gedcom-7.0/minimal70.ged":                               true,
	"testdata/gedcom-7.0/remarriage1.ged":                             true,
	"testdata/malformed/circular-reference.ged":                       true,
	"testdata/malformed/duplicate-xref.ged":                           true,
	"testdata/malformed/invalid-level.ged":                            true,
	"testdata/malformed/invalid-xref.ged":                             true,
	"testdata/malformed/level-jump-skip.ged":                          true,
	"testdata/malformed/level-jump-subordinate.ged":                   true,
	"testdata/malformed/missing-xref.ged":                             true,
}

// bodyKnownBad lists fixtures whose records do not survive a round-trip, with
// the defect responsible. Every entry is a live bug; none is an accepted
// difference. Delete an entry when its issue is fixed.
var bodyKnownBad = map[string]string{
	// #426: value parsing trims every leading space, but GEDCOM's delimiter is
	// exactly one. The rest is payload — a CONC word separator, or, in
	// "1 NAME  /Mac Imair/", an empty given name.
	"testdata/edge-cases/cont-conc.ged":                 "#426 CONC leading space",
	"testdata/edge-cases/ftm-conc-test.ged":             "#426 CONC leading space",
	"testdata/edge-cases/ftm-general.ged":               "#426 CONC leading space",
	"testdata/edge-cases/vendor-customtags-torture.ged": "#426 CONC leading space",
	"testdata/edge-cases/legacy10-2025-export.ged":      "#426 NAME with empty given name",
	"testdata/edge-cases/vendor-familyorigins5.ged":     "#426 NAME with empty given name",
	"testdata/gedcom-5.5/royal92.ged":                   "#426 DATE leading space",

	// #404: a value on a level-0 line is dropped by writeRecord.
	"testdata/edge-cases/rootsmagic-2026-export.ged": "#404 level-0 record value",
	"testdata/edge-cases/vendor-legacy.ged":          "#404 level-0 record value",

	// #425: output is UTF-8 but declares the source charset, so re-reading it
	// applies the decoding a second time.
	"testdata/encoding/ansel-lf.ged":                "#425 ANSEL declared, UTF-8 written",
	"testdata/encoding/ansi-cp1252-ftm17.ged":       "#425 CP1252 declared, UTF-8 written",
	"testdata/encoding/utf16be.ged":                 "#425 UTF-16 source, records lost",
	"testdata/encoding/utf16le.ged":                 "#425 UTF-16 source, records lost",
	"testdata/gedcom-5.5/torture-test/TGC551.ged":   "#425 ANSEL declared, UTF-8 written",
	"testdata/gedcom-5.5/torture-test/TGC551LF.ged": "#425 ANSEL declared, UTF-8 written",
	"testdata/gedcom-5.5/torture-test/TGC55C.ged":   "#425 ANSEL declared, UTF-8 written",
	"testdata/gedcom-5.5/torture-test/TGC55CLF.ged": "#425 ANSEL declared, UTF-8 written",

	// Lenient recovery rewrites the line rather than preserving it. Arguably
	// correct — the input is malformed — but it is a real byte difference and
	// belongs on the list rather than in the accepted-normalization set, so
	// that a decision to keep it is made explicitly.
	"testdata/edge-cases/indented-lines.ged": "leading whitespace on the line is stripped by recovery",
	"testdata/edge-cases/mhftb8-export.ged":  "double space after level is normalized by recovery",
	"testdata/malformed/missing-header.ged":  "synthesized header shifts the record block",
}

// undecodable lists fixtures that do not survive decode or encode at all, so
// there are no bytes to compare. Kept explicit so the count cannot drift.
var undecodable = map[string]string{
	"testdata/edge-cases/xref-case.ged":          "xref contains a space",
	"testdata/malformed/blank-lines.ged":         "empty line",
	"testdata/malformed/level-over-99.ged":       "level 100 exceeds maximum",
	"testdata/malformed/unterminated-xref.ged":   "xref missing closing @",
	"testdata/encoding/ibmpc-cp437-broskeep.ged": "invalid byte for the declared encoding",
}

// valuelessLine matches a line carrying no value: "1 CONT", "0 @I1@ INDI".
// GEDCOM makes the delimiter space optional when the value is empty, so
// "1 CONT " and "1 CONT" denote the same thing and the encoder writes the
// shorter form. This is the only whitespace difference we accept; a space that
// is followed by anything is payload, and its loss is #426.
var valuelessLine = regexp.MustCompile(`^\d+ (@[^@]+@ )?[A-Za-z_][A-Za-z0-9_]* +$`)

// normalizeGEDCOM applies the differences we accept, and only those:
// a UTF-8 BOM, the choice of line terminator, a trailing delimiter space on a
// valueless line, and a trailing blank line at end of file.
func normalizeGEDCOM(b []byte) string {
	b = bytes.TrimPrefix(b, []byte("\xEF\xBB\xBF"))
	s := string(b)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		if valuelessLine.MatchString(l) {
			lines[i] = strings.TrimRight(l, " ")
		}
	}
	return strings.Join(lines, "\n")
}

// splitHeader returns the HEAD block and everything from the first non-HEAD
// level-0 line onward, so the two can be asserted independently. Until #429 is
// fixed the header comparison would otherwise mask every body difference.
func splitHeader(s string) (header, body string) {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if i == 0 {
			continue
		}
		if strings.HasPrefix(l, "0 ") && !strings.HasPrefix(l, "0 HEAD") {
			return strings.Join(lines[:i], "\n"), strings.Join(lines[i:], "\n")
		}
	}
	return s, ""
}

func gedcomFixtures(t *testing.T) []string {
	t.Helper()
	var files []string
	err := filepath.Walk("testdata", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if !strings.EqualFold(filepath.Ext(path), ".ged") {
			return nil
		}
		// The 46 MiB scale fixture would be decoded and encoded here; see
		// decoder.TestScaleFixture for why it is kept out of ordinary runs.
		if filepath.Base(path) == "longsword.ged" {
			return nil
		}
		files = append(files, filepath.ToSlash(path))
		return nil
	})
	if err != nil {
		t.Fatalf("walk testdata: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no fixtures found under testdata")
	}
	return files
}

func TestByteRoundTrip(t *testing.T) {
	for _, path := range gedcomFixtures(t) {
		t.Run(path, func(t *testing.T) {
			orig, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}

			doc, decErr := Decode(bytes.NewReader(orig))
			var buf bytes.Buffer
			encErr := error(nil)
			if decErr == nil {
				encErr = Encode(&buf, doc)
			}

			reason, expectedBad := undecodable[path]
			switch {
			case decErr == nil && encErr == nil && expectedBad:
				t.Fatalf("fixture now round-trips but is listed as undecodable (%s); remove it from undecodable", reason)
			case decErr != nil && !expectedBad:
				t.Fatalf("decode failed and the fixture is not listed as undecodable: %v", decErr)
			case encErr != nil && !expectedBad:
				t.Fatalf("encode failed and the fixture is not listed as undecodable: %v", encErr)
			case expectedBad:
				return
			}

			wantHeader, wantBody := splitHeader(normalizeGEDCOM(orig))
			gotHeader, gotBody := splitHeader(normalizeGEDCOM(buf.Bytes()))

			t.Run("header", func(t *testing.T) {
				match := wantHeader == gotHeader
				switch {
				case match && !headerByteIdentical[path]:
					t.Errorf("header now round-trips byte for byte; add %q to headerByteIdentical", path)
				case !match && headerByteIdentical[path]:
					t.Errorf("header no longer round-trips:\n%s", firstDiff(wantHeader, gotHeader))
				}
			})

			t.Run("body", func(t *testing.T) {
				match := wantBody == gotBody
				reason, known := bodyKnownBad[path]
				switch {
				case match && known:
					t.Errorf("body now round-trips; remove %q from bodyKnownBad (was: %s)", path, reason)
				case !match && !known:
					t.Errorf("body does not round-trip:\n%s", firstDiff(wantBody, gotBody))
				}
			})
		})
	}
}

// firstDiff reports the first differing line with a little context, which is
// far more useful than a full dump of two multi-thousand-line documents.
func firstDiff(want, got string) string {
	w := strings.Split(want, "\n")
	g := strings.Split(got, "\n")
	for i := 0; i < len(w) || i < len(g); i++ {
		var a, b string
		if i < len(w) {
			a = w[i]
		}
		if i < len(g) {
			b = g[i]
		}
		if a != b {
			return fmt.Sprintf("  line %d\n  want: %q\n  got:  %q", i+1, a, b)
		}
	}
	return fmt.Sprintf("  (no differing line; %d lines vs %d)", len(w), len(g))
}
