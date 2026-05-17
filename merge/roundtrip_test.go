package merge_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/cacack/gedcom-go/decoder"
	"github.com/cacack/gedcom-go/encoder"
	"github.com/cacack/gedcom-go/gedcom"
	"github.com/cacack/gedcom-go/merge"
	"github.com/cacack/gedcom-go/validator"
)

// TestCombine_Roundtrip_PrefixDoc2 exercises the end-to-end pipeline:
// decode → combine (with collision remap) → encode → decode → assert
// equivalence.
//
// Fixtures: both inputs are GEDCOM 5.5.1 files so version/encoding
// compatibility holds. We combine comprehensive.ged with minimal.ged;
// they both define @I1@, guaranteeing the PrefixDoc2 strategy is
// exercised on at least one record.
func TestCombine_Roundtrip_PrefixDoc2(t *testing.T) {
	doc1 := mustRoundtripDecode(t, "../testdata/gedcom-5.5.1/comprehensive.ged")
	doc2 := mustRoundtripDecode(t, "../testdata/gedcom-5.5.1/minimal.ged")

	combined, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{
		CollisionStrategy: merge.PrefixDoc2,
		Prefix:            "b_",
	})
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}
	if len(report.RemappedXRefs) == 0 {
		t.Fatal("expected at least one remapped XRef given the @I1@ collision; got none")
	}

	var buf bytes.Buffer
	if err := encoder.Encode(&buf, combined); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	roundtripped, err := decoder.Decode(&buf)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	// Record count preserved through the round-trip.
	if got, want := len(roundtripped.Records), len(combined.Records); got != want {
		t.Errorf("record count after round-trip: got %d want %d", got, want)
	}

	// Every XRef in the combined document survives the round-trip.
	for xref := range combined.XRefMap {
		if roundtripped.GetRecord(xref) == nil {
			t.Errorf("xref %q present in combined but missing from round-tripped doc", xref)
		}
	}

	// Every remapped (new) XRef resolves in the round-tripped doc.
	for oldXRef, newXRef := range report.RemappedXRefs {
		if roundtripped.GetRecord(newXRef) == nil {
			t.Errorf("remapped xref %q (was %q) missing from round-trip", newXRef, oldXRef)
		}
	}

	// Round-tripped doc validates cleanly.
	v := validator.New()
	if errs := v.Validate(roundtripped); len(errs) > 0 {
		t.Errorf("validation errors after round-trip: %v", errs)
	}
}

// TestCombine_Roundtrip_NoCollision combines two fixtures that have
// disjoint XRef namespaces using ErrorOnCollision. The combine should
// succeed without remapping, and the result should round-trip cleanly.
//
// We construct disjoint inputs by combining a fixture with a remapped
// copy of itself (every XRef rewritten into a non-colliding namespace),
// which guarantees no collisions while still exercising the full
// header-merge + record-concatenation path.
func TestCombine_Roundtrip_NoCollision(t *testing.T) {
	doc1 := mustRoundtripDecode(t, "../testdata/gedcom-5.5.1/comprehensive.ged")

	// Build a non-colliding "doc2" by remapping comprehensive.ged into a
	// disjoint XRef namespace.
	doc2Source := mustRoundtripDecode(t, "../testdata/gedcom-5.5.1/comprehensive.ged")
	doc2, _, err := merge.RemapXRefs(doc2Source, func(old string) string {
		// Strip the leading "@" and prepend "@x_" so the result keeps
		// the @xref@ pointer shape but cannot collide with doc1.
		return "@x_" + old[1:]
	})
	if err != nil {
		t.Fatalf("RemapXRefs to build disjoint doc2: %v", err)
	}

	combined, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{
		CollisionStrategy: merge.ErrorOnCollision,
	})
	if err != nil {
		t.Fatalf("Combine with disjoint inputs returned error: %v", err)
	}
	if len(report.RemappedXRefs) != 0 {
		t.Errorf("expected empty RemappedXRefs for disjoint combine; got %d entries", len(report.RemappedXRefs))
	}

	var buf bytes.Buffer
	if err := encoder.Encode(&buf, combined); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	roundtripped, err := decoder.Decode(&buf)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if got, want := len(roundtripped.Records), len(combined.Records); got != want {
		t.Errorf("record count after round-trip: got %d want %d", got, want)
	}

	for xref := range combined.XRefMap {
		if roundtripped.GetRecord(xref) == nil {
			t.Errorf("xref %q present in combined but missing from round-tripped doc", xref)
		}
	}

	v := validator.New()
	if errs := v.Validate(roundtripped); len(errs) > 0 {
		t.Errorf("validation errors after round-trip: %v", errs)
	}
}

func mustRoundtripDecode(t *testing.T, path string) *gedcom.Document {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	doc, err := decoder.Decode(f)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return doc
}
