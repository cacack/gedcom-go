package validator

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/internal/payloadgrammar"
)

// payload_grammar_test.go holds the grammar check's 7.0 payload and
// identifier checks to the GEDCOM 7.0 ABNF (ADR 0013), as
// gedcom/payload_grammar_test.go does the parsers.

func grammarChecker70() *grammarChecker {
	return &grammarChecker{g: grammarFor(gedcom.Version70), v7: true}
}

func TestGrammarCheckDateExactMatchesGrammar(t *testing.T) {
	o := payloadgrammar.Load(t)
	payloadgrammar.Compare(t, payloadgrammar.DateExact, payloadAccepts(payloadDateExact),
		func(s string, libAccepts bool) []string {
			// The check reads no SCHMA, so it takes no extension month as
			// standing for a Gregorian one.
			if !libAccepts && hasExtensionToken(o, s) {
				return []string{"SI-018"}
			}
			return nil
		})
}

func TestGrammarCheckTimeMatchesGrammar(t *testing.T) {
	payloadgrammar.Compare(t, payloadgrammar.Time, payloadAccepts(payloadTime), noDepartures)
}

func TestGrammarCheckAgeMatchesGrammar(t *testing.T) {
	payloadgrammar.Compare(t, payloadgrammar.Age, payloadAccepts(payloadAge), noDepartures)
}

// TestGrammarCheckXRefsMatchGrammar holds the 7.0 record identifier check to
// Xref, and the pointer payload check to pointer, which adds @VOID@.
func TestGrammarCheckXRefsMatchGrammar(t *testing.T) {
	c := grammarChecker70()
	payloadgrammar.Compare(t, payloadgrammar.Xref, func(s string) bool {
		c.issues = nil
		c.checkRecordXRef(&gedcom.Record{Type: gedcom.RecordTypeIndividual, XRef: s})
		return len(c.issues) == 0
	}, noDepartures)
	o := payloadgrammar.Load(t)
	for _, s := range append(payloadgrammar.Inputs(t, payloadgrammar.Xref), "@VOID@") {
		if got, want := c.isPointer(s), o.Match("pointer", s); got != want {
			t.Errorf("isPointer(%q) = %v, the pointer rule says %v", s, got, want)
		}
	}
}

// payloadAccepts is the grammar check's 7.0 answer for one payload kind.
func payloadAccepts(kind payloadKind) func(string) bool {
	c := grammarChecker70()
	return func(s string) bool {
		_, ok := c.payloadValid(kind, s)
		return ok
	}
}

func noDepartures(string, bool) []string { return nil }

func hasExtensionToken(o *payloadgrammar.Oracle, s string) bool {
	for _, tok := range strings.Fields(s) {
		if o.Match("extTag", tok) {
			return true
		}
	}
	return false
}
