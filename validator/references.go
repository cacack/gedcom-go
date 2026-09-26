// references.go provides enhanced orphaned reference detection with typed validation
// for all GEDCOM cross-reference types.
//
// This module validates that all cross-references in a GEDCOM document point to
// existing records. It provides granular detection for different reference types:
// FAMC (child-in-family), FAMS (spouse-in-family), HUSB, WIFE, CHIL, and SOUR.

package validator

import (
	"fmt"
	"strings"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// ReferenceType represents the type of cross-reference being validated.
type ReferenceType string

const (
	// RefTypeFAMC is a child-in-family reference (Individual.ChildInFamilies).
	RefTypeFAMC ReferenceType = "FAMC"

	// RefTypeFAMS is a spouse-in-family reference (Individual.SpouseInFamilies).
	RefTypeFAMS ReferenceType = "FAMS"

	// RefTypeHUSB is a husband reference (Family.Husband).
	RefTypeHUSB ReferenceType = "HUSB"

	// RefTypeWIFE is a wife reference (Family.Wife).
	RefTypeWIFE ReferenceType = "WIFE"

	// RefTypeCHIL is a child reference (Family.Children).
	RefTypeCHIL ReferenceType = "CHIL"

	// RefTypeSOUR is a source reference (SourceCitation.SourceXRef).
	RefTypeSOUR ReferenceType = "SOUR"
)

// ReferenceValidator provides typed validation of cross-references in GEDCOM documents.
// It detects orphaned references (references to non-existent records) and provides
// detailed diagnostics including the reference type and field location.
type ReferenceValidator struct{}

// NewReferenceValidator creates a new ReferenceValidator.
func NewReferenceValidator() *ReferenceValidator {
	return &ReferenceValidator{}
}

// Validate checks all cross-references in the document and returns issues for
// any orphaned references found. Each issue includes the specific reference type
// and detailed context about where the broken reference was found.
func (v *ReferenceValidator) Validate(doc *gedcom.Document) []Issue {
	if doc == nil {
		return nil
	}

	var issues []Issue

	// Walk the records, not doc.Individuals()/Families(), so each entity's
	// pointer lines come from the record it was decoded from. The decoder
	// does not reject duplicate XRefs and XRefMap keeps only the last record
	// per XRef, so a lookup through XRefMap would give every duplicate the
	// last one's lines (ADR 0007).
	for _, record := range doc.Records {
		if record == nil {
			continue
		}
		if ind, ok := record.GetIndividual(); ok {
			issues = append(issues, v.checkIndividualReferences(doc, record, ind)...)
		}
		if fam, ok := record.GetFamily(); ok {
			issues = append(issues, v.checkFamilyReferences(doc, record, fam)...)
		}
	}

	return issues
}

// checkIndividualReferences validates all cross-references within an individual record.
// This includes FAMC (child-in-family), FAMS (spouse-in-family), and SOUR references.
func (v *ReferenceValidator) checkIndividualReferences(doc *gedcom.Document, record *gedcom.Record, ind *gedcom.Individual) []Issue {
	var issues []Issue

	lines := newPointerLines(record)

	// Check FAMC references (ChildInFamilies)
	for i, link := range ind.ChildInFamilies {
		if link.FamilyXRef == "" {
			continue
		}
		if doc.GetFamily(link.FamilyXRef) == nil {
			issue := NewIssue(
				SeverityError,
				CodeOrphanedFAMC,
				fmt.Sprintf("FAMC reference to non-existent family %s", link.FamilyXRef),
				ind.XRef,
			).WithRelatedXRef(link.FamilyXRef).
				WithLineNumber(lines.next("FAMC", link.FamilyXRef)).
				WithDetail("reference_type", string(RefTypeFAMC)).
				WithDetail("field", fmt.Sprintf("ChildInFamilies[%d]", i))
			issues = append(issues, issue)
		}
	}

	// Check FAMS references (SpouseInFamilies)
	for i, link := range ind.SpouseInFamilies {
		if link.FamilyXRef == "" {
			continue
		}
		if doc.GetFamily(link.FamilyXRef) == nil {
			issue := NewIssue(
				SeverityError,
				CodeOrphanedFAMS,
				fmt.Sprintf("FAMS reference to non-existent family %s", link.FamilyXRef),
				ind.XRef,
			).WithRelatedXRef(link.FamilyXRef).
				WithLineNumber(lines.next("FAMS", link.FamilyXRef)).
				WithDetail("reference_type", string(RefTypeFAMS)).
				WithDetail("field", fmt.Sprintf("SpouseInFamilies[%d]", i))
			issues = append(issues, issue)
		}
	}

	// Check SOUR references (SourceCitations)
	for i, citation := range ind.SourceCitations {
		if citation == nil || citation.SourceXRef == "" {
			continue
		}
		if doc.GetSource(citation.SourceXRef) == nil {
			issue := NewIssue(
				SeverityError,
				CodeOrphanedSOUR,
				fmt.Sprintf("SOUR reference to non-existent source %s", citation.SourceXRef),
				ind.XRef,
			).WithRelatedXRef(citation.SourceXRef).
				WithLineNumber(lines.next("SOUR", citation.SourceXRef)).
				WithDetail("reference_type", string(RefTypeSOUR)).
				WithDetail("field", fmt.Sprintf("SourceCitations[%d]", i))
			issues = append(issues, issue)
		}
	}

	return issues
}

// checkFamilyReferences validates all cross-references within a family record.
// This includes HUSB, WIFE, and CHIL references.
func (v *ReferenceValidator) checkFamilyReferences(doc *gedcom.Document, record *gedcom.Record, fam *gedcom.Family) []Issue {
	var issues []Issue
	lines := newPointerLines(record)

	// Check HUSB reference
	if fam.Husband != "" {
		if doc.GetIndividual(fam.Husband) == nil {
			issue := NewIssue(
				SeverityError,
				CodeOrphanedHUSB,
				fmt.Sprintf("HUSB reference to non-existent individual %s", fam.Husband),
				fam.XRef,
			).WithRelatedXRef(fam.Husband).
				WithLineNumber(lines.next("HUSB", fam.Husband)).
				WithDetail("reference_type", string(RefTypeHUSB)).
				WithDetail("field", "Husband")
			issues = append(issues, issue)
		}
	}

	// Check WIFE reference
	if fam.Wife != "" {
		if doc.GetIndividual(fam.Wife) == nil {
			issue := NewIssue(
				SeverityError,
				CodeOrphanedWIFE,
				fmt.Sprintf("WIFE reference to non-existent individual %s", fam.Wife),
				fam.XRef,
			).WithRelatedXRef(fam.Wife).
				WithLineNumber(lines.next("WIFE", fam.Wife)).
				WithDetail("reference_type", string(RefTypeWIFE)).
				WithDetail("field", "Wife")
			issues = append(issues, issue)
		}
	}

	// Check CHIL references
	for i, childXRef := range fam.Children {
		if childXRef == "" {
			continue
		}
		if doc.GetIndividual(childXRef) == nil {
			issue := NewIssue(
				SeverityError,
				CodeOrphanedCHIL,
				fmt.Sprintf("CHIL reference to non-existent individual %s", childXRef),
				fam.XRef,
			).WithRelatedXRef(childXRef).
				WithLineNumber(lines.next("CHIL", childXRef)).
				WithDetail("reference_type", string(RefTypeCHIL)).
				WithDetail("field", fmt.Sprintf("Children[%d]", i))
			issues = append(issues, issue)
		}
	}

	return issues
}

// pointerLines finds the source line of a typed pointer by looking up the
// level-1 raw tag it was decoded from. The typed entities carry no line
// numbers, but the record's raw Tags do (ADR 0003, ADR 0007).
type pointerLines struct {
	record *gedcom.Record
	seen   map[string]int // tag + "\x00" + xref -> occurrences already matched
}

// newPointerLines returns a line finder over record, the record the checked
// entity was decoded from, which must be non-nil. Do not look the record up by
// XRef: with duplicate XRefs, doc.XRefMap holds only the last one.
func newPointerLines(record *gedcom.Record) *pointerLines {
	return &pointerLines{record: record, seen: map[string]int{}}
}

// next returns the line number of the next unmatched level-1 tag named tag
// whose trimmed value is xref, so repeated pointers (two CHIL lines naming the
// same child) each get their own line. It returns 0 when the record has no
// such raw tag, as for an entity built in code.
func (p *pointerLines) next(tag, xref string) int {
	key := tag + "\x00" + xref
	skip := p.seen[key]
	p.seen[key]++
	for _, t := range p.record.Tags {
		if t == nil || t.Level != 1 || t.Tag != tag || strings.TrimSpace(t.Value) != xref {
			continue
		}
		if skip == 0 {
			return t.LineNumber
		}
		skip--
	}
	return 0
}

// ReferenceReport provides statistics about cross-references in a document.
type ReferenceReport struct {
	// TotalReferences is the total count of all cross-references found.
	TotalReferences int

	// ValidReferences is the count of references pointing to existing records.
	ValidReferences int

	// OrphanedReferences is the count of references pointing to non-existent records.
	OrphanedReferences int

	// ByType contains counts broken down by reference type.
	// Keys are ReferenceType values (FAMC, FAMS, HUSB, WIFE, CHIL, SOUR).
	// Values are counts for that reference type.
	ByType map[string]int

	// OrphanedByType contains orphaned reference counts broken down by type.
	OrphanedByType map[string]int
}

// Report generates a comprehensive reference statistics report for the document.
// It counts all references, validates them, and provides breakdowns by type.
func (v *ReferenceValidator) Report(doc *gedcom.Document) *ReferenceReport {
	report := &ReferenceReport{
		ByType:         make(map[string]int),
		OrphanedByType: make(map[string]int),
	}

	if doc == nil {
		return report
	}

	// Count and validate individual references
	for _, ind := range doc.Individuals() {
		v.countIndividualReferences(doc, ind, report)
	}

	// Count and validate family references
	for _, fam := range doc.Families() {
		v.countFamilyReferences(doc, fam, report)
	}

	return report
}

// countIndividualReferences counts all references in an individual record.
func (v *ReferenceValidator) countIndividualReferences(doc *gedcom.Document, ind *gedcom.Individual, report *ReferenceReport) {
	// Count FAMC references
	for _, link := range ind.ChildInFamilies {
		if link.FamilyXRef == "" {
			continue
		}
		report.TotalReferences++
		report.ByType[string(RefTypeFAMC)]++
		if doc.GetFamily(link.FamilyXRef) == nil {
			report.OrphanedReferences++
			report.OrphanedByType[string(RefTypeFAMC)]++
		} else {
			report.ValidReferences++
		}
	}

	// Count FAMS references
	for _, link := range ind.SpouseInFamilies {
		if link.FamilyXRef == "" {
			continue
		}
		report.TotalReferences++
		report.ByType[string(RefTypeFAMS)]++
		if doc.GetFamily(link.FamilyXRef) == nil {
			report.OrphanedReferences++
			report.OrphanedByType[string(RefTypeFAMS)]++
		} else {
			report.ValidReferences++
		}
	}

	// Count SOUR references
	for _, citation := range ind.SourceCitations {
		if citation == nil || citation.SourceXRef == "" {
			continue
		}
		report.TotalReferences++
		report.ByType[string(RefTypeSOUR)]++
		if doc.GetSource(citation.SourceXRef) == nil {
			report.OrphanedReferences++
			report.OrphanedByType[string(RefTypeSOUR)]++
		} else {
			report.ValidReferences++
		}
	}
}

// countFamilyReferences counts all references in a family record.
func (v *ReferenceValidator) countFamilyReferences(doc *gedcom.Document, fam *gedcom.Family, report *ReferenceReport) {
	// Count HUSB reference
	if fam.Husband != "" {
		report.TotalReferences++
		report.ByType[string(RefTypeHUSB)]++
		if doc.GetIndividual(fam.Husband) == nil {
			report.OrphanedReferences++
			report.OrphanedByType[string(RefTypeHUSB)]++
		} else {
			report.ValidReferences++
		}
	}

	// Count WIFE reference
	if fam.Wife != "" {
		report.TotalReferences++
		report.ByType[string(RefTypeWIFE)]++
		if doc.GetIndividual(fam.Wife) == nil {
			report.OrphanedReferences++
			report.OrphanedByType[string(RefTypeWIFE)]++
		} else {
			report.ValidReferences++
		}
	}

	// Count CHIL references
	for _, childXRef := range fam.Children {
		if childXRef == "" {
			continue
		}
		report.TotalReferences++
		report.ByType[string(RefTypeCHIL)]++
		if doc.GetIndividual(childXRef) == nil {
			report.OrphanedReferences++
			report.OrphanedByType[string(RefTypeCHIL)]++
		} else {
			report.ValidReferences++
		}
	}
}
