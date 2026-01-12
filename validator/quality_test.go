package validator

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/gedcom"
)

// Helper to create an individual with optional birth date, sources, and places
func makeIndividualWithDetails(xref string, birthYear int, hasSource, hasPlace, hasName bool) *gedcom.Individual {
	ind := &gedcom.Individual{XRef: xref}

	if birthYear > 0 {
		ind.Events = append(ind.Events, &gedcom.Event{
			Type:       gedcom.EventBirth,
			ParsedDate: makeYearDate(birthYear),
		})
	}

	if hasSource {
		ind.SourceCitations = []*gedcom.SourceCitation{{SourceXRef: "@S1@"}}
	}

	if hasPlace {
		if len(ind.Events) == 0 {
			ind.Events = append(ind.Events, &gedcom.Event{Type: gedcom.EventBirth})
		}
		ind.Events[0].Place = "Test Place"
	}

	if hasName {
		ind.Names = []*gedcom.PersonalName{{Full: "John /Doe/"}}
	}

	return ind
}

// Helper to create a document with sources
func makeDocumentWithSources(individuals []*gedcom.Individual, families []*gedcom.Family, sources []*gedcom.Source) *gedcom.Document {
	doc := makeDocument(individuals, families)

	for _, src := range sources {
		rec := &gedcom.Record{XRef: src.XRef, Type: gedcom.RecordTypeSource, Entity: src}
		doc.Records = append(doc.Records, rec)
		doc.XRefMap[src.XRef] = rec
	}

	return doc
}

func TestNewQualityAnalyzer(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		a := NewQualityAnalyzer()

		if a.dateLogic == nil {
			t.Error("dateLogic should not be nil")
		}
		if a.references == nil {
			t.Error("references should not be nil")
		}
		if a.duplicates == nil {
			t.Error("duplicates should not be nil")
		}
	})

	t.Run("with date logic config", func(t *testing.T) {
		config := &DateLogicConfig{MaxReasonableAge: 100}
		a := NewQualityAnalyzer(WithDateLogicConfig(config))

		if a.dateLogic.config.MaxReasonableAge != 100 {
			t.Errorf("MaxReasonableAge = %d, want 100", a.dateLogic.config.MaxReasonableAge)
		}
	})

	t.Run("with duplicate config", func(t *testing.T) {
		config := &DuplicateConfig{MinConfidence: 0.9}
		a := NewQualityAnalyzer(WithDuplicateConfig(config))

		if a.duplicates.config.MinConfidence != 0.9 {
			t.Errorf("MinConfidence = %f, want 0.9", a.duplicates.config.MinConfidence)
		}
	})

	t.Run("with multiple options", func(t *testing.T) {
		dateConfig := &DateLogicConfig{MaxReasonableAge: 110}
		dupConfig := &DuplicateConfig{MinConfidence: 0.8}

		a := NewQualityAnalyzer(
			WithDateLogicConfig(dateConfig),
			WithDuplicateConfig(dupConfig),
		)

		if a.dateLogic.config.MaxReasonableAge != 110 {
			t.Errorf("MaxReasonableAge = %d, want 110", a.dateLogic.config.MaxReasonableAge)
		}
		if a.duplicates.config.MinConfidence != 0.8 {
			t.Errorf("MinConfidence = %f, want 0.8", a.duplicates.config.MinConfidence)
		}
	})
}

func TestQualityAnalyzer_Analyze_NilDocument(t *testing.T) {
	a := NewQualityAnalyzer()
	report := a.Analyze(nil)

	if report == nil {
		t.Fatal("report should not be nil for nil document")
	}
	if report.TotalIndividuals != 0 {
		t.Errorf("TotalIndividuals = %d, want 0", report.TotalIndividuals)
	}
	if report.TotalIssues != 0 {
		t.Errorf("TotalIssues = %d, want 0", report.TotalIssues)
	}
}

func TestQualityAnalyzer_Analyze_EmptyDocument(t *testing.T) {
	a := NewQualityAnalyzer()
	doc := &gedcom.Document{
		Records: []*gedcom.Record{},
		XRefMap: make(map[string]*gedcom.Record),
	}

	report := a.Analyze(doc)

	if report.TotalIndividuals != 0 {
		t.Errorf("TotalIndividuals = %d, want 0", report.TotalIndividuals)
	}
	if report.TotalFamilies != 0 {
		t.Errorf("TotalFamilies = %d, want 0", report.TotalFamilies)
	}
	if report.TotalSources != 0 {
		t.Errorf("TotalSources = %d, want 0", report.TotalSources)
	}
}

func TestQualityAnalyzer_Analyze_RecordCounts(t *testing.T) {
	a := NewQualityAnalyzer()

	ind1 := makeIndividualWithDetails("@I1@", 1950, true, true, true)
	ind2 := makeIndividualWithDetails("@I2@", 1955, false, false, true)
	fam := &gedcom.Family{XRef: "@F1@", Husband: "@I1@", Wife: "@I2@"}
	src := &gedcom.Source{XRef: "@S1@", Title: "Test Source"}

	doc := makeDocumentWithSources([]*gedcom.Individual{ind1, ind2}, []*gedcom.Family{fam}, []*gedcom.Source{src})

	report := a.Analyze(doc)

	if report.TotalIndividuals != 2 {
		t.Errorf("TotalIndividuals = %d, want 2", report.TotalIndividuals)
	}
	if report.TotalFamilies != 1 {
		t.Errorf("TotalFamilies = %d, want 1", report.TotalFamilies)
	}
	if report.TotalSources != 1 {
		t.Errorf("TotalSources = %d, want 1", report.TotalSources)
	}
}

func TestQualityAnalyzer_Analyze_CompletenessMetrics(t *testing.T) {
	a := NewQualityAnalyzer()

	// Create 4 individuals with varying completeness
	ind1 := makeIndividualWithDetails("@I1@", 1950, true, true, true)   // Has birth, source, place, name
	ind2 := makeIndividualWithDetails("@I2@", 1955, false, false, true) // Has birth, name only
	ind3 := makeIndividualWithDetails("@I3@", 0, true, false, true)     // Has source, name only
	ind4 := makeIndividualWithDetails("@I4@", 0, false, false, false)   // Has nothing

	src := &gedcom.Source{XRef: "@S1@", Title: "Test Source"}
	doc := makeDocumentWithSources([]*gedcom.Individual{ind1, ind2, ind3, ind4}, nil, []*gedcom.Source{src})

	report := a.Analyze(doc)

	if report.IndividualsWithBirthDate != 2 {
		t.Errorf("IndividualsWithBirthDate = %d, want 2", report.IndividualsWithBirthDate)
	}
	if report.IndividualsWithSources != 2 {
		t.Errorf("IndividualsWithSources = %d, want 2", report.IndividualsWithSources)
	}
	if report.IndividualsWithPlaces != 1 {
		t.Errorf("IndividualsWithPlaces = %d, want 1", report.IndividualsWithPlaces)
	}

	// Check percentages
	if report.BirthDateCoverage != 0.5 {
		t.Errorf("BirthDateCoverage = %f, want 0.5", report.BirthDateCoverage)
	}
	if report.SourceCoverage != 0.5 {
		t.Errorf("SourceCoverage = %f, want 0.5", report.SourceCoverage)
	}
}

func TestQualityAnalyzer_Analyze_CompletenessIssues(t *testing.T) {
	a := NewQualityAnalyzer()

	// Individual without birth date, sources, or name
	ind := &gedcom.Individual{XRef: "@I1@"}
	doc := makeDocument([]*gedcom.Individual{ind}, nil)

	report := a.Analyze(doc)

	// Should have MISSING_BIRTH_DATE, NO_SOURCES, and MISSING_NAME issues
	expectedCodes := map[string]bool{
		CodeMissingBirthDate: false,
		CodeNoSources:        false,
		CodeMissingName:      false,
	}

	for _, issue := range report.CompletenessIssues {
		if _, ok := expectedCodes[issue.Code]; ok {
			expectedCodes[issue.Code] = true
		}
	}

	for code, found := range expectedCodes {
		if !found {
			t.Errorf("expected %s issue, not found", code)
		}
	}
}

func TestQualityAnalyzer_Analyze_DateLogicValidatorInvoked(t *testing.T) {
	a := NewQualityAnalyzer()

	// Create individual with death before birth
	ind := makeIndividual("@I1@", 1950, 1940)
	doc := makeDocument([]*gedcom.Individual{ind}, nil)

	report := a.Analyze(doc)

	// Should have date logic issues
	if len(report.DateLogicIssues) == 0 {
		t.Error("expected date logic issues, got none")
	}

	foundDeathBeforeBirth := false
	for _, issue := range report.DateLogicIssues {
		if issue.Code == CodeDeathBeforeBirth {
			foundDeathBeforeBirth = true
			break
		}
	}
	if !foundDeathBeforeBirth {
		t.Error("expected DEATH_BEFORE_BIRTH issue")
	}
}

func TestQualityAnalyzer_Analyze_ReferenceValidatorInvoked(t *testing.T) {
	a := NewQualityAnalyzer()

	// Create individual with orphaned family reference
	ind := &gedcom.Individual{
		XRef:            "@I1@",
		Names:           []*gedcom.PersonalName{{Full: "Test /Person/"}},
		ChildInFamilies: []gedcom.FamilyLink{{FamilyXRef: "@F999@"}}, // Non-existent family
	}
	ind.Events = []*gedcom.Event{{Type: gedcom.EventBirth, ParsedDate: makeYearDate(1950)}}

	doc := makeDocument([]*gedcom.Individual{ind}, nil)

	report := a.Analyze(doc)

	// Should have reference issues
	if len(report.ReferenceIssues) == 0 {
		t.Error("expected reference issues, got none")
	}

	foundOrphaned := false
	for _, issue := range report.ReferenceIssues {
		if issue.Code == CodeOrphanedFAMC {
			foundOrphaned = true
			break
		}
	}
	if !foundOrphaned {
		t.Error("expected ORPHANED_FAMC issue")
	}
}

func TestQualityAnalyzer_Analyze_DuplicateDetectorInvoked(t *testing.T) {
	a := NewQualityAnalyzer()

	// Create potential duplicates
	ind1 := &gedcom.Individual{
		XRef:  "@I1@",
		Names: []*gedcom.PersonalName{{Given: "John", Surname: "Doe"}},
		Sex:   "M",
		Events: []*gedcom.Event{{
			Type:       gedcom.EventBirth,
			ParsedDate: makeYearDate(1950),
		}},
	}
	ind2 := &gedcom.Individual{
		XRef:  "@I2@",
		Names: []*gedcom.PersonalName{{Given: "John", Surname: "Doe"}},
		Sex:   "M",
		Events: []*gedcom.Event{{
			Type:       gedcom.EventBirth,
			ParsedDate: makeYearDate(1950),
		}},
	}

	doc := makeDocument([]*gedcom.Individual{ind1, ind2}, nil)

	report := a.Analyze(doc)

	// Should have duplicate issues
	if len(report.DuplicateIssues) == 0 {
		t.Error("expected duplicate issues, got none")
	}

	foundDuplicate := false
	for _, issue := range report.DuplicateIssues {
		if issue.Code == CodePotentialDuplicate {
			foundDuplicate = true
			break
		}
	}
	if !foundDuplicate {
		t.Error("expected POTENTIAL_DUPLICATE issue")
	}
}

func TestQualityAnalyzer_Analyze_IssueCategorization(t *testing.T) {
	a := NewQualityAnalyzer()

	// Create a document with issues of different severities
	// Error: death before birth
	ind1 := makeIndividual("@I1@", 1950, 1940)
	ind1.Names = []*gedcom.PersonalName{{Full: "Test /One/"}}

	// Warning: impossible age (need to set up properly)
	ind2 := makeIndividual("@I2@", 1800, 1960) // 160 years old
	ind2.Names = []*gedcom.PersonalName{{Full: "Test /Two/"}}

	doc := makeDocument([]*gedcom.Individual{ind1, ind2}, nil)

	report := a.Analyze(doc)

	// Check that issues are properly categorized
	if report.ErrorCount == 0 {
		t.Error("expected at least one error")
	}
	if report.WarningCount == 0 {
		t.Error("expected at least one warning")
	}

	// Verify TotalIssues equals sum of counts
	expectedTotal := report.ErrorCount + report.WarningCount + report.InfoCount
	if report.TotalIssues != expectedTotal {
		t.Errorf("TotalIssues = %d, want %d", report.TotalIssues, expectedTotal)
	}

	// Verify Errors slice has correct count
	if len(report.Errors) != report.ErrorCount {
		t.Errorf("len(Errors) = %d, ErrorCount = %d", len(report.Errors), report.ErrorCount)
	}

	// Verify all items in Errors have Error severity
	for _, issue := range report.Errors {
		if issue.Severity != SeverityError {
			t.Errorf("issue in Errors has severity %v, want ERROR", issue.Severity)
		}
	}

	// Verify all items in Warnings have Warning severity
	for _, issue := range report.Warnings {
		if issue.Severity != SeverityWarning {
			t.Errorf("issue in Warnings has severity %v, want WARNING", issue.Severity)
		}
	}

	// Verify all items in Info have Info severity
	for _, issue := range report.Info {
		if issue.Severity != SeverityInfo {
			t.Errorf("issue in Info has severity %v, want INFO", issue.Severity)
		}
	}
}

func TestQualityReport_String(t *testing.T) {
	a := NewQualityAnalyzer()

	ind1 := makeIndividualWithDetails("@I1@", 1950, true, true, true)
	ind2 := makeIndividualWithDetails("@I2@", 0, false, false, true) // Missing birth
	fam := &gedcom.Family{XRef: "@F1@"}
	src := &gedcom.Source{XRef: "@S1@"}

	doc := makeDocumentWithSources([]*gedcom.Individual{ind1, ind2}, []*gedcom.Family{fam}, []*gedcom.Source{src})

	report := a.Analyze(doc)
	output := report.String()

	// Check required sections
	if !strings.Contains(output, "GEDCOM Quality Report") {
		t.Error("output should contain 'GEDCOM Quality Report'")
	}
	if !strings.Contains(output, "Records:") {
		t.Error("output should contain 'Records:'")
	}
	if !strings.Contains(output, "2 individuals") {
		t.Error("output should contain '2 individuals'")
	}
	if !strings.Contains(output, "Data Completeness:") {
		t.Error("output should contain 'Data Completeness:'")
	}
	if !strings.Contains(output, "Birth dates:") {
		t.Error("output should contain 'Birth dates:'")
	}
	if !strings.Contains(output, "Sources:") {
		t.Error("output should contain 'Sources:'")
	}
	if !strings.Contains(output, "Issues Found:") {
		t.Error("output should contain 'Issues Found:'")
	}
}

func TestQualityReport_JSON(t *testing.T) {
	a := NewQualityAnalyzer()

	ind := makeIndividualWithDetails("@I1@", 1950, true, true, true)
	doc := makeDocument([]*gedcom.Individual{ind}, nil)

	report := a.Analyze(doc)
	jsonBytes, err := report.JSON()

	if err != nil {
		t.Fatalf("JSON() returned error: %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("JSON output is not valid: %v", err)
	}

	// Check expected fields exist
	if _, ok := parsed["total_individuals"]; !ok {
		t.Error("JSON should contain 'total_individuals'")
	}
	if _, ok := parsed["birth_date_coverage"]; !ok {
		t.Error("JSON should contain 'birth_date_coverage'")
	}
	if _, ok := parsed["errors"]; !ok {
		t.Error("JSON should contain 'errors'")
	}
}

func TestQualityReport_IssuesForRecord(t *testing.T) {
	a := NewQualityAnalyzer()

	// Create individual with orphaned reference
	ind := &gedcom.Individual{
		XRef:            "@I1@",
		ChildInFamilies: []gedcom.FamilyLink{{FamilyXRef: "@F999@"}},
	}
	doc := makeDocument([]*gedcom.Individual{ind}, nil)

	report := a.Analyze(doc)

	// Get issues for @I1@
	issues := report.IssuesForRecord("@I1@")
	if len(issues) == 0 {
		t.Error("expected issues for @I1@, got none")
	}

	// All issues should reference @I1@
	for _, issue := range issues {
		if issue.RecordXRef != "@I1@" && issue.RelatedXRef != "@I1@" {
			t.Errorf("issue does not reference @I1@: %v", issue)
		}
	}

	// Non-existent record should return empty
	issues = report.IssuesForRecord("@NONEXISTENT@")
	if len(issues) != 0 {
		t.Errorf("expected no issues for non-existent record, got %d", len(issues))
	}
}

func TestQualityReport_IssuesByCode(t *testing.T) {
	a := NewQualityAnalyzer()

	// Create individual without birth date (will generate MISSING_BIRTH_DATE)
	ind := &gedcom.Individual{XRef: "@I1@"}
	doc := makeDocument([]*gedcom.Individual{ind}, nil)

	report := a.Analyze(doc)

	// Get issues by code
	issues := report.IssuesByCode(CodeMissingBirthDate)
	if len(issues) == 0 {
		t.Error("expected MISSING_BIRTH_DATE issues, got none")
	}

	// All returned issues should have the correct code
	for _, issue := range issues {
		if issue.Code != CodeMissingBirthDate {
			t.Errorf("issue has code %q, want %q", issue.Code, CodeMissingBirthDate)
		}
	}

	// Non-existent code should return empty
	issues = report.IssuesByCode("NONEXISTENT_CODE")
	if len(issues) != 0 {
		t.Errorf("expected no issues for non-existent code, got %d", len(issues))
	}
}

func TestQualityReport_IssuesForRecordWithRelated(t *testing.T) {
	a := NewQualityAnalyzer()

	// Create potential duplicates - both should appear when querying either
	ind1 := &gedcom.Individual{
		XRef:  "@I1@",
		Names: []*gedcom.PersonalName{{Given: "John", Surname: "Doe"}},
		Sex:   "M",
		Events: []*gedcom.Event{{
			Type:       gedcom.EventBirth,
			ParsedDate: makeYearDate(1950),
		}},
	}
	ind2 := &gedcom.Individual{
		XRef:  "@I2@",
		Names: []*gedcom.PersonalName{{Given: "John", Surname: "Doe"}},
		Sex:   "M",
		Events: []*gedcom.Event{{
			Type:       gedcom.EventBirth,
			ParsedDate: makeYearDate(1950),
		}},
	}

	doc := makeDocument([]*gedcom.Individual{ind1, ind2}, nil)
	report := a.Analyze(doc)

	// Should find duplicate issue when querying @I1@
	issues1 := report.IssuesForRecord("@I1@")
	foundDup1 := false
	for _, issue := range issues1 {
		if issue.Code == CodePotentialDuplicate {
			foundDup1 = true
			break
		}
	}
	if !foundDup1 {
		t.Error("expected duplicate issue for @I1@")
	}

	// Should also find duplicate issue when querying @I2@ (as RelatedXRef)
	issues2 := report.IssuesForRecord("@I2@")
	foundDup2 := false
	for _, issue := range issues2 {
		if issue.Code == CodePotentialDuplicate {
			foundDup2 = true
			break
		}
	}
	if !foundDup2 {
		t.Error("expected duplicate issue for @I2@ (as related record)")
	}
}

func TestQualityAnalyzer_Analyze_PlaceDetection(t *testing.T) {
	a := NewQualityAnalyzer()

	tests := []struct {
		name     string
		place    string
		hasPlace bool
	}{
		{"with place string", "Test Place", true},
		{"without place", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ind := &gedcom.Individual{
				XRef:  "@I1@",
				Names: []*gedcom.PersonalName{{Full: "Test"}},
				Events: []*gedcom.Event{{
					Type:       gedcom.EventBirth,
					ParsedDate: makeYearDate(1950),
					Place:      tt.place,
				}},
				SourceCitations: []*gedcom.SourceCitation{{SourceXRef: "@S1@"}},
			}

			src := &gedcom.Source{XRef: "@S1@"}
			doc := makeDocumentWithSources([]*gedcom.Individual{ind}, nil, []*gedcom.Source{src})

			report := a.Analyze(doc)

			if tt.hasPlace && report.IndividualsWithPlaces != 1 {
				t.Errorf("IndividualsWithPlaces = %d, want 1", report.IndividualsWithPlaces)
			}
			if !tt.hasPlace && report.IndividualsWithPlaces != 0 {
				t.Errorf("IndividualsWithPlaces = %d, want 0", report.IndividualsWithPlaces)
			}
		})
	}
}

func TestQualityAnalyzer_Analyze_PlaceDetailDetection(t *testing.T) {
	a := NewQualityAnalyzer()

	ind := &gedcom.Individual{
		XRef:  "@I1@",
		Names: []*gedcom.PersonalName{{Full: "Test"}},
		Events: []*gedcom.Event{{
			Type:        gedcom.EventBirth,
			ParsedDate:  makeYearDate(1950),
			PlaceDetail: &gedcom.PlaceDetail{Name: "Detailed Place"},
		}},
		SourceCitations: []*gedcom.SourceCitation{{SourceXRef: "@S1@"}},
	}

	src := &gedcom.Source{XRef: "@S1@"}
	doc := makeDocumentWithSources([]*gedcom.Individual{ind}, nil, []*gedcom.Source{src})

	report := a.Analyze(doc)

	if report.IndividualsWithPlaces != 1 {
		t.Errorf("IndividualsWithPlaces = %d, want 1 (PlaceDetail)", report.IndividualsWithPlaces)
	}
}

func TestQualityReport_String_TopIssues(t *testing.T) {
	a := NewQualityAnalyzer()

	// Create multiple individuals with missing birth dates to see top issues
	var individuals []*gedcom.Individual
	for i := 0; i < 5; i++ {
		ind := &gedcom.Individual{XRef: "@I" + string(rune('1'+i)) + "@"}
		individuals = append(individuals, ind)
	}

	doc := makeDocument(individuals, nil)
	report := a.Analyze(doc)
	output := report.String()

	if !strings.Contains(output, "Top Issues:") {
		t.Error("output should contain 'Top Issues:' section")
	}
	if !strings.Contains(output, CodeMissingBirthDate) {
		t.Errorf("output should contain %s in top issues", CodeMissingBirthDate)
	}
}

func TestQualityReport_EmptySlicesNotNil(t *testing.T) {
	a := NewQualityAnalyzer()

	// Analyze nil document
	report := a.Analyze(nil)

	// Slices should be initialized, not nil
	if report.Errors == nil {
		t.Error("Errors should not be nil")
	}
	if report.Warnings == nil {
		t.Error("Warnings should not be nil")
	}
	if report.Info == nil {
		t.Error("Info should not be nil")
	}
	if report.DateLogicIssues == nil {
		t.Error("DateLogicIssues should not be nil")
	}
	if report.ReferenceIssues == nil {
		t.Error("ReferenceIssues should not be nil")
	}
	if report.DuplicateIssues == nil {
		t.Error("DuplicateIssues should not be nil")
	}
	if report.CompletenessIssues == nil {
		t.Error("CompletenessIssues should not be nil")
	}
}

func TestQualityAnalyzer_Analyze_DeathDateCoverage(t *testing.T) {
	a := NewQualityAnalyzer()

	// Create individuals with varying death date status
	ind1 := makeIndividual("@I1@", 1900, 1980) // Has death date
	ind1.Names = []*gedcom.PersonalName{{Full: "Test /One/"}}

	ind2 := makeIndividual("@I2@", 1920, 0) // No death date
	ind2.Names = []*gedcom.PersonalName{{Full: "Test /Two/"}}

	doc := makeDocument([]*gedcom.Individual{ind1, ind2}, nil)
	report := a.Analyze(doc)

	if report.IndividualsWithDeathDate != 1 {
		t.Errorf("IndividualsWithDeathDate = %d, want 1", report.IndividualsWithDeathDate)
	}
	if report.DeathDateCoverage != 0.5 {
		t.Errorf("DeathDateCoverage = %f, want 0.5", report.DeathDateCoverage)
	}
}

func TestQualityAnalyzer_CustomDateLogicConfig(t *testing.T) {
	// Use a custom max age of 100
	config := &DateLogicConfig{MaxReasonableAge: 100}
	a := NewQualityAnalyzer(WithDateLogicConfig(config))

	// Create individual aged 101 (should trigger warning with custom config)
	ind := makeIndividual("@I1@", 1800, 1901) // 101 years old
	ind.Names = []*gedcom.PersonalName{{Full: "Test /Person/"}}

	doc := makeDocument([]*gedcom.Individual{ind}, nil)
	report := a.Analyze(doc)

	// Should have impossible age warning
	foundAgeWarning := false
	for _, issue := range report.DateLogicIssues {
		if issue.Code == CodeImpossibleAge {
			foundAgeWarning = true
			break
		}
	}
	if !foundAgeWarning {
		t.Error("expected IMPOSSIBLE_AGE warning with custom config (max 100)")
	}
}

func TestQualityReport_JSON_AllFields(t *testing.T) {
	a := NewQualityAnalyzer()

	// Create a comprehensive document
	ind := makeIndividual("@I1@", 1950, 1940) // death before birth
	ind.Names = []*gedcom.PersonalName{{Full: "Test /Person/"}}
	ind.SourceCitations = []*gedcom.SourceCitation{{SourceXRef: "@S1@"}}

	src := &gedcom.Source{XRef: "@S1@"}
	fam := &gedcom.Family{XRef: "@F1@"}
	doc := makeDocumentWithSources([]*gedcom.Individual{ind}, []*gedcom.Family{fam}, []*gedcom.Source{src})

	report := a.Analyze(doc)
	jsonBytes, err := report.JSON()
	if err != nil {
		t.Fatalf("JSON() error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Check all expected fields
	expectedFields := []string{
		"total_individuals",
		"total_families",
		"total_sources",
		"individuals_with_birth_date",
		"individuals_with_death_date",
		"individuals_with_sources",
		"individuals_with_places",
		"birth_date_coverage",
		"death_date_coverage",
		"source_coverage",
		"errors",
		"warnings",
		"info",
		"date_logic_issues",
		"reference_issues",
		"duplicate_issues",
		"completeness_issues",
		"total_issues",
		"error_count",
		"warning_count",
		"info_count",
	}

	for _, field := range expectedFields {
		if _, ok := parsed[field]; !ok {
			t.Errorf("JSON missing field: %s", field)
		}
	}
}

func TestQualityReport_CountIssuesByCode(t *testing.T) {
	report := &QualityReport{
		Errors: []Issue{
			{Code: CodeDeathBeforeBirth},
			{Code: CodeDeathBeforeBirth},
		},
		Warnings: []Issue{
			{Code: CodeImpossibleAge},
		},
		Info: []Issue{
			{Code: CodeMissingBirthDate},
			{Code: CodeMissingBirthDate},
			{Code: CodeMissingBirthDate},
		},
	}

	counts := report.countIssuesByCode()

	if counts[CodeDeathBeforeBirth] != 2 {
		t.Errorf("count for %s = %d, want 2", CodeDeathBeforeBirth, counts[CodeDeathBeforeBirth])
	}
	if counts[CodeImpossibleAge] != 1 {
		t.Errorf("count for %s = %d, want 1", CodeImpossibleAge, counts[CodeImpossibleAge])
	}
	if counts[CodeMissingBirthDate] != 3 {
		t.Errorf("count for %s = %d, want 3", CodeMissingBirthDate, counts[CodeMissingBirthDate])
	}
}

func TestQualityReport_ZeroCoverageWhenNoIndividuals(t *testing.T) {
	a := NewQualityAnalyzer()

	// Empty document with only families/sources
	fam := &gedcom.Family{XRef: "@F1@"}
	src := &gedcom.Source{XRef: "@S1@"}
	doc := makeDocumentWithSources(nil, []*gedcom.Family{fam}, []*gedcom.Source{src})

	report := a.Analyze(doc)

	// All percentages should be 0 (not NaN)
	if report.BirthDateCoverage != 0.0 {
		t.Errorf("BirthDateCoverage = %f, want 0.0", report.BirthDateCoverage)
	}
	if report.DeathDateCoverage != 0.0 {
		t.Errorf("DeathDateCoverage = %f, want 0.0", report.DeathDateCoverage)
	}
	if report.SourceCoverage != 0.0 {
		t.Errorf("SourceCoverage = %f, want 0.0", report.SourceCoverage)
	}
}

func TestQualityAnalyzer_AllValidatorsInvoked_Integration(t *testing.T) {
	a := NewQualityAnalyzer()

	// Create a complex document that triggers all validators

	// 1. Date logic: death before birth
	ind1 := makeIndividual("@I1@", 1950, 1940)
	ind1.Names = []*gedcom.PersonalName{{Full: "Test /One/"}}
	ind1.SourceCitations = []*gedcom.SourceCitation{{SourceXRef: "@S1@"}}

	// 2. References: orphaned FAMC
	ind2 := &gedcom.Individual{
		XRef:            "@I2@",
		Names:           []*gedcom.PersonalName{{Full: "Test /Two/"}},
		ChildInFamilies: []gedcom.FamilyLink{{FamilyXRef: "@F999@"}}, // Non-existent
		Events: []*gedcom.Event{{
			Type:       gedcom.EventBirth,
			ParsedDate: makeYearDate(1960),
		}},
		SourceCitations: []*gedcom.SourceCitation{{SourceXRef: "@S1@"}},
	}

	// 3. Duplicates: same name and birth year
	ind3 := &gedcom.Individual{
		XRef:  "@I3@",
		Names: []*gedcom.PersonalName{{Given: "Duplicate", Surname: "Person"}},
		Sex:   "M",
		Events: []*gedcom.Event{{
			Type:       gedcom.EventBirth,
			ParsedDate: makeYearDate(1970),
		}},
		SourceCitations: []*gedcom.SourceCitation{{SourceXRef: "@S1@"}},
	}
	ind4 := &gedcom.Individual{
		XRef:  "@I4@",
		Names: []*gedcom.PersonalName{{Given: "Duplicate", Surname: "Person"}},
		Sex:   "M",
		Events: []*gedcom.Event{{
			Type:       gedcom.EventBirth,
			ParsedDate: makeYearDate(1970),
		}},
		SourceCitations: []*gedcom.SourceCitation{{SourceXRef: "@S1@"}},
	}

	// 4. Completeness: individual with no name or birth
	ind5 := &gedcom.Individual{XRef: "@I5@"}

	src := &gedcom.Source{XRef: "@S1@", Title: "Test Source"}
	fam := &gedcom.Family{XRef: "@F1@"}

	doc := makeDocumentWithSources(
		[]*gedcom.Individual{ind1, ind2, ind3, ind4, ind5},
		[]*gedcom.Family{fam},
		[]*gedcom.Source{src},
	)

	report := a.Analyze(doc)

	// Verify all validator categories have issues
	if len(report.DateLogicIssues) == 0 {
		t.Error("expected DateLogicIssues, got none")
	}
	if len(report.ReferenceIssues) == 0 {
		t.Error("expected ReferenceIssues, got none")
	}
	if len(report.DuplicateIssues) == 0 {
		t.Error("expected DuplicateIssues, got none")
	}
	if len(report.CompletenessIssues) == 0 {
		t.Error("expected CompletenessIssues, got none")
	}

	// Verify issue counts are consistent
	totalFromSlices := len(report.Errors) + len(report.Warnings) + len(report.Info)
	if report.TotalIssues != totalFromSlices {
		t.Errorf("TotalIssues = %d, but sum of slices = %d", report.TotalIssues, totalFromSlices)
	}
}
