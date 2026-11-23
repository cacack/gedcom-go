package decoder

import (
	"os"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

func decodeLayout(t *testing.T, in string) *gedcom.Document {
	t.Helper()
	doc, err := DecodeWithOptions(strings.NewReader(in), &DecodeOptions{PreserveLayout: true})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return doc
}

func TestPreserveLayoutRecordsLines(t *testing.T) {
	in := "\xEF\xBB\xBF0 HEAD\r\n  1 CHAR UTF-8\n\n\r\n0 @I1@ INDI \r1 NOTE A\n2 CONT    \n0 TRLR\n\n"
	layout := decodeLayout(t, in).Layout
	if layout == nil {
		t.Fatal("no layout recorded")
	}

	if !layout.BOM {
		t.Error("BOM not recorded")
	}
	if layout.Head != 1 || layout.Trailer != 8 {
		t.Errorf("Head, Trailer = %d, %d; want 1, 8", layout.Head, layout.Trailer)
	}
	if layout.Tail != "\n" {
		t.Errorf("Tail = %q; want %q", layout.Tail, "\n")
	}
	if len(layout.Lines) != 9 {
		t.Fatalf("len(Lines) = %d; want 9", len(layout.Lines))
	}

	want := map[int]gedcom.LayoutLine{
		1: {Level: 0, Tag: "HEAD", Text: "0 HEAD", Terminator: "\r\n"},
		2: {Level: 1, Tag: "CHAR", Value: "UTF-8", Text: "  1 CHAR UTF-8", Terminator: "\n"},
		5: {Level: 0, XRef: "@I1@", Tag: "INDI", Before: "\n\r\n", Text: "0 @I1@ INDI ", Terminator: "\r"},
		6: {Level: 1, Tag: "NOTE", Value: "A", Text: "1 NOTE A", Terminator: "\n"},
		7: {Level: 2, Tag: "CONT", Text: "2 CONT    ", Terminator: "\n"},
		8: {Level: 0, Tag: "TRLR", Text: "0 TRLR", Terminator: "\n"},
	}
	for n := 1; n <= len(layout.Lines); n++ {
		got := layout.Line(n)
		w, parsed := want[n]
		switch {
		case !parsed && got != nil:
			t.Errorf("line %d: got %+v; want none (skipped)", n, *got)
		case parsed && got == nil:
			t.Errorf("line %d: got none; want %+v", n, w)
		case parsed && *got != w:
			t.Errorf("line %d:\n got %+v\nwant %+v", n, *got, w)
		}
	}
}

func TestPreserveLayoutOffRecordsNothing(t *testing.T) {
	doc, err := Decode(strings.NewReader("0 HEAD\n0 TRLR\n"))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Layout != nil {
		t.Error("Layout recorded without PreserveLayout")
	}
}

// A line lenient recovery edits is recorded as it was read, so the encoder
// can tell it was changed.
func TestPreserveLayoutBeforeRecovery(t *testing.T) {
	layout := decodeLayout(t, "0 HEAD\n0 @I1@ INDI\n1 BIRT\n4 DATE 1900\n0 TRLR\n").Layout
	if got := layout.Line(4); got == nil || got.Level != 4 {
		t.Errorf("line 4 = %+v; want level 4 as read", got)
	}
}

// A UTF-16 source's byte order mark is not a UTF-8 one, and the encoder
// writes UTF-8, so it is not recorded.
func TestPreserveLayoutUTF16BOM(t *testing.T) {
	src, err := os.ReadFile("../testdata/encoding/utf16le.ged")
	if err != nil {
		t.Fatal(err)
	}
	layout := decodeLayout(t, string(src)).Layout
	if layout.BOM {
		t.Error("UTF-16 byte order mark recorded as a UTF-8 one")
	}
	if got := layout.Line(1); got == nil || got.Text != "0 HEAD" {
		t.Errorf("line 1 = %+v; want decoded text 0 HEAD", got)
	}
}

// Input with no line that parses still gets a layout, holding the input.
func TestPreserveLayoutNoLines(t *testing.T) {
	doc, _ := DecodeWithOptions(strings.NewReader("\n\n"), &DecodeOptions{PreserveLayout: true})
	if doc == nil || doc.Layout == nil {
		t.Fatal("no layout for input with no lines")
	}
	if doc.Layout.Tail != "\n\n" || doc.Layout.Head != 0 || doc.Layout.Trailer != 0 {
		t.Errorf("layout = %+v; want only Tail %q", *doc.Layout, "\n\n")
	}
}

// A bare CR ends a line, as it does for the parser, including at the very end
// of the input.
func TestPreserveLayoutBareCR(t *testing.T) {
	layout := decodeLayout(t, "0 HEAD\r0 TRLR\r").Layout
	if got := layout.Line(2); got == nil || got.Text != "0 TRLR" || got.Terminator != "\r" {
		t.Errorf("line 2 = %+v; want 0 TRLR ended by CR", got)
	}
}

// A final line with no terminator is recorded with none.
func TestPreserveLayoutUnterminated(t *testing.T) {
	layout := decodeLayout(t, "0 HEAD\n0 TRLR").Layout
	if got := layout.Line(2); got == nil || got.Text != "0 TRLR" || got.Terminator != "" {
		t.Errorf("line 2 = %+v; want 0 TRLR with no terminator", got)
	}
}
