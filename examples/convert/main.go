// Package main demonstrates converting a GEDCOM document to another version and reading the conversion report.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/cacack/gedcom-go/v3/converter"
	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <gedcom_file> <target_version> [output_file]")
		fmt.Println("Target versions: 5.5, 5.5.1, 7.0")
		fmt.Println("Example: go run main.go ../../testdata/gedcom-5.5.1/comprehensive.ged 7.0")
		os.Exit(1)
	}

	filename := filepath.Clean(os.Args[1])
	target := gedcom.Version(os.Args[2])
	if !target.IsValid() {
		log.Fatal("Unsupported target version: want 5.5, 5.5.1 or 7.0")
	}

	// #nosec G703 -- the path is this CLI's own argv, not attacker-controlled input.
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer f.Close()

	doc, err := decoder.Decode(f)
	if err != nil {
		log.Fatalf("Failed to decode GEDCOM: %v", err)
	}

	// Convert never modifies doc; it returns a converted copy and a report
	// of everything it changed, approximated or could not carry across.
	converted, report, err := converter.Convert(doc, target)
	if err != nil {
		log.Fatalf("Failed to convert: %v", err)
	}

	fmt.Printf("Converted %s: %s -> %s\n", filename, report.SourceVersion, report.TargetVersion)
	fmt.Printf("Success: %t\n", report.Success)

	fmt.Printf("\n=== Transformations (%d) ===\n", len(report.Transformations))
	for _, t := range report.Transformations {
		fmt.Printf("  %s x%d: %s\n", t.Type, t.Count, t.Description)
	}

	fmt.Printf("\n=== Data loss (%d) ===\n", len(report.DataLoss))
	for _, d := range report.DataLoss {
		fmt.Printf("  %s: %s (%d record(s))\n", d.Feature, d.Reason, len(d.AffectedRecords))
	}

	printNotes("Dropped", report.Dropped)
	printNotes("Approximated", report.Approximated)
	printNotes("Normalized", report.Normalized)
	printNotes("Preserved", report.Preserved)

	if len(report.ValidationIssues) > 0 {
		fmt.Printf("\n=== Validation issues after conversion (%d) ===\n", len(report.ValidationIssues))
		for _, issue := range report.ValidationIssues {
			fmt.Printf("  %s\n", issue)
		}
	}

	if len(os.Args) > 3 {
		out := filepath.Clean(os.Args[3])
		// #nosec G703 -- the path is this CLI's own argv, not attacker-controlled input.
		w, err := os.Create(out)
		if err != nil {
			log.Fatalf("Failed to create output file: %v", err)
		}
		defer w.Close()
		if err := encoder.Encode(w, converted); err != nil {
			log.Fatalf("Failed to encode: %v", err)
		}
		fmt.Printf("\nWrote GEDCOM %s to %s\n", converted.Header.Version, out)
	}
}

// printNotes prints up to five notes of one report category.
func printNotes(category string, notes []gedcom.ConversionNote) {
	fmt.Printf("\n=== %s (%d) ===\n", category, len(notes))
	for i, n := range notes {
		if i == 5 {
			fmt.Printf("  ... and %d more\n", len(notes)-5)
			break
		}
		fmt.Printf("  %s: %q -> %q (%s)\n", n.Path, n.Original, n.Result, n.Reason)
	}
}
