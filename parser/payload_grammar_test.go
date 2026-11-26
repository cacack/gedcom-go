package parser

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/internal/payloadgrammar"
)

// TestParseLineXRefMatchesGrammar holds the parser's reading of a
// cross-reference identifier to the GEDCOM 7.0 Xref rule (ADR 0013). The
// parser takes any field that starts and ends with "@" as the identifier,
// leaving its body to IsPointerXRef and the grammar check (SI-010).
func TestParseLineXRefMatchesGrammar(t *testing.T) {
	payloadgrammar.Compare(t, payloadgrammar.Xref, func(s string) bool {
		line, err := NewParser().ParseLine("0 " + s + " INDI")
		return err == nil && line.XRef == s
	}, func(s string, libAccepts bool) []string {
		if libAccepts && strings.HasPrefix(s, "@") && strings.HasSuffix(s, "@") {
			return []string{"SI-010"}
		}
		return nil
	})
}
