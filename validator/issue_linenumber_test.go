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

// TestControlCharIssueCarriesLineNumber pins the two control-character paths
// that have a source line in hand. The record-value path shipped passing a
// literal 0 despite Record.LineNumber being populated and in scope; no test
// asserted the value, which is why the gap got through (panel review).
func TestControlCharIssueCarriesLineNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLine int
	}{
		{
			// The banned character sits on a subordinate tag's value.
			name: "tag value",
			input: "0 HEAD\n1 GEDC\n2 VERS 7.0\n1 CHAR UTF-8\n" +
				"0 @I1@ INDI\n1 NAME John\x07Doe\n0 TRLR\n",
			wantLine: 6,
		},
		{
			// The banned character sits on a record's level-0 value.
			name: "record value",
			input: "0 HEAD\n1 GEDC\n2 VERS 7.0\n1 CHAR UTF-8\n" +
				"0 @N1@ NOTE bad\x07text\n0 TRLR\n",
			wantLine: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := decoder.Decode(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}

			var found bool
			for _, issue := range New().ValidateEncoding(doc) {
				if issue.Code != CodeBannedControlCharacter {
					continue
				}
				found = true
				if issue.LineNumber != tt.wantLine {
					t.Errorf("LineNumber = %d, want %d", issue.LineNumber, tt.wantLine)
				}
				// The byte offset within the value is a different fact and
				// must survive alongside the line number.
				if issue.Details["position"] == "" {
					t.Error("Details[position] missing; it is not the line number and should remain")
				}
			}
			if !found {
				t.Fatal("no BANNED_CONTROL_CHARACTER issue; the fixture would prove nothing")
			}
		})
	}
}
