package merge_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/merge"
)

// keepFirstDoc1 has no NOTE or DEST in its header; keepFirstDoc2 has both,
// plus a vendor extension and a SUBM. Their records do not collide.
const (
	keepFirstDoc1 = "0 HEAD\n1 SOUR AppA\n2 VERS 1.0\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
		"0 @I1@ INDI\n1 NAME Ann /Example/\n0 TRLR\n"
	keepFirstDoc2 = "0 HEAD\n1 SOUR AppB\n1 DEST GEDCOM 5.5\n1 SUBM @U9@\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
		"1 LANG English\n1 NOTE Written by\n2 CONT another program\n1 _EXT x\n" +
		"0 @I2@ INDI\n1 NAME Bob /Sample/\n0 @U9@ SUBM\n1 NAME Sub\n0 TRLR\n"
)

func encodeHeader(t *testing.T, doc *gedcom.Document) string {
	t.Helper()
	var buf bytes.Buffer
	if err := encoder.Encode(&buf, &gedcom.Document{Header: doc.Header, Trailer: &gedcom.Trailer{}}); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return buf.String()
}

// TestCombine_KeepFirstHeader pins #931: under every strategy the combined
// header is doc1's alone, and every doc2 structure is reported.
func TestCombine_KeepFirstHeader(t *testing.T) {
	strategies := []merge.CombineOptions{
		{CollisionStrategy: merge.ErrorOnCollision},
		{CollisionStrategy: merge.PrefixDoc2, Prefix: "B_"},
		{CollisionStrategy: merge.PrefixAllDoc2, Prefix: "B_"},
		{CollisionStrategy: merge.RenumberDoc2},
		{CollisionStrategy: merge.ZipSharedXRefs},
	}
	for _, opts := range strategies {
		doc1 := decodePrefixAll(t, keepFirstDoc1)
		doc2 := decodePrefixAll(t, keepFirstDoc2)
		wantHeader := encodeHeader(t, doc1)

		// The default policy carries the structures doc1 lacks.
		out, _, err := merge.Combine(doc1, doc2, opts)
		if err != nil {
			t.Fatalf("Combine(%v): %v", opts.CollisionStrategy, err)
		}
		if got := encodeHeader(t, out); !strings.Contains(got, "1 NOTE") || !strings.Contains(got, "1 DEST") {
			t.Errorf("strategy %v default: header lost doc2's NOTE or DEST:\n%s", opts.CollisionStrategy, got)
		}

		opts.KeepFirstHeader = true
		out, report, err := merge.Combine(doc1, doc2, opts)
		if err != nil {
			t.Fatalf("Combine(%v, KeepFirstHeader): %v", opts.CollisionStrategy, err)
		}
		if got := encodeHeader(t, out); got != wantHeader {
			t.Errorf("strategy %v: header =\n%s\nwant doc1's\n%s", opts.CollisionStrategy, got, wantHeader)
		}
		if out.Header.Destination != "" || out.Header.Submitter != "" || out.Header.Language != "" {
			t.Errorf("strategy %v: typed doc2 values carried: %+v", opts.CollisionStrategy, out.Header)
		}
		got := map[string]string{}
		for _, c := range report.HeaderConflicts {
			got[c.Field] = c.Doc1
		}
		want := map[string]string{
			"Tags.SOUR": "kept", "Tags.DEST": "", "Tags.SUBM": "", "Tags.GEDC": "kept",
			"Tags.CHAR": "kept", "Tags.LANG": "", "Tags.NOTE": "", "Tags._EXT": "",
			"SourceSystem": "AppA", "Language": "", "Submitter": "",
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("strategy %v: conflicts = %v, want %v", opts.CollisionStrategy, got, want)
		}
		if out.GetIndividual(report.RemappedXRefs["@I2@"]) == nil && out.GetIndividual("@I2@") == nil {
			t.Errorf("strategy %v: doc2's records missing", opts.CollisionStrategy)
		}
	}
}

// TestCombine_KeepFirstHeader_HandBuilt pins the documented exception and
// the typed values a header built in code reports.
func TestCombine_KeepFirstHeader_HandBuilt(t *testing.T) {
	doc1 := &gedcom.Document{Header: &gedcom.Header{SourceSystem: "A"}, XRefMap: map[string]*gedcom.Record{}}
	doc2 := handBuiltHeaderDoc(&gedcom.Header{
		DeclaredVersion: gedcom.Version551, Encoding: gedcom.EncodingUTF8,
		SourceSystem: "A", Date: "1 JAN 2000", PlaceForm: "Town", Copyright: "C",
	})
	doc2.Schema = &gedcom.SchemaDefinition{TagMappings: map[string]string{"_X": "https://example.com/x"}}
	doc2.Vendor = gedcom.VendorGramps

	out, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{KeepFirstHeader: true})
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}
	h := out.Header
	if h.Version != gedcom.Version551 || h.DeclaredVersion != gedcom.Version551 {
		t.Errorf("version not taken from doc2: %q / %q", h.Version, h.DeclaredVersion)
	}
	if h.Encoding != "" || h.Date != "" || h.PlaceForm != "" || h.Copyright != "" || out.Schema != nil || out.Vendor != "" {
		t.Errorf("doc2 values carried: %+v schema %v vendor %q", h, out.Schema, out.Vendor)
	}
	var fields []string
	for _, c := range report.HeaderConflicts {
		fields = append(fields, c.Field)
	}
	want := []string{"Encoding", "Copyright", "Date", "PlaceForm", "Schema", "Vendor"}
	if !reflect.DeepEqual(fields, want) {
		t.Errorf("conflicts = %v, want %v", fields, want)
	}
}

// TestCombine_KeepFirstHeader_NilHeaders covers a missing header on either
// side.
func TestCombine_KeepFirstHeader_NilHeaders(t *testing.T) {
	withHeader := handBuiltHeaderDoc(&gedcom.Header{SourceSystem: "A"})
	bare := &gedcom.Document{XRefMap: map[string]*gedcom.Record{}}
	opts := merge.CombineOptions{KeepFirstHeader: true}

	out, report, err := merge.Combine(withHeader, bare, opts)
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}
	if out.Header.SourceSystem != "A" || len(report.HeaderConflicts) != 0 {
		t.Errorf("header = %+v, conflicts = %v", out.Header, report.HeaderConflicts)
	}

	out, report, err = merge.Combine(bare, withHeader, opts)
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}
	if out.Header.SourceSystem != "" || out.Header.Version != gedcom.Version551 {
		t.Errorf("header = %+v, want version only", out.Header)
	}
	if len(report.HeaderConflicts) != 1 || report.HeaderConflicts[0].Field != "SourceSystem" {
		t.Errorf("conflicts = %v", report.HeaderConflicts)
	}
}
