package decoder

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestDecodeChildLinksFTM checks that _FREL/_MREL under FAM.CHIL fill
// Family.ChildLinks in Family Tree Maker exports (#732), leaving Children as
// it was.
func TestDecodeChildLinksFTM(t *testing.T) {
	doc := decodeFixture(t, "../testdata/gedcom-5.5/pres2020.ged").Document

	fam := doc.GetFamily("@F1@")
	if fam == nil {
		t.Fatal("pres2020.ged has no @F1@")
	}
	if want := []string{"@I66@"}; !reflect.DeepEqual(fam.Children, want) {
		t.Errorf("@F1@ Children = %v, want %v", fam.Children, want)
	}
	want := []gedcom.ChildLink{{XRef: "@I66@", FatherRelation: "Natural", MotherRelation: "Natural"}}
	if !reflect.DeepEqual(fam.ChildLinks, want) {
		t.Errorf("@F1@ ChildLinks = %+v, want %+v", fam.ChildLinks, want)
	}

	// Every CHIL with either tag has a link (1,246 of them), each naming one
	// of its family's children; pres2020 records every child as Natural.
	links := 0
	values := map[string]int{}
	for _, fam := range doc.Families() {
		for _, link := range fam.ChildLinks {
			links++
			values["_FREL "+link.FatherRelation]++
			values["_MREL "+link.MotherRelation]++
			if !containsString(fam.Children, link.XRef) {
				t.Errorf("%s ChildLink %s is not among Children %v", fam.XRef, link.XRef, fam.Children)
			}
		}
	}
	if links != 1246 {
		t.Errorf("pres2020.ged has %d ChildLinks, want 1246", links)
	}
	for v := range values {
		if v != "_FREL Natural" && v != "_MREL Natural" && v != "_FREL " && v != "_MREL " {
			t.Errorf("unexpected relationship %q in pres2020.ged", v)
		}
	}
}

// TestDecodeChildLinksNonNatural checks the non-natural values and a CHIL
// carrying only one of the two tags.
func TestDecodeChildLinksNonNatural(t *testing.T) {
	ftm := decodeFixture(t, "../testdata/edge-cases/vendor-ftm.ged").Document
	if got, want := ftm.GetFamily("@F1@").ChildLinks,
		[]gedcom.ChildLink{{XRef: "@I1@", FatherRelation: "Step", MotherRelation: "Guardian"}}; !reflect.DeepEqual(got, want) {
		t.Errorf("vendor-ftm @F1@ ChildLinks = %+v, want %+v", got, want)
	}

	complexDoc := decodeFixture(t, "../testdata/edge-cases/relationships-complex.ged").Document
	fam := complexDoc.GetFamily("@F00003@")
	if got := fam.LinkForChild("@I00015@"); got == nil || *got != (gedcom.ChildLink{
		XRef: "@I00015@", FatherRelation: "Natural", MotherRelation: "Step",
	}) {
		t.Errorf("@F00003@ LinkForChild(@I00015@) = %+v, want Natural/Step", got)
	}
	if got := complexDoc.GetFamily("@F00004@").LinkForChild("@I00008@"); got == nil ||
		got.FatherRelation != "Natural" || got.MotherRelation != "" {
		t.Errorf("@F00004@ LinkForChild(@I00008@) = %+v, want _FREL Natural only", got)
	}
}

// TestDecodeChildLinksAbsent checks that a CHIL with no relationship tags
// adds no ChildLink, and that a relationship value is trimmed.
func TestDecodeChildLinksAbsent(t *testing.T) {
	const input = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @F1@ FAM\n1 CHIL @I1@\n1 CHIL @I2@\n2 _MREL  Adopted \n0 TRLR\n"
	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	fam := doc.GetFamily("@F1@")
	if want := []string{"@I1@", "@I2@"}; !reflect.DeepEqual(fam.Children, want) {
		t.Errorf("Children = %v, want %v", fam.Children, want)
	}
	want := []gedcom.ChildLink{{XRef: "@I2@", MotherRelation: "Adopted"}}
	if !reflect.DeepEqual(fam.ChildLinks, want) {
		t.Errorf("ChildLinks = %+v, want %+v", fam.ChildLinks, want)
	}
	if fam.LinkForChild("@I1@") != nil {
		t.Error("LinkForChild(@I1@) is non-nil for a CHIL with no relationship tags")
	}
}

// TestDecodeMarriedNameMyHeritage checks that _MARNM under INDI.NAME fills
// PersonalName.MarriedName in a MyHeritage Family Tree Builder export (#732).
func TestDecodeMarriedNameMyHeritage(t *testing.T) {
	doc := decodeFixture(t, "../testdata/edge-cases/mhftb8-export.ged").Document

	indi := doc.GetIndividual("@I589@")
	if indi == nil || len(indi.Names) == 0 {
		t.Fatal("mhftb8-export.ged has no named @I589@")
	}
	name := indi.Names[0]
	if name.MarriedName != "Gascoigne" || name.Surname != "Franke" {
		t.Errorf("@I589@ name: MarriedName %q, Surname %q; want Gascoigne, Franke", name.MarriedName, name.Surname)
	}

	married := 0
	for _, indi := range doc.Individuals() {
		for _, n := range indi.Names {
			if n != nil && n.MarriedName != "" {
				married++
			}
		}
	}
	if married != 724 {
		t.Errorf("mhftb8-export.ged has %d names with a MarriedName, want 724 (one per _MARNM)", married)
	}
}

func containsString(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
