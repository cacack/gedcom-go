// Package main demonstrates GEDCOM file validation with error categorization, grouping, and detailed reporting.
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cacack/gedcom-go/v2/decoder"
	"github.com/cacack/gedcom-go/v2/validator"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <gedcom_file>")
		fmt.Println("Example: go run main.go ../../testdata/gedcom-5.5/minimal.ged")
		os.Exit(1)
	}

	filename := filepath.Clean(os.Args[1])

	// Open and parse GEDCOM file
	// #nosec G703 -- the path is this CLI's own argv, not attacker-controlled input; see the package comment on validating against an allowed root in production.
	f, err := os.Open(filename)
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

	// Validate the document. Every element Validate returns is a
	// *validator.Issue, so errors.As recovers its code, severity and location.
	v := validator.New()
	errs := v.Validate(doc)

	// Display results
	if len(errs) == 0 {
		fmt.Println("✅ Validation passed!")
		fmt.Println("No issues found.")
		return
	}

	fmt.Printf("❌ Validation found %d issue(s):\n\n", len(errs))

	// Group issues by code
	issuesByCode := make(map[string][]*validator.Issue)
	for _, err := range errs {
		var issue *validator.Issue
		if !errors.As(err, &issue) {
			continue
		}
		issuesByCode[issue.Code] = append(issuesByCode[issue.Code], issue)
	}

	// Display issues grouped by code
	for code, issues := range issuesByCode {
		fmt.Printf("Code: %s (%d occurrence(s))\n", code, len(issues))

		// Show first 3 examples
		for i, issue := range issues {
			if i >= 3 {
				fmt.Printf("  ... and %d more\n", len(issues)-3)
				break
			}

			details := []string{issue.Severity.String()}
			if issue.LineNumber > 0 {
				details = append(details, fmt.Sprintf("line %d", issue.LineNumber))
			}
			if issue.RecordXRef != "" {
				details = append(details, fmt.Sprintf("XRef: %s", issue.RecordXRef))
			}
			fmt.Printf("  - %s (%s)\n", issue.Message, strings.Join(details, ", "))
		}
		fmt.Println()
	}

	// Summary
	fmt.Println("=== Summary ===")
	fmt.Printf("Total Records: %d\n", len(doc.Records))
	fmt.Printf("Total Issues: %d\n", len(errs))
	fmt.Printf("Issue Codes: %d\n", len(issuesByCode))

	// Exit with error code if validation failed
	os.Exit(1)
}
