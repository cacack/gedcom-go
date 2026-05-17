package merge_test

import (
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/gedcom"
	"github.com/cacack/gedcom-go/merge"
)

// buildRichFixture creates a small document with cross-references covering
// every reference class that RemapXRefs must update: family links,
// source citations, repository ownership, notes, media, submitter,
// shared notes, header submitter, and a header tag pointing into the
// record set.
func buildRichFixture() *gedcom.Document {
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version:   gedcom.Version70,
			Encoding:  gedcom.EncodingUTF8,
			Submitter: "@SUBM1@",
			Tags: []*gedcom.Tag{
				{Level: 1, Tag: "NOTE", XRef: "@N1@"},
			},
		},
		Trailer: &gedcom.Trailer{LineNumber: 999},
		Vendor:  "TestVendor",
		XRefMap: make(map[string]*gedcom.Record),
	}

	add := func(xref string, typ gedcom.RecordType, entity interface{}) {
		rec := &gedcom.Record{XRef: xref, Type: typ, Entity: entity}
		doc.Records = append(doc.Records, rec)
		doc.XRefMap[xref] = rec
	}

	add("@I1@", gedcom.RecordTypeIndividual, &gedcom.Individual{
		XRef:             "@I1@",
		SpouseInFamilies: []string{"@F1@"},
		Notes:            []string{"@N1@"},
		SourceCitations:  []*gedcom.SourceCitation{{SourceXRef: "@S1@"}},
		Media:            []*gedcom.MediaLink{{MediaXRef: "@M1@"}},
		Events: []*gedcom.Event{
			{Type: "BIRT", SourceCitations: []*gedcom.SourceCitation{{SourceXRef: "@S2@"}}},
		},
		Associations: []*gedcom.Association{
			{IndividualXRef: "@I2@", Role: "Godfather"},
		},
	})
	add("@I2@", gedcom.RecordTypeIndividual, &gedcom.Individual{
		XRef:             "@I2@",
		SpouseInFamilies: []string{"@F1@"},
	})
	add("@I3@", gedcom.RecordTypeIndividual, &gedcom.Individual{
		XRef:            "@I3@",
		ChildInFamilies: []gedcom.FamilyLink{{FamilyXRef: "@F1@"}},
	})
	add("@F1@", gedcom.RecordTypeFamily, &gedcom.Family{
		XRef:     "@F1@",
		Husband:  "@I1@",
		Wife:     "@I2@",
		Children: []string{"@I3@"},
		Notes:    []string{"@N2@"},
	})

	add("@N1@", gedcom.RecordTypeNote, &gedcom.Note{XRef: "@N1@", Text: "Note 1"})
	add("@N2@", gedcom.RecordTypeNote, &gedcom.Note{XRef: "@N2@", Text: "Note 2"})

	add("@S1@", gedcom.RecordTypeSource, &gedcom.Source{XRef: "@S1@", Title: "Source 1", RepositoryRef: "@R1@"})
	add("@S2@", gedcom.RecordTypeSource, &gedcom.Source{XRef: "@S2@", Title: "Source 2"})

	add("@R1@", gedcom.RecordTypeRepository, &gedcom.Repository{XRef: "@R1@", Name: "Repo 1"})

	add("@M1@", gedcom.RecordTypeMedia, &gedcom.MediaObject{
		XRef:            "@M1@",
		Files:           []*gedcom.MediaFile{{FileRef: "/m1.jpg"}},
		SourceCitations: []*gedcom.SourceCitation{{SourceXRef: "@S2@"}},
	})

	add("@SUBM1@", gedcom.RecordTypeSubmitter, &gedcom.Submitter{XRef: "@SUBM1@", Name: "Submitter"})

	add("@SN1@", gedcom.RecordTypeSharedNote, &gedcom.SharedNote{
		XRef:            "@SN1@",
		Text:            "Shared note",
		SourceCitations: []*gedcom.SourceCitation{{SourceXRef: "@S1@"}},
	})

	return doc
}

// prefixTransform wraps a prefix string into a valid XRef-shape
// transform: "@I1@" → "@A_I1@".
func prefixTransform(prefix string) func(string) string {
	return func(old string) string {
		// old has the @xref@ shape; strip the leading @ and re-add it
		// with the prefix.
		return "@" + prefix + old[1:]
	}
}

func TestRemapXRefs_NoOpTransformProducesEqualDocument(t *testing.T) {
	doc := buildRichFixture()
	noop := func(old string) string { return old }

	out, mapping, err := merge.RemapXRefs(doc, noop)
	if err != nil {
		t.Fatalf("RemapXRefs noop returned error: %v", err)
	}

	if len(mapping) != len(doc.Records) {
		t.Errorf("mapping size = %d, want %d (one per record with XRef)", len(mapping), len(doc.Records))
	}
	for old, newXRef := range mapping {
		if old != newXRef {
			t.Errorf("noop mapping mutated %q -> %q", old, newXRef)
		}
	}

	if !reflect.DeepEqual(doc, out) {
		t.Error("noop remap should produce a document deeply equal to the input")
	}
}

func TestRemapXRefs_PrefixTransformUpdatesAllRecordsAndReferences(t *testing.T) {
	doc := buildRichFixture()
	out, mapping, err := merge.RemapXRefs(doc, prefixTransform("A_"))
	if err != nil {
		t.Fatalf("RemapXRefs returned error: %v", err)
	}

	// Verify mapping covers every record with an XRef.
	wantOldXRefs := make([]string, 0, len(doc.Records))
	for _, r := range doc.Records {
		if r.XRef != "" {
			wantOldXRefs = append(wantOldXRefs, r.XRef)
		}
	}
	gotOldXRefs := make([]string, 0, len(mapping))
	for k := range mapping {
		gotOldXRefs = append(gotOldXRefs, k)
	}
	sort.Strings(wantOldXRefs)
	sort.Strings(gotOldXRefs)
	if !reflect.DeepEqual(wantOldXRefs, gotOldXRefs) {
		t.Errorf("mapping keys:\n  got  %v\n  want %v", gotOldXRefs, wantOldXRefs)
	}
	for old, newXRef := range mapping {
		want := "@A_" + old[1:]
		if newXRef != want {
			t.Errorf("mapping[%q] = %q, want %q", old, newXRef, want)
		}
	}

	// Verify every record XRef in the output starts with "@A_".
	for _, r := range out.Records {
		if r.XRef == "" {
			continue
		}
		if !strings.HasPrefix(r.XRef, "@A_") {
			t.Errorf("record XRef %q missing expected prefix", r.XRef)
		}
	}

	// Verify XRefMap keys match the new XRefs.
	for _, r := range out.Records {
		if r.XRef == "" {
			continue
		}
		mapped, ok := out.XRefMap[r.XRef]
		if !ok {
			t.Errorf("XRefMap missing entry for new XRef %q", r.XRef)
			continue
		}
		if mapped != r {
			t.Errorf("XRefMap[%q] does not point to the slice record", r.XRef)
		}
	}

	// Verify references inside entities were rewritten.
	f1 := out.GetFamily("@A_F1@")
	if f1 == nil {
		t.Fatal("family @A_F1@ missing from output")
	}
	if f1.Husband != "@A_I1@" {
		t.Errorf("F1.Husband = %q, want @A_I1@", f1.Husband)
	}
	if f1.Wife != "@A_I2@" {
		t.Errorf("F1.Wife = %q, want @A_I2@", f1.Wife)
	}
	if len(f1.Children) != 1 || f1.Children[0] != "@A_I3@" {
		t.Errorf("F1.Children = %v, want [@A_I3@]", f1.Children)
	}
	if len(f1.Notes) != 1 || f1.Notes[0] != "@A_N2@" {
		t.Errorf("F1.Notes = %v, want [@A_N2@]", f1.Notes)
	}

	i1 := out.GetIndividual("@A_I1@")
	if i1 == nil {
		t.Fatal("individual @A_I1@ missing from output")
	}
	if len(i1.SpouseInFamilies) != 1 || i1.SpouseInFamilies[0] != "@A_F1@" {
		t.Errorf("I1.SpouseInFamilies = %v, want [@A_F1@]", i1.SpouseInFamilies)
	}
	if len(i1.Notes) != 1 || i1.Notes[0] != "@A_N1@" {
		t.Errorf("I1.Notes = %v, want [@A_N1@]", i1.Notes)
	}
	if len(i1.SourceCitations) != 1 || i1.SourceCitations[0].SourceXRef != "@A_S1@" {
		t.Errorf("I1.SourceCitations[0].SourceXRef = %q, want @A_S1@", i1.SourceCitations[0].SourceXRef)
	}
	if len(i1.Media) != 1 || i1.Media[0].MediaXRef != "@A_M1@" {
		t.Errorf("I1.Media[0].MediaXRef = %q, want @A_M1@", i1.Media[0].MediaXRef)
	}
	if len(i1.Events) != 1 || i1.Events[0].SourceCitations[0].SourceXRef != "@A_S2@" {
		t.Errorf("I1 event citation = %q, want @A_S2@", i1.Events[0].SourceCitations[0].SourceXRef)
	}
	if len(i1.Associations) != 1 || i1.Associations[0].IndividualXRef != "@A_I2@" {
		t.Errorf("I1.Associations[0] = %q, want @A_I2@", i1.Associations[0].IndividualXRef)
	}

	// Verify source repository ref was rewritten.
	s1 := out.GetSource("@A_S1@")
	if s1 == nil {
		t.Fatal("source @A_S1@ missing")
	}
	if s1.RepositoryRef != "@A_R1@" {
		t.Errorf("S1.RepositoryRef = %q, want @A_R1@", s1.RepositoryRef)
	}

	// Verify media citation was rewritten.
	m1 := out.GetMediaObject("@A_M1@")
	if m1 == nil {
		t.Fatal("media @A_M1@ missing")
	}
	if len(m1.SourceCitations) != 1 || m1.SourceCitations[0].SourceXRef != "@A_S2@" {
		t.Errorf("M1 source citation = %q, want @A_S2@", m1.SourceCitations[0].SourceXRef)
	}

	// Verify shared note citation was rewritten.
	sn1 := out.GetSharedNote("@A_SN1@")
	if sn1 == nil {
		t.Fatal("shared note @A_SN1@ missing")
	}
	if len(sn1.SourceCitations) != 1 || sn1.SourceCitations[0].SourceXRef != "@A_S1@" {
		t.Errorf("SN1 source citation = %q, want @A_S1@", sn1.SourceCitations[0].SourceXRef)
	}
}

func TestRemapXRefs_PreservesReferentialIntegrity(t *testing.T) {
	doc := buildRichFixture()
	out, _, err := merge.RemapXRefs(doc, prefixTransform("X_"))
	if err != nil {
		t.Fatalf("RemapXRefs returned error: %v", err)
	}

	// Build the set of valid record XRefs in the output.
	validXRefs := make(map[string]bool)
	for _, r := range out.Records {
		if r.XRef != "" {
			validXRefs[r.XRef] = true
		}
	}

	// Walk every record and verify every reference resolves.
	for _, r := range out.Records {
		gedcom.Visit(r, func(ref string) {
			if !validXRefs[ref] {
				t.Errorf("record %q references %q which is not in the output", r.XRef, ref)
			}
		})
	}

	// Verify header submitter resolves if non-empty.
	if out.Header != nil && out.Header.Submitter != "" {
		if !validXRefs[out.Header.Submitter] {
			t.Errorf("Header.Submitter = %q is not a valid record XRef", out.Header.Submitter)
		}
	}
}

func TestRemapXRefs_DoesNotMutateInput(t *testing.T) {
	doc := buildRichFixture()
	snapshot := doc.Clone()

	_, _, err := merge.RemapXRefs(doc, prefixTransform("A_"))
	if err != nil {
		t.Fatalf("RemapXRefs returned error: %v", err)
	}

	if !reflect.DeepEqual(doc, snapshot) {
		t.Error("RemapXRefs mutated the input document (it should clone before remapping)")
	}
}

func TestRemapXRefs_RejectsMalformedOutput(t *testing.T) {
	cases := []struct {
		name      string
		transform func(string) string
	}{
		{
			name: "missing leading @",
			transform: func(old string) string {
				return "A_" + old[1:]
			},
		},
		{
			name: "missing trailing @",
			transform: func(old string) string {
				return strings.TrimSuffix("@A_"+old[1:], "@")
			},
		},
		{
			name: "empty string",
			transform: func(old string) string {
				return ""
			},
		},
		{
			name: "VOID sentinel",
			transform: func(old string) string {
				return "@VOID@"
			},
		},
		{
			name: "contains whitespace",
			transform: func(old string) string {
				return "@bad new@"
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := buildRichFixture()
			out, mapping, err := merge.RemapXRefs(doc, tc.transform)
			if err == nil {
				t.Fatalf("expected error for malformed transform output, got nil")
			}
			if out != nil {
				t.Error("output document should be nil on error")
			}
			if mapping != nil {
				t.Error("mapping should be nil on error")
			}
			if !errors.Is(err, merge.ErrInvalidRemap) {
				t.Errorf("error should wrap ErrInvalidRemap, got %v", err)
			}
			var rerr *merge.RemapError
			if !errors.As(err, &rerr) {
				t.Fatalf("error should be *RemapError, got %T", err)
			}
			if rerr.Old == "" {
				t.Error("RemapError.Old should name the failing input")
			}
		})
	}
}

func TestRemapXRefs_RejectsCollisions(t *testing.T) {
	doc := buildRichFixture()
	collide := func(old string) string {
		// Every input maps to the same output → collision on the
		// second record processed.
		return "@COLLIDE@"
	}

	out, mapping, err := merge.RemapXRefs(doc, collide)
	if err == nil {
		t.Fatal("expected collision error, got nil")
	}
	if out != nil {
		t.Error("output document should be nil on error")
	}
	if mapping != nil {
		t.Error("mapping should be nil on error")
	}
	if !errors.Is(err, merge.ErrInvalidRemap) {
		t.Errorf("collision error should wrap ErrInvalidRemap, got %v", err)
	}
	var rerr *merge.RemapError
	if !errors.As(err, &rerr) {
		t.Fatalf("collision error should be *RemapError, got %T", err)
	}
	if rerr.New != "@COLLIDE@" {
		t.Errorf("RemapError.New = %q, want @COLLIDE@", rerr.New)
	}
	if !strings.Contains(rerr.Reason, "collides") {
		t.Errorf("RemapError.Reason should mention collision, got %q", rerr.Reason)
	}
}

func TestRemapXRefs_NilDocReturnsError(t *testing.T) {
	out, mapping, err := merge.RemapXRefs(nil, func(s string) string { return s })
	if err == nil {
		t.Fatal("expected error for nil doc, got nil")
	}
	if out != nil || mapping != nil {
		t.Error("nil doc should return nil out and nil mapping")
	}
	// Nil-doc is a caller bug, not a malformed-remap; should NOT wrap ErrInvalidRemap.
	if errors.Is(err, merge.ErrInvalidRemap) {
		t.Error("nil doc error should not wrap ErrInvalidRemap (different failure class)")
	}
}

func TestRemapXRefs_NilTransformReturnsError(t *testing.T) {
	doc := buildRichFixture()
	out, mapping, err := merge.RemapXRefs(doc, nil)
	if err == nil {
		t.Fatal("expected error for nil transform, got nil")
	}
	if out != nil || mapping != nil {
		t.Error("nil transform should return nil out and nil mapping")
	}
	if errors.Is(err, merge.ErrInvalidRemap) {
		t.Error("nil transform error should not wrap ErrInvalidRemap (different failure class)")
	}
}

func TestRemapXRefs_HeaderSubmitterIsRemapped(t *testing.T) {
	doc := buildRichFixture()
	out, _, err := merge.RemapXRefs(doc, prefixTransform("A_"))
	if err != nil {
		t.Fatalf("RemapXRefs returned error: %v", err)
	}
	if out.Header == nil {
		t.Fatal("Header missing from output")
	}
	if out.Header.Submitter != "@A_SUBM1@" {
		t.Errorf("Header.Submitter = %q, want @A_SUBM1@", out.Header.Submitter)
	}
	// Header tag pointing into the remap set should also be rewritten.
	if len(out.Header.Tags) != 1 {
		t.Fatalf("header tags = %d, want 1", len(out.Header.Tags))
	}
	if out.Header.Tags[0].XRef != "@A_N1@" {
		t.Errorf("header tag XRef = %q, want @A_N1@", out.Header.Tags[0].XRef)
	}
}

func TestRemapXRefs_MappingIsDefensivelyCopied(t *testing.T) {
	doc := buildRichFixture()
	out, mapping, err := merge.RemapXRefs(doc, prefixTransform("A_"))
	if err != nil {
		t.Fatalf("RemapXRefs returned error: %v", err)
	}

	// Mutate the returned mapping.
	mapping["@I1@"] = "@MUTATED@"

	// A second call should produce the same result; if the internal
	// state were shared with the returned map we'd see drift.
	out2, mapping2, err := merge.RemapXRefs(doc, prefixTransform("A_"))
	if err != nil {
		t.Fatalf("second RemapXRefs returned error: %v", err)
	}
	if mapping2["@I1@"] != "@A_I1@" {
		t.Errorf("second mapping was affected by mutation of first: got %q, want @A_I1@", mapping2["@I1@"])
	}
	// Output should still resolve correctly.
	if out.GetIndividual("@A_I1@") == nil {
		t.Error("output @A_I1@ should still exist after mapping mutation")
	}
	if out2.GetIndividual("@A_I1@") == nil {
		t.Error("second output @A_I1@ missing")
	}
}

func TestRemapXRefs_RoundTripWithInverseTransform(t *testing.T) {
	doc := buildRichFixture()
	out, mapping, err := merge.RemapXRefs(doc, prefixTransform("A_"))
	if err != nil {
		t.Fatalf("forward remap returned error: %v", err)
	}

	// Build an inverse from the mapping returned by the forward pass.
	inverse := make(map[string]string, len(mapping))
	for old, newXRef := range mapping {
		inverse[newXRef] = old
	}
	inverseTransform := func(s string) string {
		if orig, ok := inverse[s]; ok {
			return orig
		}
		// Should not happen in this test — every XRef in `out` was
		// produced by the forward transform.
		t.Fatalf("inverse missing entry for %q", s)
		return s
	}

	back, _, err := merge.RemapXRefs(out, inverseTransform)
	if err != nil {
		t.Fatalf("reverse remap returned error: %v", err)
	}

	if !reflect.DeepEqual(doc, back) {
		t.Error("round-trip (forward + inverse) should produce a document equal to the original")
	}
}

func TestRemapXRefs_RecordsWithEmptyXRefAreSkipped(t *testing.T) {
	doc := buildRichFixture()
	// Add an empty-XRef record (header-style placeholder) and a nil
	// entry to exercise both skip paths in a single append.
	doc.Records = append(doc.Records,
		&gedcom.Record{
			XRef:   "",
			Type:   gedcom.RecordTypeNote,
			Entity: &gedcom.Note{Text: "anonymous"},
		},
		nil,
	)

	called := make(map[string]bool)
	transform := func(old string) string {
		called[old] = true
		return "@P_" + old[1:]
	}

	out, mapping, err := merge.RemapXRefs(doc, transform)
	if err != nil {
		t.Fatalf("RemapXRefs returned error: %v", err)
	}
	if called[""] {
		t.Error("transform should not be called with empty XRef")
	}
	if _, ok := mapping[""]; ok {
		t.Error("mapping should not contain an entry for empty XRef")
	}
	// Output should still include the empty-XRef record (it just isn't remapped).
	emptyCount := 0
	nilCount := 0
	for _, r := range out.Records {
		if r == nil {
			nilCount++
			continue
		}
		if r.XRef == "" {
			emptyCount++
		}
	}
	if emptyCount != 1 {
		t.Errorf("output should preserve 1 empty-XRef record, got %d", emptyCount)
	}
	if nilCount != 1 {
		t.Errorf("output should preserve 1 nil record (Clone fidelity), got %d", nilCount)
	}
}
