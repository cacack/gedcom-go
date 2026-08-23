package converter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/encoder"
	"github.com/cacack/gedcom-go/v2/gedcom"
)

// The encoder writes a decoded document's header from Header.Tags, so setting
// the typed Header.Version alone is no longer enough: a conversion that left
// the raw GEDC.VERS tag alone would produce a document declaring the version it
// came from. These tests guard that coupling from the converter's side, where
// the transformation is owned.

func TestConvertUpdatesVersionTag(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version:  gedcom.Version55,
			Encoding: gedcom.EncodingANSEL,
			Tags: []*gedcom.Tag{
				{Level: 1, Tag: "GEDC"},
				{Level: 2, Tag: "VERS", Value: "5.5"},
				{Level: 1, Tag: "SOUR", Value: "TestSystem"},
				{Level: 2, Tag: "VERS", Value: "1.2.3"},
			},
		},
		XRefMap: make(map[string]*gedcom.Record),
	}

	converted, _, err := ConvertWithOptions(doc, gedcom.Version551, DefaultOptions())
	if err != nil {
		t.Fatalf("convert: %v", err)
	}

	var buf bytes.Buffer
	if err := encoder.Encode(&buf, converted); err != nil {
		t.Fatalf("encode: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "2 VERS 5.5.1") {
		t.Errorf("converted document does not declare the target version:\n%s", out)
	}
	if strings.Contains(out, "2 VERS 5.5\n") {
		t.Errorf("converted document still declares the source version:\n%s", out)
	}
	if !strings.Contains(out, "2 VERS 1.2.3") {
		t.Errorf("the source system's own version was rewritten:\n%s", out)
	}
}

func TestConvertAddsVersionTagWhenHeaderHasNoGEDC(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version: gedcom.Version55,
			Tags: []*gedcom.Tag{
				{Level: 1, Tag: "SOUR", Value: "PAF 2.2"},
			},
		},
		XRefMap: make(map[string]*gedcom.Record),
	}

	converted, _, err := ConvertWithOptions(doc, gedcom.Version551, DefaultOptions())
	if err != nil {
		t.Fatalf("convert: %v", err)
	}

	var buf bytes.Buffer
	if err := encoder.Encode(&buf, converted); err != nil {
		t.Fatalf("encode: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "1 GEDC") || !strings.Contains(out, "2 VERS 5.5.1") {
		t.Errorf("conversion left the target version undeclared:\n%s", out)
	}
}

func TestConvertDoesNotMutateSourceHeaderTags(t *testing.T) {
	source := &gedcom.Document{
		Header: &gedcom.Header{
			Version: gedcom.Version55,
			Tags: []*gedcom.Tag{
				{Level: 1, Tag: "GEDC"},
				{Level: 2, Tag: "VERS", Value: "5.5"},
			},
		},
		XRefMap: make(map[string]*gedcom.Record),
	}

	if _, _, err := ConvertWithOptions(source, gedcom.Version551, DefaultOptions()); err != nil {
		t.Fatalf("convert: %v", err)
	}

	if got := source.Header.Tags[1].Value; got != "5.5" {
		t.Errorf("conversion mutated the caller's header tags: GEDC.VERS = %q, want 5.5", got)
	}
}
