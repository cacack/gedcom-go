package parser

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/iotest"
)

// T026: Write table-driven tests for line parsing (valid lines, edge cases, line endings)
func TestParseLine(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantLevel   int
		wantTag     string
		wantValue   string
		wantXRef    string
		wantErr     bool
		description string
	}{
		{
			name:        "simple level 0 tag",
			input:       "0 HEAD",
			wantLevel:   0,
			wantTag:     "HEAD",
			wantValue:   "",
			wantXRef:    "",
			wantErr:     false,
			description: "Basic header tag",
		},
		{
			name:        "level 0 with xref",
			input:       "0 @I1@ INDI",
			wantLevel:   0,
			wantTag:     "INDI",
			wantValue:   "",
			wantXRef:    "@I1@",
			wantErr:     false,
			description: "Individual record with cross-reference",
		},
		{
			name:        "level 1 with value",
			input:       "1 NAME John /Smith/",
			wantLevel:   1,
			wantTag:     "NAME",
			wantValue:   "John /Smith/",
			wantXRef:    "",
			wantErr:     false,
			description: "Name tag with value",
		},
		{
			name:        "level 2 with value",
			input:       "2 GIVN John",
			wantLevel:   2,
			wantTag:     "GIVN",
			wantValue:   "John",
			wantXRef:    "",
			wantErr:     false,
			description: "Given name tag",
		},
		{
			name:        "CRLF line ending",
			input:       "0 HEAD\r\n",
			wantLevel:   0,
			wantTag:     "HEAD",
			wantValue:   "",
			wantXRef:    "",
			wantErr:     false,
			description: "Windows line ending",
		},
		{
			name:        "LF line ending",
			input:       "0 HEAD\n",
			wantLevel:   0,
			wantTag:     "HEAD",
			wantValue:   "",
			wantXRef:    "",
			wantErr:     false,
			description: "Unix line ending",
		},
		{
			name:        "CR line ending",
			input:       "0 HEAD\r",
			wantLevel:   0,
			wantTag:     "HEAD",
			wantValue:   "",
			wantXRef:    "",
			wantErr:     false,
			description: "Old Mac line ending",
		},
		{
			name:        "value with spaces",
			input:       "1 NOTE This is a note with spaces",
			wantLevel:   1,
			wantTag:     "NOTE",
			wantValue:   "This is a note with spaces",
			wantXRef:    "",
			wantErr:     false,
			description: "Note with multiple spaces",
		},
		{
			name:        "value with pointer",
			input:       "1 HUSB @I1@",
			wantLevel:   1,
			wantTag:     "HUSB",
			wantValue:   "@I1@",
			wantXRef:    "",
			wantErr:     false,
			description: "Pointer to husband",
		},
		{
			name:        "empty value",
			input:       "1 SEX",
			wantLevel:   1,
			wantTag:     "SEX",
			wantValue:   "",
			wantXRef:    "",
			wantErr:     false,
			description: "Tag with no value",
		},
		{
			name:        "invalid - no tag",
			input:       "0",
			wantErr:     true,
			description: "Missing tag",
		},
		{
			name:        "invalid - negative level",
			input:       "-1 HEAD",
			wantErr:     true,
			description: "Negative level number",
		},
		{
			name:        "invalid - non-numeric level",
			input:       "X HEAD",
			wantErr:     true,
			description: "Non-numeric level",
		},
		{
			name:        "empty line",
			input:       "",
			wantErr:     true,
			description: "Empty line should error",
		},
		{
			name:        "whitespace only",
			input:       "   ",
			wantErr:     true,
			description: "Whitespace only should error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			line, err := p.ParseLine(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseLine() expected error but got none")
				}
				var parseErr *ParseError
				if !errors.As(err, &parseErr) {
					t.Errorf("expected *ParseError, got %T", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseLine() unexpected error: %v", err)
			}

			if line.Level != tt.wantLevel {
				t.Errorf("Level = %d, want %d", line.Level, tt.wantLevel)
			}
			if line.Tag != tt.wantTag {
				t.Errorf("Tag = %q, want %q", line.Tag, tt.wantTag)
			}
			if line.Value != tt.wantValue {
				t.Errorf("Value = %q, want %q", line.Value, tt.wantValue)
			}
			if line.XRef != tt.wantXRef {
				t.Errorf("XRef = %q, want %q", line.XRef, tt.wantXRef)
			}
		})
	}
}

// T027-T029: Write tests for GEDCOM version parsing with sample files
func TestParseGEDCOM55(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
0 TRLR`

	p := NewParser()
	lines, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(lines) == 0 {
		t.Fatal("Parse() returned no lines")
	}

	// Verify structure
	if lines[0].Level != 0 || lines[0].Tag != "HEAD" {
		t.Errorf("Expected HEAD tag, got level=%d tag=%s", lines[0].Level, lines[0].Tag)
	}
}

func TestParseGEDCOM551(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME Jane /Doe/
0 TRLR`

	p := NewParser()
	lines, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(lines) == 0 {
		t.Fatal("Parse() returned no lines")
	}
}

func TestParseGEDCOM70(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME John Smith
0 TRLR`

	p := NewParser()
	lines, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(lines) == 0 {
		t.Fatal("Parse() returned no lines")
	}
}

// Test line number tracking
func TestLineNumberTracking(t *testing.T) {
	input := `0 HEAD
1 SOUR Test
2 VERS 1.0
0 TRLR`

	p := NewParser()
	lines, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Verify line numbers are tracked correctly
	for i, line := range lines {
		expectedLineNum := i + 1
		if line.LineNumber != expectedLineNum {
			t.Errorf("Line %d: LineNumber = %d, want %d", i, line.LineNumber, expectedLineNum)
		}
	}
}

// Test nesting depth checking
func TestMaxNestingDepth(t *testing.T) {
	// Build input with >100 levels (should fail)
	var input strings.Builder
	for i := 0; i <= 101; i++ {
		input.WriteString(strings.Repeat(" ", i))
		input.WriteString("TAG\n")
	}

	// This test will verify that max depth checking works
	// Implementation should add depth checking
	p := NewParser()
	_, err := p.Parse(strings.NewReader(input.String()))

	// We expect this to eventually fail when depth checking is implemented
	_ = err // For now, just parse it
}

// Test ParseLine with tag at end of line (no value)
func TestParseLineTagAtEnd(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"level and tag only", "0 HEAD"},
		{"with xref, tag at end", "0 @I1@ INDI"},
		{"level 1 tag only", "1 BIRT"},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p.Reset()
			line, err := p.ParseLine(tt.input)
			if err != nil {
				t.Fatalf("ParseLine() error = %v", err)
			}
			if line.Value != "" {
				t.Errorf("Expected empty value, got %q", line.Value)
			}
		})
	}
}

// Test Parse with scanner error
func TestParseScannerError(t *testing.T) {
	// Use a reader that always returns an error
	testErr := fmt.Errorf("simulated read error")
	reader := iotest.ErrReader(testErr)

	p := NewParser()
	_, err := p.Parse(reader)

	// Should get an error from the scanner
	if err == nil {
		t.Fatal("Expected error from Parse with failing reader")
	}
	if !errors.Is(err, testErr) {
		t.Errorf("expected error to wrap simulated error, got %v", err)
	}
}

// Test ParseLine value preservation
func TestParseLineValueSpacing(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "single space before value",
			input: "1 NAME John",
			want:  "John",
		},
		{
			name:  "multiple spaces before value",
			input: "1 NAME    John",
			want:  "John",
		},
		{
			name:  "value with internal spaces",
			input: "1 NAME John  Smith",
			want:  "John  Smith",
		},
		{
			name:  "value with trailing spaces preserved",
			input: "1 NAME John  ",
			want:  "John  ",
		},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p.Reset()
			line, err := p.ParseLine(tt.input)
			if err != nil {
				t.Fatalf("ParseLine() error = %v", err)
			}
			if line.Value != tt.want {
				t.Errorf("Value = %q, want %q", line.Value, tt.want)
			}
		})
	}
}

// Test edge cases in value extraction
func TestParseLineValueExtraction(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValue string
		wantTag   string
	}{
		{
			name:      "tag at exact end of line",
			input:     "0 HEAD",
			wantValue: "",
			wantTag:   "HEAD",
		},
		{
			name:      "single word becomes tag only",
			input:     "1 NAMEJOHN",
			wantValue: "",
			wantTag:   "NAMEJOHN",
		},
		{
			name:      "xref with value",
			input:     "0 @I1@ INDI",
			wantValue: "",
			wantTag:   "INDI",
		},
		{
			name:      "xref with tag and value",
			input:     "0 @I1@ NOTE This is a note",
			wantValue: "This is a note",
			wantTag:   "NOTE",
		},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p.Reset()
			line, err := p.ParseLine(tt.input)
			if err != nil {
				t.Fatalf("ParseLine() error = %v", err)
			}
			if line.Value != tt.wantValue {
				t.Errorf("Value = %q, want %q", line.Value, tt.wantValue)
			}
			if line.Tag != tt.wantTag {
				t.Errorf("Tag = %q, want %q", line.Tag, tt.wantTag)
			}
		})
	}
}

// TestParseCROnlyLineEndings tests parsing of files with CR-only line endings (old Mac style)
func TestParseCROnlyLineEndings(t *testing.T) {
	p := NewParser()

	// CR-only line endings (old Macintosh style)
	input := "0 HEAD\r1 SOUR Test\r1 GEDC\r2 VERS 5.5\r0 TRLR\r"
	lines, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	expected := []struct {
		level int
		tag   string
	}{
		{0, "HEAD"},
		{1, "SOUR"},
		{1, "GEDC"},
		{2, "VERS"},
		{0, "TRLR"},
	}

	if len(lines) != len(expected) {
		t.Fatalf("Got %d lines, want %d", len(lines), len(expected))
	}

	for i, e := range expected {
		if lines[i].Level != e.level {
			t.Errorf("Line %d: Level = %d, want %d", i, lines[i].Level, e.level)
		}
		if lines[i].Tag != e.tag {
			t.Errorf("Line %d: Tag = %q, want %q", i, lines[i].Tag, e.tag)
		}
	}
}

// TestParseMixedLineEndings tests parsing of files with mixed line ending styles
func TestParseMixedLineEndings(t *testing.T) {
	p := NewParser()

	// Mixed: CRLF, LF, CR
	input := "0 HEAD\r\n1 SOUR Test\n1 GEDC\r2 VERS 5.5\r\n0 TRLR\n"
	lines, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(lines) != 5 {
		t.Fatalf("Got %d lines, want 5", len(lines))
	}

	tags := []string{"HEAD", "SOUR", "GEDC", "VERS", "TRLR"}
	for i, tag := range tags {
		if lines[i].Tag != tag {
			t.Errorf("Line %d: Tag = %q, want %q", i, lines[i].Tag, tag)
		}
	}
}

// TestParseCRAtEndNeedsMoreData tests the edge case where CR is at buffer boundary
func TestParseCRAtEndNeedsMoreData(t *testing.T) {
	p := NewParser()

	// This tests the case where CR might be at the end of a read buffer
	// and we need to determine if it's followed by LF
	input := "0 HEAD\r\n1 NAME Test\r\n0 TRLR\r\n"
	lines, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(lines) != 3 {
		t.Fatalf("Got %d lines, want 3", len(lines))
	}
}

// TestParseEmptyLines tests handling of empty input
func TestParseEmptyLines(t *testing.T) {
	p := NewParser()

	lines, err := p.Parse(strings.NewReader(""))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(lines) != 0 {
		t.Errorf("Got %d lines for empty input, want 0", len(lines))
	}
}

// TestParseFinalLineWithoutNewline tests file ending without newline
func TestParseFinalLineWithoutNewline(t *testing.T) {
	p := NewParser()

	// No newline at end
	input := "0 HEAD\n1 SOUR Test\n0 TRLR"
	lines, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(lines) != 3 {
		t.Fatalf("Got %d lines, want 3", len(lines))
	}

	if lines[2].Tag != "TRLR" {
		t.Errorf("Last line Tag = %q, want TRLR", lines[2].Tag)
	}
}

// Test XRef without tag error
func TestParseLineXRefWithoutTag(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "xref only",
			input: "0 @I1@",
		},
		{
			name:  "xref with newline",
			input: "0 @I1@\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p.Reset()
			_, err := p.ParseLine(tt.input)
			if err == nil {
				t.Fatal("Expected error for xref without tag")
			}
			var parseErr *ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("expected *ParseError, got %T", err)
			}
			if !strings.Contains(parseErr.Message, "xref must have a tag") {
				t.Errorf("expected 'xref must have a tag' in Message, got: %q", parseErr.Message)
			}
		})
	}
}

// --- ParseWithOptions Tests ---

// TestParseWithOptions_StrictMode verifies that strict mode (Lenient=false) behaves like Parse()
func TestParseWithOptions_StrictMode(t *testing.T) {
	p := NewParser()

	input := `0 HEAD
INVALID LINE
0 TRLR`

	// With nil options (defaults to strict)
	lines, parseErrors, fatalErr := p.ParseWithOptions(strings.NewReader(input), nil)
	if fatalErr == nil {
		t.Error("Expected fatal error in strict mode with nil options")
	}
	if len(lines) != 0 {
		t.Errorf("Expected no lines in strict mode with error, got %d", len(lines))
	}
	if len(parseErrors) != 0 {
		t.Errorf("Expected no parse errors slice in strict mode, got %d", len(parseErrors))
	}

	// With explicit Lenient=false
	p.Reset()
	opts := &ParseOptions{Lenient: false}
	lines, parseErrors, fatalErr = p.ParseWithOptions(strings.NewReader(input), opts)
	if fatalErr == nil {
		t.Error("Expected fatal error in strict mode")
	}
	if len(lines) != 0 {
		t.Errorf("Expected no lines in strict mode with error, got %d", len(lines))
	}
	if len(parseErrors) != 0 {
		t.Errorf("Expected no parse errors in strict mode, got %d", len(parseErrors))
	}
}

// TestParseWithOptions_LenientMode verifies that lenient mode collects errors and continues
func TestParseWithOptions_LenientMode(t *testing.T) {
	p := NewParser()

	input := `0 HEAD
1 SOUR Test
INVALID LINE
2 VERS 1.0
another bad line
0 TRLR`

	opts := &ParseOptions{Lenient: true}
	lines, parseErrors, fatalErr := p.ParseWithOptions(strings.NewReader(input), opts)

	// No fatal error
	if fatalErr != nil {
		t.Fatalf("Unexpected fatal error: %v", fatalErr)
	}

	// Should have 4 valid lines (HEAD, SOUR, VERS, TRLR)
	if len(lines) != 4 {
		t.Errorf("Expected 4 valid lines, got %d", len(lines))
	}

	// Should have 2 parse errors
	if len(parseErrors) != 2 {
		t.Errorf("Expected 2 parse errors, got %d", len(parseErrors))
	}

	// Verify line numbers in errors
	if len(parseErrors) >= 1 && parseErrors[0].Line != 3 {
		t.Errorf("First error should be at line 3, got line %d", parseErrors[0].Line)
	}
	if len(parseErrors) >= 2 && parseErrors[1].Line != 5 {
		t.Errorf("Second error should be at line 5, got line %d", parseErrors[1].Line)
	}

	// Verify the valid lines
	expectedTags := []string{"HEAD", "SOUR", "VERS", "TRLR"}
	for i, tag := range expectedTags {
		if lines[i].Tag != tag {
			t.Errorf("Line %d: expected tag %s, got %s", i, tag, lines[i].Tag)
		}
	}
}

// TestParseWithOptions_MaxErrors verifies that MaxErrors limit is respected
func TestParseWithOptions_MaxErrors(t *testing.T) {
	p := NewParser()

	// Input with 5 invalid lines
	input := `0 HEAD
invalid1
invalid2
invalid3
invalid4
invalid5
0 TRLR`

	opts := &ParseOptions{
		Lenient:   true,
		MaxErrors: 2,
	}
	lines, parseErrors, fatalErr := p.ParseWithOptions(strings.NewReader(input), opts)

	// No fatal error
	if fatalErr != nil {
		t.Fatalf("Unexpected fatal error: %v", fatalErr)
	}

	// Should only collect 2 errors (the limit)
	if len(parseErrors) != 2 {
		t.Errorf("Expected 2 parse errors (MaxErrors limit), got %d", len(parseErrors))
	}

	// Should still parse valid lines (HEAD and TRLR)
	if len(lines) != 2 {
		t.Errorf("Expected 2 valid lines, got %d", len(lines))
	}
}

// TestParseWithOptions_MaxErrorsZero verifies that MaxErrors=0 means unlimited
func TestParseWithOptions_MaxErrorsZero(t *testing.T) {
	p := NewParser()

	// Input with many invalid lines
	input := `0 HEAD
err1
err2
err3
err4
err5
err6
err7
err8
err9
err10
0 TRLR`

	opts := &ParseOptions{
		Lenient:   true,
		MaxErrors: 0, // Unlimited
	}
	lines, parseErrors, fatalErr := p.ParseWithOptions(strings.NewReader(input), opts)

	if fatalErr != nil {
		t.Fatalf("Unexpected fatal error: %v", fatalErr)
	}

	// Should collect all 10 errors
	if len(parseErrors) != 10 {
		t.Errorf("Expected 10 parse errors with unlimited, got %d", len(parseErrors))
	}

	// Should still parse valid lines
	if len(lines) != 2 {
		t.Errorf("Expected 2 valid lines, got %d", len(lines))
	}
}

// TestParseWithOptions_NegativeMaxErrors verifies that negative MaxErrors is treated as unlimited
func TestParseWithOptions_NegativeMaxErrors(t *testing.T) {
	p := NewParser()

	// Input with multiple invalid lines
	input := `0 HEAD
err1
err2
err3
0 TRLR`

	opts := &ParseOptions{
		Lenient:   true,
		MaxErrors: -5, // Negative should be normalized to unlimited
	}
	lines, parseErrors, fatalErr := p.ParseWithOptions(strings.NewReader(input), opts)

	if fatalErr != nil {
		t.Fatalf("Unexpected fatal error: %v", fatalErr)
	}

	// Should collect all 3 errors (negative treated as unlimited)
	if len(parseErrors) != 3 {
		t.Errorf("Expected 3 parse errors with negative MaxErrors (treated as unlimited), got %d", len(parseErrors))
	}

	// Should still parse valid lines
	if len(lines) != 2 {
		t.Errorf("Expected 2 valid lines, got %d", len(lines))
	}
}

// TestParseWithOptions_IOError verifies that I/O errors are returned as fatalErr
func TestParseWithOptions_IOError(t *testing.T) {
	p := NewParser()

	// Use a reader that always returns an error
	testErr := fmt.Errorf("simulated read error")
	reader := iotest.ErrReader(testErr)

	opts := &ParseOptions{Lenient: true}
	_, _, fatalErr := p.ParseWithOptions(reader, opts)

	// Should return fatal error for I/O issues
	if fatalErr == nil {
		t.Error("Expected fatal error for I/O failure")
	}
}

// TestParseWithOptions_ValidInput verifies behavior with completely valid input
func TestParseWithOptions_ValidInput(t *testing.T) {
	p := NewParser()

	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
0 TRLR`

	opts := &ParseOptions{Lenient: true}
	lines, parseErrors, fatalErr := p.ParseWithOptions(strings.NewReader(input), opts)

	if fatalErr != nil {
		t.Fatalf("Unexpected fatal error: %v", fatalErr)
	}

	if len(parseErrors) != 0 {
		t.Errorf("Expected no parse errors with valid input, got %d", len(parseErrors))
	}

	if len(lines) != 6 {
		t.Errorf("Expected 6 lines, got %d", len(lines))
	}
}

// TestParseWithOptions_EmptyInput verifies behavior with empty input
func TestParseWithOptions_EmptyInput(t *testing.T) {
	p := NewParser()

	opts := &ParseOptions{Lenient: true}
	lines, parseErrors, fatalErr := p.ParseWithOptions(strings.NewReader(""), opts)

	if fatalErr != nil {
		t.Fatalf("Unexpected fatal error: %v", fatalErr)
	}

	if len(parseErrors) != 0 {
		t.Errorf("Expected no parse errors with empty input, got %d", len(parseErrors))
	}

	if len(lines) != 0 {
		t.Errorf("Expected 0 lines, got %d", len(lines))
	}
}

// TestParseWithOptions_AllErrorTypes verifies all error types are collected
func TestParseWithOptions_AllErrorTypes(t *testing.T) {
	p := NewParser()

	// Each line triggers a different error type
	input := `0 HEAD

X INVALID_LEVEL
0

0 @I1@
0 TRLR`

	opts := &ParseOptions{Lenient: true}
	lines, parseErrors, fatalErr := p.ParseWithOptions(strings.NewReader(input), opts)

	if fatalErr != nil {
		t.Fatalf("Unexpected fatal error: %v", fatalErr)
	}

	// Should collect errors for:
	// 1. Empty line (line 2)
	// 2. Invalid level "X" (line 3)
	// 3. Missing tag (line 4) - "0" alone
	// 4. Whitespace only (line 5)
	// 5. XRef without tag (line 6)
	expectedErrors := 5
	if len(parseErrors) != expectedErrors {
		t.Errorf("Expected %d parse errors, got %d", expectedErrors, len(parseErrors))
		for i, e := range parseErrors {
			t.Logf("Error %d: line %d: %s", i, e.Line, e.Message)
		}
	}

	// Should still parse valid lines (HEAD and TRLR)
	if len(lines) != 2 {
		t.Errorf("Expected 2 valid lines, got %d", len(lines))
	}
}

// TestParseWithOptions_ErrorDetails verifies that parse errors have correct details
func TestParseWithOptions_ErrorDetails(t *testing.T) {
	p := NewParser()

	input := `0 HEAD
BAD_LINE_HERE
0 TRLR`

	opts := &ParseOptions{Lenient: true}
	_, parseErrors, _ := p.ParseWithOptions(strings.NewReader(input), opts)

	if len(parseErrors) != 1 {
		t.Fatalf("Expected 1 parse error, got %d", len(parseErrors))
	}

	err := parseErrors[0]
	if err.Line != 2 {
		t.Errorf("Error line = %d, want 2", err.Line)
	}
	if err.Context != "BAD_LINE_HERE" {
		t.Errorf("Error context = %q, want %q", err.Context, "BAD_LINE_HERE")
	}
	if err.Message == "" {
		t.Error("Error message should not be empty")
	}
}

// TestParseWithOptions_MatchesParseBehavior verifies that strict mode matches Parse()
func TestParseWithOptions_MatchesParseBehavior(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 TRLR`

	// Parse with regular Parse()
	p1 := NewParser()
	lines1, err1 := p1.Parse(strings.NewReader(input))

	// Parse with ParseWithOptions (strict mode)
	p2 := NewParser()
	lines2, parseErrors2, fatalErr2 := p2.ParseWithOptions(strings.NewReader(input), &ParseOptions{Lenient: false})

	// Both should succeed
	if err1 != nil {
		t.Fatalf("Parse() error: %v", err1)
	}
	if fatalErr2 != nil {
		t.Fatalf("ParseWithOptions() fatal error: %v", fatalErr2)
	}
	if len(parseErrors2) != 0 {
		t.Errorf("ParseWithOptions() should have no parse errors in strict mode")
	}

	// Should have same number of lines
	if len(lines1) != len(lines2) {
		t.Errorf("Parse() returned %d lines, ParseWithOptions() returned %d", len(lines1), len(lines2))
	}

	// Lines should match
	for i := range lines1 {
		if lines1[i].Level != lines2[i].Level ||
			lines1[i].Tag != lines2[i].Tag ||
			lines1[i].Value != lines2[i].Value ||
			lines1[i].XRef != lines2[i].XRef {
			t.Errorf("Line %d mismatch", i)
		}
	}
}
