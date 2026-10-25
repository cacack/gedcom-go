package merge_test

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/merge"
)

// spacedXRefDoc declares vers and links an individual to a note through a
// spaced identifier, which GEDCOM 5.5 and 5.5.1 allow and 7.0 does not.
func spacedXRefDoc(vers string) string {
	return strings.Join([]string{
		"0 HEAD",
		"1 GEDC",
		"2 VERS " + vers,
		"1 CHAR UTF-8",
		"0 @I1@ INDI",
		"1 NAME John /Smith/",
		"1 NOTE @N 1@",
		"0 @N 1@ NOTE Shared note",
		"0 TRLR",
	}, "\n") + "\n"
}

func prefixA(old string) string { return "@A_" + old[1:] }

// TestRemapXRefs_SpacedXRef551 pins issue #591: a spaced identifier is valid
// in a 5.5/5.5.1 document, so remapping it succeeds and its pointer follows.
func TestRemapXRefs_SpacedXRef551(t *testing.T) {
	for _, vers := range []string{"5.5", "5.5.1"} {
		t.Run(vers, func(t *testing.T) {
			doc, err := decoder.Decode(strings.NewReader(spacedXRefDoc(vers)))
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			out, mapping, err := merge.RemapXRefs(doc, prefixA)
			if err != nil {
				t.Fatalf("RemapXRefs: %v", err)
			}
			if mapping["@N 1@"] != "@A_N 1@" {
				t.Errorf("mapping[@N 1@] = %q, want @A_N 1@", mapping["@N 1@"])
			}
			person := out.GetIndividual("@A_I1@")
			if person == nil {
				t.Fatal("individual @A_I1@ missing")
			}
			if len(person.NoteXRefs) != 1 || out.GetNote(person.NoteXRefs[0]) == nil {
				t.Errorf("NoteXRefs = %q, want one pointer resolving to a note", person.NoteXRefs)
			}
		})
	}
}

// TestRemapXRefs_SpacedXRef70 pins that GEDCOM 7.0 does not treat a spaced
// identifier as a pointer: the record is skipped and keeps its XRef, while
// the valid record beside it is still remapped (#397).
func TestRemapXRefs_SpacedXRef70(t *testing.T) {
	result, err := decoder.DecodeWithDiagnostics(strings.NewReader(spacedXRefDoc("7.0")), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics: %v", err)
	}
	out, mapping, err := merge.RemapXRefs(result.Document, prefixA)
	if err != nil {
		t.Fatalf("RemapXRefs: %v", err)
	}
	if _, ok := mapping["@N 1@"]; ok {
		t.Errorf("mapping has entry for non-pointer @N 1@: %q", mapping)
	}
	if mapping["@I1@"] != "@A_I1@" {
		t.Errorf("mapping[@I1@] = %q, want @A_I1@", mapping["@I1@"])
	}
	if out.GetNote("@N 1@") == nil {
		t.Error("note @N 1@ missing or renamed")
	}
}

// TestRemapXRefs_SpacedXRefUndeclared pins that a document built in code,
// which declares no version, keeps the space-free grammar: the spaced
// record is skipped rather than remapped (#397).
func TestRemapXRefs_SpacedXRefUndeclared(t *testing.T) {
	rec := &gedcom.Record{XRef: "@N 1@", Type: gedcom.RecordTypeNote, Entity: &gedcom.Note{XRef: "@N 1@"}}
	doc := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version551},
		Records: []*gedcom.Record{rec},
		XRefMap: map[string]*gedcom.Record{rec.XRef: rec},
	}
	out, mapping, err := merge.RemapXRefs(doc, prefixA)
	if err != nil {
		t.Fatalf("RemapXRefs: %v", err)
	}
	if len(mapping) != 0 {
		t.Errorf("mapping = %q, want empty", mapping)
	}
	if out.Records[0].XRef != "@N 1@" {
		t.Errorf("XRef = %q, want @N 1@", out.Records[0].XRef)
	}
}

// TestCombine_SpacedXRefDoc1Undeclared pins that Combine keeps doc2's spaced
// 5.5.1 pointers recognized when doc1 declares no version: the combined
// header adopts doc2's DeclaredVersion, so Subset follows "@N 1@" (#591).
func TestCombine_SpacedXRefDoc1Undeclared(t *testing.T) {
	doc1 := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version551},
		XRefMap: map[string]*gedcom.Record{},
	}
	doc2, err := decoder.Decode(strings.NewReader(spacedXRefDoc("5.5.1")))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	out, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}
	if out.Header.DeclaredVersion != gedcom.Version551 {
		t.Errorf("DeclaredVersion = %q, want %q", out.Header.DeclaredVersion, gedcom.Version551)
	}
	for _, c := range report.HeaderConflicts {
		if c.Field == "DeclaredVersion" {
			t.Errorf("unexpected DeclaredVersion conflict %+v", c)
		}
	}
	sub, err := out.Subset([]string{"@I1@"})
	if err != nil {
		t.Fatalf("Subset: %v", err)
	}
	if sub.GetNote("@N 1@") == nil {
		t.Error("Subset did not follow the spaced pointer @N 1@")
	}
}

// TestCombine_DeclaredVersionNotPromotedOverDoc1GEDC pins that doc2's
// DeclaredVersion is not promoted when doc1 carries its own raw GEDC with an
// unrecognized VERS: doc1's GEDC wins in the encoded header, so the field
// must not claim doc2's version (#591).
func TestCombine_DeclaredVersionNotPromotedOverDoc1GEDC(t *testing.T) {
	doc1, err := decoder.Decode(strings.NewReader(spacedXRefDoc("5.5.5")))
	if err != nil {
		t.Fatalf("Decode doc1: %v", err)
	}
	if doc1.Header.DeclaredVersion != "" {
		t.Fatalf("doc1 DeclaredVersion = %q, want empty", doc1.Header.DeclaredVersion)
	}
	doc1.Header.Version = gedcom.Version551 // make the Versions agree regardless of the guess
	doc2, err := decoder.Decode(strings.NewReader(spacedXRefDoc("5.5.1")))
	if err != nil {
		t.Fatalf("Decode doc2: %v", err)
	}
	out, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{CollisionStrategy: merge.PrefixDoc2, Prefix: "B_"})
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}
	if out.Header.DeclaredVersion != "" {
		t.Errorf("DeclaredVersion = %q, want empty (doc1's GEDC wins)", out.Header.DeclaredVersion)
	}
	var gedcConflict bool
	for _, c := range report.HeaderConflicts {
		if c.Field == "Tags.GEDC" {
			gedcConflict = true
		}
	}
	if !gedcConflict {
		t.Errorf("HeaderConflicts = %+v, want a Tags.GEDC conflict", report.HeaderConflicts)
	}
}

// TestCombine_DeclaredVersionConflict pins that two differing declared
// versions keep doc1's and record a DeclaredVersion conflict.
func TestCombine_DeclaredVersionConflict(t *testing.T) {
	doc1 := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version551, DeclaredVersion: gedcom.Version55},
		XRefMap: map[string]*gedcom.Record{},
	}
	doc2 := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version551, DeclaredVersion: gedcom.Version551},
		XRefMap: map[string]*gedcom.Record{},
	}
	out, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}
	if out.Header.DeclaredVersion != gedcom.Version55 {
		t.Errorf("DeclaredVersion = %q, want %q", out.Header.DeclaredVersion, gedcom.Version55)
	}
	var found bool
	for _, c := range report.HeaderConflicts {
		if c.Field == "DeclaredVersion" && c.Doc1 == "5.5" && c.Doc2 == "5.5.1" {
			found = true
		}
	}
	if !found {
		t.Errorf("HeaderConflicts = %+v, want a DeclaredVersion conflict", report.HeaderConflicts)
	}
}
