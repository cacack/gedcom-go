package parser

import (
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
		t.Error("Expected error from Parse with failing reader")
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
				t.Error("Expected error for xref without tag")
			}
			if !strings.Contains(err.Error(), "xref must have a tag") {
				t.Errorf("Expected 'xref must have a tag' error, got: %v", err)
			}
		})
	}
}
