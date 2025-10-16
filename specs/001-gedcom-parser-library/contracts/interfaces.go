// Package contracts defines the public API interfaces for the go-gedcom library.
// These interfaces represent the contracts between packages and serve as
// documentation for the library's architecture.
//
// This file is for documentation purposes and is not compiled into the library.
package contracts

import (
	"context"
	"io"
	"time"
)

// Parser tokenizes GEDCOM files into line-based structures.
// It handles low-level concerns like line endings, encoding, and syntax.
type Parser interface {
	// ParseLine reads and tokenizes the next GEDCOM line.
	// Returns io.EOF when input is exhausted.
	ParseLine() (Line, error)

	// Reset prepares the parser to read from a new input.
	Reset(r io.Reader) error
}

// Line represents a single parsed GEDCOM line with metadata.
type Line struct {
	Level      int    // Hierarchy level (0-99)
	Tag        string // GEDCOM tag name
	Value      string // Tag value (may be empty)
	XRef       string // Cross-reference ID if present
	LineNumber int    // Original line number for error reporting
}

// Decoder converts parsed GEDCOM data into structured Document objects.
// It builds the high-level object model from the line stream.
type Decoder interface {
	// Decode reads an entire GEDCOM file and returns a Document.
	// This buffers all records in memory.
	Decode(r io.Reader) (*Document, error)

	// DecodeWithOptions provides configuration for decoding behavior.
	DecodeWithOptions(r io.Reader, opts DecodeOptions) (*Document, error)

	// DecodeStream processes GEDCOM records one at a time via callback.
	// This enables processing large files without buffering everything.
	DecodeStream(r io.Reader, handler RecordHandler) error
}

// DecodeOptions configures decoder behavior.
type DecodeOptions struct {
	// OnProgress is called periodically to report parsing progress.
	// Parameters: bytesRead, totalBytes, recordCount.
	// If nil, no progress reporting occurs.
	OnProgress ProgressFunc

	// MaxNestingDepth limits tag hierarchy depth (default: 100).
	// Prevents stack overflow from malicious deeply-nested files.
	MaxNestingDepth int

	// MaxRecordCount limits total records parsed (default: 1,000,000).
	// Prevents memory exhaustion from files with excessive records.
	MaxRecordCount int64

	// Timeout limits total parsing time (default: 5 minutes).
	// Prevents indefinite hangs on slow/corrupted input.
	Timeout time.Duration

	// StrictMode enables strict GEDCOM spec compliance.
	// When false, parser is more permissive with malformed input.
	StrictMode bool

	// Context allows cancellation of long-running operations.
	Context context.Context
}

// ProgressFunc is called during parsing to report progress.
type ProgressFunc func(bytesRead, totalBytes, recordCount int64)

// RecordHandler processes individual records during streaming decode.
// Returning an error stops decoding immediately.
type RecordHandler func(record *Record) error

// Document represents a complete GEDCOM file.
type Document struct {
	Header  Header
	Records []*Record
	Trailer Trailer
	Version Version

	// XRefMap provides fast lookup of records by cross-reference ID.
	XRefMap map[string]*Record
}

// Header contains GEDCOM file metadata.
type Header struct {
	Version      Version
	Encoding     Encoding
	SourceSystem string
	Date         time.Time
	Language     string
}

// Trailer marks the end of a GEDCOM file (usually just "0 TRLR").
type Trailer struct {
	// Empty for now, included for completeness
}

// Version identifies the GEDCOM specification version.
type Version int

const (
	Version55  Version = iota // GEDCOM 5.5
	Version551                // GEDCOM 5.5.1
	Version70                 // GEDCOM 7.0
)

// Encoding specifies character encoding used in the file.
type Encoding int

const (
	EncodingASCII   Encoding = iota // 7-bit ASCII
	EncodingUTF8                    // UTF-8 Unicode
	EncodingUTF16                   // UTF-16 Unicode
	EncodingLatin1                  // ISO-8859-1
	EncodingANSEL                   // ANSEL (genealogy-specific)
	EncodingUnknown                 // Unknown/undeclared
)

// Record is the base type for all GEDCOM records.
type Record struct {
	XRef       string
	Type       RecordType
	Tags       []Tag
	LineNumber int
}

// RecordType identifies the kind of GEDCOM record.
type RecordType string

const (
	RecordTypeIndividual  RecordType = "INDI"
	RecordTypeFamily      RecordType = "FAM"
	RecordTypeSource      RecordType = "SOUR"
	RecordTypeRepository  RecordType = "REPO"
	RecordTypeNote        RecordType = "NOTE"
	RecordTypeMediaObject RecordType = "OBJE"
	RecordTypeSubmitter   RecordType = "SUBM"
	RecordTypeSubmission  RecordType = "SUBN"
)

// Tag represents a GEDCOM tag-value pair with optional subtags.
type Tag struct {
	Level      int
	Tag        string
	Value      string
	XRef       string
	SubTags    []Tag
	LineNumber int
}

// Encoder writes GEDCOM data to an output stream.
type Encoder interface {
	// Encode writes a Document to the output stream.
	Encode(w io.Writer, doc *Document) error

	// EncodeWithOptions provides configuration for encoding behavior.
	EncodeWithOptions(w io.Writer, doc *Document, opts EncodeOptions) error
}

// EncodeOptions configures encoder behavior.
type EncodeOptions struct {
	// Version specifies the target GEDCOM version.
	// If zero, uses the document's existing version.
	Version Version

	// LineEnding specifies line terminator (CRLF, LF, or CR).
	// Default: LF (\n) on Unix, CRLF (\r\n) on Windows.
	LineEnding string

	// Encoding specifies output character encoding.
	// Default: UTF-8.
	Encoding Encoding

	// Indent adds visual indentation (not part of GEDCOM spec, for debugging).
	// Default: false (no indentation).
	Indent bool
}

// Validator checks GEDCOM data against specification rules.
type Validator interface {
	// Validate checks a Document for spec violations.
	// Returns a list of validation errors (may be warnings or errors).
	Validate(doc *Document) []ValidationError

	// ValidateWithOptions provides configuration for validation behavior.
	ValidateWithOptions(doc *Document, opts ValidateOptions) []ValidationError
}

// ValidateOptions configures validation behavior.
type ValidateOptions struct {
	// Version specifies which GEDCOM version to validate against.
	// If zero, uses the document's declared version.
	Version Version

	// CheckCircularReferences enables detection of impossible family relationships.
	// Example: person listed as their own ancestor.
	CheckCircularReferences bool

	// WarningsAsErrors treats all warnings as validation failures.
	WarningsAsErrors bool
}

// ValidationError represents a single validation problem.
type ValidationError struct {
	// Severity indicates if this is an error or warning.
	Severity Severity

	// Code provides a machine-readable error identifier.
	Code ValidationCode

	// Message is a human-readable description.
	Message string

	// Location identifies where the problem occurred.
	Location ErrorLocation

	// Suggestion provides guidance for fixing the issue.
	Suggestion string
}

// Severity classifies validation problems.
type Severity int

const (
	SeverityError   Severity = iota // Spec violation, invalid GEDCOM
	SeverityWarning                 // Non-standard but parseable
	SeverityInfo                    // Informational note
)

// ValidationCode provides machine-readable error identification.
type ValidationCode string

const (
	CodeMissingRequired    ValidationCode = "MISSING_REQUIRED"
	CodeInvalidFormat      ValidationCode = "INVALID_FORMAT"
	CodeBrokenReference    ValidationCode = "BROKEN_REFERENCE"
	CodeCircularReference  ValidationCode = "CIRCULAR_REFERENCE"
	CodeNonStandardFormat  ValidationCode = "NON_STANDARD_FORMAT"
	CodeDeprecatedTag      ValidationCode = "DEPRECATED_TAG"
)

// ErrorLocation identifies where a problem occurred in the input.
type ErrorLocation struct {
	XRef       string // Record cross-reference if applicable
	LineNumber int    // Line number in original file
	Context    string // Snippet of problematic content
}

// Converter transforms GEDCOM data between versions.
type Converter interface {
	// Convert transforms a Document from one version to another.
	// Returns a new Document and a ConversionReport describing changes.
	Convert(doc *Document, targetVersion Version) (*Document, ConversionReport, error)

	// ConvertWithOptions provides configuration for conversion behavior.
	ConvertWithOptions(doc *Document, targetVersion Version, opts ConvertOptions) (*Document, ConversionReport, error)
}

// ConvertOptions configures conversion behavior.
type ConvertOptions struct {
	// PreserveCustomTags attempts to retain extension tags when possible.
	// Default: true.
	PreserveCustomTags bool

	// FailOnDataLoss returns an error if conversion would lose data.
	// When false, data loss is reported in ConversionReport but conversion proceeds.
	// Default: false (report but continue).
	FailOnDataLoss bool
}

// ConversionReport documents changes made during version conversion.
type ConversionReport struct {
	// SourceVersion is the original GEDCOM version.
	SourceVersion Version

	// TargetVersion is the destination GEDCOM version.
	TargetVersion Version

	// TagsConverted lists tags that were transformed.
	TagsConverted []TagConversion

	// DataLost lists information that couldn't be preserved.
	DataLost []DataLoss

	// Warnings lists non-critical issues encountered.
	Warnings []string
}

// TagConversion documents a tag transformation.
type TagConversion struct {
	OldTag     string
	NewTag     string
	LineNumber int
	Rationale  string
}

// DataLoss documents information that couldn't be converted.
type DataLoss struct {
	Tag        string
	Value      string
	LineNumber int
	Reason     string
}

// VersionDetector identifies the GEDCOM version of a file.
type VersionDetector interface {
	// Detect analyzes a GEDCOM file and returns its version.
	// It reads the header and analyzes tag usage to determine version.
	Detect(r io.Reader) (Version, error)
}
