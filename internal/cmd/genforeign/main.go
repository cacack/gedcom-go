// Command genforeign writes the converter's table of version-foreign
// structures (converter/structures_gen.go) from the vendored specification
// files. It is run by go generate in the converter package:
//
//	go generate ./converter
//
//	go run ./internal/cmd/genforeign [-spec DIR] [-out FILE]
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/cacack/gedcom-go/v3/internal/specgraph"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

// run generates the table as flags in args say and returns the exit code,
// writing any error to stderr.
func run(args []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("genforeign", flag.ContinueOnError)
	flags.SetOutput(stderr)
	spec := flags.String("spec", "../testdata/spec", "directory holding the vendored gedcom-* specification files")
	out := flags.String("out", "structures_gen.go", "file to write")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	src, err := specgraph.Generate(*spec)
	if err == nil {
		err = os.WriteFile(*out, src, 0o600)
	}
	if err != nil {
		fmt.Fprintln(stderr, "genforeign:", err)
		return 1
	}
	return 0
}
