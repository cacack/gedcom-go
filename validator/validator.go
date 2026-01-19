package validator

import (
	"fmt"

	"github.com/cacack/gedcom-go/gedcom"
)

// ValidationError represents a validation error with error code, message, line number, and optional cross-reference.
type ValidationError struct {
	Code    string
	Message string
	Line    int
	XRef    string
}

func (e *ValidationError) Error() string {
	if e.XRef != "" {
		return fmt.Sprintf("[%s] %s (XRef: %s)", e.Code, e.Message, e.XRef)
	}
	if e.Line > 0 {
		return fmt.Sprintf("[%s] line %d: %s", e.Code, e.Line, e.Message)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Strictness defines the level of validation strictness.
type Strictness int

const (
	// StrictnessRelaxed reports only errors.
	StrictnessRelaxed Strictness = iota
	// StrictnessNormal reports errors and warnings (default).
	StrictnessNormal
	// StrictnessStrict reports all issues including info.
	StrictnessStrict
)

// ValidatorConfig contains configuration options for the Validator.
//
//nolint:revive // Name kept for API clarity despite repetition
type ValidatorConfig struct {
	// DateLogic configures date logic validation thresholds.
	// If nil, default values are used.
	DateLogic *DateLogicConfig

	// Duplicates configures duplicate detection thresholds.
	// If nil, default values are used.
	Duplicates *DuplicateConfig

	// Strictness controls which severity levels are included in results.
	// Default: StrictnessNormal (errors and warnings).
	Strictness Strictness

	// TagRegistry holds definitions for custom (underscore-prefixed) tags.
	// If nil, custom tag validation is disabled.
	TagRegistry *TagRegistry

	// ValidateCustomTags enables validation of custom (underscore-prefixed) tags.
	// When true and TagRegistry is set, custom tags are validated against the registry.
	// Default: false (backward compatible).
	ValidateCustomTags bool
}

// Validator validates GEDCOM documents against specification rules.
type Validator struct {
	errors       []error
	config       *ValidatorConfig
	dateLogic    *DateLogicValidator
	references   *ReferenceValidator
	duplicates   *DuplicateDetector
	quality      *QualityAnalyzer
	tagValidator *TagValidator
}

// New creates a new Validator with default configuration.
func New() *Validator {
	return NewWithConfig(nil)
}

// NewWithConfig creates a new Validator with the given configuration.
// If config is nil, default configuration is used.
func NewWithConfig(config *ValidatorConfig) *Validator {
	if config == nil {
		config = &ValidatorConfig{
			Strictness: StrictnessNormal,
		}
	}
	return &Validator{
		errors: make([]error, 0),
		config: config,
	}
}

// getDateLogicValidator returns the date logic validator, creating it lazily if needed.
func (v *Validator) getDateLogicValidator() *DateLogicValidator {
	if v.dateLogic == nil {
		var config *DateLogicConfig
		if v.config != nil {
			config = v.config.DateLogic
		}
		v.dateLogic = NewDateLogicValidator(config)
	}
	return v.dateLogic
}

// getReferenceValidator returns the reference validator, creating it lazily if needed.
func (v *Validator) getReferenceValidator() *ReferenceValidator {
	if v.references == nil {
		v.references = NewReferenceValidator()
	}
	return v.references
}

// getDuplicateDetector returns the duplicate detector, creating it lazily if needed.
func (v *Validator) getDuplicateDetector() *DuplicateDetector {
	if v.duplicates == nil {
		var config *DuplicateConfig
		if v.config != nil {
			config = v.config.Duplicates
		}
		v.duplicates = NewDuplicateDetector(config)
	}
	return v.duplicates
}

// getQualityAnalyzer returns the quality analyzer, creating it lazily if needed.
func (v *Validator) getQualityAnalyzer() *QualityAnalyzer {
	if v.quality == nil {
		var opts []QualityOption
		if v.config != nil {
			if v.config.DateLogic != nil {
				opts = append(opts, WithDateLogicConfig(v.config.DateLogic))
			}
			if v.config.Duplicates != nil {
				opts = append(opts, WithDuplicateConfig(v.config.Duplicates))
			}
			if v.config.TagRegistry != nil {
				opts = append(opts, WithTagRegistry(v.config.TagRegistry))
			}
		}
		v.quality = NewQualityAnalyzer(opts...)
	}
	return v.quality
}

// getTagValidator returns the tag validator, creating it lazily if needed.
func (v *Validator) getTagValidator() *TagValidator {
	if v.tagValidator == nil {
		var registry *TagRegistry
		validateUnknown := false
		if v.config != nil {
			registry = v.config.TagRegistry
			validateUnknown = v.config.ValidateCustomTags
		}
		v.tagValidator = NewTagValidator(registry, validateUnknown)
	}
	return v.tagValidator
}

// Validate validates a GEDCOM document and returns any validation errors.
func (v *Validator) Validate(doc *gedcom.Document) []error {
	v.errors = make([]error, 0)

	// Validate cross-references
	v.validateXRefs(doc)

	// Validate records
	v.validateRecords(doc)

	return v.errors
}

// validateXRefs checks that all cross-references are valid.
func (v *Validator) validateXRefs(doc *gedcom.Document) {
	// Track all XRef usages
	usedXRefs := make(map[string]bool)

	// Scan all records for XRef usage
	for _, record := range doc.Records {
		for _, tag := range record.Tags {
			// Check if value looks like an XRef
			if len(tag.Value) > 2 && tag.Value[0] == '@' && tag.Value[len(tag.Value)-1] == '@' {
				xref := tag.Value
				usedXRefs[xref] = true

				// Verify the XRef exists
				if doc.XRefMap[xref] == nil {
					v.errors = append(v.errors, &ValidationError{
						Code:    "BROKEN_XREF",
						Message: fmt.Sprintf("Reference to non-existent record %s", xref),
						Line:    tag.LineNumber,
					})
				}
			}
		}
	}
}

// validateRecords validates individual records.
func (v *Validator) validateRecords(doc *gedcom.Document) {
	for _, record := range doc.Records {
		switch record.Type {
		case gedcom.RecordTypeIndividual:
			v.validateIndividual(record)
		case gedcom.RecordTypeFamily:
			v.validateFamily(record)
		}
	}
}

// validateIndividual validates an individual record.
func (v *Validator) validateIndividual(record *gedcom.Record) {
	// Check for required NAME tag
	hasName := false
	for _, tag := range record.Tags {
		if tag.Tag == "NAME" {
			hasName = true
			break
		}
	}

	if !hasName {
		v.errors = append(v.errors, &ValidationError{
			Code:    "MISSING_REQUIRED_FIELD",
			Message: "Individual record missing required NAME tag",
			XRef:    record.XRef,
		})
	}
}

// validateFamily validates a family record.
func (v *Validator) validateFamily(record *gedcom.Record) {
	// Family records should have at least one spouse or child
	hasMembers := false
	for _, tag := range record.Tags {
		if tag.Tag == "HUSB" || tag.Tag == "WIFE" || tag.Tag == "CHIL" {
			hasMembers = true
			break
		}
	}

	if !hasMembers {
		v.errors = append(v.errors, &ValidationError{
			Code:    "EMPTY_FAMILY",
			Message: "Family record has no members (no HUSB, WIFE, or CHIL tags)",
			XRef:    record.XRef,
		})
	}
}

// ValidateAll returns comprehensive validation as Issues with severity levels.
// This is the enhanced API that provides more detail than Validate().
// Issues are filtered based on the configured Strictness level.
func (v *Validator) ValidateAll(doc *gedcom.Document) []Issue {
	if doc == nil {
		return nil
	}

	var allIssues []Issue

	// Run date logic validation
	allIssues = append(allIssues, v.getDateLogicValidator().Validate(doc)...)

	// Run reference validation
	allIssues = append(allIssues, v.getReferenceValidator().Validate(doc)...)

	// Run duplicate detection and convert to issues
	for _, pair := range v.getDuplicateDetector().FindDuplicates(doc) {
		allIssues = append(allIssues, pair.ToIssue())
	}

	// Run custom tag validation if a registry is configured
	if v.config != nil && v.config.TagRegistry != nil {
		allIssues = append(allIssues, v.getTagValidator().Validate(doc)...)
	}

	// Filter by strictness
	return v.filterByStrictness(allIssues)
}

// ValidateDateLogic runs date logic validation and returns any issues found.
// This checks for chronological impossibilities like death before birth,
// children born before parents, and unreasonable ages.
func (v *Validator) ValidateDateLogic(doc *gedcom.Document) []Issue {
	if doc == nil {
		return nil
	}
	issues := v.getDateLogicValidator().Validate(doc)
	return v.filterByStrictness(issues)
}

// FindOrphanedReferences checks for cross-references that point to non-existent records.
// This includes FAMC, FAMS, HUSB, WIFE, CHIL, and SOUR references.
func (v *Validator) FindOrphanedReferences(doc *gedcom.Document) []Issue {
	if doc == nil {
		return nil
	}
	issues := v.getReferenceValidator().Validate(doc)
	return v.filterByStrictness(issues)
}

// ValidateCustomTags validates custom (underscore-prefixed) tags against the configured registry.
// Returns issues for tags that violate parent or value constraints, and optionally for unknown tags.
// This method requires a TagRegistry to be configured; if none is set, it returns nil.
func (v *Validator) ValidateCustomTags(doc *gedcom.Document) []Issue {
	if doc == nil {
		return nil
	}
	// Skip if no registry configured
	if v.config == nil || v.config.TagRegistry == nil {
		return nil
	}
	issues := v.getTagValidator().Validate(doc)
	return v.filterByStrictness(issues)
}

// FindPotentialDuplicates detects potential duplicate individuals based on
// name similarity and birth date proximity.
func (v *Validator) FindPotentialDuplicates(doc *gedcom.Document) []DuplicatePair {
	if doc == nil {
		return nil
	}
	return v.getDuplicateDetector().FindDuplicates(doc)
}

// QualityReport generates a comprehensive data quality report for the document.
// The report includes all validation results and data completeness statistics.
func (v *Validator) QualityReport(doc *gedcom.Document) *QualityReport {
	if doc == nil {
		return &QualityReport{
			Errors:             []Issue{},
			Warnings:           []Issue{},
			Info:               []Issue{},
			DateLogicIssues:    []Issue{},
			ReferenceIssues:    []Issue{},
			DuplicateIssues:    []Issue{},
			CompletenessIssues: []Issue{},
		}
	}
	return v.getQualityAnalyzer().Analyze(doc)
}

// filterByStrictness filters issues based on the configured strictness level.
func (v *Validator) filterByStrictness(issues []Issue) []Issue {
	if len(issues) == 0 {
		return issues
	}

	strictness := StrictnessNormal
	if v.config != nil {
		strictness = v.config.Strictness
	}

	switch strictness {
	case StrictnessRelaxed:
		// Only errors
		var result []Issue
		for _, issue := range issues {
			if issue.Severity == SeverityError {
				result = append(result, issue)
			}
		}
		return result
	case StrictnessNormal:
		// Errors and warnings
		var result []Issue
		for _, issue := range issues {
			if issue.Severity == SeverityError || issue.Severity == SeverityWarning {
				result = append(result, issue)
			}
		}
		return result
	case StrictnessStrict:
		// All issues
		return issues
	default:
		return issues
	}
}
