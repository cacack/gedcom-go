package validator

import (
	"strings"
	"testing"
)

func TestSeverityString(t *testing.T) {
	tests := []struct {
		name     string
		severity Severity
		want     string
	}{
		{
			name:     "error severity",
			severity: SeverityError,
			want:     "ERROR",
		},
		{
			name:     "warning severity",
			severity: SeverityWarning,
			want:     "WARNING",
		},
		{
			name:     "info severity",
			severity: SeverityInfo,
			want:     "INFO",
		},
		{
			name:     "unknown severity",
			severity: Severity(99),
			want:     "UNKNOWN(99)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.severity.String()
			if got != tt.want {
				t.Errorf("Severity.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSeverityIotaValues(t *testing.T) {
	// Verify iota ordering is as expected
	if SeverityError != 0 {
		t.Errorf("SeverityError should be 0, got %d", SeverityError)
	}
	if SeverityWarning != 1 {
		t.Errorf("SeverityWarning should be 1, got %d", SeverityWarning)
	}
	if SeverityInfo != 2 {
		t.Errorf("SeverityInfo should be 2, got %d", SeverityInfo)
	}
}

func TestIssueString(t *testing.T) {
	tests := []struct {
		name  string
		issue Issue
		want  string
	}{
		{
			name: "minimal issue",
			issue: Issue{
				Severity: SeverityError,
				Code:     CodeDeathBeforeBirth,
				Message:  "Death date is before birth date",
			},
			want: "[ERROR] DEATH_BEFORE_BIRTH: Death date is before birth date",
		},
		{
			name: "issue with record xref",
			issue: Issue{
				Severity:   SeverityWarning,
				Code:       CodeMissingBirthDate,
				Message:    "Individual has no birth date",
				RecordXRef: "@I1@",
			},
			want: "[WARNING] MISSING_BIRTH_DATE: Individual has no birth date (@I1@)",
		},
		{
			name: "issue with related xref",
			issue: Issue{
				Severity:    SeverityError,
				Code:        CodeChildBeforeParent,
				Message:     "Child born before parent",
				RecordXRef:  "@I2@",
				RelatedXRef: "@I1@",
			},
			want: "[ERROR] CHILD_BEFORE_PARENT: Child born before parent (@I2@ -> @I1@)",
		},
		{
			name: "info severity issue",
			issue: Issue{
				Severity:   SeverityInfo,
				Code:       CodeNoSources,
				Message:    "No source citations found",
				RecordXRef: "@I5@",
			},
			want: "[INFO] NO_SOURCES: No source citations found (@I5@)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.issue.String()
			if got != tt.want {
				t.Errorf("Issue.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIssueError(t *testing.T) {
	issue := Issue{
		Severity:   SeverityError,
		Code:       CodeDeathBeforeBirth,
		Message:    "Death date is before birth date",
		RecordXRef: "@I1@",
	}

	// Error() should return the same as String()
	if issue.Error() != issue.String() {
		t.Errorf("Issue.Error() = %q, want %q", issue.Error(), issue.String())
	}
}

func TestIssueImplementsError(t *testing.T) {
	// Verify Issue implements error interface
	var _ error = Issue{}

	issue := NewIssue(SeverityError, CodeDeathBeforeBirth, "test", "@I1@")
	errStr := issue.Error()

	if errStr == "" {
		t.Error("Error() should return non-empty string")
	}

	if !strings.Contains(errStr, CodeDeathBeforeBirth) {
		t.Error("Error() should contain error code")
	}
}

func TestNewIssue(t *testing.T) {
	tests := []struct {
		name       string
		severity   Severity
		code       string
		message    string
		recordXRef string
	}{
		{
			name:       "error issue",
			severity:   SeverityError,
			code:       CodeDeathBeforeBirth,
			message:    "Death before birth",
			recordXRef: "@I1@",
		},
		{
			name:       "warning issue",
			severity:   SeverityWarning,
			code:       CodeImpossibleAge,
			message:    "Age exceeds 120 years",
			recordXRef: "@I2@",
		},
		{
			name:       "info issue without xref",
			severity:   SeverityInfo,
			code:       CodeNoSources,
			message:    "No sources",
			recordXRef: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := NewIssue(tt.severity, tt.code, tt.message, tt.recordXRef)

			if issue.Severity != tt.severity {
				t.Errorf("Severity = %v, want %v", issue.Severity, tt.severity)
			}
			if issue.Code != tt.code {
				t.Errorf("Code = %q, want %q", issue.Code, tt.code)
			}
			if issue.Message != tt.message {
				t.Errorf("Message = %q, want %q", issue.Message, tt.message)
			}
			if issue.RecordXRef != tt.recordXRef {
				t.Errorf("RecordXRef = %q, want %q", issue.RecordXRef, tt.recordXRef)
			}
			if issue.RelatedXRef != "" {
				t.Errorf("RelatedXRef should be empty, got %q", issue.RelatedXRef)
			}
			if issue.Details == nil {
				t.Error("Details should be initialized to non-nil map")
			}
			if len(issue.Details) != 0 {
				t.Errorf("Details should be empty, got %d entries", len(issue.Details))
			}
		})
	}
}

func TestIssueWithRelatedXRef(t *testing.T) {
	original := NewIssue(SeverityError, CodeChildBeforeParent, "Child born before parent", "@I2@")

	// Add related xref
	modified := original.WithRelatedXRef("@I1@")

	// Original should be unchanged
	if original.RelatedXRef != "" {
		t.Errorf("Original RelatedXRef should be empty, got %q", original.RelatedXRef)
	}

	// Modified should have the related xref
	if modified.RelatedXRef != "@I1@" {
		t.Errorf("Modified RelatedXRef = %q, want %q", modified.RelatedXRef, "@I1@")
	}

	// Other fields should be preserved
	if modified.Severity != original.Severity {
		t.Error("Severity should be preserved")
	}
	if modified.Code != original.Code {
		t.Error("Code should be preserved")
	}
	if modified.Message != original.Message {
		t.Error("Message should be preserved")
	}
	if modified.RecordXRef != original.RecordXRef {
		t.Error("RecordXRef should be preserved")
	}
}

func TestIssueWithDetail(t *testing.T) {
	original := NewIssue(SeverityWarning, CodeImpossibleAge, "Age exceeds limit", "@I1@")

	// Add first detail
	modified1 := original.WithDetail("age", "150")

	// Original should be unchanged
	if len(original.Details) != 0 {
		t.Errorf("Original Details should be empty, got %d entries", len(original.Details))
	}

	// Modified should have the detail
	if modified1.Details["age"] != "150" {
		t.Errorf("Modified Details[age] = %q, want %q", modified1.Details["age"], "150")
	}

	// Add second detail (chaining)
	modified2 := modified1.WithDetail("limit", "120")

	// modified1 should be unchanged (immutability check)
	if _, exists := modified1.Details["limit"]; exists {
		t.Error("modified1 should not have 'limit' key")
	}

	// modified2 should have both details
	if modified2.Details["age"] != "150" {
		t.Errorf("modified2 Details[age] = %q, want %q", modified2.Details["age"], "150")
	}
	if modified2.Details["limit"] != "120" {
		t.Errorf("modified2 Details[limit] = %q, want %q", modified2.Details["limit"], "120")
	}
}

func TestIssueWithDetailNilDetails(t *testing.T) {
	// Create issue with nil Details (bypassing NewIssue)
	issue := Issue{
		Severity:   SeverityError,
		Code:       "TEST",
		Message:    "Test message",
		RecordXRef: "@I1@",
		Details:    nil,
	}

	// WithDetail should initialize the map
	modified := issue.WithDetail("key", "value")

	if modified.Details == nil {
		t.Error("Details should be initialized")
	}
	if modified.Details["key"] != "value" {
		t.Errorf("Details[key] = %q, want %q", modified.Details["key"], "value")
	}
}

func TestFilterBySeverity(t *testing.T) {
	issues := []Issue{
		NewIssue(SeverityError, CodeDeathBeforeBirth, "Error 1", "@I1@"),
		NewIssue(SeverityWarning, CodeImpossibleAge, "Warning 1", "@I2@"),
		NewIssue(SeverityError, CodeChildBeforeParent, "Error 2", "@I3@"),
		NewIssue(SeverityInfo, CodeNoSources, "Info 1", "@I4@"),
		NewIssue(SeverityWarning, CodeMissingBirthDate, "Warning 2", "@I5@"),
	}

	tests := []struct {
		name     string
		severity Severity
		wantLen  int
	}{
		{
			name:     "filter errors",
			severity: SeverityError,
			wantLen:  2,
		},
		{
			name:     "filter warnings",
			severity: SeverityWarning,
			wantLen:  2,
		},
		{
			name:     "filter info",
			severity: SeverityInfo,
			wantLen:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := FilterBySeverity(issues, tt.severity)

			if len(filtered) != tt.wantLen {
				t.Errorf("FilterBySeverity() returned %d issues, want %d", len(filtered), tt.wantLen)
			}

			// Verify all returned issues have the correct severity
			for _, issue := range filtered {
				if issue.Severity != tt.severity {
					t.Errorf("Filtered issue has severity %v, want %v", issue.Severity, tt.severity)
				}
			}
		})
	}
}

func TestFilterBySeverityEmpty(t *testing.T) {
	var issues []Issue

	filtered := FilterBySeverity(issues, SeverityError)

	if len(filtered) != 0 {
		t.Errorf("FilterBySeverity on empty slice should return empty, got %d items", len(filtered))
	}
}

func TestFilterBySeverityNoMatch(t *testing.T) {
	issues := []Issue{
		NewIssue(SeverityError, CodeDeathBeforeBirth, "Error", "@I1@"),
		NewIssue(SeverityError, CodeChildBeforeParent, "Error", "@I2@"),
	}

	filtered := FilterBySeverity(issues, SeverityInfo)

	if len(filtered) != 0 {
		t.Errorf("FilterBySeverity should return empty when no match, got %d items", len(filtered))
	}
}

func TestFilterByCode(t *testing.T) {
	issues := []Issue{
		NewIssue(SeverityError, CodeDeathBeforeBirth, "Error 1", "@I1@"),
		NewIssue(SeverityWarning, CodeImpossibleAge, "Warning 1", "@I2@"),
		NewIssue(SeverityError, CodeDeathBeforeBirth, "Error 2", "@I3@"),
		NewIssue(SeverityInfo, CodeNoSources, "Info 1", "@I4@"),
		NewIssue(SeverityError, CodeChildBeforeParent, "Error 3", "@I5@"),
	}

	tests := []struct {
		name    string
		code    string
		wantLen int
	}{
		{
			name:    "filter death before birth",
			code:    CodeDeathBeforeBirth,
			wantLen: 2,
		},
		{
			name:    "filter impossible age",
			code:    CodeImpossibleAge,
			wantLen: 1,
		},
		{
			name:    "filter child before parent",
			code:    CodeChildBeforeParent,
			wantLen: 1,
		},
		{
			name:    "filter no sources",
			code:    CodeNoSources,
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := FilterByCode(issues, tt.code)

			if len(filtered) != tt.wantLen {
				t.Errorf("FilterByCode() returned %d issues, want %d", len(filtered), tt.wantLen)
			}

			// Verify all returned issues have the correct code
			for _, issue := range filtered {
				if issue.Code != tt.code {
					t.Errorf("Filtered issue has code %q, want %q", issue.Code, tt.code)
				}
			}
		})
	}
}

func TestFilterByCodeEmpty(t *testing.T) {
	var issues []Issue

	filtered := FilterByCode(issues, CodeDeathBeforeBirth)

	if len(filtered) != 0 {
		t.Errorf("FilterByCode on empty slice should return empty, got %d items", len(filtered))
	}
}

func TestFilterByCodeNoMatch(t *testing.T) {
	issues := []Issue{
		NewIssue(SeverityError, CodeDeathBeforeBirth, "Error", "@I1@"),
		NewIssue(SeverityError, CodeChildBeforeParent, "Error", "@I2@"),
	}

	filtered := FilterByCode(issues, CodeNoSources)

	if len(filtered) != 0 {
		t.Errorf("FilterByCode should return empty when no match, got %d items", len(filtered))
	}
}

func TestErrorCodeConstants(t *testing.T) {
	// Verify all error codes are defined and non-empty
	dateCodes := []string{
		CodeDeathBeforeBirth,
		CodeChildBeforeParent,
		CodeMarriageBeforeBirth,
		CodeImpossibleAge,
		CodeUnreasonableParentAge,
	}

	refCodes := []string{
		CodeOrphanedFAMC,
		CodeOrphanedFAMS,
		CodeOrphanedHUSB,
		CodeOrphanedWIFE,
		CodeOrphanedCHIL,
		CodeOrphanedSOUR,
	}

	dupCodes := []string{
		CodePotentialDuplicate,
	}

	qualityCodes := []string{
		CodeMissingBirthDate,
		CodeMissingDeathDate,
		CodeMissingName,
		CodeNoSources,
	}

	allCodes := make([]string, 0, len(dateCodes)+len(refCodes)+len(dupCodes)+len(qualityCodes))
	allCodes = append(allCodes, dateCodes...)
	allCodes = append(allCodes, refCodes...)
	allCodes = append(allCodes, dupCodes...)
	allCodes = append(allCodes, qualityCodes...)

	for _, code := range allCodes {
		if code == "" {
			t.Error("Found empty error code constant")
		}
	}

	// Verify codes are unique
	seen := make(map[string]bool)
	for _, code := range allCodes {
		if seen[code] {
			t.Errorf("Duplicate error code: %q", code)
		}
		seen[code] = true
	}
}

func TestIssueMethodChaining(t *testing.T) {
	// Test fluent interface for building issues
	issue := NewIssue(SeverityError, CodeChildBeforeParent, "Child born 10 years before parent", "@I2@").
		WithRelatedXRef("@I1@").
		WithDetail("child_birth", "1950").
		WithDetail("parent_birth", "1960")

	if issue.Severity != SeverityError {
		t.Errorf("Severity = %v, want %v", issue.Severity, SeverityError)
	}
	if issue.Code != CodeChildBeforeParent {
		t.Errorf("Code = %q, want %q", issue.Code, CodeChildBeforeParent)
	}
	if issue.RecordXRef != "@I2@" {
		t.Errorf("RecordXRef = %q, want %q", issue.RecordXRef, "@I2@")
	}
	if issue.RelatedXRef != "@I1@" {
		t.Errorf("RelatedXRef = %q, want %q", issue.RelatedXRef, "@I1@")
	}
	if issue.Details["child_birth"] != "1950" {
		t.Errorf("Details[child_birth] = %q, want %q", issue.Details["child_birth"], "1950")
	}
	if issue.Details["parent_birth"] != "1960" {
		t.Errorf("Details[parent_birth] = %q, want %q", issue.Details["parent_birth"], "1960")
	}
}
