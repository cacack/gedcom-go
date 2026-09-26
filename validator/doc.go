// Package validator provides GEDCOM document validation functionality.
//
// This package validates GEDCOM documents against specification rules for
// different GEDCOM versions (5.5, 5.5.1, 7.0). It checks for structural
// correctness, required fields, and valid cross-references.
//
// [Validator.ValidateAll] runs every document-wide check and returns []Issue,
// each finding carrying a severity, a Code* constant, the affected record and,
// where the check has one, the source line. [Validator.Validate] returns the
// same findings as []error, each element a *Issue.
//
// # Usage
//
// For validation returning errors:
//
//	doc, _ := decoder.Decode(reader)
//	v := validator.New()
//	errs := v.Validate(doc)
//	for _, err := range errs {
//	    fmt.Printf("%v\n", err)
//	}
//
// Recover the structured finding from an error with errors.As:
//
//	for _, err := range errs {
//	    var issue *validator.Issue
//	    if errors.As(err, &issue) && issue.Code == validator.CodeBrokenXRef {
//	        fmt.Printf("broken pointer at line %d\n", issue.LineNumber)
//	    }
//	}
//
// Or work with Issues directly:
//
//	v := validator.New()
//	issues := v.ValidateAll(doc)
//	for _, issue := range issues {
//	    fmt.Printf("[%s] %s: %s\n", issue.Severity, issue.Code, issue.Message)
//	}
//
// # Individual Validators
//
// You can run specific validators independently:
//
//	v := validator.New()
//	dateIssues := v.ValidateDateLogic(doc)      // Check date logic
//	refIssues := v.FindOrphanedReferences(doc)  // Find broken references
//	duplicates := v.FindPotentialDuplicates(doc) // Find potential duplicates
//
// Duplicate detection skips any surname group larger than
// [DuplicateConfig.MaxGroupSize]. Use [Validator.FindPotentialDuplicatesReport]
// instead of [Validator.FindPotentialDuplicates] to tell an empty result from
// an incomplete one.
//
// # Quality Reports
//
// Combine validation issues with completeness statistics. The report runs a
// subset of the checks -- see [Validator.QualityReport] -- so prefer ValidateAll
// when the issue set must be complete:
//
//	v := validator.New()
//	report := v.QualityReport(doc)
//	fmt.Printf("Errors: %d, Warnings: %d\n", report.ErrorCount, report.WarningCount)
//	fmt.Printf("Birth date coverage: %.0f%%\n", report.BirthDateCoverage*100)
//
// # Options
//
// Use [NewWithOptions] together with [ValidateOptions] to customize validation
// behavior. Every option applies to Validate and ValidateAll alike.
// [ValidatorConfig] is retained as a backward-compatible alias.
// Call [DefaultOptions] for a populated starting point.
//
//   - Strictness             — StrictnessRelaxed | StrictnessNormal (default) | StrictnessStrict
//   - MaxErrors              — cap collected issues (0 = unlimited)
//   - SkipRules              — issue codes to exclude (e.g. []string{"W001"})
//   - DateLogic              — date-logic thresholds (e.g. MaxReasonableAge)
//   - Duplicates             — duplicate-detection thresholds
//   - TagRegistry            — definitions for custom (underscore) tags
//   - ValidateCustomTags     — enable custom-tag validation against registry
//   - SkipEncodingValidation — disable GEDCOM 7.0 encoding checks
//   - SkipDuplicateDetection — drop duplicate detection from the ValidateAll sweep
//
// Example combining strictness with skip rules:
//
//	opts := &validator.ValidateOptions{
//	    Strictness: validator.StrictnessStrict,
//	    MaxErrors:  100,
//	    SkipRules:  []string{"W001"},
//	    DateLogic:  &validator.DateLogicConfig{MaxReasonableAge: 110},
//	}
//	issues := validator.NewWithOptions(opts).ValidateAll(doc)
package validator
