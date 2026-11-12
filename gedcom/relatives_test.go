package gedcom

import (
	"math"
	"reflect"
	"testing"
)

func TestAgeAt(t *testing.T) {
	tests := []struct {
		name             string
		birth, event     string
		youngest, oldest int
		ok               bool
	}{
		{"complete dates", "15 MAR 1920", "20 JUN 1985", 65, 65, true},
		{"day before birthday", "15 MAR 1920", "14 MAR 1985", 64, 64, true},
		{"on birthday", "15 MAR 1920", "15 MAR 1985", 65, 65, true},
		{"leap birthday before 1 MAR", "29 FEB 2000", "28 FEB 2001", 0, 0, true},
		{"leap birthday on 1 MAR", "29 FEB 2000", "1 MAR 2001", 1, 1, true},
		{"year-only dates", "1900", "1950", 49, 50, true},
		{"month-only dates", "DEC 1800", "JAN 1921", 120, 120, true},
		{"same year", "1900", "1900", 0, 0, true},
		{"BET range", "BET 1900 AND 1910", "1 JAN 1950", 39, 50, true},
		{"BEF birth leaves oldest open", "BEF 1900", "1950", 49, math.MaxInt, true},
		{"AFT event leaves oldest open", "1900", "AFT 1950", 49, math.MaxInt, true},
		{"AFT birth leaves youngest at zero", "AFT 1900", "1950", 0, 50, true},
		{"BEF event leaves youngest at zero", "1900", "BEF 1950", 0, 50, true},
		{"overlapping event clamps youngest", "1900", "BET 1890 AND 1920", 0, 20, true},
		{"cross calendar", "@#DJULIAN@ 1 JAN 1900", "14 JAN 1950", 50, 50, true},
		{"BCE across era", "1 JAN 10 BCE", "1 JAN 10", 19, 19, true},
		{"event before birth", "1950", "1900", 0, 0, false},
		{"nonexistent day", "31 FEB 1900", "1950", 0, 0, false},
		{"phrase", "(unknown)", "1950", 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			birth, err := ParseDate(tt.birth)
			if err != nil {
				t.Fatalf("ParseDate(%q): %v", tt.birth, err)
			}
			event, err := ParseDate(tt.event)
			if err != nil {
				t.Fatalf("ParseDate(%q): %v", tt.event, err)
			}
			youngest, oldest, ok := AgeAt(birth, event)
			if youngest != tt.youngest || oldest != tt.oldest || ok != tt.ok {
				t.Errorf("AgeAt(%q, %q) = (%d, %d, %v), want (%d, %d, %v)",
					tt.birth, tt.event, youngest, oldest, ok, tt.youngest, tt.oldest, tt.ok)
			}
		})
	}
}

func TestAgeAt_Nil(t *testing.T) {
	d, _ := ParseDate("1900")
	if _, _, ok := AgeAt(nil, d); ok {
		t.Error("AgeAt(nil, d) ok = true")
	}
	if _, _, ok := AgeAt(d, nil); ok {
		t.Error("AgeAt(d, nil) ok = true")
	}
}

// relativesDoc builds: @F1@ = @I1@ + @I2@ with children @I3@ (birth) and
// @I4@ (adopted); @F2@ = @I1@ + @I5@ with children @I6@ and @I4@ (birth);
// @F3@ = @I7@ + @I8@ fostering @I3@; @F4@ = @I2@ alone with child @I9@ and a
// dangling child pointer.
func relativesDoc() *Document {
	inds := []*Individual{
		{XRef: "@I1@", SpouseInFamilies: []FamilyLink{{FamilyXRef: "@F1@"}, {FamilyXRef: "@F2@"}}},
		{XRef: "@I2@", SpouseInFamilies: []FamilyLink{{FamilyXRef: "@F1@"}, {FamilyXRef: "@F4@"}}},
		{XRef: "@I3@", ChildInFamilies: []FamilyLink{
			{FamilyXRef: "@F1@", Pedigree: "birth"},
			{FamilyXRef: "@F3@", Pedigree: "foster"},
			{FamilyXRef: "@MISSING@"},
		}},
		{XRef: "@I4@", ChildInFamilies: []FamilyLink{
			{FamilyXRef: "@F1@", Pedigree: "adopted"},
			{FamilyXRef: "@F2@", Pedigree: "birth"},
		}},
		{XRef: "@I5@", SpouseInFamilies: []FamilyLink{{FamilyXRef: "@F2@"}}},
		{XRef: "@I6@", ChildInFamilies: []FamilyLink{{FamilyXRef: "@F2@"}}},
		{XRef: "@I7@", SpouseInFamilies: []FamilyLink{{FamilyXRef: "@F3@"}}},
		{XRef: "@I8@", SpouseInFamilies: []FamilyLink{{FamilyXRef: "@F3@"}}},
		{XRef: "@I9@", ChildInFamilies: []FamilyLink{{FamilyXRef: "@F4@"}}},
	}
	fams := []*Family{
		{XRef: "@F1@", Husband: "@I1@", Wife: "@I2@", Children: []string{"@I3@", "@I4@"}},
		{XRef: "@F2@", Husband: "@I1@", Wife: "@I5@", Children: []string{"@I6@", "@I4@"}},
		{XRef: "@F3@", Husband: "@I7@", Wife: "@I8@", Children: []string{"@I3@"}},
		{XRef: "@F4@", Wife: "@I2@", Children: []string{"@I9@", "@GONE@"}},
	}
	return createRelationshipTestDocument(inds, fams)
}

func TestIndividual_ParentLinks(t *testing.T) {
	doc := relativesDoc()
	type got struct{ parent, family, pedigree string }
	var links []got
	for _, l := range doc.GetIndividual("@I3@").ParentLinks(doc) {
		links = append(links, got{l.Individual.XRef, l.Family.XRef, l.Pedigree})
	}
	want := []got{
		{"@I1@", "@F1@", "birth"},
		{"@I2@", "@F1@", "birth"},
		{"@I7@", "@F3@", "foster"},
		{"@I8@", "@F3@", "foster"},
	}
	if !reflect.DeepEqual(links, want) {
		t.Errorf("ParentLinks = %v, want %v", links, want)
	}

	// Same parents, same order, as Parents.
	var fromParents []string
	for _, p := range doc.GetIndividual("@I3@").Parents(doc) {
		fromParents = append(fromParents, p.XRef)
	}
	if want := []string{"@I1@", "@I2@", "@I7@", "@I8@"}; !reflect.DeepEqual(fromParents, want) {
		t.Errorf("Parents = %v, want %v", fromParents, want)
	}

	if got := doc.GetIndividual("@I3@").ParentLinks(nil); got != nil {
		t.Errorf("ParentLinks(nil) = %v, want nil", got)
	}
	if got := doc.GetIndividual("@I1@").ParentLinks(doc); got != nil {
		t.Errorf("ParentLinks of individual without parents = %v, want nil", got)
	}
}

func TestIndividual_Siblings(t *testing.T) {
	doc := relativesDoc()
	type got struct {
		sibling, family string
		half            bool
	}
	siblingsOf := func(xref string) []got {
		var out []got
		for _, s := range doc.GetIndividual(xref).Siblings(doc) {
			out = append(out, got{s.Individual.XRef, s.Family.XRef, s.Half})
		}
		return out
	}

	tests := []struct {
		xref string
		want []got
	}{
		// @I4@ is reachable as full (F1) and half (via @I1@'s F2); it appears
		// once, as full. F3's parents have no other children, and the
		// dangling FAMC and CHIL pointers are skipped.
		{"@I3@", []got{{"@I4@", "@F1@", false}, {"@I6@", "@F2@", true}, {"@I9@", "@F4@", true}}},
		// Children of both of @I4@'s families are full siblings.
		{"@I4@", []got{{"@I3@", "@F1@", false}, {"@I6@", "@F2@", false}, {"@I9@", "@F4@", true}}},
		{"@I6@", []got{{"@I4@", "@F2@", false}, {"@I3@", "@F1@", true}}},
		{"@I9@", []got{{"@I3@", "@F1@", true}, {"@I4@", "@F1@", true}}},
		{"@I1@", nil},
	}
	for _, tt := range tests {
		t.Run(tt.xref, func(t *testing.T) {
			if got := siblingsOf(tt.xref); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Siblings(%s) = %v, want %v", tt.xref, got, tt.want)
			}
		})
	}

	if got := doc.GetIndividual("@I3@").Siblings(nil); got != nil {
		t.Errorf("Siblings(nil) = %v, want nil", got)
	}
}
