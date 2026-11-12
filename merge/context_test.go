package merge_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/cacack/gedcom-go/v3/internal/ctxcheck"
	"github.com/cacack/gedcom-go/v3/merge"
)

// TestCombineContextCancelledAtEveryCheck cancels Combine at each of its
// context checks in turn, for every collision strategy (#743). Every
// cancelled run returns a nil document and report and an error wrapping
// context.Canceled, and leaves both inputs unmodified; the first run the
// context outlasts matches an uncancelled Combine. Combine copies both
// documents' records (three each), so a per-record check is at least seven
// checks, not the one a start-only check would make.
func TestCombineContextCancelledAtEveryCheck(t *testing.T) {
	strategies := map[string]merge.CombineOptions{
		"ErrorOnCollision": {},
		"PrefixDoc2":       {CollisionStrategy: merge.PrefixDoc2, Prefix: "B_"},
		"RenumberDoc2":     {CollisionStrategy: merge.RenumberDoc2},
	}
	for name, opts := range strategies {
		t.Run(name, func(t *testing.T) {
			doc1 := buildDoc("A")
			doc2 := buildDoc("A")
			if name == "ErrorOnCollision" {
				doc2 = buildDoc("B")
			}
			before1, before2 := doc1.Clone(), doc2.Clone()
			want, wantReport, err := merge.Combine(doc1, doc2, opts)
			if err != nil {
				t.Fatalf("uncancelled Combine: %v", err)
			}

			checks := 0
			for ; ; checks++ {
				o := opts
				o.Context = ctxcheck.CancelAfter(int64(checks))
				got, report, err := merge.Combine(doc1, doc2, o)
				if err == nil {
					if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(report, wantReport) {
						t.Errorf("Combine outlasting %d checks differs from an uncancelled one", checks)
					}
					break
				}
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("cancelled at check %d: err = %v, want one wrapping context.Canceled", checks, err)
				}
				if got != nil || report != nil {
					t.Errorf("cancelled at check %d: got %v, %v; want nil document and report", checks, got, report)
				}
				if !reflect.DeepEqual(doc1, before1) || !reflect.DeepEqual(doc2, before2) {
					t.Fatalf("cancelled at check %d: an input document was modified", checks)
				}
				if checks > 1000 {
					t.Fatal("Combine never completes")
				}
			}
			if minChecks := 1 + 2*(1+3); checks < minChecks {
				t.Errorf("Combine checked the context %d times, want at least %d", checks, minChecks)
			}
		})
	}
}
