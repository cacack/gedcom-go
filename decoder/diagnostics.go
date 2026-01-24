// diagnostics.go provides diagnostic types for lenient parsing mode.
//
// The Diagnostic type unifies parser errors and entity-level issues, enabling
// the decoder to collect multiple problems without stopping on the first error.
// This supports lenient parsing mode where malformed GEDCOM files can be
// partially processed while tracking all encountered issues.

package decoder

import (
	"fmt"
	"strings"
)

// Severity represents the severity level of a diagnostic.
// This type mirrors validator.Severity to avoid import cycles between
// decoder and validator packages.
type Severity int

const (
	// SeverityError indicates a data integrity issue that must be fixed.
	SeverityError Severity = iota

	// SeverityWarning indicates a potential problem that should be reviewed.
	SeverityWarning

	// SeverityInfo indicates an informational data quality suggestion.
	SeverityInfo
)

// String returns the human-readable name of the severity level.
func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "ERROR"
	case SeverityWarning:
		return "WARNING"
	case SeverityInfo:
		return "INFO"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", s)
	}
}

// Parse-phase error codes for diagnostic reporting.
const (
	// CodeSyntaxError indicates a general syntax error in the GEDCOM line.
	CodeSyntaxError = "SYNTAX_ERROR"

	// CodeInvalidLevel indicates the level number could not be parsed or is invalid.
	CodeInvalidLevel = "INVALID_LEVEL"

	// CodeInvalidXRef indicates a malformed cross-reference identifier.
	CodeInvalidXRef = "INVALID_XREF"

	// CodeBadLevelJump indicates an invalid level increment (e.g., jumping from 0 to 2).
	CodeBadLevelJump = "BAD_LEVEL_JUMP"

	// CodeEmptyLine indicates an unexpected empty line in the GEDCOM data.
	CodeEmptyLine = "EMPTY_LINE"
)

// Entity-level diagnostic codes for semantic issues during entity population.
const (
	// CodeUnknownTag indicates an unrecognized tag was encountered.
	// These tags are preserved in raw form but not parsed into typed fields.
	CodeUnknownTag = "UNKNOWN_TAG"

	// CodeInvalidValue indicates a value doesn't match the expected format.
	// The raw value is preserved, but typed parsing may have failed.
	CodeInvalidValue = "INVALID_VALUE"

	// CodeMissingRequired indicates a required subordinate tag is missing.
	CodeMissingRequired = "MISSING_REQUIRED"

	// CodeSkippedRecord indicates an entire record was skipped due to errors.
	CodeSkippedRecord = "SKIPPED_RECORD"
)

// Diagnostic represents a single issue encountered during parsing or decoding.
// It can represent both parser-level errors and entity-level warnings.
type Diagnostic struct {
	// Line is the 1-based line number where the issue occurred.
	Line int

	// Severity indicates the importance level of this diagnostic.
	Severity Severity

	// Code is a machine-readable identifier for the diagnostic type.
	Code string

	// Message is a human-readable description of the issue.
	Message string

	// Context provides the actual content that caused the issue.
	Context string
}

// String returns a human-friendly representation of the diagnostic.
//
//nolint:gocritic // Value receiver intentional for immutability
func (d Diagnostic) String() string {
	var sb strings.Builder

	// Format: [SEVERITY] line N: CODE: Message (context: "...")
	sb.WriteString("[")
	sb.WriteString(d.Severity.String())
	sb.WriteString("] line ")
	sb.WriteString(fmt.Sprintf("%d", d.Line))
	sb.WriteString(": ")
	sb.WriteString(d.Code)
	sb.WriteString(": ")
	sb.WriteString(d.Message)

	if d.Context != "" {
		sb.WriteString(" (context: ")
		sb.WriteString(fmt.Sprintf("%q", d.Context))
		sb.WriteString(")")
	}

	return sb.String()
}

// Error implements the error interface, returning a formatted error string.
//
//nolint:gocritic // Value receiver intentional for immutability
func (d Diagnostic) Error() string {
	return d.String()
}

// NewDiagnostic creates a new Diagnostic with the given parameters.
func NewDiagnostic(line int, severity Severity, code, message, context string) Diagnostic {
	return Diagnostic{
		Line:     line,
		Severity: severity,
		Code:     code,
		Message:  message,
		Context:  context,
	}
}

// NewParseError creates a new Diagnostic with SeverityError for parse-phase errors.
// This is a convenience function for the common case of parser errors.
func NewParseError(line int, code, message, context string) Diagnostic {
	return NewDiagnostic(line, SeverityError, code, message, context)
}

// Diagnostics is a collection of Diagnostic instances with helper methods.
type Diagnostics []Diagnostic

// HasErrors returns true if any diagnostic has SeverityError.
func (ds Diagnostics) HasErrors() bool {
	for _, d := range ds {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Errors returns a new Diagnostics containing only error-severity diagnostics.
func (ds Diagnostics) Errors() Diagnostics {
	var result Diagnostics
	for _, d := range ds {
		if d.Severity == SeverityError {
			result = append(result, d)
		}
	}
	return result
}

// Warnings returns a new Diagnostics containing only warning-severity diagnostics.
func (ds Diagnostics) Warnings() Diagnostics {
	var result Diagnostics
	for _, d := range ds {
		if d.Severity == SeverityWarning {
			result = append(result, d)
		}
	}
	return result
}

// String returns a formatted multi-line summary of all diagnostics.
func (ds Diagnostics) String() string {
	if len(ds) == 0 {
		return "no diagnostics"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d diagnostic(s):\n", len(ds)))
	for _, d := range ds {
		sb.WriteString("  ")
		sb.WriteString(d.String())
		sb.WriteString("\n")
	}
	return sb.String()
}
