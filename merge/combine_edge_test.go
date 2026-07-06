package merge_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cacack/gedcom-go/v2/gedcom"
	"github.com/cacack/gedcom-go/v2/merge"
)

// indRec builds a minimal individual record at the given XRef. These
// edge-case tests exercise combine.go's internal branches directly
// through Combine, so the records only need an XRef and a type.
func indRec(xref string) *gedcom.Record {
	return &gedcom.Record{
		XRef:   xref,
		Type:   gedcom.RecordTypeIndividual,
		Entity: &gedcom.Individual{XRef: xref},
	}
}

// docWith builds a GEDCOM 7.0 document from the given records. XRefMap
// is intentionally left nil — Combine clones its inputs and rebuilds
// the map, so callers don't need to populate it.
func docWith(records ...*gedcom.Record) *gedcom.Document {
	return &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version70},
		Records: records,
	}
}

// TestCombineError_RenumberCollisionMessage covers the
// "renumber-collision" arm of CombineError.Error, which only fires on
// the (defensive) path where renumbering somehow produced a fresh
// collision.
func TestCombineError_RenumberCollisionMessage(t *testing.T) {
	err := &merge.CombineError{Kind: "renumber-collision", Colliding: []string{"@I1@", "@I2@"}}
	got := err.Error()
	if !strings.Contains(got, "renumber produced new collision") {
		t.Errorf("Error() = %q, want it to describe a renumber collision", got)
	}
	if !strings.Contains(got, "@I1@") {
		t.Errorf("Error() = %q, want it to list the colliding XRefs", got)
	}
}

// TestCombine_FindCollisions_SkipsNilAndEmptyXRef ensures nil and
// empty-XRef records in doc2 are ignored during collision detection.
func TestCombine_FindCollisions_SkipsNilAndEmptyXRef(t *testing.T) {
	doc1 := docWith(indRec("@I1@"))
	doc2 := docWith(nil, &gedcom.Record{XRef: ""}, indRec("@I2@"))

	out, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}
	if len(report.RemappedXRefs) != 0 {
		t.Errorf("expected no remaps (no real collision), got %v", report.RemappedXRefs)
	}
	if out.GetIndividual("@I2@") == nil {
		t.Error("doc2's @I2@ missing from combined document")
	}
}

// TestCombine_PrefixDoc2_SkipsNilAndEmptyInPostRemap runs the prefix
// strategy with nil and empty-XRef records present so the
// post-remap collision scan skips them.
func TestCombine_PrefixDoc2_SkipsNilAndEmptyInPostRemap(t *testing.T) {
	doc1 := docWith(indRec("@I1@"))
	doc2 := docWith(nil, &gedcom.Record{XRef: ""}, indRec("@I1@"))

	out, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{
		CollisionStrategy: merge.PrefixDoc2,
		Prefix:            "z_",
	})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}
	if out.GetIndividual("@z_I1@") == nil {
		t.Error("prefixed individual @z_I1@ missing from combined document")
	}
}

// TestCombine_PrefixDoc2_TransformProducesDuplicate covers the
// RemapXRefs error path inside the prefix strategy: doc2 already
// contains the prefixed name, so remapping the colliding XRef produces
// a duplicate within doc2.
func TestCombine_PrefixDoc2_TransformProducesDuplicate(t *testing.T) {
	// doc2's @I1@ collides with doc1 and gets "z_"; doc2 already has
	// @z_I1@ (non-colliding, unchanged) → two records map to @z_I1@.
	doc1 := docWith(indRec("@I1@"))
	doc2 := docWith(indRec("@I1@"), indRec("@z_I1@"))

	_, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{
		CollisionStrategy: merge.PrefixDoc2,
		Prefix:            "z_",
	})
	// The failure must come from RemapXRefs' duplicate-output check, not
	// the post-remap collision scan — assert the concrete error type and
	// reason so a future reroute through postRemapCollisions is caught.
	var remapErr *merge.RemapError
	if !errors.As(err, &remapErr) {
		t.Fatalf("expected a *merge.RemapError from the duplicate remap, got %v", err)
	}
	if !strings.Contains(remapErr.Reason, "collides") {
		t.Errorf("RemapError.Reason = %q, want it to describe a collision", remapErr.Reason)
	}
}

// TestCombine_RenumberDoc2_EdgeCases exercises renumberTransform's
// less-common branches in one pass:
//   - nil / empty-XRef records skipped while seeding counters and
//     building the type lookup,
//   - unknown record types falling back to the XRef's letter prefix
//     (parseable) or "X" (unparseable),
//   - non-colliding records passing through the transform unchanged,
//   - extractIDNumber rejecting too-short, non-numeric-suffix, and
//     overflowing XRefs during counter seeding.
func TestCombine_RenumberDoc2_EdgeCases(t *testing.T) {
	const unknownType = gedcom.RecordType("XYZ")

	// doc1 carries the colliding XRefs plus several malformed XRefs
	// that only exercise extractIDNumber during seeding. doc1 is never
	// remapped, so malformed pointers here are harmless.
	doc1 := docWith(
		indRec("@I1@"),  // collides
		indRec("@I7@"),  // collides (doc2 marks it an unknown type)
		indRec("@FOO@"), // collides (doc2 marks it an unknown type)
		&gedcom.Record{XRef: "@@", Type: gedcom.RecordTypeIndividual},                      // len < 3
		&gedcom.Record{XRef: "@I1A@", Type: gedcom.RecordTypeIndividual},                   // non-numeric suffix
		&gedcom.Record{XRef: "@I99999999999999999999@", Type: gedcom.RecordTypeIndividual}, // overflows Atoi
		nil, // skipped while seeding
	)

	doc2 := docWith(
		indRec("@I1@"), // collides → renumbered
		&gedcom.Record{XRef: "@I7@", Type: unknownType},  // unknown type, parseable XRef → prefix "I"
		&gedcom.Record{XRef: "@FOO@", Type: unknownType}, // unknown type, unparseable XRef → prefix "X"
		indRec("@I5@"), // non-colliding → passes through unchanged
		nil,            // skipped while seeding and typing
	)

	out, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{
		CollisionStrategy: merge.RenumberDoc2,
	})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}
	// The colliding XRefs must be renumbered to specific fresh values,
	// pinning down the prefix-selection logic this test exists to guard.
	// Counters seed above the highest numeric suffix seen across both
	// docs (I7 → next free is I8), so the two I-collisions become I8 and
	// I9 in collision order; @FOO@'s unknown type and unparseable XRef
	// fall back to the "X" prefix, which starts at 0.
	wantRemap := map[string]string{
		"@I1@":  "@I8@", // Individual type → "I" prefix
		"@I7@":  "@I9@", // unknown type, parseable XRef → "I" fallback
		"@FOO@": "@X0@", // unknown type, unparseable XRef → "X" fallback
	}
	for old, want := range wantRemap {
		if got := report.RemappedXRefs[old]; got != want {
			t.Errorf("RemappedXRefs[%s] = %q, want %q (full mapping %v)", old, got, want, report.RemappedXRefs)
		}
	}
	// The non-colliding @I5@ must survive unchanged.
	if out.GetIndividual("@I5@") == nil {
		t.Error("non-colliding @I5@ missing from combined document")
	}
}

// TestCombine_Doc1NilHeader covers mergeHeaders taking doc2's header
// wholesale when doc1 has none.
func TestCombine_Doc1NilHeader(t *testing.T) {
	doc1 := docWith(indRec("@I1@"))
	doc1.Header = nil
	doc2 := docWith(indRec("@I2@"))
	doc2.Header.SourceSystem = "AppB"

	out, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}
	if out.Header == nil {
		t.Fatal("combined header is nil; expected doc2's header")
	}
	if out.Header.SourceSystem != "AppB" {
		t.Errorf("SourceSystem = %q, want AppB (doc2's header adopted)", out.Header.SourceSystem)
	}
}

// TestCombine_AdoptsDoc2DateWhenDoc1HasNone covers the header Date
// promotion branch: doc1 has a zero Date, doc2 has one.
func TestCombine_AdoptsDoc2DateWhenDoc1HasNone(t *testing.T) {
	doc1 := docWith(indRec("@I1@"))
	doc2 := docWith(indRec("@I2@"))
	want := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	doc2.Header.Date = want

	out, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}
	if !out.Header.Date.Equal(want) {
		t.Errorf("Date = %v, want %v (adopted from doc2)", out.Header.Date, want)
	}
}

// TestCombine_PromotesDoc2HeaderFieldsWhenDoc1Empty covers the
// promote-h2 setters for every scalar header field.
func TestCombine_PromotesDoc2HeaderFieldsWhenDoc1Empty(t *testing.T) {
	doc1 := docWith(indRec("@I1@"))
	doc1.Header = &gedcom.Header{} // every scalar field empty
	doc2 := docWith(indRec("@I2@"))
	doc2.Header = &gedcom.Header{
		Version:        gedcom.Version70,
		Encoding:       gedcom.EncodingUTF8,
		SourceSystem:   "App",
		Language:       "English",
		Copyright:      "(c) X",
		AncestryTreeID: "tree",
	}

	out, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}
	h := out.Header
	if h.Version != gedcom.Version70 || h.Encoding != gedcom.EncodingUTF8 ||
		h.SourceSystem != "App" || h.Language != "English" ||
		h.Copyright != "(c) X" || h.AncestryTreeID != "tree" {
		t.Errorf("promoted header = %+v, want all doc2 values", h)
	}
	if len(report.HeaderConflicts) != 0 {
		t.Errorf("expected no conflicts when doc1 fields are empty, got %v", report.HeaderConflicts)
	}
}

// TestCombine_AdoptsDoc2SchemaWhenDoc1HasNone covers cloneSchema's
// deep-copy of TagMappings when doc1 has no schema.
func TestCombine_AdoptsDoc2SchemaWhenDoc1HasNone(t *testing.T) {
	doc1 := docWith(indRec("@I1@"))
	doc2 := docWith(indRec("@I2@"))
	doc2.Schema = &gedcom.SchemaDefinition{TagMappings: map[string]string{"_X": "http://example.test/x"}}

	out, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}
	if out.Schema == nil || out.Schema.TagMappings["_X"] != "http://example.test/x" {
		t.Errorf("expected doc2 schema adopted, got %+v", out.Schema)
	}
}

// TestCombine_SchemaConflictRecorded covers schemasEqual's
// length-mismatch branch and describeSchema's empty-schema result:
// doc1 has an empty schema, doc2 a populated one.
func TestCombine_SchemaConflictRecorded(t *testing.T) {
	doc1 := docWith(indRec("@I1@"))
	doc1.Schema = &gedcom.SchemaDefinition{} // non-nil but empty
	doc2 := docWith(indRec("@I2@"))
	doc2.Schema = &gedcom.SchemaDefinition{TagMappings: map[string]string{"_X": "u"}}

	_, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}
	var found bool
	for _, hc := range report.HeaderConflicts {
		if hc.Field != "Schema" {
			continue
		}
		found = true
		if hc.Doc1 != "" {
			t.Errorf("Doc1 schema description = %q, want empty", hc.Doc1)
		}
		if hc.Doc2 != "1 tag mapping(s)" {
			t.Errorf("Doc2 schema description = %q, want %q", hc.Doc2, "1 tag mapping(s)")
		}
	}
	if !found {
		t.Error("expected a Schema HeaderConflict to be recorded")
	}
}

// TestCombine_EqualSchemasNoConflict covers schemasEqual returning true
// for two structurally-equal schemas (no conflict recorded).
func TestCombine_EqualSchemasNoConflict(t *testing.T) {
	mk := func() *gedcom.SchemaDefinition {
		return &gedcom.SchemaDefinition{TagMappings: map[string]string{"_X": "u"}}
	}
	doc1 := docWith(indRec("@I1@"))
	doc1.Schema = mk()
	doc2 := docWith(indRec("@I2@"))
	doc2.Schema = mk()

	_, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}
	for _, hc := range report.HeaderConflicts {
		if hc.Field == "Schema" {
			t.Errorf("unexpected Schema conflict for equal schemas: %+v", hc)
		}
	}
}
