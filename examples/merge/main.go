// Package main demonstrates combining two GEDCOM documents, resolving XRef collisions and reading the combine report.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/merge"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <first_file> <second_file> [output_file]")
		fmt.Println("Example: go run main.go ../../testdata/gedcom-5.5.1/comprehensive.ged ../../testdata/gedcom-5.5.1/attribute-event-detail.ged")
		os.Exit(1)
	}

	doc1 := decodeFile(os.Args[1])
	doc2 := decodeFile(os.Args[2])

	// The zero-value CombineOptions refuses any XRef the two documents
	// share. RenumberDoc2 instead gives each colliding doc2 record a fresh
	// XRef and rewrites every reference to it, so the combined document
	// keeps its links. Combine never modifies doc1 or doc2.
	combined, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{
		CollisionStrategy: merge.RenumberDoc2,
	})
	if err != nil {
		log.Fatalf("Failed to combine: %v", err)
	}

	fmt.Printf("First:    %d records\n", len(doc1.Records))
	fmt.Printf("Second:   %d records\n", len(doc2.Records))
	fmt.Printf("Combined: %d records (%d individuals, %d families)\n",
		len(combined.Records), len(combined.Individuals()), len(combined.Families()))

	fmt.Printf("\n=== Renumbered XRefs from the second file (%d) ===\n", len(report.RemappedXRefs))
	olds := make([]string, 0, len(report.RemappedXRefs))
	for old := range report.RemappedXRefs {
		olds = append(olds, old)
	}
	slices.Sort(olds)
	for _, old := range olds {
		fmt.Printf("  %s -> %s\n", old, report.RemappedXRefs[old])
	}

	// Header fields both files set differently, and header structures both
	// declare: the first file's wins and the second's is dropped.
	fmt.Printf("\n=== Header conflicts (%d) ===\n", len(report.HeaderConflicts))
	for _, c := range report.HeaderConflicts {
		fmt.Printf("  %s: first %q, second %q\n", c.Field, c.Doc1, c.Doc2)
	}

	if len(os.Args) > 3 {
		out := filepath.Clean(os.Args[3])
		// #nosec G703 -- the path is this CLI's own argv, not attacker-controlled input.
		w, err := os.Create(out)
		if err != nil {
			log.Fatalf("Failed to create output file: %v", err)
		}
		defer w.Close()
		if err := encoder.Encode(w, combined); err != nil {
			log.Fatalf("Failed to encode: %v", err)
		}
		fmt.Printf("\nWrote combined document to %s\n", out)
	}
}

// decodeFile opens and decodes one GEDCOM file, exiting on failure.
func decodeFile(name string) *gedcom.Document {
	filename := filepath.Clean(name)
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
	return doc
}
