package validator

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Issue #778: the streaming validator keeps only what Finalize can still
// report, so its memory stays bounded on files of millions of records.

// TestStreamingValidator_DeclaredReferencesNotKept pins that a reference to
// an XRef already declared is not kept: it can never be orphaned.
func TestStreamingValidator_DeclaredReferencesNotKept(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})
	sv.ValidateRecord(&gedcom.Record{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: &gedcom.Family{XRef: "@F1@"}})
	for i := range 10_000 {
		xref := fmt.Sprintf("@I%d@", i)
		ind := &gedcom.Individual{XRef: xref, SpouseInFamilies: []gedcom.FamilyLink{{FamilyXRef: "@F1@"}}}
		sv.ValidateRecord(&gedcom.Record{XRef: xref, Type: gedcom.RecordTypeIndividual, Entity: ind})
	}
	if n := sv.refs.pending.len(); n != 0 {
		t.Errorf("pending references = %d, want 0: every reference names a declared XRef", n)
	}
	if got := sv.UsedXRefCount(); got != 1 {
		t.Errorf("UsedXRefCount = %d, want 1 (@F1@)", got)
	}
}

// TestStreamingValidator_ResolvedReferencesDropped pins that forward
// references are discarded once their XRef is declared, so the pending list
// does not keep growing over a file that resolves its references.
func TestStreamingValidator_ResolvedReferencesDropped(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})
	const n = 50_000
	for i := range n {
		ind := fmt.Sprintf("@I%d@", i)
		fam := fmt.Sprintf("@F%d@", i)
		sv.ValidateRecord(&gedcom.Record{XRef: ind, Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{
			XRef: ind, SpouseInFamilies: []gedcom.FamilyLink{{FamilyXRef: fam}},
		}})
		sv.ValidateRecord(&gedcom.Record{XRef: fam, Type: gedcom.RecordTypeFamily, Entity: &gedcom.Family{XRef: fam, Husband: ind}})
	}
	if got := sv.refs.pending.len(); got > 2*minCompactAt {
		t.Errorf("pending references = %d after %d resolved forward references, want at most %d", got, n, 2*minCompactAt)
	}
	if issues := sv.Finalize(); len(issues) != 0 {
		t.Errorf("Finalize = %v, want none", issues)
	}
	if sv.SeenXRefCount() != 2*n || sv.UsedXRefCount() != 2*n {
		t.Errorf("SeenXRefCount, UsedXRefCount = %d, %d, want %d, %d", sv.SeenXRefCount(), sv.UsedXRefCount(), 2*n, 2*n)
	}
}

// TestStreamingValidator_OrphansMatchReferenceModel checks Finalize, the
// counts, and duplicate reports against a direct map-based model of what
// they mean, over a generated stream of records with forward, backward,
// orphaned, duplicate and @VOID@ references, large enough to grow and
// rehash every internal structure.
func TestStreamingValidator_OrphansMatchReferenceModel(t *testing.T) {
	for _, ver := range []gedcom.Version{"", gedcom.Version551, gedcom.Version70} {
		t.Run(string(ver), func(t *testing.T) {
			rng := rand.New(rand.NewPCG(778, 1))
			sv := NewStreamingValidator(StreamingOptions{Version: ver})
			voidIsNull := ver != gedcom.Version551

			type use struct{ record, context, field string }
			declared := map[string]int{}
			used := map[string][]use{}
			var wantDups, gotDups []string
			pick := func(prefix string) string {
				if rng.IntN(50) == 0 {
					return "@VOID@"
				}
				return fmt.Sprintf("@%s%d@", prefix, rng.IntN(30_000))
			}

			for line := 1; line <= 40_000; line++ {
				var rec *gedcom.Record
				xref := pick("I")
				if rng.IntN(2) == 0 {
					xref = pick("F")
				}
				if xref == "@VOID@" {
					continue
				}
				if xref[1] == 'I' {
					ind := &gedcom.Individual{XRef: xref}
					for i := range rng.IntN(3) {
						f := pick("F")
						ind.SpouseInFamilies = append(ind.SpouseInFamilies, gedcom.FamilyLink{FamilyXRef: f})
						used[f] = append(used[f], use{xref, "FAMS", fmt.Sprintf("SpouseInFamilies[%d]", i)})
					}
					if rng.IntN(4) == 0 {
						s := pick("S")
						ind.NoteXRefs = []string{s}
						used[s] = append(used[s], use{xref, "NOTE", "NoteXRefs[0]"})
					}
					rec = &gedcom.Record{XRef: xref, Type: gedcom.RecordTypeIndividual, Entity: ind, LineNumber: line}
				} else {
					fam := &gedcom.Family{XRef: xref}
					if rng.IntN(2) == 0 {
						fam.Husband = pick("I")
						used[fam.Husband] = append(used[fam.Husband], use{xref, "HUSB", "Husband"})
					}
					for i := range rng.IntN(3) {
						c := pick("I")
						fam.Children = append(fam.Children, c)
						used[c] = append(used[c], use{xref, "CHIL", fmt.Sprintf("Children[%d]", i)})
					}
					rec = &gedcom.Record{XRef: xref, Type: gedcom.RecordTypeFamily, Entity: fam, LineNumber: line}
				}
				if first, ok := declared[xref]; ok {
					wantDups = append(wantDups, fmt.Sprintf("%s %d %d", xref, line, first))
				} else {
					declared[xref] = line
				}
				for _, issue := range sv.ValidateRecord(rec) {
					if issue.Code == CodeDuplicateXRef {
						var first int
						_, _ = fmt.Sscanf(issue.Message[strings.LastIndex(issue.Message, " ")+1:], "%d", &first)
						gotDups = append(gotDups, fmt.Sprintf("%s %d %d", issue.RecordXRef, issue.LineNumber, first))
					}
				}
			}

			var want []string
			for xref, uses := range used {
				if _, ok := declared[xref]; ok || (xref == "@VOID@" && voidIsNull) {
					continue
				}
				for _, u := range uses {
					want = append(want, fmt.Sprintf("%s %s %s %s", u.record, xref, u.context, u.field))
				}
			}
			var got []string
			for _, issue := range sv.Finalize() {
				got = append(got, fmt.Sprintf("%s %s %s %s", issue.RecordXRef, issue.RelatedXRef,
					issue.Details["reference_type"], issue.Details["field"]))
			}
			sort.Strings(want)
			sort.Strings(got)
			if len(want) == 0 || len(wantDups) == 0 {
				t.Fatal("the generated stream has no orphans or no duplicates; it no longer exercises Finalize")
			}
			if strings.Join(got, "\n") != strings.Join(want, "\n") {
				t.Errorf("Finalize reports %d orphaned references, the model %d", len(got), len(want))
			}
			if strings.Join(gotDups, "\n") != strings.Join(wantDups, "\n") {
				t.Errorf("duplicate reports differ from the model: got %d, want %d", len(gotDups), len(wantDups))
			}
			if sv.SeenXRefCount() != len(declared) {
				t.Errorf("SeenXRefCount = %d, want %d", sv.SeenXRefCount(), len(declared))
			}
			if sv.UsedXRefCount() != len(used) {
				t.Errorf("UsedXRefCount = %d, want %d", sv.UsedXRefCount(), len(used))
			}
		})
	}
}

// TestStreamingValidator_DuplicateXRef_Lines pins that a duplicate report
// names the first declaration's line whatever its size, and names none for
// a line that is not positive, as for an unknown line.
func TestStreamingValidator_DuplicateXRef_Lines(t *testing.T) {
	for _, tc := range []struct {
		line int
		want string
	}{
		{1, "XRef @S1@ is already defined at line 1"},
		{maxInfoLine, fmt.Sprintf("XRef @S1@ is already defined at line %d", maxInfoLine)},
		{maxInfoLine + 1, fmt.Sprintf("XRef @S1@ is already defined at line %d", maxInfoLine+1)},
		{math.MaxInt, fmt.Sprintf("XRef @S1@ is already defined at line %d", math.MaxInt)},
		{0, "XRef @S1@ is defined by more than one record"},
		{-5, "XRef @S1@ is defined by more than one record"},
	} {
		sv := NewStreamingValidator(StreamingOptions{})
		sv.ValidateRecord(&gedcom.Record{XRef: "@S1@", Type: gedcom.RecordTypeSource, LineNumber: tc.line})
		issues := sv.ValidateRecord(&gedcom.Record{XRef: "@S1@", Type: gedcom.RecordTypeSource, LineNumber: 9})
		if len(issues) != 1 || issues[0].Message != tc.want {
			t.Errorf("first line %d: issues = %v, want one with message %q", tc.line, issues, tc.want)
		}
	}
}
