package decoder_test

import (
	"fmt"
	"strings"

	"github.com/cacack/gedcom-go/decoder"
)

// Example demonstrates basic GEDCOM file decoding.
func Example() {
	// GEDCOM data can come from a file, network, or any io.Reader
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
1 SEX M
1 BIRT
2 DATE 15 MAR 1920
0 @I2@ INDI
1 NAME Jane /Doe/
1 SEX F
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(gedcomData))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Access individuals
	individuals := doc.Individuals()
	fmt.Printf("Found %d individuals\n", len(individuals))

	// Access families
	families := doc.Families()
	fmt.Printf("Found %d families\n", len(families))

	// Output:
	// Found 2 individuals
	// Found 1 families
}

// ExampleDecode shows how to decode GEDCOM data from an io.Reader.
func ExampleDecode() {
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME Alice /Johnson/
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(gedcomData))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Lookup individual by XRef
	individual := doc.GetIndividual("@I1@")
	if individual != nil && len(individual.Names) > 0 {
		fmt.Printf("Found: %s\n", individual.Names[0].Full)
	}

	// Output:
	// Found: Alice /Johnson/
}

// ExampleDecodeWithOptions shows how to decode with custom options.
func ExampleDecodeWithOptions() {
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME Bob /Williams/
0 TRLR`

	// Create custom options
	opts := &decoder.DecodeOptions{
		MaxNestingDepth: 50,
		StrictMode:      false,
	}

	doc, err := decoder.DecodeWithOptions(strings.NewReader(gedcomData), opts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Decoded %d records\n", len(doc.Records))

	// Output:
	// Decoded 1 records
}

// ExampleDecodeWithOptions_progress shows how to track decoding progress.
func ExampleDecodeWithOptions_progress() {
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME Carol /Davis/
0 TRLR`

	// Track progress during decoding (useful for large files)
	opts := &decoder.DecodeOptions{
		TotalSize: int64(len(gedcomData)),
		OnProgress: func(bytesRead, totalBytes int64) {
			// Progress callback is called periodically during parsing
			// In real applications, update a progress bar or UI here
		},
	}

	doc, err := decoder.DecodeWithOptions(strings.NewReader(gedcomData), opts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Decoded successfully: %d individuals\n", len(doc.Individuals()))

	// Output:
	// Decoded successfully: 1 individuals
}

// ExampleDecodeWithDiagnostics demonstrates lenient parsing mode with diagnostic collection.
// This is useful when processing GEDCOM files that may contain errors but you still want
// to extract as much valid data as possible.
func ExampleDecodeWithDiagnostics() {
	// This GEDCOM data contains an error: line 8 has a negative level number
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
1 SEX M
0 @I2@ INDI
-1 NAME Jane /Doe/
1 NAME Jane /Doe/
0 TRLR`

	// Use DecodeWithDiagnostics for lenient parsing
	opts := &decoder.DecodeOptions{
		StrictMode: false, // This is the default, shown for clarity
	}

	result, err := decoder.DecodeWithDiagnostics(strings.NewReader(gedcomData), opts)
	if err != nil {
		fmt.Printf("Fatal error: %v\n", err)
		return
	}

	// Check if there were any issues
	if len(result.Diagnostics) > 0 {
		fmt.Printf("Found %d diagnostic(s)\n", len(result.Diagnostics))

		// Check specifically for errors vs warnings
		if result.Diagnostics.HasErrors() {
			fmt.Printf("  Errors: %d\n", len(result.Diagnostics.Errors()))
		}
	}

	// The document still contains successfully parsed data
	fmt.Printf("Parsed %d individuals despite errors\n", len(result.Document.Individuals()))

	// Output:
	// Found 1 diagnostic(s)
	//   Errors: 1
	// Parsed 2 individuals despite errors
}

// ExampleDecodeWithDiagnostics_filterBySeverity shows how to filter diagnostics by severity.
func ExampleDecodeWithDiagnostics_filterBySeverity() {
	// GEDCOM with an invalid line (missing level number)
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME Alice /Johnson/
invalid line without level
0 TRLR`

	result, err := decoder.DecodeWithDiagnostics(strings.NewReader(gedcomData), nil)
	if err != nil {
		fmt.Printf("Fatal error: %v\n", err)
		return
	}

	// Filter to get only errors (not warnings or info)
	errors := result.Diagnostics.Errors()
	for _, diag := range errors {
		fmt.Printf("Line %d: %s\n", diag.Line, diag.Code)
	}

	// Output:
	// Line 6: INVALID_LEVEL
}

// ExampleDecodeWithDiagnostics_inspectDetails shows how to inspect diagnostic details.
func ExampleDecodeWithDiagnostics_inspectDetails() {
	// GEDCOM with a negative level error
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
-1 NAME Bob /Wilson/
0 TRLR`

	result, _ := decoder.DecodeWithDiagnostics(strings.NewReader(gedcomData), nil)

	// Inspect the first diagnostic's details
	if len(result.Diagnostics) > 0 {
		d := result.Diagnostics[0]
		fmt.Printf("Severity: %s\n", d.Severity)
		fmt.Printf("Code: %s\n", d.Code)
		fmt.Printf("Line: %d\n", d.Line)
		fmt.Printf("Message: %s\n", d.Message)
	}

	// Output:
	// Severity: ERROR
	// Code: INVALID_LEVEL
	// Line: 5
	// Message: level cannot be negative
}
