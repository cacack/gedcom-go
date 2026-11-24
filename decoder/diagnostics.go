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

	"github.com/cacack/gedcom-go/v3/internal/clip"
	"github.com/cacack/gedcom-go/v3/parser"
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

	// CodeInvalidLevel indicates the level number could not be parsed or is
	// invalid: non-numeric, negative, or deeper than the level field's two
	// digits allow (see parser.MaxNestingDepth). The line is always dropped.
	CodeInvalidLevel = "INVALID_LEVEL"

	// CodeInvalidXRef indicates a malformed cross-reference identifier.
	CodeInvalidXRef = "INVALID_XREF"

	// CodeBadLevelJump indicates a level jump of more than one step (e.g.,
	// `1 BIRT` directly followed by `4 DATE`, as seen in some real-world
	// genealogy exports). In lenient mode the level is clamped to prevLevel+1
	// and the document remains usable; emitted as SeverityWarning, not Error.
	// The line is always kept: a dropped line reports CodeInvalidLevel instead.
	CodeBadLevelJump = "BAD_LEVEL_JUMP"

	// CodeEmptyLine indicates a blank or whitespace-only line in the GEDCOM
	// data. No GEDCOM version's grammar has one, so strict mode fails on it;
	// lenient mode skips the line, losing nothing, and reports it as a
	// SeverityWarning. The first [MaxEmptyLineDiagnostics] blank lines are
	// reported one per line; one further diagnostic, at the next blank line,
	// counts the rest. It follows the other parse diagnostics.
	CodeEmptyLine = "EMPTY_LINE"

	// CodeInvalidEncoding indicates a byte that is not valid in the input's
	// encoding: a byte outside any valid UTF-8 sequence in input read as
	// UTF-8 (declared UTF-8 or ASCII, or undeclared). In lenient mode the byte
	// is replaced with U+FFFD and decoding continues; one diagnostic is
	// emitted per byte, with its column in the message, for the first
	// [MaxInvalidEncodingDiagnostics] bytes, and one more counts any bytes
	// after those. Strict mode fails with a *charset.InvalidUTF8Error instead.
	CodeInvalidEncoding = "INVALID_ENCODING"

	// CodeAmbiguousCodePage indicates a header declaring CHAR ANSI, which
	// names no code page, over bytes that read differently as Windows-1252
	// and Windows-1250. The file is decoded as Windows-1252 either way; if it
	// came from a Central European system, letters such as "č" have arrived
	// as "è". It is a SeverityWarning, emitted at most once per decode, in
	// lenient mode, at the first such byte, and never when
	// DecodeOptions.ANSIEncoding names the code page.
	CodeAmbiguousCodePage = "AMBIGUOUS_CODEPAGE"
)

// Entity-level diagnostic codes for semantic issues during entity population.
const (
	// CodeUnknownTag indicates an unrecognized tag was encountered.
	// These tags are preserved in raw form but not parsed into typed fields.
	// A tag the document's version defines, but not under the line's parent,
	// is reported as [CodeMisplacedTag] instead; a standard tag the decoder
	// does not yet read where the grammar allows it (a decoder gap) is still
	// reported with this code, as is a defined tag whose parent's tag allows
	// it under some other structure (see CodeMisplacedTag). Code that treats
	// UNKNOWN_TAG as "a line the decoder rejected" must also take
	// CodeMisplacedTag, which before v3.11.0 was reported with this code.
	// It is also reported for a level-0 record whose tag is neither a standard
	// record type nor an extension ("_" prefix), including a standard type in
	// the wrong case ("indi"); the record is kept with that Type and no entity.
	// A subordinate tag is reported once per line for its first
	// [MaxUnknownTagDiagnosticsPerTag] lines; one further diagnostic, at the
	// next line with that tag, counts the lines not reported individually.
	// The allowance is per tag name across the whole document, whatever the
	// record or parent structure; the counting diagnostics have an empty
	// Context and follow all other diagnostics, ordered by line.
	CodeUnknownTag = "UNKNOWN_TAG"

	// CodeMisplacedTag indicates a tag the document's GEDCOM version defines,
	// but under no structure with the tag of the line's parent: a MARR
	// directly under INDI, an EMAIL under a 5.5.1 INDI, a DATE under an OBJE
	// record. The fix is to move the line, where an UNKNOWN_TAG is renamed or
	// dropped. The message names the parent ("MARR is not allowed under
	// INDI"). Like CodeUnknownTag it is a SeverityWarning, the line is
	// preserved in raw form, and each tag name is reported line by line
	// [MaxUnknownTagDiagnosticsPerTag] times, then once more counting its
	// remaining lines (an allowance separate from UNKNOWN_TAG's). Parents
	// are matched by tag, so a tag any structure with that tag allows is not
	// reported as misplaced, and a tag under an extension or unknown parent
	// is reported as CodeUnknownTag, if at all: this is a best-effort hint,
	// not a conformance check. GEDCOM 7.0's CONT, which may continue any
	// line, is never misplaced. Level-0 lines are never reported with it.
	CodeMisplacedTag = "MISPLACED_TAG"

	// CodeInvalidValue indicates a value doesn't match the expected format.
	// The raw value is preserved, but typed parsing may have failed.
	CodeInvalidValue = "INVALID_VALUE"

	// CodeNonStandardDate indicates a DATE value that is not a GEDCOM date but
	// was read as one: a date spelled out in English ("June 1880",
	// "about 1873"; the list is on [gedcom.ParseDateLenient]). The typed Date
	// is set, the message names the GEDCOM form the value was read as
	// ("JUN 1880"), and the raw value is preserved and written back unchanged.
	// Emitted as SeverityWarning. Before v3.12.0 these values were reported as
	// CodeInvalidValue and left unparsed.
	CodeNonStandardDate = "NONSTANDARD_DATE"

	// CodeMissingRequired indicates a required subordinate tag is missing.
	CodeMissingRequired = "MISSING_REQUIRED"

	// CodeSkippedRecord indicates a record was skipped because its level-0 line
	// could not be parsed.
	// In lenient mode it is emitted when a record's level-0 line is dropped
	// (alongside that line's parse diagnostic): the record's subordinate lines,
	// up to the next level-0 line, are skipped rather than attached to the
	// preceding record. The message names the skipped line range.
	CodeSkippedRecord = "SKIPPED_RECORD"

	// CodeDuplicateXRef indicates a level-0 record defines a cross-reference
	// identifier an earlier record already defined. It carries SeverityError
	// and the line of the redefinition; the message names the first
	// definition's line. Both records are kept in Document.Records, and
	// Document.XRefMap (so every pointer) resolves to the first.
	CodeDuplicateXRef = "DUPLICATE_XREF"

	// CodeCharsetMismatch indicates the input was decoded in an encoding other
	// than the one its HEAD.CHAR line declares: a UTF-8 BOM overrode the
	// declaration, or bytes declared as ANSEL, ANSI/LATIN1 or ASCII were
	// recognized as UTF-8 (see charset.NewDetectingReader). Header.Encoding
	// keeps the declared value and Header.DetectedEncoding names the one
	// applied. Emitted as SeverityWarning, on the CHAR line.
	CodeCharsetMismatch = "CHARSET_MISMATCH"
)

// CodeDiagnosticsTruncated reports that a lenient decode left diagnostics out
// because [DecodeOptions.MaxDiagnostics] was reached. It is always the last
// diagnostic, its message gives the number left out, and its Line is 0, since
// it reports on the decode rather than on one line. Its severity is the
// highest among the diagnostics left out, so [Diagnostics.HasErrors] stays
// true when an error was one of them.
const CodeDiagnosticsTruncated = "DIAGNOSTICS_TRUNCATED"

// Diagnostic represents a single issue encountered during parsing or decoding.
// It can represent both parser-level errors and entity-level warnings.
type Diagnostic struct {
	// Line is the 1-based line number where the issue occurred.
	Line int

	// Severity indicates the importance level of this diagnostic.
	Severity Severity

	// Code is a machine-readable identifier for the diagnostic type.
	Code string

	// Message is a human-readable description of the issue. One built by
	// [NewDiagnostic], as every diagnostic the decoder reports is, holds at
	// most parser.MaxContextBytes bytes, so a value it quotes is clipped.
	Message string

	// Context provides the actual content that caused the issue. One built by
	// [NewDiagnostic] is clipped like Message: at most parser.MaxContextBytes
	// bytes, cut back to the start of a UTF-8 character and followed by "…".
	// The full content is not retained.
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
	fmt.Fprintf(&sb, "%d", d.Line)
	sb.WriteString(": ")
	sb.WriteString(d.Code)
	sb.WriteString(": ")
	sb.WriteString(d.Message)

	if d.Context != "" {
		sb.WriteString(" (context: ")
		fmt.Fprintf(&sb, "%q", d.Context)
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

// NewDiagnostic creates a new Diagnostic with the given parameters. Message
// and context are clipped to parser.MaxContextBytes (see [Diagnostic]).
func NewDiagnostic(line int, severity Severity, code, message, context string) Diagnostic {
	return Diagnostic{
		Line:     line,
		Severity: severity,
		Code:     code,
		Message:  clip.String(message, parser.MaxContextBytes),
		Context:  clip.String(context, parser.MaxContextBytes),
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
	fmt.Fprintf(&sb, "%d diagnostic(s):\n", len(ds))
	for _, d := range ds {
		sb.WriteString("  ")
		sb.WriteString(d.String())
		sb.WriteString("\n")
	}
	return sb.String()
}
