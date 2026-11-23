package gedcom_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

func decodeDiffDoc(t *testing.T, records string) *gedcom.Document {
	t.Helper()
	doc, err := decoder.Decode(strings.NewReader("0 HEAD\n1 GEDC\n2 VERS 5.5.1\n" + records + "0 TRLR\n"))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return doc
}

// renderChange writes a hunk as "path: -line +line" for compact assertions.
func renderChange(c gedcom.LineChange) string {
	var parts []string
	for _, t := range c.Removed {
		parts = append(parts, "-"+renderLine(t))
	}
	for _, t := range c.Added {
		parts = append(parts, "+"+renderLine(t))
	}
	return strings.Join(c.Path, ".") + ": " + strings.Join(parts, " | ")
}

func renderLine(t *gedcom.Tag) string {
	s := fmt.Sprintf("%d", t.Level)
	if t.XRef != "" {
		s += " " + t.XRef
	}
	s += " " + t.Tag
	if t.Value != "" {
		s += " " + t.Value
	}
	return s
}

func renderChanges(rd *gedcom.RecordDiff) []string {
	var out []string
	for _, c := range rd.Changes {
		out = append(out, renderChange(c))
	}
	return out
}

func TestDiff_AddedRemovedModified(t *testing.T) {
	oldDoc := decodeDiffDoc(t, `0 @I1@ INDI
1 NAME Ann /Example/
1 BIRT
2 DATE 1 JAN 1900
0 @I2@ INDI
1 NAME Bob /Example/
0 @S1@ SOUR
1 TITL Example Register
`)
	newDoc := decodeDiffDoc(t, `0 @I1@ INDI
1 NAME Ann /Example/
1 BIRT
2 DATE 2 JAN 1900
0 @S1@ SOUR
1 TITL Example Register
0 @I3@ INDI
1 NAME Cy /Example/
`)
	d := gedcom.Diff(oldDoc, newDoc, nil)

	if len(d.Removed) != 1 || d.Removed[0].XRef != "@I2@" {
		t.Errorf("Removed = %v, want @I2@", d.Removed)
	}
	if len(d.Added) != 1 || d.Added[0].XRef != "@I3@" {
		t.Errorf("Added = %v, want @I3@", d.Added)
	}
	if len(d.Modified) != 1 {
		t.Fatalf("Modified = %d records, want 1", len(d.Modified))
	}
	rd := d.Modified[0]
	if rd.XRef != "@I1@" || rd.Type != gedcom.RecordTypeIndividual {
		t.Errorf("Modified[0] = %s %s, want @I1@ INDI", rd.XRef, rd.Type)
	}
	if rd.Old != oldDoc.GetRecord("@I1@") || rd.New != newDoc.GetRecord("@I1@") {
		t.Error("Old/New are not the documents' own records")
	}
	want := []string{"BIRT: -2 DATE 1 JAN 1900 | +2 DATE 2 JAN 1900"}
	if got := renderChanges(rd); !reflect.DeepEqual(got, want) {
		t.Errorf("Changes = %q, want %q", got, want)
	}
	if rd.Changes[0].Removed[0].LineNumber != 7 {
		t.Errorf("removed line number = %d, want 7", rd.Changes[0].Removed[0].LineNumber)
	}
	if d.IsEmpty() {
		t.Error("IsEmpty() = true")
	}
}

func TestDiff_Identical(t *testing.T) {
	src := "0 @I1@ INDI\n1 NAME Ann /Example/\n0 @N1@ NOTE Some text\n"
	d := gedcom.Diff(decodeDiffDoc(t, src), decodeDiffDoc(t, src), &gedcom.DiffOptions{IncludeHeader: true})
	if !d.IsEmpty() {
		t.Errorf("diff of identical documents = %+v", d)
	}
}

func TestDiff_MeaningPreserved(t *testing.T) {
	tests := []struct {
		name     string
		old, new string
	}{
		{
			name: "different-tag siblings reordered",
			old:  "0 @I1@ INDI\n1 NAME Ann /Example/\n1 SEX F\n1 BIRT\n2 PLAC Exampleton\n2 DATE 1900\n",
			new:  "0 @I1@ INDI\n1 BIRT\n2 DATE 1900\n2 PLAC Exampleton\n1 SEX F\n1 NAME Ann /Example/\n",
		},
		{
			name: "value split differently across CONC",
			old:  "0 @N1@ NOTE Hello wo\n1 CONC rld\n1 CONT second line\n",
			new:  "0 @N1@ NOTE Hel\n1 CONC lo world\n1 CONT second\n2 CONC  line\n",
		},
		{
			name: "records in a different order",
			old:  "0 @I1@ INDI\n1 SEX F\n0 @I2@ INDI\n1 SEX M\n",
			new:  "0 @I2@ INDI\n1 SEX M\n0 @I1@ INDI\n1 SEX F\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := gedcom.Diff(decodeDiffDoc(t, tt.old), decodeDiffDoc(t, tt.new), nil)
			if !d.IsEmpty() {
				for _, m := range d.Modified {
					t.Errorf("%s reported modified: %q", m.XRef, renderChanges(m))
				}
				t.Errorf("Added %d, Removed %d", len(d.Added), len(d.Removed))
			}
		})
	}
}

func TestDiff_LineChanges(t *testing.T) {
	tests := []struct {
		name     string
		old, new string
		want     []string
	}{
		{
			name: "same-tag siblings reordered",
			old:  "0 @F1@ FAM\n1 CHIL @I1@\n1 CHIL @I2@\n",
			new:  "0 @F1@ FAM\n1 CHIL @I2@\n1 CHIL @I1@\n",
			want: []string{": -1 CHIL @I1@", ": +1 CHIL @I1@"},
		},
		{
			name: "same-tag siblings aligned around several matches",
			old:  "0 @F1@ FAM\n1 CHIL @I1@\n1 CHIL @I9@\n1 CHIL @I2@\n1 CHIL @I3@\n1 CHIL @I8@\n",
			new:  "0 @F1@ FAM\n1 CHIL @I2@\n1 CHIL @I7@\n1 CHIL @I3@\n1 CHIL @I6@\n",
			want: []string{
				": -1 CHIL @I1@ | -1 CHIL @I9@",
				": +1 CHIL @I7@",
				": -1 CHIL @I8@ | +1 CHIL @I6@",
			},
		},
		{
			name: "structure inserted among same-tag siblings",
			old:  "0 @I1@ INDI\n1 NAME Ann /Example/\n1 NAME Anna /Example/\n",
			new:  "0 @I1@ INDI\n1 NAME Ann /Example/\n1 NAME Annie /Example/\n2 TYPE aka\n1 NAME Anna /Example/\n",
			want: []string{": +1 NAME Annie /Example/ | +2 TYPE aka"},
		},
		{
			name: "whole structure removed",
			old:  "0 @I1@ INDI\n1 SEX F\n1 DEAT\n2 DATE 1950\n2 PLAC Exampleton\n",
			new:  "0 @I1@ INDI\n1 SEX F\n",
			want: []string{": -1 DEAT | -2 DATE 1950 | -2 PLAC Exampleton"},
		},
		{
			name: "level-0 value with continuation",
			old:  "0 @N1@ NOTE First\n1 CONT second\n",
			new:  "0 @N1@ NOTE First\n1 CONT changed\n",
			want: []string{": -0 @N1@ NOTE First | -1 CONT second | +0 @N1@ NOTE First | +1 CONT changed"},
		},
		{
			name: "own line and substructure both changed",
			old:  "0 @I1@ INDI\n1 EVEN Old\n2 TYPE Example\n2 DATE 1900\n",
			new:  "0 @I1@ INDI\n1 EVEN New\n2 TYPE Example\n2 DATE 1901\n",
			want: []string{
				": -1 EVEN Old | +1 EVEN New",
				"EVEN: -2 DATE 1900 | +2 DATE 1901",
			},
		},
		{
			name: "case in a value is a change",
			old:  "0 @I1@ INDI\n1 NAME ann /Example/\n",
			new:  "0 @I1@ INDI\n1 NAME Ann /Example/\n",
			want: []string{": -1 NAME ann /Example/ | +1 NAME Ann /Example/"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := gedcom.Diff(decodeDiffDoc(t, tt.old), decodeDiffDoc(t, tt.new), nil)
			if len(d.Modified) != 1 {
				t.Fatalf("Modified = %d, want 1 (Added %d, Removed %d)", len(d.Modified), len(d.Added), len(d.Removed))
			}
			if got := renderChanges(d.Modified[0]); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Changes =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

func TestDiff_IgnoreTags(t *testing.T) {
	oldDoc := decodeDiffDoc(t, `0 @I1@ INDI
1 NAME Ann /Example/
1 _UID AAAA
1 CHAN
2 DATE 1 JAN 2020
0 @U1@ _USER
1 NAME first
`)
	newDoc := decodeDiffDoc(t, `0 @I1@ INDI
1 NAME Ann /Example/
1 CHAN
2 DATE 2 FEB 2021
1 _UID BBBB
1 _UID CCCC
0 @U2@ _USER
1 NAME second
`)
	if d := gedcom.Diff(oldDoc, newDoc, nil); d.IsEmpty() {
		t.Fatal("unfiltered diff is empty")
	}
	d := gedcom.Diff(oldDoc, newDoc, &gedcom.DiffOptions{IgnoreTags: []string{"CHAN", "_UID", "_USER"}})
	if !d.IsEmpty() {
		t.Errorf("diff ignoring CHAN, _UID and _USER = %+v", d)
	}

	// Only the named tag's subtree is skipped: the following sibling counts.
	newDoc2 := decodeDiffDoc(t, "0 @I1@ INDI\n1 NAME Ann /Example/\n1 CHAN\n2 DATE 3 MAR 2022\n1 SEX F\n")
	d = gedcom.Diff(decodeDiffDoc(t, "0 @I1@ INDI\n1 NAME Ann /Example/\n"), newDoc2,
		&gedcom.DiffOptions{IgnoreTags: []string{"CHAN"}})
	if len(d.Modified) != 1 || !reflect.DeepEqual(renderChanges(d.Modified[0]), []string{": +1 SEX F"}) {
		t.Errorf("diff = %+v", d)
	}
}

func TestDiff_Header(t *testing.T) {
	oldDoc := decodeDiffDoc(t, "")
	newDoc, err := decoder.Decode(strings.NewReader("0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 DATE 1 JAN 2024\n0 TRLR\n"))
	if err != nil {
		t.Fatal(err)
	}
	if d := gedcom.Diff(oldDoc, newDoc, nil); !d.IsEmpty() {
		t.Errorf("header compared without IncludeHeader: %+v", d)
	}
	d := gedcom.Diff(oldDoc, newDoc, &gedcom.DiffOptions{IncludeHeader: true})
	if d.Header == nil || d.Header.Type != "HEAD" {
		t.Fatalf("Header = %+v", d.Header)
	}
	if got := renderChanges(d.Header); !reflect.DeepEqual(got, []string{": +1 DATE 1 JAN 2024"}) {
		t.Errorf("header changes = %q", got)
	}
	if d := gedcom.Diff(oldDoc, newDoc, &gedcom.DiffOptions{IncludeHeader: true, IgnoreTags: []string{"DATE"}}); !d.IsEmpty() {
		t.Errorf("header DATE not ignored: %+v", d.Header)
	}
}

func TestDiff_MatchingRules(t *testing.T) {
	// A type change at the same XRef is a removal and an addition.
	d := gedcom.Diff(decodeDiffDoc(t, "0 @X1@ NOTE text\n"), decodeDiffDoc(t, "0 @X1@ SOUR\n1 TITL text\n"), nil)
	if len(d.Removed) != 1 || len(d.Added) != 1 || len(d.Modified) != 0 {
		t.Errorf("type change: %+v", d)
	}

	// Duplicate XRefs are matched by occurrence.
	oldDoc := &gedcom.Document{Records: []*gedcom.Record{
		{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Tags: []*gedcom.Tag{{Level: 1, Tag: "SEX", Value: "F"}}},
		{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Tags: []*gedcom.Tag{{Level: 1, Tag: "SEX", Value: "M"}}},
	}}
	newDoc := &gedcom.Document{Records: []*gedcom.Record{
		{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Tags: []*gedcom.Tag{{Level: 1, Tag: "SEX", Value: "F"}}},
		{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Tags: []*gedcom.Tag{{Level: 1, Tag: "SEX", Value: "U"}}},
	}}
	d = gedcom.Diff(oldDoc, newDoc, nil)
	if len(d.Modified) != 1 || d.Modified[0].Old != oldDoc.Records[1] {
		t.Errorf("duplicate XRefs: %+v", d)
	}
}

func TestDiff_NilAndMalformedInput(t *testing.T) {
	doc := decodeDiffDoc(t, "0 @I1@ INDI\n1 SEX F\n")
	if d := gedcom.Diff(nil, doc, nil); len(d.Added) != 1 {
		t.Errorf("Diff(nil, doc).Added = %d, want 1", len(d.Added))
	}
	if d := gedcom.Diff(doc, nil, &gedcom.DiffOptions{IncludeHeader: true}); len(d.Removed) != 1 || d.Header == nil {
		t.Errorf("Diff(doc, nil) = %+v", d)
	}
	if d := gedcom.Diff(nil, nil, &gedcom.DiffOptions{IncludeHeader: true}); !d.IsEmpty() {
		t.Errorf("Diff(nil, nil) = %+v", d)
	}
	var nilDiff *gedcom.DocumentDiff
	if !nilDiff.IsEmpty() {
		t.Error("nil DocumentDiff is not empty")
	}

	// Nil records and tags are skipped; a level jump nests under the
	// nearest shallower tag.
	withNils := &gedcom.Document{Records: []*gedcom.Record{nil, {
		XRef: "@I1@", Type: gedcom.RecordTypeIndividual,
		Tags: []*gedcom.Tag{nil, {Level: 1, Tag: "BIRT"}, {Level: 3, Tag: "DATE", Value: "1900"}},
	}}}
	changed := &gedcom.Document{Records: []*gedcom.Record{{
		XRef: "@I1@", Type: gedcom.RecordTypeIndividual,
		Tags: []*gedcom.Tag{{Level: 1, Tag: "BIRT"}, {Level: 3, Tag: "DATE", Value: "1901"}},
	}}}
	d := gedcom.Diff(withNils, changed, nil)
	if len(d.Modified) != 1 ||
		!reflect.DeepEqual(renderChanges(d.Modified[0]), []string{"BIRT: -3 DATE 1900 | +3 DATE 1901"}) {
		t.Errorf("diff = %+v", d)
	}
}

func TestDiff_HunksOwnTheirPath(t *testing.T) {
	oldDoc := decodeDiffDoc(t, "0 @I1@ INDI\n1 BIRT\n2 DATE 1900\n2 PLAC Exampleton\n")
	newDoc := decodeDiffDoc(t, "0 @I1@ INDI\n1 BIRT\n2 DATE 1901\n2 PLAC Exampleville\n")
	d := gedcom.Diff(oldDoc, newDoc, nil)
	if len(d.Modified) != 1 || len(d.Modified[0].Changes) != 2 {
		t.Fatalf("diff = %+v", d)
	}
	changes := d.Modified[0].Changes
	changes[0].Path[0] = "MUTATED"
	if changes[1].Path[0] != "BIRT" {
		t.Errorf("second hunk Path = %q, want [BIRT]", changes[1].Path)
	}
}

func TestDiff_LargeSameTagRunSkipsAlignment(t *testing.T) {
	// Past the alignment bound, the unequal middle is compared by position.
	// n*n must exceed the unexported maxLCSCells (1<<20) in diff.go.
	var oldSrc, newSrc strings.Builder
	oldSrc.WriteString("0 @S1@ SOUR\n")
	newSrc.WriteString("0 @S1@ SOUR\n")
	const n = 1100
	for i := 0; i < n; i++ {
		fmt.Fprintf(&oldSrc, "1 NOTE line %d\n", i)
		fmt.Fprintf(&newSrc, "1 NOTE line %d\n", (i+1)%n)
	}
	d := gedcom.Diff(decodeDiffDoc(t, oldSrc.String()), decodeDiffDoc(t, newSrc.String()), nil)
	if len(d.Modified) != 1 || len(d.Modified[0].Changes) != n {
		t.Fatalf("got %d records, want 1 with %d changes", len(d.Modified), n)
	}
	got := renderChanges(d.Modified[0])
	if first, last := got[0], got[n-1]; first != ": -1 NOTE line 0 | +1 NOTE line 1" ||
		last != fmt.Sprintf(": -1 NOTE line %d | +1 NOTE line 0", n-1) {
		t.Errorf("first, last = %q, %q", first, last)
	}
}

func ExampleDiff() {
	oldDoc, _ := decoder.Decode(strings.NewReader(`0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME Ann /Example/
1 BIRT
2 DATE 1 JAN 1900
1 CHAN
2 DATE 1 JAN 2020
0 @I2@ INDI
1 NAME Bob /Example/
0 TRLR
`))
	newDoc, _ := decoder.Decode(strings.NewReader(`0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 BIRT
2 DATE 2 JAN 1900
1 NAME Ann /Example/
1 CHAN
2 DATE 5 MAY 2025
0 @I3@ INDI
1 NAME Cy /Example/
0 TRLR
`))

	d := gedcom.Diff(oldDoc, newDoc, &gedcom.DiffOptions{IgnoreTags: []string{"CHAN"}})
	for _, r := range d.Added {
		fmt.Println("added", r.XRef)
	}
	for _, r := range d.Removed {
		fmt.Println("removed", r.XRef)
	}
	for _, m := range d.Modified {
		for _, c := range m.Changes {
			for _, line := range c.Removed {
				fmt.Printf("%s - %d %s %s\n", m.XRef, line.Level, line.Tag, line.Value)
			}
			for _, line := range c.Added {
				fmt.Printf("%s + %d %s %s\n", m.XRef, line.Level, line.Tag, line.Value)
			}
		}
	}
	// Output:
	// added @I3@
	// removed @I2@
	// @I1@ - 2 DATE 1 JAN 1900
	// @I1@ + 2 DATE 2 JAN 1900
}
