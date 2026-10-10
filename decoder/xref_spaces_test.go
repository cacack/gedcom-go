package decoder

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/parser"
)

// spacedXRefDoc is the issue #579 reproduction: a document whose identifiers
// contain spaces, declaring the given GEDCOM version.
func spacedXRefDoc(vers string) string {
	return strings.Join([]string{
		"0 HEAD",
		"1 SOUR EXAMPLE",
		"1 SUBM @U1@",
		"1 GEDC",
		"2 VERS " + vers,
		"2 FORM LINEAGE-LINKED",
		"1 CHAR UTF-8",
		"0 @I 1@ INDI",
		"1 NAME Alex /Example/",
		"1 NOTE @N 1@",
		"1 NOTE @N 1@ is not a pointer",
		"0 @N 1@ NOTE Shared note",
		"0 @U1@ SUBM",
		"1 NAME Example",
		"0 TRLR",
	}, "\n") + "\n"
}

// TestSpacedXRef551 pins that GEDCOM 5.5 and 5.5.1, whose grammar lists the
// space among pointer characters, accept spaced identifiers as definitions
// and as pointers, in strict and lenient mode alike (#579).
func TestSpacedXRef551(t *testing.T) {
	for _, vers := range []string{"5.5", "5.5.1"} {
		for _, strict := range []bool{true, false} {
			name := vers + " lenient"
			if strict {
				name = vers + " strict"
			}
			t.Run(name, func(t *testing.T) {
				result, err := DecodeWithDiagnostics(strings.NewReader(spacedXRefDoc(vers)),
					&DecodeOptions{StrictMode: strict})
				if err != nil {
					t.Fatalf("decode: %v", err)
				}
				for _, d := range result.Diagnostics {
					if d.Code == CodeInvalidXRef {
						t.Errorf("unexpected diagnostic: %v", d)
					}
				}
				doc := result.Document
				person := doc.GetIndividual("@I 1@")
				if person == nil {
					t.Fatal("individual @I 1@ not found")
				}
				if want := []string{"@N 1@"}; !slices.Equal(person.NoteXRefs, want) {
					t.Errorf("NoteXRefs = %q, want %q", person.NoteXRefs, want)
				}
				// A value that merely starts with a pointer is text.
				if want := []string{"@N 1@ is not a pointer"}; !slices.Equal(person.InlineNotes, want) {
					t.Errorf("InlineNotes = %q, want %q", person.InlineNotes, want)
				}
				if doc.GetNote("@N 1@") == nil {
					t.Error("note @N 1@ not found")
				}
			})
		}
	}
}

// TestSpacedXRef70 pins that GEDCOM 7.0, whose Xref grammar has no space,
// keeps rejecting spaced identifiers and treating spaced values as text.
func TestSpacedXRef70(t *testing.T) {
	data := spacedXRefDoc("7.0")

	_, err := DecodeWithDiagnostics(strings.NewReader(data), &DecodeOptions{StrictMode: true})
	if !errors.Is(err, parser.ErrXRefContainsSpace) {
		t.Errorf("strict decode error = %v, want %v", err, parser.ErrXRefContainsSpace)
	}

	result, err := DecodeWithDiagnostics(strings.NewReader(data), nil)
	if err != nil {
		t.Fatalf("lenient decode: %v", err)
	}
	var lines []int
	for _, d := range result.Diagnostics {
		if d.Code == CodeInvalidXRef {
			lines = append(lines, d.Line)
		}
	}
	if want := []int{8, 12}; !slices.Equal(lines, want) {
		t.Errorf("INVALID_XREF lines = %v, want %v", lines, want)
	}
	person := result.Document.GetIndividual("@I 1@")
	if person == nil {
		t.Fatal("individual @I 1@ not recovered")
	}
	if len(person.NoteXRefs) != 0 {
		t.Errorf("NoteXRefs = %q, want none", person.NoteXRefs)
	}
	if want := []string{"@N 1@", "@N 1@ is not a pointer"}; !slices.Equal(person.InlineNotes, want) {
		t.Errorf("InlineNotes = %q, want %q", person.InlineNotes, want)
	}
}

// TestSpacedXRef551StrictOtherErrors pins that accepting spaced identifiers
// does not hide a different syntax error from strict mode.
func TestSpacedXRef551StrictOtherErrors(t *testing.T) {
	data := strings.Replace(spacedXRefDoc("5.5.1"), "1 NAME Example\n", "1 NAME Example\nbad line\n", 1)
	_, err := DecodeWithDiagnostics(strings.NewReader(data), &DecodeOptions{StrictMode: true})
	var pe *parser.ParseError
	if !errors.As(err, &pe) || pe.Line != 15 {
		t.Errorf("strict decode error = %v, want a parse error on line 15", err)
	}
}
