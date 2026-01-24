package decoder

import (
	"context"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/parser"
)

// TestDecodeWithDiagnosticsBasic tests basic usage of DecodeWithDiagnostics
func TestDecodeWithDiagnosticsBasic(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
0 TRLR`

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}

	if result == nil {
		t.Fatal("DecodeWithDiagnostics() returned nil result")
	}

	if result.Document == nil {
		t.Fatal("DecodeWithDiagnostics() returned nil document")
	}

	if len(result.Diagnostics) != 0 {
		t.Errorf("Expected no diagnostics, got %d", len(result.Diagnostics))
	}

	// Verify document content
	if len(result.Document.Records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(result.Document.Records))
	}
}

// TestDecodeWithDiagnosticsLenientMode tests lenient parsing with errors
func TestDecodeWithDiagnosticsLenientMode(t *testing.T) {
	// Input with invalid lines mixed with valid lines
	input := `0 HEAD
1 GEDC
2 VERS 5.5
invalid line here
0 @I1@ INDI
1 NAME John /Smith/
another bad line
0 TRLR`

	opts := DefaultOptions()
	// StrictMode is false by default, enabling lenient parsing

	result, err := DecodeWithDiagnostics(strings.NewReader(input), opts)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}

	if result == nil {
		t.Fatal("DecodeWithDiagnostics() returned nil result")
	}

	// Should have collected diagnostics for the invalid lines
	if len(result.Diagnostics) < 2 {
		t.Errorf("Expected at least 2 diagnostics, got %d", len(result.Diagnostics))
	}

	// Should still have parsed the valid lines
	if result.Document == nil {
		t.Fatal("DecodeWithDiagnostics() returned nil document")
	}

	// Verify we got the individual record
	if len(result.Document.Records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(result.Document.Records))
	}

	// Verify diagnostics contain expected info
	for _, diag := range result.Diagnostics {
		if diag.Severity != SeverityError {
			t.Errorf("Expected SeverityError, got %v", diag.Severity)
		}
		if diag.Code == "" {
			t.Error("Diagnostic code should not be empty")
		}
		if diag.Line == 0 {
			t.Error("Diagnostic line number should not be 0")
		}
	}
}

// TestDecodeWithDiagnosticsStrictMode tests strict mode behavior
func TestDecodeWithDiagnosticsStrictMode(t *testing.T) {
	// Input with an invalid line
	input := `0 HEAD
1 GEDC
2 VERS 5.5
invalid line here
0 @I1@ INDI
0 TRLR`

	opts := &DecodeOptions{
		StrictMode: true,
	}

	result, err := DecodeWithDiagnostics(strings.NewReader(input), opts)

	// In strict mode, first error should cause failure
	if err == nil {
		t.Fatal("DecodeWithDiagnostics() expected error in strict mode")
	}

	// Result should be nil on error
	if result != nil {
		t.Error("DecodeWithDiagnostics() expected nil result on strict mode error")
	}
}

// TestDecodeWithDiagnosticsEmptyInput tests handling of empty input
func TestDecodeWithDiagnosticsEmptyInput(t *testing.T) {
	input := ""

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}

	if result == nil {
		t.Fatal("DecodeWithDiagnostics() returned nil result")
	}

	if result.Document == nil {
		t.Fatal("DecodeWithDiagnostics() returned nil document")
	}

	// Empty input is valid, no diagnostics expected
	if len(result.Diagnostics) != 0 {
		t.Errorf("Expected no diagnostics for empty input, got %d", len(result.Diagnostics))
	}
}

// TestDecodeWithDiagnosticsAllInvalidLines tests when all lines are invalid
func TestDecodeWithDiagnosticsAllInvalidLines(t *testing.T) {
	input := `not a valid line
also invalid
still invalid`

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)

	// When no valid lines can be parsed, an error should be returned
	if err == nil {
		t.Fatal("DecodeWithDiagnostics() expected error when all lines are invalid")
	}

	// But we should still get the result with diagnostics
	if result == nil {
		t.Fatal("DecodeWithDiagnostics() should return result with diagnostics even on error")
	}

	if len(result.Diagnostics) != 3 {
		t.Errorf("Expected 3 diagnostics, got %d", len(result.Diagnostics))
	}
}

// TestDecodeWithDiagnosticsErrorCodes tests error code classification
func TestDecodeWithDiagnosticsErrorCodes(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedCode string
	}{
		{
			name:         "empty line",
			input:        "0 HEAD\n\n0 TRLR",
			expectedCode: CodeEmptyLine,
		},
		{
			name:         "invalid level",
			input:        "0 HEAD\nXYZ TAG value\n0 TRLR",
			expectedCode: CodeInvalidLevel,
		},
		{
			name:         "syntax error - no tag",
			input:        "0 HEAD\n1\n0 TRLR",
			expectedCode: CodeSyntaxError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := DecodeWithDiagnostics(strings.NewReader(tt.input), nil)

			if result == nil || len(result.Diagnostics) == 0 {
				t.Fatal("Expected at least one diagnostic")
			}

			// Check that the expected code is present in diagnostics
			found := false
			for _, diag := range result.Diagnostics {
				if diag.Code == tt.expectedCode {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected diagnostic with code %s, got codes: %v",
					tt.expectedCode, result.Diagnostics)
			}
		})
	}
}

// TestDecodeWithDiagnosticsPreservesContext tests that context is preserved
func TestDecodeWithDiagnosticsPreservesContext(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
bad line with context
0 TRLR`

	result, _ := DecodeWithDiagnostics(strings.NewReader(input), nil)

	if result == nil || len(result.Diagnostics) == 0 {
		t.Fatal("Expected at least one diagnostic")
	}

	diag := result.Diagnostics[0]
	if diag.Context != "bad line with context" {
		t.Errorf("Expected context 'bad line with context', got %q", diag.Context)
	}

	// Line number should be 4 (1-indexed)
	if diag.Line != 4 {
		t.Errorf("Expected line number 4, got %d", diag.Line)
	}
}

// TestDecodeWithDiagnosticsHasErrors tests the HasErrors helper
func TestDecodeWithDiagnosticsHasErrors(t *testing.T) {
	input := `0 HEAD
invalid line
0 TRLR`

	result, _ := DecodeWithDiagnostics(strings.NewReader(input), nil)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if !result.Diagnostics.HasErrors() {
		t.Error("Expected HasErrors() to return true")
	}

	errorDiags := result.Diagnostics.Errors()
	if len(errorDiags) == 0 {
		t.Error("Expected at least one error diagnostic")
	}
}

// TestDecodeWithDiagnosticsNilOptions tests default behavior with nil options
func TestDecodeWithDiagnosticsNilOptions(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 TRLR`

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}

	if result == nil {
		t.Fatal("DecodeWithDiagnostics() returned nil result")
	}

	if result.Document == nil {
		t.Fatal("DecodeWithDiagnostics() returned nil document")
	}
}

// TestClassifyParseError tests the error classification function
func TestClassifyParseError(t *testing.T) {
	tests := []struct {
		message  string
		expected string
	}{
		{"empty line", CodeEmptyLine},
		{"Empty Line", CodeEmptyLine},
		{"invalid level number", CodeInvalidLevel},
		{"level cannot be negative", CodeInvalidLevel},
		{"invalid xref format", CodeInvalidXRef},
		{"line with xref must have a tag", CodeInvalidXRef},
		{"maximum nesting depth exceeded", CodeBadLevelJump},
		{"bad level jump", CodeBadLevelJump},
		{"some other error", CodeSyntaxError},
		{"line must have at least level and tag", CodeSyntaxError}, // Does not contain "invalid level"
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			result := classifyParseError(tt.message)
			if result != tt.expected {
				t.Errorf("classifyParseError(%q) = %q, want %q", tt.message, result, tt.expected)
			}
		})
	}
}

// TestDecodeWithDiagnosticsContextCancellation tests context cancellation
func TestDecodeWithDiagnosticsContextCancellation(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 TRLR`

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	opts := &DecodeOptions{
		Context: ctx,
	}

	result, err := DecodeWithDiagnostics(strings.NewReader(input), opts)

	if err == nil {
		t.Error("Expected error for cancelled context")
	}

	if result != nil {
		t.Error("Expected nil result for cancelled context")
	}

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

// TestBuildDocumentWithVersion tests the helper function for partial results
func TestBuildDocumentWithVersion(t *testing.T) {
	// Create parser lines manually to test the helper
	lines := []*parser.Line{
		{Level: 0, Tag: "HEAD", LineNumber: 1},
		{Level: 1, Tag: "GEDC", LineNumber: 2},
		{Level: 2, Tag: "VERS", Value: "5.5", LineNumber: 3},
		{Level: 0, Tag: "@I1@", Value: "", LineNumber: 4, XRef: "@I1@"},
		{Level: 0, Tag: "TRLR", LineNumber: 5},
	}

	// Fix: the parser Line stores XRef separately from Tag
	lines[3].Tag = "INDI"

	opts := DefaultOptions()
	doc := buildDocumentWithVersion(lines, opts)

	if doc == nil {
		t.Fatal("buildDocumentWithVersion() returned nil")
	}

	// Verify version was detected
	if doc.Header.Version != "5.5" {
		t.Errorf("Expected version 5.5, got %s", doc.Header.Version)
	}
}

// TestConvertParseErrors tests the parse error conversion function
func TestConvertParseErrors(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result := convertParseErrors(nil)
		if result != nil {
			t.Errorf("Expected nil, got %v", result)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		result := convertParseErrors([]*parser.ParseError{})
		if result != nil {
			t.Errorf("Expected nil, got %v", result)
		}
	})

	t.Run("single error", func(t *testing.T) {
		errors := []*parser.ParseError{
			{
				Line:    5,
				Message: "empty line",
				Context: "",
			},
		}
		result := convertParseErrors(errors)
		if len(result) != 1 {
			t.Fatalf("Expected 1 diagnostic, got %d", len(result))
		}
		if result[0].Line != 5 {
			t.Errorf("Expected line 5, got %d", result[0].Line)
		}
		if result[0].Code != CodeEmptyLine {
			t.Errorf("Expected code %s, got %s", CodeEmptyLine, result[0].Code)
		}
		if result[0].Severity != SeverityError {
			t.Errorf("Expected SeverityError, got %v", result[0].Severity)
		}
	})
}

// TestEntityLevelDiagnosticsUnknownTag tests that unknown tags generate diagnostics.
func TestEntityLevelDiagnosticsUnknownTag(t *testing.T) {
	// Input with an unknown tag on an individual
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
1 UNKNOWNTAG some value
0 TRLR`

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}

	if result == nil {
		t.Fatal("DecodeWithDiagnostics() returned nil result")
	}

	// Should have collected a diagnostic for the unknown tag
	if len(result.Diagnostics) < 1 {
		t.Fatal("Expected at least 1 diagnostic for unknown tag")
	}

	// Find the unknown tag diagnostic
	found := false
	for _, diag := range result.Diagnostics {
		if diag.Code == CodeUnknownTag {
			found = true
			if diag.Severity != SeverityWarning {
				t.Errorf("Expected SeverityWarning, got %v", diag.Severity)
			}
			if !strings.Contains(diag.Message, "UNKNOWNTAG") {
				t.Errorf("Expected message to mention UNKNOWNTAG, got: %s", diag.Message)
			}
			break
		}
	}
	if !found {
		t.Errorf("Expected diagnostic with code %s, got: %v", CodeUnknownTag, result.Diagnostics)
	}

	// Document should still be valid
	if len(result.Document.Records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(result.Document.Records))
	}
}

// TestEntityLevelDiagnosticsVendorExtensionNotWarned tests that vendor extensions (_prefix) don't generate warnings.
func TestEntityLevelDiagnosticsVendorExtensionNotWarned(t *testing.T) {
	// Input with a vendor extension tag (starts with _)
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
1 _CUSTOMTAG vendor extension value
0 TRLR`

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}

	// Should NOT have any diagnostics for vendor extension tags
	for _, diag := range result.Diagnostics {
		if diag.Code == CodeUnknownTag {
			t.Errorf("Vendor extension tag should not generate UNKNOWN_TAG diagnostic: %v", diag)
		}
	}

	// Document should be valid
	if len(result.Document.Records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(result.Document.Records))
	}
}

// TestEntityLevelDiagnosticsInvalidValue tests that invalid values generate diagnostics.
func TestEntityLevelDiagnosticsInvalidValue(t *testing.T) {
	// Input with an invalid QUAY value (should be 0-3)
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
1 SOUR @S1@
2 QUAY invalid
0 TRLR`

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}

	// Should have collected a diagnostic for the invalid QUAY value
	found := false
	for _, diag := range result.Diagnostics {
		if diag.Code == CodeInvalidValue {
			found = true
			if diag.Severity != SeverityWarning {
				t.Errorf("Expected SeverityWarning, got %v", diag.Severity)
			}
			if !strings.Contains(diag.Message, "QUAY") {
				t.Errorf("Expected message to mention QUAY, got: %s", diag.Message)
			}
			break
		}
	}
	if !found {
		t.Errorf("Expected diagnostic with code %s, got: %v", CodeInvalidValue, result.Diagnostics)
	}
}

// TestEntityLevelDiagnosticsStrictModeNoCollection tests that strict mode doesn't collect entity diagnostics.
func TestEntityLevelDiagnosticsStrictModeNoCollection(t *testing.T) {
	// Input with an unknown tag (but valid syntax)
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
1 UNKNOWNTAG some value
0 TRLR`

	opts := &DecodeOptions{
		StrictMode: true,
	}

	result, err := DecodeWithDiagnostics(strings.NewReader(input), opts)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}

	// Strict mode should not collect entity-level diagnostics
	// (only parser-level errors would cause it to fail)
	if len(result.Diagnostics) != 0 {
		t.Errorf("Expected no diagnostics in strict mode for valid syntax, got %d", len(result.Diagnostics))
	}
}

// TestDiagnosticCollectorNilSafe tests that nil collector doesn't cause panics.
func TestDiagnosticCollectorNilSafe(t *testing.T) {
	var collector *diagnosticCollector

	// These should not panic
	collector.add(Diagnostic{})
	collector.addUnknownTag(1, "TAG", "value")
	collector.addInvalidValue(1, "TAG", "value", "reason")

	// Verify no diagnostics were added (collector is nil)
	if collector != nil {
		t.Error("Collector should remain nil")
	}
}

// TestEntityDiagnosticsMergedWithParserDiagnostics tests that both types are merged.
func TestEntityDiagnosticsMergedWithParserDiagnostics(t *testing.T) {
	// Input with both parser error AND unknown entity tag
	input := `0 HEAD
1 GEDC
2 VERS 5.5
invalid parser line
0 @I1@ INDI
1 NAME John /Smith/
1 UNKNOWNTAG value
0 TRLR`

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}

	// Should have diagnostics from both parser (syntax error) and entity (unknown tag)
	hasParserError := false
	hasEntityWarning := false
	for _, diag := range result.Diagnostics {
		if diag.Severity == SeverityError {
			hasParserError = true
		}
		if diag.Code == CodeUnknownTag {
			hasEntityWarning = true
		}
	}

	if !hasParserError {
		t.Error("Expected parser-level error diagnostic")
	}
	if !hasEntityWarning {
		t.Error("Expected entity-level warning diagnostic")
	}
}
