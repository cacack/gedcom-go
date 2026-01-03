// Package decoder provides high-level GEDCOM file decoding functionality.
//
// The decoder package converts GEDCOM files into structured Go data types,
// building on the lower-level parser package. It handles character encoding,
// validates the GEDCOM structure, and constructs a complete Document with
// cross-reference resolution.
//
// Example usage:
//
//	f, err := os.Open("family.ged")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer f.Close()
//
//	doc, err := decoder.Decode(f)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Found %d individuals\n", len(doc.Individuals()))
package decoder

import (
	"io"

	"github.com/cacack/gedcom-go/charset"
	"github.com/cacack/gedcom-go/gedcom"
	"github.com/cacack/gedcom-go/parser"
	"github.com/cacack/gedcom-go/version"
)

// Decode parses a GEDCOM file from an io.Reader and returns a Document.
// This is a convenience function that uses default options.
func Decode(r io.Reader) (*gedcom.Document, error) {
	return DecodeWithOptions(r, DefaultOptions())
}

// DecodeWithOptions parses a GEDCOM file with custom options.
func DecodeWithOptions(r io.Reader, opts *DecodeOptions) (*gedcom.Document, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	// Check context cancellation before starting
	if opts.Context != nil {
		select {
		case <-opts.Context.Done():
			return nil, opts.Context.Err()
		default:
		}
	}

	// Wrap reader with UTF-8 validation
	validatedReader := charset.NewReader(r)

	// Parse all lines
	p := parser.NewParser()
	lines, err := p.Parse(validatedReader)
	if err != nil {
		// Preserve charset errors in the error message
		return nil, err
	}

	// Check context after parsing
	if opts.Context != nil {
		select {
		case <-opts.Context.Done():
			return nil, opts.Context.Err()
		default:
		}
	}

	// Detect GEDCOM version
	detectedVersion, err := version.DetectVersion(lines)
	if err != nil {
		return nil, err
	}

	// Build document from lines
	doc := buildDocument(lines, detectedVersion)

	// Convert raw tags to proper entity types
	populateEntities(doc)

	return doc, nil
}

// buildDocument constructs a Document from parsed lines.
func buildDocument(lines []*parser.Line, ver gedcom.Version) *gedcom.Document {
	doc := &gedcom.Document{
		XRefMap: make(map[string]*gedcom.Record),
		Header:  &gedcom.Header{Version: ver},
		Trailer: &gedcom.Trailer{},
	}

	if len(lines) == 0 {
		return doc
	}

	// Build header
	buildHeader(doc, lines, ver)

	// Build records and XRefMap
	buildRecords(doc, lines)

	return doc
}

// buildHeader extracts header information from lines.
func buildHeader(doc *gedcom.Document, lines []*parser.Line, ver gedcom.Version) {
	inHead := false

	for _, line := range lines {
		if line.Level == 0 && line.Tag == "HEAD" {
			inHead = true
			continue
		}

		if line.Level == 0 {
			inHead = false
		}

		if !inHead {
			continue
		}

		// Extract header fields
		switch line.Tag {
		case "CHAR":
			doc.Header.Encoding = gedcom.Encoding(line.Value)
		case "SOUR":
			if line.Level == 1 {
				doc.Header.SourceSystem = line.Value
			}
		case "LANG":
			doc.Header.Language = line.Value
		case "COPR":
			doc.Header.Copyright = line.Value
		}
	}

	// Ensure header has a version
	if doc.Header.Version == "" {
		doc.Header.Version = ver
	}

	// Detect vendor from source system
	doc.Vendor = gedcom.DetectVendor(doc.Header.SourceSystem)
}

// buildRecords extracts records from lines and builds the XRefMap.
func buildRecords(doc *gedcom.Document, lines []*parser.Line) {
	var currentRecord *gedcom.Record
	var currentTags []*gedcom.Tag

	for _, line := range lines {
		// Level 0 lines are records or structural tags
		if line.Level == 0 {
			// Save previous record if exists
			if currentRecord != nil {
				currentRecord.Tags = currentTags
				doc.Records = append(doc.Records, currentRecord)
				currentTags = nil
			}

			// Skip HEAD and TRLR
			if line.Tag == "HEAD" || line.Tag == "TRLR" {
				currentRecord = nil
				continue
			}

			// Start new record
			currentRecord = &gedcom.Record{
				XRef:       line.XRef,
				Type:       gedcom.RecordType(line.Tag),
				Value:      line.Value,
				LineNumber: line.LineNumber,
			}

			// Index in XRefMap if it has an XRef
			if line.XRef != "" {
				doc.XRefMap[line.XRef] = currentRecord
			}

			continue
		}

		// Add tags to current record
		if currentRecord != nil {
			tag := &gedcom.Tag{
				Level:      line.Level,
				Tag:        line.Tag,
				Value:      line.Value,
				LineNumber: line.LineNumber,
			}
			currentTags = append(currentTags, tag)
		}
	}

	// Save last record
	if currentRecord != nil {
		currentRecord.Tags = currentTags
		doc.Records = append(doc.Records, currentRecord)
	}
}
