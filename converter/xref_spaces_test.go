package converter_test

import (
	"os"
	"testing"

	"github.com/cacack/gedcom-go/v3/converter"
	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestConvertSpacedXRefTo70 pins issue #579: uppercasing a mixed-case spaced
// 5.5.1 identifier for GEDCOM 7.0 must rewrite its pointers too, not leave
// them dangling at the old identifier.
func TestConvertSpacedXRefTo70(t *testing.T) {
	f, err := os.Open("../testdata/edge-cases/xref-case.ged")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	opts := decoder.DefaultOptions()
	opts.StrictMode = true
	doc, err := decoder.DecodeWithOptions(f, opts)
	if err != nil {
		t.Fatalf("strict decode: %v", err)
	}

	got, _, err := converter.Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}

	subm := got.GetSubmitter("@TEST@")
	if subm == nil {
		t.Fatal("submitter @TEST@ missing after conversion")
	}
	if len(subm.NoteXRefs) != 2 {
		t.Fatalf("NoteXRefs = %q, want 2 entries", subm.NoteXRefs)
	}
	for _, ref := range subm.NoteXRefs {
		if got.GetRecord(ref) == nil {
			t.Errorf("NoteXRefs entry %q does not resolve", ref)
		}
	}
	for _, tag := range got.GetRecord("@TEST@").Tags {
		if tag.Tag == "NOTE" && got.GetRecord(tag.Value) == nil {
			t.Errorf("raw NOTE tag value %q does not resolve", tag.Value)
		}
	}
}
