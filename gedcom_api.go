// Package gedcomgo provides a unified API for processing GEDCOM genealogical data files.
//
// This package is the recommended entry point for most users. It provides simple,
// high-level functions for common operations while re-exporting the most frequently
// used types for single-import convenience.
//
// # Quick Start
//
// Parse a GEDCOM file:
//
//	file, _ := os.Open("family.ged")
//	doc, err := gedcomgo.Decode(file)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, ind := range doc.Individuals() {
//	    fmt.Println(ind.Names[0].Full)
//	}
//
// Write a GEDCOM file:
//
//	file, _ := os.Create("output.ged")
//	err := gedcomgo.Encode(file, doc)
//
// Validate a document:
//
//	errors := gedcomgo.Validate(doc)
//	for _, err := range errors {
//	    fmt.Println(err)
//	}
//
// Convert between versions:
//
//	converted, report, err := gedcomgo.Convert(doc, gedcomgo.Version70)
//
// # Power Users
//
// For advanced use cases requiring custom options, import the underlying packages directly:
//
//   - github.com/cacack/gedcom-go/decoder - Custom decode options, progress callbacks, diagnostics
//   - github.com/cacack/gedcom-go/encoder - Custom line endings, encoding options
//   - github.com/cacack/gedcom-go/validator - Configurable validation rules, quality reports
//   - github.com/cacack/gedcom-go/converter - Custom conversion options, strict mode
package gedcomgo

import (
	"io"

	"github.com/cacack/gedcom-go/converter"
	"github.com/cacack/gedcom-go/decoder"
	"github.com/cacack/gedcom-go/encoder"
	"github.com/cacack/gedcom-go/gedcom"
	"github.com/cacack/gedcom-go/validator"
)

// Type re-exports for single-import convenience.
// These allow users to work with common types without importing multiple packages.
type (
	// Document represents a complete GEDCOM file with all its records.
	// Use Individuals(), Families(), Sources() to access typed collections.
	Document = gedcom.Document

	// Individual represents a person in the GEDCOM file.
	Individual = gedcom.Individual

	// Family represents a family unit (husband, wife, and children).
	Family = gedcom.Family

	// Version represents a GEDCOM specification version.
	Version = gedcom.Version

	// DecodeResult contains the result of decoding a GEDCOM file with diagnostics.
	// In lenient mode, Document may contain partial data even when diagnostics are present.
	DecodeResult = decoder.DecodeResult

	// Issue represents a validation finding with severity, context, and actionable information.
	Issue = validator.Issue

	// ConversionReport contains the results of a GEDCOM version conversion.
	ConversionReport = gedcom.ConversionReport
)

// Version constants for convenience.
const (
	// Version55 represents GEDCOM 5.5 specification.
	Version55 Version = gedcom.Version55

	// Version551 represents GEDCOM 5.5.1 specification.
	Version551 Version = gedcom.Version551

	// Version70 represents GEDCOM 7.0 specification.
	Version70 Version = gedcom.Version70
)

// Decode parses a GEDCOM file from an io.Reader and returns a Document.
// This is the simplest way to parse a GEDCOM file using default options.
//
// For custom options (progress callbacks, context cancellation), use the
// decoder package directly: decoder.DecodeWithOptions().
func Decode(r io.Reader) (*Document, error) {
	return decoder.Decode(r)
}

// DecodeWithDiagnostics parses a GEDCOM file and returns both the document and any diagnostics.
// In lenient mode (the default), parse errors are collected as diagnostics rather than
// stopping parsing, allowing partial documents to be returned.
//
// For custom options (strict mode, progress callbacks), use the
// decoder package directly: decoder.DecodeWithDiagnostics() with custom options.
func DecodeWithDiagnostics(r io.Reader) (*DecodeResult, error) {
	return decoder.DecodeWithDiagnostics(r, nil)
}

// Encode writes a GEDCOM document to a writer using default options.
// The output uses CRLF line endings as per the GEDCOM specification.
//
// For custom options (line endings, encoding), use the
// encoder package directly: encoder.EncodeWithOptions().
func Encode(w io.Writer, doc *Document) error {
	return encoder.Encode(w, doc)
}

// Validate validates a GEDCOM document and returns any validation errors.
// This performs basic structural validation including cross-reference checks
// and required field validation.
//
// For comprehensive validation with severity levels, use ValidateAll().
// For custom validation configuration, use the validator package directly.
func Validate(doc *Document) []error {
	return validator.New().Validate(doc)
}

// ValidateAll returns comprehensive validation as Issues with severity levels.
// This is the enhanced API that provides more detail than Validate(), including
// date logic validation, reference checking, and quality analysis.
//
// Issues are categorized by severity: Error, Warning, and Info.
// For custom validation configuration (strictness, thresholds), use the
// validator package directly: validator.NewWithConfig().
func ValidateAll(doc *Document) []Issue {
	return validator.New().ValidateAll(doc)
}

// Convert converts a GEDCOM document to the target version.
// It returns the converted document, a report detailing any transformations
// or data loss, and an error if conversion failed.
//
// Supported conversions:
//   - 5.5 <-> 5.5.1 (minimal changes, mostly compatible)
//   - 5.5 <-> 7.0 (text handling, xref normalization)
//   - 5.5.1 <-> 7.0 (text handling, xref normalization)
//
// For custom options (strict data loss mode, validation), use the
// converter package directly: converter.ConvertWithOptions().
func Convert(doc *Document, targetVersion Version) (converted *Document, report *ConversionReport, err error) {
	return converter.Convert(doc, targetVersion)
}
