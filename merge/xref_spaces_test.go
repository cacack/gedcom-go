package merge_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/merge"
)

// spacedXRefDoc declares vers and links an individual to a note through a
// spaced identifier, which GEDCOM 5.5 and 5.5.1 allow and 7.0 does not.
func spacedXRefDoc(vers string) string {
	return strings.Join([]string{
		"0 HEAD",
		"1 GEDC",
		"2 VERS " + vers,
		"1 CHAR UTF-8",
		"0 @I1@ INDI",
		"1 NAME John /Smith/",
		"1 NOTE @N 1@",
		"0 @N 1@ NOTE Shared note",
		"0 TRLR",
	}, "\n") + "\n"
}

func prefixA(old string) string { return "@A_" + old[1:] }

// TestRemapXRefs_SpacedXRef551 pins issue #591: a spaced identifier is valid
// in a 5.5/5.5.1 document, so remapping it succeeds and its pointer follows.
func TestRemapXRefs_SpacedXRef551(t *testing.T) {
	for _, vers := range []string{"5.5", "5.5.1"} {
		t.Run(vers, func(t *testing.T) {
			doc, err := decoder.Decode(strings.NewReader(spacedXRefDoc(vers)))
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			out, mapping, err := merge.RemapXRefs(doc, prefixA)
			if err != nil {
				t.Fatalf("RemapXRefs: %v", err)
			}
			if mapping["@N 1@"] != "@A_N 1@" {
				t.Errorf("mapping[@N 1@] = %q, want @A_N 1@", mapping["@N 1@"])
			}
			person := out.GetIndividual("@A_I1@")
			if person == nil {
				t.Fatal("individual @A_I1@ missing")
			}
			if len(person.NoteXRefs) != 1 || out.GetNote(person.NoteXRefs[0]) == nil {
				t.Errorf("NoteXRefs = %q, want one pointer resolving to a note", person.NoteXRefs)
			}
		})
	}
}

// TestRemapXRefs_SpacedXRef70 pins that GEDCOM 7.0 still rejects a spaced
// identifier as malformed.
func TestRemapXRefs_SpacedXRef70(t *testing.T) {
	result, err := decoder.DecodeWithDiagnostics(strings.NewReader(spacedXRefDoc("7.0")), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics: %v", err)
	}
	_, _, err = merge.RemapXRefs(result.Document, prefixA)
	var re *merge.RemapError
	if !errors.As(err, &re) || re.Old != "@N 1@" {
		t.Errorf("RemapXRefs error = %v, want *RemapError for @N 1@", err)
	}
}

// TestRemapXRefs_SpacedXRefUndeclared pins that a document built in code,
// which declares no version, keeps the space-free grammar.
func TestRemapXRefs_SpacedXRefUndeclared(t *testing.T) {
	rec := &gedcom.Record{XRef: "@N 1@", Type: gedcom.RecordTypeNote, Entity: &gedcom.Note{XRef: "@N 1@"}}
	doc := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version551},
		Records: []*gedcom.Record{rec},
		XRefMap: map[string]*gedcom.Record{rec.XRef: rec},
	}
	if _, _, err := merge.RemapXRefs(doc, prefixA); !errors.Is(err, merge.ErrInvalidRemap) {
		t.Errorf("RemapXRefs error = %v, want ErrInvalidRemap", err)
	}
}
