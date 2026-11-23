package merge_test

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand/v2"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/merge"
)

// zipFixture is four linked families over three generations, with the
// pointer shapes a bounded subset cuts and zipping restores: HUSB, WIFE and
// CHIL (one with subordinate lines, one after an event). Every
// non-individual record is referenced, so the bounded subset of every
// individual is the whole document.
const zipFixture = `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
1 SUBM @U1@
0 @U1@ SUBM
1 NAME Example Submitter
0 @R1@ REPO
1 NAME Example Archive
0 @S1@ SOUR
1 TITL Example Parish Register
1 REPO @R1@
0 @S2@ SOUR
1 TITL Example Census
1 NOTE @N2@
0 @N1@ NOTE A shared note
0 @N2@ NOTE A note on a source
0 @O1@ OBJE
1 FILE example.jpg
2 FORM jpg
0 @I1@ INDI
1 NAME Adam /Example/
1 SEX M
1 SOUR @S1@
1 FAMS @F1@
0 @I2@ INDI
1 NAME Beth /Example/
1 SEX F
1 FAMS @F1@
0 @I3@ INDI
1 NAME Carl /Example/
1 SEX M
1 FAMC @F1@
2 PEDI birth
1 FAMS @F2@
0 @I4@ INDI
1 NAME Dora /Example/
1 SEX F
1 FAMC @F1@
1 FAMS @F4@
0 @I5@ INDI
1 NAME Eli /Example/
1 FAMC @F1@
1 BIRT
2 DATE 1905
2 SOUR @S2@
0 @I6@ INDI
1 NAME Fay /Sample/
1 SEX F
1 FAMC @F3@
1 FAMS @F2@
1 NOTE @N1@
0 @I7@ INDI
1 NAME Gus /Example/
1 FAMC @F2@
1 OBJE @O1@
0 @I8@ INDI
1 NAME Hal /Sample/
1 FAMS @F3@
0 @I9@ INDI
1 NAME Ida /Sample/
1 FAMS @F3@
0 @I10@ INDI
1 NAME Jon /Other/
1 FAMS @F4@
0 @I11@ INDI
1 NAME Kit /Other/
1 FAMC @F4@
0 @I12@ INDI
1 NAME Lou /Other/
1 FAMC @F4@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
2 _FREL Natural
2 _MREL Natural
1 CHIL @I4@
1 MARR
2 DATE 1900
2 SOUR @S1@
1 CHIL @I5@
0 @F2@ FAM
1 HUSB @I3@
1 WIFE @I6@
1 CHIL @I7@
1 NOTE @N1@
0 @F3@ FAM
1 HUSB @I8@
1 WIFE @I9@
1 CHIL @I6@
0 @F4@ FAM
1 HUSB @I10@
1 WIFE @I4@
1 CHIL @I11@
1 CHIL @I12@
1 DIV
0 TRLR
`

func decodeZip(t *testing.T, text string) *gedcom.Document {
	t.Helper()
	doc, err := decoder.Decode(strings.NewReader(text))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return doc
}

func individualXRefs(doc *gedcom.Document) []string {
	var out []string
	for _, r := range doc.Records {
		if r.IsIndividual() {
			out = append(out, r.XRef)
		}
	}
	return out
}

func bounded(t *testing.T, doc *gedcom.Document, set []string) *gedcom.Document {
	t.Helper()
	sub, _, err := doc.BoundedSubset(set)
	if err != nil {
		t.Fatalf("BoundedSubset(%v): %v", set, err)
	}
	return sub
}

func zip(t *testing.T, a, b *gedcom.Document) *gedcom.Document {
	t.Helper()
	out, report, err := merge.Combine(a, b, merge.CombineOptions{CollisionStrategy: merge.ZipSharedXRefs})
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}
	if len(report.RecordConflicts) != 0 || len(report.RemappedXRefs) != 0 || len(report.HeaderConflicts) != 0 {
		t.Fatalf("report = %+v, want no conflicts or remaps", report)
	}
	return out
}

// lines renders a record's Tags without line numbers.
func lines(r *gedcom.Record) []string {
	var out []string
	for _, tag := range r.Tags {
		out = append(out, fmt.Sprintf("%d %s %s %s", tag.Level, tag.XRef, tag.Tag, tag.Value))
	}
	return out
}

// assertSameRecords fails unless got and want hold the same records, XRef for
// XRef: the same type, value, lines in the same order, and typed Entity.
// With sortLines the lines are compared as a set instead.
func assertSameRecords(t *testing.T, label string, got, want *gedcom.Document, sortLines bool) {
	t.Helper()
	gotX, wantX := recordXRefSet(got), recordXRefSet(want)
	if !slices.Equal(gotX, wantX) {
		t.Fatalf("%s: records = %v, want %v", label, gotX, wantX)
	}
	for _, xref := range wantX {
		g, w := got.GetRecord(xref), want.GetRecord(xref)
		gl, wl := lines(g), lines(w)
		if sortLines {
			slices.Sort(gl)
			slices.Sort(wl)
		}
		if g.Type != w.Type || g.Value != w.Value || !slices.Equal(gl, wl) {
			t.Errorf("%s: %s lines = %q, want %q", label, xref, gl, wl)
			continue
		}
		if !sortLines && !reflect.DeepEqual(g.Entity, w.Entity) {
			t.Errorf("%s: %s typed entity differs", label, xref)
		}
	}
}

func recordXRefSet(doc *gedcom.Document) []string {
	var out []string
	for _, r := range doc.Records {
		out = append(out, r.XRef)
	}
	slices.Sort(out)
	return out
}

// TestZip_BoundedSubsetsProperty pins the zip guarantee: the bounded subsets
// of S1 and S2 cut from one document zip into the bounded subset of S1 ∪ S2,
// and into the document itself when S1 ∪ S2 is every individual.
func TestZip_BoundedSubsetsProperty(t *testing.T) {
	doc := decodeZip(t, zipFixture)
	people := individualXRefs(doc)
	rng := rand.New(rand.NewPCG(888, 1))

	for trial := range 300 {
		var s1, s2, union []string
		for _, p := range people {
			// Each individual goes to S1, S2, both or neither; a third of
			// the trials leave nobody out, so S1 ∪ S2 is everyone.
			pick := rng.IntN(4)
			if trial%3 == 0 {
				pick = 1 + rng.IntN(3)
			}
			if pick&1 != 0 {
				s1 = append(s1, p)
			}
			if pick&2 != 0 {
				s2 = append(s2, p)
			}
			if pick != 0 {
				union = append(union, p)
			}
		}
		label := fmt.Sprintf("trial %d S1=%v S2=%v", trial, s1, s2)
		got := zip(t, bounded(t, doc, s1), bounded(t, doc, s2))
		assertSameRecords(t, label, got, bounded(t, doc, union), false)
		if len(union) == len(people) {
			assertSameRecords(t, label+" (everyone)", got, doc, false)
		}
		if t.Failed() {
			return
		}
	}
}

// TestZip_AfterFileRoundtrip zips subsets that were encoded and decoded
// again, so their line numbers are their own files'. The records hold the
// same lines; a run of pointer lines split between the files need not keep
// the master's order.
func TestZip_AfterFileRoundtrip(t *testing.T) {
	doc := decodeZip(t, zipFixture)
	roundtrip := func(sub *gedcom.Document) *gedcom.Document {
		var buf bytes.Buffer
		if err := encoder.Encode(&buf, sub); err != nil {
			t.Fatal(err)
		}
		return decodeZip(t, buf.String())
	}
	s1 := []string{"@I1@", "@I3@", "@I5@", "@I8@", "@I10@", "@I11@"}
	s2 := []string{"@I2@", "@I4@", "@I6@", "@I7@", "@I9@", "@I12@"}
	got := zip(t, roundtrip(bounded(t, doc, s1)), roundtrip(bounded(t, doc, s2)))
	assertSameRecords(t, "roundtrip", got, doc, true)
}

func TestZip_ConflictFailsWithoutOutput(t *testing.T) {
	doc := decodeZip(t, zipFixture)
	a := bounded(t, doc, []string{"@I3@"})
	edited := strings.Replace(zipFixture, "1 NAME Carl /Example/", "1 NAME Karl /Example/", 1)
	b := bounded(t, decodeZip(t, edited), []string{"@I3@", "@I7@"})

	out, report, err := merge.Combine(a, b, merge.CombineOptions{CollisionStrategy: merge.ZipSharedXRefs})
	if !errors.Is(err, merge.ErrRecordConflict) || out != nil || report != nil {
		t.Fatalf("Combine = %v, %v, %v; want ErrRecordConflict and no output", out, report, err)
	}
	var ce *merge.CombineError
	if !errors.As(err, &ce) || !slices.Equal(ce.Colliding, []string{"@I3@"}) {
		t.Fatalf("err = %#v, want a conflict naming @I3@", err)
	}
	if want := []merge.RecordConflict{{XRef: "@I3@", Reason: merge.ConflictLines}}; !reflect.DeepEqual(ce.Conflicts, want) {
		t.Errorf("conflicts = %+v, want %+v", ce.Conflicts, want)
	}
	if !strings.Contains(err.Error(), "@I3@") || !strings.Contains(err.Error(), "lines") {
		t.Errorf("error %q does not name the xref and reason", err)
	}

	out, report, err = merge.Combine(a, b, merge.CombineOptions{CollisionStrategy: merge.ZipSharedXRefs, KeepFirstOnConflict: true})
	if err != nil {
		t.Fatal(err)
	}
	if want := []merge.RecordConflict{{XRef: "@I3@", Reason: merge.ConflictLines}}; !reflect.DeepEqual(report.RecordConflicts, want) {
		t.Errorf("RecordConflicts = %+v, want %+v", report.RecordConflicts, want)
	}
	if got := out.GetIndividual("@I3@").Names[0].Full; got != "Carl /Example/" {
		t.Errorf("kept name = %q, want doc1's", got)
	}
	// The rest still zips: @F2@ gains @I7@ from doc2.
	if got := out.GetFamily("@F2@").Children; !slices.Equal(got, []string{"@I7@"}) {
		t.Errorf("F2 children = %v", got)
	}
}

func TestZip_ConflictReasons(t *testing.T) {
	rec := func(typ gedcom.RecordType, value string, tags ...*gedcom.Tag) *gedcom.Record {
		return &gedcom.Record{XRef: "@X1@", Type: typ, Value: value, Tags: tags}
	}
	docWith := func(r *gedcom.Record) *gedcom.Document {
		doc := gedcom.NewDocument(gedcom.Version551)
		if err := doc.AddRecord(r); err != nil {
			t.Fatal(err)
		}
		return doc
	}
	line := &gedcom.Tag{Level: 1, Tag: "CONT", Value: "more"}
	tests := []struct {
		name   string
		a, b   *gedcom.Record
		reason merge.ConflictReason
	}{
		{"type", rec(gedcom.RecordTypeNote, "x", line), rec(gedcom.RecordTypeSource, "x", line), merge.ConflictType},
		{"value", rec(gedcom.RecordTypeNote, "x", line), rec(gedcom.RecordTypeNote, "y", line), merge.ConflictValue},
		{"lines", rec(gedcom.RecordTypeNote, "x", line), rec(gedcom.RecordTypeNote, "x", &gedcom.Tag{Level: 1, Tag: "CONT", Value: "other"}), merge.ConflictLines},
		{"untagged", rec(gedcom.RecordTypeNote, "x"), rec(gedcom.RecordTypeNote, "x", line), merge.ConflictUntagged},
		{"anchor count", rec(gedcom.RecordTypeNote, "x", line), rec(gedcom.RecordTypeNote, "x", line, line), merge.ConflictLines},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, report, err := merge.Combine(docWith(tc.a), docWith(tc.b), merge.CombineOptions{
				CollisionStrategy: merge.ZipSharedXRefs, KeepFirstOnConflict: true,
			})
			if err != nil {
				t.Fatal(err)
			}
			want := []merge.RecordConflict{{XRef: "@X1@", Reason: tc.reason}}
			if !reflect.DeepEqual(report.RecordConflicts, want) {
				t.Errorf("RecordConflicts = %+v, want %+v", report.RecordConflicts, want)
			}
		})
	}
}

// TestZip_IdenticalAndUntagged covers records zipping collapses to one:
// identical tagged records, and records without Tags whose entities match.
func TestZip_IdenticalAndUntagged(t *testing.T) {
	build := func() *gedcom.Document {
		doc := gedcom.NewDocument(gedcom.Version551)
		for _, r := range []*gedcom.Record{
			{XRef: "@N1@", Type: gedcom.RecordTypeNote, Value: "x", Tags: []*gedcom.Tag{{Level: 1, Tag: "CONT", Value: "y"}}},
			{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: "@I1@", Sex: "F"}},
		} {
			if err := doc.AddRecord(r); err != nil {
				t.Fatal(err)
			}
		}
		return doc
	}
	out := zip(t, build(), build())
	if got := recordXRefSet(out); !slices.Equal(got, []string{"@I1@", "@N1@"}) {
		t.Errorf("records = %v", got)
	}
}

func TestZip_DifferingHeadersMerge(t *testing.T) {
	doc := decodeZip(t, zipFixture)
	other := decodeZip(t, strings.Replace(zipFixture, "1 CHAR UTF-8\n", "1 CHAR UTF-8\n1 LANG English\n", 1))
	_, report, err := merge.Combine(bounded(t, doc, []string{"@I1@"}), bounded(t, other, []string{"@I2@"}),
		merge.CombineOptions{CollisionStrategy: merge.ZipSharedXRefs})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.HeaderConflicts) == 0 {
		t.Error("want the differing headers merged, with conflicts reported")
	}
}

// TestZip_HeaderPointers covers header lines that differ between two
// bounded subsets of one master: a vendor line naming an individual only
// one subset holds is unioned, and a note the header points at is in both.
func TestZip_HeaderPointers(t *testing.T) {
	text := strings.Replace(zipFixture, "1 SUBM @U1@\n", "1 SUBM @U1@\n1 _HOME @I1@\n1 NOTE @N3@\n", 1)
	text = strings.Replace(text, "0 TRLR\n", "0 @N3@ NOTE A header note\n0 TRLR\n", 1)
	doc := decodeZip(t, text)
	a, b := bounded(t, doc, []string{"@I1@"}), bounded(t, doc, []string{"@I7@"})
	if b.GetRecord("@N3@") == nil {
		t.Fatal("subset lacks the note its header points at")
	}
	out := zip(t, a, b)
	var lines []string
	for _, tag := range out.Header.Tags {
		if tag.Level == 1 {
			lines = append(lines, tag.Tag+" "+tag.Value)
		}
	}
	want := []string{"GEDC ", "CHAR UTF-8", "SUBM @U1@", "_HOME @I1@", "NOTE @N3@"}
	if !slices.Equal(lines, want) {
		t.Errorf("header lines = %q, want %q", lines, want)
	}
}
