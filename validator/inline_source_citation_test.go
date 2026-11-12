package validator

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
)

// TestInlineSourceCitationNotOrphaned: a 5.5.1 citation without a source
// record ("1 SOUR Family bible") is not a reference, so it is not reported as
// one to a non-existent source (issue #738).
func TestInlineSourceCitationNotOrphaned(t *testing.T) {
	const input = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @I1@ INDI\n1 SOUR Family bible\n1 SOUR @S9@\n0 TRLR\n"
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	var orphaned []string
	for _, issue := range New().ValidateAll(doc) {
		if issue.Code == CodeOrphanedSOUR {
			orphaned = append(orphaned, issue.RelatedXRef)
		}
	}
	if len(orphaned) != 1 || orphaned[0] != "@S9@" {
		t.Errorf("orphaned SOUR references = %q, want only @S9@", orphaned)
	}
}
