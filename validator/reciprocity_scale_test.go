package validator

import (
	"fmt"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/internal/scalecheck"
)

// largeFamilyDoc builds a document with one family @F1@ of n reciprocated
// children: CHIL on the family and a FAMC on each child.
func largeFamilyDoc(n int) *gedcom.Document {
	doc := &gedcom.Document{XRefMap: map[string]*gedcom.Record{}}
	add := func(xref string, typ gedcom.RecordType, entity interface{}) {
		rec := &gedcom.Record{XRef: xref, Type: typ, Entity: entity}
		doc.Records = append(doc.Records, rec)
		doc.XRefMap[xref] = rec
	}
	fam := &gedcom.Family{XRef: "@F1@", Children: make([]string, n)}
	for k := range fam.Children {
		fam.Children[k] = fmt.Sprintf("@C%d@", k)
		add(fam.Children[k], gedcom.RecordTypeIndividual, &gedcom.Individual{
			XRef:            fam.Children[k],
			ChildInFamilies: []gedcom.FamilyLink{{FamilyXRef: "@F1@"}},
		})
	}
	add("@F1@", gedcom.RecordTypeFamily, fam)
	return doc
}

// TestReferenceValidator_LinearInFamilySize guards that the FAMC and CHIL
// reciprocity checks stay linear in a family's size. The measurement is
// scalecheck.Linear's: one validation of a large family against as many
// validations of a small one as make up the same number of children.
func TestReferenceValidator_LinearInFamilySize(t *testing.T) {
	scalecheck.Linear(t, "Validate", 2000, 32000, func(n int) func() {
		doc := largeFamilyDoc(n)
		return func() {
			if issues := NewReferenceValidator().Validate(doc); len(issues) != 0 {
				t.Fatalf("Validate reported %d issues on a reciprocated family, want 0", len(issues))
			}
		}
	})
}

func BenchmarkReferenceValidator_LargeFamily(b *testing.B) {
	doc := largeFamilyDoc(40000)
	v := NewReferenceValidator()
	b.ResetTimer()
	for b.Loop() {
		_ = v.Validate(doc)
	}
}
