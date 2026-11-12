// Package main demonstrates lazy, indexed access to a GEDCOM file: build a record index once, then parse and decode only the records you ask for.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/parser"
	"github.com/cacack/gedcom-go/v3/version"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <gedcom_file> [xref ...]")
		fmt.Println("Example: go run main.go ../../testdata/gedcom-5.5.1/comprehensive.ged @I1@ @F1@")
		os.Exit(1)
	}

	filename := filepath.Clean(os.Args[1])

	// LazyParser needs an io.ReadSeeker: it seeks straight to each record's
	// byte offset in the file rather than reading the whole file. It reads
	// the bytes as they are, with no charset conversion, so it suits UTF-8
	// files; decode an ANSEL or UTF-16 file with the decoder package instead.
	// #nosec G703 -- the path is this CLI's own argv, not attacker-controlled input.
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer f.Close()

	lp := parser.NewLazyParser(f)

	// One pass over the file records each level-0 record's offset. The index
	// can be saved with SaveIndex and reloaded with LoadIndex to skip this
	// pass on the next run.
	if err := lp.BuildIndex(); err != nil {
		log.Fatalf("Failed to build index: %v", err)
	}
	fmt.Printf("Indexed %s: %d records\n", filename, lp.RecordCount())

	// The header gives the version DecodeRecord needs to interpret records.
	head, err := lp.FindRecordByType("HEAD")
	if err != nil {
		log.Fatalf("Failed to read header: %v", err)
	}
	ver := version.DeclaredVersion(head.Lines)
	if ver == "" {
		fmt.Println("Version: not declared (records are read by the GEDCOM 5.5 rules)")
	} else {
		fmt.Printf("Version: %s\n", ver)
	}

	xrefs := os.Args[2:]
	if len(xrefs) == 0 {
		// No XRefs given: look up the first indexed one in sorted order.
		all := lp.XRefs()
		slices.Sort(all)
		if len(all) == 0 {
			fmt.Println("No records with cross-references.")
			return
		}
		xrefs = all[:1]
	}

	for _, xref := range xrefs {
		// FindRecord parses only the requested record's lines.
		raw, err := lp.FindRecord(xref)
		if err != nil {
			fmt.Printf("\n%s: %v\n", xref, err)
			continue
		}
		fmt.Printf("\n%s %s at byte %d (%d lines)\n", raw.XRef, raw.Type, raw.ByteOffset, len(raw.Lines))

		// DecodeRecord turns the raw lines into the typed record Decode
		// would build, without decoding the rest of the file.
		rec, err := decoder.DecodeRecord(raw, ver, nil)
		if err != nil {
			fmt.Printf("  decode: %v\n", err)
			continue
		}
		if ind, ok := rec.GetIndividual(); ok {
			fmt.Printf("  Individual: %s\n", ind.FullName())
			if birth := ind.BirthDate(); birth != nil {
				fmt.Printf("  Born: %s\n", birth)
			}
		} else if fam, ok := rec.GetFamily(); ok {
			fmt.Printf("  Family: husband %s, wife %s, %d child(ren)\n", fam.Husband, fam.Wife, len(fam.Children))
		} else {
			fmt.Printf("  %s record with %d tags\n", rec.Type, len(rec.Tags))
		}
	}
}
