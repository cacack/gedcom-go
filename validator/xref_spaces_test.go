package validator

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
)

// spacedPointerDoc declares vers and points at two spaced identifiers: @A 1@,
// which exists, and @A 9@, which does not.
func spacedPointerDoc(vers string) string {
	return strings.Join([]string{
		"0 HEAD",
		"1 GEDC",
		"2 VERS " + vers,
		"1 CHAR UTF-8",
		"0 @I1@ INDI",
		"1 NAME John /Smith/",
		"1 ASSO @A 1@",
		"1 ASSO @A 9@",
		"0 @A 1@ INDI",
		"1 NAME Ann /Smith/",
		"0 TRLR",
	}, "\n") + "\n"
}

// TestBrokenXRefSpacedPointer pins issue #591: in a 5.5/5.5.1 document a
// spaced value is a pointer, so a dangling one is reported and a resolving
// one is not.
func TestBrokenXRefSpacedPointer(t *testing.T) {
	for _, vers := range []string{"5.5", "5.5.1"} {
		t.Run(vers, func(t *testing.T) {
			doc, err := decoder.Decode(strings.NewReader(spacedPointerDoc(vers)))
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			got := structureIssues(t, New().Validate(doc))
			if len(got) != 1 {
				t.Fatalf("structure issues = %v, want exactly one BROKEN_XREF", got)
			}
			if got[0].Code != CodeBrokenXRef || got[0].RelatedXRef != "@A 9@" || got[0].LineNumber != 8 {
				t.Errorf("issue = %v, want BROKEN_XREF for @A 9@ on line 8", got[0])
			}
		})
	}
}

// TestBrokenXRefSpacedValue70 pins that GEDCOM 7.0, whose Xref grammar has no
// space, still treats a spaced value as text rather than a broken pointer.
func TestBrokenXRefSpacedValue70(t *testing.T) {
	result, err := decoder.DecodeWithDiagnostics(strings.NewReader(spacedPointerDoc("7.0")), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	if got := structureIssues(t, New().Validate(result.Document)); len(got) != 0 {
		t.Errorf("structure issues = %v, want none", got)
	}
}
