package gedcomgo

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/converter"
	"github.com/cacack/gedcom-go/v2/decoder"
	"github.com/cacack/gedcom-go/v2/encoder"
	"github.com/cacack/gedcom-go/v2/gedcom"
	"github.com/cacack/gedcom-go/v2/merge"
)

// Cross-package guards for the header write path.
//
// The encoder writes a decoded document's HEAD from Header.Tags (#429). Three
// other packages produce that field -- merge concatenates two documents' tags,
// and the converter rewrites and removes them -- and all three were written
// against an encoder that discarded it. Each defect below was invisible to
// every existing test because none of them encoded the output of a merge or a
// conversion and looked at the header it produced.
//
// These live in the root package because that is the only one that may import
// merge, converter and encoder together.

func decodeFixture(t *testing.T, path string) *gedcom.Document {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	doc, err := decoder.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return doc
}

// encodeHead returns the HEAD block of doc's encoded form.
func encodeHead(t *testing.T, doc *gedcom.Document) string {
	t.Helper()

	var buf bytes.Buffer
	if err := encoder.Encode(&buf, doc); err != nil {
		t.Fatalf("encode: %v", err)
	}
	out := buf.String()
	if i := strings.Index(out, "\n0 @"); i > 0 {
		return out[:i]
	}
	if i := strings.Index(out, "\n0 TRLR"); i > 0 {
		return out[:i]
	}
	return out
}

// countLines reports how many lines of head begin with prefix.
func countLines(head, prefix string) int {
	n := 0
	for _, line := range strings.Split(head, "\n") {
		if strings.HasPrefix(line, prefix) {
			n++
		}
	}
	return n
}

// A GEDCOM header has no repeatable substructures, so a merged document must
// not declare two of anything. merge concatenated both headers' tags, which was
// harmless only while the encoder ignored them.
func TestMergedDocumentEncodesOneHeader(t *testing.T) {
	doc1 := decodeFixture(t, "testdata/gedcom-5.5.1/comprehensive.ged")
	doc2 := decodeFixture(t, "testdata/edge-cases/calendar-dates.ged")

	combined, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{
		CollisionStrategy: merge.PrefixDoc2,
		Prefix:            "B",
	})
	if err != nil {
		t.Fatalf("combine: %v", err)
	}

	head := encodeHead(t, combined)

	// Both fixtures carry every one of these, so each is a real collision.
	for _, structure := range []string{"1 SOUR", "1 GEDC", "1 CHAR", "1 NOTE"} {
		if n := countLines(head, structure); n > 1 {
			t.Errorf("merged HEAD declares %s %d times:\n%s", structure, n, head)
		}
	}

	// The drop is reported, not silent -- matching how the scalar fields
	// already report a doc2 value discarded in favour of doc1.
	var reportedTagConflict bool
	for _, c := range report.HeaderConflicts {
		if strings.HasPrefix(c.Field, "Tags.") {
			reportedTagConflict = true
		}
	}
	if !reportedTagConflict {
		t.Errorf("dropped header structures were not reported: %+v", report.HeaderConflicts)
	}
}

// A vendor extension is outside the grammar that makes header structures
// singular, so merge still keeps both documents' copies.
func TestMergeKeepsCustomHeaderTagsFromBoth(t *testing.T) {
	mk := func(xref, vendor string) *gedcom.Document {
		return &gedcom.Document{
			Header: &gedcom.Header{
				Version: gedcom.Version551,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "SOUR", Value: "System" + xref},
					{Level: 1, Tag: "_VENDOR", Value: vendor},
				},
			},
			XRefMap: make(map[string]*gedcom.Record),
		}
	}

	combined, _, err := merge.Combine(mk("A", "alpha"), mk("B", "beta"), merge.CombineOptions{
		CollisionStrategy: merge.PrefixDoc2,
		Prefix:            "B",
	})
	if err != nil {
		t.Fatalf("combine: %v", err)
	}

	head := encodeHead(t, combined)
	for _, want := range []string{"_VENDOR alpha", "_VENDOR beta"} {
		if !strings.Contains(head, want) {
			t.Errorf("merged HEAD lost %q:\n%s", want, head)
		}
	}
	if n := countLines(head, "1 SOUR"); n != 1 {
		t.Errorf("merged HEAD declares SOUR %d times, want 1:\n%s", n, head)
	}
}

// CHAR is mandatory in 5.5 and 5.5.1 and absent from 7.0. A 7.0 source has no
// CHAR tag to rewrite, so a downgrade has to add one; an upgrade has to remove
// the one a 5.5.x source carried. Neither is reachable through the typed
// Header.Encoding field, which the encoder no longer consults for a document
// that has raw tags.
func TestConversionKeepsCharDeclarationCorrect(t *testing.T) {
	t.Run("7.0 to 5.5.1 declares CHAR", func(t *testing.T) {
		doc := decodeFixture(t, "testdata/gedcom-7.0/minimal70.ged")
		converted, _, err := converter.ConvertWithOptions(doc, gedcom.Version551, converter.DefaultOptions())
		if err != nil {
			t.Fatalf("convert: %v", err)
		}

		head := encodeHead(t, converted)
		if !strings.Contains(head, "1 CHAR UTF-8") {
			t.Errorf("downgraded 5.5.1 header has no CHAR line, which the version requires:\n%s", head)
		}
	})

	t.Run("5.5.1 to 7.0 drops CHAR and GEDC.FORM", func(t *testing.T) {
		doc := decodeFixture(t, "testdata/gedcom-5.5.1/comprehensive.ged")
		converted, _, err := converter.ConvertWithOptions(doc, gedcom.Version70, converter.DefaultOptions())
		if err != nil {
			t.Fatalf("convert: %v", err)
		}

		head := encodeHead(t, converted)
		if strings.Contains(head, "1 CHAR") {
			t.Errorf("7.0 header declares CHAR, which 7.0 removed:\n%s", head)
		}
		if strings.Contains(head, "2 FORM LINEAGE-LINKED") {
			t.Errorf("7.0 header carries GEDC.FORM, which 7.0 removed:\n%s", head)
		}
		// PLAC.FORM is a different structure and stays valid in 7.0.
		if !strings.Contains(head, "1 PLAC") {
			t.Errorf("PLAC structure was removed along with GEDC.FORM:\n%s", head)
		}
	})
}

// Header.Tags is a flat, level-encoded list, so removing a structure means
// removing its subtree. Dropping the SCHMA line alone left its TAG children to
// re-parent under the preceding structure -- maximal70 downgraded declared its
// extension URIs under HEAD.NOTE.
func TestDowngradeRemovesSchmaSubtree(t *testing.T) {
	doc := decodeFixture(t, "testdata/gedcom-7.0/maximal70.ged")
	converted, _, err := converter.ConvertWithOptions(doc, gedcom.Version551, converter.DefaultOptions())
	if err != nil {
		t.Fatalf("convert: %v", err)
	}

	head := encodeHead(t, converted)
	if strings.Contains(head, "1 SCHMA") {
		t.Errorf("SCHMA survived into 5.5.1:\n%s", head)
	}
	for _, orphan := range []string{"TAG _SKYPEID", "TAG _JABBERID"} {
		if strings.Contains(head, orphan) {
			t.Errorf("SCHMA child %q survived as an orphan:\n%s", orphan, head)
		}
	}
	if !strings.Contains(head, "2 VERS 5.5.1") {
		t.Errorf("downgraded header does not declare 5.5.1:\n%s", head)
	}
}

// Whatever a conversion produces must survive its own encoder: a converted
// document that cannot be re-decoded is not a conversion.
func TestConvertedDocumentsReDecode(t *testing.T) {
	cases := []struct {
		fixture string
		target  gedcom.Version
	}{
		{"testdata/gedcom-7.0/maximal70.ged", gedcom.Version551},
		{"testdata/gedcom-7.0/minimal70.ged", gedcom.Version55},
		{"testdata/gedcom-5.5.1/comprehensive.ged", gedcom.Version70},
		{"testdata/gedcom-5.5/royal92.ged", gedcom.Version551},
	}

	for _, tc := range cases {
		t.Run(tc.fixture+" -> "+tc.target.String(), func(t *testing.T) {
			doc := decodeFixture(t, tc.fixture)
			converted, _, err := converter.ConvertWithOptions(doc, tc.target, converter.DefaultOptions())
			if err != nil {
				t.Fatalf("convert: %v", err)
			}

			var buf bytes.Buffer
			if err := encoder.Encode(&buf, converted); err != nil {
				t.Fatalf("encode: %v", err)
			}
			redecoded, err := decoder.Decode(bytes.NewReader(buf.Bytes()))
			if err != nil {
				t.Fatalf("re-decode of converted output: %v", err)
			}
			if len(redecoded.Records) != len(converted.Records) {
				t.Errorf("record count after re-decode = %d, want %d",
					len(redecoded.Records), len(converted.Records))
			}
		})
	}
}
