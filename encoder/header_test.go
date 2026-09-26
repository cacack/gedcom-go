package encoder

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// encodeFixtureHeader decodes a fixture, re-encodes it, and returns the HEAD
// block of the output.
func encodeFixtureHeader(t *testing.T, path string, opts *EncodeOptions) string {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	doc, err := decoder.Decode(f)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}

	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, doc, opts); err != nil {
		t.Fatalf("encode %s: %v", path, err)
	}

	out := buf.String()
	if i := strings.Index(out, "\n0 @"); i > 0 {
		out = out[:i]
	}
	return out
}

// TestHeaderStructurePreserved covers the substructures the encoder used to
// discard when it rebuilt HEAD from four scalar fields (#429). SCHMA and SUBM
// are correctness failures rather than cosmetic loss: 7.0 output that emits
// extension tags without declaring their URIs no longer defines its own
// vocabulary, and 5.5.1 without SUBM is invalid.
func TestHeaderStructurePreserved(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		want    []string
	}{
		{
			name:    "7.0 keeps SCHMA and its TAG declarations",
			fixture: "../testdata/gedcom-7.0/maximal70.ged",
			want:    []string{"1 SCHMA", "2 TAG _SKYPEID", "2 TAG _JABBERID"},
		},
		{
			name:    "5.5.1 keeps the mandatory SUBM pointer",
			fixture: "../testdata/gedcom-5.5.1/comprehensive.ged",
			want:    []string{"1 SUBM @SUBM1@"},
		},
		{
			name:    "the SOUR subtree survives",
			fixture: "../testdata/gedcom-5.5.1/comprehensive.ged",
			want:    []string{"1 SOUR ", "2 VERS ", "2 NAME ", "2 CORP "},
		},
		{
			name:    "DEST and DATE survive",
			fixture: "../testdata/gedcom-5.5/royal92.ged",
			want:    []string{"1 DEST ", "1 DATE "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			head := encodeFixtureHeader(t, tt.fixture, DefaultOptions())
			for _, want := range tt.want {
				if !strings.Contains(head, want) {
					t.Errorf("header lost %q:\n%s", want, head)
				}
			}
		})
	}
}

// TestHeaderCharDeclaresWhatWasWritten covers #425. Encode converts nothing on
// the way out, so echoing the source charset produced a file contradicting its
// own bytes -- unreadable for ANSEL, silently double-encoded for CP1252.
func TestHeaderCharDeclaresWhatWasWritten(t *testing.T) {
	for _, fixture := range []string{
		"../testdata/encoding/ansel-lf.ged",
		"../testdata/encoding/ansi-cp1252-ftm17.ged",
		"../testdata/encoding/utf16le.ged",
	} {
		t.Run(fixture, func(t *testing.T) {
			head := encodeFixtureHeader(t, fixture, DefaultOptions())

			if !strings.Contains(head, "1 CHAR UTF-8") {
				t.Errorf("header does not declare UTF-8:\n%s", head)
			}
			for _, stale := range []string{"1 CHAR ANSEL", "1 CHAR ANSI", "1 CHAR UNICODE"} {
				if strings.Contains(head, stale) {
					t.Errorf("header still declares the source charset %q:\n%s", stale, head)
				}
			}
		})
	}
}

// TestHeaderVersionTagPreservedVerbatim guards a normalization that
// reconstruction performed silently: 555SAMPLE.GED declares 5.5.5, a version
// this library does not model, and the typed Header.Version reads 5.5.1. With
// the raw tag written through, the file keeps what it said.
func TestHeaderVersionTagPreservedVerbatim(t *testing.T) {
	head := encodeFixtureHeader(t, "../testdata/gedcom-5.5/555SAMPLE.GED", DefaultOptions())

	if !strings.Contains(head, "2 VERS 5.5.5") {
		t.Errorf("GEDC.VERS was rewritten; expected the original 5.5.5:\n%s", head)
	}
}

// TestHeaderTargetVersionRewritesOnlyGEDC pins the parent-context rule. VERS is
// not unique in a header: it names the specification version under GEDC and the
// source system's own version under SOUR. comprehensive.ged carries both
// ("1 SOUR FamilyTreeMaker / 2 VERS 16.0"), so matching on the tag name alone
// would relabel Family Tree Maker as version 7.0.
func TestHeaderTargetVersionRewritesOnlyGEDC(t *testing.T) {
	opts := DefaultOptions()
	opts.TargetVersion = gedcom.Version70

	head := encodeFixtureHeader(t, "../testdata/gedcom-5.5.1/comprehensive.ged", opts)

	if !strings.Contains(head, "2 VERS 7.0") {
		t.Errorf("GEDC.VERS was not retargeted:\n%s", head)
	}
	if !strings.Contains(head, "2 VERS 16.0") {
		t.Errorf("SOUR.VERS was rewritten; the source system's version is not a GEDCOM version:\n%s", head)
	}
	if strings.Contains(head, "2 VERS 5.5.1") {
		t.Errorf("the source GEDCOM version survived a retarget:\n%s", head)
	}
}

// TestHeaderTargetVersionWithoutGEDCInSource covers the gap the parent-context
// rule leaves: royal92.ged declares no GEDC block at all, so there is no VERS
// to override. Preserving that absence is right for a plain re-encode -- the
// round-trip table depends on it -- but a caller who explicitly retargets must
// get a document that states the version they asked for.
func TestHeaderTargetVersionWithoutGEDCInSource(t *testing.T) {
	const fixture = "../testdata/gedcom-5.5/royal92.ged"

	t.Run("absence preserved without a target", func(t *testing.T) {
		head := encodeFixtureHeader(t, fixture, DefaultOptions())
		if strings.Contains(head, "1 GEDC") {
			t.Errorf("a plain re-encode invented a GEDC block the source did not have:\n%s", head)
		}
	})

	t.Run("target version declared when asked for", func(t *testing.T) {
		opts := DefaultOptions()
		opts.TargetVersion = gedcom.Version551

		head := encodeFixtureHeader(t, fixture, opts)
		if !strings.Contains(head, "1 GEDC") || !strings.Contains(head, "2 VERS 5.5.1") {
			t.Errorf("retarget produced no version declaration:\n%s", head)
		}
	})
}

// TestHeaderFieldsPathWhenNoTags covers the hand-built document: with no raw
// tags to preserve, the header is still reconstructed from the typed fields --
// and CHAR still declares what was written rather than what was set.
func TestHeaderFieldsPathWhenNoTags(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version:      gedcom.Version551,
			Encoding:     gedcom.EncodingANSEL,
			SourceSystem: "TestSystem",
			Language:     "English",
		},
	}

	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("encode: %v", err)
	}

	out := buf.String()
	for _, want := range []string{"0 HEAD", "1 GEDC", "2 VERS 5.5.1", "1 CHAR UTF-8", "1 SOUR TestSystem", "1 LANG English"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "1 CHAR ANSEL") {
		t.Errorf("declared ANSEL while writing UTF-8:\n%s", out)
	}
}

// TestHeaderTagsWriteErrorsPropagate covers the error paths on the tag-writing
// header path. Each write can fail, and a header that silently stops half-way
// through would leave a truncated HEAD block behind with no error to show for
// it. failAfter is tuned to fail inside each of the three writes the synthesis
// path makes.
func TestHeaderTagsWriteErrorsPropagate(t *testing.T) {
	// A header with a GEDC carrying no VERS of its own, so a retarget has to
	// synthesize one, and a second header with no GEDC at all.
	withBareGEDC := &gedcom.Document{
		Header: &gedcom.Header{
			Tags: []*gedcom.Tag{
				{Level: 1, Tag: "CHAR", Value: "ANSEL"},
				{Level: 1, Tag: "GEDC"},
			},
		},
	}
	withoutGEDC := &gedcom.Document{
		Header: &gedcom.Header{
			Tags: []*gedcom.Tag{{Level: 1, Tag: "SOUR", Value: "PAF 2.2"}},
		},
	}

	tests := []struct {
		name      string
		doc       *gedcom.Document
		failAfter int
	}{
		{"overridden CHAR tag", withBareGEDC, 1},
		{"GEDC tag", withBareGEDC, 2},
		{"synthesized VERS inside GEDC", withBareGEDC, 3},
		{"synthesized GEDC when the source has none", withoutGEDC, 2},
		{"synthesized VERS when the source has no GEDC", withoutGEDC, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := DefaultOptions()
			opts.TargetVersion = gedcom.Version551

			err := EncodeWithOptions(&failWriter{failAfter: tt.failAfter}, tt.doc, opts)
			if err == nil {
				t.Error("write failure was swallowed; a truncated header must report an error")
			}
		})
	}
}

// TestHeaderTagsSkipsNilTag covers the nil guard on the tag path. A hand-built
// document can hold a nil where a decoded one never does; it must be skipped
// rather than panic, and it must not end the GEDC structure it sits inside --
// otherwise a retarget would synthesize a second VERS beside the real one.
func TestHeaderTagsSkipsNilTag(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Tags: []*gedcom.Tag{
				{Level: 1, Tag: "GEDC"},
				nil,
				{Level: 2, Tag: "VERS", Value: "5.5"},
			},
		},
	}

	opts := DefaultOptions()
	opts.TargetVersion = gedcom.Version551

	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, doc, opts); err != nil {
		t.Fatalf("encode: %v", err)
	}

	out := buf.String()
	if n := strings.Count(out, "2 VERS"); n != 1 {
		t.Errorf("expected exactly one VERS, got %d:\n%s", n, out)
	}
	if !strings.Contains(out, "2 VERS 5.5.1") {
		t.Errorf("the existing VERS was not retargeted:\n%s", out)
	}
}

// TestHeaderRetargetWhenGEDCIsNotLast covers the scan leaving the GEDC subtree
// at the next level-1 structure rather than at the end of the header. A GEDC
// carrying children but no VERS, followed by another structure, still needs its
// version supplied -- and the VERS must land inside GEDC, not after CHAR.
func TestHeaderRetargetWhenGEDCIsNotLast(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Tags: []*gedcom.Tag{
				{Level: 1, Tag: "GEDC"},
				{Level: 2, Tag: "FORM", Value: "LINEAGE-LINKED"},
				{Level: 1, Tag: "CHAR", Value: "ANSEL"},
			},
		},
	}

	opts := DefaultOptions()
	opts.TargetVersion = gedcom.Version551

	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, doc, opts); err != nil {
		t.Fatalf("encode: %v", err)
	}

	out := buf.String()
	if n := strings.Count(out, "2 VERS 5.5.1"); n != 1 {
		t.Errorf("expected exactly one synthesized VERS, got %d:\n%s", n, out)
	}
	if strings.Index(out, "2 VERS 5.5.1") > strings.Index(out, "1 CHAR") {
		t.Errorf("VERS landed outside the GEDC structure it belongs to:\n%s", out)
	}
}

// TestHeaderEncodeDoesNotMutateDocument guards the substitution mechanism: the
// overrides write a copy of the tag, because encoding a document must not edit
// it. A caller that encodes twice, or encodes and then inspects, must see what
// it decoded.
func TestHeaderEncodeDoesNotMutateDocument(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version:  gedcom.Version55,
			Encoding: gedcom.EncodingANSEL,
			Tags: []*gedcom.Tag{
				{Level: 1, Tag: "CHAR", Value: "ANSEL"},
				{Level: 1, Tag: "GEDC"},
				{Level: 2, Tag: "VERS", Value: "5.5"},
			},
		},
	}

	opts := DefaultOptions()
	opts.TargetVersion = gedcom.Version70

	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, doc, opts); err != nil {
		t.Fatalf("encode: %v", err)
	}

	if got := doc.Header.Tags[0].Value; got != "ANSEL" {
		t.Errorf("CHAR tag mutated in the source document: %q", got)
	}
	if got := doc.Header.Tags[2].Value; got != "5.5" {
		t.Errorf("GEDC.VERS tag mutated in the source document: %q", got)
	}
}

// assertLineOrder checks that each line appears in out, in the order given.
// Containment alone cannot catch a grammar violation: every field being present
// is exactly what the encoder did before it emitted them in a legal order.
func assertLineOrder(t *testing.T, out string, lines ...string) {
	t.Helper()

	prev := -1
	for _, line := range lines {
		i := strings.Index(out, line)
		if i < 0 {
			t.Errorf("output missing %q:\n%s", line, out)
			continue
		}
		if i < prev {
			t.Errorf("%q appears out of grammar order:\n%s", line, out)
		}
		prev = i
	}
}

// TestHeaderFieldsSubmitterAndOrder covers the hand-built path (issue #503).
// Header.Submitter had no writer at all there, so a document assembled in
// memory could set the field the validator requires for 5.5/5.5.1 and still
// encode to a header without it. Order is asserted alongside it because the
// same path emitted its other fields in an order no GEDCOM grammar allows.
func TestHeaderFieldsSubmitterAndOrder(t *testing.T) {
	t.Run("5.5.1 leads with SOUR and declares GEDC late", func(t *testing.T) {
		doc := &gedcom.Document{
			Header: &gedcom.Header{
				Version:      gedcom.Version551,
				Encoding:     gedcom.EncodingUTF8,
				SourceSystem: "TestSystem",
				Submitter:    "@U1@",
				Language:     "English",
			},
		}

		var buf bytes.Buffer
		if err := Encode(&buf, doc); err != nil {
			t.Fatalf("encode: %v", err)
		}

		assertLineOrder(t, buf.String(),
			"0 HEAD",
			"1 SOUR TestSystem",
			"1 SUBM @U1@",
			"1 GEDC",
			"2 VERS 5.5.1",
			"1 CHAR UTF-8",
			"1 LANG English",
		)
	})

	t.Run("7.0 leads with GEDC", func(t *testing.T) {
		doc := &gedcom.Document{
			Header: &gedcom.Header{
				Version:      gedcom.Version70,
				SourceSystem: "TestSystem",
				Submitter:    "@U1@",
				Language:     "en",
			},
		}

		var buf bytes.Buffer
		if err := Encode(&buf, doc); err != nil {
			t.Fatalf("encode: %v", err)
		}

		out := buf.String()
		assertLineOrder(t, out,
			"0 HEAD",
			"1 GEDC",
			"2 VERS 7.0",
			"1 SOUR TestSystem",
			"1 SUBM @U1@",
			"1 LANG en",
		)
		// Encoding is unset, so the CHAR guard still suppresses a tag 7.0
		// removed.
		if strings.Contains(out, "1 CHAR") {
			t.Errorf("7.0 header declared a CHAR the document never had:\n%s", out)
		}
	})

	t.Run("no submitter emits no SUBM", func(t *testing.T) {
		doc := &gedcom.Document{
			Header: &gedcom.Header{
				Version:      gedcom.Version551,
				SourceSystem: "TestSystem",
			},
		}

		var buf bytes.Buffer
		if err := Encode(&buf, doc); err != nil {
			t.Fatalf("encode: %v", err)
		}

		if strings.Contains(buf.String(), "SUBM") {
			t.Errorf("synthesized a SUBM the header never had:\n%s", buf.String())
		}
	})
}

// TestHeaderTagsSubmitterRoundTrip pins the decoded path for the same field.
// Header.Tags is authoritative when encoding, so a header SUBM survives
// byte-identically through writeHeaderTags whether or not the typed
// Header.Submitter field is populated -- this test holds either way, and is
// here so #503's fix to the typed field cannot be mistaken for the reason a
// decoded document round-trips.
func TestHeaderTagsSubmitterRoundTrip(t *testing.T) {
	const input = "0 HEAD\n" +
		"1 SOUR TestApp\n" +
		"1 SUBM @U1@\n" +
		"1 GEDC\n" +
		"2 VERS 5.5.1\n" +
		"1 CHAR UTF-8\n" +
		"0 @U1@ SUBM\n" +
		"1 NAME Tester\n" +
		"0 TRLR\n"

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("encode: %v", err)
	}

	// Compare the HEAD block alone: the records that follow are a different
	// path's concern.
	head := func(s string) string {
		if i := strings.Index(s, "\n0 @"); i > 0 {
			return s[:i+1]
		}
		return s
	}

	if got, want := head(buf.String()), head(input); got != want {
		t.Errorf("header did not round-trip byte-identically:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// TestHeaderFieldsDate covers the hand-built path for Header.Date (#495): the
// field is written verbatim as "1 DATE", between SOUR and SUBM in both the
// 5.5.1 and 7.0 grammars, and omitted when empty.
func TestHeaderFieldsDate(t *testing.T) {
	encode := func(t *testing.T, h *gedcom.Header) string {
		t.Helper()
		var buf bytes.Buffer
		if err := Encode(&buf, &gedcom.Document{Header: h}); err != nil {
			t.Fatalf("encode: %v", err)
		}
		return buf.String()
	}

	t.Run("5.5.1 places DATE after SOUR, before SUBM", func(t *testing.T) {
		out := encode(t, &gedcom.Header{
			Version:      gedcom.Version551,
			Encoding:     gedcom.EncodingUTF8,
			SourceSystem: "TestSystem",
			Date:         "7 AUG 2026",
			Submitter:    "@U1@",
		})
		assertLineOrder(t, out,
			"0 HEAD",
			"1 SOUR TestSystem",
			"1 DATE 7 AUG 2026",
			"1 SUBM @U1@",
			"1 GEDC",
			"2 VERS 5.5.1",
			"1 CHAR UTF-8",
		)
	})

	t.Run("7.0 places DATE after SOUR, before SUBM", func(t *testing.T) {
		out := encode(t, &gedcom.Header{
			Version:      gedcom.Version70,
			SourceSystem: "TestSystem",
			Date:         "7 AUG 2026",
			Submitter:    "@U1@",
		})
		assertLineOrder(t, out,
			"0 HEAD",
			"1 GEDC",
			"2 VERS 7.0",
			"1 SOUR TestSystem",
			"1 DATE 7 AUG 2026",
			"1 SUBM @U1@",
		)
	})

	t.Run("value is written verbatim", func(t *testing.T) {
		// Not a well-formed DATE_EXACT: the encoder does not normalize it.
		out := encode(t, &gedcom.Header{Version: gedcom.Version551, Date: "abt 1 jan 2000"})
		if !strings.Contains(out, "\n1 DATE abt 1 jan 2000\n") {
			t.Errorf("DATE not written verbatim:\n%s", out)
		}
	})

	t.Run("line break in Date cannot inject a line", func(t *testing.T) {
		// A hand-built Date taken from user input must not be able to start
		// a new GEDCOM line: the break becomes a CONT under DATE instead.
		out := encode(t, &gedcom.Header{Version: gedcom.Version551, Date: "1 JAN 2000\n0 @X@ INDI"})
		for _, line := range strings.Split(out, "\n") {
			if strings.HasPrefix(line, "0 @X@") {
				t.Fatalf("Date injected a level-0 record:\n%s", out)
			}
		}
		if !strings.Contains(out, "\n1 DATE 1 JAN 2000\n2 CONT 0 @X@ INDI\n") {
			t.Errorf("line break not written as CONT:\n%s", out)
		}
		doc, err := decoder.Decode(strings.NewReader(out))
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if _, ok := doc.XRefMap["@X@"]; ok {
			t.Error("injected @X@ record decoded from the encoded header")
		}
	})

	t.Run("empty Date emits no DATE", func(t *testing.T) {
		out := encode(t, &gedcom.Header{Version: gedcom.Version551, SourceSystem: "TestSystem"})
		if strings.Contains(out, "DATE") {
			t.Errorf("synthesized a DATE the header never had:\n%s", out)
		}
	})

	t.Run("hand-built Date survives encode and decode", func(t *testing.T) {
		out := encode(t, &gedcom.Header{Version: gedcom.Version551, Date: "7 AUG 2026"})
		doc, err := decoder.Decode(strings.NewReader(out))
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if doc.Header.Date != "7 AUG 2026" {
			t.Errorf("Header.Date = %q, want %q", doc.Header.Date, "7 AUG 2026")
		}
	})
}

// TestHeaderDateRoundTrip is the #495 regression: a header carrying
// "1 DATE" with a "2 TIME" subordinate decodes Header.Date verbatim and
// re-encodes byte-identically, TIME included. TIME has no typed field; it
// survives through Header.Tags.
func TestHeaderDateRoundTrip(t *testing.T) {
	versions := []struct {
		name  string
		input string
	}{
		{"5.5", "0 HEAD\n" +
			"1 SOUR TestApp\n" +
			"1 DATE 7 AUG 2026\n" +
			"2 TIME 12:34:56\n" +
			"1 GEDC\n" +
			"2 VERS 5.5\n" +
			"2 FORM LINEAGE-LINKED\n" +
			"1 CHAR UTF-8\n" +
			"0 TRLR\n"},
		{"5.5.1", "0 HEAD\n" +
			"1 SOUR TestApp\n" +
			"2 DATA Source Data\n" +
			"3 DATE 1 JAN 1999\n" +
			"1 DATE 7 AUG 2026\n" +
			"2 TIME 12:34:56.78\n" +
			"1 GEDC\n" +
			"2 VERS 5.5.1\n" +
			"2 FORM LINEAGE-LINKED\n" +
			"1 CHAR UTF-8\n" +
			"0 TRLR\n"},
		{"7.0", "0 HEAD\n" +
			"1 GEDC\n" +
			"2 VERS 7.0\n" +
			"1 SOUR TestApp\n" +
			"2 DATA Source Data\n" +
			"3 DATE 1 JAN 1999\n" +
			"1 DATE 7 AUG 2026\n" +
			"2 TIME 12:34:56Z\n" +
			"0 TRLR\n"},
	}

	for _, tc := range versions {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := decoder.Decode(strings.NewReader(tc.input))
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if doc.Header.Date != "7 AUG 2026" {
				t.Errorf("Header.Date = %q, want %q", doc.Header.Date, "7 AUG 2026")
			}

			var buf bytes.Buffer
			if err := Encode(&buf, doc); err != nil {
				t.Fatalf("encode: %v", err)
			}
			if got := buf.String(); got != tc.input {
				t.Errorf("header did not round-trip byte-identically:\ngot:\n%s\nwant:\n%s", got, tc.input)
			}

			again, err := decoder.Decode(strings.NewReader(buf.String()))
			if err != nil {
				t.Fatalf("re-decode: %v", err)
			}
			if again.Header.Date != doc.Header.Date {
				t.Errorf("re-decoded Header.Date = %q, want %q", again.Header.Date, doc.Header.Date)
			}
		})
	}
}

// TestHeaderDateIgnoresSourceDataDate pins the decoder's level-1 guard: the
// HEAD.SOUR.DATA.DATE publication date is not the transmission date, whether
// it comes after HEAD.DATE (where it would overwrite it) or is the only DATE
// in the header (where it would be adopted).
func TestHeaderDateIgnoresSourceDataDate(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"after HEAD.DATE", "0 HEAD\n" +
			"1 DATE 7 AUG 2026\n" +
			"1 SOUR TestApp\n" +
			"2 DATA Source Data\n" +
			"3 DATE 1 JAN 1999\n" +
			"1 GEDC\n" +
			"2 VERS 5.5.1\n" +
			"2 FORM LINEAGE-LINKED\n" +
			"1 CHAR UTF-8\n" +
			"0 TRLR\n", "7 AUG 2026"},
		{"only DATE in header", "0 HEAD\n" +
			"1 SOUR TestApp\n" +
			"2 DATA Source Data\n" +
			"3 DATE 1 JAN 1999\n" +
			"1 GEDC\n" +
			"2 VERS 5.5.1\n" +
			"2 FORM LINEAGE-LINKED\n" +
			"1 CHAR UTF-8\n" +
			"0 TRLR\n", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := decoder.Decode(strings.NewReader(tc.input))
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if doc.Header.Date != tc.want {
				t.Errorf("Header.Date = %q, want %q", doc.Header.Date, tc.want)
			}

			var buf bytes.Buffer
			if err := Encode(&buf, doc); err != nil {
				t.Fatalf("encode: %v", err)
			}
			if got := buf.String(); got != tc.input {
				t.Errorf("header did not round-trip byte-identically:\ngot:\n%s\nwant:\n%s", got, tc.input)
			}
		})
	}
}
