package validator

import (
	"fmt"
	"sort"
	"testing"

	"github.com/cacack/gedcom-go/gedcom"
)

func TestNewStreamingValidator(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})
	if sv == nil {
		t.Fatal("NewStreamingValidator() returned nil")
	}
	if sv.seenXRefs == nil {
		t.Error("seenXRefs map not initialized")
	}
	if sv.usedXRefs == nil {
		t.Error("usedXRefs map not initialized")
	}
	if sv.xrefTypes == nil {
		t.Error("xrefTypes map not initialized")
	}
	if sv.dateLogic == nil {
		t.Error("dateLogic validator not initialized")
	}
}

func TestNewStreamingValidator_WithOptions(t *testing.T) {
	opts := StreamingOptions{
		DateLogic: &DateLogicConfig{
			MaxReasonableAge: 100,
		},
		Strictness: StrictnessStrict,
	}
	sv := NewStreamingValidator(opts)

	if sv.opts.Strictness != StrictnessStrict {
		t.Errorf("Expected Strictness StrictnessStrict, got %v", sv.opts.Strictness)
	}
}

func TestStreamingValidator_ValidateRecord_NilRecord(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})
	issues := sv.ValidateRecord(nil)
	if issues != nil {
		t.Errorf("ValidateRecord(nil) should return nil, got %d issues", len(issues))
	}
}

func TestStreamingValidator_ValidateRecord_ValidIndividual(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	ind := &gedcom.Individual{
		XRef:  "@I1@",
		Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
	}
	record := &gedcom.Record{
		XRef:   "@I1@",
		Type:   gedcom.RecordTypeIndividual,
		Entity: ind,
	}

	issues := sv.ValidateRecord(record)
	if len(issues) != 0 {
		t.Errorf("Expected 0 issues for valid individual, got %d", len(issues))
	}

	// Verify XRef was registered
	if _, exists := sv.seenXRefs["@I1@"]; !exists {
		t.Error("XRef @I1@ should be registered in seenXRefs")
	}
}

func TestStreamingValidator_ValidateRecord_RegistersXRefs(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	// Create records with various XRefs
	records := []*gedcom.Record{
		{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: "@I1@"}},
		{XRef: "@I2@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: "@I2@"}},
		{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: &gedcom.Family{XRef: "@F1@"}},
		{XRef: "@S1@", Type: gedcom.RecordTypeSource, Entity: &gedcom.Source{XRef: "@S1@"}},
	}

	for _, record := range records {
		sv.ValidateRecord(record)
	}

	// Verify all XRefs were registered
	expectedXRefs := []string{"@I1@", "@I2@", "@F1@", "@S1@"}
	for _, xref := range expectedXRefs {
		if _, exists := sv.seenXRefs[xref]; !exists {
			t.Errorf("XRef %s should be registered in seenXRefs", xref)
		}
	}

	// Verify record types were registered
	if sv.xrefTypes["@I1@"] != gedcom.RecordTypeIndividual {
		t.Errorf("Expected @I1@ type Individual, got %v", sv.xrefTypes["@I1@"])
	}
	if sv.xrefTypes["@F1@"] != gedcom.RecordTypeFamily {
		t.Errorf("Expected @F1@ type Family, got %v", sv.xrefTypes["@F1@"])
	}
}

func TestStreamingValidator_ValidateRecord_CollectsFAMC(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	ind := &gedcom.Individual{
		XRef: "@I1@",
		ChildInFamilies: []gedcom.FamilyLink{
			{FamilyXRef: "@F1@"},
			{FamilyXRef: "@F2@"},
		},
	}
	record := &gedcom.Record{
		XRef:   "@I1@",
		Type:   gedcom.RecordTypeIndividual,
		Entity: ind,
	}

	sv.ValidateRecord(record)

	// Verify FAMC references were collected
	if len(sv.usedXRefs["@F1@"]) != 1 {
		t.Errorf("Expected 1 usage for @F1@, got %d", len(sv.usedXRefs["@F1@"]))
	}
	if len(sv.usedXRefs["@F2@"]) != 1 {
		t.Errorf("Expected 1 usage for @F2@, got %d", len(sv.usedXRefs["@F2@"]))
	}

	// Verify usage details
	usage := sv.usedXRefs["@F1@"][0]
	if usage.RecordXRef != "@I1@" {
		t.Errorf("Expected RecordXRef @I1@, got %s", usage.RecordXRef)
	}
	if usage.Context != "FAMC" {
		t.Errorf("Expected Context FAMC, got %s", usage.Context)
	}
	if usage.Field != "ChildInFamilies[0]" {
		t.Errorf("Expected Field ChildInFamilies[0], got %s", usage.Field)
	}
}

func TestStreamingValidator_ValidateRecord_CollectsFAMS(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	ind := &gedcom.Individual{
		XRef:             "@I1@",
		SpouseInFamilies: []string{"@F1@", "@F2@"},
	}
	record := &gedcom.Record{
		XRef:   "@I1@",
		Type:   gedcom.RecordTypeIndividual,
		Entity: ind,
	}

	sv.ValidateRecord(record)

	// Verify FAMS references were collected
	if len(sv.usedXRefs["@F1@"]) != 1 {
		t.Errorf("Expected 1 usage for @F1@, got %d", len(sv.usedXRefs["@F1@"]))
	}

	usage := sv.usedXRefs["@F1@"][0]
	if usage.Context != "FAMS" {
		t.Errorf("Expected Context FAMS, got %s", usage.Context)
	}
}

func TestStreamingValidator_ValidateRecord_CollectsSOUR(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	ind := &gedcom.Individual{
		XRef: "@I1@",
		SourceCitations: []*gedcom.SourceCitation{
			{SourceXRef: "@S1@"},
			{SourceXRef: "@S2@"},
		},
	}
	record := &gedcom.Record{
		XRef:   "@I1@",
		Type:   gedcom.RecordTypeIndividual,
		Entity: ind,
	}

	sv.ValidateRecord(record)

	// Verify SOUR references were collected
	if len(sv.usedXRefs["@S1@"]) != 1 {
		t.Errorf("Expected 1 usage for @S1@, got %d", len(sv.usedXRefs["@S1@"]))
	}

	usage := sv.usedXRefs["@S1@"][0]
	if usage.Context != "SOUR" {
		t.Errorf("Expected Context SOUR, got %s", usage.Context)
	}
}

func TestStreamingValidator_ValidateRecord_CollectsHUSBWIFECHIL(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	fam := &gedcom.Family{
		XRef:     "@F1@",
		Husband:  "@I1@",
		Wife:     "@I2@",
		Children: []string{"@I3@", "@I4@"},
	}
	record := &gedcom.Record{
		XRef:   "@F1@",
		Type:   gedcom.RecordTypeFamily,
		Entity: fam,
	}

	sv.ValidateRecord(record)

	// Verify HUSB reference
	if len(sv.usedXRefs["@I1@"]) != 1 {
		t.Errorf("Expected 1 usage for @I1@, got %d", len(sv.usedXRefs["@I1@"]))
	}
	if sv.usedXRefs["@I1@"][0].Context != "HUSB" {
		t.Errorf("Expected Context HUSB, got %s", sv.usedXRefs["@I1@"][0].Context)
	}

	// Verify WIFE reference
	if len(sv.usedXRefs["@I2@"]) != 1 {
		t.Errorf("Expected 1 usage for @I2@, got %d", len(sv.usedXRefs["@I2@"]))
	}
	if sv.usedXRefs["@I2@"][0].Context != "WIFE" {
		t.Errorf("Expected Context WIFE, got %s", sv.usedXRefs["@I2@"][0].Context)
	}

	// Verify CHIL references
	if len(sv.usedXRefs["@I3@"]) != 1 {
		t.Errorf("Expected 1 usage for @I3@, got %d", len(sv.usedXRefs["@I3@"]))
	}
	if sv.usedXRefs["@I3@"][0].Context != "CHIL" {
		t.Errorf("Expected Context CHIL, got %s", sv.usedXRefs["@I3@"][0].Context)
	}
}

func TestStreamingValidator_ValidateRecord_DeathBeforeBirth(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	// Create individual with death before birth
	ind := &gedcom.Individual{
		XRef: "@I1@",
		Events: []*gedcom.Event{
			{Type: gedcom.EventBirth, Date: "1900", ParsedDate: &gedcom.Date{Year: 1900, Original: "1900"}},
			{Type: gedcom.EventDeath, Date: "1850", ParsedDate: &gedcom.Date{Year: 1850, Original: "1850"}},
		},
	}
	record := &gedcom.Record{
		XRef:   "@I1@",
		Type:   gedcom.RecordTypeIndividual,
		Entity: ind,
	}

	issues := sv.ValidateRecord(record)

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue for death before birth, got %d", len(issues))
	}

	if issues[0].Code != CodeDeathBeforeBirth {
		t.Errorf("Expected code %s, got %s", CodeDeathBeforeBirth, issues[0].Code)
	}
	if issues[0].Severity != SeverityError {
		t.Errorf("Expected severity ERROR, got %s", issues[0].Severity)
	}
}

func TestStreamingValidator_ValidateRecord_SkipsEmptyXRefs(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	ind := &gedcom.Individual{
		XRef: "@I1@",
		ChildInFamilies: []gedcom.FamilyLink{
			{FamilyXRef: ""},     // Empty - should be skipped
			{FamilyXRef: "@F1@"}, // Valid
		},
		SpouseInFamilies: []string{"", "@F2@"}, // First empty, second valid
		SourceCitations: []*gedcom.SourceCitation{
			nil,                  // Nil - should be skipped
			{SourceXRef: ""},     // Empty - should be skipped
			{SourceXRef: "@S1@"}, // Valid
		},
	}
	record := &gedcom.Record{
		XRef:   "@I1@",
		Type:   gedcom.RecordTypeIndividual,
		Entity: ind,
	}

	sv.ValidateRecord(record)

	// Should only have 3 used XRefs: @F1@, @F2@, @S1@
	if len(sv.usedXRefs) != 3 {
		t.Errorf("Expected 3 used XRefs, got %d", len(sv.usedXRefs))
	}
}

func TestStreamingValidator_Finalize_NoOrphanedReferences(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	// Create valid linked structure
	ind1 := &gedcom.Individual{XRef: "@I1@", SpouseInFamilies: []string{"@F1@"}}
	ind2 := &gedcom.Individual{XRef: "@I2@", SpouseInFamilies: []string{"@F1@"}}
	ind3 := &gedcom.Individual{XRef: "@I3@", ChildInFamilies: []gedcom.FamilyLink{{FamilyXRef: "@F1@"}}}
	fam := &gedcom.Family{XRef: "@F1@", Husband: "@I1@", Wife: "@I2@", Children: []string{"@I3@"}}

	sv.ValidateRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: ind1})
	sv.ValidateRecord(&gedcom.Record{XRef: "@I2@", Type: gedcom.RecordTypeIndividual, Entity: ind2})
	sv.ValidateRecord(&gedcom.Record{XRef: "@I3@", Type: gedcom.RecordTypeIndividual, Entity: ind3})
	sv.ValidateRecord(&gedcom.Record{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: fam})

	issues := sv.Finalize()

	if len(issues) != 0 {
		t.Errorf("Expected 0 issues for valid references, got %d", len(issues))
		for _, issue := range issues {
			t.Logf("  Issue: %s", issue.String())
		}
	}
}

func TestStreamingValidator_Finalize_OrphanedFAMC(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	ind := &gedcom.Individual{
		XRef:            "@I1@",
		ChildInFamilies: []gedcom.FamilyLink{{FamilyXRef: "@F999@"}}, // Non-existent family
	}
	sv.ValidateRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: ind})

	issues := sv.Finalize()

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]
	if issue.Code != CodeOrphanedFAMC {
		t.Errorf("Expected code %s, got %s", CodeOrphanedFAMC, issue.Code)
	}
	if issue.RecordXRef != "@I1@" {
		t.Errorf("Expected RecordXRef @I1@, got %s", issue.RecordXRef)
	}
	if issue.RelatedXRef != "@F999@" {
		t.Errorf("Expected RelatedXRef @F999@, got %s", issue.RelatedXRef)
	}
}

func TestStreamingValidator_Finalize_OrphanedFAMS(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	ind := &gedcom.Individual{
		XRef:             "@I1@",
		SpouseInFamilies: []string{"@F999@"}, // Non-existent family
	}
	sv.ValidateRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: ind})

	issues := sv.Finalize()

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(issues))
	}

	if issues[0].Code != CodeOrphanedFAMS {
		t.Errorf("Expected code %s, got %s", CodeOrphanedFAMS, issues[0].Code)
	}
}

func TestStreamingValidator_Finalize_OrphanedHUSB(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	fam := &gedcom.Family{
		XRef:    "@F1@",
		Husband: "@I999@", // Non-existent individual
	}
	sv.ValidateRecord(&gedcom.Record{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: fam})

	issues := sv.Finalize()

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(issues))
	}

	if issues[0].Code != CodeOrphanedHUSB {
		t.Errorf("Expected code %s, got %s", CodeOrphanedHUSB, issues[0].Code)
	}
}

func TestStreamingValidator_Finalize_OrphanedWIFE(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	fam := &gedcom.Family{
		XRef: "@F1@",
		Wife: "@I999@", // Non-existent individual
	}
	sv.ValidateRecord(&gedcom.Record{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: fam})

	issues := sv.Finalize()

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(issues))
	}

	if issues[0].Code != CodeOrphanedWIFE {
		t.Errorf("Expected code %s, got %s", CodeOrphanedWIFE, issues[0].Code)
	}
}

func TestStreamingValidator_Finalize_OrphanedCHIL(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	fam := &gedcom.Family{
		XRef:     "@F1@",
		Children: []string{"@I999@"}, // Non-existent individual
	}
	sv.ValidateRecord(&gedcom.Record{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: fam})

	issues := sv.Finalize()

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(issues))
	}

	if issues[0].Code != CodeOrphanedCHIL {
		t.Errorf("Expected code %s, got %s", CodeOrphanedCHIL, issues[0].Code)
	}
}

func TestStreamingValidator_Finalize_OrphanedSOUR(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	ind := &gedcom.Individual{
		XRef:            "@I1@",
		SourceCitations: []*gedcom.SourceCitation{{SourceXRef: "@S999@"}}, // Non-existent source
	}
	sv.ValidateRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: ind})

	issues := sv.Finalize()

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(issues))
	}

	if issues[0].Code != CodeOrphanedSOUR {
		t.Errorf("Expected code %s, got %s", CodeOrphanedSOUR, issues[0].Code)
	}
}

func TestStreamingValidator_Finalize_MultipleOrphanedReferences(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	// Create records with multiple orphaned references
	ind := &gedcom.Individual{
		XRef:             "@I1@",
		ChildInFamilies:  []gedcom.FamilyLink{{FamilyXRef: "@F999@"}},
		SpouseInFamilies: []string{"@F998@"},
		SourceCitations:  []*gedcom.SourceCitation{{SourceXRef: "@S999@"}},
	}
	fam := &gedcom.Family{
		XRef:     "@F1@",
		Husband:  "@I999@",
		Wife:     "@I998@",
		Children: []string{"@I997@"},
	}

	sv.ValidateRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: ind})
	sv.ValidateRecord(&gedcom.Record{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: fam})

	issues := sv.Finalize()

	// Should have 6 orphaned references:
	// FAMC @F999@, FAMS @F998@, SOUR @S999@, HUSB @I999@, WIFE @I998@, CHIL @I997@
	if len(issues) != 6 {
		t.Errorf("Expected 6 issues, got %d", len(issues))
		for _, issue := range issues {
			t.Logf("  Issue: %s", issue.String())
		}
	}

	// Count by code
	counts := make(map[string]int)
	for _, issue := range issues {
		counts[issue.Code]++
	}

	expectedCounts := map[string]int{
		CodeOrphanedFAMC: 1,
		CodeOrphanedFAMS: 1,
		CodeOrphanedSOUR: 1,
		CodeOrphanedHUSB: 1,
		CodeOrphanedWIFE: 1,
		CodeOrphanedCHIL: 1,
	}

	for code, expected := range expectedCounts {
		if counts[code] != expected {
			t.Errorf("Expected %d %s issues, got %d", expected, code, counts[code])
		}
	}
}

func TestStreamingValidator_Reset(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	// Add some data
	ind := &gedcom.Individual{
		XRef:             "@I1@",
		SpouseInFamilies: []string{"@F1@"},
	}
	sv.ValidateRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: ind})

	// Verify data was added
	if len(sv.seenXRefs) == 0 {
		t.Error("seenXRefs should have data before reset")
	}
	if len(sv.usedXRefs) == 0 {
		t.Error("usedXRefs should have data before reset")
	}

	// Reset
	sv.Reset()

	// Verify data was cleared
	if len(sv.seenXRefs) != 0 {
		t.Error("seenXRefs should be empty after reset")
	}
	if len(sv.usedXRefs) != 0 {
		t.Error("usedXRefs should be empty after reset")
	}
	if len(sv.xrefTypes) != 0 {
		t.Error("xrefTypes should be empty after reset")
	}
}

func TestStreamingValidator_Reset_CanBeReused(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	// First file: has orphaned reference
	ind1 := &gedcom.Individual{
		XRef:             "@I1@",
		SpouseInFamilies: []string{"@F999@"},
	}
	sv.ValidateRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: ind1})
	issues1 := sv.Finalize()

	if len(issues1) != 1 {
		t.Errorf("First file: expected 1 issue, got %d", len(issues1))
	}

	// Reset for second file
	sv.Reset()

	// Second file: valid references
	ind2 := &gedcom.Individual{
		XRef:             "@I1@",
		SpouseInFamilies: []string{"@F1@"},
	}
	fam := &gedcom.Family{XRef: "@F1@", Husband: "@I1@"}

	sv.ValidateRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: ind2})
	sv.ValidateRecord(&gedcom.Record{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: fam})
	issues2 := sv.Finalize()

	if len(issues2) != 0 {
		t.Errorf("Second file: expected 0 issues, got %d", len(issues2))
		for _, issue := range issues2 {
			t.Logf("  Issue: %s", issue.String())
		}
	}
}

func TestStreamingValidator_SeenXRefCount(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	if sv.SeenXRefCount() != 0 {
		t.Error("SeenXRefCount should be 0 initially")
	}

	// Add records
	sv.ValidateRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: "@I1@"}})
	sv.ValidateRecord(&gedcom.Record{XRef: "@I2@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: "@I2@"}})
	sv.ValidateRecord(&gedcom.Record{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: &gedcom.Family{XRef: "@F1@"}})

	if sv.SeenXRefCount() != 3 {
		t.Errorf("SeenXRefCount should be 3, got %d", sv.SeenXRefCount())
	}
}

func TestStreamingValidator_UsedXRefCount(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	if sv.UsedXRefCount() != 0 {
		t.Error("UsedXRefCount should be 0 initially")
	}

	// Add individual with references
	ind := &gedcom.Individual{
		XRef:             "@I1@",
		SpouseInFamilies: []string{"@F1@", "@F2@"},
		SourceCitations:  []*gedcom.SourceCitation{{SourceXRef: "@S1@"}},
	}
	sv.ValidateRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: ind})

	// Should have 3 unique used XRefs: @F1@, @F2@, @S1@
	if sv.UsedXRefCount() != 3 {
		t.Errorf("UsedXRefCount should be 3, got %d", sv.UsedXRefCount())
	}
}

func TestStreamingValidator_MemoryUsage(t *testing.T) {
	// Test that memory usage is proportional to unique XRefs, not record count
	sv := NewStreamingValidator(StreamingOptions{})

	// Create 1000 individuals, all referencing the same family
	for i := 0; i < 1000; i++ {
		xref := fmt.Sprintf("@I%d@", i)
		ind := &gedcom.Individual{
			XRef:             xref,
			SpouseInFamilies: []string{"@F1@"}, // All reference same family
		}
		sv.ValidateRecord(&gedcom.Record{XRef: xref, Type: gedcom.RecordTypeIndividual, Entity: ind})
	}

	// Should have 1000 seen XRefs (one per individual)
	if sv.SeenXRefCount() != 1000 {
		t.Errorf("SeenXRefCount should be 1000, got %d", sv.SeenXRefCount())
	}

	// But only 1 used XRef (@F1@) - memory is O(unique XRefs), not O(records)
	if sv.UsedXRefCount() != 1 {
		t.Errorf("UsedXRefCount should be 1, got %d", sv.UsedXRefCount())
	}

	// The @F1@ usage should have 1000 entries though
	if len(sv.usedXRefs["@F1@"]) != 1000 {
		t.Errorf("@F1@ should have 1000 usages, got %d", len(sv.usedXRefs["@F1@"]))
	}
}

func TestStreamingValidator_Strictness_Relaxed(t *testing.T) {
	// Note: Currently StreamingValidator only produces ERROR severity issues
	// from date logic validation and cross-reference checking.
	// This test verifies the strictness filtering works correctly.
	sv := NewStreamingValidator(StreamingOptions{Strictness: StrictnessRelaxed})

	// Death before birth produces ERROR
	ind := &gedcom.Individual{
		XRef: "@I1@",
		Events: []*gedcom.Event{
			{Type: gedcom.EventBirth, ParsedDate: &gedcom.Date{Year: 1900, Original: "1900"}},
			{Type: gedcom.EventDeath, ParsedDate: &gedcom.Date{Year: 1850, Original: "1850"}},
		},
	}
	issues := sv.ValidateRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: ind})

	// ERROR issues should be included in relaxed mode
	if len(issues) != 1 {
		t.Errorf("Expected 1 ERROR issue in relaxed mode, got %d", len(issues))
	}
}

func TestStreamingValidator_Strictness_FiltersWarnings(t *testing.T) {
	// Test that strictness filtering is applied to Finalize() results
	sv := NewStreamingValidator(StreamingOptions{Strictness: StrictnessRelaxed})

	// Add orphaned reference (produces ERROR)
	ind := &gedcom.Individual{XRef: "@I1@", SpouseInFamilies: []string{"@F999@"}}
	sv.ValidateRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: ind})

	issues := sv.Finalize()

	// Should still get the ERROR-level orphaned reference issue
	if len(issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(issues))
	}
}

func TestStreamingValidator_EquivalenceToBatchValidator(t *testing.T) {
	// Test that streaming validation produces equivalent results to batch validation
	// for the same document

	// Create a test document
	doc := newTestDocument()

	// Valid individuals and families
	ind1 := &gedcom.Individual{XRef: "@I1@", SpouseInFamilies: []string{"@F1@"}}
	ind2 := &gedcom.Individual{XRef: "@I2@", SpouseInFamilies: []string{"@F1@"}}
	ind3 := &gedcom.Individual{XRef: "@I3@", ChildInFamilies: []gedcom.FamilyLink{{FamilyXRef: "@F1@"}}}
	fam := &gedcom.Family{XRef: "@F1@", Husband: "@I1@", Wife: "@I2@", Children: []string{"@I3@"}}
	src := &gedcom.Source{XRef: "@S1@"}

	// Add valid source citation
	ind1.SourceCitations = []*gedcom.SourceCitation{{SourceXRef: "@S1@"}}

	// Add orphaned references
	ind1.ChildInFamilies = []gedcom.FamilyLink{{FamilyXRef: "@F999@"}} // Orphaned FAMC

	addIndividual(doc, ind1)
	addIndividual(doc, ind2)
	addIndividual(doc, ind3)
	addFamily(doc, fam)
	addSource(doc, src)

	// Batch validation
	batchV := NewReferenceValidator()
	batchIssues := batchV.Validate(doc)

	// Streaming validation
	streamV := NewStreamingValidator(StreamingOptions{})
	for _, record := range doc.Records {
		streamV.ValidateRecord(record)
	}
	streamIssues := streamV.Finalize()

	// Compare results
	// Both should find the same orphaned FAMC reference
	if len(batchIssues) != len(streamIssues) {
		t.Errorf("Issue count mismatch: batch=%d, streaming=%d", len(batchIssues), len(streamIssues))
		t.Log("Batch issues:")
		for _, issue := range batchIssues {
			t.Logf("  %s", issue.String())
		}
		t.Log("Streaming issues:")
		for _, issue := range streamIssues {
			t.Logf("  %s", issue.String())
		}
		return
	}

	// Sort issues by code and RecordXRef for comparison
	sortIssues := func(issues []Issue) {
		sort.Slice(issues, func(i, j int) bool {
			if issues[i].Code != issues[j].Code {
				return issues[i].Code < issues[j].Code
			}
			return issues[i].RecordXRef < issues[j].RecordXRef
		})
	}

	sortIssues(batchIssues)
	sortIssues(streamIssues)

	for i := range batchIssues {
		if batchIssues[i].Code != streamIssues[i].Code {
			t.Errorf("Code mismatch at %d: batch=%s, streaming=%s", i, batchIssues[i].Code, streamIssues[i].Code)
		}
		if batchIssues[i].RecordXRef != streamIssues[i].RecordXRef {
			t.Errorf("RecordXRef mismatch at %d: batch=%s, streaming=%s", i, batchIssues[i].RecordXRef, streamIssues[i].RecordXRef)
		}
		if batchIssues[i].RelatedXRef != streamIssues[i].RelatedXRef {
			t.Errorf("RelatedXRef mismatch at %d: batch=%s, streaming=%s", i, batchIssues[i].RelatedXRef, streamIssues[i].RelatedXRef)
		}
	}
}

func TestStreamingValidator_EmptyFamilyReferences(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	// Family with empty references - should not create issues
	fam := &gedcom.Family{
		XRef:     "@F1@",
		Husband:  "",           // Empty
		Wife:     "",           // Empty
		Children: []string{""}, // Empty child reference
	}
	sv.ValidateRecord(&gedcom.Record{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: fam})

	issues := sv.Finalize()

	if len(issues) != 0 {
		t.Errorf("Expected 0 issues for empty references, got %d", len(issues))
		for _, issue := range issues {
			t.Logf("  Issue: %s", issue.String())
		}
	}
}

func TestStreamingValidator_NoteReferences(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	ind := &gedcom.Individual{
		XRef:  "@I1@",
		Notes: []string{"@N1@", "@N999@"}, // Second is orphaned
	}
	sv.ValidateRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: ind})
	sv.ValidateRecord(&gedcom.Record{XRef: "@N1@", Type: gedcom.RecordTypeNote, Entity: &gedcom.Note{XRef: "@N1@"}})

	issues := sv.Finalize()

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue for orphaned NOTE, got %d", len(issues))
	}

	if issues[0].Details["reference_type"] != "NOTE" {
		t.Errorf("Expected reference_type NOTE, got %s", issues[0].Details["reference_type"])
	}
}

func TestStreamingValidator_AssociationReferences(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	ind := &gedcom.Individual{
		XRef: "@I1@",
		Associations: []*gedcom.Association{
			{IndividualXRef: "@I2@"},   // Valid
			{IndividualXRef: "@I999@"}, // Orphaned
		},
	}
	sv.ValidateRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: ind})
	sv.ValidateRecord(&gedcom.Record{XRef: "@I2@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: "@I2@"}})

	issues := sv.Finalize()

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue for orphaned ASSO, got %d", len(issues))
	}

	if issues[0].Details["reference_type"] != "ASSO" {
		t.Errorf("Expected reference_type ASSO, got %s", issues[0].Details["reference_type"])
	}
}

func TestStreamingValidator_SourceRepositoryReference(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	src := &gedcom.Source{
		XRef:          "@S1@",
		RepositoryRef: "@R999@", // Orphaned repository reference
	}
	sv.ValidateRecord(&gedcom.Record{XRef: "@S1@", Type: gedcom.RecordTypeSource, Entity: src})

	issues := sv.Finalize()

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue for orphaned REPO, got %d", len(issues))
	}

	if issues[0].Details["reference_type"] != "REPO" {
		t.Errorf("Expected reference_type REPO, got %s", issues[0].Details["reference_type"])
	}
}

func TestStreamingValidator_FamilyNoteReferences(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	fam := &gedcom.Family{
		XRef:  "@F1@",
		Notes: []string{"@N999@"}, // Orphaned note
	}
	sv.ValidateRecord(&gedcom.Record{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: fam})

	issues := sv.Finalize()

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue for orphaned NOTE in family, got %d", len(issues))
	}

	if issues[0].RecordXRef != "@F1@" {
		t.Errorf("Expected RecordXRef @F1@, got %s", issues[0].RecordXRef)
	}
}

func TestStreamingValidator_FamilySourceReferences(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	fam := &gedcom.Family{
		XRef:            "@F1@",
		SourceCitations: []*gedcom.SourceCitation{{SourceXRef: "@S999@"}}, // Orphaned source
	}
	sv.ValidateRecord(&gedcom.Record{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: fam})

	issues := sv.Finalize()

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue for orphaned SOUR in family, got %d", len(issues))
	}

	if issues[0].Code != CodeOrphanedSOUR {
		t.Errorf("Expected code %s, got %s", CodeOrphanedSOUR, issues[0].Code)
	}
}

func TestStreamingValidator_SourceNoteReferences(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	src := &gedcom.Source{
		XRef:  "@S1@",
		Notes: []string{"@N999@"}, // Orphaned note
	}
	sv.ValidateRecord(&gedcom.Record{XRef: "@S1@", Type: gedcom.RecordTypeSource, Entity: src})

	issues := sv.Finalize()

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue for orphaned NOTE in source, got %d", len(issues))
	}

	if issues[0].RecordXRef != "@S1@" {
		t.Errorf("Expected RecordXRef @S1@, got %s", issues[0].RecordXRef)
	}
}

func TestStreamingValidator_RecordWithNoXRef(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	// Record with empty XRef (shouldn't crash)
	record := &gedcom.Record{
		XRef:   "",
		Type:   gedcom.RecordTypeIndividual,
		Entity: &gedcom.Individual{XRef: ""},
	}

	// Should not panic
	issues := sv.ValidateRecord(record)

	if len(issues) != 0 {
		t.Errorf("Expected 0 issues for record with no XRef, got %d", len(issues))
	}

	// Empty XRef should not be added to seenXRefs
	if _, exists := sv.seenXRefs[""]; exists {
		t.Error("Empty XRef should not be added to seenXRefs")
	}
}

func TestStreamingValidator_NilEntity(t *testing.T) {
	sv := NewStreamingValidator(StreamingOptions{})

	// Record with nil Entity (shouldn't crash)
	record := &gedcom.Record{
		XRef:   "@I1@",
		Type:   gedcom.RecordTypeIndividual,
		Entity: nil,
	}

	// Should not panic
	issues := sv.ValidateRecord(record)

	if len(issues) != 0 {
		t.Errorf("Expected 0 issues for record with nil Entity, got %d", len(issues))
	}

	// XRef should still be registered even with nil Entity
	if _, exists := sv.seenXRefs["@I1@"]; !exists {
		t.Error("XRef should be registered even with nil Entity")
	}
}

func TestStreamingValidator_filterByStrictness_AllBranches(t *testing.T) {
	// Create test issues with different severities
	errorIssue := NewIssue(SeverityError, "TEST_ERROR", "Error issue", "@I1@")
	warningIssue := NewIssue(SeverityWarning, "TEST_WARNING", "Warning issue", "@I1@")
	infoIssue := NewIssue(SeverityInfo, "TEST_INFO", "Info issue", "@I1@")
	allIssues := []Issue{errorIssue, warningIssue, infoIssue}

	tests := []struct {
		name          string
		strictness    Strictness
		inputIssues   []Issue
		expectedCount int
	}{
		{
			name:          "relaxed with all severities",
			strictness:    StrictnessRelaxed,
			inputIssues:   allIssues,
			expectedCount: 1, // Only errors
		},
		{
			name:          "normal with all severities",
			strictness:    StrictnessNormal,
			inputIssues:   allIssues,
			expectedCount: 2, // Errors and warnings
		},
		{
			name:          "strict with all severities",
			strictness:    StrictnessStrict,
			inputIssues:   allIssues,
			expectedCount: 3, // All issues
		},
		{
			name:          "empty issues",
			strictness:    StrictnessNormal,
			inputIssues:   []Issue{},
			expectedCount: 0, // Empty returns empty
		},
		{
			name:          "default (zero value) strictness",
			strictness:    Strictness(99), // Invalid value triggers default
			inputIssues:   allIssues,
			expectedCount: 2, // Defaults to normal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sv := NewStreamingValidator(StreamingOptions{Strictness: tt.strictness})
			result := sv.filterByStrictness(tt.inputIssues)

			if len(result) != tt.expectedCount {
				t.Errorf("Expected %d issues, got %d", tt.expectedCount, len(result))
			}
		})
	}
}

func TestStreamingValidator_Strictness_Normal(t *testing.T) {
	// Test that Normal strictness filters out Info but keeps Error and Warning
	sv := NewStreamingValidator(StreamingOptions{Strictness: StrictnessNormal})

	// Add orphaned reference (produces ERROR)
	ind := &gedcom.Individual{XRef: "@I1@", SpouseInFamilies: []string{"@F999@"}}
	sv.ValidateRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: ind})

	issues := sv.Finalize()

	// Should get the ERROR-level orphaned reference issue
	if len(issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(issues))
	}
	if len(issues) > 0 && issues[0].Severity != SeverityError {
		t.Errorf("Expected ERROR severity, got %s", issues[0].Severity)
	}
}

func TestStreamingValidator_Strictness_Strict(t *testing.T) {
	// Test that Strict strictness includes all issues
	sv := NewStreamingValidator(StreamingOptions{Strictness: StrictnessStrict})

	// Add orphaned reference (produces ERROR)
	ind := &gedcom.Individual{XRef: "@I1@", SpouseInFamilies: []string{"@F999@"}}
	sv.ValidateRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: ind})

	issues := sv.Finalize()

	// Should get the ERROR-level orphaned reference issue
	if len(issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(issues))
	}
}
