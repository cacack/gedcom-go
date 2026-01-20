// header.go provides validation for GEDCOM header requirements.
//
// This module validates version-specific header requirements, including
// the SUBM (Submitter) reference which is required in GEDCOM 5.5 and 5.5.1
// but optional in GEDCOM 7.0.

package validator

import (
	"fmt"

	"github.com/cacack/gedcom-go/gedcom"
)

// HeaderValidator validates GEDCOM header requirements.
type HeaderValidator struct{}

// NewHeaderValidator creates a new HeaderValidator.
func NewHeaderValidator() *HeaderValidator {
	return &HeaderValidator{}
}

// ValidateHeader validates the header of a GEDCOM document against
// version-specific requirements.
//
// Validations performed:
//   - SUBM reference: Required for GEDCOM 5.5 and 5.5.1 (cardinality {1:1}),
//     optional for GEDCOM 7.0.
func (h *HeaderValidator) ValidateHeader(doc *gedcom.Document) []Issue {
	if doc == nil || doc.Header == nil {
		return nil
	}

	var issues []Issue

	// SUBM is required for GEDCOM 5.5 and 5.5.1, optional for 7.0
	// Check if version is before 7.0 (i.e., 5.5 or 5.5.1)
	version := doc.Header.Version
	if (version == gedcom.Version55 || version == gedcom.Version551) && doc.Header.Submitter == "" {
		issues = append(issues, NewIssue(
			SeverityWarning,
			CodeMissingSUBM,
			fmt.Sprintf("GEDCOM %s requires SUBM reference in header", version),
			"",
		).WithDetail("version", string(version)))
	}

	return issues
}
