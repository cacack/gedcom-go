package validator

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/decoder"
	"github.com/cacack/gedcom-go/gedcom"
)

func TestValidateBrokenXRef(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John Smith
1 FAMS @F999@
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	v := New()
	errs := v.Validate(doc)

	if len(errs) == 0 {
		t.Fatal("Expected validation errors for broken XRef")
	}

	found := false
	for _, e := range errs {
		var valErr *ValidationError
		if errors.As(e, &valErr) && valErr.Code == "BROKEN_XREF" {
			found = true
			t.Logf("Found expected error: %v", e)
		}
	}

	if !found {
		t.Error("Expected BROKEN_XREF error")
	}
}

func TestValidateMissingName(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 SEX M
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	v := New()
	errs := v.Validate(doc)

	if len(errs) == 0 {
		t.Fatal("Expected validation errors for missing NAME")
	}

	found := false
	for _, e := range errs {
		var valErr *ValidationError
		if errors.As(e, &valErr) && valErr.Code == "MISSING_REQUIRED_FIELD" {
			found = true
			t.Logf("Found expected error: %v", e)
		}
	}

	if !found {
		t.Error("Expected MISSING_REQUIRED_FIELD error")
	}
}

func TestValidateValidFile(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
0 @F1@ FAM
1 HUSB @I1@
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	v := New()
	errs := v.Validate(doc)

	if len(errs) != 0 {
		t.Errorf("Expected no validation errors for valid file, got %d errors:", len(errs))
		for _, err := range errs {
			t.Logf("  - %v", err)
		}
	}
}

// TestValidationErrorFormatting tests the Error() method of ValidationError
func TestValidationErrorFormatting(t *testing.T) {
	tests := []struct {
		name string
		err  ValidationError
		want string
	}{
		{
			name: "with XRef only",
			err:  ValidationError{Code: "ERR1", Message: "test error", XRef: "@I1@"},
			want: "[ERR1] test error (XRef: @I1@)",
		},
		{
			name: "with Line only",
			err:  ValidationError{Code: "ERR2", Message: "test error", Line: 42},
			want: "[ERR2] line 42: test error",
		},
		{
			name: "with both XRef and Line (XRef takes precedence)",
			err:  ValidationError{Code: "ERR3", Message: "test error", XRef: "@I1@", Line: 42},
			want: "[ERR3] test error (XRef: @I1@)",
		},
		{
			name: "minimal error (code and message only)",
			err:  ValidationError{Code: "ERR4", Message: "test error"},
			want: "[ERR4] test error",
		},
		{
			name: "with complex message",
			err:  ValidationError{Code: "BROKEN_XREF", Message: "cross-reference @F999@ not found", XRef: "@I1@"},
			want: "[BROKEN_XREF] cross-reference @F999@ not found (XRef: @I1@)",
		},
		{
			name: "with line number and detailed message",
			err:  ValidationError{Code: "MISSING_REQUIRED", Message: "required tag NAME missing", Line: 15},
			want: "[MISSING_REQUIRED] line 15: required tag NAME missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("ValidationError.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestValidationErrorImplementsError verifies ValidationError implements error interface
func TestValidationErrorImplementsError(t *testing.T) {
	var _ error = &ValidationError{}

	err := &ValidationError{
		Code:    "TEST",
		Message: "test message",
		Line:    10,
		XRef:    "@I1@",
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() should return non-empty string")
	}

	if !strings.Contains(errStr, "TEST") {
		t.Error("Error() should contain error code")
	}
	if !strings.Contains(errStr, "test message") {
		t.Error("Error() should contain error message")
	}
}

// TestValidateFamilyEdgeCases tests edge cases in family validation
func TestValidateFamilyEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorCode   string
	}{
		{
			name: "empty family (no members)",
			input: `0 HEAD
1 GEDC
2 VERS 5.5
0 @F1@ FAM
0 TRLR`,
			expectError: true,
			errorCode:   "EMPTY_FAMILY",
		},
		{
			name: "family with only children",
			input: `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME Child /One/
0 @F1@ FAM
1 CHIL @I1@
0 TRLR`,
			expectError: false,
		},
		{
			name: "family with only wife",
			input: `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME Jane /Doe/
0 @F1@ FAM
1 WIFE @I1@
0 TRLR`,
			expectError: false,
		},
		{
			name: "family with only husband",
			input: `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Doe/
0 @F1@ FAM
1 HUSB @I1@
0 TRLR`,
			expectError: false,
		},
		{
			name: "family with all members",
			input: `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Doe/
0 @I2@ INDI
1 NAME Jane /Doe/
0 @I3@ INDI
1 NAME Child /Doe/
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
0 TRLR`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := decoder.Decode(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}

			v := New()
			errs := v.Validate(doc)

			if tt.expectError {
				if len(errs) == 0 {
					t.Fatal("Expected validation error but got none")
				}

				found := false
				for _, e := range errs {
					var valErr *ValidationError
					if errors.As(e, &valErr) && valErr.Code == tt.errorCode {
						found = true
						t.Logf("Found expected error: %v", e)
						break
					}
				}

				if !found {
					t.Errorf("Expected error code %q, got errors: %v", tt.errorCode, errs)
				}
			} else if len(errs) != 0 {
				t.Errorf("Expected no validation errors, got %d errors:", len(errs))
				for _, e := range errs {
					t.Logf("  - %v", e)
				}
			}
		})
	}
}

// Test backward compatibility of New()
func TestNewBackwardCompatibility(t *testing.T) {
	v := New()
	if v == nil {
		t.Fatal("New() returned nil")
	}

	// Should have default config
	if v.config == nil {
		t.Fatal("New() should set default config")
	}
	if v.config.Strictness != StrictnessNormal {
		t.Errorf("Default strictness = %v, want StrictnessNormal", v.config.Strictness)
	}
}

// Test NewWithConfig
func TestNewWithConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *ValidatorConfig
		want   Strictness
	}{
		{
			name:   "nil config uses defaults",
			config: nil,
			want:   StrictnessNormal,
		},
		{
			name:   "relaxed strictness",
			config: &ValidatorConfig{Strictness: StrictnessRelaxed},
			want:   StrictnessRelaxed,
		},
		{
			name:   "normal strictness",
			config: &ValidatorConfig{Strictness: StrictnessNormal},
			want:   StrictnessNormal,
		},
		{
			name:   "strict strictness",
			config: &ValidatorConfig{Strictness: StrictnessStrict},
			want:   StrictnessStrict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewWithConfig(tt.config)
			if v == nil {
				t.Fatal("NewWithConfig() returned nil")
			}
			if v.config.Strictness != tt.want {
				t.Errorf("Strictness = %v, want %v", v.config.Strictness, tt.want)
			}
		})
	}
}

// Test ValidateAll returns Issues
func TestValidateAllReturnsIssues(t *testing.T) {
	// Create document with date logic issue (death before birth)
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 1950
1 DEAT
2 DATE 1940
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	v := New()
	issues := v.ValidateAll(doc)

	// Should have at least one issue for death before birth
	if len(issues) == 0 {
		t.Fatal("Expected at least one issue")
	}

	found := false
	for _, issue := range issues {
		if issue.Code == CodeDeathBeforeBirth {
			found = true
			if issue.Severity != SeverityError {
				t.Errorf("DeathBeforeBirth should be SeverityError, got %v", issue.Severity)
			}
			if issue.RecordXRef != "@I1@" {
				t.Errorf("RecordXRef = %q, want @I1@", issue.RecordXRef)
			}
			break
		}
	}

	if !found {
		t.Error("Expected DEATH_BEFORE_BIRTH issue")
	}
}

// Test ValidateAll with nil document
func TestValidateAllNilDocument(t *testing.T) {
	v := New()
	issues := v.ValidateAll(nil)
	if issues != nil {
		t.Errorf("ValidateAll(nil) = %v, want nil", issues)
	}
}

// Test ValidateDateLogic
func TestValidateDateLogic(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME Old /Person/
1 BIRT
2 DATE 1800
1 DEAT
2 DATE 1950
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// With normal strictness, should get warning for impossible age
	v := New()
	issues := v.ValidateDateLogic(doc)

	found := false
	for _, issue := range issues {
		if issue.Code == CodeImpossibleAge {
			found = true
			if issue.Severity != SeverityWarning {
				t.Errorf("ImpossibleAge should be SeverityWarning, got %v", issue.Severity)
			}
			break
		}
	}

	if !found {
		t.Error("Expected IMPOSSIBLE_AGE warning")
	}
}

// Test ValidateDateLogic with nil document
func TestValidateDateLogicNilDocument(t *testing.T) {
	v := New()
	issues := v.ValidateDateLogic(nil)
	if issues != nil {
		t.Errorf("ValidateDateLogic(nil) = %v, want nil", issues)
	}
}

// Test FindOrphanedReferences
func TestFindOrphanedReferences(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Smith/
1 FAMS @F999@
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	v := New()
	issues := v.FindOrphanedReferences(doc)

	if len(issues) == 0 {
		t.Fatal("Expected orphaned reference issue")
	}

	found := false
	for _, issue := range issues {
		if issue.Code == CodeOrphanedFAMS {
			found = true
			if issue.Severity != SeverityError {
				t.Errorf("OrphanedFAMS should be SeverityError, got %v", issue.Severity)
			}
			if issue.RelatedXRef != "@F999@" {
				t.Errorf("RelatedXRef = %q, want @F999@", issue.RelatedXRef)
			}
			break
		}
	}

	if !found {
		t.Error("Expected ORPHANED_FAMS issue")
	}
}

// Test FindOrphanedReferences with nil document
func TestFindOrphanedReferencesNilDocument(t *testing.T) {
	v := New()
	issues := v.FindOrphanedReferences(nil)
	if issues != nil {
		t.Errorf("FindOrphanedReferences(nil) = %v, want nil", issues)
	}
}

// Test FindPotentialDuplicates
func TestFindPotentialDuplicates(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 1950
1 SEX M
0 @I2@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 1951
1 SEX M
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	v := New()
	duplicates := v.FindPotentialDuplicates(doc)

	if len(duplicates) == 0 {
		t.Fatal("Expected potential duplicate")
	}

	pair := duplicates[0]
	if pair.Individual1 == nil || pair.Individual2 == nil {
		t.Fatal("DuplicatePair individuals should not be nil")
	}
	if pair.Confidence < 0.7 {
		t.Errorf("Confidence = %v, want >= 0.7", pair.Confidence)
	}
	if len(pair.MatchReasons) == 0 {
		t.Error("MatchReasons should not be empty")
	}
}

// Test FindPotentialDuplicates with nil document
func TestFindPotentialDuplicatesNilDocument(t *testing.T) {
	v := New()
	duplicates := v.FindPotentialDuplicates(nil)
	if duplicates != nil {
		t.Errorf("FindPotentialDuplicates(nil) = %v, want nil", duplicates)
	}
}

// Test QualityReport
func TestQualityReport(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 1950
0 @I2@ INDI
1 NAME Jane /Doe/
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
0 @S1@ SOUR
1 TITL Test Source
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	v := New()
	report := v.QualityReport(doc)

	if report == nil {
		t.Fatal("QualityReport() returned nil")
	}

	// Check summary counts
	if report.TotalIndividuals != 2 {
		t.Errorf("TotalIndividuals = %d, want 2", report.TotalIndividuals)
	}
	if report.TotalFamilies != 1 {
		t.Errorf("TotalFamilies = %d, want 1", report.TotalFamilies)
	}
	if report.TotalSources != 1 {
		t.Errorf("TotalSources = %d, want 1", report.TotalSources)
	}

	// Check completeness metrics
	if report.IndividualsWithBirthDate != 1 {
		t.Errorf("IndividualsWithBirthDate = %d, want 1", report.IndividualsWithBirthDate)
	}
	if report.BirthDateCoverage != 0.5 {
		t.Errorf("BirthDateCoverage = %v, want 0.5", report.BirthDateCoverage)
	}

	// Check that completeness issues are generated
	if report.TotalIssues == 0 {
		t.Error("Expected completeness issues")
	}
}

// Test QualityReport with nil document
func TestQualityReportNilDocument(t *testing.T) {
	v := New()
	report := v.QualityReport(nil)

	if report == nil {
		t.Fatal("QualityReport(nil) should return empty report, not nil")
	}
	if report.TotalIndividuals != 0 {
		t.Errorf("TotalIndividuals = %d, want 0", report.TotalIndividuals)
	}
	if len(report.Errors) != 0 {
		t.Errorf("Errors = %v, want empty", report.Errors)
	}
}

// Test QualityReport JSON output
func TestQualityReportJSON(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Smith/
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	v := New()
	report := v.QualityReport(doc)

	jsonBytes, err := report.JSON()
	if err != nil {
		t.Fatalf("JSON() error = %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("JSON output not valid: %v", err)
	}

	// Check some expected keys
	if _, ok := parsed["total_individuals"]; !ok {
		t.Error("JSON missing total_individuals key")
	}
}

// Test Strictness filtering
func TestStrictnessFiltering(t *testing.T) {
	// Create document with issues of different severities
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME Old /Person/
1 BIRT
2 DATE 1800
1 DEAT
2 DATE 1950
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	tests := []struct {
		name       string
		strictness Strictness
		wantErrors bool
		wantWarns  bool
		wantInfo   bool
	}{
		{
			name:       "relaxed only shows errors",
			strictness: StrictnessRelaxed,
			wantErrors: true,
			wantWarns:  false,
			wantInfo:   false,
		},
		{
			name:       "normal shows errors and warnings",
			strictness: StrictnessNormal,
			wantErrors: true,
			wantWarns:  true,
			wantInfo:   false,
		},
		{
			name:       "strict shows all",
			strictness: StrictnessStrict,
			wantErrors: true,
			wantWarns:  true,
			wantInfo:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewWithConfig(&ValidatorConfig{Strictness: tt.strictness})

			// Use ValidateDateLogic which can return warnings (impossible age)
			issues := v.ValidateDateLogic(doc)

			hasWarnings := false
			for _, issue := range issues {
				if issue.Severity == SeverityWarning {
					hasWarnings = true
					break
				}
			}

			if hasWarnings != tt.wantWarns {
				t.Errorf("hasWarnings = %v, want %v", hasWarnings, tt.wantWarns)
			}
		})
	}
}

// Test configuration with custom DateLogic config
func TestConfigWithCustomDateLogic(t *testing.T) {
	// Create document with person who lived to 130 years
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME Old /Person/
1 BIRT
2 DATE 1800
1 DEAT
2 DATE 1930
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// With default config (max 120), should get warning
	v1 := New()
	issues1 := v1.ValidateDateLogic(doc)
	hasWarning1 := false
	for _, issue := range issues1 {
		if issue.Code == CodeImpossibleAge {
			hasWarning1 = true
			break
		}
	}
	if !hasWarning1 {
		t.Error("Default config should warn about 130 year lifespan")
	}

	// With custom config (max 140), should not get warning
	v2 := NewWithConfig(&ValidatorConfig{
		DateLogic: &DateLogicConfig{
			MaxReasonableAge: 140,
		},
	})
	issues2 := v2.ValidateDateLogic(doc)
	hasWarning2 := false
	for _, issue := range issues2 {
		if issue.Code == CodeImpossibleAge {
			hasWarning2 = true
			break
		}
	}
	if hasWarning2 {
		t.Error("Custom config with max 140 should not warn about 130 year lifespan")
	}
}

// Test configuration with custom Duplicates config
func TestConfigWithCustomDuplicates(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 1950
1 SEX M
0 @I2@ INDI
1 NAME Jon /Smith/
1 BIRT
2 DATE 1951
1 SEX M
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// With high similarity threshold, should not match
	config := &ValidatorConfig{
		Duplicates: &DuplicateConfig{
			RequireExactSurname: true,
			NormalizeNames:      true,
			MinNameSimilarity:   0.99, // Very high threshold
			MaxBirthYearDiff:    2,
			MinConfidence:       0.9,
		},
	}
	v := NewWithConfig(config)
	duplicates := v.FindPotentialDuplicates(doc)

	// John and Jon should not match with such high similarity threshold
	if len(duplicates) != 0 {
		t.Errorf("With high similarity threshold, expected no duplicates, got %d", len(duplicates))
	}
}

// Test lazy initialization of sub-validators
func TestLazyInitialization(t *testing.T) {
	v := New()

	// Initially, sub-validators should be nil
	if v.dateLogic != nil {
		t.Error("dateLogic should be nil initially")
	}
	if v.references != nil {
		t.Error("references should be nil initially")
	}
	if v.duplicates != nil {
		t.Error("duplicates should be nil initially")
	}
	if v.quality != nil {
		t.Error("quality should be nil initially")
	}

	// Create a minimal document
	doc := &gedcom.Document{}

	// After calling ValidateDateLogic, dateLogic should be initialized
	v.ValidateDateLogic(doc)
	if v.dateLogic == nil {
		t.Error("dateLogic should be initialized after ValidateDateLogic")
	}

	// Other validators should still be nil
	if v.references != nil {
		t.Error("references should still be nil")
	}
}

// Test that Validate() still works (backward compatibility)
func TestValidateBackwardCompatibility(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
0 @F1@ FAM
1 HUSB @I1@
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	v := New()
	errs := v.Validate(doc)

	// This should still return []error, not []Issue
	if len(errs) != 0 {
		t.Errorf("Expected no errors for valid file, got %d", len(errs))
	}

	// Test that returned errors implement error interface
	for _, err := range errs {
		_ = err.Error() // Should not panic
	}
}

// Test ValidateCustomTags method
func TestValidateCustomTags(t *testing.T) {
	t.Run("returns nil when no registry configured", func(t *testing.T) {
		v := New() // No registry configured
		doc := &gedcom.Document{
			Records: []*gedcom.Record{
				{
					XRef: "@I1@",
					Type: gedcom.RecordTypeIndividual,
					Tags: []*gedcom.Tag{
						{Level: 1, Tag: "_CUSTOM", Value: "value"},
					},
				},
			},
		}

		issues := v.ValidateCustomTags(doc)
		if issues != nil {
			t.Errorf("expected nil when no registry configured, got %d issues", len(issues))
		}
	})

	t.Run("returns nil for nil document", func(t *testing.T) {
		registry := NewTagRegistry()
		v := NewWithConfig(&ValidatorConfig{
			TagRegistry: registry,
		})

		issues := v.ValidateCustomTags(nil)
		if issues != nil {
			t.Errorf("expected nil for nil document, got %v", issues)
		}
	})

	t.Run("validates custom tags with registry", func(t *testing.T) {
		registry := NewTagRegistry()
		_ = registry.Register("_MILT", TagDefinition{
			Tag:            "_MILT",
			AllowedParents: []string{"INDI"},
		})

		v := NewWithConfig(&ValidatorConfig{
			TagRegistry:        registry,
			ValidateCustomTags: true,
		})

		doc := &gedcom.Document{
			Records: []*gedcom.Record{
				{
					XRef: "@F1@",
					Type: gedcom.RecordTypeFamily,
					Tags: []*gedcom.Tag{
						{Level: 1, Tag: "_MILT", Value: "Army"}, // Invalid: FAM is not INDI
					},
				},
			},
		}

		issues := v.ValidateCustomTags(doc)
		if len(issues) != 1 {
			t.Errorf("expected 1 issue, got %d", len(issues))
		}
		if len(issues) > 0 && issues[0].Code != CodeInvalidTagParent {
			t.Errorf("expected code %s, got %s", CodeInvalidTagParent, issues[0].Code)
		}
	})
}

// Test ValidateAll includes custom tag validation
func TestValidateAllWithCustomTags(t *testing.T) {
	registry := NewTagRegistry()
	_ = registry.Register("_MILT", TagDefinition{
		Tag:            "_MILT",
		AllowedParents: []string{"INDI"},
	})

	v := NewWithConfig(&ValidatorConfig{
		TagRegistry:        registry,
		ValidateCustomTags: true,
		Strictness:         StrictnessStrict,
	})

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{
				XRef: "@F1@",
				Type: gedcom.RecordTypeFamily,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "_MILT", Value: "Army"}, // Invalid parent
				},
			},
		},
	}

	issues := v.ValidateAll(doc)

	// Should find the invalid parent issue
	found := false
	for _, issue := range issues {
		if issue.Code == CodeInvalidTagParent {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected ValidateAll to include custom tag validation issues")
	}
}

// Test ValidatorConfig with TagRegistry
func TestValidatorConfigWithTagRegistry(t *testing.T) {
	registry := NewTagRegistry()

	config := &ValidatorConfig{
		TagRegistry:        registry,
		ValidateCustomTags: true,
	}

	v := NewWithConfig(config)
	if v.config.TagRegistry != registry {
		t.Error("expected TagRegistry to be set in config")
	}
	if !v.config.ValidateCustomTags {
		t.Error("expected ValidateCustomTags to be true")
	}
}
