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
// The lists here are self-cleaning: a fixture that starts passing fails the
// test asking to be promoted, and a fixture that starts failing fails the test
// as a regression. Fixing an issue deletes entries; nothing goes stale quietly.

// headerKnownBad lists the fixtures whose HEAD block still does not survive a
// round-trip byte for byte, with the defect responsible.
//
// This was an allow-list of the 13 fixtures out of 96 that happened to survive,
// because the encoder rebuilt HEAD from four scalar fields and discarded
// Header.Tags. #429 fixed that, so it inverted into a deny-list as the comment
// here promised: every decodable fixture's header must now match, and anything
// that does not is named with its cause.
//
// An allow-list could not see a fixture that was failing and unlisted; a
// deny-list fails on exactly that case.
//
// Unlike bodyKnownBad, not every entry here is a bug. The first group is a
// limit of byte comparison itself; the second is deliberate behaviour whose
// desirability is an open question, recorded so the choice is visible rather
// than assumed.
var headerKnownBad = map[string]string{
	// The source is not UTF-8, so its header payload bytes are ANSEL or UTF-16
	// while the output's are UTF-8 (TGC's "1 COPR © 1997..." is the ANSEL
	// copyright sign; the UTF-16 files differ from the BOM onward). The CHAR
	// declaration is normalized, but the text it describes cannot be, so a
	// byte-for-byte comparison against the original can never match. These same
	// fixtures are on bodyKnownBad for the same reason.
	"testdata/encoding/utf16be.ged":                 "non-UTF-8 source; payload transcoded",
	"testdata/encoding/utf16le.ged":                 "non-UTF-8 source; payload transcoded",
	"testdata/gedcom-5.5/torture-test/TGC551.ged":   "non-UTF-8 source; payload transcoded",
	"testdata/gedcom-5.5/torture-test/TGC551LF.ged": "non-UTF-8 source; payload transcoded",
	"testdata/gedcom-5.5/torture-test/TGC55C.ged":   "non-UTF-8 source; payload transcoded",
	"testdata/gedcom-5.5/torture-test/TGC55CLF.ged": "non-UTF-8 source; payload transcoded",

	// The encoder synthesizes what a header must have when the source has no
	// header at all, or an empty one: "0 HEAD" for the first, "1 GEDC / 2 VERS"
	// from the detected version for the second. Both produce a more valid file
	// than the input, and both are additions the input did not contain. The
	// allow-list could not see either, because neither fixture was on it.
	"testdata/malformed/missing-header.ged":          "encoder writes 0 HEAD for a headerless file",
	"testdata/edge-cases/bare-header-geo-coords.ged": "encoder writes GEDC.VERS for an empty header",
}

// bodyKnownBad lists fixtures whose records do not survive a round-trip, with
// the defect responsible. Every entry is a live bug; none is an accepted
// difference. Delete an entry when its issue is fixed.
var bodyKnownBad = map[string]string{
	// #404: a value on a level-0 line is dropped by writeRecord.
	// vendor-customtags-torture.ged was attributed to #426 until that fix
	// landed and uncovered this second defect underneath it ("0 _ROOT Root
	// element"); it stays on the list under its remaining cause.
	"testdata/edge-cases/rootsmagic-2026-export.ged":    "#404 level-0 record value",
	"testdata/edge-cases/vendor-legacy.ged":             "#404 level-0 record value",
	"testdata/edge-cases/vendor-customtags-torture.ged": "#404 level-0 record value",

	// Not a bug: these sources are not UTF-8, so their record bytes are ANSEL,
	// CP1252 or UTF-16 in the original and UTF-8 in the output. Byte comparison
	// against an untranscoded original cannot match, whatever the encoder does.
	//
	// They were attributed to #425 while the output also declared the source
	// charset -- which made them genuinely unreadable, not merely incomparable.
	// That is fixed; these fixtures now decode, re-encode and re-decode
	// cleanly, and TestRoundTrip covers them. Only the byte-level comparison
	// still cannot see it.
	"testdata/encoding/ansel-lf.ged":                "non-UTF-8 source; payload transcoded",
	"testdata/encoding/ansi-cp1252-ftm17.ged":       "non-UTF-8 source; payload transcoded",
	"testdata/encoding/utf16be.ged":                 "non-UTF-8 source; payload transcoded",
	"testdata/encoding/utf16le.ged":                 "non-UTF-8 source; payload transcoded",
	"testdata/gedcom-5.5/torture-test/TGC551.ged":   "non-UTF-8 source; payload transcoded",
	"testdata/gedcom-5.5/torture-test/TGC551LF.ged": "non-UTF-8 source; payload transcoded",
	"testdata/gedcom-5.5/torture-test/TGC55C.ged":   "non-UTF-8 source; payload transcoded",
	"testdata/gedcom-5.5/torture-test/TGC55CLF.ged": "non-UTF-8 source; payload transcoded",

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

// charLine matches the level-1 CHAR declaration.
var charLine = regexp.MustCompile(`^1 CHAR .+$`)

// normalizeGEDCOM applies the differences we accept, and only those:
// a UTF-8 BOM, the choice of line terminator, a trailing delimiter space on a
// valueless line, a trailing blank line at end of file, and the charset the
// CHAR line declares.
//
// CHAR is accepted because the encoder writes UTF-8 whatever it read and now
// declares that (#425) -- a file that says ANSEL over UTF-8 bytes is one this
// library cannot re-read. The declaration is normalized on both sides rather
// than skipped, so a missing CHAR line is still a difference; only its value is
// exempt. compatibility.md records the same normalization.
//
// Note this does not make a non-UTF-8 fixture comparable: its *payload* bytes
// are still ANSEL or UTF-16 in the original and UTF-8 in the output. Those
// fixtures are listed in headerKnownBad and bodyKnownBad with that as the
// reason.
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
		if charLine.MatchString(lines[i]) {
			lines[i] = "1 CHAR <normalized>"
		}
	}
	return strings.Join(lines, "\n")
}

// splitHeader returns the HEAD block and everything from the first non-HEAD
// level-0 line onward, so the two can be asserted independently. The split
// existed because header collapse (#429) would otherwise have masked every body
// difference; it stays because the two have genuinely different failure modes --
// a header can be synthesized where none existed, which no record can.
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
				if wantHeader != gotHeader {
					reason, known := headerKnownBad[path]
					if !known {
						t.Errorf("header does not round-trip:\n%s", firstDiff(wantHeader, gotHeader))
						return
					}
					_ = reason
					return
				}
				if reason, known := headerKnownBad[path]; known {
					t.Errorf("header now round-trips; remove %q from headerKnownBad (was: %s)", path, reason)
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
