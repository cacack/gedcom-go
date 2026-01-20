package validator

import (
	"testing"

	"github.com/cacack/gedcom-go/gedcom"
)

func TestNewHeaderValidator(t *testing.T) {
	v := NewHeaderValidator()
	if v == nil {
		t.Error("NewHeaderValidator() returned nil")
	}
}

func TestHeaderValidatorValidateHeader_NilDocument(t *testing.T) {
	v := NewHeaderValidator()
	issues := v.ValidateHeader(nil)
	if issues != nil {
		t.Errorf("ValidateHeader(nil) should return nil, got %d issues", len(issues))
	}
}

func TestHeaderValidatorValidateHeader_NilHeader(t *testing.T) {
	v := NewHeaderValidator()
	doc := &gedcom.Document{
		Header:  nil,
		Records: []*gedcom.Record{},
		XRefMap: make(map[string]*gedcom.Record),
	}
	issues := v.ValidateHeader(doc)
	if issues != nil {
		t.Errorf("ValidateHeader with nil Header should return nil, got %d issues", len(issues))
	}
}

func TestHeaderValidatorValidateHeader_SUBM(t *testing.T) {
	tests := []struct {
		name           string
		version        gedcom.Version
		submitter      string
		wantIssues     int
		wantCode       string
		wantSeverity   Severity
		wantVersionDet string
	}{
		{
			name:       "GEDCOM 5.5 with SUBM - no issues",
			version:    gedcom.Version55,
			submitter:  "@U1@",
			wantIssues: 0,
		},
		{
			name:       "GEDCOM 5.5.1 with SUBM - no issues",
			version:    gedcom.Version551,
			submitter:  "@U1@",
			wantIssues: 0,
		},
		{
			name:       "GEDCOM 7.0 with SUBM - no issues",
			version:    gedcom.Version70,
			submitter:  "@U1@",
			wantIssues: 0,
		},
		{
			name:       "GEDCOM 7.0 without SUBM - no issues (optional in 7.0)",
			version:    gedcom.Version70,
			submitter:  "",
			wantIssues: 0,
		},
		{
			name:           "GEDCOM 5.5 without SUBM - warning",
			version:        gedcom.Version55,
			submitter:      "",
			wantIssues:     1,
			wantCode:       CodeMissingSUBM,
			wantSeverity:   SeverityWarning,
			wantVersionDet: "5.5",
		},
		{
			name:           "GEDCOM 5.5.1 without SUBM - warning",
			version:        gedcom.Version551,
			submitter:      "",
			wantIssues:     1,
			wantCode:       CodeMissingSUBM,
			wantSeverity:   SeverityWarning,
			wantVersionDet: "5.5.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewHeaderValidator()
			doc := &gedcom.Document{
				Header: &gedcom.Header{
					Version:   tt.version,
					Submitter: tt.submitter,
				},
				Records: []*gedcom.Record{},
				XRefMap: make(map[string]*gedcom.Record),
			}

			issues := v.ValidateHeader(doc)

			if len(issues) != tt.wantIssues {
				t.Errorf("ValidateHeader() returned %d issues, want %d", len(issues), tt.wantIssues)
				for _, issue := range issues {
					t.Logf("  Issue: %s", issue.String())
				}
				return
			}

			if tt.wantIssues > 0 {
				issue := issues[0]

				if issue.Code != tt.wantCode {
					t.Errorf("issue.Code = %q, want %q", issue.Code, tt.wantCode)
				}

				if issue.Severity != tt.wantSeverity {
					t.Errorf("issue.Severity = %v, want %v", issue.Severity, tt.wantSeverity)
				}

				if issue.RecordXRef != "" {
					t.Errorf("issue.RecordXRef = %q, want empty string", issue.RecordXRef)
				}

				if issue.Details["version"] != tt.wantVersionDet {
					t.Errorf("issue.Details[\"version\"] = %q, want %q", issue.Details["version"], tt.wantVersionDet)
				}
			}
		})
	}
}

func TestHeaderValidatorValidateHeader_MessageContainsVersion(t *testing.T) {
	tests := []struct {
		name           string
		version        gedcom.Version
		expectContains string
	}{
		{
			name:           "GEDCOM 5.5 message includes version",
			version:        gedcom.Version55,
			expectContains: "5.5",
		},
		{
			name:           "GEDCOM 5.5.1 message includes version",
			version:        gedcom.Version551,
			expectContains: "5.5.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewHeaderValidator()
			doc := &gedcom.Document{
				Header: &gedcom.Header{
					Version:   tt.version,
					Submitter: "",
				},
				Records: []*gedcom.Record{},
				XRefMap: make(map[string]*gedcom.Record),
			}

			issues := v.ValidateHeader(doc)

			if len(issues) != 1 {
				t.Fatalf("Expected 1 issue, got %d", len(issues))
			}

			if !contains(issues[0].Message, tt.expectContains) {
				t.Errorf("Message %q does not contain %q", issues[0].Message, tt.expectContains)
			}
		})
	}
}

func TestHeaderValidatorValidateHeader_UnknownVersion(t *testing.T) {
	// Unknown versions should not produce SUBM warnings since we can't determine requirements
	v := NewHeaderValidator()
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version:   gedcom.Version("9.9"), // Unknown version
			Submitter: "",
		},
		Records: []*gedcom.Record{},
		XRefMap: make(map[string]*gedcom.Record),
	}

	issues := v.ValidateHeader(doc)

	if len(issues) != 0 {
		t.Errorf("Expected 0 issues for unknown version, got %d", len(issues))
		for _, issue := range issues {
			t.Logf("  Issue: %s", issue.String())
		}
	}
}

func TestHeaderValidatorValidateHeader_EmptyVersion(t *testing.T) {
	// Empty version should not produce SUBM warnings
	v := NewHeaderValidator()
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version:   "",
			Submitter: "",
		},
		Records: []*gedcom.Record{},
		XRefMap: make(map[string]*gedcom.Record),
	}

	issues := v.ValidateHeader(doc)

	if len(issues) != 0 {
		t.Errorf("Expected 0 issues for empty version, got %d", len(issues))
	}
}

func TestHeaderValidatorValidateHeader_IssueImplementsError(t *testing.T) {
	v := NewHeaderValidator()
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version:   gedcom.Version55,
			Submitter: "",
		},
		Records: []*gedcom.Record{},
		XRefMap: make(map[string]*gedcom.Record),
	}

	issues := v.ValidateHeader(doc)

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(issues))
	}

	// Verify issue implements error interface
	var _ error = issues[0]

	// Verify Error() returns non-empty string
	if issues[0].Error() == "" {
		t.Error("Issue.Error() should return non-empty string")
	}
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s != "" && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
