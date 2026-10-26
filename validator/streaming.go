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

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// StreamingOptions configures the StreamingValidator behavior.
type StreamingOptions struct {
	// DateLogic configures date logic validation thresholds.
	// If nil, default values are used.
	DateLogic *DateLogicConfig

	// Strictness controls which severity levels are included in results.
	// The zero value is StrictnessNormal (errors and warnings).
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
//
// opts is stored as given; no field is defaulted. The zero value is already
// the intended default: Strictness is StrictnessNormal, and a nil DateLogic
// makes NewDateLogicValidator use its own defaults.
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
//
// Per-record-type coverage:
//
//   - RecordTypeIndividual: full validation (date logic, structure)
//   - RecordTypeFamily: full validation (date logic, structure)
//   - RecordTypeSource: reference collection only (for orphan detection in
//     Finalize); no source-content validation
//   - RecordTypeRepository, RecordTypeNote, RecordTypeSubmitter,
//     RecordTypeObject, others: not validated (the record is still
//     registered for XRef tracking, but no rules are evaluated)
//
// Callers who need full validation coverage for all record types should use
// the batch [Validator.Validate] / [Validator.ValidateAll] instead — they
// trade memory for completeness.
//
// Parent-child date logic checks (child born before parent) require a
// complete document and are not supported in streaming mode.
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
		if ind, ok := record.GetIndividual(); ok {
			issues = append(issues, sv.validateIndividual(ind)...)
		}
	case gedcom.RecordTypeFamily:
		if fam, ok := record.GetFamily(); ok {
			issues = append(issues, sv.validateFamily(fam)...)
		}
	case gedcom.RecordTypeSource:
		if src, ok := record.GetSource(); ok {
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
	for i, link := range ind.SpouseInFamilies {
		if link.FamilyXRef != "" {
			sv.usedXRefs[link.FamilyXRef] = append(sv.usedXRefs[link.FamilyXRef], usageLocation{
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
	sv.collectNoteXRefs(ind.XRef, "", ind.NoteXRefs)

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

	// Collect NOTE references nested in substructures (#535). Event, attribute,
	// name and citation own notes are not yet collected.
	for i := range ind.ChildInFamilies {
		sv.collectNoteXRefs(ind.XRef, fmt.Sprintf("ChildInFamilies[%d].", i), ind.ChildInFamilies[i].NoteXRefs)
	}
	for i := range ind.SpouseInFamilies {
		sv.collectNoteXRefs(ind.XRef, fmt.Sprintf("SpouseInFamilies[%d].", i), ind.SpouseInFamilies[i].NoteXRefs)
	}
	sv.collectAssociationNotes(ind.XRef, "", ind.Associations)
	sv.collectMediaLinkNotes(ind.XRef, "", ind.Media)
	sv.collectEventNotes(ind.XRef, ind.Events)
	sv.collectAttributeNotes(ind.XRef, ind.Attributes)

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
	sv.collectNoteXRefs(fam.XRef, "", fam.NoteXRefs)

	// Collect NOTE references nested in substructures
	sv.collectMediaLinkNotes(fam.XRef, "", fam.Media)
	sv.collectEventNotes(fam.XRef, fam.Events)
	sv.collectAttributeNotes(fam.XRef, fam.Attributes)

	return nil
}

// collectSourceReferences collects XRef references from a Source record.
func (sv *StreamingValidator) collectSourceReferences(src *gedcom.Source) {
	// Collect REPO references, one per repository link
	for i, link := range src.RepositoryLinks {
		if link == nil || link.XRef == "" {
			continue
		}
		sv.usedXRefs[link.XRef] = append(sv.usedXRefs[link.XRef], usageLocation{
			RecordXRef: src.XRef,
			Context:    "REPO",
			Field:      fmt.Sprintf("RepositoryLinks[%d].XRef", i),
			Index:      i,
		})
	}

	// Collect NOTE references
	sv.collectNoteXRefs(src.XRef, "", src.NoteXRefs)

	// Collect NOTE references nested in substructures
	for i, link := range src.RepositoryLinks {
		if link != nil {
			sv.collectNoteXRefs(src.XRef, fmt.Sprintf("RepositoryLinks[%d].", i), link.NoteXRefs)
		}
	}
	sv.collectMediaLinkNotes(src.XRef, "", src.Media)
}

// collectNoteXRefs records each non-empty note pointer as a NOTE usage by
// recordXRef. prefix locates the slice below the record (e.g. "Media[0].")
// and is prepended to the reported field; "" means the record's own notes.
// For a nested slice Index is the position within that slice; Field is the
// full path.
func (sv *StreamingValidator) collectNoteXRefs(recordXRef, prefix string, noteXRefs []string) {
	for i, noteXRef := range noteXRefs {
		if noteXRef != "" {
			sv.usedXRefs[noteXRef] = append(sv.usedXRefs[noteXRef], usageLocation{
				RecordXRef: recordXRef,
				Context:    "NOTE",
				Field:      fmt.Sprintf("%sNoteXRefs[%d]", prefix, i),
				Index:      i,
			})
		}
	}
}

// collectAssociationNotes collects the note pointers on each ASSO structure.
func (sv *StreamingValidator) collectAssociationNotes(recordXRef, prefix string, assocs []*gedcom.Association) {
	for i, assoc := range assocs {
		if assoc != nil {
			sv.collectNoteXRefs(recordXRef, fmt.Sprintf("%sAssociations[%d].", prefix, i), assoc.NoteXRefs)
		}
	}
}

// collectMediaLinkNotes collects the note pointers on each OBJE link.
func (sv *StreamingValidator) collectMediaLinkNotes(recordXRef, prefix string, links []*gedcom.MediaLink) {
	for i, link := range links {
		if link != nil {
			sv.collectNoteXRefs(recordXRef, fmt.Sprintf("%sMedia[%d].", prefix, i), link.NoteXRefs)
		}
	}
}

// collectPlaceDetailNotes collects the note pointers on a PLAC structure.
func (sv *StreamingValidator) collectPlaceDetailNotes(recordXRef, prefix string, place *gedcom.PlaceDetail) {
	if place != nil {
		sv.collectNoteXRefs(recordXRef, prefix+"PlaceDetail.", place.NoteXRefs)
	}
}

// collectEventNotes collects the note pointers on the PLAC, OBJE and ASSO
// substructures of each event.
func (sv *StreamingValidator) collectEventNotes(recordXRef string, events []*gedcom.Event) {
	for i, ev := range events {
		if ev == nil {
			continue
		}
		prefix := fmt.Sprintf("Events[%d].", i)
		sv.collectPlaceDetailNotes(recordXRef, prefix, ev.PlaceDetail)
		sv.collectMediaLinkNotes(recordXRef, prefix, ev.Media)
		sv.collectAssociationNotes(recordXRef, prefix, ev.Associations)
	}
}

// collectAttributeNotes collects the note pointers on the PLAC, OBJE and ASSO
// substructures of each attribute.
func (sv *StreamingValidator) collectAttributeNotes(recordXRef string, attrs []*gedcom.Attribute) {
	for i, attr := range attrs {
		if attr == nil {
			continue
		}
		prefix := fmt.Sprintf("Attributes[%d].", i)
		sv.collectPlaceDetailNotes(recordXRef, prefix, attr.PlaceDetail)
		sv.collectMediaLinkNotes(recordXRef, prefix, attr.Media)
		sv.collectAssociationNotes(recordXRef, prefix, attr.Associations)
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
