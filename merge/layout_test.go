package merge_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
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
