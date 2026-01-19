package version_test

import (
	"fmt"

	"github.com/cacack/gedcom-go/gedcom"
	"github.com/cacack/gedcom-go/parser"
	"github.com/cacack/gedcom-go/version"
)

// Example demonstrates basic GEDCOM version detection.
func Example() {
	// Parsed lines from a GEDCOM file (typically via parser.Parse)
	lines := []*parser.Line{
		{Level: 0, Tag: "HEAD"},
		{Level: 1, Tag: "GEDC"},
		{Level: 2, Tag: "VERS", Value: "5.5.1"},
		{Level: 0, Tag: "TRLR"},
	}

	ver, err := version.DetectVersion(lines)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Detected version: %s\n", ver)

	// Output:
	// Detected version: 5.5.1
}

// ExampleDetectVersion shows how to detect the GEDCOM version from parsed lines.
func ExampleDetectVersion() {
	// DetectVersion examines header for GEDC.VERS tag
	lines := []*parser.Line{
		{Level: 0, Tag: "HEAD"},
		{Level: 1, Tag: "GEDC"},
		{Level: 2, Tag: "VERS", Value: "5.5"},
		{Level: 1, Tag: "CHAR", Value: "UTF-8"},
		{Level: 0, Tag: "@I1@", XRef: "@I1@"},
		{Level: 0, Tag: "TRLR"},
	}

	ver, err := version.DetectVersion(lines)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Version: %s\n", ver)
	fmt.Printf("Is valid: %v\n", version.IsValidVersion(ver))

	// Output:
	// Version: 5.5
	// Is valid: true
}

// ExampleDetectVersion_v55 demonstrates detecting GEDCOM 5.5 version.
func ExampleDetectVersion_v55() {
	// GEDCOM 5.5 is the most common version
	lines := []*parser.Line{
		{Level: 0, Tag: "HEAD"},
		{Level: 1, Tag: "GEDC"},
		{Level: 2, Tag: "VERS", Value: "5.5"},
		{Level: 2, Tag: "FORM", Value: "LINEAGE-LINKED"},
		{Level: 0, Tag: "TRLR"},
	}

	ver, _ := version.DetectVersion(lines)
	fmt.Printf("Version: %s\n", ver)
	fmt.Printf("Is 5.5: %v\n", ver == gedcom.Version55)

	// Output:
	// Version: 5.5
	// Is 5.5: true
}

// ExampleDetectVersion_v70 demonstrates detecting GEDCOM 7.0 version.
func ExampleDetectVersion_v70() {
	// GEDCOM 7.0 is the newest specification
	lines := []*parser.Line{
		{Level: 0, Tag: "HEAD"},
		{Level: 1, Tag: "GEDC"},
		{Level: 2, Tag: "VERS", Value: "7.0"},
		{Level: 0, Tag: "TRLR"},
	}

	ver, _ := version.DetectVersion(lines)
	fmt.Printf("Version: %s\n", ver)
	fmt.Printf("Is 7.0: %v\n", ver == gedcom.Version70)

	// Output:
	// Version: 7.0
	// Is 7.0: true
}

// ExampleDetectVersion_tagFallback shows version detection via tag heuristics.
func ExampleDetectVersion_tagFallback() {
	// When header lacks version info, DetectVersion uses tag-based heuristics.
	// GEDCOM 7.0-specific tags (EXID, PHRASE, SNOTE, etc.) indicate version 7.0.
	// GEDCOM 5.5.1-specific tags (MAP, LATI, LONG, EMAIL, etc.) indicate 5.5.1.
	lines := []*parser.Line{
		{Level: 0, Tag: "HEAD"},
		{Level: 0, XRef: "@I1@", Tag: "INDI"},
		{Level: 1, Tag: "NAME", Value: "John /Smith/"},
		{Level: 1, Tag: "EXID", Value: "external-id-123"}, // GEDCOM 7.0 tag
		{Level: 0, Tag: "TRLR"},
	}

	ver, _ := version.DetectVersion(lines)
	fmt.Printf("Detected version: %s\n", ver)

	// Output:
	// Detected version: 7.0
}

// ExampleIsValidVersion demonstrates validating version constants.
func ExampleIsValidVersion() {
	// Check if a version is one of the supported GEDCOM versions
	fmt.Printf("5.5 valid: %v\n", version.IsValidVersion(gedcom.Version55))
	fmt.Printf("5.5.1 valid: %v\n", version.IsValidVersion(gedcom.Version551))
	fmt.Printf("7.0 valid: %v\n", version.IsValidVersion(gedcom.Version70))
	fmt.Printf("empty valid: %v\n", version.IsValidVersion(gedcom.Version("")))

	// Output:
	// 5.5 valid: true
	// 5.5.1 valid: true
	// 7.0 valid: true
	// empty valid: false
}
