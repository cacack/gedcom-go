package validator_test

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/validator"
)

// Example demonstrates basic document validation. Each error Validate
// returns is a *validator.Issue.
func Example() {
	// GEDCOM 7.0, where the header SUBM is optional.
	gedcomData := `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME John /Smith/
1 ASSO @I404@
2 ROLE GODP
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I999@
0 TRLR`

	doc, _ := decoder.Decode(strings.NewReader(gedcomData))

	v := validator.New()
	errs := v.Validate(doc)

	fmt.Printf("Found %d validation errors\n", len(errs))
	for _, err := range errs {
		fmt.Println(err)
	}

	var issue *validator.Issue
	if len(errs) > 0 && errors.As(errs[len(errs)-1], &issue) {
		fmt.Printf("%s at line %d\n", issue.Code, issue.LineNumber)
	}

	// Output:
	// Found 2 validation errors
	// [ERROR] ORPHANED_WIFE: WIFE reference to non-existent individual @I999@ (@F1@ -> @I999@) [line 10]
	// [ERROR] BROKEN_XREF: ASSO reference to non-existent record @I404@ (@I1@ -> @I404@) [line 6]
	// BROKEN_XREF at line 6
}

// ExampleNew shows creating a validator with default options.
func ExampleNew() {
	v := validator.New()

	gedcomData := `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME Alice /Johnson/
0 TRLR`

	doc, _ := decoder.Decode(strings.NewReader(gedcomData))

	errs := v.Validate(doc)
	fmt.Printf("Validation errors: %d\n", len(errs))

	// Output:
	// Validation errors: 0
}

// ExampleValidator_ValidateAll shows enhanced validation with severity levels.
func ExampleValidator_ValidateAll() {
	// Using GEDCOM 7.0 where SUBM is optional
	gedcomData := `0 HEAD
1 GEDC
2 VERS 7.0
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

	// Using GEDCOM 7.0 where SUBM is optional
	gedcomData := `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME John /Smith/
0 TRLR`

	doc, _ := decoder.Decode(strings.NewReader(gedcomData))

	issues := v.ValidateAll(doc)
	fmt.Printf("Issues found: %d\n", len(issues))

	// Output:
	// Issues found: 0
}

// ExampleNewWithOptions demonstrates combining Strictness, SkipRules, and MaxErrors.
func ExampleNewWithOptions() {
	opts := &validator.ValidateOptions{
		Strictness: validator.StrictnessStrict,
		MaxErrors:  10,
		SkipRules:  []string{"ENCODING_BANNED_C0"},
	}

	v := validator.NewWithOptions(opts)

	gedcomData := `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 1 JAN 2000
1 DEAT
2 DATE 1 JAN 1900
0 TRLR`

	doc, _ := decoder.Decode(strings.NewReader(gedcomData))

	issues := v.ValidateAll(doc)
	fmt.Printf("Issues found: %d\n", len(issues))
	if len(issues) > 0 {
		fmt.Printf("First: [%s] %s\n", issues[0].Severity, issues[0].Code)
	}

	// Output:
	// Issues found: 1
	// First: [ERROR] DEATH_BEFORE_BIRTH
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
