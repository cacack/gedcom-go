package merge_test

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/merge"
)

func nameGroupLines(groups []merge.NameGroup) [][][]string {
	var out [][][]string
	for _, g := range groups {
		var members [][]string
		for _, m := range g.Members {
			members = append(members, tagLines(m.Tags))
		}
		out = append(out, members)
	}
	return out
}

func mustFoldNames(t *testing.T, doc *gedcom.Document, xrefs ...string) (*gedcom.Document, *merge.FoldReport) {
	t.Helper()
	out, report, err := merge.FoldRepeatedNames(doc, xrefs...)
	if err != nil {
		t.Fatalf("FoldRepeatedNames: %v", err)
	}
	return out, report
}

func TestRepeatedNames(t *testing.T) {
	tests := []struct {
		name   string
		lines  string
		groups [][][]string
		folded []string
	}{
		{
			name:  "case, spacing and punctuation",
			lines: "1 NAME John /Smith/ Jr.\n2 SOUR @S1@\n1 NAME john  smith, jr\n2 NOTE From the census.\n",
			groups: [][][]string{{
				{"1 NAME John /Smith/ Jr.", "2 SOUR @S1@"},
				{"1 NAME john  smith, jr", "2 NOTE From the census."},
			}},
			folded: []string{"1 NAME John /Smith/ Jr.", "2 SOUR @S1@", "2 NOTE From the census."},
		},
		{
			name:   "different names",
			lines:  "1 NAME John /Smith/\n1 NAME Jon /Smith/\n",
			folded: []string{"1 NAME John /Smith/", "1 NAME Jon /Smith/"},
		},
		{
			name:   "empty values",
			lines:  "1 NAME\n2 GIVN John\n1 NAME\n2 SURN Smith\n",
			folded: []string{"1 NAME", "2 GIVN John", "1 NAME", "2 SURN Smith"},
		},
		{
			name:  "a piece only one has moves over",
			lines: "1 NAME John /Smith/\n2 GIVN John\n1 NAME John /Smith/\n2 SURN Smith\n2 NICK Jack\n",
			groups: [][][]string{{
				{"1 NAME John /Smith/", "2 GIVN John"},
				{"1 NAME John /Smith/", "2 SURN Smith", "2 NICK Jack"},
			}},
			folded: []string{"1 NAME John /Smith/", "2 GIVN John", "2 SURN Smith", "2 NICK Jack"},
		},
		{
			name:   "pieces written differently",
			lines:  "1 NAME John /Smith/\n2 GIVN John\n1 NAME john /smith/\n2 GIVN john\n",
			folded: []string{"1 NAME John /Smith/", "2 GIVN John", "1 NAME john /smith/", "2 GIVN john"},
		},
		{
			name:   "different TYPE",
			lines:  "1 NAME Mary /Smith/\n2 TYPE birth\n1 NAME Mary /Smith/\n2 TYPE married\n",
			folded: []string{"1 NAME Mary /Smith/", "2 TYPE birth", "1 NAME Mary /Smith/", "2 TYPE married"},
		},
		{
			name:   "TYPE on one only",
			lines:  "1 NAME Mary /Smith/\n1 NAME Mary /Smith/\n2 TYPE married\n",
			folded: []string{"1 NAME Mary /Smith/", "1 NAME Mary /Smith/", "2 TYPE married"},
		},
		{
			name:  "same TYPE",
			lines: "1 NAME Mary /Smith/\n2 TYPE married\n1 NAME Mary Smith\n2 TYPE married\n2 SOUR @S1@\n",
			groups: [][][]string{{
				{"1 NAME Mary /Smith/", "2 TYPE married"},
				{"1 NAME Mary Smith", "2 TYPE married", "2 SOUR @S1@"},
			}},
			folded: []string{"1 NAME Mary /Smith/", "2 TYPE married", "2 SOUR @S1@"},
		},
		{
			name:   "a piece repeated under one",
			lines:  "1 NAME John /Smith/\n2 GIVN John\n2 GIVN Jack\n1 NAME John /Smith/\n",
			folded: []string{"1 NAME John /Smith/", "2 GIVN John", "2 GIVN Jack", "1 NAME John /Smith/"},
		},
		{
			name:  "three copies, the third after another tag",
			lines: "1 NAME Ann /Example/\n1 SEX F\n1 NAME ann /example/\n2 SOUR @S1@\n1 NAME Ann Example\n2 SOUR @S1@\n",
			groups: [][][]string{{
				{"1 NAME Ann /Example/"},
				{"1 NAME ann /example/", "2 SOUR @S1@"},
				{"1 NAME Ann Example", "2 SOUR @S1@"},
			}},
			folded: []string{"1 NAME Ann /Example/", "2 SOUR @S1@", "1 SEX F"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := decodeZip(t, repeatTree(tt.lines))
			groups := merge.RepeatedNames(doc)
			if got := nameGroupLines(groups); !reflect.DeepEqual(got, tt.groups) {
				t.Errorf("groups =\n%q\nwant\n%q", got, tt.groups)
			}
			for _, g := range groups {
				if g.XRef != "@I1@" {
					t.Errorf("group XRef = %q, want @I1@", g.XRef)
				}
			}
			out, report := mustFoldNames(t, doc)
			if got := recordLines(t, out, "@I1@"); !reflect.DeepEqual(got, tt.folded) {
				t.Errorf("folded @I1@ =\n%q\nwant\n%q", got, tt.folded)
			}
			want := 0
			for _, g := range tt.groups {
				want += len(g) - 1
			}
			if len(report.Folded) != want {
				t.Errorf("len(Folded) = %d, want %d", len(report.Folded), want)
			}
		})
	}
}

// TestRepeatedNames_NoneInFile checks a file with no repeated name, events
// repeated or not, has no group.
func TestRepeatedNames_NoneInFile(t *testing.T) {
	doc := decodeZip(t, "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n"+
		"0 @I1@ INDI\n1 NAME John /Smith/\n1 NAME Johnny /Smith/\n1 BIRT\n2 DATE 1900\n1 BIRT\n2 DATE 1900\n"+
		"0 @I2@ INDI\n1 NAME Ann /Smith/\n0 @F1@ FAM\n1 HUSB @I1@\n0 TRLR\n")
	if groups := merge.RepeatedNames(doc); len(groups) != 0 {
		t.Errorf("RepeatedNames = %q, want none", nameGroupLines(groups))
	}
}

// TestFoldRepeatedNames_Citations folds two copies with different
// citations, and one both hold, into one NAME holding each distinct
// citation once; a layout-preserving encode writes every other line
// unchanged, and doc is not modified.
func TestFoldRepeatedNames_Citations(t *testing.T) {
	head := "0 HEAD\r\n1 GEDC\r\n2 VERS 5.5.1\r\n1 CHAR UTF-8\r\n0 @S1@ SOUR\r\n1 TITL Census\r\n0 @S2@ SOUR\r\n1 TITL Bible\r\n"
	other := "1 SEX M\r\n\r\n1 BIRT\r\n2 DATE  1 JAN 1900\r\n"
	unchanged := "0 @I2@ INDI\r\n1 NAME Bob /Example/\r\n1 NAME Bob /Example/\r\n2 TYPE aka\r\n"
	text := head +
		"0 @I1@ INDI\r\n1 NAME John /Smith/ Jr.\r\n2 SOUR @S1@\r\n3 PAGE 12\r\n" + other +
		"1 NAME john  smith, jr\r\n2 SOUR @S1@\r\n3 PAGE 12\r\n2 SOUR @S2@\r\n3 PAGE 3\r\n" +
		unchanged + "0 TRLR\r\n"
	doc, err := decoder.DecodeWithOptions(strings.NewReader(text), &decoder.DecodeOptions{PreserveLayout: true})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	encode := func(d *gedcom.Document) string {
		var buf bytes.Buffer
		if err := encoder.EncodeWithOptions(&buf, d, &encoder.EncodeOptions{PreserveLayout: true}); err != nil {
			t.Fatalf("encode: %v", err)
		}
		return buf.String()
	}
	before := encode(doc)
	beforeDoc := doc.Clone()

	out, report := mustFoldNames(t, doc)
	want := head +
		"0 @I1@ INDI\r\n1 NAME John /Smith/ Jr.\r\n2 SOUR @S1@\r\n3 PAGE 12\r\n2 SOUR @S2@\r\n3 PAGE 3\r\n" + other +
		unchanged + "0 TRLR\r\n"
	if got := encode(out); got != want {
		t.Errorf("encoded =\n%q\nwant\n%q", got, want)
	}
	if len(report.Folded) != 1 || report.Folded[0].XRef != "@I1@" {
		t.Errorf("Folded = %+v, want one fold of @I1@", report.Folded)
	}
	if names := out.GetIndividual("@I1@").Names; len(names) != 1 || len(names[0].SourceCitations) != 2 {
		t.Errorf("typed names = %+v, want one with two citations", names)
	}
	if encode(doc) != before || !reflect.DeepEqual(doc, beforeDoc) {
		t.Error("doc was modified")
	}
	if !reflect.DeepEqual(out.GetRecord("@I2@").Tags, doc.GetRecord("@I2@").Tags) {
		t.Error("@I2@ Tags changed")
	}
}

// TestFoldRepeatedNames_Idempotent checks folding the output folds nothing.
func TestFoldRepeatedNames_Idempotent(t *testing.T) {
	doc := decodeZip(t, repeatTree("1 NAME Ann /Example/\n1 NAME ann example\n2 GIVN Ann\n"+
		"1 NAME Ann /Example/\n2 GIVN Anne\n1 NAME Ann /Example/\n2 SOUR @S1@\n"))
	once, report := mustFoldNames(t, doc)
	if len(report.Folded) != 2 {
		t.Fatalf("first fold Folded = %d, want 2", len(report.Folded))
	}
	if groups := merge.RepeatedNames(once); len(groups) != 0 {
		t.Errorf("RepeatedNames on output = %q, want none", nameGroupLines(groups))
	}
	twice, report := mustFoldNames(t, once)
	if len(report.Folded) != 0 || !reflect.DeepEqual(recordLines(t, twice, "@I1@"), recordLines(t, once, "@I1@")) {
		t.Errorf("second fold Folded = %+v, @I1@ = %q", report.Folded, recordLines(t, twice, "@I1@"))
	}
}

func TestRepeatedNames_XRefs(t *testing.T) {
	doc := decodeZip(t, "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n"+
		"0 @I1@ INDI\n1 NAME A /B/\n1 NAME a b\n0 @I2@ INDI\n1 NAME C /D/\n1 NAME c d\n0 TRLR\n")
	if groups := merge.RepeatedNames(doc, "@I2@"); len(groups) != 1 || groups[0].XRef != "@I2@" {
		t.Errorf("RepeatedNames(@I2@) = %+v, want one group of @I2@", groups)
	}
	if groups := merge.RepeatedNames(doc, "@X9@"); len(groups) != 0 {
		t.Errorf("RepeatedNames(@X9@) = %+v, want none", groups)
	}
	if groups := merge.RepeatedNames(nil); groups != nil {
		t.Errorf("RepeatedNames(nil) = %+v, want nil", groups)
	}
	out, report := mustFoldNames(t, doc, "@I1@")
	if len(report.Folded) != 1 || report.Folded[0].XRef != "@I1@" || len(recordLines(t, out, "@I2@")) != 2 {
		t.Errorf("Folded = %+v, @I2@ = %q; want @I1@ folded alone", report.Folded, recordLines(t, out, "@I2@"))
	}
}

func TestFoldRepeatedNames_Errors(t *testing.T) {
	doc := decodeZip(t, "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @I1@ INDI\n1 SEX F\n0 @F1@ FAM\n1 WIFE @I1@\n0 @S1@ SOUR\n1 TITL Census\n0 TRLR\n")
	doc.Records = append(doc.Records, &gedcom.Record{XRef: "@I9@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: "@I9@"}})

	if _, _, err := merge.FoldRepeatedNames(nil); !errors.Is(err, merge.ErrNilDocument) {
		t.Errorf("nil doc: err = %v, want ErrNilDocument", err)
	}
	tests := []struct {
		xref    string
		unknown bool
	}{
		{"@X1@", true},
		{"@F1@", false},
		{"@S1@", false},
		{"@I9@", false},
	}
	for _, tt := range tests {
		out, report, err := merge.FoldRepeatedNames(doc, "@I1@", tt.xref)
		if !errors.Is(err, merge.ErrInvalidMerge) || errors.Is(err, gedcom.ErrUnknownXRef) != tt.unknown || out != nil || report != nil {
			t.Errorf("%s: out = %v, report = %v, err = %v", tt.xref, out, report, err)
		}
	}
	if _, report := mustFoldNames(t, doc); len(report.Folded) != 0 {
		t.Errorf("Folded = %+v, want none", report.Folded)
	}
}
