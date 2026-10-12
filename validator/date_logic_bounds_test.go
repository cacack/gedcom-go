package validator

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Regression tests for #566: an ordering is an ERROR only when the dates'
// bounds prove it; an ordering that only Compare's reading suggests is a
// WARNING, and none is raised when neither holds.

// dateOrderCase is one ordering check: first is the date that must not come
// before second (death vs birth, child's birth vs parent's, marriage vs birth).
type dateOrderCase struct {
	name          string
	first, second string
	wantSeverity  *Severity // nil means no issue
}

func severityPtr(s Severity) *Severity { return &s }

var dateOrderCases = []dateOrderCase{
	{"provable year order", "1940", "1950", severityPtr(SeverityError)},
	{"provable from open-ended BEF", "BEF 1950", "1960", severityPtr(SeverityError)},
	{"year against a day in that year", "1503", "2 FEB 1503", severityPtr(SeverityWarning)},
	{"AFT against a later ABT", "AFT 1078", "ABT 1090", severityPtr(SeverityWarning)},
	{"range overlapping a year", "BET 1840 AND 1860", "1850", severityPtr(SeverityWarning)},
	{"month against a day in that month", "MAR 1850", "15 MAR 1850", severityPtr(SeverityWarning)},
	{"dual date read in New Style is later", "10 FEB 1731/32", "20 DEC 1731", nil},
	{"dual date read in New Style proves order", "20 DEC 1731", "10 FEB 1731/32", severityPtr(SeverityError)},
	// #632: an ABT, CAL or EST date caps the ordering at WARNING, even when
	// the bounds are disjoint; INT names an exact date and can still prove it.
	{"ABT against a later ABT", "ABT 1849", "ABT 1850", severityPtr(SeverityWarning)},
	{"exact against a later ABT", "1849", "ABT 1850", severityPtr(SeverityWarning)},
	{"CAL against a later exact", "CAL 1849", "1850", severityPtr(SeverityWarning)},
	{"exact against a far later EST", "1700", "EST 1900", severityPtr(SeverityWarning)},
	{"INT against a later exact", "INT 1849 (from a letter)", "1850", severityPtr(SeverityError)},
	{"ABT in order", "ABT 1850", "ABT 1849", nil},
	{"in order", "1950", "1940", nil},
	{"same day", "2 FEB 1503", "2 FEB 1503", nil},
}

func mustParseDate(t *testing.T, s string) *gedcom.Date {
	t.Helper()
	d, err := gedcom.ParseDate(s)
	if err != nil {
		t.Fatalf("ParseDate(%q) error: %v", s, err)
	}
	return d
}

func checkDateOrderIssues(t *testing.T, issues []Issue, code string, want *Severity) {
	t.Helper()
	if want == nil {
		if len(issues) != 0 {
			t.Fatalf("got %d issues (%v), want none", len(issues), issues)
		}
		return
	}
	if len(issues) != 1 {
		t.Fatalf("got %d issues (%v), want 1", len(issues), issues)
	}
	if issues[0].Code != code {
		t.Errorf("Code = %q, want %q", issues[0].Code, code)
	}
	if issues[0].Severity != *want {
		t.Errorf("Severity = %v, want %v (message %q)", issues[0].Severity, *want, issues[0].Message)
	}
	// Only an unproven ordering says "possibly before".
	if possibly := strings.Contains(issues[0].Message, "possibly before"); possibly != (*want == SeverityWarning) {
		t.Errorf("Message = %q, contains \"possibly before\" = %v, want %v", issues[0].Message, possibly, *want == SeverityWarning)
	}
}

func TestDateLogicValidator_DeathBeforeBirth_Bounds(t *testing.T) {
	v := NewDateLogicValidator(nil)
	for _, tt := range dateOrderCases {
		t.Run(tt.name, func(t *testing.T) {
			ind := &gedcom.Individual{XRef: "@I1@", Events: []*gedcom.Event{
				{Type: gedcom.EventBirth, ParsedDate: mustParseDate(t, tt.second)},
				{Type: gedcom.EventDeath, ParsedDate: mustParseDate(t, tt.first)},
			}}
			var issues []Issue
			if issue := v.checkDeathBeforeBirth(ind); issue != nil {
				issues = append(issues, *issue)
			}
			checkDateOrderIssues(t, issues, CodeDeathBeforeBirth, tt.wantSeverity)
		})
	}
}

func TestDateLogicValidator_ChildBeforeParent_Bounds(t *testing.T) {
	v := NewDateLogicValidator(nil)
	for _, tt := range dateOrderCases {
		t.Run(tt.name, func(t *testing.T) {
			child := &gedcom.Individual{
				XRef:            "@I1@",
				Events:          []*gedcom.Event{{Type: gedcom.EventBirth, ParsedDate: mustParseDate(t, tt.first)}},
				ChildInFamilies: []gedcom.FamilyLink{{FamilyXRef: "@F1@"}},
			}
			parent := &gedcom.Individual{
				XRef:   "@I2@",
				Events: []*gedcom.Event{{Type: gedcom.EventBirth, ParsedDate: mustParseDate(t, tt.second)}},
			}
			family := &gedcom.Family{XRef: "@F1@", Husband: "@I2@", Children: []string{"@I1@"}}
			doc := makeDocument([]*gedcom.Individual{child, parent}, []*gedcom.Family{family})
			checkDateOrderIssues(t, v.checkChildBeforeParent(doc, child), CodeChildBeforeParent, tt.wantSeverity)
		})
	}
}

func TestDateLogicValidator_MarriageBeforeBirth_Bounds(t *testing.T) {
	v := NewDateLogicValidator(nil)
	for _, tt := range dateOrderCases {
		t.Run(tt.name, func(t *testing.T) {
			ind := &gedcom.Individual{
				XRef:             "@I1@",
				Events:           []*gedcom.Event{{Type: gedcom.EventBirth, ParsedDate: mustParseDate(t, tt.second)}},
				SpouseInFamilies: []gedcom.FamilyLink{{FamilyXRef: "@F1@"}},
			}
			family := &gedcom.Family{XRef: "@F1@", Husband: "@I1@", Events: []*gedcom.Event{
				{Type: gedcom.EventMarriage, ParsedDate: mustParseDate(t, tt.first)},
			}}
			doc := makeDocument([]*gedcom.Individual{ind}, []*gedcom.Family{family})
			checkDateOrderIssues(t, v.checkMarriageBeforeBirth(doc, ind), CodeMarriageBeforeBirth, tt.wantSeverity)
		})
	}
}

// TestOrderSeverity_NoBounds covers dates Bounds cannot place: the order then
// rests on Compare alone, so it is never an ERROR.
func TestOrderSeverity_NoBounds(t *testing.T) {
	unknown := gedcom.Calendar(99)
	early := &gedcom.Date{Year: 1940, Calendar: unknown}
	late := &gedcom.Date{Year: 1950, Calendar: unknown}

	if severity, found := orderSeverity(early, late); !found || severity != SeverityWarning {
		t.Errorf("orderSeverity(early, late) = (%v, %v), want (%v, true)", severity, found, SeverityWarning)
	}
	if severity, found := orderSeverity(late, early); found || severity != SeverityInfo {
		t.Errorf("orderSeverity(late, early) = (%v, %v), want (%v, false)", severity, found, SeverityInfo)
	}
}

// TestOrderSeverity_InvalidDay covers a date Bounds rejects, such as 31 FEB:
// the order rests on Compare alone, so it is at most a WARNING.
func TestOrderSeverity_InvalidDay(t *testing.T) {
	a := mustParseDate(t, "31 FEB 1850")
	b := mustParseDate(t, "1 MAR 1850")
	if severity, found := orderSeverity(a, b); !found || severity != SeverityWarning {
		t.Errorf("orderSeverity = (%v, %v), want (%v, true)", severity, found, SeverityWarning)
	}
}
