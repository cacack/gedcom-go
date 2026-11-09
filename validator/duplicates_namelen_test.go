package validator

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Issue #717: the name length duplicate detection compares is bounded, since
// per-pair cost is quadratic in it and MaxGroupSize bounds only the pair count.

func longNameDocument(n, runes int) *gedcom.Document {
	doc := &gedcom.Document{XRefMap: map[string]*gedcom.Record{}}
	for i := 0; i < n; i++ {
		// Distinct names of equal length, so the length-based early exit in
		// compareGivenNames does not apply and the full DP would run.
		given := strings.Repeat("a", runes-1) + string(rune('a'+i%26))
		xref := "@I" + strconv.Itoa(i+1) + "@"
		rec := &gedcom.Record{XRef: xref, Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{
			XRef:  xref,
			Names: []*gedcom.PersonalName{{Given: given, Surname: "Smith", Full: given + " /Smith/"}},
		}}
		doc.Records = append(doc.Records, rec)
		doc.XRefMap[xref] = rec
	}
	return doc
}

func limitIssue(t *testing.T, issues []Issue) *Issue {
	t.Helper()
	var found *Issue
	for i := range issues {
		if issues[i].Code == CodeDuplicateDetectionLimited {
			if found != nil {
				t.Fatalf("more than one %s issue", CodeDuplicateDetectionLimited)
			}
			found = &issues[i]
		}
	}
	return found
}

// TestValidateAll_LongNamesAreBounded is the probe from #717: two 100,000-rune
// given names took 51 s before the bound.
func TestValidateAll_LongNamesAreBounded(t *testing.T) {
	doc := longNameDocument(2, 100_000)
	start := time.Now()
	issues := New().ValidateAll(doc)
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("ValidateAll took %v; the name bound is not applied", elapsed)
	}

	issue := limitIssue(t, issues)
	if issue == nil {
		t.Fatalf("no %s issue for names compared on a prefix", CodeDuplicateDetectionLimited)
	}
	if issue.Severity != SeverityWarning {
		t.Errorf("severity = %v, want Warning", issue.Severity)
	}
	if got := issue.Details["truncated_names"]; got != "2" {
		t.Errorf("Details[truncated_names] = %q, want 2", got)
	}
	if got := issue.Details["max_name_runes"]; got != strconv.Itoa(DefaultMaxNameRunes) {
		t.Errorf("Details[max_name_runes] = %q, want %d", got, DefaultMaxNameRunes)
	}
	if got := issue.Details["skipped_groups"]; got != "0" {
		t.Errorf("Details[skipped_groups] = %q, want 0", got)
	}
	if _, has := issue.Details["surnames"]; has {
		t.Error("Details[surnames] present although no group was skipped")
	}
	if !strings.Contains(issue.Message, "MaxNameRunes") || strings.Contains(issue.Message, "MaxGroupSize") {
		t.Errorf("message should describe the name bound only: %q", issue.Message)
	}
}

// TestMaxNameRunes_TriState mirrors TestMaxGroupSize_TriState.
func TestMaxNameRunes_TriState(t *testing.T) {
	tests := []struct {
		name  string
		value int
		want  int
	}{
		{"zero selects the default", 0, DefaultMaxNameRunes},
		{"negative is unlimited", UnlimitedNameRunes, int(^uint(0) >> 1)},
		{"positive is taken as given", 10, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDuplicateDetector(&DuplicateConfig{MaxNameRunes: tt.value})
			if got := d.maxNameRunes(); got != tt.want {
				t.Errorf("maxNameRunes() = %d, want %d", got, tt.want)
			}
		})
	}

	if DefaultDuplicateConfig().MaxNameRunes != DefaultMaxNameRunes {
		t.Error("DefaultDuplicateConfig does not spell out MaxNameRunes")
	}
}

// TestMaxNameRunes_PrefixComparison checks that names within the bound are
// untouched, names beyond it are compared on their prefix, and the unlimited
// setting compares them in full.
func TestMaxNameRunes_PrefixComparison(t *testing.T) {
	// Two names that agree on their first 8 runes and differ after.
	a := "Johannes Ambrosius"
	b := "Johannes Zacharias"
	doc := &gedcom.Document{XRefMap: map[string]*gedcom.Record{}}
	for i, given := range []string{a, b} {
		xref := "@I" + strconv.Itoa(i+1) + "@"
		rec := &gedcom.Record{XRef: xref, Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{
			XRef: xref, Sex: "M", Names: []*gedcom.PersonalName{{Given: given, Surname: "Müller"}},
		}}
		doc.Records = append(doc.Records, rec)
		doc.XRefMap[xref] = rec
	}

	full := NewDuplicateDetector(&DuplicateConfig{MaxNameRunes: UnlimitedNameRunes}).FindDuplicatesReport(doc)
	if len(full.Pairs) != 0 || len(full.LimitIssues) != 0 {
		t.Fatalf("unlimited: pairs %d limit issues %d, want none (names differ)", len(full.Pairs), len(full.LimitIssues))
	}

	within := NewDuplicateDetector(&DuplicateConfig{MaxNameRunes: 32}).FindDuplicatesReport(doc)
	if len(within.Pairs) != 0 || len(within.LimitIssues) != 0 {
		t.Fatalf("bound above the names: pairs %d limit issues %d, want none", len(within.Pairs), len(within.LimitIssues))
	}

	prefix := NewDuplicateDetector(&DuplicateConfig{MaxNameRunes: 8}).FindDuplicatesReport(doc)
	if len(prefix.Pairs) != 1 {
		t.Errorf("bound of 8: %d pairs, want 1 (identical prefixes)", len(prefix.Pairs))
	}
	if len(prefix.LimitIssues) != 1 || prefix.LimitIssues[0].Details["truncated_names"] != "2" {
		t.Errorf("bound of 8: limit issues = %+v, want one counting 2 truncated names", prefix.LimitIssues)
	}
}

// TestLimitIssue_BothCausesInOneMessage checks a document that trips both
// bounds yields one issue describing both.
func TestLimitIssue_BothCausesInOneMessage(t *testing.T) {
	// Smith: 3 members with 200-rune given names, compared on a prefix.
	doc := longNameDocument(3, 200)
	// Jones: 4 members, skipped whole under MaxGroupSize=3.
	for i := 0; i < 4; i++ {
		xref := "@J" + strconv.Itoa(i+1) + "@"
		rec := &gedcom.Record{XRef: xref, Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{
			XRef: xref, Names: []*gedcom.PersonalName{{Given: "Ann", Surname: "Jones"}},
		}}
		doc.Records = append(doc.Records, rec)
		doc.XRefMap[xref] = rec
	}

	report := NewDuplicateDetector(&DuplicateConfig{MaxGroupSize: 3}).FindDuplicatesReport(doc)
	if len(report.LimitIssues) != 1 {
		t.Fatalf("limit issues = %d, want 1", len(report.LimitIssues))
	}
	issue := report.LimitIssues[0]
	if issue.Details["skipped_groups"] != "1" || issue.Details["surnames"] != "jones" {
		t.Errorf("skipped group details = %v", issue.Details)
	}
	if issue.Details["truncated_names"] != "3" {
		t.Errorf("Details[truncated_names] = %q, want 3", issue.Details["truncated_names"])
	}
	if !strings.Contains(issue.Message, "MaxGroupSize=3") || !strings.Contains(issue.Message, "MaxNameRunes") {
		t.Errorf("message should describe both bounds: %q", issue.Message)
	}
}
