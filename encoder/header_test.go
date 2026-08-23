package encoder

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/decoder"
	"github.com/cacack/gedcom-go/v2/gedcom"
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
