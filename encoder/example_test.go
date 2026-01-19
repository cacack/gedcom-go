package encoder_test

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/cacack/gedcom-go/encoder"
	"github.com/cacack/gedcom-go/gedcom"
)

// Example demonstrates basic GEDCOM document encoding.
func Example() {
	// Create a document with header and records
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version:  gedcom.Version55,
			Encoding: gedcom.EncodingUTF8,
		},
		Trailer: &gedcom.Trailer{},
	}

	// Encode to a buffer
	var buf bytes.Buffer
	if err := encoder.Encode(&buf, doc); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Show the output has required structure
	output := buf.String()
	hasHead := strings.Contains(output, "0 HEAD")
	hasTrlr := strings.Contains(output, "0 TRLR")

	fmt.Printf("Has HEAD: %v\n", hasHead)
	fmt.Printf("Has TRLR: %v\n", hasTrlr)

	// Output:
	// Has HEAD: true
	// Has TRLR: true
}

// ExampleEncode shows basic encoding to an io.Writer.
func ExampleEncode() {
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version:      gedcom.Version551,
			Encoding:     gedcom.EncodingUTF8,
			SourceSystem: "MyApp",
		},
		Trailer: &gedcom.Trailer{},
	}

	var buf bytes.Buffer
	if err := encoder.Encode(&buf, doc); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Encoded %d bytes\n", buf.Len())

	// Output:
	// Encoded 60 bytes
}

// ExampleEncodeWithOptions shows encoding with custom line endings.
func ExampleEncodeWithOptions() {
	doc := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version55},
		Trailer: &gedcom.Trailer{},
	}

	// Use Windows-style line endings (CRLF)
	opts := &encoder.EncodeOptions{
		LineEnding: "\r\n",
	}

	var buf bytes.Buffer
	if err := encoder.EncodeWithOptions(&buf, doc, opts); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Count CRLF occurrences
	crlfCount := strings.Count(buf.String(), "\r\n")
	fmt.Printf("Lines with CRLF: %d\n", crlfCount)

	// Output:
	// Lines with CRLF: 4
}

// ExampleNewStreamEncoder demonstrates streaming encoding for large files.
func ExampleNewStreamEncoder() {
	var buf bytes.Buffer

	// Create a streaming encoder
	enc := encoder.NewStreamEncoder(&buf)

	// Write header first
	if err := enc.WriteHeader(&gedcom.Header{
		Version:  gedcom.Version55,
		Encoding: gedcom.EncodingUTF8,
	}); err != nil {
		fmt.Printf("Error writing header: %v\n", err)
		return
	}

	// Write records one at a time (memory efficient for large files)
	record := &gedcom.Record{
		XRef: "@I1@",
		Type: gedcom.RecordTypeIndividual,
		Tags: []*gedcom.Tag{
			{Level: 1, Tag: "NAME", Value: "John /Doe/"},
			{Level: 1, Tag: "SEX", Value: "M"},
		},
	}

	if err := enc.WriteRecord(record); err != nil {
		fmt.Printf("Error writing record: %v\n", err)
		return
	}

	// Write trailer to complete the document
	if err := enc.WriteTrailer(); err != nil {
		fmt.Printf("Error writing trailer: %v\n", err)
		return
	}

	// Close flushes any buffered data
	if err := enc.Close(); err != nil {
		fmt.Printf("Error closing: %v\n", err)
		return
	}

	// Verify output structure
	output := buf.String()
	fmt.Printf("Has HEAD: %v\n", strings.Contains(output, "0 HEAD"))
	fmt.Printf("Has INDI: %v\n", strings.Contains(output, "@I1@ INDI"))
	fmt.Printf("Has TRLR: %v\n", strings.Contains(output, "0 TRLR"))

	// Output:
	// Has HEAD: true
	// Has INDI: true
	// Has TRLR: true
}

// ExampleStreamEncoder_largeFile shows the pattern for encoding large files.
func ExampleStreamEncoder_largeFile() {
	var buf bytes.Buffer
	enc := encoder.NewStreamEncoder(&buf)

	// Start with header
	_ = enc.WriteHeader(&gedcom.Header{Version: gedcom.Version55})

	// In a real application, you might iterate over a database or channel:
	//   for record := range recordChannel {
	//       enc.WriteRecord(record)
	//   }

	// Write multiple records
	for i := 1; i <= 3; i++ {
		record := &gedcom.Record{
			XRef: fmt.Sprintf("@I%d@", i),
			Type: gedcom.RecordTypeIndividual,
			Tags: []*gedcom.Tag{
				{Level: 1, Tag: "NAME", Value: fmt.Sprintf("Person %d", i)},
			},
		}
		_ = enc.WriteRecord(record)
	}

	_ = enc.WriteTrailer()
	_ = enc.Close()

	// Count records in output
	recordCount := strings.Count(buf.String(), "INDI")
	fmt.Printf("Encoded %d individual records\n", recordCount)

	// Output:
	// Encoded 3 individual records
}
