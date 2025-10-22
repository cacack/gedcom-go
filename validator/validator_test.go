package validator

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/decoder"
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
	errors := v.Validate(doc)

	if len(errors) == 0 {
		t.Fatal("Expected validation errors for broken XRef")
	}

	found := false
	for _, err := range errors {
		if strings.Contains(err.Error(), "BROKEN_XREF") {
			found = true
			t.Logf("Found expected error: %v", err)
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
	errors := v.Validate(doc)

	if len(errors) == 0 {
		t.Fatal("Expected validation errors for missing NAME")
	}

	found := false
	for _, err := range errors {
		if strings.Contains(err.Error(), "MISSING_REQUIRED_FIELD") {
			found = true
			t.Logf("Found expected error: %v", err)
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
	errors := v.Validate(doc)

	if len(errors) != 0 {
		t.Errorf("Expected no validation errors for valid file, got %d errors:", len(errors))
		for _, err := range errors {
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
			errors := v.Validate(doc)

			if tt.expectError {
				if len(errors) == 0 {
					t.Fatal("Expected validation error but got none")
				}

				found := false
				for _, err := range errors {
					if strings.Contains(err.Error(), tt.errorCode) {
						found = true
						t.Logf("Found expected error: %v", err)
						break
					}
				}

				if !found {
					t.Errorf("Expected error code %q, got errors: %v", tt.errorCode, errors)
				}
			} else if len(errors) != 0 {
				t.Errorf("Expected no validation errors, got %d errors:", len(errors))
				for _, err := range errors {
					t.Logf("  - %v", err)
				}
			}
		})
	}
}
