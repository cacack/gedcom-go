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

// repeatedNameSource holds one name twice, in a layout a normalized encode
// would rewrite: CRLF terminators and indentation by level. The second
// cites @S1@ as the first does, and adds a nickname, a second citation and
// a vendor tag.
const repeatedNameSource = "0 HEAD\r\n" +
	"  1 GEDC\r\n" +
	"    2 VERS 5.5.1\r\n" +
	"0 @S1@ SOUR\r\n" +
	"  1 TITL Census\r\n" +
	"0 @S2@ SOUR\r\n" +
	"  1 TITL Bible\r\n" +
	"0 @I1@ INDI\r\n" +
	"  1 NAME Ann /Example/\r\n" +
	"    2 GIVN Ann\r\n" +
	"    2 SOUR @S1@\r\n" +
	"      3 PAGE 4\r\n" +
	"    2 NOTE First\r\n" +
	"  1 SEX F\r\n" +
	"  1 NAME Ann /Example/\r\n" +
	"    2 SOUR @S1@\r\n" +
	"      3 PAGE 4\r\n" +
	"    2 NICK Nan\r\n" +
	"    2 SOUR @S2@\r\n" +
	"    2 _CUSTOM kept\r\n" +
	"0 TRLR\r\n"

// TestFoldSubstructure folds the second NAME into the first: the lines the
// first lacks move under it, the identical citation is kept once, every
// line keeps its LineNumber, and a layout-preserving encode changes no
// other line.
func TestFoldSubstructure(t *testing.T) {
	doc, err := decoder.DecodeWithOptions(strings.NewReader(repeatedNameSource),
		&decoder.DecodeOptions{PreserveLayout: true})
	if err != nil {
		t.Fatal(err)
	}
	rec := doc.GetRecord("@I1@")
	lines := map[*gedcom.Tag]int{}
	for _, tag := range rec.Tags {
		lines[tag] = tag.LineNumber
	}
	names := rec.FindAll("NAME")

	if !merge.FoldSubstructure(rec, names[0], names[1]) {
		t.Fatal("FoldSubstructure = false for two names")
	}
	for _, tag := range rec.Tags {
		if tag.LineNumber != lines[tag] {
			t.Errorf("%s %s: LineNumber %d, was %d", tag.Tag, tag.Value, tag.LineNumber, lines[tag])
		}
	}
	if got := rec.FindAll("NAME", "SOUR"); len(got) != 2 {
		t.Errorf("%d citations, want @S1@ once and @S2@", len(got))
	}

	want := "0 HEAD\r\n" +
		"  1 GEDC\r\n" +
		"    2 VERS 5.5.1\r\n" +
		"0 @S1@ SOUR\r\n" +
		"  1 TITL Census\r\n" +
		"0 @S2@ SOUR\r\n" +
		"  1 TITL Bible\r\n" +
		"0 @I1@ INDI\r\n" +
		"  1 NAME Ann /Example/\r\n" +
		"    2 GIVN Ann\r\n" +
		"    2 SOUR @S1@\r\n" +
		"      3 PAGE 4\r\n" +
		"    2 SOUR @S2@\r\n" +
		"    2 NOTE First\r\n" +
		"    2 NICK Nan\r\n" +
		"    2 _CUSTOM kept\r\n" +
		"  1 SEX F\r\n" +
		"0 TRLR\r\n"
	var buf bytes.Buffer
	if err := encoder.EncodeWithOptions(&buf, doc, &encoder.EncodeOptions{PreserveLayout: true}); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != want {
		t.Errorf("encoded:\n%s\nwant:\n%s", got, want)
	}
}

// TestFoldSubstructureNested folds two citations under one event, keeping
// the PAGE once, and leaves the record's other lines alone.
func TestFoldSubstructureNested(t *testing.T) {
	rec := &gedcom.Record{Tags: []*gedcom.Tag{
		{Level: 1, Tag: "BIRT"},
		{Level: 2, Tag: "SOUR", Value: "@S1@"},
		{Level: 3, Tag: "PAGE", Value: "4"},
		{Level: 2, Tag: "SOUR", Value: "@S1@"},
		{Level: 3, Tag: "PAGE", Value: "4"},
		{Level: 3, Tag: "QUAY", Value: "3"},
		{Level: 1, Tag: "DEAT"},
	}}
	if !merge.FoldSubstructure(rec, rec.Tags[3], rec.Tags[1]) {
		t.Fatal("FoldSubstructure = false for two citations")
	}
	want := []string{"1 BIRT", "2 SOUR @S1@", "3 PAGE 4", "3 QUAY 3", "1 DEAT"}
	if got := tagLines(rec.Tags); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("Tags = %q, want %q", got, want)
	}
}

// TestFoldSubstructureRejects checks the cases that change nothing.
func TestFoldSubstructureRejects(t *testing.T) {
	rec := &gedcom.Record{Tags: []*gedcom.Tag{
		{Level: 1, Tag: "NAME", Value: "Ann"},
		{Level: 2, Tag: "SOUR", Value: "@S1@"},
		{Level: 1, Tag: "BIRT"},
		{Level: 2, Tag: "SOUR", Value: "@S1@"},
	}}
	orig := strings.Join(tagLines(rec.Tags), "|")
	name, nameSour, birt, birtSour := rec.Tags[0], rec.Tags[1], rec.Tags[2], rec.Tags[3]
	stray := &gedcom.Tag{Level: 1, Tag: "NAME"}

	tests := []struct {
		name       string
		rec        *gedcom.Record
		keep, drop *gedcom.Tag
	}{
		{"nil record", nil, name, birt},
		{"nil keep", rec, nil, birt},
		{"nil drop", rec, name, nil},
		{"same tag", rec, name, name},
		{"different levels", rec, name, birtSour},
		{"different parents", rec, nameSour, birtSour},
		{"keep not in record", rec, stray, birt},
		{"drop not in record", rec, name, stray},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if merge.FoldSubstructure(tt.rec, tt.keep, tt.drop) {
				t.Error("FoldSubstructure = true")
			}
			if got := strings.Join(tagLines(rec.Tags), "|"); got != orig {
				t.Errorf("Tags = %q, want %q", got, orig)
			}
		})
	}
}
