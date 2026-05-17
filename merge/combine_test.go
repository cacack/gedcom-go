package merge_test

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/decoder"
	"github.com/cacack/gedcom-go/v2/gedcom"
	"github.com/cacack/gedcom-go/v2/merge"
	"github.com/cacack/gedcom-go/v2/validator"
)

// buildDoc constructs a small document with the given prefix on its
// XRefs. Each builder returns a self-consistent document with one
// individual, one family pointing at the individual as husband, and
// one submitter referenced from the header.
func buildDoc(prefix string) *gedcom.Document {
	indXRef := "@" + prefix + "I1@"
	famXRef := "@" + prefix + "F1@"
	submXRef := "@" + prefix + "U1@"

	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version:   gedcom.Version70,
			Encoding:  gedcom.EncodingUTF8,
			Submitter: submXRef,
		},
		Trailer: &gedcom.Trailer{},
		XRefMap: make(map[string]*gedcom.Record),
	}

	add := func(xref string, typ gedcom.RecordType, entity interface{}) {
		rec := &gedcom.Record{
			XRef:   xref,
			Type:   typ,
			Entity: entity,
			Tags:   []*gedcom.Tag{{Level: 0, Tag: string(typ), XRef: xref}},
		}
		doc.Records = append(doc.Records, rec)
		doc.XRefMap[xref] = rec
	}

	add(indXRef, gedcom.RecordTypeIndividual, &gedcom.Individual{
		XRef:             indXRef,
		Names:            []*gedcom.PersonalName{{Full: "Test /" + prefix + "Person/"}},
		SpouseInFamilies: []string{famXRef},
	})
	// Append NAME and FAMS tags so the document passes basic
	// validation (Individual requires NAME, Family requires HUSB/WIFE/CHIL).
	doc.Records[len(doc.Records)-1].Tags = append(doc.Records[len(doc.Records)-1].Tags,
		&gedcom.Tag{Level: 1, Tag: "NAME", Value: "Test /" + prefix + "Person/"},
		&gedcom.Tag{Level: 1, Tag: "FAMS", Value: famXRef},
	)

	add(famXRef, gedcom.RecordTypeFamily, &gedcom.Family{
		XRef:    famXRef,
		Husband: indXRef,
	})
	doc.Records[len(doc.Records)-1].Tags = append(doc.Records[len(doc.Records)-1].Tags,
		&gedcom.Tag{Level: 1, Tag: "HUSB", Value: indXRef},
	)

	add(submXRef, gedcom.RecordTypeSubmitter, &gedcom.Submitter{
		XRef: submXRef,
		Name: "Test Submitter " + prefix,
	})
	doc.Records[len(doc.Records)-1].Tags = append(doc.Records[len(doc.Records)-1].Tags,
		&gedcom.Tag{Level: 1, Tag: "NAME", Value: "Test Submitter " + prefix},
	)

	return doc
}

// buildCollidingDoc returns a document whose XRefs are unprefixed
// ("@I1@", "@F1@", "@U1@") so it collides with another buildDoc
// caller that uses the same convention.
func buildCollidingDoc() *gedcom.Document {
	return buildDoc("")
}

func TestCombine_NilDocs(t *testing.T) {
	d := buildDoc("a_")
	if _, _, err := merge.Combine(nil, d, merge.CombineOptions{}); err == nil {
		t.Error("expected error for nil doc1")
	}
	if _, _, err := merge.Combine(d, nil, merge.CombineOptions{}); err == nil {
		t.Error("expected error for nil doc2")
	}
}

func TestCombine_NoCollisions_NoRemapNeeded(t *testing.T) {
	doc1 := buildDoc("a_")
	doc2 := buildDoc("b_")

	snap1 := doc1.Clone()
	snap2 := doc2.Clone()

	out, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}

	if len(report.RemappedXRefs) != 0 {
		t.Errorf("RemappedXRefs should be empty when no collisions; got %v", report.RemappedXRefs)
	}

	// Inputs unmodified.
	if !reflect.DeepEqual(doc1, snap1) {
		t.Error("doc1 was mutated")
	}
	if !reflect.DeepEqual(doc2, snap2) {
		t.Error("doc2 was mutated")
	}

	// Combined output has all six records.
	if got, want := len(out.Records), len(doc1.Records)+len(doc2.Records); got != want {
		t.Errorf("combined record count = %d, want %d", got, want)
	}

	// Both XRefs lookable.
	if out.GetIndividual("@a_I1@") == nil {
		t.Error("missing individual @a_I1@ from combined doc")
	}
	if out.GetIndividual("@b_I1@") == nil {
		t.Error("missing individual @b_I1@ from combined doc")
	}
}

func TestCombine_ErrorOnCollision(t *testing.T) {
	doc1 := buildCollidingDoc()
	doc2 := buildCollidingDoc()

	out, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{
		CollisionStrategy: merge.ErrorOnCollision,
	})
	if err == nil {
		t.Fatal("expected collision error, got nil")
	}
	if out != nil || report != nil {
		t.Error("error path should return nil doc and nil report")
	}
	if !errors.Is(err, merge.ErrXRefCollision) {
		t.Errorf("error should wrap ErrXRefCollision, got %v", err)
	}
	var cerr *merge.CombineError
	if !errors.As(err, &cerr) {
		t.Fatalf("expected *CombineError, got %T", err)
	}
	if cerr.Kind != "collision" {
		t.Errorf("Kind = %q, want collision", cerr.Kind)
	}
	if len(cerr.Colliding) == 0 {
		t.Error("Colliding should list the offending XRefs")
	}
}

func TestCombine_ErrorOnCollision_TruncatesLongList(t *testing.T) {
	doc1 := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version70},
		XRefMap: make(map[string]*gedcom.Record),
	}
	doc2 := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version70},
		XRefMap: make(map[string]*gedcom.Record),
	}
	for i := 0; i < 15; i++ {
		xref := "@I" + itoa(i) + "@"
		rec1 := &gedcom.Record{XRef: xref, Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: xref}}
		rec2 := &gedcom.Record{XRef: xref, Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: xref}}
		doc1.Records = append(doc1.Records, rec1)
		doc1.XRefMap[xref] = rec1
		doc2.Records = append(doc2.Records, rec2)
		doc2.XRefMap[xref] = rec2
	}

	_, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "more") {
		t.Errorf("error should mention truncation: %v", err)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	out := ""
	for i > 0 {
		out = string(rune('0'+i%10)) + out
		i /= 10
	}
	return out
}

func TestCombine_PrefixDoc2(t *testing.T) {
	doc1 := buildCollidingDoc()
	doc2 := buildCollidingDoc()

	out, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{
		CollisionStrategy: merge.PrefixDoc2,
		Prefix:            "b_",
	})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}

	// Every doc2 XRef collided, so every doc2 XRef should be in the report.
	wantRemaps := map[string]string{
		"@I1@": "@b_I1@",
		"@F1@": "@b_F1@",
		"@U1@": "@b_U1@",
	}
	for old, want := range wantRemaps {
		got, ok := report.RemappedXRefs[old]
		if !ok {
			t.Errorf("RemappedXRefs missing %q", old)
			continue
		}
		if got != want {
			t.Errorf("RemappedXRefs[%q] = %q, want %q", old, got, want)
		}
	}
	if len(report.RemappedXRefs) != len(wantRemaps) {
		t.Errorf("RemappedXRefs size = %d, want %d (no-op entries should be filtered)", len(report.RemappedXRefs), len(wantRemaps))
	}

	// Original doc1 XRefs preserved, doc2 XRefs prefixed.
	if out.GetIndividual("@I1@") == nil {
		t.Error("doc1 individual @I1@ missing")
	}
	if out.GetIndividual("@b_I1@") == nil {
		t.Error("doc2 individual @b_I1@ missing (should have been prefixed)")
	}

	// Family in doc2 must point at the prefixed husband.
	f2 := out.GetFamily("@b_F1@")
	if f2 == nil {
		t.Fatal("doc2 family @b_F1@ missing")
	}
	if f2.Husband != "@b_I1@" {
		t.Errorf("@b_F1@.Husband = %q, want @b_I1@", f2.Husband)
	}
}

func TestCombine_PrefixDoc2_PartialCollisions(t *testing.T) {
	// Set up doc2 to have ONE colliding XRef and one non-colliding.
	doc1 := buildDoc("") // @I1@, @F1@, @U1@
	doc2 := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version70},
		XRefMap: make(map[string]*gedcom.Record),
	}
	// Collides with doc1.
	rec1 := &gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: "@I1@", Names: []*gedcom.PersonalName{{Full: "X"}}}, Tags: []*gedcom.Tag{{Level: 1, Tag: "NAME", Value: "X"}}}
	doc2.Records = append(doc2.Records, rec1)
	doc2.XRefMap["@I1@"] = rec1
	// Unique.
	rec2 := &gedcom.Record{XRef: "@I99@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: "@I99@", Names: []*gedcom.PersonalName{{Full: "Y"}}}, Tags: []*gedcom.Tag{{Level: 1, Tag: "NAME", Value: "Y"}}}
	doc2.Records = append(doc2.Records, rec2)
	doc2.XRefMap["@I99@"] = rec2

	out, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{
		CollisionStrategy: merge.PrefixDoc2,
		Prefix:            "z_",
	})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}

	// Only the colliding XRef should be in the report.
	if len(report.RemappedXRefs) != 1 {
		t.Errorf("RemappedXRefs size = %d, want 1", len(report.RemappedXRefs))
	}
	if got := report.RemappedXRefs["@I1@"]; got != "@z_I1@" {
		t.Errorf("RemappedXRefs[@I1@] = %q, want @z_I1@", got)
	}
	if _, ok := report.RemappedXRefs["@I99@"]; ok {
		t.Error("non-colliding @I99@ should not appear in RemappedXRefs")
	}

	// Non-colliding XRef preserved as-is in output.
	if out.GetIndividual("@I99@") == nil {
		t.Error("non-colliding @I99@ should be preserved without remap")
	}
}

func TestCombine_PrefixDoc2_RequiresPrefix(t *testing.T) {
	doc1 := buildCollidingDoc()
	doc2 := buildCollidingDoc()

	_, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{
		CollisionStrategy: merge.PrefixDoc2,
		// Prefix intentionally empty.
	})
	if err == nil {
		t.Fatal("expected error for missing Prefix")
	}
	if !strings.Contains(err.Error(), "Prefix") {
		t.Errorf("error should mention Prefix; got %v", err)
	}
}

func TestCombine_PrefixDoc2_PrefixProducesSecondaryCollision(t *testing.T) {
	// doc1 has @I1@ and @z_I1@ already. If doc2 has @I1@ and we
	// prefix with "z_", the new XRef @z_I1@ collides with doc1.
	doc1 := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version70},
		XRefMap: make(map[string]*gedcom.Record),
	}
	add := func(d *gedcom.Document, xref string) {
		rec := &gedcom.Record{
			XRef: xref, Type: gedcom.RecordTypeIndividual,
			Entity: &gedcom.Individual{XRef: xref, Names: []*gedcom.PersonalName{{Full: "X"}}},
			Tags:   []*gedcom.Tag{{Level: 1, Tag: "NAME", Value: "X"}},
		}
		d.Records = append(d.Records, rec)
		d.XRefMap[xref] = rec
	}
	add(doc1, "@I1@")
	add(doc1, "@z_I1@")

	doc2 := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version70},
		XRefMap: make(map[string]*gedcom.Record),
	}
	add(doc2, "@I1@")

	_, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{
		CollisionStrategy: merge.PrefixDoc2,
		Prefix:            "z_",
	})
	if err == nil {
		t.Fatal("expected secondary-collision error")
	}
	if !errors.Is(err, merge.ErrXRefCollision) {
		t.Errorf("error should wrap ErrXRefCollision, got %v", err)
	}
	var cerr *merge.CombineError
	if !errors.As(err, &cerr) {
		t.Fatalf("expected *CombineError, got %T", err)
	}
	if cerr.Kind != "prefix-collision" {
		t.Errorf("Kind = %q, want prefix-collision", cerr.Kind)
	}
}

func TestCombine_RenumberDoc2(t *testing.T) {
	// doc1 has @I1@, @I2@; doc2 has @I1@, @I2@ (all colliding).
	doc1 := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version70},
		XRefMap: make(map[string]*gedcom.Record),
	}
	add := func(d *gedcom.Document, xref string, typ gedcom.RecordType) {
		var entity interface{}
		switch typ {
		case gedcom.RecordTypeIndividual:
			entity = &gedcom.Individual{XRef: xref, Names: []*gedcom.PersonalName{{Full: "X"}}}
		case gedcom.RecordTypeFamily:
			entity = &gedcom.Family{XRef: xref, Husband: "@I1@"}
		}
		rec := &gedcom.Record{XRef: xref, Type: typ, Entity: entity}
		d.Records = append(d.Records, rec)
		d.XRefMap[xref] = rec
	}
	add(doc1, "@I1@", gedcom.RecordTypeIndividual)
	add(doc1, "@I2@", gedcom.RecordTypeIndividual)

	doc2 := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version70},
		XRefMap: make(map[string]*gedcom.Record),
	}
	add(doc2, "@I1@", gedcom.RecordTypeIndividual)
	add(doc2, "@I2@", gedcom.RecordTypeIndividual)

	out, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{
		CollisionStrategy: merge.RenumberDoc2,
	})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}

	// doc2's @I1@ and @I2@ should each be renumbered. The next free
	// IDs after both docs' max(2) are 3 and 4.
	got1, ok1 := report.RemappedXRefs["@I1@"]
	if !ok1 {
		t.Fatal("@I1@ should be in RemappedXRefs")
	}
	got2, ok2 := report.RemappedXRefs["@I2@"]
	if !ok2 {
		t.Fatal("@I2@ should be in RemappedXRefs")
	}
	if got1 != "@I3@" {
		t.Errorf("RemappedXRefs[@I1@] = %q, want @I3@", got1)
	}
	if got2 != "@I4@" {
		t.Errorf("RemappedXRefs[@I2@] = %q, want @I4@", got2)
	}

	// Output should have all four individuals.
	if got, want := len(out.Records), 4; got != want {
		t.Errorf("combined record count = %d, want %d", got, want)
	}
	for _, x := range []string{"@I1@", "@I2@", "@I3@", "@I4@"} {
		if out.GetIndividual(x) == nil {
			t.Errorf("missing individual %s in combined doc", x)
		}
	}
}

func TestCombine_RenumberDoc2_PerTypeSequential(t *testing.T) {
	// Two record types, each maxing out at different numbers in doc1.
	// doc1: @I5@, @F2@. doc2: @I5@, @F2@ (both colliding).
	// Expect doc2 to be renumbered to @I6@ and @F3@ respectively.
	build := func(individuals, families []string) *gedcom.Document {
		d := &gedcom.Document{
			Header:  &gedcom.Header{Version: gedcom.Version70},
			XRefMap: make(map[string]*gedcom.Record),
		}
		for _, x := range individuals {
			rec := &gedcom.Record{XRef: x, Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: x}}
			d.Records = append(d.Records, rec)
			d.XRefMap[x] = rec
		}
		for _, x := range families {
			rec := &gedcom.Record{XRef: x, Type: gedcom.RecordTypeFamily, Entity: &gedcom.Family{XRef: x}}
			d.Records = append(d.Records, rec)
			d.XRefMap[x] = rec
		}
		return d
	}
	doc1 := build([]string{"@I5@"}, []string{"@F2@"})
	doc2 := build([]string{"@I5@"}, []string{"@F2@"})

	_, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{
		CollisionStrategy: merge.RenumberDoc2,
	})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}

	if got := report.RemappedXRefs["@I5@"]; got != "@I6@" {
		t.Errorf("RemappedXRefs[@I5@] = %q, want @I6@", got)
	}
	if got := report.RemappedXRefs["@F2@"]; got != "@F3@" {
		t.Errorf("RemappedXRefs[@F2@] = %q, want @F3@", got)
	}
}

func TestCombine_VersionMismatch(t *testing.T) {
	doc1 := buildDoc("a_")
	doc2 := buildDoc("b_")
	doc1.Header.Version = gedcom.Version551
	doc2.Header.Version = gedcom.Version70

	_, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err == nil {
		t.Fatal("expected version mismatch error")
	}
	if !errors.Is(err, merge.ErrIncompatibleHeader) {
		t.Errorf("error should wrap ErrIncompatibleHeader, got %v", err)
	}
	var cerr *merge.CombineError
	if !errors.As(err, &cerr) {
		t.Fatalf("expected *CombineError, got %T", err)
	}
	if cerr.Kind != "version" {
		t.Errorf("Kind = %q, want version", cerr.Kind)
	}
}

func TestCombine_VersionOneEmptyIsOK(t *testing.T) {
	doc1 := buildDoc("a_")
	doc2 := buildDoc("b_")
	doc1.Header.Version = gedcom.Version70
	doc2.Header.Version = "" // unspecified, no conflict

	_, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine with one empty version should succeed, got %v", err)
	}
}

func TestCombine_EncodingMismatch(t *testing.T) {
	doc1 := buildDoc("a_")
	doc2 := buildDoc("b_")
	doc1.Header.Encoding = gedcom.EncodingUTF8
	doc2.Header.Encoding = gedcom.EncodingANSEL

	_, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err == nil {
		t.Fatal("expected encoding mismatch error")
	}
	if !errors.Is(err, merge.ErrIncompatibleHeader) {
		t.Errorf("error should wrap ErrIncompatibleHeader, got %v", err)
	}
	var cerr *merge.CombineError
	if !errors.As(err, &cerr) {
		t.Fatalf("expected *CombineError, got %T", err)
	}
	if cerr.Kind != "encoding" {
		t.Errorf("Kind = %q, want encoding", cerr.Kind)
	}
}

func TestCombine_HeaderFieldConflictsRecorded(t *testing.T) {
	doc1 := buildDoc("a_")
	doc2 := buildDoc("b_")

	doc1.Header.SourceSystem = "AppA"
	doc2.Header.SourceSystem = "AppB"
	doc1.Header.Language = "English"
	doc2.Header.Language = "French"
	doc1.Header.Copyright = "(c) Alice"
	doc2.Header.Copyright = "(c) Bob"
	doc1.Header.AncestryTreeID = "tree-a"
	doc2.Header.AncestryTreeID = "tree-b"

	out, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}

	want := map[string][2]string{
		"SourceSystem":   {"AppA", "AppB"},
		"Language":       {"English", "French"},
		"Copyright":      {"(c) Alice", "(c) Bob"},
		"AncestryTreeID": {"tree-a", "tree-b"},
	}
	gotByField := make(map[string][2]string, len(report.HeaderConflicts))
	for _, hc := range report.HeaderConflicts {
		gotByField[hc.Field] = [2]string{hc.Doc1, hc.Doc2}
	}
	for field, vals := range want {
		got, ok := gotByField[field]
		if !ok {
			t.Errorf("HeaderConflicts missing field %s", field)
			continue
		}
		if got != vals {
			t.Errorf("HeaderConflict[%s] = %v, want %v", field, got, vals)
		}
	}

	// doc1's values must win in the output header.
	if out.Header.SourceSystem != "AppA" {
		t.Errorf("SourceSystem = %q, want AppA", out.Header.SourceSystem)
	}
	if out.Header.Language != "English" {
		t.Errorf("Language = %q, want English", out.Header.Language)
	}
}

func TestCombine_Doc1EmptyHeaderFieldPromotesDoc2(t *testing.T) {
	doc1 := buildDoc("a_")
	doc2 := buildDoc("b_")
	doc1.Header.SourceSystem = ""
	doc2.Header.SourceSystem = "AppB"

	out, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}
	if out.Header.SourceSystem != "AppB" {
		t.Errorf("SourceSystem = %q, want AppB", out.Header.SourceSystem)
	}
	for _, hc := range report.HeaderConflicts {
		if hc.Field == "SourceSystem" {
			t.Errorf("SourceSystem conflict should not be recorded when doc1 is empty; got %+v", hc)
		}
	}
}

func TestCombine_SubmitterDoc1Wins(t *testing.T) {
	doc1 := buildDoc("a_") // has @a_U1@
	doc2 := buildDoc("b_") // has @b_U1@
	out, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}
	if out.Header.Submitter != "@a_U1@" {
		t.Errorf("Submitter = %q, want @a_U1@ (doc1 wins)", out.Header.Submitter)
	}
}

func TestCombine_SubmitterFromDoc2WhenDoc1HasNone(t *testing.T) {
	doc1 := buildDoc("a_")
	doc2 := buildDoc("b_")
	doc1.Header.Submitter = "" // doc1 has no submitter

	out, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}
	// doc2's submitter is @b_U1@ (no collision), so it should be
	// used as-is.
	if out.Header.Submitter != "@b_U1@" {
		t.Errorf("Submitter = %q, want @b_U1@", out.Header.Submitter)
	}
}

func TestCombine_SubmitterFromDoc2WhenRemapped(t *testing.T) {
	// Both docs use @U1@; doc1 has no Submitter header field, so
	// doc2's submitter is adopted. Because we prefix doc2, the new
	// XRef is @z_U1@ and the header must point at the new name.
	doc1 := buildCollidingDoc()
	doc1.Header.Submitter = "" // doc1 has no submitter; doc2's wins

	doc2 := buildCollidingDoc() // @U1@ submitter

	out, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{
		CollisionStrategy: merge.PrefixDoc2,
		Prefix:            "z_",
	})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}
	if out.Header.Submitter != "@z_U1@" {
		t.Errorf("Submitter = %q, want @z_U1@ (doc2's submitter, remapped)", out.Header.Submitter)
	}
	// Verify the submitter record actually exists at that XRef.
	if out.GetSubmitter("@z_U1@") == nil {
		t.Error("submitter record @z_U1@ missing from combined doc")
	}
}

func TestCombine_InputsUnmodified(t *testing.T) {
	doc1 := buildCollidingDoc()
	doc2 := buildCollidingDoc()
	snap1 := doc1.Clone()
	snap2 := doc2.Clone()

	_, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{
		CollisionStrategy: merge.PrefixDoc2,
		Prefix:            "x_",
	})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}
	if !reflect.DeepEqual(doc1, snap1) {
		t.Error("doc1 was mutated by Combine")
	}
	if !reflect.DeepEqual(doc2, snap2) {
		t.Error("doc2 was mutated by Combine")
	}
}

func TestCombine_OutputValidatesCleanly(t *testing.T) {
	doc1 := buildDoc("a_")
	doc2 := buildDoc("b_")

	out, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}

	v := validator.New()
	if errs := v.Validate(out); len(errs) != 0 {
		t.Errorf("combined doc reported %d validation errors:", len(errs))
		for _, e := range errs {
			t.Logf("  %v", e)
		}
	}
}

func TestCombine_OutputValidatesCleanlyAfterPrefixRemap(t *testing.T) {
	doc1 := buildCollidingDoc()
	doc2 := buildCollidingDoc()

	out, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{
		CollisionStrategy: merge.PrefixDoc2,
		Prefix:            "z_",
	})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}

	v := validator.New()
	if errs := v.Validate(out); len(errs) != 0 {
		t.Errorf("combined doc (prefix remap) reported %d validation errors:", len(errs))
		for _, e := range errs {
			t.Logf("  %v", e)
		}
	}
}

func TestCombine_NilHeaders(t *testing.T) {
	doc1 := &gedcom.Document{XRefMap: make(map[string]*gedcom.Record)}
	doc2 := &gedcom.Document{XRefMap: make(map[string]*gedcom.Record)}

	_, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine of two header-less docs should succeed, got %v", err)
	}
}

func TestCombine_OneHeaderNil(t *testing.T) {
	doc1 := buildDoc("a_")
	doc2 := buildDoc("b_")
	doc2.Header = nil

	out, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}
	if out.Header == nil {
		t.Fatal("output header should not be nil when doc1 has one")
	}
	if out.Header.Version != gedcom.Version70 {
		t.Errorf("Version = %q, want 7.0", out.Header.Version)
	}
}

func TestCombine_VendorAndSchemaConflicts(t *testing.T) {
	doc1 := buildDoc("a_")
	doc2 := buildDoc("b_")
	doc1.Vendor = "VendorA"
	doc2.Vendor = "VendorB"
	doc1.Schema = &gedcom.SchemaDefinition{TagMappings: map[string]string{"_X": "http://a"}}
	doc2.Schema = &gedcom.SchemaDefinition{TagMappings: map[string]string{"_Y": "http://b"}}

	out, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}

	if out.Vendor != "VendorA" {
		t.Errorf("Vendor = %q, want VendorA", out.Vendor)
	}
	if out.Schema == nil || out.Schema.TagMappings["_X"] != "http://a" {
		t.Error("Schema should preserve doc1's mappings")
	}

	gotFields := make(map[string]bool)
	for _, hc := range report.HeaderConflicts {
		gotFields[hc.Field] = true
	}
	if !gotFields["Vendor"] {
		t.Error("HeaderConflicts should include Vendor")
	}
	if !gotFields["Schema"] {
		t.Error("HeaderConflicts should include Schema")
	}
}

func TestCombine_VendorAndSchemaAdoptedWhenDoc1Empty(t *testing.T) {
	doc1 := buildDoc("a_")
	doc2 := buildDoc("b_")
	doc1.Vendor = ""
	doc2.Vendor = "VendorB"
	doc1.Schema = nil
	doc2.Schema = &gedcom.SchemaDefinition{TagMappings: map[string]string{"_Y": "http://b"}}

	out, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}
	if out.Vendor != "VendorB" {
		t.Errorf("Vendor = %q, want VendorB", out.Vendor)
	}
	if out.Schema == nil || out.Schema.TagMappings["_Y"] != "http://b" {
		t.Error("Schema should be adopted from doc2 when doc1 has none")
	}
	for _, hc := range report.HeaderConflicts {
		if hc.Field == "Vendor" || hc.Field == "Schema" {
			t.Errorf("no conflict expected when doc1 is empty; got %+v", hc)
		}
	}
}

func TestCombine_TrailerFallsBackToDoc2(t *testing.T) {
	doc1 := buildDoc("a_")
	doc2 := buildDoc("b_")
	doc1.Trailer = nil
	doc2.Trailer = &gedcom.Trailer{LineNumber: 42}

	out, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}
	if out.Trailer == nil {
		t.Fatal("Trailer should fall back to doc2's when doc1 has none")
	}
	if out.Trailer.LineNumber != 42 {
		t.Errorf("Trailer.LineNumber = %d, want 42", out.Trailer.LineNumber)
	}
}

func TestCombine_HeaderTagsConcatenated(t *testing.T) {
	doc1 := buildDoc("a_")
	doc2 := buildDoc("b_")
	doc1.Header.Tags = []*gedcom.Tag{{Level: 1, Tag: "_CUSTOM1", Value: "from-doc1"}}
	doc2.Header.Tags = []*gedcom.Tag{{Level: 1, Tag: "_CUSTOM2", Value: "from-doc2"}}

	out, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}
	if len(out.Header.Tags) != 2 {
		t.Fatalf("expected 2 header tags, got %d", len(out.Header.Tags))
	}
	if out.Header.Tags[0].Tag != "_CUSTOM1" || out.Header.Tags[1].Tag != "_CUSTOM2" {
		t.Errorf("tag order = [%s, %s], want [_CUSTOM1, _CUSTOM2]",
			out.Header.Tags[0].Tag, out.Header.Tags[1].Tag)
	}
}

func TestCombine_UnknownStrategy(t *testing.T) {
	doc1 := buildDoc("a_")
	doc2 := buildDoc("b_")
	_, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{
		CollisionStrategy: 99,
	})
	if err == nil {
		t.Fatal("expected error for unknown CollisionStrategy")
	}
}

func TestCombineError_ErrorMessages(t *testing.T) {
	cases := []struct {
		name    string
		err     *merge.CombineError
		want    string
		wantIs  []error
		wantNot []error
	}{
		{
			name:   "version",
			err:    &merge.CombineError{Kind: "version", Doc1: "5.5.1", Doc2: "7.0"},
			want:   "merge: incompatible versions: 5.5.1 vs 7.0",
			wantIs: []error{merge.ErrIncompatibleHeader},
		},
		{
			name:   "encoding",
			err:    &merge.CombineError{Kind: "encoding", Doc1: "UTF-8", Doc2: "ANSEL"},
			want:   "merge: incompatible encodings: UTF-8 vs ANSEL",
			wantIs: []error{merge.ErrIncompatibleHeader},
		},
		{
			name:   "collision",
			err:    &merge.CombineError{Kind: "collision", Colliding: []string{"@I1@", "@I2@"}},
			want:   "merge: xref collision: @I1@, @I2@",
			wantIs: []error{merge.ErrXRefCollision},
		},
		{
			name:   "prefix-collision",
			err:    &merge.CombineError{Kind: "prefix-collision", Colliding: []string{"@a_I1@"}},
			want:   "merge: prefix produced new collision: @a_I1@",
			wantIs: []error{merge.ErrXRefCollision},
		},
		{
			name: "missing-prefix",
			err:  &merge.CombineError{Kind: "missing-prefix"},
			want: "merge: PrefixDoc2 strategy requires non-empty Prefix",
		},
		{
			name: "unknown",
			err:  &merge.CombineError{Kind: "weird-kind"},
			want: "merge: weird-kind",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.err.Error(); got != tc.want {
				t.Errorf("Error() = %q, want %q", got, tc.want)
			}
			for _, target := range tc.wantIs {
				if !errors.Is(tc.err, target) {
					t.Errorf("expected errors.Is(%v) to be true", target)
				}
			}
			// Also verify CombineError does NOT satisfy unrelated sentinels.
			if errors.Is(tc.err, errors.New("unrelated")) {
				t.Error("CombineError should not satisfy unrelated sentinel via Is()")
			}
		})
	}
}

func TestRemapError_StillReadable(t *testing.T) {
	// Smoke test that RemapError.Error() formats as expected. This
	// keeps coverage on errors.go alive since RemapXRefs callers see
	// the formatted message.
	re := &merge.RemapError{Old: "@I1@", New: "broken", Reason: "test reason"}
	got := re.Error()
	if !strings.Contains(got, "@I1@") || !strings.Contains(got, "broken") || !strings.Contains(got, "test reason") {
		t.Errorf("RemapError.Error() = %q, missing expected substrings", got)
	}
}

func TestCombine_RealFixturesValidateCleanly(t *testing.T) {
	// Combine two GEDCOM 7.0 fixtures that share XRefs (@I1@, @F1@).
	// Both come from the project's own testdata, so this exercises
	// the full decode → combine → validate pipeline.
	loadDoc := func(path string) *gedcom.Document {
		t.Helper()
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("open %s: %v", path, err)
		}
		defer func() { _ = f.Close() }()
		doc, err := decoder.Decode(f)
		if err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		return doc
	}

	doc1 := loadDoc("../testdata/gedcom-7.0/remarriage1.ged")
	doc2 := loadDoc("../testdata/gedcom-7.0/remarriage1.ged")

	out, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{
		CollisionStrategy: merge.PrefixDoc2,
		Prefix:            "b_",
	})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}

	// Every doc2 XRef should have been remapped (full overlap).
	if len(report.RemappedXRefs) == 0 {
		t.Error("expected remappings for fully-overlapping fixtures")
	}

	// Combined doc validates cleanly.
	v := validator.New()
	if errs := v.Validate(out); len(errs) != 0 {
		t.Errorf("combined real-fixture doc reported %d validation errors:", len(errs))
		for _, e := range errs {
			t.Logf("  %v", e)
		}
	}

	// Record count should be the sum of both inputs (no records dropped).
	if got, want := len(out.Records), len(doc1.Records)+len(doc2.Records); got != want {
		t.Errorf("combined record count = %d, want %d", got, want)
	}
}

func TestExtractIDNumber_ViaRenumber(t *testing.T) {
	// Indirectly exercise extractIDNumber via Renumber: a doc2 record
	// with a non-numeric XRef ("@MYID@") that collides should still
	// be renamed using its record-type letter prefix.
	doc1 := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version70},
		XRefMap: make(map[string]*gedcom.Record),
	}
	rec1 := &gedcom.Record{XRef: "@MYID@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: "@MYID@"}}
	doc1.Records = append(doc1.Records, rec1)
	doc1.XRefMap["@MYID@"] = rec1

	doc2 := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version70},
		XRefMap: make(map[string]*gedcom.Record),
	}
	rec2 := &gedcom.Record{XRef: "@MYID@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: "@MYID@"}}
	doc2.Records = append(doc2.Records, rec2)
	doc2.XRefMap["@MYID@"] = rec2

	_, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{
		CollisionStrategy: merge.RenumberDoc2,
	})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}
	got, ok := report.RemappedXRefs["@MYID@"]
	if !ok {
		t.Fatal("@MYID@ should be remapped")
	}
	// Should start with @I (Individual prefix) and be a valid pointer shape.
	if !gedcom.IsPointerXRef(got) {
		t.Errorf("remapped XRef %q is not a valid pointer", got)
	}
	if !strings.HasPrefix(got, "@I") {
		t.Errorf("remapped XRef %q should start with @I (Individual)", got)
	}
}
