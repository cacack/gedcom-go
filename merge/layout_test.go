package merge_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/merge"
)

// TestCombineDropsLayout pins that a combined document has no layout: doc2's
// line numbers would otherwise index doc1's layout, and doc2's lines would
// borrow doc1's blank lines and terminators when encoded with PreserveLayout.
func TestCombineDropsLayout(t *testing.T) {
	opts := &decoder.DecodeOptions{PreserveLayout: true}
	doc1, err := decoder.DecodeWithOptions(strings.NewReader(
		"0 HEAD\r\n1 CHAR UTF-8\r\n\r\n0 @I1@ INDI\r\n1 NAME Ann /Example/\r\n0 TRLR\r\n"), opts)
	if err != nil {
		t.Fatalf("decode doc1: %v", err)
	}
	doc2, err := decoder.DecodeWithOptions(strings.NewReader(
		"0 HEAD\n1 CHAR UTF-8\n0 @I2@ INDI\n1 SEX F\n0 TRLR\n"), opts)
	if err != nil {
		t.Fatalf("decode doc2: %v", err)
	}

	out, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}
	if out.Layout != nil {
		t.Fatal("combined document kept doc1's layout")
	}
	if doc1.Layout == nil {
		t.Fatal("Combine cleared doc1's own layout")
	}

	var buf bytes.Buffer
	if err := encoder.EncodeWithOptions(&buf, out, &encoder.EncodeOptions{PreserveLayout: true}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	if got := buf.String(); strings.Contains(got, "\r") || strings.Contains(got, "\n\n") {
		t.Errorf("combined output borrowed doc1's layout: %q", got)
	}
}

// TestCombineZipDropsLayout pins the same for ZipSharedXRefs, which unions
// the two documents on its own path.
func TestCombineZipDropsLayout(t *testing.T) {
	opts := &decoder.DecodeOptions{PreserveLayout: true}
	doc1, err := decoder.DecodeWithOptions(strings.NewReader(
		"0 HEAD\r\n1 CHAR UTF-8\r\n\r\n0 @I1@ INDI\r\n1 NAME Ann /Example/\r\n0 TRLR\r\n"), opts)
	if err != nil {
		t.Fatalf("decode doc1: %v", err)
	}
	doc2, err := decoder.DecodeWithOptions(strings.NewReader(
		"0 HEAD\n1 CHAR UTF-8\n0 @I1@ INDI\n1 NAME Ann /Example/\n0 @I2@ INDI\n1 SEX F\n0 TRLR\n"), opts)
	if err != nil {
		t.Fatalf("decode doc2: %v", err)
	}

	out, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{CollisionStrategy: merge.ZipSharedXRefs})
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}
	if out.Layout != nil {
		t.Error("zipped document kept doc1's layout")
	}
	if doc1.Layout == nil || doc2.Layout == nil {
		t.Error("Combine cleared an input's own layout")
	}

	var preserved, plain bytes.Buffer
	if err := encoder.EncodeWithOptions(&preserved, out, &encoder.EncodeOptions{PreserveLayout: true}); err != nil {
		t.Fatalf("encode with PreserveLayout: %v", err)
	}
	if err := encoder.EncodeWithOptions(&plain, out, &encoder.EncodeOptions{}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	if !bytes.Equal(preserved.Bytes(), plain.Bytes()) {
		t.Errorf("zipped output borrowed a layout:\n got %q\nwant %q", preserved.String(), plain.String())
	}
}

// TestSingleDocumentEditsKeepLayout pins the other side: Individuals,
// Families and RemapXRefs edit one document, whose line numbers still index
// its own layout, so the result keeps it and only the edited lines differ.
func TestSingleDocumentEditsKeepLayout(t *testing.T) {
	const src = "0 HEAD\r\n1 CHAR UTF-8\r\n\r\n" +
		"0 @I1@ INDI\r\n  1 NAME Ann /Example/\r\n1 FAMS @F1@\r\n\r\n" +
		"0 @I2@ INDI\r\n1 NAME Ann /Example/\r\n\r\n1 NAME Annie /Example/\r\n1 FAMS @F2@\r\n\r\n" +
		"0 @F1@ FAM\r\n1 WIFE @I1@\r\n1 MARR\r\n\r\n" +
		"0 @F2@ FAM\r\n1 WIFE @I2@\r\n 1 DIV\r\n0 TRLR\r\n"
	doc, err := decoder.DecodeWithOptions(strings.NewReader(src), &decoder.DecodeOptions{PreserveLayout: true})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	merged, _, err := merge.Individuals(doc, "@I1@", "@I2@", merge.RecordMergeOptions{})
	if err != nil {
		t.Fatalf("Individuals: %v", err)
	}
	families, _, err := merge.Families(merged, "@F1@", "@F2@", merge.RecordMergeOptions{})
	if err != nil {
		t.Fatalf("Families: %v", err)
	}
	remapped, _, err := merge.RemapXRefs(doc, func(old string) string { return "@X" + old[1:] })
	if err != nil {
		t.Fatalf("RemapXRefs: %v", err)
	}

	tests := []struct {
		name string
		got  *gedcom.Document
		want string
	}{
		{"Individuals", merged, "0 HEAD\r\n1 CHAR UTF-8\r\n\r\n" +
			"0 @I1@ INDI\r\n  1 NAME Ann /Example/\r\n\r\n1 NAME Annie /Example/\r\n1 FAMS @F1@\r\n1 FAMS @F2@\r\n\r\n" +
			"0 @F1@ FAM\r\n1 WIFE @I1@\r\n1 MARR\r\n\r\n" +
			"0 @F2@ FAM\r\n1 WIFE @I1@\r\n 1 DIV\r\n0 TRLR\r\n"},
		{"Families", families, "0 HEAD\r\n1 CHAR UTF-8\r\n\r\n" +
			"0 @I1@ INDI\r\n  1 NAME Ann /Example/\r\n\r\n1 NAME Annie /Example/\r\n1 FAMS @F1@\r\n\r\n" +
			"0 @F1@ FAM\r\n1 WIFE @I1@\r\n1 MARR\r\n 1 DIV\r\n0 TRLR\r\n"},
		{"RemapXRefs", remapped, strings.NewReplacer("@I", "@XI", "@F", "@XF").Replace(src)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got.Layout != doc.Layout {
				t.Error("result does not carry the source's layout")
			}
			var buf bytes.Buffer
			if err := encoder.EncodeWithOptions(&buf, tt.got, &encoder.EncodeOptions{PreserveLayout: true}); err != nil {
				t.Fatalf("encode: %v", err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("got %q\nwant %q", got, tt.want)
			}
		})
	}
}
