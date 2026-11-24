package merge_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
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
		"ZipSharedXRefs":   {CollisionStrategy: merge.ZipSharedXRefs},
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

// TestCombineDanglingPassChecksContext shows the pass that collects doc2's
// dangling pointers checks the context before each record (#944). Under
// PrefixDoc2 and RenumberDoc2 a doc2 record is visited by four passes: the
// collision scan, the dangling-pointer pass, the remap and the copy. So each
// record added to doc2 adds at least four checks, and a run cancelled at any
// of them returns an error wrapping context.Canceled.
func TestCombineDanglingPassChecksContext(t *testing.T) {
	strategies := map[string]merge.CombineOptions{
		"PrefixDoc2":   {CollisionStrategy: merge.PrefixDoc2, Prefix: "B_"},
		"RenumberDoc2": {CollisionStrategy: merge.RenumberDoc2},
	}
	// checksToComplete counts the context checks a Combine with extra
	// more records in doc2 makes.
	checksToComplete := func(t *testing.T, opts merge.CombineOptions, extra int) int {
		t.Helper()
		doc1, doc2 := buildDoc("A"), buildDoc("A")
		for i := range extra {
			xref := fmt.Sprintf("@N%d@", i+1)
			rec := &gedcom.Record{XRef: xref, Type: gedcom.RecordTypeNote, Tags: []*gedcom.Tag{{Level: 0, Tag: "NOTE", XRef: xref}}}
			doc2.Records = append(doc2.Records, rec)
			doc2.XRefMap[xref] = rec
		}
		for checks := 0; checks <= 1000; checks++ {
			opts.Context = ctxcheck.CancelAfter(int64(checks))
			got, _, err := merge.Combine(doc1, doc2, opts)
			if err == nil {
				return checks
			}
			if !errors.Is(err, context.Canceled) || got != nil {
				t.Fatalf("cancelled at check %d: got %v, %v; want nil and an error wrapping context.Canceled", checks, got, err)
			}
		}
		t.Fatal("Combine never completes")
		return 0
	}
	for name, opts := range strategies {
		t.Run(name, func(t *testing.T) {
			const extra = 5
			perRecord := (checksToComplete(t, opts, extra) - checksToComplete(t, opts, 0)) / extra
			if perRecord < 4 {
				t.Errorf("each doc2 record adds %d context checks, want at least 4", perRecord)
			}
		})
	}
}

// TestRemapXRefsContext pins the cancellation of RemapXRefsContext (#944):
// an already-cancelled context returns a nil document and mapping and an
// error wrapping context.Canceled, leaving the input unmodified; a nil or
// live context gives what RemapXRefs gives.
func TestRemapXRefsContext(t *testing.T) {
	transform := func(old string) string { return "@B_" + old[1:] }
	doc := buildDoc("A")
	before := doc.Clone()
	want, wantMapping, err := merge.RemapXRefs(doc, transform)
	if err != nil {
		t.Fatalf("RemapXRefs: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got, mapping, err := merge.RemapXRefsContext(ctx, doc, transform)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled: err = %v, want one wrapping context.Canceled", err)
	}
	if got != nil || mapping != nil {
		t.Errorf("cancelled: got document %v and mapping %v, want nil", got, mapping)
	}
	if !reflect.DeepEqual(doc, before) {
		t.Error("cancelled: input document was modified")
	}

	for name, live := range map[string]context.Context{"nil": nil, "background": context.Background()} {
		got, mapping, err := merge.RemapXRefsContext(live, doc, transform)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(mapping, wantMapping) {
			t.Errorf("%s: result differs from RemapXRefs", name)
		}
	}
}
