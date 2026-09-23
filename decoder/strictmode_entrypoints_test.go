package decoder

import (
	"errors"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
	"github.com/cacack/gedcom-go/v2/parser"
)

// strictModeFixture carries both kinds of problem StrictMode governs: a level
// jump on line 7 (clamped by normalizeLevelJumps in lenient mode, accepted as-is
// in strict mode) and a malformed level on line 8 (skipped in lenient mode,
// fatal in strict mode). @I2@ follows the malformed line, so its presence shows
// that decoding continued past it.
const strictModeFixture = "0 HEAD\n" +
	"1 GEDC\n" +
	"2 VERS 5.5.1\n" +
	"0 @I1@ INDI\n" +
	"1 NAME John /Doe/\n" +
	"1 BIRT\n" +
	"3 DATE 1 JAN 1900\n" +
	"X BAD LINE\n" +
	"0 @I2@ INDI\n" +
	"1 NAME Jane /Doe/\n" +
	"0 TRLR\n"

const strictModeFixtureBadLine = 8

// TestStrictMode_AllEntryPoints asserts that StrictMode means the same thing for
// every decode entry point (#490): the entry points differ only in what they
// return, never in how they treat malformed input. Decode takes no options and
// always uses the lenient default.
func TestStrictMode_AllEntryPoints(t *testing.T) {
	type decodeFunc func(opts *DecodeOptions) (*gedcom.Document, Diagnostics, error)

	entryPoints := []struct {
		name string
		// hasDiagnostics is true when the entry point returns diagnostics.
		hasDiagnostics bool
		// lenientOnly is true when the entry point cannot be made strict.
		lenientOnly bool
		decode      decodeFunc
	}{
		{
			name:        "Decode",
			lenientOnly: true,
			decode: func(_ *DecodeOptions) (*gedcom.Document, Diagnostics, error) {
				doc, err := Decode(strings.NewReader(strictModeFixture))
				return doc, nil, err
			},
		},
		{
			name: "DecodeWithOptions",
			decode: func(opts *DecodeOptions) (*gedcom.Document, Diagnostics, error) {
				doc, err := DecodeWithOptions(strings.NewReader(strictModeFixture), opts)
				return doc, nil, err
			},
		},
		{
			name:           "DecodeWithDiagnostics",
			hasDiagnostics: true,
			decode: func(opts *DecodeOptions) (*gedcom.Document, Diagnostics, error) {
				res, err := DecodeWithDiagnostics(strings.NewReader(strictModeFixture), opts)
				if res == nil {
					return nil, nil, err
				}
				return res.Document, res.Diagnostics, err
			},
		},
	}

	for _, ep := range entryPoints {
		for _, strict := range []bool{true, false} {
			if strict && ep.lenientOnly {
				continue
			}
			name := ep.name + "/lenient"
			if strict {
				name = ep.name + "/strict"
			}
			t.Run(name, func(t *testing.T) {
				doc, diags, err := ep.decode(&DecodeOptions{StrictMode: strict})
				if strict {
					assertStrictFailure(t, doc, err)
					return
				}
				assertLenientRecovery(t, doc, err)
				if ep.hasDiagnostics {
					assertHasCode(t, diags, CodeBadLevelJump)
					assertHasCode(t, diags, CodeInvalidLevel)
				}
			})
		}
	}
}

// TestLenientDecode_NoValidLines asserts that a lenient decode which recovers
// nothing still returns the structured *parser.ParseError (ADR 0007), wrapped,
// alongside an empty document.
func TestLenientDecode_NoValidLines(t *testing.T) {
	doc, err := Decode(strings.NewReader("This is not GEDCOM at all!"))
	if err == nil {
		t.Fatal("expected an error when no line parses")
	}
	var parseErr *parser.ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("error = %T (%v), want it to wrap *parser.ParseError", err, err)
	}
	if parseErr.Line != 1 {
		t.Errorf("ParseError.Line = %d, want 1", parseErr.Line)
	}
	if doc == nil || len(doc.Records) != 0 {
		t.Errorf("document = %v, want an empty non-nil document", doc)
	}
}

// TestDecode_EmptyInputVersion asserts the empty-document branch reports the
// same default version as any other decode with no version evidence.
func TestDecode_EmptyInputVersion(t *testing.T) {
	doc, err := Decode(strings.NewReader(""))
	if err != nil {
		t.Fatalf("Decode(\"\") error = %v", err)
	}
	if doc.Header.Version != gedcom.Version55 {
		t.Errorf("Header.Version = %q, want %q", doc.Header.Version, gedcom.Version55)
	}
}

func assertStrictFailure(t *testing.T, doc *gedcom.Document, err error) {
	t.Helper()
	if doc != nil {
		t.Errorf("strict: document = %v, want nil", doc)
	}
	var parseErr *parser.ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("strict: error = %T (%v), want *parser.ParseError", err, err)
	}
	if parseErr.Line != strictModeFixtureBadLine {
		t.Errorf("strict: ParseError.Line = %d, want %d", parseErr.Line, strictModeFixtureBadLine)
	}
}

func assertLenientRecovery(t *testing.T, doc *gedcom.Document, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("lenient: error = %v, want nil", err)
	}
	if doc == nil {
		t.Fatal("lenient: document is nil")
	}
	if doc.GetIndividual("@I2@") == nil {
		t.Error("lenient: @I2@ after the malformed line is missing")
	}
	rec := doc.XRefMap["@I1@"]
	if rec == nil {
		t.Fatal("lenient: @I1@ is missing")
	}
	for _, tag := range rec.Tags {
		if tag.Tag == "DATE" {
			if tag.Level != 2 {
				t.Errorf("lenient: jumped DATE level = %d, want 2 (clamped)", tag.Level)
			}
			return
		}
	}
	t.Error("lenient: DATE under BIRT is missing")
}

func assertHasCode(t *testing.T, diags Diagnostics, code string) {
	t.Helper()
	for _, d := range diags {
		if d.Code == code {
			return
		}
	}
	t.Errorf("diagnostics lack %s: %v", code, diags)
}
