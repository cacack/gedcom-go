package converter

import (
	"github.com/cacack/gedcom-go/gedcom"
	"github.com/cacack/gedcom-go/validator"
)

// validateConverted runs validation on the converted document.
// Validation issues are added to the report.
// Returns nil - validation issues are informational and don't fail conversion.
//
//nolint:unparam // error return kept for potential future validation failures
func validateConverted(doc *gedcom.Document, report *gedcom.ConversionReport) error {
	v := validator.New()
	errs := v.Validate(doc)

	// Convert errors to strings for report
	for _, err := range errs {
		report.ValidationIssues = append(report.ValidationIssues, err.Error())
	}

	return nil
}
