// Package validator provides GEDCOM document validation functionality.
//
// This package validates GEDCOM documents against specification rules for
// different GEDCOM versions (5.5, 5.5.1, 7.0). It checks for structural
// correctness, required fields, and valid cross-references.
//
// The validator provides two APIs:
//
//   - Validate() - Original API returning []error for backward compatibility
//   - ValidateAll() - Enhanced API returning []Issue with severity levels
//
// # Basic Usage
//
// For simple validation returning errors:
//
//	doc, _ := decoder.Decode(reader)
//	v := validator.New()
//	errors := v.Validate(doc)
//	if len(errors) > 0 {
//	    for _, err := range errors {
//	        fmt.Printf("%v\n", err)
//	    }
//	}
//
// # Enhanced Validation
//
// For detailed validation with severity levels:
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
// # Quality Reports
//
// Generate comprehensive data quality reports:
//
//	v := validator.New()
//	report := v.QualityReport(doc)
//	fmt.Printf("Errors: %d, Warnings: %d\n", report.ErrorCount, report.WarningCount)
//	fmt.Printf("Birth date coverage: %.0f%%\n", report.BirthDateCoverage*100)
//
// # Options
//
// Use [NewWithOptions] together with [ValidateOptions] to customize validation
// behavior. [ValidatorConfig] is retained as a backward-compatible alias.
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
