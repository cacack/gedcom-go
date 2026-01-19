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
