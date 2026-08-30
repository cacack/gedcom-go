package validator

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/decoder"
)

// TestIssueLineNumberIsPopulated covers the LineNumber field added in #509.
// ADR 0008 declared it from the start; the shipped struct had the ADR's field
// list minus that one, so the line number was stuffed into Details as a string
// under a key only 2 of 23 codes ever set.
func TestIssueLineNumberIsPopulated(t *testing.T) {
	// A custom tag the registry does not know, on a known line.
	const input = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n1 SUBM @U1@\n" +
		"0 @U1@ SUBM\n1 NAME Test\n" +
		"0 @I1@ INDI\n1 NAME John /Doe/\n1 _WEIRD custom\n0 TRLR\n"

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// Strictness is named explicitly: its zero value is Relaxed here, which
	// reports errors only and would drop the warning this fixture depends on.
	// v3 renumbers the constants so the zero value is Normal.
	v := NewWithOptions(&ValidateOptions{
		TagRegistry:        NewTagRegistry(),
		ValidateCustomTags: true,
		Strictness:         StrictnessNormal,
	})

	var found bool
	for _, issue := range v.ValidateAll(doc) {
		if issue.Code != CodeUnknownCustomTag {
			continue
		}
		found = true
		if issue.LineNumber == 0 {
			t.Errorf("%s carries no LineNumber", issue.Code)
		}
		// The field must agree with the legacy key while both exist.
		if got := issue.Details["line_number"]; got != "" {
			if want := itoa(issue.LineNumber); got != want {
				t.Errorf("Details[line_number] = %q, LineNumber = %s; they disagree", got, want)
			}
		}
	}
	if !found {
		t.Fatal("no UNKNOWN_CUSTOM_TAG issue produced; the fixture would prove nothing")
	}
}

// TestIssueWithLineNumber covers the builder in isolation, including that it
// preserves the value-receiver immutability the other With* helpers promise.
func TestIssueWithLineNumber(t *testing.T) {
	base := NewIssue(SeverityWarning, CodeMissingSUBM, "msg", "@I1@")
	withLine := base.WithLineNumber(42)

	if withLine.LineNumber != 42 {
		t.Errorf("LineNumber = %d, want 42", withLine.LineNumber)
	}
	if base.LineNumber != 0 {
		t.Errorf("WithLineNumber mutated the receiver: LineNumber = %d", base.LineNumber)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
