package parser_test

import (
	"fmt"
	"strings"

	"github.com/cacack/gedcom-go/parser"
)

// Example demonstrates basic GEDCOM line parsing.
func Example() {
	// GEDCOM data as a string (typically read from a file)
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
0 TRLR`

	// Create a parser and parse the content
	p := parser.NewParser()
	lines, err := p.Parse(strings.NewReader(gedcomData))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Parsed %d lines\n", len(lines))

	// Output:
	// Parsed 6 lines
}

// ExampleParse shows how to parse complete GEDCOM content from an io.Reader.
func ExampleParser_Parse() {
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME Alice /Johnson/
1 SEX F
0 @I2@ INDI
1 NAME Bob /Johnson/
1 SEX M
0 TRLR`

	p := parser.NewParser()
	lines, err := p.Parse(strings.NewReader(gedcomData))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Count records at level 0 (top-level records)
	records := 0
	for _, line := range lines {
		if line.Level == 0 {
			records++
		}
	}

	fmt.Printf("Total lines: %d\n", len(lines))
	fmt.Printf("Top-level records: %d\n", records)

	// Output:
	// Total lines: 10
	// Top-level records: 4
}

// ExampleParser_ParseLine shows line-by-line parsing for streaming scenarios.
func ExampleParser_ParseLine() {
	// Line-by-line parsing is useful for streaming or custom parsing logic
	p := parser.NewParser()

	inputLines := []string{
		"0 HEAD",
		"1 GEDC",
		"2 VERS 5.5",
		"0 @I1@ INDI",
		"1 NAME John /Smith/",
	}

	for _, input := range inputLines {
		line, err := p.ParseLine(input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Level %d: %s\n", line.Level, line.Tag)
	}

	// Output:
	// Level 0: HEAD
	// Level 1: GEDC
	// Level 2: VERS
	// Level 0: INDI
	// Level 1: NAME
}

// ExampleLine demonstrates accessing parsed Line fields.
func ExampleLine() {
	p := parser.NewParser()

	// Parse a line with XRef (cross-reference identifier)
	line1, _ := p.ParseLine("0 @I1@ INDI")
	fmt.Printf("XRef: %s, Tag: %s\n", line1.XRef, line1.Tag)

	// Parse a line with a value
	line2, _ := p.ParseLine("1 NAME John /Smith/")
	fmt.Printf("Tag: %s, Value: %s\n", line2.Tag, line2.Value)

	// Parse a nested line
	line3, _ := p.ParseLine("2 GIVN John")
	fmt.Printf("Level: %d, Tag: %s, Value: %s\n", line3.Level, line3.Tag, line3.Value)

	// Output:
	// XRef: @I1@, Tag: INDI
	// Tag: NAME, Value: John /Smith/
	// Level: 2, Tag: GIVN, Value: John
}

// ExampleLine_lineNumber shows how line numbers are tracked for error reporting.
func ExampleLine_lineNumber() {
	p := parser.NewParser()

	// Parse multiple lines - line numbers are tracked automatically
	lines := []string{
		"0 HEAD",
		"1 SOUR MyApp",
		"0 @I1@ INDI",
	}

	for _, input := range lines {
		line, _ := p.ParseLine(input)
		fmt.Printf("Line %d: %s\n", line.LineNumber, line.Tag)
	}

	// Output:
	// Line 1: HEAD
	// Line 2: SOUR
	// Line 3: INDI
}
