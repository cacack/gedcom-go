package gedcom

import (
	"fmt"
	"testing"

	"github.com/cacack/gedcom-go/v3/internal/scalecheck"
)

// largeFamily builds a document with one family @F1@ of n reciprocated
// children: CHIL on the family and a FAMC on each child.
func largeFamily(n int) *Document {
	t := newLinkTree().indi("@P@", []string{"@F1@"}, nil)
	children := make([]string, n)
	for k := range children {
		children[k] = fmt.Sprintf("@C%d@", k)
		t.indi(children[k], nil, []string{"@F1@"})
	}
	t.fam("@F1@", "@P@", "", children...)
	return t.doc
}

// TestReciprocity_LinearInFamilySize guards that walking a family's CHIL
// list under the reciprocal-link rule stays linear in its size. The
// measurement is scalecheck.Linear's: one walk of a large family against as
// many walks of a small one as make up the same number of children.
func TestReciprocity_LinearInFamilySize(t *testing.T) {
	scalecheck.Linear(t, "Descendants", 2000, 32000, func(n int) func() {
		doc := largeFamily(n)
		return func() {
			if got := len(doc.Descendants("@P@")); got != n {
				t.Fatalf("Descendants returned %d, want %d", got, n)
			}
		}
	})
}

func BenchmarkDescendants_LargeFamily(b *testing.B) {
	doc := largeFamily(40000)
	b.ResetTimer()
	for b.Loop() {
		_ = doc.Descendants("@P@")
	}
}
