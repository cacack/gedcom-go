// Package main demonstrates lenient decoding with diagnostics: what the decoder recovered from or skipped, and where.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/cacack/gedcom-go/v3/decoder"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <gedcom_file>")
		fmt.Println("Example: go run main.go ../../testdata/malformed/invalid-level.ged")
		os.Exit(1)
	}

	filename := filepath.Clean(os.Args[1])

	// #nosec G703 -- the path is this CLI's own argv, not attacker-controlled input.
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer f.Close()

	// DecodeWithDiagnostics decodes leniently: malformed lines are recovered
	// from or skipped, and each one is reported as a Diagnostic instead of
	// failing the whole decode. Even when err is non-nil (an I/O failure part
	// way through, or no line decodable at all) result may still hold a
	// partial document, so inspect both.
	result, err := decoder.DecodeWithDiagnostics(f, decoder.DefaultOptions())
	if result == nil {
		log.Fatalf("Failed to decode GEDCOM: %v", err)
	}
	if err != nil {
		fmt.Printf("Decode stopped early: %v\n", err)
	}

	doc := result.Document
	fmt.Printf("GEDCOM File: %s\n", filename)
	fmt.Printf("Records decoded: %d\n", len(doc.Records))

	if len(result.Diagnostics) == 0 {
		fmt.Println("No diagnostics: the file decoded cleanly.")
		return
	}

	fmt.Printf("Diagnostics: %d (%d error(s), %d warning(s))\n\n",
		len(result.Diagnostics), len(result.Diagnostics.Errors()), len(result.Diagnostics.Warnings()))
	for _, d := range result.Diagnostics {
		// Each Diagnostic carries a line number, severity, machine-readable
		// code, message and the offending content.
		fmt.Printf("  %s\n", d)
	}

	if result.Diagnostics.HasErrors() {
		fmt.Println("\nThe document is partial: some lines could not be decoded.")
	}
}
