package gedcom

import (
	"fmt"
	"testing"
	"time"
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

// fastest returns the shortest of a few runs of f, to damp scheduler noise.
func fastest(f func()) time.Duration {
	best := time.Duration(1<<63 - 1)
	for range 5 {
		start := time.Now()
		f()
		if d := time.Since(start); d < best {
			best = d
		}
	}
	return best
}

// TestReciprocity_LinearInFamilySize guards that walking a family's CHIL
// list under the reciprocal-link rule stays linear in its size: 8x the
// children must cost well under the 64x a quadratic walk would.
func TestReciprocity_LinearInFamilySize(t *testing.T) {
	const small, large = 2000, 16000
	run := func(n int) time.Duration {
		doc := largeFamily(n)
		return fastest(func() {
			if got := len(doc.Descendants("@P@")); got != n {
				t.Fatalf("Descendants returned %d, want %d", got, n)
			}
		})
	}
	tSmall, tLarge := run(small), run(large)
	if ratio := float64(tLarge) / float64(tSmall); ratio > 24 {
		t.Errorf("Descendants: %d children took %v, %d took %v (%.0fx for 8x the children); want linear",
			small, tSmall, large, tLarge, ratio)
	}
}

func BenchmarkDescendants_LargeFamily(b *testing.B) {
	doc := largeFamily(40000)
	b.ResetTimer()
	for b.Loop() {
		_ = doc.Descendants("@P@")
	}
}
