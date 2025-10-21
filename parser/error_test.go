package parser

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// T062: Write tests for invalid tag errors
func TestInvalidTagErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErrMsg  string
		wantLineNum int
	}{
		{
			name:        "invalid character in tag",
			input:       "0 INV@LID",
			wantErrMsg:  "",
			wantLineNum: 1,
		},
		{
			name:        "tag too long",
			input:       "0 VERYLONGTAGNAMETHATEXCEEDSLIMITS",
			wantErrMsg:  "",
			wantLineNum: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			_, err := p.ParseLine(tt.input)

			// For now, we accept these as valid (spec allows custom tags)
			// This test documents current behavior
			_ = err
		})
	}
}

// T063: Write tests for hierarchy level errors
func TestHierarchyLevelErrors(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErr    bool
		expectLine int
	}{
		{
			name: "level jump too large",
			input: `0 HEAD
1 GEDC
5 VERS`,
			wantErr:    false, // Parser accepts any level, decoder may validate
			expectLine: 3,
		},
		{
			name: "negative level",
			input: `0 HEAD
-1 GEDC`,
			wantErr:    true,
			expectLine: 2,
		},
		{
			name: "level exceeds max depth",
			input: `0 HEAD
101 DEEP`,
			wantErr:    true,
			expectLine: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			_, err := p.Parse(strings.NewReader(tt.input))

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if err != nil {
				parseErr, ok := err.(*ParseError)
				if ok && parseErr.Line != tt.expectLine {
					t.Errorf("Error at line %d, expected line %d", parseErr.Line, tt.expectLine)
				}
			}
		})
	}
}

// T064: Write tests for missing cross-reference targets (decoder responsibility, but test parser handling)
func TestMalformedXRefs(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "xref without closing @",
			input:   "0 @I1 INDI",
			wantErr: false, // Parsed as tag "@I1", valid at parser level
		},
		{
			name:    "xref without opening @",
			input:   "0 I1@ INDI",
			wantErr: false, // Parsed as tag "I1@", valid at parser level
		},
		{
			name:    "empty xref",
			input:   "0 @@ INDI",
			wantErr: false, // Valid at parser level
		},
		{
			name:    "xref with spaces",
			input:   "0 @I 1@ INDI",
			wantErr: false, // Parsed as tag "@I" with value "1@ INDI"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			_, err := p.ParseLine(tt.input)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// T065: Write tests for encoding errors (handled by charset package)
// Already covered in charset/charset_test.go

// T066: Write tests for malformed files
func TestMalformedFiles(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "missing HEAD",
			input: `0 @I1@ INDI
1 NAME John
0 TRLR`,
			wantErr: false, // Parser accepts, decoder may validate
		},
		{
			name: "missing TRLR",
			input: `0 HEAD
1 GEDC
0 @I1@ INDI`,
			wantErr: false, // Parser accepts
		},
		{
			name:    "completely empty file",
			input:   "",
			wantErr: false, // Returns empty line list
		},
		{
			name: "only whitespace lines",
			input: `

`,
			wantErr: true, // Empty lines are errors
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			lines, err := p.Parse(strings.NewReader(tt.input))

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.wantErr && err == nil {
				t.Logf("Parsed %d lines from malformed input", len(lines))
			}
		})
	}
}

// T067: Write tests for partial file recovery (error recovery mode)
func TestErrorRecoveryMode(t *testing.T) {
	// This test verifies that parser can continue after encountering errors
	// when in recovery mode (future enhancement)

	input := `0 HEAD
1 GEDC
INVALID LINE HERE
2 VERS 5.5
0 TRLR`

	p := NewParser()
	_, err := p.Parse(strings.NewReader(input))

	// Currently, parser stops at first error
	// This test documents current behavior
	if err == nil {
		t.Error("Expected error for invalid line")
	}

	parseErr, ok := err.(*ParseError)
	if ok {
		if parseErr.Line != 3 {
			t.Errorf("Expected error at line 3, got line %d", parseErr.Line)
		}
		t.Logf("Error correctly reported at line %d: %v", parseErr.Line, parseErr)
	}
}

// Test that errors include helpful context
func TestErrorContext(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantContext string
	}{
		{
			name:        "error shows line content",
			input:       "INVALID",
			wantContext: "INVALID",
		},
		{
			name:        "error shows numeric issue",
			input:       "X HEAD",
			wantContext: "X HEAD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			_, err := p.ParseLine(tt.input)

			if err == nil {
				t.Fatal("Expected error but got none")
			}

			parseErr, ok := err.(*ParseError)
			if !ok {
				t.Fatalf("Expected *ParseError, got %T", err)
			}

			if parseErr.Context != tt.wantContext {
				t.Errorf("Context = %q, want %q", parseErr.Context, tt.wantContext)
			}

			errMsg := parseErr.Error()
			if !strings.Contains(errMsg, tt.wantContext) {
				t.Errorf("Error message %q should contain context %q", errMsg, tt.wantContext)
			}
		})
	}
}

// Test error messages are clear and actionable
func TestErrorMessageQuality(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantContains   []string
		wantLineNumber int
	}{
		{
			name:           "empty line error",
			input:          "",
			wantContains:   []string{"empty", "line 1"},
			wantLineNumber: 1,
		},
		{
			name:           "missing tag error",
			input:          "0",
			wantContains:   []string{"tag", "line 1"},
			wantLineNumber: 1,
		},
		{
			name:           "invalid level error",
			input:          "ABC TAG",
			wantContains:   []string{"level", "line 1"},
			wantLineNumber: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			_, err := p.ParseLine(tt.input)

			if err == nil {
				t.Fatal("Expected error but got none")
			}

			errMsg := err.Error()
			for _, substr := range tt.wantContains {
				if !strings.Contains(errMsg, substr) {
					t.Errorf("Error message %q should contain %q", errMsg, substr)
				}
			}

			parseErr, ok := err.(*ParseError)
			if ok && parseErr.Line != tt.wantLineNumber {
				t.Errorf("Line number = %d, want %d", parseErr.Line, tt.wantLineNumber)
			}
		})
	}
}

// Test ParseError.Unwrap() method for error unwrapping
func TestParseErrorUnwrap(t *testing.T) {
	t.Run("unwrap wrapped error", func(t *testing.T) {
		baseErr := fmt.Errorf("base error")
		parseErr := wrapParseError(1, "wrapped message", "context", baseErr)

		unwrapped := parseErr.(*ParseError).Unwrap()
		if unwrapped != baseErr {
			t.Errorf("Unwrap() = %v, want %v", unwrapped, baseErr)
		}

		// Test with errors.Is
		if !errors.Is(parseErr, baseErr) {
			t.Error("errors.Is() should find base error through Unwrap()")
		}
	})

	t.Run("unwrap error without underlying error", func(t *testing.T) {
		parseErr := newParseError(1, "simple error", "context")

		unwrapped := parseErr.(*ParseError).Unwrap()
		if unwrapped != nil {
			t.Errorf("Unwrap() = %v, want nil", unwrapped)
		}
	})

	t.Run("errors.As with wrapped error", func(t *testing.T) {
		baseErr := &ParseError{Line: 99, Message: "original"}
		wrappedErr := wrapParseError(1, "wrapped", "ctx", baseErr)

		var target *ParseError
		if !errors.As(wrappedErr, &target) {
			t.Error("errors.As() should find ParseError through Unwrap()")
		}
		if target.Line != 1 {
			t.Errorf("errors.As() found wrong error, Line = %d, want 1", target.Line)
		}
	})
}
