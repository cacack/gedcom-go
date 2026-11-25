package gedcom

import (
	"errors"
	"reflect"
	"testing"
)

func TestDocument_ReplaceHeader(t *testing.T) {
	d := editDocument()
	d.Vendor = VendorAncestry
	d.Schema = &SchemaDefinition{TagMappings: map[string]string{"_OLD": "https://example.com/old"}}
	records := append([]*Record(nil), d.Records...)

	h := &Header{
		Version:      Version70,
		SourceSystem: "RootsMagic",
		Tags: []*Tag{
			{Level: 1, Tag: "SOUR", Value: "RootsMagic"},
			{Level: 1, Tag: "SCHMA"},
			{Level: 2, Tag: "TAG", Value: "_NEW https://example.com/new"},
		},
	}
	if err := d.ReplaceHeader(h); err != nil {
		t.Fatal(err)
	}
	if d.Header != h {
		t.Error("Header is not the new header")
	}
	if d.Vendor != VendorRootsMagic {
		t.Errorf("Vendor = %q, want %q", d.Vendor, VendorRootsMagic)
	}
	want := map[string]string{"_NEW": "https://example.com/new"}
	if d.Schema == nil || !reflect.DeepEqual(d.Schema.TagMappings, want) {
		t.Errorf("Schema = %+v, want %v", d.Schema, want)
	}
	if !reflect.DeepEqual(d.Records, records) {
		t.Error("records changed")
	}

	// A header with no SCHMA, or not 7.0, leaves no schema behind.
	if err := d.ReplaceHeader(&Header{Version: Version551, Tags: h.Tags}); err != nil {
		t.Fatal(err)
	}
	if d.Schema != nil || d.Vendor != VendorUnknown {
		t.Errorf("Schema = %+v, Vendor = %q; want nil and unknown", d.Schema, d.Vendor)
	}
}

func TestDocument_ReplaceHeader_NilArguments(t *testing.T) {
	var nilDoc *Document
	if err := nilDoc.ReplaceHeader(&Header{}); !errors.Is(err, ErrNilDocument) {
		t.Errorf("nil document: error = %v, want ErrNilDocument", err)
	}
	d := editDocument()
	old := d.Header
	if err := d.ReplaceHeader(nil); !errors.Is(err, ErrNilHeader) {
		t.Errorf("nil header: error = %v, want ErrNilHeader", err)
	}
	if d.Header != old {
		t.Error("nil header changed the document")
	}
}

func TestHeaderSchema(t *testing.T) {
	tests := []struct {
		name string
		h    *Header
		want map[string]string // nil: no schema
	}{
		{name: "nil header"},
		{name: "not 7.0", h: &Header{Version: Version551, Tags: []*Tag{
			{Level: 1, Tag: "SCHMA"},
			{Level: 2, Tag: "TAG", Value: "_A https://example.com/a"},
		}}},
		{name: "no SCHMA", h: &Header{Version: Version70, Tags: []*Tag{
			{Level: 1, Tag: "SOUR", Value: "X"},
			{Level: 2, Tag: "TAG", Value: "_A https://example.com/a"},
		}}},
		{name: "TAG lines", h: &Header{Version: Version70, Tags: []*Tag{
			nil,
			{Level: 1, Tag: "SCHMA"},
			{Level: 2, Tag: "TAG", Value: "_A  https://example.com/a "},
			{Level: 2, Tag: "TAG", Value: "_NOURI"},
			{Level: 3, Tag: "TAG", Value: "_DEEP https://example.com/deep"},
			{Level: 1, Tag: "NOTE", Value: "after"},
			{Level: 2, Tag: "TAG", Value: "_OUTSIDE https://example.com/outside"},
		}}, want: map[string]string{"_A": "https://example.com/a"}},
		{name: "later SCHMA replaces", h: &Header{Version: Version70, Tags: []*Tag{
			{Level: 1, Tag: "SCHMA"},
			{Level: 2, Tag: "TAG", Value: "_A https://example.com/a"},
			{Level: 1, Tag: "SCHMA"},
			{Level: 2, Tag: "TAG", Value: "_B https://example.com/b"},
		}}, want: map[string]string{"_B": "https://example.com/b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := headerSchema(tt.h)
			if tt.want == nil {
				if got != nil {
					t.Errorf("schema = %+v, want nil", got)
				}
				return
			}
			if got == nil || !reflect.DeepEqual(got.TagMappings, tt.want) {
				t.Errorf("schema = %+v, want %v", got, tt.want)
			}
		})
	}
}
