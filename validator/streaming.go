// streaming.go provides streaming validation for memory-efficient GEDCOM processing.
//
// The StreamingValidator validates records incrementally as they are parsed,
// without requiring a complete Document in memory. This enables:
//   - Validating files larger than available memory
//   - Detecting errors early (fail-fast)
//   - Tracking cross-references for eventual consistency checking
//
// Memory usage is proportional to the number of declared XRefs plus the
// references to XRefs not yet declared, not to the total file size: a
// reference to a record already seen is checked at once and not kept.
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

	// Version is the GEDCOM version of the records being validated. It
	// decides what Finalize makes of a pointer to "@VOID@" that no record
	// declares: in GEDCOM 7.0 it is the null pointer and is never reported,
	// while in 5.5 and 5.5.1 it is an ordinary identifier and is reported as
	// orphaned like any other. The streaming validator never sees the
	// header, so the version has to be given here; leave it empty when
	// unknown, and @VOID@ is taken as the null pointer, the rule
	// [gedcom.Document.IsNullPointer] applies to a document that declares
	// no version (#703).
	Version gedcom.Version
}

// usageLocation records where an XRef is referenced.
type usageLocation struct {
	// RecordXRef is the XRef of the record containing the reference.
	RecordXRef string

	// Context describes where the reference appears (e.g., "HUSB", "WIFE", "FAMC").
	Context string

	// Field is the path of the reference within the record
	// (e.g., "ChildInFamilies[0]", "Events[1].NoteXRefs[0]").
	Field string
}

// StreamingValidator validates GEDCOM records incrementally without requiring
// a complete Document in memory. Cross-reference validation is deferred until
// Finalize() is called.
type StreamingValidator struct {
	opts StreamingOptions

	// refs tracks declared and referenced XRefs for Finalize.
	refs xrefTracker

	// nullScope decides which pointer is the null pointer under
	// opts.Version; see addRef.
	nullScope *gedcom.Document

	// dateLogic provides date validation for individual records.
	dateLogic *DateLogicValidator
}

// NewStreamingValidator creates a new StreamingValidator with the given options.
//
// opts is stored as given; no field is defaulted. The zero value is already
// the intended default: Strictness is StrictnessNormal, and a nil DateLogic
// makes NewDateLogicValidator use its own defaults.
func NewStreamingValidator(opts StreamingOptions) *StreamingValidator {
	sv := &StreamingValidator{
		opts:      opts,
		dateLogic: NewDateLogicValidator(opts.DateLogic),
	}
	// A null pointer names no record by design. The rule is
	// gedcom.Document.IsNullPointer's, applied to a document that carries
	// only the version opts.Version names, so the batch and streaming
	// validators cannot disagree about which value is null.
	sv.nullScope = &gedcom.Document{}
	if opts.Version != "" {
		sv.nullScope.Header = &gedcom.Header{Version: opts.Version, DeclaredVersion: opts.Version}
	}
	sv.Reset()
	return sv
}

// ValidateRecord validates a single record and returns immediate issues.
// Cross-reference issues are deferred to Finalize().
//
// The record must carry its typed Entity, as records from the decoder do. A
// record holding only raw Tags (one built from a parser.RawRecord by hand)
// is registered as declared but nothing in it is validated or collected, so
// it reports no issues; decode each streamed record with decoder.DecodeRecord
// first, as the ValidateRecord example shows.
//
// This method:
//   - Registers the record's XRef as declared, reporting CodeDuplicateXRef
//     if an earlier record already declared it
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
//     RecordTypeMedia, others: not validated (the record is still
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

	// Register this record's XRef as declared. A redefinition is reported
	// at once, and the first declaration stays registered.
	if record.XRef != "" {
		if firstLine, dup := sv.refs.declare(record.XRef, record.LineNumber); dup {
			issues = append(issues, duplicateXRefIssue(record, firstLine))
		}
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
			sv.addRef(link.FamilyXRef, ind.XRef, "FAMC", fmt.Sprintf("ChildInFamilies[%d]", i))
		}
	}

	// Collect FAMS references
	for i, link := range ind.SpouseInFamilies {
		if link.FamilyXRef != "" {
			sv.addRef(link.FamilyXRef, ind.XRef, "FAMS", fmt.Sprintf("SpouseInFamilies[%d]", i))
		}
	}

	// Collect SOUR references from individual
	for i, citation := range ind.SourceCitations {
		if citation != nil && citation.SourceXRef != "" {
			sv.addRef(citation.SourceXRef, ind.XRef, "SOUR", fmt.Sprintf("SourceCitations[%d]", i))
		}
	}

	// Collect NOTE references
	sv.collectNoteXRefs(ind.XRef, "", ind.NoteXRefs)

	// Collect ASSO references
	for i, assoc := range ind.Associations {
		if assoc != nil && assoc.IndividualXRef != "" {
			sv.addRef(assoc.IndividualXRef, ind.XRef, "ASSO", fmt.Sprintf("Associations[%d]", i))
		}
	}

	// Collect NOTE references nested in substructures (#535, #673)
	sv.collectIndividualSubstructureNotes(ind)

	return issues
}

// collectIndividualSubstructureNotes collects the note pointers on every typed
// structure below an individual that carries notes.
func (sv *StreamingValidator) collectIndividualSubstructureNotes(ind *gedcom.Individual) {
	for i := range ind.ChildInFamilies {
		sv.collectNoteXRefs(ind.XRef, fmt.Sprintf("ChildInFamilies[%d].", i), ind.ChildInFamilies[i].NoteXRefs)
	}
	for i := range ind.SpouseInFamilies {
		sv.collectNoteXRefs(ind.XRef, fmt.Sprintf("SpouseInFamilies[%d].", i), ind.SpouseInFamilies[i].NoteXRefs)
	}
	for i, name := range ind.Names {
		if name != nil {
			prefix := fmt.Sprintf("Names[%d].", i)
			sv.collectNoteXRefs(ind.XRef, prefix, name.NoteXRefs)
			sv.collectCitationNotes(ind.XRef, prefix, name.SourceCitations)
		}
	}
	sv.collectCitationNotes(ind.XRef, "", ind.SourceCitations)
	sv.collectAssociationNotes(ind.XRef, "", ind.Associations)
	sv.collectMediaLinkNotes(ind.XRef, "", ind.Media)
	sv.collectEventNotes(ind.XRef, ind.Events)
	sv.collectAttributeNotes(ind.XRef, ind.Attributes)
	sv.collectLDSOrdinanceNotes(ind.XRef, ind.LDSOrdinances)
	sv.collectChangeDateNotes(ind.XRef, "ChangeDate.", ind.ChangeDate)
	sv.collectChangeDateNotes(ind.XRef, "CreationDate.", ind.CreationDate)
}

// validateFamily validates a Family record and collects XRef references.
// Currently only collects references; validation rules may be added later.
//
//nolint:unparam // Returns nil now but signature kept for future validation rules
func (sv *StreamingValidator) validateFamily(fam *gedcom.Family) []Issue {
	// Collect HUSB reference
	if fam.Husband != "" {
		sv.addRef(fam.Husband, fam.XRef, "HUSB", "Husband")
	}

	// Collect WIFE reference
	if fam.Wife != "" {
		sv.addRef(fam.Wife, fam.XRef, "WIFE", "Wife")
	}

	// Collect CHIL references
	for i, childXRef := range fam.Children {
		if childXRef != "" {
			sv.addRef(childXRef, fam.XRef, "CHIL", fmt.Sprintf("Children[%d]", i))
		}
	}

	// Collect SOUR references from family
	for i, citation := range fam.SourceCitations {
		if citation != nil && citation.SourceXRef != "" {
			sv.addRef(citation.SourceXRef, fam.XRef, "SOUR", fmt.Sprintf("SourceCitations[%d]", i))
		}
	}

	// Collect NOTE references
	sv.collectNoteXRefs(fam.XRef, "", fam.NoteXRefs)

	// Collect NOTE references nested in substructures
	sv.collectCitationNotes(fam.XRef, "", fam.SourceCitations)
	sv.collectMediaLinkNotes(fam.XRef, "", fam.Media)
	sv.collectEventNotes(fam.XRef, fam.Events)
	sv.collectAttributeNotes(fam.XRef, fam.Attributes)
	sv.collectLDSOrdinanceNotes(fam.XRef, fam.LDSOrdinances)
	sv.collectChangeDateNotes(fam.XRef, "ChangeDate.", fam.ChangeDate)
	sv.collectChangeDateNotes(fam.XRef, "CreationDate.", fam.CreationDate)

	return nil
}

// collectSourceReferences collects XRef references from a Source record.
func (sv *StreamingValidator) collectSourceReferences(src *gedcom.Source) {
	// Collect REPO references, one per repository link
	for i, link := range src.RepositoryLinks {
		if link == nil || link.XRef == "" {
			continue
		}
		sv.addRef(link.XRef, src.XRef, "REPO", fmt.Sprintf("RepositoryLinks[%d].XRef", i))
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
	if src.Data != nil {
		sv.collectNoteXRefs(src.XRef, "Data.", src.Data.NoteXRefs)
		for i, ev := range src.Data.Events {
			if ev != nil {
				sv.collectPlaceDetailNotes(src.XRef, fmt.Sprintf("Data.Events[%d].", i), ev.PlaceDetail)
			}
		}
	}
	sv.collectChangeDateNotes(src.XRef, "ChangeDate.", src.ChangeDate)
	sv.collectChangeDateNotes(src.XRef, "CreationDate.", src.CreationDate)
}

// addRef records that recordXRef references target at field, in the
// structure context names ("FAMC", "NOTE", ...). A null pointer is counted
// but never reported as orphaned.
func (sv *StreamingValidator) addRef(target, recordXRef, context, field string) {
	sv.refs.reference(target, recordXRef, context, field, !sv.nullScope.IsNullPointer(target))
}

// collectNoteXRefs records each non-empty note pointer as a NOTE usage by
// recordXRef. prefix locates the slice below the record (e.g. "Media[0].")
// and is prepended to the reported field; "" means the record's own notes.
// For a nested slice Index is the position within that slice; Field is the
// full path.
func (sv *StreamingValidator) collectNoteXRefs(recordXRef, prefix string, noteXRefs []string) {
	for i, noteXRef := range noteXRefs {
		if noteXRef != "" {
			sv.addRef(noteXRef, recordXRef, "NOTE", fmt.Sprintf("%sNoteXRefs[%d]", prefix, i))
		}
	}
}

// collectAssociationNotes collects the note pointers on each ASSO structure
// and on its source citations.
func (sv *StreamingValidator) collectAssociationNotes(recordXRef, prefix string, assocs []*gedcom.Association) {
	for i, assoc := range assocs {
		if assoc != nil {
			assocPrefix := fmt.Sprintf("%sAssociations[%d].", prefix, i)
			sv.collectNoteXRefs(recordXRef, assocPrefix, assoc.NoteXRefs)
			sv.collectCitationNotes(recordXRef, assocPrefix, assoc.SourceCitations)
		}
	}
}

// collectCitationNotes collects the note pointers on each SOUR citation.
func (sv *StreamingValidator) collectCitationNotes(recordXRef, prefix string, citations []*gedcom.SourceCitation) {
	for i, citation := range citations {
		if citation != nil {
			citePrefix := fmt.Sprintf("%sSourceCitations[%d].", prefix, i)
			sv.collectNoteXRefs(recordXRef, citePrefix, citation.NoteXRefs)
			sv.collectMediaLinkNotes(recordXRef, citePrefix, citation.Media)
		}
	}
}

// collectLDSOrdinanceNotes collects the note pointers on each LDS ordinance.
func (sv *StreamingValidator) collectLDSOrdinanceNotes(recordXRef string, ords []*gedcom.LDSOrdinance) {
	for i, ord := range ords {
		if ord != nil {
			sv.collectNoteXRefs(recordXRef, fmt.Sprintf("LDSOrdinances[%d].", i), ord.NoteXRefs)
		}
	}
}

// collectChangeDateNotes collects the note pointers on a CHAN or CREA
// structure; prefix names the field holding it (e.g. "ChangeDate.").
func (sv *StreamingValidator) collectChangeDateNotes(recordXRef, prefix string, cd *gedcom.ChangeDate) {
	if cd != nil {
		sv.collectNoteXRefs(recordXRef, prefix, cd.NoteXRefs)
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

// collectEventNotes collects the note pointers on each event and on its PLAC,
// SOUR, OBJE and ASSO substructures.
func (sv *StreamingValidator) collectEventNotes(recordXRef string, events []*gedcom.Event) {
	for i, ev := range events {
		if ev == nil {
			continue
		}
		prefix := fmt.Sprintf("Events[%d].", i)
		sv.collectNoteXRefs(recordXRef, prefix, ev.NoteXRefs)
		sv.collectPlaceDetailNotes(recordXRef, prefix, ev.PlaceDetail)
		sv.collectCitationNotes(recordXRef, prefix, ev.SourceCitations)
		sv.collectMediaLinkNotes(recordXRef, prefix, ev.Media)
		sv.collectAssociationNotes(recordXRef, prefix, ev.Associations)
	}
}

// collectAttributeNotes collects the note pointers on each attribute and on its
// PLAC, SOUR, OBJE and ASSO substructures.
func (sv *StreamingValidator) collectAttributeNotes(recordXRef string, attrs []*gedcom.Attribute) {
	for i, attr := range attrs {
		if attr == nil {
			continue
		}
		prefix := fmt.Sprintf("Attributes[%d].", i)
		sv.collectNoteXRefs(recordXRef, prefix, attr.NoteXRefs)
		sv.collectPlaceDetailNotes(recordXRef, prefix, attr.PlaceDetail)
		sv.collectCitationNotes(recordXRef, prefix, attr.SourceCitations)
		sv.collectMediaLinkNotes(recordXRef, prefix, attr.Media)
		sv.collectAssociationNotes(recordXRef, prefix, attr.Associations)
	}
}

// Finalize completes validation and returns cross-reference issues.
// This method should be called after all records have been validated with ValidateRecord.
//
// Returns issues for:
//   - Orphaned references (XRefs used but never declared), except the GEDCOM
//     7.0 null pointer "@VOID@" under [StreamingOptions.Version]
func (sv *StreamingValidator) Finalize() []Issue {
	var issues []Issue

	sv.refs.orphans(func(xref string, usage usageLocation) {
		issues = append(issues, sv.createOrphanedReferenceIssue(xref, usage))
	})

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
	sv.refs = newXRefTracker()
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
	return sv.refs.seenCount
}

// UsedXRefCount returns the number of unique XRefs referenced by records.
// This is useful for memory usage monitoring.
func (sv *StreamingValidator) UsedXRefCount() int {
	return sv.refs.usedCount
}
