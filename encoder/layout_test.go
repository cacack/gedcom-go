package encoder

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// decodeLayout decodes in with its layout recorded.
func decodeLayout(t *testing.T, in string) *gedcom.Document {
	t.Helper()
	doc, err := decoder.DecodeWithOptions(strings.NewReader(in), &decoder.DecodeOptions{PreserveLayout: true})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return doc
}

// encodeLayout encodes doc with PreserveLayout on.
func encodeLayout(t *testing.T, doc *gedcom.Document) string {
	t.Helper()
	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, doc, &EncodeOptions{PreserveLayout: true}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.String()
}

// TestPreserveLayoutUnmodified pins the first two criteria of #876: an
// unmodified document comes back byte for byte, for each layout real exports
// carry that the default encoder normalizes.
func TestPreserveLayoutUnmodified(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"UTF-8 BOM", "\xEF\xBB\xBF0 HEAD\n1 CHAR UTF-8\n0 @I1@ INDI\n1 NAME Ann /Example/\n0 TRLR\n"},
		{"CRLF", "0 HEAD\r\n1 CHAR UTF-8\r\n0 @I1@ INDI\r\n1 NAME Ann /Example/\r\n0 TRLR\r\n"},
		{"bare CR", "0 HEAD\r1 CHAR UTF-8\r0 @I1@ INDI\r1 NAME Ann /Example/\r0 TRLR\r"},
		{"mixed LF and CRLF", "0 HEAD\n1 CHAR UTF-8\n0 @O1@ OBJE\n1 _DSCR First\r\n2 CONT Second\r\n1 FILE a.jpg\n0 TRLR\n"},
		{"leading indentation", "0 HEAD\n  1 CHAR UTF-8\n0 @I1@ INDI\n  1 NAME Ann /Example/\n    2 GIVN Ann\n0 TRLR\n"},
		{"blank lines between records", "0 HEAD\n1 CHAR UTF-8\n\n0 @I1@ INDI\n1 NAME Ann /Example/\n\n\n0 @I2@ INDI\n1 NAME Bob /Example/\n\n0 TRLR\n"},
		{"delimiter space on valueless lines", "0 HEAD \n1 CHAR UTF-8\n0 @I1@ INDI \n1 NOTE First\n2 CONT \n1 ADDR \n2 CITY Exampleton\n0 TRLR\n"},
		{"whitespace-only value", "0 HEAD\n1 CHAR UTF-8\n0 @I1@ INDI\n1 NOTE First\n2 CONT    \n2 CONT Third\n0 TRLR\n"},
		{"no final terminator", "0 HEAD\n1 CHAR UTF-8\n0 @I1@ INDI\n0 TRLR"},
		{"trailing blank lines", "0 HEAD\n1 CHAR UTF-8\n0 @I1@ INDI\n0 TRLR\n\n\n"},
		{"double space after level", "0 HEAD\n1  CHAR UTF-8\n0 @I1@ INDI\n1  NAME Ann /Example/\n0 TRLR\n"},
		{"tab indentation", "0 HEAD\n\t1 CHAR UTF-8\n0 @I1@ INDI\n\t1 SEX F\n0 TRLR\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := encodeLayout(t, decodeLayout(t, tt.in)); got != tt.in {
				t.Errorf("not byte-identical\n got: %q\nwant: %q", got, tt.in)
			}
		})
	}
}

// TestPreserveLayoutEdit pins the third criterion of #876: changing one
// value changes that line and nothing else. The changed line keeps its
// source line's indentation and terminator.
func TestPreserveLayoutEdit(t *testing.T) {
	in := "\xEF\xBB\xBF0 HEAD\r\n  1 CHAR UTF-8\r\n\r\n0 @I1@ INDI\r\n  1 NAME Ann /Example/\r\n  1 NOTE  \r\n\r\n0 @I2@ INDI\r\n  1 NAME Bob /Example/\n0 TRLR\r\n"
	doc := decodeLayout(t, in)
	doc.GetRecord("@I1@").Tags[0].Value = "Anne /Example/"

	got := encodeLayout(t, doc)
	want := strings.Replace(in, "Ann /Example/", "Anne /Example/", 1)
	if got != want {
		t.Errorf("edit reached beyond its line\n got: %q\nwant: %q", got, want)
	}
}

// TestPreserveLayoutStructureEdits covers adding, removing and rewriting
// lines: each changes only the lines concerned, and an added line takes the
// source's terminator.
func TestPreserveLayoutStructureEdits(t *testing.T) {
	in := "0 HEAD\r\n1 CHAR UTF-8\r\n0 @I1@ INDI\r\n  1 NAME Ann /Example/\r\n  1 SEX F\r\n\r\n0 @F1@ FAM\r\n  1 HUSB @I2@\r\n0 TRLR\r\n"

	t.Run("add a line", func(t *testing.T) {
		doc := decodeLayout(t, in)
		rec := doc.GetRecord("@I1@")
		rec.Tags = append(rec.Tags, &gedcom.Tag{Level: 1, Tag: "OCCU", Value: "Example maker"})
		want := strings.Replace(in, "  1 SEX F\r\n", "  1 SEX F\r\n1 OCCU Example maker\r\n", 1)
		if got := encodeLayout(t, doc); got != want {
			t.Errorf("\n got: %q\nwant: %q", got, want)
		}
	})

	t.Run("remove a line", func(t *testing.T) {
		doc := decodeLayout(t, in)
		rec := doc.GetRecord("@I1@")
		rec.Tags = rec.Tags[:1]
		want := strings.Replace(in, "  1 SEX F\r\n", "", 1)
		if got := encodeLayout(t, doc); got != want {
			t.Errorf("\n got: %q\nwant: %q", got, want)
		}
	})

	t.Run("edit a record line", func(t *testing.T) {
		doc := decodeLayout(t, in)
		doc.GetRecord("@F1@").XRef = "@F9@"
		want := strings.Replace(in, "0 @F1@ FAM", "0 @F9@ FAM", 1)
		if got := encodeLayout(t, doc); got != want {
			t.Errorf("\n got: %q\nwant: %q", got, want)
		}
	})

	t.Run("value gains a line break", func(t *testing.T) {
		doc := decodeLayout(t, in)
		doc.GetRecord("@I1@").Tags[0].Value = "Ann\nAnother"
		want := strings.Replace(in, "  1 NAME Ann /Example/\r\n", "  1 NAME Ann\r\n2 CONT Another\r\n", 1)
		if got := encodeLayout(t, doc); got != want {
			t.Errorf("\n got: %q\nwant: %q", got, want)
		}
	})

	t.Run("a moved line keeps its own layout", func(t *testing.T) {
		doc := decodeLayout(t, in)
		rec := doc.GetRecord("@I1@")
		rec.Tags[0], rec.Tags[1] = rec.Tags[1], rec.Tags[0]
		want := strings.Replace(in, "  1 NAME Ann /Example/\r\n  1 SEX F\r\n", "  1 SEX F\r\n  1 NAME Ann /Example/\r\n", 1)
		if got := encodeLayout(t, doc); got != want {
			t.Errorf("\n got: %q\nwant: %q", got, want)
		}
	})
}

// TestPreserveLayoutOff pins the fourth criterion of #876: without the
// option the output is the default encoder's, whether or not the document
// carries a layout.
func TestPreserveLayoutOff(t *testing.T) {
	in := "\xEF\xBB\xBF0 HEAD\r\n  1 CHAR UTF-8\r\n\r\n0 @I1@ INDI \r\n  1 NOTE A\r\n  2 CONT    \r\n0 TRLR\r\n\r\n"
	plain, err := decoder.Decode(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	var want bytes.Buffer
	if err := Encode(&want, plain); err != nil {
		t.Fatal(err)
	}

	withLayout := decodeLayout(t, in)
	if withLayout.Layout == nil {
		t.Fatal("decode recorded no layout")
	}
	var got bytes.Buffer
	if err := Encode(&got, withLayout); err != nil {
		t.Fatal(err)
	}
	if got.String() != want.String() {
		t.Errorf("a recorded layout changed default output\n got: %q\nwant: %q", got.String(), want.String())
	}

	// The option on a document with no layout changes nothing either.
	var noLayout bytes.Buffer
	if err := EncodeWithOptions(&noLayout, plain, &EncodeOptions{PreserveLayout: true}); err != nil {
		t.Fatal(err)
	}
	if noLayout.String() != want.String() {
		t.Errorf("PreserveLayout without a layout changed output\n got: %q\nwant: %q", noLayout.String(), want.String())
	}
}

// TestPreserveLayoutEncoderRules checks that the encoder's own rules still
// apply: a line they change is a changed line.
func TestPreserveLayoutEncoderRules(t *testing.T) {
	t.Run("CHAR is written as UTF-8", func(t *testing.T) {
		in := "0 HEAD\r\n  1 CHAR ASCII\r\n0 TRLR\r\n"
		want := "0 HEAD\r\n  1 CHAR UTF-8\r\n0 TRLR\r\n"
		if got := encodeLayout(t, decodeLayout(t, in)); got != want {
			t.Errorf("\n got: %q\nwant: %q", got, want)
		}
	})

	t.Run("DropUnknownTags drops a custom tag and the lines before it", func(t *testing.T) {
		in := "0 HEAD\n1 CHAR UTF-8\n0 @I1@ INDI\n\n1 _CUSTOM x\n2 _SUB y\n1 SEX F\n0 TRLR\n"
		want := "0 HEAD\n1 CHAR UTF-8\n0 @I1@ INDI\n1 SEX F\n0 TRLR\n"
		var buf bytes.Buffer
		err := EncodeWithOptions(&buf, decodeLayout(t, in), &EncodeOptions{PreserveLayout: true, DropUnknownTags: true})
		if err != nil {
			t.Fatal(err)
		}
		if buf.String() != want {
			t.Errorf("\n got: %q\nwant: %q", buf.String(), want)
		}
	})

	t.Run("a clamped level is written as recovered", func(t *testing.T) {
		in := "0 HEAD\n1 CHAR UTF-8\n0 @I1@ INDI\n1 BIRT\n   4 DATE 1900\n0 TRLR\n"
		want := "0 HEAD\n1 CHAR UTF-8\n0 @I1@ INDI\n1 BIRT\n   2 DATE 1900\n0 TRLR\n"
		if got := encodeLayout(t, decodeLayout(t, in)); got != want {
			t.Errorf("\n got: %q\nwant: %q", got, want)
		}
	})

	t.Run("a missing trailer is supplied after an unterminated last line", func(t *testing.T) {
		in := "0 HEAD\r\n1 CHAR UTF-8\r\n0 @I1@ INDI"
		want := "0 HEAD\r\n1 CHAR UTF-8\r\n0 @I1@ INDI\r\n0 TRLR\r\n"
		if got := encodeLayout(t, decodeLayout(t, in)); got != want {
			t.Errorf("\n got: %q\nwant: %q", got, want)
		}
	})

	t.Run("a line the encoder rejects is rejected", func(t *testing.T) {
		doc := decodeLayout(t, "0 HEAD\n1 CHAR UTF-8\n0 @I1@ INDI\n1 SEX F\n0 TRLR\n")
		doc.GetRecord("@I1@").Tags[0].Tag = "S X"
		err := EncodeWithOptions(&bytes.Buffer{}, doc, &EncodeOptions{PreserveLayout: true})
		if err == nil {
			t.Fatal("encoded a tag name holding a space")
		}
	})
}

// TestPreserveLayoutStreaming checks EncodeStreamingWithOptions writes the
// layout back as EncodeWithOptions does.
func TestPreserveLayoutStreaming(t *testing.T) {
	in := "\xEF\xBB\xBF0 HEAD\r\n  1 CHAR UTF-8\r\n\r\n0 @I1@ INDI\r\n  1 NAME Ann /Example/\n0 TRLR\r\n\r\n"
	var buf bytes.Buffer
	if err := EncodeStreamingWithOptions(&buf, decodeLayout(t, in), &EncodeOptions{PreserveLayout: true}); err != nil {
		t.Fatal(err)
	}
	if buf.String() != in {
		t.Errorf("\n got: %q\nwant: %q", buf.String(), in)
	}
}

// TestPreserveLayoutClone checks a cloned document keeps its source's
// layout, so cloning before an edit loses nothing.
func TestPreserveLayoutClone(t *testing.T) {
	in := "0 HEAD\r\n  1 CHAR UTF-8\r\n0 @I1@ INDI\r\n  1 SEX F\r\n0 TRLR\r\n"
	if got := encodeLayout(t, decodeLayout(t, in).Clone()); got != in {
		t.Errorf("\n got: %q\nwant: %q", got, in)
	}
}

// TestPreserveLayoutSyncTags checks a typed edit carried to Tags by SyncTags
// changes only its own line: SyncTags keeps every unchanged line's Tag, so
// each still finds its source line.
func TestPreserveLayoutSyncTags(t *testing.T) {
	in := "0 HEAD\r\n1 CHAR UTF-8\r\n0 @I1@ INDI\r\n  1 NAME Ann /Example/\r\n  1 BIRT\r\n    2 DATE 1 JAN 1900\r\n    2 PLAC Exampleton\r\n0 TRLR\r\n"
	doc := decodeLayout(t, in)
	doc.GetIndividual("@I1@").Events[0].Date = "2 JAN 1900"
	if err := SyncTags(doc); err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(in, "1 JAN 1900", "2 JAN 1900", 1)
	if got := encodeLayout(t, doc); got != want {
		t.Errorf("\n got: %q\nwant: %q", got, want)
	}
}

// TestPreserveLayoutWriteErrors checks a write error at any point of a
// layout-preserving encode is returned, not swallowed.
func TestPreserveLayoutWriteErrors(t *testing.T) {
	in := "\xEF\xBB\xBF0 HEAD\n  1 CHAR ASCII\n\n0 @I1@ INDI\n1 SEX F\n0 TRLR\n\n"
	doc := decodeLayout(t, in)
	writes := &failWriter{failAfter: 1 << 30}
	if err := EncodeWithOptions(writes, doc, &EncodeOptions{PreserveLayout: true}); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < writes.count; n++ {
		w := &failWriter{failAfter: n}
		if err := EncodeWithOptions(w, doc, &EncodeOptions{PreserveLayout: true}); err == nil {
			t.Errorf("write %d failed but EncodeWithOptions returned nil", n+1)
		}
	}
}

// TestPreserveLayoutNoTerminator checks a source with no line terminator at
// all leaves new lines to LineEnding.
func TestPreserveLayoutNoTerminator(t *testing.T) {
	doc := decodeLayout(t, "0 HEAD")
	doc.Records = append(doc.Records, &gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual})
	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, doc, &EncodeOptions{PreserveLayout: true, LineEnding: "\r\n"}); err != nil {
		t.Fatal(err)
	}
	// The empty header is filled in as it is without PreserveLayout.
	want := "0 HEAD\r\n1 GEDC\r\n2 VERS 5.5\r\n2 FORM LINEAGE-LINKED\r\n1 CHAR UTF-8\r\n0 @I1@ INDI\r\n0 TRLR\r\n"
	if buf.String() != want {
		t.Errorf("\n got: %q\nwant: %q", buf.String(), want)
	}
}
