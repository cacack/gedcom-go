package validator

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/gedcom"
)

// Helper to create a document with specific version and XRefs
func newTestDocumentWithVersion(version gedcom.Version, xrefs ...string) *gedcom.Document {
	doc := &gedcom.Document{
		Header:  &gedcom.Header{Version: version},
		Records: []*gedcom.Record{},
		XRefMap: make(map[string]*gedcom.Record),
	}
	for _, xref := range xrefs {
		record := &gedcom.Record{XRef: xref}
		doc.Records = append(doc.Records, record)
		doc.XRefMap[xref] = record
	}
	return doc
}

func TestNewXRefValidator(t *testing.T) {
	v := NewXRefValidator()
	if v == nil {
		t.Error("NewXRefValidator() returned nil")
	}
}

func TestXRefValidator_ValidateXRefs_NilDocument(t *testing.T) {
	v := NewXRefValidator()
	issues := v.ValidateXRefs(nil)
	if len(issues) != 0 {
		t.Errorf("ValidateXRefs(nil) should return empty slice, got %d issues", len(issues))
	}
}

func TestXRefValidator_ValidateXRefs_EmptyDocument(t *testing.T) {
	v := NewXRefValidator()
	doc := newTestDocumentWithVersion(gedcom.Version55)

	issues := v.ValidateXRefs(doc)
	if len(issues) != 0 {
		t.Errorf("ValidateXRefs on empty document should return 0 issues, got %d", len(issues))
	}
}

func TestXRefValidator_ValidateXRefs_ExactlyMaxLength(t *testing.T) {
	// XRef with exactly 20 characters (excluding @ delimiters) should pass
	v := NewXRefValidator()

	// 20 character content: "12345678901234567890"
	xref := "@12345678901234567890@"
	content := strings.Trim(xref, "@")
	if len(content) != 20 {
		t.Fatalf("Test setup error: expected 20 chars, got %d", len(content))
	}

	tests := []struct {
		name    string
		version gedcom.Version
	}{
		{"GEDCOM 5.5", gedcom.Version55},
		{"GEDCOM 5.5.1", gedcom.Version551},
		{"GEDCOM 7.0", gedcom.Version70},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := newTestDocumentWithVersion(tt.version, xref)
			issues := v.ValidateXRefs(doc)
			if len(issues) != 0 {
				t.Errorf("XRef with exactly 20 chars should pass for %s, got %d issues", tt.version, len(issues))
			}
		})
	}
}

func TestXRefValidator_ValidateXRefs_ExceedsMaxLength(t *testing.T) {
	// XRef with 21 characters (excluding @ delimiters) should fail for 5.5/5.5.1
	v := NewXRefValidator()

	// 21 character content: "123456789012345678901"
	xref := "@123456789012345678901@"
	content := strings.Trim(xref, "@")
	if len(content) != 21 {
		t.Fatalf("Test setup error: expected 21 chars, got %d", len(content))
	}

	tests := []struct {
		name        string
		version     gedcom.Version
		expectIssue bool
	}{
		{"GEDCOM 5.5 fails", gedcom.Version55, true},
		{"GEDCOM 5.5.1 fails", gedcom.Version551, true},
		{"GEDCOM 7.0 passes", gedcom.Version70, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := newTestDocumentWithVersion(tt.version, xref)
			issues := v.ValidateXRefs(doc)

			if tt.expectIssue {
				if len(issues) != 1 {
					t.Fatalf("Expected 1 issue for %s, got %d", tt.version, len(issues))
				}

				issue := issues[0]
				if issue.Code != CodeXRefTooLong {
					t.Errorf("Expected code %s, got %s", CodeXRefTooLong, issue.Code)
				}
				if issue.Severity != SeverityWarning {
					t.Errorf("Expected severity WARNING, got %s", issue.Severity)
				}
				if issue.RecordXRef != xref {
					t.Errorf("Expected RecordXRef %s, got %s", xref, issue.RecordXRef)
				}
				if issue.Details["length"] != "21" {
					t.Errorf("Expected length detail '21', got %s", issue.Details["length"])
				}
				if issue.Details["version"] != string(tt.version) {
					t.Errorf("Expected version detail %s, got %s", tt.version, issue.Details["version"])
				}
			} else if len(issues) != 0 {
				t.Errorf("Expected no issues for %s, got %d", tt.version, len(issues))
			}
		})
	}
}

func TestXRefValidator_ValidateXRefs_MultipleXRefs(t *testing.T) {
	v := NewXRefValidator()

	// Mix of valid and invalid XRefs
	validXRef := "@I1@"                                 // 2 chars - valid
	exactXRef := "@12345678901234567890@"               // 20 chars - valid
	longXRef1 := "@123456789012345678901@"              // 21 chars - invalid
	longXRef2 := "@1234567890123456789012345678901234@" // 34 chars - invalid

	doc := newTestDocumentWithVersion(gedcom.Version55, validXRef, exactXRef, longXRef1, longXRef2)
	issues := v.ValidateXRefs(doc)

	if len(issues) != 2 {
		t.Errorf("Expected 2 issues, got %d", len(issues))
		for _, issue := range issues {
			t.Logf("  Issue: %s", issue.String())
		}
	}

	// Verify both long XRefs are reported
	foundXRefs := make(map[string]bool)
	for _, issue := range issues {
		foundXRefs[issue.RecordXRef] = true
	}

	if !foundXRefs[longXRef1] {
		t.Errorf("Expected issue for %s", longXRef1)
	}
	if !foundXRefs[longXRef2] {
		t.Errorf("Expected issue for %s", longXRef2)
	}
}

func TestXRefValidator_ValidateXRefs_VersionBoundary(t *testing.T) {
	// Test that version boundary is exactly at 7.0
	v := NewXRefValidator()

	longXRef := "@123456789012345678901@" // 21 chars

	tests := []struct {
		name        string
		version     gedcom.Version
		expectIssue bool
	}{
		{"5.5 reports issue", gedcom.Version55, true},
		{"5.5.1 reports issue", gedcom.Version551, true},
		{"7.0 no issue", gedcom.Version70, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := newTestDocumentWithVersion(tt.version, longXRef)
			issues := v.ValidateXRefs(doc)

			if tt.expectIssue && len(issues) == 0 {
				t.Errorf("Expected issue for version %s", tt.version)
			}
			if !tt.expectIssue && len(issues) > 0 {
				t.Errorf("Expected no issue for version %s, got %d", tt.version, len(issues))
			}
		})
	}
}

func TestXRefValidator_ValidateXRefs_IssueDetails(t *testing.T) {
	v := NewXRefValidator()

	// XRef with 25 characters content
	xref := "@1234567890123456789012345@"
	content := strings.Trim(xref, "@")
	expectedLength := len(content)

	doc := newTestDocumentWithVersion(gedcom.Version551, xref)
	issues := v.ValidateXRefs(doc)

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]

	// Verify all expected details are present
	if issue.Details["length"] != "25" {
		t.Errorf("Expected length %d, got %s", expectedLength, issue.Details["length"])
	}
	if issue.Details["version"] != "5.5.1" {
		t.Errorf("Expected version '5.5.1', got %s", issue.Details["version"])
	}

	// Verify message contains XRef and limit
	if !strings.Contains(issue.Message, xref) {
		t.Errorf("Message should contain XRef, got: %s", issue.Message)
	}
	if !strings.Contains(issue.Message, "20") {
		t.Errorf("Message should contain limit '20', got: %s", issue.Message)
	}
	if !strings.Contains(issue.Message, "5.5.1") {
		t.Errorf("Message should contain version, got: %s", issue.Message)
	}

	// Verify issue implements error interface
	var _ error = issue

	// Verify Error() returns non-empty string
	if issue.Error() == "" {
		t.Error("Issue.Error() should return non-empty string")
	}
}

func TestXRefValidator_ValidateXRefs_ShortXRefs(t *testing.T) {
	// Very short XRefs should always pass
	v := NewXRefValidator()

	shortXRefs := []string{"@I1@", "@F1@", "@S1@", "@N1@"}

	for _, version := range []gedcom.Version{gedcom.Version55, gedcom.Version551, gedcom.Version70} {
		doc := newTestDocumentWithVersion(version, shortXRefs...)
		issues := v.ValidateXRefs(doc)

		if len(issues) != 0 {
			t.Errorf("Short XRefs should pass for %s, got %d issues", version, len(issues))
		}
	}
}

func TestXRefValidator_ValidateXRefs_UnknownVersion(t *testing.T) {
	// Unknown version should be treated conservatively (validate like older versions)
	v := NewXRefValidator()

	longXRef := "@123456789012345678901@" // 21 chars

	doc := newTestDocumentWithVersion(gedcom.Version(""), longXRef)
	issues := v.ValidateXRefs(doc)

	// Empty/unknown version should be before 7.0, so issues are expected
	if len(issues) != 1 {
		t.Errorf("Expected 1 issue for unknown version, got %d", len(issues))
	}
}

func TestMaxXRefLengthConstant(t *testing.T) {
	// Verify the constant value
	if MaxXRefLength != 20 {
		t.Errorf("MaxXRefLength = %d, want 20", MaxXRefLength)
	}
}
