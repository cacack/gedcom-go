// streaming.go provides streaming validation for memory-efficient GEDCOM processing.
//
// The StreamingValidator validates records incrementally as they are parsed,
// without requiring a complete Document in memory. This enables:
//   - Validating files larger than available memory
//   - Detecting errors early (fail-fast)
//   - Tracking cross-references for eventual consistency checking
//
// Memory usage is proportional to the number of unique XRefs, not the total file size.
//
// # Basic Usage
//
//	sv := validator.NewStreamingValidator(validator.StreamingOptions{})
//	var issues []validator.Issue
//
//	// Process records as they are parsed
//	for _, record := range doc.Records {
//	    issues = append(issues, sv.ValidateRecord(record)...)
//	}
//
//	// Check cross-reference consistency
//	issues = append(issues, sv.Finalize()...)

package validator

import (
	"fmt"

	"github.com/cacack/gedcom-go/gedcom"
)

// StreamingOptions configures the StreamingValidator behavior.
type StreamingOptions struct {
	// DateLogic configures date logic validation thresholds.
	// If nil, default values are used.
	DateLogic *DateLogicConfig

	// Strictness controls which severity levels are included in results.
	// Default: StrictnessNormal (errors and warnings).
	Strictness Strictness
}

// usageLocation records where an XRef is referenced.
type usageLocation struct {
	// RecordXRef is the XRef of the record containing the reference.
	RecordXRef string

	// Context describes where the reference appears (e.g., "HUSB", "WIFE", "FAMC").
	Context string

	// Field provides additional detail about the reference location.
	Field string

	// Index is the position in a list (for CHIL, ChildInFamilies, etc.).
	Index int
}

// StreamingValidator validates GEDCOM records incrementally without requiring
// a complete Document in memory. Cross-reference validation is deferred until
// Finalize() is called.
type StreamingValidator struct {
	opts StreamingOptions

	// seenXRefs tracks all declared XRefs (record identifiers).
	seenXRefs map[string]struct{}

	// usedXRefs maps each referenced XRef to where it's used.
	usedXRefs map[string][]usageLocation

	// xrefTypes maps XRefs to their record types for context-aware validation.
	xrefTypes map[string]gedcom.RecordType

	// dateLogic provides date validation for individual records.
	dateLogic *DateLogicValidator
}

// NewStreamingValidator creates a new StreamingValidator with the given options.
// If opts is the zero value, default options are used.
func NewStreamingValidator(opts StreamingOptions) *StreamingValidator {
	return &StreamingValidator{
		opts:      opts,
		seenXRefs: make(map[string]struct{}),
		usedXRefs: make(map[string][]usageLocation),
		xrefTypes: make(map[string]gedcom.RecordType),
		dateLogic: NewDateLogicValidator(opts.DateLogic),
	}
}

// ValidateRecord validates a single record and returns immediate issues.
// Cross-reference issues are deferred to Finalize().
//
// This method:
//   - Registers the record's XRef as declared
//   - Validates record structure and entity-specific rules
//   - Collects XRef references for deferred validation
//   - Returns immediate issues (malformed dates, invalid structure, etc.)
func (sv *StreamingValidator) ValidateRecord(record *gedcom.Record) []Issue {
	if record == nil {
		return nil
	}

	var issues []Issue

	// Register this record's XRef as declared
	if record.XRef != "" {
		sv.seenXRefs[record.XRef] = struct{}{}
		sv.xrefTypes[record.XRef] = record.Type
	}

	// Validate based on record type
	switch record.Type {
	case gedcom.RecordTypeIndividual:
		if ind, ok := record.GetIndividual(); ok && ind != nil {
			issues = append(issues, sv.validateIndividual(ind)...)
		}
	case gedcom.RecordTypeFamily:
		if fam, ok := record.GetFamily(); ok && fam != nil {
			issues = append(issues, sv.validateFamily(fam)...)
		}
	case gedcom.RecordTypeSource:
		if src, ok := record.GetSource(); ok && src != nil {
			sv.collectSourceReferences(src)
		}
	}

	return sv.filterByStrictness(issues)
}

// validateIndividual validates an Individual record and collects XRef references.
func (sv *StreamingValidator) validateIndividual(ind *gedcom.Individual) []Issue {
	var issues []Issue

	// Validate date logic (death before birth, etc.) - these are immediate issues
	// Note: We can only do individual-level checks without a Document.
	// Parent-child checks require a full document and are not supported in streaming mode.
	if issue := sv.dateLogic.checkDeathBeforeBirth(ind); issue != nil {
		issues = append(issues, *issue)
	}

	// Collect FAMC references
	for i, link := range ind.ChildInFamilies {
		if link.FamilyXRef != "" {
			sv.usedXRefs[link.FamilyXRef] = append(sv.usedXRefs[link.FamilyXRef], usageLocation{
				RecordXRef: ind.XRef,
				Context:    "FAMC",
				Field:      fmt.Sprintf("ChildInFamilies[%d]", i),
				Index:      i,
			})
		}
	}

	// Collect FAMS references
	for i, famXRef := range ind.SpouseInFamilies {
		if famXRef != "" {
			sv.usedXRefs[famXRef] = append(sv.usedXRefs[famXRef], usageLocation{
				RecordXRef: ind.XRef,
				Context:    "FAMS",
				Field:      fmt.Sprintf("SpouseInFamilies[%d]", i),
				Index:      i,
			})
		}
	}

	// Collect SOUR references from individual
	for i, citation := range ind.SourceCitations {
		if citation != nil && citation.SourceXRef != "" {
			sv.usedXRefs[citation.SourceXRef] = append(sv.usedXRefs[citation.SourceXRef], usageLocation{
				RecordXRef: ind.XRef,
				Context:    "SOUR",
				Field:      fmt.Sprintf("SourceCitations[%d]", i),
				Index:      i,
			})
		}
	}

	// Collect NOTE references
	for i, noteXRef := range ind.Notes {
		if noteXRef != "" {
			sv.usedXRefs[noteXRef] = append(sv.usedXRefs[noteXRef], usageLocation{
				RecordXRef: ind.XRef,
				Context:    "NOTE",
				Field:      fmt.Sprintf("Notes[%d]", i),
				Index:      i,
			})
		}
	}

	// Collect ASSO references
	for i, assoc := range ind.Associations {
		if assoc != nil && assoc.IndividualXRef != "" {
			sv.usedXRefs[assoc.IndividualXRef] = append(sv.usedXRefs[assoc.IndividualXRef], usageLocation{
				RecordXRef: ind.XRef,
				Context:    "ASSO",
				Field:      fmt.Sprintf("Associations[%d]", i),
				Index:      i,
			})
		}
	}

	return issues
}

// validateFamily validates a Family record and collects XRef references.
// Currently only collects references; validation rules may be added later.
//
//nolint:unparam // Returns nil now but signature kept for future validation rules
func (sv *StreamingValidator) validateFamily(fam *gedcom.Family) []Issue {
	// Collect HUSB reference
	if fam.Husband != "" {
		sv.usedXRefs[fam.Husband] = append(sv.usedXRefs[fam.Husband], usageLocation{
			RecordXRef: fam.XRef,
			Context:    "HUSB",
			Field:      "Husband",
			Index:      0,
		})
	}

	// Collect WIFE reference
	if fam.Wife != "" {
		sv.usedXRefs[fam.Wife] = append(sv.usedXRefs[fam.Wife], usageLocation{
			RecordXRef: fam.XRef,
			Context:    "WIFE",
			Field:      "Wife",
			Index:      0,
		})
	}

	// Collect CHIL references
	for i, childXRef := range fam.Children {
		if childXRef != "" {
			sv.usedXRefs[childXRef] = append(sv.usedXRefs[childXRef], usageLocation{
				RecordXRef: fam.XRef,
				Context:    "CHIL",
				Field:      fmt.Sprintf("Children[%d]", i),
				Index:      i,
			})
		}
	}

	// Collect SOUR references from family
	for i, citation := range fam.SourceCitations {
		if citation != nil && citation.SourceXRef != "" {
			sv.usedXRefs[citation.SourceXRef] = append(sv.usedXRefs[citation.SourceXRef], usageLocation{
				RecordXRef: fam.XRef,
				Context:    "SOUR",
				Field:      fmt.Sprintf("SourceCitations[%d]", i),
				Index:      i,
			})
		}
	}

	// Collect NOTE references
	for i, noteXRef := range fam.Notes {
		if noteXRef != "" {
			sv.usedXRefs[noteXRef] = append(sv.usedXRefs[noteXRef], usageLocation{
				RecordXRef: fam.XRef,
				Context:    "NOTE",
				Field:      fmt.Sprintf("Notes[%d]", i),
				Index:      i,
			})
		}
	}

	return nil
}

// collectSourceReferences collects XRef references from a Source record.
func (sv *StreamingValidator) collectSourceReferences(src *gedcom.Source) {
	// Collect REPO reference
	if src.RepositoryRef != "" {
		sv.usedXRefs[src.RepositoryRef] = append(sv.usedXRefs[src.RepositoryRef], usageLocation{
			RecordXRef: src.XRef,
			Context:    "REPO",
			Field:      "RepositoryRef",
			Index:      0,
		})
	}

	// Collect NOTE references
	for i, noteXRef := range src.Notes {
		if noteXRef != "" {
			sv.usedXRefs[noteXRef] = append(sv.usedXRefs[noteXRef], usageLocation{
				RecordXRef: src.XRef,
				Context:    "NOTE",
				Field:      fmt.Sprintf("Notes[%d]", i),
				Index:      i,
			})
		}
	}
}

// Finalize completes validation and returns cross-reference issues.
// This method should be called after all records have been validated with ValidateRecord.
//
// Returns issues for:
//   - Orphaned references (XRefs used but never declared)
func (sv *StreamingValidator) Finalize() []Issue {
	var issues []Issue

	// Check for orphaned references
	for xref, usages := range sv.usedXRefs {
		if _, exists := sv.seenXRefs[xref]; !exists {
			// XRef is used but was never declared
			for _, usage := range usages {
				issues = append(issues, sv.createOrphanedReferenceIssue(xref, usage))
			}
		}
	}

	return sv.filterByStrictness(issues)
}

// createOrphanedReferenceIssue creates an issue for an orphaned reference.
func (sv *StreamingValidator) createOrphanedReferenceIssue(xref string, usage usageLocation) Issue {
	var code string
	var message string

	switch usage.Context {
	case "FAMC":
		code = CodeOrphanedFAMC
		message = fmt.Sprintf("FAMC reference to non-existent family %s", xref)
	case "FAMS":
		code = CodeOrphanedFAMS
		message = fmt.Sprintf("FAMS reference to non-existent family %s", xref)
	case "HUSB":
		code = CodeOrphanedHUSB
		message = fmt.Sprintf("HUSB reference to non-existent individual %s", xref)
	case "WIFE":
		code = CodeOrphanedWIFE
		message = fmt.Sprintf("WIFE reference to non-existent individual %s", xref)
	case "CHIL":
		code = CodeOrphanedCHIL
		message = fmt.Sprintf("CHIL reference to non-existent individual %s", xref)
	case "SOUR":
		code = CodeOrphanedSOUR
		message = fmt.Sprintf("SOUR reference to non-existent source %s", xref)
	default:
		// Generic orphaned reference for NOTE, ASSO, REPO, etc.
		code = "ORPHANED_" + usage.Context
		message = fmt.Sprintf("%s reference to non-existent record %s", usage.Context, xref)
	}

	return NewIssue(SeverityError, code, message, usage.RecordXRef).
		WithRelatedXRef(xref).
		WithDetail("reference_type", usage.Context).
		WithDetail("field", usage.Field)
}

// Reset clears all internal state, allowing the validator to be reused.
func (sv *StreamingValidator) Reset() {
	sv.seenXRefs = make(map[string]struct{})
	sv.usedXRefs = make(map[string][]usageLocation)
	sv.xrefTypes = make(map[string]gedcom.RecordType)
}

// filterByStrictness filters issues based on the configured strictness level.
func (sv *StreamingValidator) filterByStrictness(issues []Issue) []Issue {
	if len(issues) == 0 {
		return issues
	}

	switch sv.opts.Strictness {
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
		// Default to normal strictness
		var result []Issue
		for _, issue := range issues {
			if issue.Severity == SeverityError || issue.Severity == SeverityWarning {
				result = append(result, issue)
			}
		}
		return result
	}
}

// SeenXRefCount returns the number of declared XRefs tracked by the validator.
// This is useful for memory usage monitoring.
func (sv *StreamingValidator) SeenXRefCount() int {
	return len(sv.seenXRefs)
}

// UsedXRefCount returns the number of unique XRefs referenced by records.
// This is useful for memory usage monitoring.
func (sv *StreamingValidator) UsedXRefCount() int {
	return len(sv.usedXRefs)
}
