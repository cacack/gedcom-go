// issue.go provides enhanced validation issue types with severity levels and rich context.
//
// The Issue type complements the existing ValidationError type, providing more detailed
// information for data quality validation including severity classification, error codes,
// and contextual details for actionable diagnostics.

package validator

import (
	"fmt"
	"strings"
)

// Severity represents the severity level of a validation issue.
type Severity int

const (
	// SeverityError indicates a data integrity issue that must be fixed.
	// Examples: death before birth, orphaned cross-references.
	SeverityError Severity = iota

	// SeverityWarning indicates a potential problem that should be reviewed.
	// Examples: unusual age at marriage, missing recommended fields.
	SeverityWarning

	// SeverityInfo indicates an informational data quality suggestion.
	// Examples: missing sources, potential duplicates.
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

// Error codes for date logic validation.
const (
	// CodeDeathBeforeBirth indicates a person's death date is before their birth date.
	CodeDeathBeforeBirth = "DEATH_BEFORE_BIRTH"

	// CodeChildBeforeParent indicates a child was born before their parent.
	CodeChildBeforeParent = "CHILD_BEFORE_PARENT"

	// CodeMarriageBeforeBirth indicates a marriage occurred before one spouse was born.
	CodeMarriageBeforeBirth = "MARRIAGE_BEFORE_BIRTH"

	// CodeImpossibleAge indicates an age that is biologically implausible (e.g., >120 years).
	CodeImpossibleAge = "IMPOSSIBLE_AGE"

	// CodeUnreasonableParentAge indicates a parent's age at child's birth is implausible.
	// Used when parent is too young (e.g., <12) or too old (e.g., mother >55, father >90).
	CodeUnreasonableParentAge = "UNREASONABLE_PARENT_AGE"
)

// Error codes for cross-reference validation.
const (
	// CodeOrphanedFAMC indicates a FAMC reference points to a non-existent family.
	CodeOrphanedFAMC = "ORPHANED_FAMC"

	// CodeOrphanedFAMS indicates a FAMS reference points to a non-existent family.
	CodeOrphanedFAMS = "ORPHANED_FAMS"

	// CodeOrphanedHUSB indicates a HUSB reference points to a non-existent individual.
	CodeOrphanedHUSB = "ORPHANED_HUSB"

	// CodeOrphanedWIFE indicates a WIFE reference points to a non-existent individual.
	CodeOrphanedWIFE = "ORPHANED_WIFE"

	// CodeOrphanedCHIL indicates a CHIL reference points to a non-existent individual.
	CodeOrphanedCHIL = "ORPHANED_CHIL"

	// CodeOrphanedSOUR indicates a SOUR reference points to a non-existent source.
	CodeOrphanedSOUR = "ORPHANED_SOUR"
)

// Error codes for duplicate detection.
const (
	// CodePotentialDuplicate indicates two records may represent the same entity.
	CodePotentialDuplicate = "POTENTIAL_DUPLICATE"
)

// Error codes for data quality validation.
const (
	// CodeMissingBirthDate indicates an individual has no birth date recorded.
	CodeMissingBirthDate = "MISSING_BIRTH_DATE"

	// CodeMissingDeathDate indicates a deceased individual has no death date recorded.
	CodeMissingDeathDate = "MISSING_DEATH_DATE"

	// CodeMissingName indicates an individual has no name recorded.
	CodeMissingName = "MISSING_NAME"

	// CodeNoSources indicates a record has no source citations.
	CodeNoSources = "NO_SOURCES"
)

// Error codes for custom tag registry validation.
const (
	// CodeInvalidTagParent indicates a custom tag appears under an invalid parent tag.
	CodeInvalidTagParent = "INVALID_TAG_PARENT"

	// CodeInvalidTagValue indicates a custom tag's value does not match its expected pattern.
	CodeInvalidTagValue = "INVALID_TAG_VALUE"

	// CodeUnknownCustomTag indicates an underscore-prefixed tag not in the registry.
	CodeUnknownCustomTag = "UNKNOWN_CUSTOM_TAG"
)

// Error codes for header validation.
const (
	// CodeMissingSUBM indicates the header is missing a required SUBM (Submitter) reference.
	// GEDCOM 5.5 and 5.5.1 require exactly one SUBM reference with cardinality {1:1}.
	// GEDCOM 7.0 made SUBM optional.
	CodeMissingSUBM = "MISSING_SUBM"
)

// Error codes for XRef validation.
const (
	// CodeXRefTooLong indicates an XRef identifier exceeds the 20-character limit
	// specified by GEDCOM 5.5 and 5.5.1. This limit was removed in GEDCOM 7.0.
	CodeXRefTooLong = "XREF_TOO_LONG"
)

// Error codes for encoding validation.
const (
	// CodeInvalidEncodingForVersion indicates the file's encoding is not supported
	// by the detected GEDCOM version. GEDCOM 7.0 requires UTF-8 exclusively.
	CodeInvalidEncodingForVersion = "INVALID_ENCODING_FOR_VERSION"

	// CodeBannedControlCharacter indicates a banned C0 control character was found.
	// GEDCOM 7.0 bans U+0000-U+001F except TAB (U+0009), LF (U+000A), CR (U+000D).
	CodeBannedControlCharacter = "BANNED_CONTROL_CHARACTER"
)

// Issue represents a validation finding with severity, context, and actionable information.
type Issue struct {
	// Severity indicates the importance level of this issue.
	Severity Severity

	// Code is a machine-readable identifier for the issue type.
	// Use the Code* constants defined in this package.
	Code string

	// Message is a human-readable description of the issue.
	Message string

	// RecordXRef is the cross-reference of the primary affected record (e.g., "@I1@").
	RecordXRef string

	// RelatedXRef is the cross-reference of a related record, if applicable.
	// For example, when checking if a child was born before a parent,
	// RecordXRef would be the child and RelatedXRef would be the parent.
	RelatedXRef string

	// Details contains additional context as key-value pairs.
	// Common keys include "field", "value", "expected", "actual".
	Details map[string]string
}

// Error implements the error interface, returning a formatted error string.
//
//nolint:gocritic // Value receiver intentional for immutability
func (i Issue) Error() string {
	return i.String()
}

// String returns a human-friendly representation of the issue.
func (i Issue) String() string {
	var sb strings.Builder

	// Format: [SEVERITY] CODE: Message (XRef: @I1@)
	sb.WriteString("[")
	sb.WriteString(i.Severity.String())
	sb.WriteString("] ")
	sb.WriteString(i.Code)
	sb.WriteString(": ")
	sb.WriteString(i.Message)

	if i.RecordXRef != "" {
		sb.WriteString(" (")
		sb.WriteString(i.RecordXRef)
		if i.RelatedXRef != "" {
			sb.WriteString(" -> ")
			sb.WriteString(i.RelatedXRef)
		}
		sb.WriteString(")")
	}

	return sb.String()
}

// NewIssue creates a new Issue with the given severity, code, message, and record XRef.
// The Details map is initialized to an empty map.
func NewIssue(severity Severity, code, message, recordXRef string) Issue {
	return Issue{
		Severity:   severity,
		Code:       code,
		Message:    message,
		RecordXRef: recordXRef,
		Details:    make(map[string]string),
	}
}

// WithRelatedXRef returns a copy of the Issue with the RelatedXRef set.
//
//nolint:gocritic // Value receiver intentional for immutability
func (i Issue) WithRelatedXRef(xref string) Issue {
	i.RelatedXRef = xref
	return i
}

// WithDetail returns a copy of the Issue with an additional detail key-value pair.
// If Details is nil, it initializes the map first.
//
//nolint:gocritic // Value receiver intentional for immutability
func (i Issue) WithDetail(key, value string) Issue {
	if i.Details == nil {
		i.Details = make(map[string]string)
	} else {
		// Create a copy of the map to maintain immutability
		newDetails := make(map[string]string, len(i.Details)+1)
		for k, v := range i.Details {
			newDetails[k] = v
		}
		i.Details = newDetails
	}
	i.Details[key] = value
	return i
}

// FilterBySeverity returns a slice containing only issues with the specified severity.
func FilterBySeverity(issues []Issue, severity Severity) []Issue {
	var result []Issue
	for _, issue := range issues {
		if issue.Severity == severity {
			result = append(result, issue)
		}
	}
	return result
}

// FilterByCode returns a slice containing only issues with the specified code.
func FilterByCode(issues []Issue, code string) []Issue {
	var result []Issue
	for _, issue := range issues {
		if issue.Code == code {
			result = append(result, issue)
		}
	}
	return result
}
