package validator

import (
	"testing"

	"github.com/cacack/gedcom-go/gedcom"
)

// Helper to create a minimal document with XRefMap
func newTestDocument() *gedcom.Document {
	return &gedcom.Document{
		Records: []*gedcom.Record{},
		XRefMap: make(map[string]*gedcom.Record),
	}
}

// Helper to add an individual to the document
func addIndividual(doc *gedcom.Document, ind *gedcom.Individual) {
	record := &gedcom.Record{
		XRef:   ind.XRef,
		Type:   gedcom.RecordTypeIndividual,
		Entity: ind,
	}
	doc.Records = append(doc.Records, record)
	doc.XRefMap[ind.XRef] = record
}

// Helper to add a family to the document
func addFamily(doc *gedcom.Document, fam *gedcom.Family) {
	record := &gedcom.Record{
		XRef:   fam.XRef,
		Type:   gedcom.RecordTypeFamily,
		Entity: fam,
	}
	doc.Records = append(doc.Records, record)
	doc.XRefMap[fam.XRef] = record
}

// Helper to add a source to the document
func addSource(doc *gedcom.Document, src *gedcom.Source) {
	record := &gedcom.Record{
		XRef:   src.XRef,
		Type:   gedcom.RecordTypeSource,
		Entity: src,
	}
	doc.Records = append(doc.Records, record)
	doc.XRefMap[src.XRef] = record
}

func TestNewReferenceValidator(t *testing.T) {
	v := NewReferenceValidator()
	if v == nil {
		t.Error("NewReferenceValidator() returned nil")
	}
}

func TestReferenceValidatorValidate_NilDocument(t *testing.T) {
	v := NewReferenceValidator()
	issues := v.Validate(nil)
	if issues != nil {
		t.Errorf("Validate(nil) should return nil, got %d issues", len(issues))
	}
}

func TestReferenceValidatorValidate_EmptyDocument(t *testing.T) {
	v := NewReferenceValidator()
	doc := newTestDocument()

	issues := v.Validate(doc)
	if len(issues) != 0 {
		t.Errorf("Validate on empty document should return 0 issues, got %d", len(issues))
	}
}

func TestReferenceValidatorValidate_AllValid(t *testing.T) {
	v := NewReferenceValidator()
	doc := newTestDocument()

	// Create valid structure: Family with husband, wife, and child
	ind1 := &gedcom.Individual{XRef: "@I1@"}
	ind2 := &gedcom.Individual{XRef: "@I2@"}
	ind3 := &gedcom.Individual{XRef: "@I3@"}
	fam := &gedcom.Family{
		XRef:     "@F1@",
		Husband:  "@I1@",
		Wife:     "@I2@",
		Children: []string{"@I3@"},
	}
	src := &gedcom.Source{XRef: "@S1@"}

	// Set up bidirectional links
	ind1.SpouseInFamilies = []string{"@F1@"}
	ind2.SpouseInFamilies = []string{"@F1@"}
	ind3.ChildInFamilies = []gedcom.FamilyLink{{FamilyXRef: "@F1@"}}
	ind1.SourceCitations = []*gedcom.SourceCitation{{SourceXRef: "@S1@"}}

	addIndividual(doc, ind1)
	addIndividual(doc, ind2)
	addIndividual(doc, ind3)
	addFamily(doc, fam)
	addSource(doc, src)

	issues := v.Validate(doc)
	if len(issues) != 0 {
		t.Errorf("Expected 0 issues for valid references, got %d", len(issues))
		for _, issue := range issues {
			t.Logf("  Issue: %s", issue.String())
		}
	}
}

func TestReferenceValidatorValidate_OrphanedFAMC(t *testing.T) {
	v := NewReferenceValidator()
	doc := newTestDocument()

	ind := &gedcom.Individual{
		XRef: "@I1@",
		ChildInFamilies: []gedcom.FamilyLink{
			{FamilyXRef: "@F999@"}, // Non-existent family
		},
	}
	addIndividual(doc, ind)

	issues := v.Validate(doc)

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]
	if issue.Code != CodeOrphanedFAMC {
		t.Errorf("Expected code %s, got %s", CodeOrphanedFAMC, issue.Code)
	}
	if issue.Severity != SeverityError {
		t.Errorf("Expected severity ERROR, got %s", issue.Severity)
	}
	if issue.RecordXRef != "@I1@" {
		t.Errorf("Expected RecordXRef @I1@, got %s", issue.RecordXRef)
	}
	if issue.RelatedXRef != "@F999@" {
		t.Errorf("Expected RelatedXRef @F999@, got %s", issue.RelatedXRef)
	}
	if issue.Details["reference_type"] != "FAMC" {
		t.Errorf("Expected reference_type FAMC, got %s", issue.Details["reference_type"])
	}
	if issue.Details["field"] != "ChildInFamilies[0]" {
		t.Errorf("Expected field ChildInFamilies[0], got %s", issue.Details["field"])
	}
}

func TestReferenceValidatorValidate_OrphanedFAMS(t *testing.T) {
	v := NewReferenceValidator()
	doc := newTestDocument()

	ind := &gedcom.Individual{
		XRef:             "@I1@",
		SpouseInFamilies: []string{"@F999@"}, // Non-existent family
	}
	addIndividual(doc, ind)

	issues := v.Validate(doc)

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]
	if issue.Code != CodeOrphanedFAMS {
		t.Errorf("Expected code %s, got %s", CodeOrphanedFAMS, issue.Code)
	}
	if issue.Severity != SeverityError {
		t.Errorf("Expected severity ERROR, got %s", issue.Severity)
	}
	if issue.RecordXRef != "@I1@" {
		t.Errorf("Expected RecordXRef @I1@, got %s", issue.RecordXRef)
	}
	if issue.RelatedXRef != "@F999@" {
		t.Errorf("Expected RelatedXRef @F999@, got %s", issue.RelatedXRef)
	}
	if issue.Details["reference_type"] != "FAMS" {
		t.Errorf("Expected reference_type FAMS, got %s", issue.Details["reference_type"])
	}
	if issue.Details["field"] != "SpouseInFamilies[0]" {
		t.Errorf("Expected field SpouseInFamilies[0], got %s", issue.Details["field"])
	}
}

func TestReferenceValidatorValidate_OrphanedHUSB(t *testing.T) {
	v := NewReferenceValidator()
	doc := newTestDocument()

	fam := &gedcom.Family{
		XRef:    "@F1@",
		Husband: "@I999@", // Non-existent individual
	}
	addFamily(doc, fam)

	issues := v.Validate(doc)

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]
	if issue.Code != CodeOrphanedHUSB {
		t.Errorf("Expected code %s, got %s", CodeOrphanedHUSB, issue.Code)
	}
	if issue.Severity != SeverityError {
		t.Errorf("Expected severity ERROR, got %s", issue.Severity)
	}
	if issue.RecordXRef != "@F1@" {
		t.Errorf("Expected RecordXRef @F1@, got %s", issue.RecordXRef)
	}
	if issue.RelatedXRef != "@I999@" {
		t.Errorf("Expected RelatedXRef @I999@, got %s", issue.RelatedXRef)
	}
	if issue.Details["reference_type"] != "HUSB" {
		t.Errorf("Expected reference_type HUSB, got %s", issue.Details["reference_type"])
	}
	if issue.Details["field"] != "Husband" {
		t.Errorf("Expected field Husband, got %s", issue.Details["field"])
	}
}

func TestReferenceValidatorValidate_OrphanedWIFE(t *testing.T) {
	v := NewReferenceValidator()
	doc := newTestDocument()

	fam := &gedcom.Family{
		XRef: "@F1@",
		Wife: "@I999@", // Non-existent individual
	}
	addFamily(doc, fam)

	issues := v.Validate(doc)

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]
	if issue.Code != CodeOrphanedWIFE {
		t.Errorf("Expected code %s, got %s", CodeOrphanedWIFE, issue.Code)
	}
	if issue.Severity != SeverityError {
		t.Errorf("Expected severity ERROR, got %s", issue.Severity)
	}
	if issue.RecordXRef != "@F1@" {
		t.Errorf("Expected RecordXRef @F1@, got %s", issue.RecordXRef)
	}
	if issue.RelatedXRef != "@I999@" {
		t.Errorf("Expected RelatedXRef @I999@, got %s", issue.RelatedXRef)
	}
	if issue.Details["reference_type"] != "WIFE" {
		t.Errorf("Expected reference_type WIFE, got %s", issue.Details["reference_type"])
	}
	if issue.Details["field"] != "Wife" {
		t.Errorf("Expected field Wife, got %s", issue.Details["field"])
	}
}

func TestReferenceValidatorValidate_OrphanedCHIL(t *testing.T) {
	v := NewReferenceValidator()
	doc := newTestDocument()

	fam := &gedcom.Family{
		XRef:     "@F1@",
		Children: []string{"@I999@"}, // Non-existent individual
	}
	addFamily(doc, fam)

	issues := v.Validate(doc)

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]
	if issue.Code != CodeOrphanedCHIL {
		t.Errorf("Expected code %s, got %s", CodeOrphanedCHIL, issue.Code)
	}
	if issue.Severity != SeverityError {
		t.Errorf("Expected severity ERROR, got %s", issue.Severity)
	}
	if issue.RecordXRef != "@F1@" {
		t.Errorf("Expected RecordXRef @F1@, got %s", issue.RecordXRef)
	}
	if issue.RelatedXRef != "@I999@" {
		t.Errorf("Expected RelatedXRef @I999@, got %s", issue.RelatedXRef)
	}
	if issue.Details["reference_type"] != "CHIL" {
		t.Errorf("Expected reference_type CHIL, got %s", issue.Details["reference_type"])
	}
	if issue.Details["field"] != "Children[0]" {
		t.Errorf("Expected field Children[0], got %s", issue.Details["field"])
	}
}

func TestReferenceValidatorValidate_OrphanedSOUR(t *testing.T) {
	v := NewReferenceValidator()
	doc := newTestDocument()

	ind := &gedcom.Individual{
		XRef: "@I1@",
		SourceCitations: []*gedcom.SourceCitation{
			{SourceXRef: "@S999@"}, // Non-existent source
		},
	}
	addIndividual(doc, ind)

	issues := v.Validate(doc)

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]
	if issue.Code != CodeOrphanedSOUR {
		t.Errorf("Expected code %s, got %s", CodeOrphanedSOUR, issue.Code)
	}
	if issue.Severity != SeverityError {
		t.Errorf("Expected severity ERROR, got %s", issue.Severity)
	}
	if issue.RecordXRef != "@I1@" {
		t.Errorf("Expected RecordXRef @I1@, got %s", issue.RecordXRef)
	}
	if issue.RelatedXRef != "@S999@" {
		t.Errorf("Expected RelatedXRef @S999@, got %s", issue.RelatedXRef)
	}
	if issue.Details["reference_type"] != "SOUR" {
		t.Errorf("Expected reference_type SOUR, got %s", issue.Details["reference_type"])
	}
	if issue.Details["field"] != "SourceCitations[0]" {
		t.Errorf("Expected field SourceCitations[0], got %s", issue.Details["field"])
	}
}

func TestReferenceValidatorValidate_MultipleOrphans(t *testing.T) {
	v := NewReferenceValidator()
	doc := newTestDocument()

	ind := &gedcom.Individual{
		XRef: "@I1@",
		ChildInFamilies: []gedcom.FamilyLink{
			{FamilyXRef: "@F999@"},
			{FamilyXRef: "@F998@"},
		},
		SpouseInFamilies: []string{"@F997@"},
	}
	addIndividual(doc, ind)

	fam := &gedcom.Family{
		XRef:     "@F1@",
		Husband:  "@I999@",
		Wife:     "@I998@",
		Children: []string{"@I997@", "@I996@"},
	}
	addFamily(doc, fam)

	issues := v.Validate(doc)

	// Should have: 2 FAMC + 1 FAMS + 1 HUSB + 1 WIFE + 2 CHIL = 7 issues
	if len(issues) != 7 {
		t.Errorf("Expected 7 issues, got %d", len(issues))
		for _, issue := range issues {
			t.Logf("  Issue: %s", issue.String())
		}
	}

	// Count by code
	counts := make(map[string]int)
	for _, issue := range issues {
		counts[issue.Code]++
	}

	if counts[CodeOrphanedFAMC] != 2 {
		t.Errorf("Expected 2 ORPHANED_FAMC, got %d", counts[CodeOrphanedFAMC])
	}
	if counts[CodeOrphanedFAMS] != 1 {
		t.Errorf("Expected 1 ORPHANED_FAMS, got %d", counts[CodeOrphanedFAMS])
	}
	if counts[CodeOrphanedHUSB] != 1 {
		t.Errorf("Expected 1 ORPHANED_HUSB, got %d", counts[CodeOrphanedHUSB])
	}
	if counts[CodeOrphanedWIFE] != 1 {
		t.Errorf("Expected 1 ORPHANED_WIFE, got %d", counts[CodeOrphanedWIFE])
	}
	if counts[CodeOrphanedCHIL] != 2 {
		t.Errorf("Expected 2 ORPHANED_CHIL, got %d", counts[CodeOrphanedCHIL])
	}
}

func TestReferenceValidatorValidate_EmptyXRefs(t *testing.T) {
	v := NewReferenceValidator()
	doc := newTestDocument()

	// Empty xrefs should be skipped, not reported as errors
	ind := &gedcom.Individual{
		XRef: "@I1@",
		ChildInFamilies: []gedcom.FamilyLink{
			{FamilyXRef: ""}, // Empty - should be skipped
		},
		SpouseInFamilies: []string{""}, // Empty - should be skipped
		SourceCitations: []*gedcom.SourceCitation{
			{SourceXRef: ""}, // Empty - should be skipped
			nil,              // Nil - should be skipped
		},
	}
	addIndividual(doc, ind)

	fam := &gedcom.Family{
		XRef:     "@F1@",
		Husband:  "",           // Empty - should be skipped
		Wife:     "",           // Empty - should be skipped
		Children: []string{""}, // Empty - should be skipped
	}
	addFamily(doc, fam)

	issues := v.Validate(doc)
	if len(issues) != 0 {
		t.Errorf("Expected 0 issues for empty xrefs, got %d", len(issues))
		for _, issue := range issues {
			t.Logf("  Issue: %s", issue.String())
		}
	}
}

func TestReferenceValidatorValidate_IndexTracking(t *testing.T) {
	v := NewReferenceValidator()
	doc := newTestDocument()

	// Create valid family to mix with orphaned references
	fam := &gedcom.Family{XRef: "@F1@"}
	addFamily(doc, fam)

	ind := &gedcom.Individual{
		XRef: "@I1@",
		ChildInFamilies: []gedcom.FamilyLink{
			{FamilyXRef: "@F1@"},   // Valid (index 0)
			{FamilyXRef: "@F999@"}, // Invalid (index 1)
			{FamilyXRef: "@F1@"},   // Valid (index 2)
			{FamilyXRef: "@F998@"}, // Invalid (index 3)
		},
	}
	addIndividual(doc, ind)

	issues := v.Validate(doc)

	if len(issues) != 2 {
		t.Fatalf("Expected 2 issues, got %d", len(issues))
	}

	// Check field indices are correct
	fields := make(map[string]bool)
	for _, issue := range issues {
		fields[issue.Details["field"]] = true
	}

	if !fields["ChildInFamilies[1]"] {
		t.Error("Expected issue for ChildInFamilies[1]")
	}
	if !fields["ChildInFamilies[3]"] {
		t.Error("Expected issue for ChildInFamilies[3]")
	}
}

func TestReferenceValidatorReport_NilDocument(t *testing.T) {
	v := NewReferenceValidator()
	report := v.Report(nil)

	if report == nil {
		t.Fatal("Report(nil) should not return nil")
	}
	if report.TotalReferences != 0 {
		t.Errorf("Expected TotalReferences 0, got %d", report.TotalReferences)
	}
	if report.ValidReferences != 0 {
		t.Errorf("Expected ValidReferences 0, got %d", report.ValidReferences)
	}
	if report.OrphanedReferences != 0 {
		t.Errorf("Expected OrphanedReferences 0, got %d", report.OrphanedReferences)
	}
	if report.ByType == nil {
		t.Error("ByType should be initialized")
	}
	if report.OrphanedByType == nil {
		t.Error("OrphanedByType should be initialized")
	}
}

func TestReferenceValidatorReport_EmptyDocument(t *testing.T) {
	v := NewReferenceValidator()
	doc := newTestDocument()

	report := v.Report(doc)

	if report.TotalReferences != 0 {
		t.Errorf("Expected TotalReferences 0, got %d", report.TotalReferences)
	}
	if report.ValidReferences != 0 {
		t.Errorf("Expected ValidReferences 0, got %d", report.ValidReferences)
	}
	if report.OrphanedReferences != 0 {
		t.Errorf("Expected OrphanedReferences 0, got %d", report.OrphanedReferences)
	}
}

func TestReferenceValidatorReport_AllValid(t *testing.T) {
	v := NewReferenceValidator()
	doc := newTestDocument()

	// Create valid structure
	ind1 := &gedcom.Individual{XRef: "@I1@"}
	ind2 := &gedcom.Individual{XRef: "@I2@"}
	ind3 := &gedcom.Individual{XRef: "@I3@"}
	fam := &gedcom.Family{
		XRef:     "@F1@",
		Husband:  "@I1@",
		Wife:     "@I2@",
		Children: []string{"@I3@"},
	}
	src := &gedcom.Source{XRef: "@S1@"}

	ind1.SpouseInFamilies = []string{"@F1@"}
	ind2.SpouseInFamilies = []string{"@F1@"}
	ind3.ChildInFamilies = []gedcom.FamilyLink{{FamilyXRef: "@F1@"}}
	ind1.SourceCitations = []*gedcom.SourceCitation{{SourceXRef: "@S1@"}}

	addIndividual(doc, ind1)
	addIndividual(doc, ind2)
	addIndividual(doc, ind3)
	addFamily(doc, fam)
	addSource(doc, src)

	report := v.Report(doc)

	// Total: 2 FAMS + 1 FAMC + 1 SOUR + 1 HUSB + 1 WIFE + 1 CHIL = 7
	if report.TotalReferences != 7 {
		t.Errorf("Expected TotalReferences 7, got %d", report.TotalReferences)
	}
	if report.ValidReferences != 7 {
		t.Errorf("Expected ValidReferences 7, got %d", report.ValidReferences)
	}
	if report.OrphanedReferences != 0 {
		t.Errorf("Expected OrphanedReferences 0, got %d", report.OrphanedReferences)
	}

	// Check ByType
	if report.ByType["FAMS"] != 2 {
		t.Errorf("Expected ByType[FAMS] 2, got %d", report.ByType["FAMS"])
	}
	if report.ByType["FAMC"] != 1 {
		t.Errorf("Expected ByType[FAMC] 1, got %d", report.ByType["FAMC"])
	}
	if report.ByType["SOUR"] != 1 {
		t.Errorf("Expected ByType[SOUR] 1, got %d", report.ByType["SOUR"])
	}
	if report.ByType["HUSB"] != 1 {
		t.Errorf("Expected ByType[HUSB] 1, got %d", report.ByType["HUSB"])
	}
	if report.ByType["WIFE"] != 1 {
		t.Errorf("Expected ByType[WIFE] 1, got %d", report.ByType["WIFE"])
	}
	if report.ByType["CHIL"] != 1 {
		t.Errorf("Expected ByType[CHIL] 1, got %d", report.ByType["CHIL"])
	}
}

func TestReferenceValidatorReport_WithOrphans(t *testing.T) {
	v := NewReferenceValidator()
	doc := newTestDocument()

	// Create mix of valid and orphaned references
	ind1 := &gedcom.Individual{XRef: "@I1@"}
	fam := &gedcom.Family{
		XRef:     "@F1@",
		Husband:  "@I1@",                     // Valid
		Wife:     "@I999@",                   // Orphaned
		Children: []string{"@I1@", "@I998@"}, // 1 valid, 1 orphaned
	}

	ind1.SpouseInFamilies = []string{"@F1@", "@F999@"} // 1 valid, 1 orphaned

	addIndividual(doc, ind1)
	addFamily(doc, fam)

	report := v.Report(doc)

	// Total: 2 FAMS + 1 HUSB + 1 WIFE + 2 CHIL = 6
	if report.TotalReferences != 6 {
		t.Errorf("Expected TotalReferences 6, got %d", report.TotalReferences)
	}

	// Valid: 1 FAMS + 1 HUSB + 1 CHIL = 3
	if report.ValidReferences != 3 {
		t.Errorf("Expected ValidReferences 3, got %d", report.ValidReferences)
	}

	// Orphaned: 1 FAMS + 1 WIFE + 1 CHIL = 3
	if report.OrphanedReferences != 3 {
		t.Errorf("Expected OrphanedReferences 3, got %d", report.OrphanedReferences)
	}

	// Check OrphanedByType
	if report.OrphanedByType["FAMS"] != 1 {
		t.Errorf("Expected OrphanedByType[FAMS] 1, got %d", report.OrphanedByType["FAMS"])
	}
	if report.OrphanedByType["WIFE"] != 1 {
		t.Errorf("Expected OrphanedByType[WIFE] 1, got %d", report.OrphanedByType["WIFE"])
	}
	if report.OrphanedByType["CHIL"] != 1 {
		t.Errorf("Expected OrphanedByType[CHIL] 1, got %d", report.OrphanedByType["CHIL"])
	}

	// Verify counts add up
	if report.ValidReferences+report.OrphanedReferences != report.TotalReferences {
		t.Error("ValidReferences + OrphanedReferences should equal TotalReferences")
	}
}

func TestReferenceValidatorReport_AllOrphanedTypes(t *testing.T) {
	// Test that Report correctly counts all orphaned reference types
	v := NewReferenceValidator()
	doc := newTestDocument()

	ind := &gedcom.Individual{
		XRef:            "@I1@",
		ChildInFamilies: []gedcom.FamilyLink{{FamilyXRef: "@F999@"}},      // Orphaned FAMC
		SourceCitations: []*gedcom.SourceCitation{{SourceXRef: "@S999@"}}, // Orphaned SOUR
	}
	addIndividual(doc, ind)

	fam := &gedcom.Family{
		XRef:    "@F1@",
		Husband: "@I999@", // Orphaned HUSB
	}
	addFamily(doc, fam)

	report := v.Report(doc)

	// Check orphaned FAMC was counted
	if report.OrphanedByType["FAMC"] != 1 {
		t.Errorf("Expected OrphanedByType[FAMC] 1, got %d", report.OrphanedByType["FAMC"])
	}

	// Check orphaned SOUR was counted
	if report.OrphanedByType["SOUR"] != 1 {
		t.Errorf("Expected OrphanedByType[SOUR] 1, got %d", report.OrphanedByType["SOUR"])
	}

	// Check orphaned HUSB was counted
	if report.OrphanedByType["HUSB"] != 1 {
		t.Errorf("Expected OrphanedByType[HUSB] 1, got %d", report.OrphanedByType["HUSB"])
	}

	if report.TotalReferences != 3 {
		t.Errorf("Expected TotalReferences 3, got %d", report.TotalReferences)
	}
	if report.OrphanedReferences != 3 {
		t.Errorf("Expected OrphanedReferences 3, got %d", report.OrphanedReferences)
	}
}

func TestReferenceValidatorReport_EmptyXRefs(t *testing.T) {
	v := NewReferenceValidator()
	doc := newTestDocument()

	// Empty xrefs should not be counted
	ind := &gedcom.Individual{
		XRef: "@I1@",
		ChildInFamilies: []gedcom.FamilyLink{
			{FamilyXRef: ""},
		},
		SpouseInFamilies: []string{""},
		SourceCitations: []*gedcom.SourceCitation{
			{SourceXRef: ""},
			nil,
		},
	}
	addIndividual(doc, ind)

	fam := &gedcom.Family{
		XRef:     "@F1@",
		Husband:  "",
		Wife:     "",
		Children: []string{""},
	}
	addFamily(doc, fam)

	report := v.Report(doc)

	if report.TotalReferences != 0 {
		t.Errorf("Expected TotalReferences 0 for empty xrefs, got %d", report.TotalReferences)
	}
}

func TestReferenceTypeConstants(t *testing.T) {
	// Verify reference type constants match expected GEDCOM tags
	types := []struct {
		refType  ReferenceType
		expected string
	}{
		{RefTypeFAMC, "FAMC"},
		{RefTypeFAMS, "FAMS"},
		{RefTypeHUSB, "HUSB"},
		{RefTypeWIFE, "WIFE"},
		{RefTypeCHIL, "CHIL"},
		{RefTypeSOUR, "SOUR"},
	}

	for _, tt := range types {
		if string(tt.refType) != tt.expected {
			t.Errorf("ReferenceType %v = %q, want %q", tt.refType, string(tt.refType), tt.expected)
		}
	}
}

func TestReferenceValidatorValidate_IssueDetails(t *testing.T) {
	// Test that all issue details are properly set
	v := NewReferenceValidator()
	doc := newTestDocument()

	ind := &gedcom.Individual{
		XRef: "@I1@",
		ChildInFamilies: []gedcom.FamilyLink{
			{FamilyXRef: "@F999@"},
		},
	}
	addIndividual(doc, ind)

	issues := v.Validate(doc)

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]

	// Verify all expected details are present
	if _, ok := issue.Details["reference_type"]; !ok {
		t.Error("Details should contain 'reference_type'")
	}
	if _, ok := issue.Details["field"]; !ok {
		t.Error("Details should contain 'field'")
	}

	// Verify issue implements error interface
	var _ error = issue

	// Verify Error() returns non-empty string
	if issue.Error() == "" {
		t.Error("Issue.Error() should return non-empty string")
	}
}

func TestReferenceValidatorValidate_MessageFormat(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(*gedcom.Document)
		expectedCode string
		expectedMsg  string
	}{
		{
			name: "FAMC message",
			setup: func(doc *gedcom.Document) {
				ind := &gedcom.Individual{
					XRef:            "@I1@",
					ChildInFamilies: []gedcom.FamilyLink{{FamilyXRef: "@F999@"}},
				}
				addIndividual(doc, ind)
			},
			expectedCode: CodeOrphanedFAMC,
			expectedMsg:  "FAMC reference to non-existent family @F999@",
		},
		{
			name: "FAMS message",
			setup: func(doc *gedcom.Document) {
				ind := &gedcom.Individual{
					XRef:             "@I1@",
					SpouseInFamilies: []string{"@F999@"},
				}
				addIndividual(doc, ind)
			},
			expectedCode: CodeOrphanedFAMS,
			expectedMsg:  "FAMS reference to non-existent family @F999@",
		},
		{
			name: "HUSB message",
			setup: func(doc *gedcom.Document) {
				fam := &gedcom.Family{XRef: "@F1@", Husband: "@I999@"}
				addFamily(doc, fam)
			},
			expectedCode: CodeOrphanedHUSB,
			expectedMsg:  "HUSB reference to non-existent individual @I999@",
		},
		{
			name: "WIFE message",
			setup: func(doc *gedcom.Document) {
				fam := &gedcom.Family{XRef: "@F1@", Wife: "@I999@"}
				addFamily(doc, fam)
			},
			expectedCode: CodeOrphanedWIFE,
			expectedMsg:  "WIFE reference to non-existent individual @I999@",
		},
		{
			name: "CHIL message",
			setup: func(doc *gedcom.Document) {
				fam := &gedcom.Family{XRef: "@F1@", Children: []string{"@I999@"}}
				addFamily(doc, fam)
			},
			expectedCode: CodeOrphanedCHIL,
			expectedMsg:  "CHIL reference to non-existent individual @I999@",
		},
		{
			name: "SOUR message",
			setup: func(doc *gedcom.Document) {
				ind := &gedcom.Individual{
					XRef:            "@I1@",
					SourceCitations: []*gedcom.SourceCitation{{SourceXRef: "@S999@"}},
				}
				addIndividual(doc, ind)
			},
			expectedCode: CodeOrphanedSOUR,
			expectedMsg:  "SOUR reference to non-existent source @S999@",
		},
	}

	v := NewReferenceValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := newTestDocument()
			tt.setup(doc)

			issues := v.Validate(doc)

			if len(issues) != 1 {
				t.Fatalf("Expected 1 issue, got %d", len(issues))
			}

			if issues[0].Code != tt.expectedCode {
				t.Errorf("Code = %q, want %q", issues[0].Code, tt.expectedCode)
			}
			if issues[0].Message != tt.expectedMsg {
				t.Errorf("Message = %q, want %q", issues[0].Message, tt.expectedMsg)
			}
		})
	}
}

func TestReferenceValidatorReport_Accuracy(t *testing.T) {
	// Comprehensive test to verify report counts are accurate
	v := NewReferenceValidator()
	doc := newTestDocument()

	// Create 3 valid individuals and 1 valid family and 1 valid source
	ind1 := &gedcom.Individual{XRef: "@I1@"}
	ind2 := &gedcom.Individual{XRef: "@I2@"}
	ind3 := &gedcom.Individual{XRef: "@I3@"}
	fam := &gedcom.Family{XRef: "@F1@"}
	src := &gedcom.Source{XRef: "@S1@"}

	addIndividual(doc, ind1)
	addIndividual(doc, ind2)
	addIndividual(doc, ind3)
	addFamily(doc, fam)
	addSource(doc, src)

	// Individual 1: 1 valid FAMS, 1 orphaned FAMS, 1 valid SOUR
	ind1.SpouseInFamilies = []string{"@F1@", "@F999@"}
	ind1.SourceCitations = []*gedcom.SourceCitation{{SourceXRef: "@S1@"}}

	// Individual 2: 1 valid FAMC, 1 orphaned FAMC
	ind2.ChildInFamilies = []gedcom.FamilyLink{
		{FamilyXRef: "@F1@"},
		{FamilyXRef: "@F998@"},
	}

	// Family 1: 1 valid HUSB, 1 orphaned WIFE, 2 valid CHIL, 1 orphaned CHIL
	fam.Husband = "@I1@"
	fam.Wife = "@I997@"
	fam.Children = []string{"@I2@", "@I3@", "@I996@"}

	report := v.Report(doc)

	// Expected totals:
	// FAMS: 2 (1 valid, 1 orphaned)
	// SOUR: 1 (1 valid)
	// FAMC: 2 (1 valid, 1 orphaned)
	// HUSB: 1 (1 valid)
	// WIFE: 1 (0 valid, 1 orphaned)
	// CHIL: 3 (2 valid, 1 orphaned)
	// Total: 10, Valid: 6, Orphaned: 4

	expectedTotal := 10
	expectedValid := 6
	expectedOrphaned := 4

	if report.TotalReferences != expectedTotal {
		t.Errorf("TotalReferences = %d, want %d", report.TotalReferences, expectedTotal)
	}
	if report.ValidReferences != expectedValid {
		t.Errorf("ValidReferences = %d, want %d", report.ValidReferences, expectedValid)
	}
	if report.OrphanedReferences != expectedOrphaned {
		t.Errorf("OrphanedReferences = %d, want %d", report.OrphanedReferences, expectedOrphaned)
	}

	// Verify sum equals total
	totalByType := 0
	for _, count := range report.ByType {
		totalByType += count
	}
	if totalByType != report.TotalReferences {
		t.Errorf("Sum of ByType (%d) != TotalReferences (%d)", totalByType, report.TotalReferences)
	}

	// Verify orphaned sum
	totalOrphanedByType := 0
	for _, count := range report.OrphanedByType {
		totalOrphanedByType += count
	}
	if totalOrphanedByType != report.OrphanedReferences {
		t.Errorf("Sum of OrphanedByType (%d) != OrphanedReferences (%d)", totalOrphanedByType, report.OrphanedReferences)
	}
}
