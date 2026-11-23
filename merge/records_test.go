package merge_test

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/merge"
	"github.com/cacack/gedcom-go/v3/validator"
)

// treeA and treeB are two overlapping synthetic trees: one couple and a
// daughter in both, under different XRefs, with treeB adding a death, a
// note, a second name, a marriage, a son and pointers of other kinds.
const treeA = `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
1 SUBM @U1@
0 @U1@ SUBM
1 NAME Example Submitter
0 @I1@ INDI
1 NAME John /Example/
1 SEX M
1 BIRT
2 DATE 1 JAN 1900
2 PLAC Springfield
1 FAMS @F1@
0 @I2@ INDI
1 NAME Mary /Sample/
1 SEX F
1 FAMS @F1@
0 @I3@ INDI
1 NAME Ann /Example/
1 SEX F
1 FAMC @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
0 TRLR
`

const treeB = `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
1 SUBM @U2@
0 @U2@ SUBM
1 NAME Example Submitter
0 @I11@ INDI
1 NAME John /Example/
1 SEX M
1 BIRT
2 DATE 1 JAN 1900
2 PLAC Springfield
1 DEAT
2 DATE 1970
1 NOTE Seen in a census.
1 FAMS @F11@
0 @I12@ INDI
1 NAME Mary /Sample/
1 NAME Mary /Example/
1 SEX F
1 FAMS @F11@
0 @I13@ INDI
1 NAME Ann /Example/
1 SEX F
1 FAMC @F11@
0 @I14@ INDI
1 NAME Paul /Example/
1 SEX M
1 FAMC @F11@
1 ASSO @I11@
2 RELA Godfather
1 _LINK @I11@
0 @F11@ FAM
1 HUSB @I11@
1 WIFE @I12@
1 CHIL @I13@
1 CHIL @I14@
1 MARR
2 DATE 1925
0 TRLR
`

func overlappingTrees(t *testing.T) *gedcom.Document {
	t.Helper()
	doc, _, err := merge.Combine(decodeZip(t, treeA), decodeZip(t, treeB), merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}
	return doc
}

// pointersTo returns every typed or raw pointer to xref left in doc.
func pointersTo(doc *gedcom.Document, xref string) int {
	n := 0
	for _, r := range doc.Records {
		gedcom.Visit(r, func(ref string) {
			if ref == xref {
				n++
			}
		})
		for _, tag := range r.Tags {
			if tag.Value == xref || tag.XRef == xref {
				n++
			}
		}
	}
	return n
}

func tagLines(tags []*gedcom.Tag) []string {
	var out []string
	for _, t := range tags {
		out = append(out, strings.TrimSpace(strings.Join([]string{string(rune('0' + t.Level)), t.Tag, t.Value}, " ")))
	}
	return out
}

func recordLines(t *testing.T, doc *gedcom.Document, xref string) []string {
	t.Helper()
	r := doc.GetRecord(xref)
	if r == nil {
		t.Fatalf("no record %s", xref)
	}
	return tagLines(r.Tags)
}

func mustMerge(t *testing.T, fn func(*gedcom.Document, string, string, merge.RecordMergeOptions) (*gedcom.Document, *merge.RecordMergeReport, error), doc *gedcom.Document, keep, drop string) (*gedcom.Document, *merge.RecordMergeReport) {
	t.Helper()
	out, report, err := fn(doc, keep, drop, merge.RecordMergeOptions{})
	if err != nil {
		t.Fatalf("merge %s into %s: %v", drop, keep, err)
	}
	if out.GetRecord(drop) != nil {
		t.Errorf("%s still in XRefMap", drop)
	}
	for _, r := range out.Records {
		if r.XRef == drop {
			t.Errorf("%s still in Records", drop)
		}
	}
	if n := pointersTo(out, drop); n != 0 {
		t.Errorf("%d pointers still name %s", n, drop)
	}
	return out, report
}

func errorIssues(doc *gedcom.Document) []validator.Issue {
	var out []validator.Issue
	for _, is := range validator.New().ValidateAll(doc) {
		if is.Severity == validator.SeverityError {
			out = append(out, is)
		}
	}
	return out
}

func TestIndividuals_OverlappingTrees(t *testing.T) {
	doc := overlappingTrees(t)
	before := doc.Clone()
	baseline := errorIssues(doc)

	out, report := mustMerge(t, merge.Individuals, doc, "@I1@", "@I11@")
	if !reflect.DeepEqual(doc, before) {
		t.Fatal("Individuals modified its input")
	}

	want := []string{
		"1 NAME John /Example/", "1 SEX M", "1 BIRT", "2 DATE 1 JAN 1900", "2 PLAC Springfield",
		"1 FAMS @F1@", "1 FAMS @F11@", "1 DEAT", "2 DATE 1970", "1 NOTE Seen in a census.",
	}
	if got := recordLines(t, out, "@I1@"); !reflect.DeepEqual(got, want) {
		t.Errorf("@I1@ lines =\n%q\nwant\n%q", got, want)
	}
	var moved, deduped []string
	for _, s := range report.Moved {
		moved = append(moved, s.Tags[0].Tag)
	}
	for _, s := range report.Deduplicated {
		deduped = append(deduped, s.Tags[0].Tag)
	}
	if want := []string{"DEAT", "NOTE", "FAMS"}; !reflect.DeepEqual(moved, want) {
		t.Errorf("Moved = %v, want %v", moved, want)
	}
	if want := []string{"NAME", "SEX", "BIRT"}; !reflect.DeepEqual(deduped, want) {
		t.Errorf("Deduplicated = %v, want %v", deduped, want)
	}
	if report.Keep != "@I1@" || report.Drop != "@I11@" || len(report.Collapsed) != 0 {
		t.Errorf("report = %+v", report)
	}
	var repointed []string
	for _, ref := range report.Repointed {
		repointed = append(repointed, ref.XRef+" "+strings.Join(ref.Path, "."))
	}
	if want := []string{"@I14@ INDI.ASSO", "@I14@ INDI._LINK", "@F11@ FAM.HUSB"}; !reflect.DeepEqual(repointed, want) {
		t.Errorf("Repointed = %v, want %v", repointed, want)
	}

	// The typed view follows the raw lines.
	indi := out.GetIndividual("@I1@")
	if indi == nil || len(indi.SpouseInFamilies) != 2 || indi.DeathEvent() == nil {
		t.Errorf("typed @I1@ = %+v, want two FAMS and a death", indi)
	}
	if f := out.GetFamily("@F11@"); f.Husband != "@I1@" {
		t.Errorf("@F11@ husband = %q, want @I1@", f.Husband)
	}
	if a := out.GetIndividual("@I14@").Associations; len(a) != 1 || a[0].IndividualXRef != "@I1@" {
		t.Errorf("@I14@ associations = %+v", a)
	}

	// Fold the rest of the overlap and both families.
	out, _ = mustMerge(t, merge.Individuals, out, "@I2@", "@I12@")
	out, _ = mustMerge(t, merge.Individuals, out, "@I3@", "@I13@")
	out, report = mustMerge(t, merge.Families, out, "@F1@", "@F11@")

	want = []string{"1 HUSB @I1@", "1 WIFE @I2@", "1 CHIL @I3@", "1 CHIL @I14@", "1 MARR", "2 DATE 1925"}
	if got := recordLines(t, out, "@F1@"); !reflect.DeepEqual(got, want) {
		t.Errorf("@F1@ lines = %q, want %q", got, want)
	}
	if f := out.GetFamily("@F1@"); !reflect.DeepEqual(f.Children, []string{"@I3@", "@I14@"}) {
		t.Errorf("@F1@ children = %v", f.Children)
	}
	var collapsed []string
	for _, ref := range report.Collapsed {
		collapsed = append(collapsed, ref.XRef+" "+strings.Join(ref.Path, "."))
	}
	if want := []string{"@I1@ INDI.FAMS", "@I2@ INDI.FAMS", "@I3@ INDI.FAMC"}; !reflect.DeepEqual(collapsed, want) {
		t.Errorf("Collapsed = %v, want %v", collapsed, want)
	}
	if want := []string{"1 NAME Mary /Sample/", "1 NAME Mary /Example/", "1 SEX F", "1 FAMS @F1@"}; !reflect.DeepEqual(recordLines(t, out, "@I2@"), want) {
		t.Errorf("@I2@ lines = %q, want %q", recordLines(t, out, "@I2@"), want)
	}
	if want := []string{"1 NAME Ann /Example/", "1 SEX F", "1 FAMC @F1@"}; !reflect.DeepEqual(recordLines(t, out, "@I3@"), want) {
		t.Errorf("@I3@ lines = %q, want %q", recordLines(t, out, "@I3@"), want)
	}
	if links := out.GetIndividual("@I3@").ChildInFamilies; len(links) != 1 || links[0].FamilyXRef != "@F1@" {
		t.Errorf("@I3@ FAMC links = %+v", links)
	}

	if got := errorIssues(out); len(got) > len(baseline) {
		t.Errorf("merged tree has %d error issues, the combined trees %d:", len(got), len(baseline))
		for _, is := range got {
			t.Logf("  %s %s %s", is.Code, is.RecordXRef, is.Message)
		}
	}

	// The result encodes and reads back as the same records.
	var buf bytes.Buffer
	if err := encoder.Encode(&buf, out); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	back, err := decoder.Decode(&buf)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got := len(back.Individuals()); got != 4 {
		t.Errorf("read back %d individuals, want 4", got)
	}
}

func TestIndividuals_Errors(t *testing.T) {
	untaggedKeep := func() *gedcom.Document {
		d := decodeZip(t, treeA)
		d.GetRecord("@I1@").Tags = nil
		return d
	}
	untaggedPointer := func() *gedcom.Document {
		d := decodeZip(t, treeA)
		d.GetRecord("@F1@").Tags = nil
		return d
	}
	defined := func() *gedcom.Document {
		d := decodeZip(t, treeA)
		d.Records = append(d.Records, d.GetRecord("@I2@").Clone())
		return d
	}
	tests := []struct {
		name       string
		doc        func() *gedcom.Document
		fn         func(*gedcom.Document, string, string, merge.RecordMergeOptions) (*gedcom.Document, *merge.RecordMergeReport, error)
		keep, drop string
		unknown    bool
	}{
		{"into itself", func() *gedcom.Document { return decodeZip(t, treeA) }, merge.Individuals, "@I1@", "@I1@", false},
		{"missing drop", func() *gedcom.Document { return decodeZip(t, treeA) }, merge.Individuals, "@I1@", "@I9@", true},
		{"missing keep", func() *gedcom.Document { return decodeZip(t, treeA) }, merge.Individuals, "@I9@", "@I1@", true},
		{"family as individual", func() *gedcom.Document { return decodeZip(t, treeA) }, merge.Individuals, "@I1@", "@F1@", false},
		{"individual as family", func() *gedcom.Document { return decodeZip(t, treeA) }, merge.Families, "@F1@", "@I1@", false},
		{"submitter as individual", func() *gedcom.Document { return decodeZip(t, treeA) }, merge.Individuals, "@U1@", "@I1@", false},
		{"defined twice", defined, merge.Individuals, "@I1@", "@I2@", false},
		{"untagged keep", untaggedKeep, merge.Individuals, "@I1@", "@I2@", false},
		{"untagged pointer", untaggedPointer, merge.Individuals, "@I1@", "@I2@", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := tt.doc()
			before := doc.Clone()
			out, report, err := tt.fn(doc, tt.keep, tt.drop, merge.RecordMergeOptions{})
			if !errors.Is(err, merge.ErrInvalidMerge) {
				t.Fatalf("err = %v, want ErrInvalidMerge", err)
			}
			if got := errors.Is(err, gedcom.ErrUnknownXRef); got != tt.unknown {
				t.Errorf("errors.Is(err, ErrUnknownXRef) = %t, want %t", got, tt.unknown)
			}
			if out != nil || report != nil {
				t.Error("want nil document and report")
			}
			if !reflect.DeepEqual(doc, before) {
				t.Error("document modified")
			}
		})
	}

	if _, _, err := merge.Families(nil, "@F1@", "@F2@", merge.RecordMergeOptions{}); !errors.Is(err, merge.ErrNilDocument) {
		t.Errorf("nil doc: err = %v, want ErrNilDocument", err)
	}
}

func TestIndividuals_UntaggedRecordNotPointingAtDropIsFine(t *testing.T) {
	doc := decodeZip(t, treeA)
	doc.GetRecord("@U1@").Tags = nil
	if _, _, err := merge.Individuals(doc, "@I1@", "@I3@", merge.RecordMergeOptions{}); err != nil {
		t.Fatalf("Individuals: %v", err)
	}
}

// TestFamilies_CollapseKeepsDistinctLines checks only a line that would
// repeat a sibling exactly is collapsed: a FAMC with a different PEDI is
// repointed, and a repeat that was already in the document stays.
func TestFamilies_CollapseKeepsDistinctLines(t *testing.T) {
	doc := decodeZip(t, `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 FAMC @F1@
2 PEDI birth
1 FAMC @F2@
2 PEDI adopted
0 @I2@ INDI
1 FAMC @F1@
1 FAMC @F1@
1 FAMC @F2@
0 @F1@ FAM
1 CHIL @I1@
1 CHIL @I2@
1 CHIL @I2@
0 @F2@ FAM
1 CHIL @I1@
1 CHIL @I2@
0 TRLR
`)
	out, report := mustMerge(t, merge.Families, doc, "@F1@", "@F2@")
	if want := []string{"1 FAMC @F1@", "2 PEDI birth", "1 FAMC @F1@", "2 PEDI adopted"}; !reflect.DeepEqual(recordLines(t, out, "@I1@"), want) {
		t.Errorf("@I1@ = %q, want %q", recordLines(t, out, "@I1@"), want)
	}
	if want := []string{"1 FAMC @F1@", "1 FAMC @F1@"}; !reflect.DeepEqual(recordLines(t, out, "@I2@"), want) {
		t.Errorf("@I2@ = %q, want %q", recordLines(t, out, "@I2@"), want)
	}
	if want := []string{"1 CHIL @I1@", "1 CHIL @I2@", "1 CHIL @I2@"}; !reflect.DeepEqual(recordLines(t, out, "@F1@"), want) {
		t.Errorf("@F1@ = %q, want %q", recordLines(t, out, "@F1@"), want)
	}
	if len(report.Deduplicated) != 2 || len(report.Moved) != 0 || len(report.Collapsed) != 1 || len(report.Repointed) != 1 {
		t.Errorf("report = %+v", report)
	}
}

func TestIndividuals_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out, report, err := merge.Individuals(decodeZip(t, treeA), "@I1@", "@I3@", merge.RecordMergeOptions{Context: ctx})
	if !errors.Is(err, context.Canceled) || out != nil || report != nil {
		t.Errorf("got %v, %v, %v; want nil, nil and context.Canceled", out, report, err)
	}
}
