package merge

import (
	"context"
	"errors"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/internal/ctxcheck"
)

// TestDanglingPointersContext pins the cancellation of the pass Combine and
// RemapXRefs share (#944): with a context already cancelled, or cancelled
// partway through the records, it returns an error wrapping context.Canceled
// and no pointers; with a nil or live context it completes.
func TestDanglingPointersContext(t *testing.T) {
	doc := &gedcom.Document{Records: []*gedcom.Record{
		{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Tags: []*gedcom.Tag{{Level: 1, Tag: "FAMS", Value: "@F9@"}}},
		{XRef: "@I2@", Type: gedcom.RecordTypeIndividual, Tags: []*gedcom.Tag{{Level: 1, Tag: "FAMC", Value: "@F8@"}}},
	}}

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	for name, ctx := range map[string]context.Context{
		"already cancelled":       cancelledCtx,
		"cancelled at record two": ctxcheck.CancelAfter(1),
	} {
		got, err := danglingPointers(ctx, doc)
		if !errors.Is(err, context.Canceled) || got != nil {
			t.Errorf("%s: got %v, %v; want nil and an error wrapping context.Canceled", name, got, err)
		}
	}

	for name, ctx := range map[string]context.Context{"nil": nil, "background": context.Background()} {
		got, err := danglingPointers(ctx, doc)
		if err != nil || len(got) != 2 || got[0] != "@F8@" || got[1] != "@F9@" {
			t.Errorf("%s context: got %v, %v; want [@F8@ @F9@], nil", name, got, err)
		}
	}
}
