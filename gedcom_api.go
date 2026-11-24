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
//	    fmt.Println(ind.FullName())
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
// Report decode diagnostics and validation issues together. The two lists
// overlap on one code, DUPLICATE_XREF; [WithoutDiagnosed] drops the repeats:
//
//	result, err := gedcomgo.DecodeWithDiagnostics(file)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	issues := gedcomgo.ValidateAll(result.Document)
//	issues = gedcomgo.WithoutDiagnosed(issues, result.Diagnostics)
//
// Convert between versions:
//
//	converted, report, err := gedcomgo.Convert(doc, gedcomgo.Version70)
//
// # Power Users
//
// For advanced use cases requiring custom options, import the underlying packages directly:
//
//   - github.com/cacack/gedcom-go/v3/decoder - Custom decode options, progress callbacks, diagnostics
//   - github.com/cacack/gedcom-go/v3/encoder - Custom line endings, encoding options
//   - github.com/cacack/gedcom-go/v3/validator - Configurable validation rules, quality reports
//   - github.com/cacack/gedcom-go/v3/converter - Custom conversion options, strict mode
package gedcomgo

import (
	"io"
	"slices"

	"github.com/cacack/gedcom-go/v3/converter"
	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/validator"
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

	// DecodeOptions configures GEDCOM decoding. See [decoder.DecodeOptions].
	DecodeOptions = decoder.DecodeOptions

	// EncodeOptions configures GEDCOM encoding. See [encoder.EncodeOptions].
	EncodeOptions = encoder.EncodeOptions

	// ValidateOptions configures GEDCOM validation. See [validator.ValidateOptions].
	ValidateOptions = validator.ValidateOptions
)

// Error re-exports, so an error returned by this package can be matched with
// errors.Is without importing the package that defines it.
var (
	// ErrNilDocument is returned by Encode and EncodeWithOptions when doc is nil.
	// See [encoder.ErrNilDocument].
	ErrNilDocument = encoder.ErrNilDocument

	// ErrLineBreak is returned by Encode and EncodeWithOptions when a tag
	// name or cross-reference identifier contains a line break.
	// See [encoder.ErrLineBreak].
	ErrLineBreak = encoder.ErrLineBreak
)

// DefaultDecodeOptions returns the default decoding options.
func DefaultDecodeOptions() *DecodeOptions {
	return decoder.DefaultOptions()
}

// DefaultEncodeOptions returns the default encoding options.
func DefaultEncodeOptions() *EncodeOptions {
	return encoder.DefaultOptions()
}

// DefaultValidateOptions returns the default validation options.
func DefaultValidateOptions() *ValidateOptions {
	return validator.DefaultOptions()
}

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
// This is the simplest way to parse a GEDCOM file using default options,
// which decode leniently: malformed lines are recovered from or skipped
// rather than failing the decode.
//
// For custom options (strict mode, progress callbacks, context cancellation),
// use [DecodeWithOptions].
func Decode(r io.Reader) (*Document, error) {
	return decoder.Decode(r)
}

// DecodeWithOptions parses a GEDCOM file with the given options.
// If opts is nil, default options are used. [DecodeOptions.StrictMode] means
// the same thing here as in [DecodeWithDiagnostics]; this function returns the
// document and discards the diagnostics.
func DecodeWithOptions(r io.Reader, opts *DecodeOptions) (*Document, error) {
	return decoder.DecodeWithOptions(r, opts)
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
// The output uses LF line endings; pass [EncodeOptions.LineEnding] = "\r\n"
// to write CRLF.
//
// For custom options (line endings, target version, line wrapping, custom-tag
// preservation), use [EncodeWithOptions].
//
// Returns [ErrNilDocument] if doc is nil, and an error wrapping [ErrLineBreak]
// if a tag name or identifier contains a line break. See the [encoder] package
// documentation for how nil values inside a document are handled.
func Encode(w io.Writer, doc *Document) error {
	return encoder.Encode(w, doc)
}

// EncodeWithOptions writes a GEDCOM document with the given options.
// If opts is nil, default options are used.
//
// Returns [ErrNilDocument] if doc is nil, and an error wrapping [ErrLineBreak]
// if a tag name or identifier contains a line break. See the [encoder] package
// documentation for how nil values inside a document are handled.
func EncodeWithOptions(w io.Writer, doc *Document, opts *EncodeOptions) error {
	return encoder.EncodeWithOptions(w, doc, opts)
}

// Validate validates a GEDCOM document with default options and returns its
// issues as errors. It is [ValidateAll] in the error-interface shape: the same
// checks and the same results, each element a *[Issue], so errors.As recovers
// the code, severity and line number. At the default strictness the result
// holds warnings as well as errors.
//
// For custom validation configuration (strictness, MaxErrors, SkipRules),
// use [ValidateWithOptions] or [ValidateAllWithOptions].
func Validate(doc *Document) []error {
	return validator.New().Validate(doc)
}

// ValidateWithOptions validates a GEDCOM document with the given options and
// returns its issues as errors, each element a *[Issue]. If opts is nil,
// default options are used.
//
// Every [ValidateOptions] field applies, exactly as it does to
// [ValidateAllWithOptions]: Strictness selects the severities returned,
// SkipRules drops the listed codes, MaxErrors caps the count, and the
// remaining fields configure or disable individual checks. No field is
// ignored.
//
// A done Context stops validation early, and only the issues found before
// it are returned, with nothing marking them incomplete: a caller that sets
// Context should check the context's Err afterwards, or call
// validator.NewWithOptions(opts).ValidateAllWithError, which reports it.
func ValidateWithOptions(doc *Document, opts *ValidateOptions) []error {
	return validator.NewWithOptions(opts).Validate(doc)
}

// ValidateAll returns every validation finding as Issues with severity levels:
// header, date logic, cross-reference, record structure, nil element, XRef
// length, note pointer, duplicate and encoding checks. [Validate] returns the same result
// as []error.
//
// Issues are categorized by severity: Error, Warning, and Info.
// For custom validation configuration (strictness, thresholds, skip rules),
// use [ValidateAllWithOptions].
func ValidateAll(doc *Document) []Issue {
	return validator.New().ValidateAll(doc)
}

// ValidateAllWithOptions runs comprehensive validation with the given options.
// If opts is nil, default options are used.
//
// A done Context stops validation early, and only the issues found before
// it are returned, with nothing marking them incomplete: a caller that sets
// Context should check the context's Err afterwards, or call
// validator.NewWithOptions(opts).ValidateAllWithError, which reports it.
func ValidateAllWithOptions(doc *Document, opts *ValidateOptions) []Issue {
	return validator.NewWithOptions(opts).ValidateAll(doc)
}

// sharedFindingCodes are the codes the decoder and the validator both report,
// under the same string, for the same defect. [WithoutDiagnosed] removes that
// overlap; TestSharedFindingCodes fails when the packages gain another.
var sharedFindingCodes = []string{validator.CodeDuplicateXRef}

// WithoutDiagnosed returns issues without the findings diagnostics already
// holds, for a caller that reports a decode's diagnostics and the decoded
// document's validation issues together.
//
// The decoder and the validator each report a redefined XRef as
// DUPLICATE_XREF ([decoder.CodeDuplicateXRef], [validator.CodeDuplicateXRef]),
// the one code they share, so the two lists hold one finding each for the same
// line. An issue is dropped when its code is DUPLICATE_XREF and a diagnostic
// has that code on the issue's LineNumber for the issue's RecordXRef (a
// diagnostic with no XRef, as [decoder.NewDiagnostic] builds, matches on the
// line alone). Every other issue is kept, in order: one with no line (a
// record built in code), one for a record given another XRef after decoding,
// and one the decoder did not report, as when the document was never
// decoded, was decoded in strict mode, or [DecodeOptions.MaxDiagnostics] cut
// the diagnostic off.
//
// issues is not modified. Pass the diagnostics of the decode that produced
// the validated document; lines from another file would match by accident.
func WithoutDiagnosed(issues []Issue, diagnostics decoder.Diagnostics) []Issue {
	type finding struct {
		code string
		line int
		xref string
	}
	diagnosed := make(map[finding]bool)
	for _, d := range diagnostics {
		if slices.Contains(sharedFindingCodes, d.Code) {
			diagnosed[finding{d.Code, d.Line, d.XRef}] = true
		}
	}
	kept := make([]Issue, 0, len(issues))
	for _, issue := range issues {
		if issue.LineNumber > 0 &&
			(diagnosed[finding{issue.Code, issue.LineNumber, issue.RecordXRef}] ||
				diagnosed[finding{issue.Code, issue.LineNumber, ""}]) {
			continue
		}
		kept = append(kept, issue)
	}
	return kept
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
