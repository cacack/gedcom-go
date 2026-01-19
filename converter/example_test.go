package converter_test

import (
	"fmt"
	"strings"

	"github.com/cacack/gedcom-go/converter"
	"github.com/cacack/gedcom-go/decoder"
	"github.com/cacack/gedcom-go/gedcom"
)

// Example demonstrates basic GEDCOM version conversion.
func Example() {
	// Parse a GEDCOM 5.5 document
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
1 SEX M
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(gedcomData))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Convert to GEDCOM 7.0
	converted, report, err := converter.Convert(doc, gedcom.Version70)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Converted: %s -> %s\n", report.SourceVersion, report.TargetVersion)
	fmt.Printf("Success: %t\n", report.Success)
	fmt.Printf("Target version: %s\n", converted.Header.Version)

	// Output:
	// Converted: 5.5 -> 7.0
	// Success: true
	// Target version: 7.0
}

// ExampleConvert shows how to convert a document to a different GEDCOM version.
func ExampleConvert() {
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Alice /Johnson/
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(gedcomData))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Convert from 5.5.1 to 7.0
	converted, report, err := converter.Convert(doc, gedcom.Version70)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Source: %s\n", report.SourceVersion)
	fmt.Printf("Target: %s\n", report.TargetVersion)
	fmt.Printf("Individuals preserved: %d\n", len(converted.Individuals()))

	// Output:
	// Source: 5.5.1
	// Target: 7.0
	// Individuals preserved: 1
}

// ExampleConvertWithOptions shows how to convert with custom options.
func ExampleConvertWithOptions() {
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Bob /Williams/
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(gedcomData))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Configure conversion options
	opts := &converter.ConvertOptions{
		Validate:            true,  // Validate after conversion
		StrictDataLoss:      false, // Allow conversions with data loss
		PreserveUnknownTags: true,  // Keep vendor extensions
	}

	converted, report, err := converter.ConvertWithOptions(doc, gedcom.Version70, opts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Converted successfully: %t\n", report.Success)
	fmt.Printf("Target version: %s\n", converted.Header.Version)

	// Output:
	// Converted successfully: true
	// Target version: 7.0
}

// ExampleConvert_report shows how to examine the conversion report.
func ExampleConvert_report() {
	// GEDCOM 5.5 with lowercase xrefs (will be normalized)
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
1 CHAR UTF-8
0 @i1@ INDI
1 NAME Carol /Davis/
0 @f1@ FAM
1 HUSB @i1@
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(gedcomData))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Convert to GEDCOM 7.0
	_, report, err := converter.Convert(doc, gedcom.Version70)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Conversion: %s -> %s\n", report.SourceVersion, report.TargetVersion)
	fmt.Printf("Transformations: %d\n", len(report.Transformations))
	fmt.Printf("Data loss: %t\n", report.HasDataLoss())

	// Output:
	// Conversion: 5.5 -> 7.0
	// Transformations: 2
	// Data loss: false
}
