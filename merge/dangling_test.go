package merge_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/merge"
	"github.com/cacack/gedcom-go/v3/validator"
)

// danglingP1 and danglingP2 are the fixture of #925: p2's person points at
// family @F9@, which p2 does not define and p1 does.
const (
	danglingHead = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n"
	danglingP1   = danglingHead +
		"0 @I1@ INDI\n1 NAME Ann /Example/\n1 FAMS @F9@\n" +
		"0 @F9@ FAM\n1 WIFE @I1@\n0 TRLR\n"
	danglingP2 = danglingHead +
		"0 @I1@ INDI\n1 NAME Bob /Sample/\n1 FAMC @F9@\n0 TRLR\n"
)

// assertStillDangling checks that doc2's person, at xref in out, points at
// a family out does not define (so not doc1's @F9@), and that validating
// out still reports the broken FAMC.
func assertStillDangling(t *testing.T, out *gedcom.Document, xref string) {
	t.Helper()
	famc := rawValue(t, out, xref, "FAMC")
	if famc == "@F9@" {
		t.Fatalf("doc2's FAMC resolves to doc1's @F9@")
	}
	if out.GetRecord(famc) != nil {
		t.Errorf("doc2's FAMC %s names a record; want it dangling", famc)
	}
	if ind := out.GetIndividual(xref); ind == nil || len(ind.ChildInFamilies) != 1 ||
		ind.ChildInFamilies[0].FamilyXRef != famc {
		t.Errorf("typed FAMC not rewritten to %s: %+v", famc, ind)
	}
	if got := rawValue(t, out, "@I1@", "FAMS"); got != "@F9@" {
		t.Errorf("doc1 FAMS = %q, want @F9@", got)
	}
	found := false
	for _, e := range validator.New().Validate(out) {
		if strings.Contains(e.Error(), famc) {
			found = true
		}
	}
	if !found {
		t.Errorf("validation of the result does not report the dangling %s", famc)
	}
}

// TestCombine_DanglingPointer_Strategies pins #925 for the strategies that
// rename only collisions: the dangling @F9@ is renamed with doc2.
func TestCombine_DanglingPointer_Strategies(t *testing.T) {
	tests := []struct {
		name    string
		opts    merge.CombineOptions
		person  string
		mapping map[string]string
	}{
		{
			name:    "PrefixDoc2",
			opts:    merge.CombineOptions{CollisionStrategy: merge.PrefixDoc2, Prefix: "B_"},
			person:  "@B_I1@",
			mapping: map[string]string{"@I1@": "@B_I1@", "@F9@": "@B_F9@"},
		},
		{
			name:    "RenumberDoc2",
			opts:    merge.CombineOptions{CollisionStrategy: merge.RenumberDoc2},
			person:  "@I2@",
			mapping: map[string]string{"@I1@": "@I2@", "@F9@": "@F10@"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p1 := decodePrefixAll(t, danglingP1)
			p2 := decodePrefixAll(t, danglingP2)
			out, report, err := merge.Combine(p1, p2, tt.opts)
			if err != nil {
				t.Fatalf("Combine: %v", err)
			}
			if !reflect.DeepEqual(report.RemappedXRefs, tt.mapping) {
				t.Errorf("RemappedXRefs = %v, want %v", report.RemappedXRefs, tt.mapping)
			}
			assertStillDangling(t, out, tt.person)
			if got := rawValue(t, p2, "@I1@", "FAMC"); got != "@F9@" {
				t.Errorf("input doc2 was mutated: FAMC = %q", got)
			}
		})
	}
}

// TestCombine_DanglingPointer_RemapThenCombine pins #925 for a caller that
// prefixes doc2 with RemapXRefs and then combines with ErrorOnCollision.
func TestCombine_DanglingPointer_RemapThenCombine(t *testing.T) {
	p1 := decodePrefixAll(t, danglingP1)
	p2 := decodePrefixAll(t, danglingP2)
	remapped, mapping, err := merge.RemapXRefs(p2, func(old string) string { return "@B_" + old[1:] })
	if err != nil {
		t.Fatalf("RemapXRefs: %v", err)
	}
	want := map[string]string{"@I1@": "@B_I1@", "@F9@": "@B_F9@"}
	if !reflect.DeepEqual(mapping, want) {
		t.Errorf("mapping = %v, want %v", mapping, want)
	}
	out, _, err := merge.Combine(p1, remapped, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}
	assertStillDangling(t, out, "@B_I1@")
}

// TestCombine_DanglingPointer_Zip pins that ZipSharedXRefs is unchanged: a
// shared XRef is the same record by design, so doc2's @F9@ joins doc1's.
func TestCombine_DanglingPointer_Zip(t *testing.T) {
	p1 := decodePrefixAll(t, danglingP1)
	p2 := decodePrefixAll(t, danglingHead+"0 @I2@ INDI\n1 NAME Bob /Sample/\n1 FAMC @F9@\n0 TRLR\n")
	out, report, err := merge.Combine(p1, p2, merge.CombineOptions{CollisionStrategy: merge.ZipSharedXRefs})
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}
	if len(report.RemappedXRefs) != 0 {
		t.Errorf("RemappedXRefs = %v, want none", report.RemappedXRefs)
	}
	if got := rawValue(t, out, "@I2@", "FAMC"); got != "@F9@" {
		t.Errorf("FAMC = %q, want @F9@", got)
	}
}

// TestCombine_DanglingPointer_LeftAloneWhenDoc1Lacks pins that PrefixDoc2
// and RenumberDoc2 rename a dangling pointer only when doc1 defines its XRef.
func TestCombine_DanglingPointer_LeftAloneWhenDoc1Lacks(t *testing.T) {
	p1 := decodePrefixAll(t, danglingHead+"0 @I1@ INDI\n0 TRLR\n")
	p2 := decodePrefixAll(t, danglingP2)
	for _, opts := range []merge.CombineOptions{
		{CollisionStrategy: merge.PrefixDoc2, Prefix: "B_"},
		{CollisionStrategy: merge.RenumberDoc2},
	} {
		out, report, err := merge.Combine(p1, p2, opts)
		if err != nil {
			t.Fatalf("Combine(%v): %v", opts.CollisionStrategy, err)
		}
		if _, ok := report.RemappedXRefs["@F9@"]; ok {
			t.Errorf("strategy %v renamed @F9@: %v", opts.CollisionStrategy, report.RemappedXRefs)
		}
		person := report.RemappedXRefs["@I1@"]
		if got := rawValue(t, out, person, "FAMC"); got != "@F9@" {
			t.Errorf("strategy %v FAMC = %q, want @F9@", opts.CollisionStrategy, got)
		}
	}
}

// TestCombine_DanglingPointer_PrefixCollision pins that under PrefixDoc2 a
// renamed dangling pointer may not land on a doc1 record.
func TestCombine_DanglingPointer_PrefixCollision(t *testing.T) {
	p1 := decodePrefixAll(t, danglingP1[:len(danglingP1)-len("0 TRLR\n")]+"0 @B_F9@ FAM\n0 TRLR\n")
	p2 := decodePrefixAll(t, danglingHead+"0 @I2@ INDI\n1 FAMC @F9@\n0 TRLR\n")
	_, _, err := merge.Combine(p1, p2, merge.CombineOptions{CollisionStrategy: merge.PrefixDoc2, Prefix: "B_"})
	var cerr *merge.CombineError
	if !errors.As(err, &cerr) {
		t.Fatalf("err = %v, want *CombineError", err)
	}
	if cerr.Kind != "prefix-collision" || !reflect.DeepEqual(cerr.Colliding, []string{"@B_F9@"}) {
		t.Errorf("Kind = %q, Colliding = %q, want prefix-collision [@B_F9@]", cerr.Kind, cerr.Colliding)
	}
}

// TestCombine_DanglingPointer_RenumberSkipsDanglingNames pins that
// RenumberDoc2 never gives a renumbered record the name of a pointer doc2
// leaves dangling.
func TestCombine_DanglingPointer_RenumberSkipsDanglingNames(t *testing.T) {
	p1 := decodePrefixAll(t, danglingHead+"0 @I1@ INDI\n0 TRLR\n")
	p2 := decodePrefixAll(t, danglingHead+"0 @I1@ INDI\n1 ASSO @I5@\n2 RELA Friend\n0 TRLR\n")
	_, report, err := merge.Combine(p1, p2, merge.CombineOptions{CollisionStrategy: merge.RenumberDoc2})
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}
	if got := report.RemappedXRefs["@I1@"]; got != "@I6@" {
		t.Errorf("@I1@ renumbered to %s, want @I6@ (past the dangling @I5@)", got)
	}
}

// TestRemapXRefs_DanglingPointer pins that RemapXRefs passes a dangling
// pointer to transform and rewrites the pointers to it.
func TestRemapXRefs_DanglingPointer(t *testing.T) {
	doc := decodePrefixAll(t, danglingP2)
	var seen []string
	out, mapping, err := merge.RemapXRefs(doc, func(old string) string {
		seen = append(seen, old)
		return "@X" + old[1:]
	})
	if err != nil {
		t.Fatalf("RemapXRefs: %v", err)
	}
	if !reflect.DeepEqual(seen, []string{"@I1@", "@F9@"}) {
		t.Errorf("transform saw %v, want records then dangling pointers", seen)
	}
	if mapping["@F9@"] != "@XF9@" {
		t.Errorf("mapping = %v", mapping)
	}
	if got := rawValue(t, out, "@XI1@", "FAMC"); got != "@XF9@" {
		t.Errorf("FAMC = %q, want @XF9@", got)
	}
}
