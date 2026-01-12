// references.go provides enhanced orphaned reference detection with typed validation
// for all GEDCOM cross-reference types.
//
// This module validates that all cross-references in a GEDCOM document point to
// existing records. It provides granular detection for different reference types:
// FAMC (child-in-family), FAMS (spouse-in-family), HUSB, WIFE, CHIL, and SOUR.

package validator

import (
	"fmt"

	"github.com/cacack/gedcom-go/gedcom"
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

	// Check individual references
	for _, ind := range doc.Individuals() {
		issues = append(issues, v.checkIndividualReferences(doc, ind)...)
	}

	// Check family references
	for _, fam := range doc.Families() {
		issues = append(issues, v.checkFamilyReferences(doc, fam)...)
	}

	return issues
}

// checkIndividualReferences validates all cross-references within an individual record.
// This includes FAMC (child-in-family), FAMS (spouse-in-family), and SOUR references.
func (v *ReferenceValidator) checkIndividualReferences(doc *gedcom.Document, ind *gedcom.Individual) []Issue {
	var issues []Issue

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
				WithDetail("reference_type", string(RefTypeFAMC)).
				WithDetail("field", fmt.Sprintf("ChildInFamilies[%d]", i))
			issues = append(issues, issue)
		}
	}

	// Check FAMS references (SpouseInFamilies)
	for i, famXRef := range ind.SpouseInFamilies {
		if famXRef == "" {
			continue
		}
		if doc.GetFamily(famXRef) == nil {
			issue := NewIssue(
				SeverityError,
				CodeOrphanedFAMS,
				fmt.Sprintf("FAMS reference to non-existent family %s", famXRef),
				ind.XRef,
			).WithRelatedXRef(famXRef).
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
				WithDetail("reference_type", string(RefTypeSOUR)).
				WithDetail("field", fmt.Sprintf("SourceCitations[%d]", i))
			issues = append(issues, issue)
		}
	}

	return issues
}

// checkFamilyReferences validates all cross-references within a family record.
// This includes HUSB, WIFE, and CHIL references.
func (v *ReferenceValidator) checkFamilyReferences(doc *gedcom.Document, fam *gedcom.Family) []Issue {
	var issues []Issue

	// Check HUSB reference
	if fam.Husband != "" {
		if doc.GetIndividual(fam.Husband) == nil {
			issue := NewIssue(
				SeverityError,
				CodeOrphanedHUSB,
				fmt.Sprintf("HUSB reference to non-existent individual %s", fam.Husband),
				fam.XRef,
			).WithRelatedXRef(fam.Husband).
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
				WithDetail("reference_type", string(RefTypeCHIL)).
				WithDetail("field", fmt.Sprintf("Children[%d]", i))
			issues = append(issues, issue)
		}
	}

	return issues
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
	for _, famXRef := range ind.SpouseInFamilies {
		if famXRef == "" {
			continue
		}
		report.TotalReferences++
		report.ByType[string(RefTypeFAMS)]++
		if doc.GetFamily(famXRef) == nil {
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
