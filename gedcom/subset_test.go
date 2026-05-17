package gedcom

import (
	"errors"
	"reflect"
	"sort"
	"testing"
)

// buildRichFixture creates a small document with cross-references covering
// every reference class the closure walker needs to follow: family links,
// source citations, repository ownership, notes, media, and a submitter.
func buildRichFixture() *Document {
	doc := &Document{
		Header: &Header{
			Version:   Version70,
			Encoding:  EncodingUTF8,
			Submitter: "@SUBM1@",
		},
		Trailer: &Trailer{LineNumber: 100},
		Vendor:  "TestVendor",
		Schema: &SchemaDefinition{
			TagMappings: map[string]string{"_TEST": "http://example.com/test"},
		},
		XRefMap: make(map[string]*Record),
	}

	add := func(xref string, typ RecordType, entity interface{}) {
		rec := &Record{XRef: xref, Type: typ, Entity: entity}
		doc.Records = append(doc.Records, rec)
		doc.XRefMap[xref] = rec
	}

	add("@I1@", RecordTypeIndividual, &Individual{
		XRef:             "@I1@",
		SpouseInFamilies: []string{"@F1@"},
		Notes:            []string{"@N1@"},
		SourceCitations:  []*SourceCitation{{SourceXRef: "@S1@"}},
		Media:            []*MediaLink{{MediaXRef: "@M1@"}},
		Events: []*Event{
			{Type: "BIRT", SourceCitations: []*SourceCitation{{SourceXRef: "@S2@"}}},
		},
	})
	add("@I2@", RecordTypeIndividual, &Individual{
		XRef:             "@I2@",
		SpouseInFamilies: []string{"@F1@"},
	})
	add("@I3@", RecordTypeIndividual, &Individual{
		XRef:            "@I3@",
		ChildInFamilies: []FamilyLink{{FamilyXRef: "@F1@"}},
	})
	add("@F1@", RecordTypeFamily, &Family{
		XRef:     "@F1@",
		Husband:  "@I1@",
		Wife:     "@I2@",
		Children: []string{"@I3@"},
		Notes:    []string{"@N2@"},
	})

	add("@N1@", RecordTypeNote, &Note{XRef: "@N1@", Text: "Note 1"})
	add("@N2@", RecordTypeNote, &Note{XRef: "@N2@", Text: "Note 2"})

	add("@S1@", RecordTypeSource, &Source{XRef: "@S1@", Title: "Source 1", RepositoryRef: "@R1@"})
	add("@S2@", RecordTypeSource, &Source{XRef: "@S2@", Title: "Source 2"})

	add("@R1@", RecordTypeRepository, &Repository{XRef: "@R1@", Name: "Repo 1"})

	add("@M1@", RecordTypeMedia, &MediaObject{
		XRef:            "@M1@",
		Files:           []*MediaFile{{FileRef: "/m1.jpg"}},
		SourceCitations: []*SourceCitation{{SourceXRef: "@S2@"}},
	})

	// Unreferenced submitter — should not be pulled in by ref-closure.
	add("@SUBM1@", RecordTypeSubmitter, &Submitter{XRef: "@SUBM1@", Name: "Submitter"})

	// Unreferenced extra individual — should not be in the closure.
	add("@I99@", RecordTypeIndividual, &Individual{XRef: "@I99@"})

	return doc
}

func TestSubset_NilDocument(t *testing.T) {
	var doc *Document
	if _, err := doc.Subset([]string{"@I1@"}); err == nil {
		t.Error("Subset on nil doc should error")
	}
}

func TestSubset_EmptySeeds(t *testing.T) {
	doc := buildRichFixture()
	sub, err := doc.Subset(nil)
	if err != nil {
		t.Fatalf("Subset(nil seeds) errored: %v", err)
	}
	if len(sub.Records) != 0 {
		t.Errorf("Subset(empty) records = %d, want 0", len(sub.Records))
	}
	if sub.Header == nil {
		t.Error("Subset(empty) header should be carried over, got nil")
	}
}

func TestSubset_MalformedSeedReturnsError(t *testing.T) {
	doc := buildRichFixture()
	cases := []struct{ name, seed string }{
		{"empty string", ""},
		{"no @ delimiters", "garbage"},
		{"missing closing @", "@unterminated"},
		{"VOID sentinel as seed", "@VOID@"},
		{"contains whitespace", "@bad seed@"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := doc.Subset([]string{tc.seed})
			if err == nil {
				t.Fatalf("expected error for malformed seed %q, got nil", tc.seed)
			}
			if !errors.Is(err, ErrUnknownXRef) {
				t.Errorf("error should wrap ErrUnknownXRef, got %v", err)
			}
			var uerr *UnknownXRefError
			if !errors.As(err, &uerr) {
				t.Fatalf("error should be *UnknownXRefError, got %T", err)
			}
			if !uerr.IsSeed {
				t.Error("UnknownXRefError.IsSeed must be true for seed errors")
			}
			if uerr.XRef != tc.seed {
				t.Errorf("UnknownXRefError.XRef = %q, want %q", uerr.XRef, tc.seed)
			}
		})
	}
}

func TestSubset_UnknownSeedReturnsError(t *testing.T) {
	doc := buildRichFixture()
	_, err := doc.Subset([]string{"@I999@"})
	if err == nil {
		t.Fatal("expected error for unknown seed xref")
	}
	if !errors.Is(err, ErrUnknownXRef) {
		t.Errorf("error should wrap ErrUnknownXRef, got %v", err)
	}
	var uerr *UnknownXRefError
	if !errors.As(err, &uerr) {
		t.Fatalf("error should be *UnknownXRefError, got %T", err)
	}
	if uerr.XRef != "@I999@" {
		t.Errorf("UnknownXRefError.XRef = %q, want @I999@", uerr.XRef)
	}
	if !uerr.IsSeed {
		t.Error("UnknownXRefError.IsSeed should be true for missing seed")
	}
}

func TestSubset_DanglingReferenceReturnsError(t *testing.T) {
	doc := buildRichFixture()
	// Mutate one record to reference a non-existent xref.
	ind := doc.GetIndividual("@I1@")
	ind.Notes = append(ind.Notes, "@MISSING@")

	_, err := doc.Subset([]string{"@I1@"})
	if err == nil {
		t.Fatal("expected error for dangling reference")
	}
	if !errors.Is(err, ErrUnknownXRef) {
		t.Errorf("error should wrap ErrUnknownXRef, got %v", err)
	}
	var uerr *UnknownXRefError
	if !errors.As(err, &uerr) {
		t.Fatalf("error should be *UnknownXRefError, got %T", err)
	}
	if uerr.XRef != "@MISSING@" {
		t.Errorf("UnknownXRefError.XRef = %q, want @MISSING@", uerr.XRef)
	}
	if uerr.IsSeed {
		t.Error("UnknownXRefError.IsSeed should be false for non-seed dangling refs")
	}
}

func TestSubset_ClosureContainsExpectedRecords(t *testing.T) {
	doc := buildRichFixture()
	sub, err := doc.Subset([]string{"@I1@"})
	if err != nil {
		t.Fatalf("Subset errored: %v", err)
	}

	want := []string{"@I1@", "@F1@", "@I2@", "@I3@", "@N1@", "@N2@", "@S1@", "@S2@", "@R1@", "@M1@"}
	got := make([]string, 0, len(sub.Records))
	for _, rec := range sub.Records {
		got = append(got, rec.XRef)
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("closure records:\n  got  %v\n  want %v", got, want)
	}
}

func TestSubset_UnreferencedRecordsExcluded(t *testing.T) {
	doc := buildRichFixture()
	sub, err := doc.Subset([]string{"@I1@"})
	if err != nil {
		t.Fatalf("Subset errored: %v", err)
	}
	for _, rec := range sub.Records {
		if rec.XRef == "@I99@" {
			t.Error("unreferenced @I99@ should not be in subset")
		}
		if rec.XRef == "@SUBM1@" {
			t.Error("unreferenced @SUBM1@ should not be in subset (header pointer alone doesn't pull it in)")
		}
	}
}

func TestSubset_HeaderSubmitterClearedWhenSubmitterNotInClosure(t *testing.T) {
	doc := buildRichFixture()
	sub, err := doc.Subset([]string{"@I1@"})
	if err != nil {
		t.Fatalf("Subset errored: %v", err)
	}
	if sub.Header.Submitter != "" {
		t.Errorf("Subset Header.Submitter = %q, want empty (SUBM1 not in closure)", sub.Header.Submitter)
	}
	if sub.Header.Version != Version70 {
		t.Errorf("Header.Version = %v, want %v", sub.Header.Version, Version70)
	}
	if sub.Header.Encoding != EncodingUTF8 {
		t.Errorf("Header.Encoding = %v, want %v", sub.Header.Encoding, EncodingUTF8)
	}
}

func TestSubset_HeaderSubmitterPreservedWhenInClosure(t *testing.T) {
	doc := buildRichFixture()
	// Add a tag on @I1@ that references @SUBM1@ via XRef field.
	ind := doc.GetRecord("@I1@")
	ind.Tags = append(ind.Tags, &Tag{Level: 1, Tag: "SUBM", XRef: "@SUBM1@"})

	sub, err := doc.Subset([]string{"@I1@"})
	if err != nil {
		t.Fatalf("Subset errored: %v", err)
	}
	if sub.Header.Submitter != "@SUBM1@" {
		t.Errorf("Subset Header.Submitter = %q, want @SUBM1@ (now in closure via tag ref)", sub.Header.Submitter)
	}
	found := false
	for _, rec := range sub.Records {
		if rec.XRef == "@SUBM1@" {
			found = true
			break
		}
	}
	if !found {
		t.Error("@SUBM1@ should be included in records when reachable")
	}
}

func TestSubset_DoesNotMutateSource(t *testing.T) {
	doc := buildRichFixture()
	originalLen := len(doc.Records)
	originalXRefMapLen := len(doc.XRefMap)
	originalI1Notes := append([]string(nil), doc.GetIndividual("@I1@").Notes...)

	sub, err := doc.Subset([]string{"@I1@"})
	if err != nil {
		t.Fatalf("Subset errored: %v", err)
	}

	if len(doc.Records) != originalLen {
		t.Errorf("source Records length changed: got %d, want %d", len(doc.Records), originalLen)
	}
	if len(doc.XRefMap) != originalXRefMapLen {
		t.Errorf("source XRefMap length changed: got %d, want %d", len(doc.XRefMap), originalXRefMapLen)
	}
	if !reflect.DeepEqual(doc.GetIndividual("@I1@").Notes, originalI1Notes) {
		t.Error("source @I1@.Notes was mutated")
	}

	// Mutate the subset and verify source is independent.
	subI1 := sub.GetIndividual("@I1@")
	if subI1 == nil {
		t.Fatal("subset @I1@ missing")
	}
	subI1.Sex = "MUTATED"
	if doc.GetIndividual("@I1@").Sex == "MUTATED" {
		t.Error("mutating subset @I1@ mutated source")
	}
}

func TestSubset_DuplicateSeedsAreDeduplicated(t *testing.T) {
	doc := buildRichFixture()
	sub, err := doc.Subset([]string{"@I1@", "@I1@", "@I1@"})
	if err != nil {
		t.Fatalf("Subset errored: %v", err)
	}
	// Count occurrences of @I1@ in subset records.
	count := 0
	for _, rec := range sub.Records {
		if rec.XRef == "@I1@" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("@I1@ appears %d times in subset, want 1", count)
	}
}

func TestSubset_NonIndividualSeedWorks(t *testing.T) {
	doc := buildRichFixture()
	sub, err := doc.Subset([]string{"@S1@"})
	if err != nil {
		t.Fatalf("Subset errored: %v", err)
	}
	want := map[string]bool{"@S1@": true, "@R1@": true}
	if len(sub.Records) != len(want) {
		t.Fatalf("Subset(@S1@) = %d records, want %d", len(sub.Records), len(want))
	}
	for _, rec := range sub.Records {
		if !want[rec.XRef] {
			t.Errorf("unexpected record %q in subset", rec.XRef)
		}
	}
}

func TestSubset_XRefMapMatchesRecords(t *testing.T) {
	doc := buildRichFixture()
	sub, err := doc.Subset([]string{"@I1@"})
	if err != nil {
		t.Fatalf("Subset errored: %v", err)
	}
	if len(sub.XRefMap) != len(sub.Records) {
		t.Errorf("XRefMap size = %d, Records size = %d", len(sub.XRefMap), len(sub.Records))
	}
	for _, rec := range sub.Records {
		mapped, ok := sub.XRefMap[rec.XRef]
		if !ok {
			t.Errorf("XRefMap missing entry for %q", rec.XRef)
			continue
		}
		if mapped != rec {
			t.Errorf("XRefMap[%q] does not point to the slice record", rec.XRef)
		}
	}
}

func TestSubset_SchemaIsCopiedAndIndependent(t *testing.T) {
	doc := buildRichFixture()
	sub, err := doc.Subset([]string{"@I1@"})
	if err != nil {
		t.Fatalf("Subset errored: %v", err)
	}
	if sub.Schema == doc.Schema {
		t.Error("subset Schema should be deep-copied, not shared")
	}
	if sub.Schema.TagMappings["_TEST"] != "http://example.com/test" {
		t.Errorf("Schema not preserved: got %v", sub.Schema.TagMappings)
	}
	sub.Schema.TagMappings["_TEST"] = "modified"
	if doc.Schema.TagMappings["_TEST"] == "modified" {
		t.Error("mutating subset Schema mutated source")
	}
}

func TestSubset_NilSourceHeaderProducesEmptyHeader(t *testing.T) {
	doc := &Document{
		Header:  nil,
		XRefMap: make(map[string]*Record),
	}
	rec := &Record{XRef: "@I1@", Type: RecordTypeIndividual, Entity: &Individual{XRef: "@I1@"}}
	doc.Records = []*Record{rec}
	doc.XRefMap["@I1@"] = rec

	sub, err := doc.Subset([]string{"@I1@"})
	if err != nil {
		t.Fatalf("Subset errored on nil-header source: %v", err)
	}
	if sub.Header == nil {
		t.Fatal("Subset must return a non-nil Header even when source.Header is nil (callers rely on the invariant)")
	}
	if sub.Header.Version != "" || sub.Header.Submitter != "" {
		t.Errorf("nil-source-header subset should have zero-value Header, got %+v", sub.Header)
	}
}

func TestSubset_InlineNoteShapedLikeXRefIsIgnored(t *testing.T) {
	// Individual.Notes can hold either a "@N1@" pointer or inline text.
	// Inline text that accidentally has the @...@ shape (rare but
	// possible — e.g., a UID-style identifier inside note text) must
	// not cause spurious ErrUnknownXRef.
	doc := buildRichFixture()
	// Real GEDCOM data: 1 NOTE "@user@example.com" would land here as
	// inline text but doesn't have the right shape. Construct a case
	// that DOES have @...@ shape with whitespace inside (inline text),
	// plus one with no whitespace (must be treated as a pointer — would
	// be a real bug to silently drop). The whitespace case is the one
	// flagged in review.
	ind := doc.GetIndividual("@I1@")
	ind.Notes = append(ind.Notes, "@inline note with spaces@")

	if _, err := doc.Subset([]string{"@I1@"}); err != nil {
		t.Fatalf("inline note shaped like @...@ with internal whitespace must be ignored, got error: %v", err)
	}
}

func TestSubset_HeaderTagsFilteredByClosure(t *testing.T) {
	doc := buildRichFixture()
	doc.Header.Tags = []*Tag{
		{Level: 1, Tag: "_CUSTOM", Value: "kept-vendor-tag"},      // no xref → kept
		{Level: 1, Tag: "NOTE", XRef: "@N1@"},                     // in closure → kept (after we walk @I1@)
		{Level: 1, Tag: "NOTE", XRef: "@UNREACHABLE@", Value: ""}, // not in closure → dropped
	}
	// Need @UNREACHABLE@ to exist in the document so the dangling check
	// in subsetClosure doesn't trigger; header tag pointers are not
	// followed by the closure walker.
	doc.Records = append(doc.Records, &Record{XRef: "@UNREACHABLE@", Type: RecordTypeNote, Entity: &Note{XRef: "@UNREACHABLE@"}})
	doc.XRefMap["@UNREACHABLE@"] = doc.Records[len(doc.Records)-1]

	sub, err := doc.Subset([]string{"@I1@"})
	if err != nil {
		t.Fatalf("Subset errored: %v", err)
	}

	gotTags := map[string]bool{}
	for _, tag := range sub.Header.Tags {
		gotTags[tag.Tag+"/"+tag.XRef] = true
	}
	if !gotTags["_CUSTOM/"] {
		t.Error("non-xref vendor header tag should be preserved")
	}
	if !gotTags["NOTE/@N1@"] {
		t.Error("header tag pointing into closure should be preserved")
	}
	if gotTags["NOTE/@UNREACHABLE@"] {
		t.Error("header tag pointing outside closure should be dropped")
	}
}
