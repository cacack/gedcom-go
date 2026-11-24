package gedcom

import (
	"reflect"
	"testing"
)

// linkTree builds a document from individuals and families written as their
// FAMS/FAMC and HUSB/WIFE/CHIL pointers, so a test can leave out one side of
// a link.
type linkTree struct{ doc *Document }

func newLinkTree() *linkTree {
	return &linkTree{doc: &Document{XRefMap: map[string]*Record{}}}
}

func (t *linkTree) add(xref string, typ RecordType, entity interface{}) {
	rec := &Record{XRef: xref, Type: typ, Entity: entity}
	t.doc.Records = append(t.doc.Records, rec)
	t.doc.XRefMap[xref] = rec
}

func (t *linkTree) indi(xref string, fams, famc []string) *linkTree {
	ind := &Individual{XRef: xref}
	for _, f := range fams {
		ind.SpouseInFamilies = append(ind.SpouseInFamilies, FamilyLink{FamilyXRef: f})
	}
	for _, f := range famc {
		ind.ChildInFamilies = append(ind.ChildInFamilies, FamilyLink{FamilyXRef: f})
	}
	t.add(xref, RecordTypeIndividual, ind)
	return t
}

func (t *linkTree) fam(xref, husband, wife string, children ...string) *linkTree {
	t.add(xref, RecordTypeFamily, &Family{XRef: xref, Husband: husband, Wife: wife, Children: children})
	return t
}

func indXRefs(inds []*Individual) []string {
	out := []string{}
	for _, ind := range inds {
		out = append(out, ind.XRef)
	}
	return out
}

func famXRefs(fams []*Family) []string {
	out := []string{}
	for _, fam := range fams {
		out = append(out, fam.XRef)
	}
	return out
}

func siblingXRefs(sibs []Sibling) []string {
	out := []string{}
	for _, s := range sibs {
		out = append(out, s.Individual.XRef)
	}
	return out
}

func wantXRefs(t *testing.T, what string, got []string, want ...string) {
	t.Helper()
	if want == nil {
		want = []string{}
	}
	if got == nil {
		got = []string{}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

func TestIsSpouseInIsChildIn(t *testing.T) {
	doc := newLinkTree().
		indi("@I1@", []string{"@F1@"}, nil).
		indi("@I2@", nil, []string{"@F1@"}).
		indi("@I3@", []string{"@F1@"}, []string{"@F1@"}).
		fam("@F1@", "@I1@", "", "@I2@").doc
	fam := doc.GetFamily("@F1@")

	tests := []struct {
		name          string
		ind           *Individual
		fam           *Family
		spouse, child bool
	}{
		{"husband both sides", doc.GetIndividual("@I1@"), fam, true, false},
		{"child both sides", doc.GetIndividual("@I2@"), fam, false, true},
		{"links the family does not reciprocate", doc.GetIndividual("@I3@"), fam, false, false},
		{"nil individual", nil, fam, false, false},
		{"nil family", doc.GetIndividual("@I1@"), nil, false, false},
		{"empty xrefs never match", &Individual{SpouseInFamilies: []FamilyLink{{}}}, &Family{}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ind.IsSpouseIn(tt.fam); got != tt.spouse {
				t.Errorf("IsSpouseIn = %v, want %v", got, tt.spouse)
			}
			if got := tt.ind.IsChildIn(tt.fam); got != tt.child {
				t.Errorf("IsChildIn = %v, want %v", got, tt.child)
			}
		})
	}
}

// TestReciprocity_Issue930 is the fixture from #930: @I2@ carries a FAMS to
// @F2@, which names @I3@ and @I4@, so @I5@ is not a descendant of @I1@.
func TestReciprocity_Issue930(t *testing.T) {
	doc := newLinkTree().
		indi("@I1@", []string{"@F1@"}, nil).
		indi("@I2@", []string{"@F2@"}, []string{"@F1@"}).
		indi("@I3@", []string{"@F2@"}, nil).
		indi("@I4@", []string{"@F2@"}, nil).
		indi("@I5@", nil, []string{"@F2@"}).
		fam("@F1@", "@I1@", "", "@I2@").
		fam("@F2@", "@I3@", "@I4@", "@I5@").doc

	wantXRefs(t, "Descendants(@I1@)", doc.Descendants("@I1@"), "@I2@")
	if got, want := doc.DescendantsWithGenerations("@I1@", 0), []Relative{{"@I2@", 1}}; !reflect.DeepEqual(got, want) {
		t.Errorf("DescendantsWithGenerations(@I1@) = %v, want %v", got, want)
	}
	wantXRefs(t, "Ancestors(@I5@)", doc.Ancestors("@I5@"), "@I3@", "@I4@")
}

// TestReciprocity_OneWayFAMS covers a FAMS whose family names other people
// as HUSB and WIFE.
func TestReciprocity_OneWayFAMS(t *testing.T) {
	doc := newLinkTree().
		indi("@I1@", []string{"@F1@"}, nil). // stray FAMS
		indi("@I2@", []string{"@F1@"}, nil).
		indi("@I3@", []string{"@F1@"}, nil).
		indi("@I4@", nil, []string{"@F1@"}).
		fam("@F1@", "@I2@", "@I3@", "@I4@").doc
	stray := doc.GetIndividual("@I1@")

	wantXRefs(t, "FamiliesAsSpouse", famXRefs(stray.FamiliesAsSpouse(doc)))
	wantXRefs(t, "Spouses", indXRefs(stray.Spouses(doc)))
	wantXRefs(t, "Children", indXRefs(stray.Children(doc)))
	wantXRefs(t, "Descendants", doc.Descendants("@I1@"))
	wantXRefs(t, "Ancestors(@I4@)", doc.Ancestors("@I4@"), "@I2@", "@I3@")
}

// TestReciprocity_OneWayFAMC covers a FAMC whose family does not name the
// individual as CHIL.
func TestReciprocity_OneWayFAMC(t *testing.T) {
	doc := newLinkTree().
		indi("@I1@", []string{"@F1@"}, nil).
		indi("@I2@", nil, []string{"@F1@"}).
		indi("@I3@", nil, []string{"@F1@"}). // stray FAMC
		fam("@F1@", "@I1@", "", "@I2@").doc
	stray := doc.GetIndividual("@I3@")

	wantXRefs(t, "FamiliesAsChild", famXRefs(stray.FamiliesAsChild(doc)))
	wantXRefs(t, "Parents", indXRefs(stray.Parents(doc)))
	if got := stray.ParentLinks(doc); len(got) != 0 {
		t.Errorf("ParentLinks = %v, want none", got)
	}
	wantXRefs(t, "Siblings", siblingXRefs(stray.Siblings(doc)))
	wantXRefs(t, "Ancestors", doc.Ancestors("@I3@"))
	wantXRefs(t, "Children(@I1@)", indXRefs(doc.GetIndividual("@I1@").Children(doc)), "@I2@")
	wantXRefs(t, "Descendants(@I1@)", doc.Descendants("@I1@"), "@I2@")
	wantXRefs(t, "Siblings(@I2@)", siblingXRefs(doc.GetIndividual("@I2@").Siblings(doc)))
}

// TestReciprocity_OneWayHUSBWIFE covers a HUSB and a WIFE naming people who
// carry no FAMS to the family.
func TestReciprocity_OneWayHUSBWIFE(t *testing.T) {
	doc := newLinkTree().
		indi("@I1@", nil, nil). // named as HUSB, no FAMS
		indi("@I2@", []string{"@F1@"}, nil).
		indi("@I3@", nil, []string{"@F1@"}).
		indi("@I4@", nil, nil). // named as WIFE, no FAMS
		indi("@I5@", []string{"@F2@"}, nil).
		fam("@F1@", "@I1@", "@I2@", "@I3@").
		fam("@F2@", "@I5@", "@I4@").doc
	f1, f2 := doc.GetFamily("@F1@"), doc.GetFamily("@F2@")

	if got := f1.HusbandIndividual(doc); got != nil {
		t.Errorf("HusbandIndividual = %s, want nil", got.XRef)
	}
	if got := f2.WifeIndividual(doc); got != nil {
		t.Errorf("WifeIndividual = %s, want nil", got.XRef)
	}
	wantXRefs(t, "AllMembers(@F1@)", indXRefs(f1.AllMembers(doc)), "@I2@", "@I3@")
	wantXRefs(t, "Spouses(@I2@)", indXRefs(doc.GetIndividual("@I2@").Spouses(doc)))
	wantXRefs(t, "Spouses(@I5@)", indXRefs(doc.GetIndividual("@I5@").Spouses(doc)))
	wantXRefs(t, "Parents(@I3@)", indXRefs(doc.GetIndividual("@I3@").Parents(doc)), "@I2@")
	wantXRefs(t, "Ancestors(@I3@)", doc.Ancestors("@I3@"), "@I2@")
	wantXRefs(t, "Children(@I1@)", indXRefs(doc.GetIndividual("@I1@").Children(doc)))
	wantXRefs(t, "Descendants(@I1@)", doc.Descendants("@I1@"))
	wantXRefs(t, "FamiliesAsSpouse(@I1@)", famXRefs(doc.GetIndividual("@I1@").FamiliesAsSpouse(doc)))
}

// TestReciprocity_OneWayCHIL covers a CHIL naming a person who carries no
// FAMC to the family.
func TestReciprocity_OneWayCHIL(t *testing.T) {
	doc := newLinkTree().
		indi("@I1@", []string{"@F1@"}, nil).
		indi("@I2@", nil, []string{"@F1@"}).
		indi("@I3@", nil, nil). // named as CHIL, no FAMC
		fam("@F1@", "@I1@", "", "@I2@", "@I3@").doc
	stray := doc.GetIndividual("@I3@")

	wantXRefs(t, "ChildrenIndividuals", indXRefs(doc.GetFamily("@F1@").ChildrenIndividuals(doc)), "@I2@")
	wantXRefs(t, "Children(@I1@)", indXRefs(doc.GetIndividual("@I1@").Children(doc)), "@I2@")
	wantXRefs(t, "Descendants(@I1@)", doc.Descendants("@I1@"), "@I2@")
	wantXRefs(t, "Siblings(@I2@)", siblingXRefs(doc.GetIndividual("@I2@").Siblings(doc)))
	wantXRefs(t, "Parents(@I3@)", indXRefs(stray.Parents(doc)))
	wantXRefs(t, "Ancestors(@I3@)", doc.Ancestors("@I3@"))
}
