// Package main demonstrates GEDCOM file validation with error categorization, grouping, and detailed reporting.
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cacack/gedcom-go/decoder"
	"github.com/cacack/gedcom-go/validator"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <gedcom_file>")
		fmt.Println("Example: go run main.go ../../testdata/gedcom-5.5/minimal.ged")
		os.Exit(1)
	}

	filename := os.Args[1]

	// Open and parse GEDCOM file
	f, err := os.Open(filename) // #nosec G304 -- CLI tool accepts user-provided paths
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer f.Close()

	doc, err := decoder.Decode(f)
	if err != nil {
		log.Fatalf("Failed to decode GEDCOM: %v", err)
	}

	fmt.Printf("Validating GEDCOM File: %s\n", filename)
	fmt.Printf("Version: %s\n", doc.Header.Version)
	fmt.Printf("Encoding: %s\n\n", doc.Header.Encoding)

	// Validate the document
	v := validator.New()
	errors := v.Validate(doc)

	// Display results
	if len(errors) == 0 {
		fmt.Println("✅ Validation passed!")
		fmt.Println("No errors found.")
		return
	}

	fmt.Printf("❌ Validation failed with %d error(s):\n\n", len(errors))

	// Group errors by code
	errorsByCode := make(map[string][]error)
	for _, err := range errors {
		// Try to get the code from ValidationError
		code := "UNKNOWN"
		if verr, ok := err.(*validator.ValidationError); ok {
			code = verr.Code
		}
		errorsByCode[code] = append(errorsByCode[code], err)
	}

	// Display errors grouped by code
	for code, errs := range errorsByCode {
		fmt.Printf("Error Code: %s (%d occurrence(s))\n", code, len(errs))

		// Show first 3 examples
		for i, err := range errs {
			if i >= 3 {
				fmt.Printf("  ... and %d more\n", len(errs)-3)
				break
			}

			// Display error message
			if verr, ok := err.(*validator.ValidationError); ok {
				details := []string{}
				if verr.Line > 0 {
					details = append(details, fmt.Sprintf("line %d", verr.Line))
				}
				if verr.XRef != "" {
					details = append(details, fmt.Sprintf("XRef: %s", verr.XRef))
				}

				if len(details) > 0 {
					fmt.Printf("  - %s (%s)\n", verr.Message, strings.Join(details, ", "))
				} else {
					fmt.Printf("  - %s\n", verr.Message)
				}
			} else {
				fmt.Printf("  - %v\n", err)
			}
		}
		fmt.Println()
	}

	// Summary
	fmt.Println("=== Summary ===")
	fmt.Printf("Total Records: %d\n", len(doc.Records))
	fmt.Printf("Total Errors: %d\n", len(errors))
	fmt.Printf("Error Types: %d\n", len(errorsByCode))

	// Exit with error code if validation failed
	os.Exit(1)
}
