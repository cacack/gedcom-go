package converter

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/internal/ctxcheck"
)

const contextTestInput = `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 1 JAN 1900
0 @I2@ INDI
1 NAME Jane /Smith/
0 TRLR
`

// TestConvertContextCancelledAtEveryCheck cancels a 5.5.1 -> 7.0 conversion
// at each of its context checks in turn (#743). Every cancelled run returns
// no document and an error wrapping context.Canceled, and leaves the input
// unmodified; the first run the context outlasts matches an uncancelled
// conversion. A conversion that checked only before and after its work
// would have two such checks, not one per record copied and per step.
func TestConvertContextCancelledAtEveryCheck(t *testing.T) {
	doc, err := decoder.Decode(strings.NewReader(contextTestInput))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	before := doc.Clone()
	want, wantReport, err := ConvertWithOptions(doc, gedcom.Version70, DefaultOptions())
	if err != nil {
		t.Fatalf("uncancelled conversion: %v", err)
	}

	checks := 0
	for ; ; checks++ {
		opts := DefaultOptions()
		opts.Context = ctxcheck.CancelAfter(int64(checks))
		got, report, err := ConvertWithOptions(doc, gedcom.Version70, opts)
		if err == nil {
			if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(report, wantReport) {
				t.Errorf("conversion outlasting %d checks differs from an uncancelled one", checks)
			}
			break
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled at check %d: err = %v, want one wrapping context.Canceled", checks, err)
		}
		if got != nil {
			t.Errorf("cancelled at check %d: got a document, want nil", checks)
		}
		if report != nil && report.Success {
			t.Errorf("cancelled at check %d: report.Success = true", checks)
		}
		if !reflect.DeepEqual(doc, before) {
			t.Fatalf("cancelled at check %d: input document modified", checks)
		}
		if checks > 1000 {
			t.Fatal("conversion never completes")
		}
	}
	// Start (1), the copy (itself and its two records: 3), each of the 12
	// 5.5.1 -> 7.0 steps, and the check after validation (1), at the least.
	if minChecks := 1 + 3 + 12 + 1; checks < minChecks {
		t.Errorf("conversion checked the context %d times, want at least %d", checks, minChecks)
	}
	t.Logf("conversion checked the context %d times", checks)
}

// TestConvertSameVersionContextCancelledWhileCopying pins that the 7.0 -> 7.0
// path, which deep-copies the document to rename XRefs or add a missing VERS,
// checks the context while copying as ConvertOptions.Context documents (#743):
// cancelled after the start check it returns no document and an error
// wrapping context.Canceled, and leaves the input unmodified.
func TestConvertSameVersionContextCancelledWhileCopying(t *testing.T) {
	inputs := map[string]string{
		"xref rename":  "0 HEAD\n1 GEDC\n2 VERS 7.0\n0 @I.1@ INDI\n1 NAME A /B/\n0 TRLR\n",
		"missing VERS": "0 HEAD\n1 GEDC\n0 @I1@ INDI\n1 NAME A /B/\n0 TRLR\n",
	}
	for name, input := range inputs {
		t.Run(name, func(t *testing.T) {
			doc, err := decoder.Decode(strings.NewReader(input))
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			doc.Header.Version = gedcom.Version70
			want, _, err := ConvertWithOptions(doc, gedcom.Version70, DefaultOptions())
			if err != nil {
				t.Fatalf("uncancelled conversion: %v", err)
			}
			if want == doc {
				t.Fatal("input needs no copy; the test would not reach the copy")
			}
			before := doc.Clone()

			opts := DefaultOptions()
			opts.Context = ctxcheck.CancelAfter(1)
			got, _, err := ConvertWithOptions(doc, gedcom.Version70, opts)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("err = %v, want one wrapping context.Canceled", err)
			}
			if got != nil {
				t.Error("got a document, want nil")
			}
			if !reflect.DeepEqual(doc, before) {
				t.Error("input document modified")
			}
		})
	}
}
