package validator_test

import (
	"fmt"
	"strings"

	"github.com/cacack/gedcom-go/decoder"
	"github.com/cacack/gedcom-go/validator"
)

// Example demonstrates basic document validation.
func Example() {
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I999@
0 TRLR`

	doc, _ := decoder.Decode(strings.NewReader(gedcomData))

	v := validator.New()
	errors := v.Validate(doc)

	if len(errors) > 0 {
		fmt.Printf("Found %d validation errors\n", len(errors))
	} else {
		fmt.Println("No validation errors")
	}

	// Output:
	// Found 1 validation errors
}

// ExampleNew shows creating a validator with default options.
func ExampleNew() {
	v := validator.New()

	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME Alice /Johnson/
0 TRLR`

	doc, _ := decoder.Decode(strings.NewReader(gedcomData))

	errors := v.Validate(doc)
	fmt.Printf("Validation errors: %d\n", len(errors))

	// Output:
	// Validation errors: 0
}

// ExampleValidator_ValidateAll shows enhanced validation with severity levels.
func ExampleValidator_ValidateAll() {
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 1 JAN 1900
1 DEAT
2 DATE 1 JAN 1850
0 TRLR`

	doc, _ := decoder.Decode(strings.NewReader(gedcomData))

	v := validator.New()
	issues := v.ValidateAll(doc)

	for _, issue := range issues {
		fmt.Printf("[%s] %s: %s\n", issue.Severity, issue.Code, issue.Message)
	}

	// Output:
	// [ERROR] DEATH_BEFORE_BIRTH: death date (1 JAN 1850) is before birth date (1 JAN 1900)
}

// ExampleNewWithConfig demonstrates custom validator configuration.
func ExampleNewWithConfig() {
	config := &validator.ValidatorConfig{
		Strictness: validator.StrictnessStrict,
		DateLogic: &validator.DateLogicConfig{
			MaxReasonableAge: 110,
			MinParentAge:     12,
			MaxMotherAge:     55,
			MaxFatherAge:     90,
		},
	}

	v := validator.NewWithConfig(config)

	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
0 TRLR`

	doc, _ := decoder.Decode(strings.NewReader(gedcomData))

	issues := v.ValidateAll(doc)
	fmt.Printf("Issues found: %d\n", len(issues))

	// Output:
	// Issues found: 0
}

// ExampleValidator_QualityReport shows generating a data quality report.
func ExampleValidator_QualityReport() {
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 15 MAR 1920
1 DEAT
2 DATE 20 JUN 1985
0 @I2@ INDI
1 NAME Jane /Doe/
0 TRLR`

	doc, _ := decoder.Decode(strings.NewReader(gedcomData))

	v := validator.New()
	report := v.QualityReport(doc)

	fmt.Printf("Total individuals: %d\n", report.TotalIndividuals)
	fmt.Printf("Birth date coverage: %.0f%%\n", report.BirthDateCoverage*100)

	// Output:
	// Total individuals: 2
	// Birth date coverage: 50%
}

// ExampleValidator_FindOrphanedReferences shows finding broken cross-references.
func ExampleValidator_FindOrphanedReferences() {
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I999@
0 TRLR`

	doc, _ := decoder.Decode(strings.NewReader(gedcomData))

	v := validator.New()
	issues := v.FindOrphanedReferences(doc)

	for _, issue := range issues {
		fmt.Printf("Orphaned reference: %s\n", issue.Message)
	}

	// Output:
	// Orphaned reference: WIFE reference to non-existent individual @I999@
}

// ExampleNewStreamingValidator demonstrates memory-efficient streaming validation.
func ExampleNewStreamingValidator() {
	// Streaming validation is useful for very large files
	sv := validator.NewStreamingValidator(validator.StreamingOptions{
		Strictness: validator.StrictnessNormal,
	})

	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 15 MAR 1920
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I999@
0 TRLR`

	doc, _ := decoder.Decode(strings.NewReader(gedcomData))

	// Validate records incrementally
	var allIssues []validator.Issue
	for _, record := range doc.Records {
		issues := sv.ValidateRecord(record)
		allIssues = append(allIssues, issues...)
	}

	// Finalize to check cross-references
	finalIssues := sv.Finalize()
	allIssues = append(allIssues, finalIssues...)

	fmt.Printf("Total issues: %d\n", len(allIssues))

	// Output:
	// Total issues: 1
}

// ExampleStreamingValidator_Reset shows reusing a streaming validator.
func ExampleStreamingValidator_Reset() {
	sv := validator.NewStreamingValidator(validator.StreamingOptions{})

	// After validating one file, reset for the next
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME Test /User/
0 TRLR`

	doc, _ := decoder.Decode(strings.NewReader(gedcomData))

	for _, record := range doc.Records {
		sv.ValidateRecord(record)
	}
	sv.Finalize()

	// Reset and reuse for another file
	sv.Reset()
	fmt.Printf("After reset - seen XRefs: %d\n", sv.SeenXRefCount())

	// Output:
	// After reset - seen XRefs: 0
}
