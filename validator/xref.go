// xref.go provides XRef length validation for GEDCOM 5.5/5.5.1 compliance.
//
// GEDCOM 5.5 and 5.5.1 specify that cross-reference identifiers (XRefs) should not
// exceed 20 characters (excluding the @ delimiters). This limit was removed in GEDCOM 7.0.
//
// Files with long XRefs may fail to import into legacy genealogy software, so validation
// warns users about potential interoperability issues.

package validator

import (
	"fmt"
	"strings"

	"github.com/cacack/gedcom-go/gedcom"
)

// MaxXRefLength is the maximum XRef content length (excluding @ delimiters)
// specified by GEDCOM 5.5 and 5.5.1.
const MaxXRefLength = 20

// XRefValidator validates XRef identifiers for GEDCOM version compliance.
// GEDCOM 5.5 and 5.5.1 limit XRef content to 20 characters; this limit was removed in 7.0.
type XRefValidator struct{}

// NewXRefValidator creates a new XRefValidator.
func NewXRefValidator() *XRefValidator {
	return &XRefValidator{}
}

// ValidateXRefs checks all XRefs in the document for version compliance.
// For GEDCOM 5.5 and 5.5.1, XRefs exceeding 20 characters (excluding @ delimiters)
// generate warnings. GEDCOM 7.0 has no length limit, so no issues are returned.
// Unknown versions are treated conservatively as pre-7.0.
func (x *XRefValidator) ValidateXRefs(doc *gedcom.Document) []Issue {
	var issues []Issue

	if doc == nil || doc.Header == nil {
		return issues
	}

	// No limit for GEDCOM 7.0+
	// Use explicit check for 7.0 to handle unknown versions conservatively
	if doc.Header.Version == gedcom.Version70 {
		return issues
	}

	for xref := range doc.XRefMap {
		content := strings.Trim(xref, "@")
		if len(content) > MaxXRefLength {
			issues = append(issues, NewIssue(
				SeverityWarning,
				CodeXRefTooLong,
				fmt.Sprintf("XRef %s exceeds %d-character limit for GEDCOM %s", xref, MaxXRefLength, doc.Header.Version),
				xref,
			).
				WithDetail("length", fmt.Sprintf("%d", len(content))).
				WithDetail("version", string(doc.Header.Version)))
		}
	}

	return issues
}
