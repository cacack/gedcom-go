package merge_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/merge"
)

// recoveredXRefDoc holds one record whose XRef the decoder recovers from a
// malformed "0 @I1 INDI" line as the non-pointer "@I1", beside a valid @I9@.
const recoveredXRefDoc = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n" +
	"0 @I1 INDI\n1 NAME Broken /Xref/\n" +
	"0 @I9@ INDI\n1 NAME Valid /Xref/\n" +
	"0 TRLR\n"

func decodeRecovered(t *testing.T) *gedcom.Document {
	t.Helper()
	result, err := decoder.DecodeWithDiagnostics(strings.NewReader(recoveredXRefDoc), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics: %v", err)
	}
	if result.Document.GetIndividual("@I1") == nil {
		t.Fatal("precondition: recovered record @I1 missing")
	}
	return result.Document
}

// TestRemapXRefs_RecoveredNonPointerXRef pins #397: a recovered non-pointer
// XRef is skipped, as gedcom.Visit skips it, instead of failing the whole
// call; valid records are still remapped.
func TestRemapXRefs_RecoveredNonPointerXRef(t *testing.T) {
	transforms := map[string]func(string) string{
		"identity": func(old string) string { return old },
		"prefix":   prefixA,
	}
	for name, transform := range transforms {
		t.Run(name, func(t *testing.T) {
			doc := decodeRecovered(t)
			out, mapping, err := merge.RemapXRefs(doc, transform)
			if err != nil {
				t.Fatalf("RemapXRefs: %v", err)
			}
			if _, ok := mapping["@I1"]; ok {
				t.Errorf("mapping has entry for non-pointer @I1: %q", mapping)
			}
			if want := transform("@I9@"); mapping["@I9@"] != want || out.GetIndividual(want) == nil {
				t.Errorf("@I9@ not remapped to %s: mapping %q", want, mapping)
			}
			if out.GetIndividual("@I1") == nil {
				t.Error("recovered record @I1 missing or renamed")
			}
		})
	}
}

// TestCombine_RecoveredNonPointerXRef pins #397: Combine succeeds when doc2
// holds a recovered non-pointer XRef, and remaps doc2's colliding records.
func TestCombine_RecoveredNonPointerXRef(t *testing.T) {
	doc1, err := decoder.Decode(strings.NewReader(
		"0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n0 @I9@ INDI\n1 NAME Other /Person/\n0 TRLR\n"))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	doc2 := decodeRecovered(t)
	for _, strategy := range []merge.CollisionStrategy{merge.PrefixDoc2, merge.RenumberDoc2} {
		out, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{CollisionStrategy: strategy, Prefix: "B_"})
		if err != nil {
			t.Fatalf("Combine(strategy %d): %v", strategy, err)
		}
		if out.GetIndividual("@I1") == nil {
			t.Errorf("strategy %d: recovered record @I1 missing", strategy)
		}
		newXRef, ok := report.RemappedXRefs["@I9@"]
		if !ok || out.GetIndividual(newXRef) == nil {
			t.Errorf("strategy %d: doc2's @I9@ not remapped: %q", strategy, report.RemappedXRefs)
		}
	}
}

// TestCombine_SharedNonPointerXRefCollides pins that Combine cannot rename a
// non-pointer XRef both documents hold, so it reports a collision rather than
// emitting two records with one identifier.
func TestCombine_SharedNonPointerXRefCollides(t *testing.T) {
	doc1, doc2 := decodeRecovered(t), decodeRecovered(t)
	for _, strategy := range []merge.CollisionStrategy{merge.ErrorOnCollision, merge.PrefixDoc2, merge.RenumberDoc2} {
		_, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{CollisionStrategy: strategy, Prefix: "B_"})
		if !errors.Is(err, merge.ErrXRefCollision) {
			t.Errorf("strategy %d: err = %v, want ErrXRefCollision", strategy, err)
		}
	}
}
