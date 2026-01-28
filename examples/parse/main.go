// Package main demonstrates basic GEDCOM file parsing with summary statistics, record counting, and validation.
package main

import (
	"fmt"
	"log"
	"os"

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

	// Display basic information
	fmt.Printf("GEDCOM File: %s\n", filename)
	fmt.Printf("Version: %s\n", doc.Header.Version)
	fmt.Printf("Encoding: %s\n", doc.Header.Encoding)
	if doc.Header.SourceSystem != "" {
		fmt.Printf("Source System: %s\n", doc.Header.SourceSystem)
	}
	fmt.Printf("\nTotal Records: %d\n", len(doc.Records))
	fmt.Printf("Cross-references: %d\n", len(doc.XRefMap))

	// Count record types
	recordCounts := make(map[string]int)
	for _, record := range doc.Records {
		recordCounts[string(record.Type)]++
	}

	fmt.Println("\nRecord Types:")
	for recordType, count := range recordCounts {
		fmt.Printf("  %s: %d\n", recordType, count)
	}

	// Validate the document
	v := validator.New()
	errors := v.Validate(doc)

	if len(errors) > 0 {
		fmt.Printf("\nValidation Errors (%d):\n", len(errors))
		for i, err := range errors {
			if i < 10 { // Show first 10 errors
				fmt.Printf("  - %v\n", err)
			}
		}
		if len(errors) > 10 {
			fmt.Printf("  ... and %d more\n", len(errors)-10)
		}
	} else {
		fmt.Println("\n✓ No validation errors")
	}
}
