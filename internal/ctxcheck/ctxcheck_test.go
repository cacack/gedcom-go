package ctxcheck

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

func TestErr(t *testing.T) {
	if err := Err(nil); err != nil { //nolint:staticcheck // a nil Context is the case under test
		t.Errorf("Err(nil) = %v, want nil", err)
	}
	if err := Err(context.Background()); err != nil {
		t.Errorf("Err(Background) = %v, want nil", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Err(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("Err(cancelled) = %v, want context.Canceled", err)
	}
}

func TestCancelAfter(t *testing.T) {
	ctx := CancelAfter(2)
	for i := range 2 {
		if err := Err(ctx); err != nil {
			t.Fatalf("call %d: Err = %v, want nil", i+1, err)
		}
	}
	if err := Err(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("call 3: Err = %v, want context.Canceled", err)
	}
}

func TestClone(t *testing.T) {
	doc := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version551, SourceSystem: "X"},
		Trailer: &gedcom.Trailer{},
		Vendor:  gedcom.VendorRootsMagic,
		Records: []*gedcom.Record{
			{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Tags: []*gedcom.Tag{{Level: 1, Tag: "SEX", Value: "M"}}},
			nil,
			{Type: "_ROOT"},
		},
	}
	doc.XRefMap = map[string]*gedcom.Record{"@I1@": doc.Records[0]}

	got, err := Clone(context.Background(), doc)
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if want := doc.Clone(); !reflect.DeepEqual(got, want) {
		t.Errorf("Clone = %+v, want Document.Clone's %+v", got, want)
	}
	if got.Records[0] == doc.Records[0] || got.XRefMap["@I1@"] != got.Records[0] {
		t.Error("Clone shares records with its input or maps @I1@ to the wrong record")
	}

	if got, err := Clone(CancelAfter(2), doc); !errors.Is(err, context.Canceled) || got != nil {
		t.Errorf("Clone cancelled at the second record = %v, %v; want nil, context.Canceled", got, err)
	}
	if got, err := Clone(context.Background(), nil); got != nil || err != nil {
		t.Errorf("Clone(nil) = %v, %v; want nil, nil", got, err)
	}
}
