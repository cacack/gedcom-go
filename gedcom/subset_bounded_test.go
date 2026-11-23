package gedcom_test

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// boundedFixture is three linked families: @F3@ (parents of @I6@), @F1@
// (parents of @I3@, @I4@, @I5@) and @F2@ (@I3@ and @I6@, parents of @I7@).
const boundedFixture = `0 HEAD
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
0 @N1@ NOTE A shared note
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
1 FAMS @F2@
0 @I4@ INDI
1 NAME Dora /Example/
1 SEX F
1 FAMC @F1@
1 ASSO @I8@
2 RELA Godfather
0 @I5@ INDI
1 NAME Eli /Example/
1 FAMC @F1@
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
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
2 _FREL Natural
1 CHIL @I4@
1 MARR
2 DATE 1900
2 SOUR @S1@
1 CHIL @I5@
0 @F2@ FAM
1 HUSB @I3@
1 WIFE @I6@
1 CHIL @I7@
0 @F3@ FAM
1 HUSB @I8@
1 WIFE @I9@
1 CHIL @I6@
0 TRLR
`

func decodeBounded(t *testing.T) *gedcom.Document {
	t.Helper()
	doc, err := decoder.Decode(strings.NewReader(boundedFixture))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return doc
}

func recordXRefs(doc *gedcom.Document) []string {
	var out []string
	for _, r := range doc.Records {
		out = append(out, r.XRef)
	}
	slices.Sort(out)
	return out
}

// assertSelfContained fails if any record in doc points at a record doc
// does not hold.
func assertSelfContained(t *testing.T, doc *gedcom.Document) {
	t.Helper()
	for _, r := range doc.Records {
		gedcom.VisitForVersion(r, doc.Header.DeclaredVersion, func(ref string) {
			if doc.GetRecord(ref) == nil {
				t.Errorf("%s points outside the result at %s", r.XRef, ref)
			}
		})
	}
}

func TestBoundedSubset_OneIndividual(t *testing.T) {
	doc := decodeBounded(t)
	sub, dropped, err := doc.BoundedSubset([]string{"@I3@"})
	if err != nil {
		t.Fatalf("BoundedSubset: %v", err)
	}

	want := []string{"@F1@", "@F2@", "@I3@", "@R1@", "@S1@", "@U1@"}
	if got := recordXRefs(sub); !slices.Equal(got, want) {
		t.Errorf("records = %v, want %v", got, want)
	}
	assertSelfContained(t, sub)

	wantDropped := []gedcom.DroppedPointer{
		{XRef: "@F1@", Tag: "HUSB", Target: "@I1@", Restorable: true},
		{XRef: "@F1@", Tag: "WIFE", Target: "@I2@", Restorable: true},
		{XRef: "@F1@", Tag: "CHIL", Target: "@I4@", Restorable: true},
		{XRef: "@F1@", Tag: "CHIL", Target: "@I5@", Restorable: true},
		{XRef: "@F2@", Tag: "WIFE", Target: "@I6@", Restorable: true},
		{XRef: "@F2@", Tag: "CHIL", Target: "@I7@", Restorable: true},
	}
	if !reflect.DeepEqual(dropped, wantDropped) {
		t.Errorf("dropped = %+v\nwant %+v", dropped, wantDropped)
	}

	f1 := sub.GetFamily("@F1@")
	if f1.Husband != "" || f1.Wife != "" || !slices.Equal(f1.Children, []string{"@I3@"}) {
		t.Errorf("typed F1 = husband %q wife %q children %v", f1.Husband, f1.Wife, f1.Children)
	}
	// The CHIL kept its subordinate _FREL; the MARR and its citation stay.
	var lines []string
	for _, tag := range sub.GetRecord("@F1@").Tags {
		lines = append(lines, tag.Tag+" "+tag.Value)
	}
	wantLines := []string{"CHIL @I3@", "_FREL Natural", "MARR ", "DATE 1900", "SOUR @S1@"}
	if !slices.Equal(lines, wantLines) {
		t.Errorf("F1 lines = %q, want %q", lines, wantLines)
	}

	// The source is not mutated.
	if got := doc.GetFamily("@F1@").Husband; got != "@I1@" {
		t.Errorf("source F1 husband = %q", got)
	}
}

// TestBoundedSubset_TypedMatchesTags pins the typed pruning to what the
// decoder builds from the pruned Tags.
func TestBoundedSubset_TypedMatchesTags(t *testing.T) {
	doc := decodeBounded(t)
	for _, seeds := range [][]string{{"@I3@"}, {"@I4@"}, {"@I4@", "@I8@"}, {"@I6@", "@I9@"}, {"@I7@"}} {
		sub, _, err := doc.BoundedSubset(seeds)
		if err != nil {
			t.Fatalf("%v: %v", seeds, err)
		}
		rebuilt := sub.Clone()
		decoder.SyncEntities(rebuilt)
		for i, r := range sub.Records {
			if !reflect.DeepEqual(r.Entity, rebuilt.Records[i].Entity) {
				t.Errorf("%v: %s typed entity differs from one built from its Tags", seeds, r.XRef)
			}
		}
	}
}

func TestBoundedSubset_Association(t *testing.T) {
	doc := decodeBounded(t)
	sub, dropped, err := doc.BoundedSubset([]string{"@I4@"})
	if err != nil {
		t.Fatal(err)
	}
	if got := sub.GetIndividual("@I4@").Associations; len(got) != 0 {
		t.Errorf("associations = %+v, want none", got)
	}
	want := gedcom.DroppedPointer{XRef: "@I4@", Tag: "ASSO", Target: "@I8@", Restorable: false}
	if !slices.Contains(dropped, want) {
		t.Errorf("dropped = %+v, want it to hold %+v", dropped, want)
	}

	// With the associate inside the set, the ASSO stays.
	sub, _, err = doc.BoundedSubset([]string{"@I4@", "@I8@"})
	if err != nil {
		t.Fatal(err)
	}
	if got := sub.GetIndividual("@I4@").Associations; len(got) != 1 {
		t.Errorf("associations = %+v, want one", got)
	}
}

func TestBoundedSubset_AllIndividualsIsWholeDocument(t *testing.T) {
	doc := decodeBounded(t)
	var all []string
	for _, r := range doc.Records {
		if r.IsIndividual() {
			all = append(all, r.XRef)
		}
	}
	sub, dropped, err := doc.BoundedSubset(all)
	if err != nil {
		t.Fatal(err)
	}
	if len(dropped) != 0 {
		t.Errorf("dropped = %+v, want none", dropped)
	}
	if !slices.Equal(recordXRefs(sub), recordXRefs(doc)) {
		t.Errorf("records = %v, want %v", recordXRefs(sub), recordXRefs(doc))
	}
}

func TestBoundedSubset_Errors(t *testing.T) {
	doc := decodeBounded(t)

	_, _, err := doc.BoundedSubset([]string{"@I99@"})
	var unknown *gedcom.UnknownXRefError
	if !errors.As(err, &unknown) || !unknown.IsSeed || unknown.XRef != "@I99@" {
		t.Errorf("unknown seed: err = %v", err)
	}

	_, _, err = doc.BoundedSubset([]string{"@F1@"})
	if !errors.Is(err, gedcom.ErrNotIndividual) {
		t.Errorf("family seed: err = %v, want ErrNotIndividual", err)
	}

	var nilDoc *gedcom.Document
	if _, _, err := nilDoc.BoundedSubset(nil); err == nil {
		t.Error("nil document: want an error")
	}

	dangling, err := decoder.Decode(strings.NewReader(strings.Replace(boundedFixture, "1 NOTE @N1@", "1 NOTE @N9@", 1)))
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = dangling.BoundedSubset([]string{"@I6@"})
	if !errors.As(err, &unknown) || unknown.IsSeed || unknown.XRef != "@N9@" {
		t.Errorf("dangling reference: err = %v", err)
	}
}

func TestBoundedSubset_EmptySeedsKeepsSubmitter(t *testing.T) {
	doc := decodeBounded(t)
	sub, dropped, err := doc.BoundedSubset(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := recordXRefs(sub); !slices.Equal(got, []string{"@U1@"}) || len(dropped) != 0 {
		t.Errorf("records = %v, dropped = %v", got, dropped)
	}
	if sub.Header.Submitter != "@U1@" {
		t.Errorf("header submitter = %q", sub.Header.Submitter)
	}
}

// TestBoundedSubset_HandBuilt covers a record with no raw Tags, whose
// pointers are removed from the typed Entity and reported without a tag.
func TestBoundedSubset_HandBuilt(t *testing.T) {
	doc := gedcom.NewDocument(gedcom.Version551)
	add := func(r *gedcom.Record) {
		if err := doc.AddRecord(r); err != nil {
			t.Fatal(err)
		}
	}
	add(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{
		XRef:             "@I1@",
		SpouseInFamilies: []gedcom.FamilyLink{{FamilyXRef: "@F1@"}},
		Associations:     []*gedcom.Association{{IndividualXRef: "@I2@"}},
		Events:           []*gedcom.Event{{Type: "ADOP", FamilyXRef: "@F2@"}},
		Attributes: []*gedcom.Attribute{{Type: "OCCU", Associations: []*gedcom.Association{
			{IndividualXRef: "@I2@"}, {IndividualXRef: "@I1@"},
		}}},
	}})
	add(&gedcom.Record{XRef: "@I2@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: "@I2@"}})
	add(&gedcom.Record{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: &gedcom.Family{
		XRef: "@F1@", Husband: "@I1@", Wife: "@I2@", Children: []string{"@I2@"},
		ChildLinks: []gedcom.ChildLink{{XRef: "@I2@"}},
		Events:     []*gedcom.Event{{Type: "MARR", Associations: []*gedcom.Association{{IndividualXRef: "@I2@"}}}},
	}})
	add(&gedcom.Record{XRef: "@F2@", Type: gedcom.RecordTypeFamily, Entity: &gedcom.Family{XRef: "@F2@", Wife: "@I2@"}})

	sub, dropped, err := doc.BoundedSubset([]string{"@I1@"})
	if err != nil {
		t.Fatal(err)
	}
	assertSelfContained(t, sub)
	indi := sub.GetIndividual("@I1@")
	if len(indi.Associations) != 0 || indi.Events[0].FamilyXRef != "" ||
		len(indi.SpouseInFamilies) != 1 || indi.SpouseInFamilies[0].FamilyXRef != "@F1@" {
		t.Errorf("individual = %+v", indi)
	}
	if got := indi.Attributes[0].Associations; len(got) != 1 || got[0].IndividualXRef != "@I1@" {
		t.Errorf("attribute associations = %+v, want only @I1@", got)
	}
	fam := sub.GetFamily("@F1@")
	if fam.Husband != "@I1@" || fam.Wife != "" || len(fam.Children) != 0 || len(fam.ChildLinks) != 0 {
		t.Errorf("family = %+v", fam)
	}
	if got := fam.Events[0].Associations; got != nil {
		t.Errorf("family event associations = %+v, want none", got)
	}
	want := []gedcom.DroppedPointer{
		{XRef: "@I1@", Target: "@I2@"}, // ASSO
		{XRef: "@I1@", Target: "@F2@"}, // ADOP FAMC
		{XRef: "@I1@", Target: "@I2@"}, // OCCU ASSO
		{XRef: "@F1@", Target: "@I2@"}, // WIFE
		{XRef: "@F1@", Target: "@I2@"}, // CHIL
		{XRef: "@F1@", Target: "@I2@"}, // child link
		{XRef: "@F1@", Target: "@I2@"}, // MARR ASSO
	}
	if !reflect.DeepEqual(dropped, want) {
		t.Errorf("dropped = %+v, want %+v", dropped, want)
	}
}
