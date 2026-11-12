// Command gengrammar writes validator/grammar_tables.go from the specification
// inventories under testdata/spec (see internal/specgrammar). The validator
// package runs it through go generate:
//
//	go run ./internal/cmd/gengrammar [-spec DIR] [-out FILE]
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/cacack/gedcom-go/v3/internal/specgrammar"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

func run(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("gengrammar", flag.ContinueOnError)
	fs.SetOutput(stderr)
	spec := fs.String("spec", "../testdata/spec", "the testdata/spec directory")
	out := fs.String("out", "grammar_tables.go", "the file to write")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	src, err := specgrammar.Generate(*spec)
	if err != nil {
		fmt.Fprintf(stderr, "gengrammar: %v\n", err)
		return 1
	}
	if err := os.WriteFile(*out, src, 0o644); err != nil { // #nosec G306 -- a checked-in source file
		fmt.Fprintf(stderr, "gengrammar: %v\n", err)
		return 1
	}
	return 0
}
