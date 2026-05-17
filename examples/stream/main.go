// Package main demonstrates streaming GEDCOM processing for memory-efficient
// handling of very large files.
//
// Two streaming halves are shown:
//
//  1. Streaming parse via parser.Records — iterate level-0 records one at a
//     time from any io.Reader. The current record is the only one held in
//     memory; previous records are eligible for GC.
//  2. Streaming encode via encoder.NewStreamEncoder — write a document
//     record-by-record to any io.Writer. No full Document is constructed.
//
// Use streaming whenever the document is too large to materialize in memory,
// or when you only need a subset of records and want to skip the cost of a
// full Document build.
//
// Note: this example uses os.Args paths directly with filepath.Clean. For
// production use that takes user-supplied paths (e.g., a web handler),
// also validate the path is under an allowed root before opening or
// creating files — filepath.Clean alone does not prevent path traversal.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/cacack/gedcom-go/charset"
	"github.com/cacack/gedcom-go/encoder"
	"github.com/cacack/gedcom-go/gedcom"
	"github.com/cacack/gedcom-go/parser"
)

const individualCount = 1000

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <gedcom_file> [output.ged]")
		fmt.Println("Example: go run main.go ../../testdata/gedcom-5.5/pres2020.ged /tmp/out.ged")
		os.Exit(1)
	}

	inputPath := filepath.Clean(os.Args[1])

	// Part 1: Streaming parse — iterate records without building a Document.
	fmt.Println("=== Streaming Parse ===")
	counts, totalLines, err := streamingParse(inputPath)
	if err != nil {
		log.Fatalf("Streaming parse failed: %v", err)
	}
	fmt.Printf("Input: %s\n", inputPath)
	fmt.Printf("Total lines processed: %d\n", totalLines)
	fmt.Println("Record counts by type:")
	for typ, n := range counts {
		fmt.Printf("  %s: %d\n", typ, n)
	}
	reportHeap("after streaming parse")

	// Part 2: Streaming encode — write a generated document record-by-record.
	if len(os.Args) < 3 {
		fmt.Println("\nNo output path provided — skipping streaming encode demo.")
		fmt.Println("Pass a second argument to write a streamed output file:")
		fmt.Println("  go run main.go", os.Args[1], "/tmp/out.ged")
		return
	}

	outputPath := filepath.Clean(os.Args[2])
	fmt.Println("\n=== Streaming Encode ===")
	if err := streamingEncode(outputPath, individualCount); err != nil {
		log.Fatalf("Streaming encode failed: %v", err)
	}
	fmt.Printf("Wrote %d individuals to %s using StreamEncoder.\n", individualCount, outputPath)
	reportHeap("after streaming encode")
}

// streamingParse iterates records from a GEDCOM file without building a
// Document. parser.Records yields one *parser.RawRecord at a time; previous
// records fall out of scope and become eligible for GC immediately.
func streamingParse(path string) (map[string]int, int, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, 0, fmt.Errorf("open input: %w", err)
	}
	defer f.Close()

	// Wrap with charset.NewReader so BOM and non-UTF-8 encodings are handled
	// transparently before the parser sees the bytes.
	r := charset.NewReader(f)

	counts := make(map[string]int)
	totalLines := 0

	for rec, err := range parser.Records(r) {
		if err != nil {
			return nil, totalLines, fmt.Errorf("parse: %w", err)
		}
		counts[rec.Type]++
		totalLines += len(rec.Lines)
	}

	return counts, totalLines, nil
}

// streamingEncode writes records one at a time with encoder.StreamEncoder.
// No full Document is constructed; each record is emitted and becomes
// eligible for GC.
//
// The named return value `err` is captured by the deferred Close calls so
// any error from finalising the stream (or closing the file) is surfaced
// to the caller — a plain `defer enc.Close()` would silently drop those
// errors, including ErrTrailerNotWritten if an earlier write returned
// early before WriteTrailer ran.
func streamingEncode(path string, count int) (err error) {
	f, err := os.Create(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close output: %w", cerr)
		}
	}()

	enc := encoder.NewStreamEncoder(f)
	defer func() {
		// Close flushes any buffered data and verifies the trailer was
		// written. Surface its error if no earlier error already preempted.
		if cerr := enc.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close encoder: %w", cerr)
		}
	}()

	header := &gedcom.Header{
		Version:      "5.5",
		Encoding:     "UTF-8",
		SourceSystem: "gedcom-go streaming example",
		Language:     "English",
	}
	if err := enc.WriteHeader(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	for i := 1; i <= count; i++ {
		xref := fmt.Sprintf("@I%d@", i)
		rec := &gedcom.Record{
			XRef: xref,
			Type: gedcom.RecordTypeIndividual,
			Tags: []*gedcom.Tag{
				{Level: 1, Tag: "NAME", Value: fmt.Sprintf("Person%d /Streamed/", i)},
				{Level: 1, Tag: "SEX", Value: "U"},
			},
		}
		if err := enc.WriteRecord(rec); err != nil {
			return fmt.Errorf("write record %s: %w", xref, err)
		}
		// rec is no longer referenced; the GC can reclaim it on the next iteration.
	}

	if err := enc.WriteTrailer(); err != nil {
		return fmt.Errorf("write trailer: %w", err)
	}
	// Deferred enc.Close() flushes — no explicit Flush needed.
	return nil
}

// reportHeap prints current heap allocation as a rough indicator of memory
// behavior. For rigorous measurement, use the TestMemory* tests in
// decoder/ and encoder/ that report both cumulative allocations and
// retained heap after the operation returns.
func reportHeap(label string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("[heap %s] HeapAlloc=%dKB Sys=%dKB\n",
		label, m.HeapAlloc/1024, m.Sys/1024)
}
