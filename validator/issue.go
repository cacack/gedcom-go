// issue.go provides the validation Issue type with severity levels and rich context.
//
// Issue is the single finding type of this package: every check reports one, with a
// severity classification, a Code* constant and contextual details for actionable
// diagnostics. Issue implements error, and Validator.Validate returns *Issue values.

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
//
// The three ordering codes (DEATH_BEFORE_BIRTH, CHILD_BEFORE_PARENT,
// MARRIAGE_BEFORE_BIRTH) are SeverityError when gedcom.Date.Bounds proves the
// order and SeverityWarning when the dates are too imprecise to prove it, or
// when either date is approximate (ABT, CAL or EST). An INT date counts as
// exact. Under StrictnessRelaxed, ValidateAll, ValidateDateLogic and
// StreamingValidator drop the SeverityWarning findings; QualityReport ignores
// Strictness and keeps them.
const (
	// CodeDeathBeforeBirth indicates a person's death date is before their birth
	// date: proven at SeverityError, possible at SeverityWarning.
	CodeDeathBeforeBirth = "DEATH_BEFORE_BIRTH"

	// CodeChildBeforeParent indicates a child was born before their parent:
	// proven at SeverityError, possible at SeverityWarning.
	CodeChildBeforeParent = "CHILD_BEFORE_PARENT"

	// CodeMarriageBeforeBirth indicates a marriage occurred before one spouse was
	// born: proven at SeverityError, possible at SeverityWarning.
	CodeMarriageBeforeBirth = "MARRIAGE_BEFORE_BIRTH"

	// CodeImpossibleAge indicates an age that is biologically implausible (e.g., >120 years).
	CodeImpossibleAge = "IMPOSSIBLE_AGE"

	// CodeInvalidDate indicates a date that does not exist on its calendar,
	// such as 31 FEB 1900 or 29 FEB 1900, including the end date of a range
	// or period, or whose dual year is not the following year (1750/53), or
	// a range or period that ends before it starts (BET 1860 AND 1850).
	// The dual year is checked on every date; the day is checked on
	// Gregorian, Julian, Hebrew and French Republican dates but not on
	// partial dates, as in gedcom.Date.Validate.
	// A value gedcom.ParseDate rejects, such as 0 JAN 1850, is not reported
	// here; the decoder reports it as an INVALID_VALUE diagnostic.
	CodeInvalidDate = "INVALID_DATE"

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

	// CodeBrokenXRef indicates a pointer in a record's raw tags names no record
	// in the document. It carries SeverityError, the line of the pointing tag,
	// the missing pointer as RelatedXRef and the tag name under Details["tag"].
	//
	// It covers every pointer the typed Orphaned* checks do not: those are
	// reported under their own code instead, never under both. In a GEDCOM
	// 5.5/5.5.1 document a spaced pointer ("@N 1@") is checked like any other.
	CodeBrokenXRef = "BROKEN_XREF"
)

// Error codes for record structure validation.
const (
	// CodeMissingRequiredField was reported for an individual record with no
	// NAME.
	//
	// Deprecated: no check reports it. INDI.NAME is optional ({0:M}) in GEDCOM
	// 5.5, 5.5.1 and 7.0, so a nameless individual is reported as the
	// data-quality heuristic [CodeMissingName] instead. A SkipRules entry or
	// code match on this constant now matches nothing; use [CodeMissingName].
	CodeMissingRequiredField = "MISSING_REQUIRED_FIELD"

	// CodeEmptyFamily indicates a family record has no HUSB, WIFE or CHIL. It
	// is a data-quality heuristic, not a spec rule: every GEDCOM version
	// allows such a family. It carries SeverityWarning and the line of the
	// record, and [QualityReport] files it under CompletenessIssues.
	CodeEmptyFamily = "EMPTY_FAMILY"

	// CodeDuplicateXRef indicates two or more records share a cross-reference
	// identifier. It carries SeverityError and is reported once per record
	// after the first, with that record's line; the message names the line
	// of the first definition, which Document.XRefMap resolves pointers to.
	CodeDuplicateXRef = "DUPLICATE_XREF"
)

// Error codes for duplicate detection.
const (
	// CodePotentialDuplicate indicates two records may represent the same entity.
	CodePotentialDuplicate = "POTENTIAL_DUPLICATE"

	// CodeDuplicateDetectionLimited indicates duplicate detection did not
	// compare the whole document in full: it skipped one or more surname
	// groups because DuplicateConfig.MaxGroupSize was exceeded, or compared
	// one or more given names on a prefix because they exceeded
	// DuplicateConfig.MaxNameRunes, or both, so some duplicates may be
	// unreported or approximate. The "skipped_groups" and "truncated_names"
	// details say which applied; either may be "0".
	//
	// Unlike CodePotentialDuplicate this carries SeverityWarning and no
	// RecordXRef: it describes the completeness of the analysis rather than a
	// finding about any one record. Its "surnames" detail, present only when
	// a group was skipped, samples the group keys, which are normalized unless
	// DuplicateConfig.DisableNameNormalization is set and so need not match
	// the source document verbatim.
	CodeDuplicateDetectionLimited = "DUPLICATE_DETECTION_LIMITED"
)

// Error codes for data quality validation.
const (
	// CodeMissingBirthDate indicates an individual has no birth date recorded.
	CodeMissingBirthDate = "MISSING_BIRTH_DATE"

	// CodeMissingDeathDate indicates a deceased individual has no death date recorded.
	CodeMissingDeathDate = "MISSING_DEATH_DATE"

	// CodeMissingName indicates an individual record has no level-1 NAME tag
	// and no typed name. It is a data-quality heuristic, not a spec rule:
	// INDI.NAME is optional in every GEDCOM version, but a nameless person is
	// rarely intended. It carries SeverityWarning, the line of the record and
	// "NAME" under Details["field"]; [Validator.ValidateAll] reports it and
	// [QualityReport] files it under CompletenessIssues.
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

// Error codes for note pointer validation.
const (
	// CodeOverlappingNotePointers indicates a media object lists the same note
	// pointer in both NoteXRefs and SharedNoteXRefs, which are documented to
	// partition a record's note pointers and never overlap.
	CodeOverlappingNotePointers = "OVERLAPPING_NOTE_POINTERS"
)

// Error codes for document construction.
const (
	// CodeNilElement indicates a nil where the document holds an element: a nil
	// record, a nil element in a slice, or a Record.Entity holding a typed nil.
	// Only a document built in code can hold one, and the encoder silently skips
	// it, so it almost always marks a bug in the code that built the document.
	// Details["field"] is the path to the nil, such as "Events[2].SourceCitations[1]".
	CodeNilElement = "NIL_ELEMENT"
)

// Error codes for the write path.
const (
	// CodeEntityTagsDiverged indicates the header or a record whose typed
	// fields would encode differently from its raw Tags (a Warning).
	// Usually a typed field was edited on a decoded document, and because
	// Tags is authoritative on encode the edit will be absent from the
	// output; encoder.SyncTags carries it into Tags. The check has no
	// direction, though: Tags edited by hand without the Entity are reported
	// too, and SyncTags would revert them (decoder.SyncEntities is the other
	// direction).
	// The header fields compared are those SyncTags carries. RecordXRef is
	// the record, empty for the header; Details["record"] is the record
	// type, or "HEAD". When SyncTags fails on the document, a single Info
	// issue with Details["error"] says the comparison was not made. The
	// check is opt-in: ValidatorConfig.CheckEntityTags, or
	// WithEntityTagsCheck for a QualityAnalyzer.
	CodeEntityTagsDiverged = "ENTITY_TAGS_DIVERGED"
)

// Error codes for the grammar conformance check, which holds every structure
// against the specification's structure tables for the declared version. The
// check is opt-in: ValidatorConfig.CheckGrammar, or WithGrammarCheck for a
// QualityAnalyzer. Every issue carries Details["version"], the version whose
// grammar was applied, and Details["tag"], the tag at fault. A header that
// declares no version (no HEAD.GEDC.VERS) is not held to any grammar: only
// the missing VERS, or GEDC, is reported, with an empty Details["version"].
const (
	// CodeUnexpectedSubstructure indicates a standard tag the version's
	// grammar does not permit where it stands, either under its
	// superstructure (Details["superstructure"]) or at level 0 (an Error).
	CodeUnexpectedSubstructure = "UNEXPECTED_SUBSTRUCTURE"

	// CodeVersionForeignStructure indicates a tag the declared version does
	// not define anywhere but another version does, such as SNOTE or EXID in
	// a 5.5.1 file, or CONC in a 7.0 file (an Error). Details["defined_in"]
	// names the version that defines it.
	CodeVersionForeignStructure = "VERSION_FOREIGN_STRUCTURE"

	// CodeMissingRequiredSubstructure indicates a structure lacking a
	// substructure its grammar requires, such as HEAD.GEDC.VERS or a 7.0
	// SUBM.NAME (an Error). It is reported on the superstructure's line.
	// HEAD.SUBM in 5.5.x is left to CodeMissingSUBM.
	CodeMissingRequiredSubstructure = "MISSING_REQUIRED_SUBSTRUCTURE"

	// CodeCardinalityExceeded indicates a substructure repeated more often
	// than its cardinality allows, such as a second SEX (an Error). Each
	// occurrence past the limit is reported; Details["cardinality"] is the
	// limit, as "{0:1}".
	CodeCardinalityExceeded = "CARDINALITY_EXCEEDED"

	// CodeInvalidEnumValue indicates an enumerated payload that is not one of
	// its values, such as SEX Z or PEDI BOGUS (an Error). 7.0 values are
	// compared case-sensitively and extension values ("_X") are accepted;
	// 5.5.x values are compared without regard to case. Details["value"] is
	// the value and Details["allowed"] the comma-separated set.
	CodeInvalidEnumValue = "INVALID_ENUM_VALUE"

	// CodeInvalidPayload indicates a payload that is not of its structure's
	// datatype (an Error): an exact date (CHAN.DATE, HEAD.DATE), a time, an
	// age, a language tag, a non-negative integer, a record pointer, or Y.
	// Details["datatype"] names the datatype and Details["value"] the payload.
	// Dates other than exact dates are CodeInvalidDate's.
	CodeInvalidPayload = "INVALID_PAYLOAD"

	// CodeInvalidRecordXRef indicates a 5.5.x level-0 record with no
	// cross-reference identifier, or a 7.0 record identifier using
	// characters outside A-Z, 0-9 and _, or the reserved null pointer @VOID@
	// (an Error). 7.0 permits a record with no identifier.
	CodeInvalidRecordXRef = "INVALID_RECORD_XREF"

	// CodeDeprecatedStructure indicates a 7.0 structure or enumeration value
	// the specification deprecates: ADR1, ADR2, ADR3, HEAD.SOUR.DATA, an
	// EXID without a TYPE, and the ordinance statuses COMPLETED, EXCLUDED,
	// INFANT, PRE_1970, SUBMITTED and UNCLEARED (a Warning).
	CodeDeprecatedStructure = "DEPRECATED_STRUCTURE"
)

// grammarCodes lists the grammar conformance check's codes.
var grammarCodes = []string{
	CodeUnexpectedSubstructure, CodeVersionForeignStructure, CodeMissingRequiredSubstructure,
	CodeCardinalityExceeded, CodeInvalidEnumValue, CodeInvalidPayload, CodeInvalidRecordXRef,
	CodeDeprecatedStructure,
}

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

	// LineNumber is the 1-based source line the issue was raised against, or 0
	// when the check has no single line in hand.
	//
	// 0 is common and is not a defect. Checks that compare whole entities
	// (date logic, duplicate detection) or the document as a whole (a missing
	// header SUBM) have no single line to point at, and the typed Header
	// fields are not attributed to one. Codes raised while walking raw tags --
	// the custom-tag and control-character checks, the XRef-length check, and
	// the broken-pointer and record-structure checks -- do carry it.
	//
	// Do not read 0 as "line 1".
	LineNumber int

	// Details carries context specific to one code, as key-value pairs.
	// Common keys include "field", "value", "expected", "actual".
	//
	// It is not the place for anything Issue models as a field: a source line
	// belongs in LineNumber, not under a stringified "line_number" key that
	// only some codes set and every caller has to strconv.Atoi.
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

	// Format: [SEVERITY] CODE: Message (XRef: @I1@) [line N]
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

	// A line number is only rendered when one is known; 0 means the check had
	// none, and printing "line 0" would read as a real location.
	if i.LineNumber > 0 {
		fmt.Fprintf(&sb, " [line %d]", i.LineNumber)
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

// WithLineNumber returns a copy of the Issue with LineNumber set.
//
//nolint:gocritic // Value receiver intentional for immutability
func (i Issue) WithLineNumber(line int) Issue {
	i.LineNumber = line
	return i
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
