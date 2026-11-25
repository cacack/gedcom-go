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

// repeatTree builds a document of one individual @I1@ from its level-1
// lines.
func repeatTree(lines string) string {
	return "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @S1@ SOUR\n1 TITL Census\n0 @I1@ INDI\n" + lines + "0 TRLR\n"
}

func groupLines(groups []merge.EventGroup) [][][]string {
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

func mustFoldRepeated(t *testing.T, doc *gedcom.Document, xrefs ...string) (*gedcom.Document, *merge.FoldReport) {
	t.Helper()
	out, report, err := merge.FoldRepeatedEvents(doc, xrefs...)
	if err != nil {
		t.Fatalf("FoldRepeatedEvents: %v", err)
	}
	return out, report
}

func TestRepeatedEvents(t *testing.T) {
	tests := []struct {
		name   string
		lines  string
		groups [][][]string
		folded []string
	}{
		{
			name:  "equal dates written differently, same place",
			lines: "1 BIRT\n2 DATE 7 DEC 1977\n2 PLAC Springfield\n2 SOUR @S1@\n1 BIRT\n2 DATE 07 Dec 1977\n2 PLAC Springfield\n2 NOTE Family bible.\n",
			groups: [][][]string{{
				{"1 BIRT", "2 DATE 7 DEC 1977", "2 PLAC Springfield", "2 SOUR @S1@"},
				{"1 BIRT", "2 DATE 07 Dec 1977", "2 PLAC Springfield", "2 NOTE Family bible."},
			}},
			folded: []string{"1 BIRT", "2 DATE 7 DEC 1977", "2 PLAC Springfield", "2 SOUR @S1@", "2 NOTE Family bible."},
		},
		{
			name:   "different dates",
			lines:  "1 BIRT\n2 DATE 7 DEC 1977\n1 BIRT\n2 DATE 8 DEC 1977\n",
			folded: []string{"1 BIRT", "2 DATE 7 DEC 1977", "1 BIRT", "2 DATE 8 DEC 1977"},
		},
		{
			name:   "undated RESI differing only in ADDR",
			lines:  "1 RESI\n2 ADDR 1 Main St\n1 RESI\n2 ADDR 2 Elm St\n",
			folded: []string{"1 RESI", "2 ADDR 1 Main St", "1 RESI", "2 ADDR 2 Elm St"},
		},
		{
			name:   "different places",
			lines:  "1 BIRT\n2 DATE 7 DEC 1977\n2 PLAC Springfield\n1 BIRT\n2 DATE 7 DEC 1977\n2 PLAC Shelbyville\n",
			folded: []string{"1 BIRT", "2 DATE 7 DEC 1977", "2 PLAC Springfield", "1 BIRT", "2 DATE 7 DEC 1977", "2 PLAC Shelbyville"},
		},
		{
			name:  "Y and empty value",
			lines: "1 DEAT Y\n2 DATE 2000\n1 DEAT\n2 DATE 2000\n2 SOUR @S1@\n",
			groups: [][][]string{{
				{"1 DEAT Y", "2 DATE 2000"},
				{"1 DEAT", "2 DATE 2000", "2 SOUR @S1@"},
			}},
			folded: []string{"1 DEAT Y", "2 DATE 2000", "2 SOUR @S1@"},
		},
		{
			name:   "CHAN not an event",
			lines:  "1 CHAN\n2 DATE 7 DEC 1977\n1 CHAN\n2 DATE 7 DEC 1977\n",
			folded: []string{"1 CHAN", "2 DATE 7 DEC 1977", "1 CHAN", "2 DATE 7 DEC 1977"},
		},
		{
			// The place only the second has is folded into the first, so
			// the third, at another place, no longer records its event.
			name:  "compared with the first after folds",
			lines: "1 BIRT\n2 DATE 1977\n1 BIRT\n2 DATE 1977\n2 PLAC Springfield\n1 BIRT\n2 DATE 1977\n2 PLAC Shelbyville\n",
			groups: [][][]string{{
				{"1 BIRT", "2 DATE 1977"},
				{"1 BIRT", "2 DATE 1977", "2 PLAC Springfield"},
			}},
			folded: []string{"1 BIRT", "2 DATE 1977", "2 PLAC Springfield", "1 BIRT", "2 DATE 1977", "2 PLAC Shelbyville"},
		},
		{
			name:  "groups in order of first member",
			lines: "1 BIRT\n2 DATE 1977\n1 DEAT\n2 DATE 2000\n1 DEAT\n2 DATE 2000\n2 NOTE a\n1 BIRT\n2 DATE 1977\n2 NOTE b\n1 BIRT\n2 DATE 1977\n2 NOTE c\n",
			groups: [][][]string{
				{{"1 BIRT", "2 DATE 1977"}, {"1 BIRT", "2 DATE 1977", "2 NOTE b"}, {"1 BIRT", "2 DATE 1977", "2 NOTE c"}},
				{{"1 DEAT", "2 DATE 2000"}, {"1 DEAT", "2 DATE 2000", "2 NOTE a"}},
			},
			folded: []string{"1 BIRT", "2 DATE 1977", "2 NOTE b", "2 NOTE c", "1 DEAT", "2 DATE 2000", "2 NOTE a"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := decodeZip(t, repeatTree(tt.lines))
			groups := merge.RepeatedEvents(doc)
			if got := groupLines(groups); !reflect.DeepEqual(got, tt.groups) {
				t.Errorf("groups =\n%q\nwant\n%q", got, tt.groups)
			}
			for _, g := range groups {
				if g.XRef != "@I1@" {
					t.Errorf("group XRef = %q, want @I1@", g.XRef)
				}
			}
			out, report := mustFoldRepeated(t, doc)
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

// TestRepeatedEvents_Families checks family records are covered, and the
// typed view follows the folded lines.
func TestRepeatedEvents_Families(t *testing.T) {
	doc := decodeZip(t, "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @S1@ SOUR\n1 TITL Census\n"+
		"0 @F1@ FAM\n1 MARR\n2 DATE 1 JUN 1925\n2 SOUR @S1@\n1 MARR\n2 DATE 01 JUN 1925\n2 PLAC Springfield\n0 TRLR\n")
	if got := groupLines(merge.RepeatedEvents(doc)); len(got) != 1 || len(got[0]) != 2 {
		t.Fatalf("groups = %q, want one of two MARR", got)
	}
	out, report := mustFoldRepeated(t, doc)
	if want := []string{"1 MARR", "2 DATE 1 JUN 1925", "2 SOUR @S1@", "2 PLAC Springfield"}; !reflect.DeepEqual(recordLines(t, out, "@F1@"), want) {
		t.Errorf("@F1@ = %q, want %q", recordLines(t, out, "@F1@"), want)
	}
	if len(report.Folded) != 1 || report.Folded[0].XRef != "@F1@" {
		t.Errorf("Folded = %+v, want one fold of @F1@", report.Folded)
	}
	if ev := out.GetFamily("@F1@").Events; len(ev) != 1 || ev[0].PlaceName() != "Springfield" {
		t.Errorf("typed events = %+v, want one MARR at Springfield", ev)
	}
	if ev := doc.GetFamily("@F1@").Events; len(ev) != 2 {
		t.Errorf("doc's typed events = %d, want 2: doc was modified", len(ev))
	}
}

// TestFoldRepeatedEvents_KeepsEveryLine checks every citation and note line
// of every member survives, and the report lists one fold per member
// removed, labeled with its record.
func TestFoldRepeatedEvents_KeepsEveryLine(t *testing.T) {
	text := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @S1@ SOUR\n1 TITL Census\n0 @S2@ SOUR\n1 TITL Bible\n" +
		"0 @I1@ INDI\n1 NAME Ann /Example/\n" +
		"1 BIRT\n2 DATE 7 DEC 1977\n2 SOUR @S1@\n3 PAGE 12\n" +
		"1 BIRT\n2 DATE 07 Dec 1977\n2 SOUR @S1@\n3 PAGE 13\n2 NOTE First.\n" +
		"1 BIRT\n2 DATE 7 Dec 1977\n2 SOUR @S2@\n2 NOTE Second.\n" +
		"0 @I2@ INDI\n1 DEAT\n2 DATE 2000\n2 NOTE Third.\n1 DEAT\n2 DATE 2000\n2 SOUR @S2@\n0 TRLR\n"
	doc := decodeZip(t, text)
	out, report := mustFoldRepeated(t, doc)

	want := map[string][]string{
		"@I1@": {"1 NAME Ann /Example/", "1 BIRT", "2 DATE 7 DEC 1977", "2 SOUR @S1@", "3 PAGE 12", "2 SOUR @S1@", "3 PAGE 13", "2 SOUR @S2@", "2 NOTE First.", "2 NOTE Second."},
		"@I2@": {"1 DEAT", "2 DATE 2000", "2 NOTE Third.", "2 SOUR @S2@"},
	}
	for xref, lines := range want {
		if got := recordLines(t, out, xref); !reflect.DeepEqual(got, lines) {
			t.Errorf("%s =\n%q\nwant\n%q", xref, got, lines)
		}
	}

	var got []string
	for _, f := range report.Folded {
		got = append(got, f.XRef+" "+strings.Join(tagLines(f.Drop.Tags), "|"))
	}
	wantFolds := []string{
		"@I1@ 1 BIRT|2 DATE 07 Dec 1977|2 SOUR @S1@|3 PAGE 13|2 NOTE First.",
		"@I1@ 1 BIRT|2 DATE 7 Dec 1977|2 SOUR @S2@|2 NOTE Second.",
		"@I2@ 1 DEAT|2 DATE 2000|2 SOUR @S2@",
	}
	if !reflect.DeepEqual(got, wantFolds) {
		t.Errorf("Folded =\n%q\nwant\n%q", got, wantFolds)
	}
	if into := tagLines(report.Folded[0].Into.Tags); !reflect.DeepEqual(into, want["@I1@"][1:]) {
		t.Errorf("Folded[0].Into = %q, want %q", into, want["@I1@"][1:])
	}
	if out.GetIndividual("@I1@").Events[0].SourceCitations == nil || len(out.GetIndividual("@I1@").Events) != 1 {
		t.Errorf("typed @I1@ events = %+v, want one BIRT with citations", out.GetIndividual("@I1@").Events)
	}
}

// TestFoldRepeatedEvents_Unchanged checks doc is not modified, and a record
// with no group keeps its Tags, so a layout-preserving encode writes it
// byte-identical.
func TestFoldRepeatedEvents_Unchanged(t *testing.T) {
	unchanged := "0 @I2@ INDI\r\n1 NAME Bob /Example/\r\n\r\n1 BIRT\r\n2 DATE 1950\r\n1 BIRT\r\n2 DATE 1951\r\n"
	text := "0 HEAD\r\n1 GEDC\r\n2 VERS 5.5.1\r\n1 CHAR UTF-8\r\n" +
		"0 @I1@ INDI\r\n1 BIRT\r\n2 DATE 1977\r\n1 BIRT\r\n2 DATE 1977\r\n2 NOTE a\r\n" +
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
	beforeTags := tagLines(doc.GetRecord("@I1@").Tags)

	out, report := mustFoldRepeated(t, doc)
	if len(report.Folded) != 1 {
		t.Fatalf("Folded = %+v, want one", report.Folded)
	}
	if !reflect.DeepEqual(out.GetRecord("@I2@").Tags, doc.GetRecord("@I2@").Tags) {
		t.Error("@I2@ Tags changed")
	}
	if got := encode(out); !strings.Contains(got, unchanged) {
		t.Errorf("encoded output does not hold @I2@ byte-identical:\n%q", got)
	}
	if encode(doc) != before || !reflect.DeepEqual(tagLines(doc.GetRecord("@I1@").Tags), beforeTags) {
		t.Error("doc was modified")
	}
}

// TestFoldRepeatedEvents_Idempotent checks folding the output folds
// nothing, and RepeatedEvents finds no group in it.
func TestFoldRepeatedEvents_Idempotent(t *testing.T) {
	doc := decodeZip(t, repeatTree("1 BIRT\n2 DATE 1977\n1 BIRT\n2 DATE 1977\n2 PLAC Springfield\n"+
		"1 BIRT\n2 DATE 1977\n2 PLAC Shelbyville\n1 BIRT\n2 DATE ABT 1977\n1 BIRT Y\n2 DATE 1977\n3 TIME 10:00\n"+
		"1 RESI\n2 ADDR 1 Main St\n1 RESI\n2 ADDR 1 Main St\n"))
	once, _ := mustFoldRepeated(t, doc)
	if groups := merge.RepeatedEvents(once); len(groups) != 0 {
		t.Errorf("RepeatedEvents on output = %q, want none", groupLines(groups))
	}
	twice, report := mustFoldRepeated(t, once)
	if len(report.Folded) != 0 {
		t.Errorf("second fold Folded = %+v, want none", report.Folded)
	}
	if !reflect.DeepEqual(recordLines(t, twice, "@I1@"), recordLines(t, once, "@I1@")) {
		t.Errorf("second fold changed @I1@: %q", recordLines(t, twice, "@I1@"))
	}
}

func TestRepeatedEvents_XRefs(t *testing.T) {
	doc := decodeZip(t, "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n"+
		"0 @I1@ INDI\n1 BIRT\n2 DATE 1977\n1 BIRT\n2 DATE 1977\n"+
		"0 @I2@ INDI\n1 DEAT\n2 DATE 2000\n1 DEAT\n2 DATE 2000\n0 TRLR\n")
	if groups := merge.RepeatedEvents(doc, "@I2@"); len(groups) != 1 || groups[0].XRef != "@I2@" {
		t.Errorf("RepeatedEvents(@I2@) = %+v, want one group of @I2@", groups)
	}
	if groups := merge.RepeatedEvents(doc, "@X9@"); len(groups) != 0 {
		t.Errorf("RepeatedEvents(@X9@) = %+v, want none", groups)
	}
	if groups := merge.RepeatedEvents(nil); groups != nil {
		t.Errorf("RepeatedEvents(nil) = %+v, want nil", groups)
	}
	out, report := mustFoldRepeated(t, doc, "@I1@")
	if len(report.Folded) != 1 || report.Folded[0].XRef != "@I1@" || len(recordLines(t, out, "@I2@")) != 4 {
		t.Errorf("Folded = %+v, @I2@ = %q; want @I1@ folded alone", report.Folded, recordLines(t, out, "@I2@"))
	}
}

func TestFoldRepeatedEvents_Errors(t *testing.T) {
	doc := decodeZip(t, "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @I1@ INDI\n1 SEX F\n0 @S1@ SOUR\n1 TITL Census\n0 TRLR\n")
	doc.Records = append(doc.Records, &gedcom.Record{XRef: "@I9@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: "@I9@"}})

	if _, _, err := merge.FoldRepeatedEvents(nil); !errors.Is(err, merge.ErrNilDocument) {
		t.Errorf("nil doc: err = %v, want ErrNilDocument", err)
	}
	tests := []struct {
		xref    string
		unknown bool
	}{
		{"@X1@", true},
		{"@S1@", false},
		{"@I9@", false},
	}
	for _, tt := range tests {
		out, report, err := merge.FoldRepeatedEvents(doc, "@I1@", tt.xref)
		if !errors.Is(err, merge.ErrInvalidMerge) || errors.Is(err, gedcom.ErrUnknownXRef) != tt.unknown || out != nil || report != nil {
			t.Errorf("%s: out = %v, report = %v, err = %v", tt.xref, out, report, err)
		}
	}
	// A hand-built record not named is skipped.
	if _, report := mustFoldRepeated(t, doc); len(report.Folded) != 0 {
		t.Errorf("Folded = %+v, want none", report.Folded)
	}
}
