package validator

import (
	"fmt"
	"testing"
	"time"

	"github.com/cacack/gedcom-go/v3/gedcom"
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
// reciprocity checks stay linear in a family's size: 8x the children must
// cost well under the 64x a quadratic check would.
func TestReferenceValidator_LinearInFamilySize(t *testing.T) {
	const small, large = 2000, 16000
	run := func(n int) time.Duration {
		doc := largeFamilyDoc(n)
		best := time.Duration(1<<63 - 1)
		for range 5 {
			start := time.Now()
			if issues := NewReferenceValidator().Validate(doc); len(issues) != 0 {
				t.Fatalf("Validate reported %d issues on a reciprocated family, want 0", len(issues))
			}
			if d := time.Since(start); d < best {
				best = d
			}
		}
		return best
	}
	tSmall, tLarge := run(small), run(large)
	if ratio := float64(tLarge) / float64(tSmall); ratio > 24 {
		t.Errorf("Validate: %d children took %v, %d took %v (%.0fx for 8x the children); want linear",
			small, tSmall, large, tLarge, ratio)
	}
}

func BenchmarkReferenceValidator_LargeFamily(b *testing.B) {
	doc := largeFamilyDoc(40000)
	v := NewReferenceValidator()
	b.ResetTimer()
	for b.Loop() {
		_ = v.Validate(doc)
	}
}
