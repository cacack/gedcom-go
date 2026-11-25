package gedcom_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// twoNamesSource has two names with their parts, citations and a vendor
// tag, in a layout a normalized encode would rewrite: CRLF terminators,
// indentation by level and blank lines between records.
const twoNamesSource = "0 HEAD\r\n" +
	"  1 GEDC\r\n" +
	"    2 VERS 5.5.1\r\n" +
	"  1 CHAR UTF-8\r\n" +
	"\r\n" +
	"0 @S1@ SOUR\r\n" +
	"  1 TITL Census\r\n" +
	"\r\n" +
	"0 @I1@ INDI\r\n" +
	"  1 NAME Ann /Example/\r\n" +
	"    2 GIVN Ann\r\n" +
	"    2 SOUR @S1@\r\n" +
	"      3 PAGE 4\r\n" +
	"  1 NAME Annie /Example/\r\n" +
	"    2 TYPE aka\r\n" +
	"    2 NOTE Used at\r\n" +
	"      3 CONC  school\r\n" +
	"    2 _CUSTOM kept\r\n" +
	"  1 SEX F\r\n" +
	"\r\n" +
	"0 TRLR\r\n"

func decodeTwoNames(t *testing.T) (*gedcom.Document, *gedcom.Record) {
	t.Helper()
	doc, err := decoder.DecodeWithOptions(strings.NewReader(twoNamesSource),
		&decoder.DecodeOptions{PreserveLayout: true})
	if err != nil {
		t.Fatal(err)
	}
	return doc, doc.XRefMap["@I1@"]
}

func encodeLayout(t *testing.T, doc *gedcom.Document) string {
	t.Helper()
	var buf bytes.Buffer
	if err := encoder.EncodeWithOptions(&buf, doc, &encoder.EncodeOptions{PreserveLayout: true}); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// lineNumbers records each tag's LineNumber, to check that an edit left the
// tags it did not touch as they were.
func lineNumbers(tags []*gedcom.Tag) map[*gedcom.Tag]int {
	m := make(map[*gedcom.Tag]int, len(tags))
	for _, t := range tags {
		m[t] = t.LineNumber
	}
	return m
}

func checkLineNumbers(t *testing.T, rec *gedcom.Record, before map[*gedcom.Tag]int) {
	t.Helper()
	for _, tag := range rec.Tags {
		if n, ok := before[tag]; ok && tag.LineNumber != n {
			t.Errorf("%s %s: LineNumber %d, was %d", tag.Tag, tag.Value, tag.LineNumber, n)
		}
	}
}

// TestMoveSubstructureLayout moves the second NAME before the first: every
// line beneath it moves with it, and a layout-preserving encode writes the
// file with only those lines reordered.
func TestMoveSubstructureLayout(t *testing.T) {
	doc, rec := decodeTwoNames(t)
	before := lineNumbers(rec.Tags)
	names := rec.FindAll("NAME")

	if !rec.MoveSubstructure(names[1], names[0]) {
		t.Fatal("MoveSubstructure = false for two names")
	}
	checkLineNumbers(t, rec, before)
	if got := rec.FindAll("NAME"); got[0] != names[1] || got[1] != names[0] {
		t.Fatalf("names = %v, want the second first", got)
	}

	first := "  1 NAME Ann /Example/\r\n" +
		"    2 GIVN Ann\r\n" +
		"    2 SOUR @S1@\r\n" +
		"      3 PAGE 4\r\n"
	second := "  1 NAME Annie /Example/\r\n" +
		"    2 TYPE aka\r\n" +
		"    2 NOTE Used at\r\n" +
		"      3 CONC  school\r\n" +
		"    2 _CUSTOM kept\r\n"
	want := strings.Replace(twoNamesSource, first+second, second+first, 1)
	if got := encodeLayout(t, doc); got != want {
		t.Errorf("encoded:\n%s\nwant:\n%s", got, want)
	}

	// And back, by moving the first name after its last sibling.
	if !rec.MoveSubstructure(names[1], nil) {
		t.Fatal("MoveSubstructure(nil) = false")
	}
	if !rec.MoveSubstructure(rec.Find("SEX"), nil) {
		t.Fatal("MoveSubstructure(SEX) = false")
	}
	if got := encodeLayout(t, doc); got != twoNamesSource {
		t.Errorf("moved back:\n%s\nwant the source", got)
	}
}

// TestMoveSubstructureNested moves a level-2 line within its parent, and to
// the end of its parent when before is nil, leaving the next structure in
// place.
func TestMoveSubstructureNested(t *testing.T) {
	_, rec := decodeTwoNames(t)
	aka := rec.FindAll("NAME")[1]
	note := rec.Find("NAME", "NOTE")
	typ := rec.Find("NAME", "TYPE")

	if !rec.MoveSubstructure(typ, nil) {
		t.Fatal("MoveSubstructure(TYPE, nil) = false")
	}
	got := tagLines(rec.Subtree(aka))
	want := []string{"1 NAME Annie /Example/", "2 NOTE Used at", "3 CONC  school", "2 _CUSTOM kept", "2 TYPE aka"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("NAME = %q, want %q", got, want)
	}
	if last := rec.Tags[len(rec.Tags)-1]; last.Tag != "SEX" {
		t.Errorf("last tag = %s, want SEX", last.Tag)
	}
	if !rec.MoveSubstructure(note, note) {
		t.Error("MoveSubstructure(t, t) = false")
	}
}

// TestMoveSubstructureRejects checks the cases that change nothing.
func TestMoveSubstructureRejects(t *testing.T) {
	_, rec := decodeTwoNames(t)
	names := rec.FindAll("NAME")
	givn := rec.Find("NAME", "GIVN")
	typ := rec.Find("NAME", "TYPE")
	conc := rec.Find("NAME", "NOTE", "CONC")
	stray := &gedcom.Tag{Level: 1, Tag: "NAME"}
	orig := append([]*gedcom.Tag(nil), rec.Tags...)
	var none *gedcom.Record

	tests := []struct {
		name      string
		rec       *gedcom.Record
		t, before *gedcom.Tag
	}{
		{"nil record", none, names[0], nil},
		{"nil tag", rec, nil, names[0]},
		{"tag not in record", rec, stray, names[0]},
		{"before not in record", rec, names[1], stray},
		{"before at another level", rec, givn, names[0]},
		{"before under another parent", rec, typ, givn},
		{"continuation line", rec, conc, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.rec.MoveSubstructure(tt.t, tt.before) {
				t.Error("MoveSubstructure = true")
			}
			if strings.Join(tagLines(rec.Tags), "|") != strings.Join(tagLines(orig), "|") {
				t.Error("record changed")
			}
		})
	}
}

// TestSetChild replaces the first child's value, dropping its old CONC line,
// and adds a child the structure lacks, under that structure only; a
// layout-preserving encode changes only those lines.
func TestSetChild(t *testing.T) {
	doc, rec := decodeTwoNames(t)
	before := lineNumbers(rec.Tags)
	aka := rec.FindAll("NAME")[1]
	note := rec.Find("NAME", "NOTE")

	if got := rec.SetChild(aka, "NOTE", "Used at work", gedcom.Version551); got != note {
		t.Fatalf("SetChild(NOTE) = %v, want the existing NOTE", got)
	}
	nick := rec.SetChild(aka, "NICK", "Nan", gedcom.Version551)
	if nick == nil || nick.Level != 2 || nick.LineNumber != 0 {
		t.Fatalf("SetChild(NICK) = %+v, want a new level-2 tag", nick)
	}
	if got := rec.SetChild(nil, "SEX", "F", gedcom.Version551); got != rec.Find("SEX") {
		t.Errorf("SetChild(nil, SEX) = %v, want the existing SEX", got)
	}
	checkLineNumbers(t, rec, before)

	want := strings.Replace(twoNamesSource,
		"    2 NOTE Used at\r\n      3 CONC  school\r\n    2 _CUSTOM kept\r\n",
		"    2 NOTE Used at work\r\n    2 _CUSTOM kept\r\n2 NICK Nan\r\n", 1)
	if got := encodeLayout(t, doc); got != want {
		t.Errorf("encoded:\n%s\nwant:\n%s", got, want)
	}

	// A value with a line break gets a CONT; a new level-1 line goes last.
	rec.SetChild(aka, "NOTE", "Used at\nwork", gedcom.Version551)
	if got := tagLines(rec.Subtree(note)); strings.Join(got, "|") != "2 NOTE Used at|3 CONT work" {
		t.Errorf("NOTE = %q", got)
	}
	rin := rec.SetChild(nil, "RIN", "7", gedcom.Version551)
	if rec.Tags[len(rec.Tags)-1] != rin || rin.Level != 1 {
		t.Errorf("RIN = %+v, want the record's last line at level 1", rin)
	}
}

// TestSetChildRejects checks the cases that set nothing.
func TestSetChildRejects(t *testing.T) {
	_, rec := decodeTwoNames(t)
	name := rec.Find("NAME")
	stray := &gedcom.Tag{Level: 1, Tag: "NAME"}
	var none *gedcom.Record
	n := len(rec.Tags)
	for _, tt := range []struct {
		name   string
		rec    *gedcom.Record
		parent *gedcom.Tag
		tag    string
	}{
		{"nil record", none, name, "NICK"},
		{"parent not in record", rec, stray, "NICK"},
		{"empty tag", rec, name, ""},
		{"CONT", rec, name, "CONT"},
		{"CONC", rec, name, "CONC"},
	} {
		if got := tt.rec.SetChild(tt.parent, tt.tag, "x", gedcom.Version551); got != nil {
			t.Errorf("%s: SetChild = %+v, want nil", tt.name, got)
		}
	}
	if len(rec.Tags) != n {
		t.Errorf("len(Tags) = %d, want %d", len(rec.Tags), n)
	}
}

// TestRemoveChildren removes a child with its subtree, under that structure
// only; a layout-preserving encode drops only those lines.
func TestRemoveChildren(t *testing.T) {
	doc, rec := decodeTwoNames(t)
	before := lineNumbers(rec.Tags)
	aka := rec.FindAll("NAME")[1]

	if got := rec.RemoveChildren(aka, "NOTE"); got != 1 {
		t.Fatalf("RemoveChildren(NOTE) = %d, want 1", got)
	}
	if got := rec.RemoveChildren(aka, "SOUR"); got != 0 {
		t.Errorf("RemoveChildren(SOUR) under the second name = %d, want 0", got)
	}
	if rec.Find("NAME", "SOUR") == nil {
		t.Error("the first name's SOUR was removed")
	}
	if got := rec.RemoveChildren(aka, "NOTE"); got != 0 {
		t.Errorf("second RemoveChildren(NOTE) = %d, want 0", got)
	}
	checkLineNumbers(t, rec, before)

	want := strings.Replace(twoNamesSource, "    2 NOTE Used at\r\n      3 CONC  school\r\n", "", 1)
	if got := encodeLayout(t, doc); got != want {
		t.Errorf("encoded:\n%s\nwant:\n%s", got, want)
	}

	if got := rec.RemoveChildren(nil, "NAME"); got != 2 {
		t.Errorf("RemoveChildren(nil, NAME) = %d, want 2", got)
	}
	if got := strings.Join(tagLines(rec.Tags), "|"); got != "1 SEX F" {
		t.Errorf("Tags = %q, want only SEX", got)
	}
	var none *gedcom.Record
	if none.RemoveChildren(nil, "SEX") != 0 || rec.RemoveChildren(&gedcom.Tag{Level: 1}, "SEX") != 0 {
		t.Error("RemoveChildren removed from a nil record or under a parent not in the record")
	}
}

// TestRemoveChildrenContinuation checks that CONT and CONC lines, which
// continue the parent's value, are not removed.
func TestRemoveChildrenContinuation(t *testing.T) {
	_, rec := decodeTwoNames(t)
	note := rec.Find("NAME", "NOTE")
	for _, tag := range []string{"CONC", "CONT"} {
		if got := rec.RemoveChildren(note, tag); got != 0 {
			t.Errorf("RemoveChildren(NOTE, %s) = %d, want 0", tag, got)
		}
	}
	if got := rec.FullValue(note); got != "Used at school" {
		t.Errorf("FullValue(NOTE) = %q, want %q", got, "Used at school")
	}
}

func tagLines(tags []*gedcom.Tag) []string {
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		out = append(out, strings.TrimSpace(fmt.Sprint(t.Level, " ", t.Tag, " ", t.Value)))
	}
	return out
}
