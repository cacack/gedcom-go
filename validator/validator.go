package validator

import (
	"fmt"

	"github.com/elliotchance/go-gedcom/gedcom"
)

// ValidationError represents a validation error.
type ValidationError struct {
	Code    string
	Message string
	Line    int
	XRef    string
}

func (e *ValidationError) Error() string {
	if e.XRef != "" {
		return fmt.Sprintf("[%s] %s (XRef: %s)", e.Code, e.Message, e.XRef)
	}
	if e.Line > 0 {
		return fmt.Sprintf("[%s] line %d: %s", e.Code, e.Line, e.Message)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Validator validates GEDCOM documents against specification rules.
type Validator struct {
	errors []error
}

// New creates a new Validator.
func New() *Validator {
	return &Validator{
		errors: make([]error, 0),
	}
}

// Validate validates a GEDCOM document and returns any validation errors.
func (v *Validator) Validate(doc *gedcom.Document) []error {
	v.errors = make([]error, 0)

	// Validate cross-references
	v.validateXRefs(doc)

	// Validate records
	v.validateRecords(doc)

	return v.errors
}

// validateXRefs checks that all cross-references are valid.
func (v *Validator) validateXRefs(doc *gedcom.Document) {
	// Track all XRef usages
	usedXRefs := make(map[string]bool)

	// Scan all records for XRef usage
	for _, record := range doc.Records {
		for _, tag := range record.Tags {
			// Check if value looks like an XRef
			if len(tag.Value) > 2 && tag.Value[0] == '@' && tag.Value[len(tag.Value)-1] == '@' {
				xref := tag.Value
				usedXRefs[xref] = true

				// Verify the XRef exists
				if doc.XRefMap[xref] == nil {
					v.errors = append(v.errors, &ValidationError{
						Code:    "BROKEN_XREF",
						Message: fmt.Sprintf("Reference to non-existent record %s", xref),
						Line:    tag.LineNumber,
					})
				}
			}
		}
	}
}

// validateRecords validates individual records.
func (v *Validator) validateRecords(doc *gedcom.Document) {
	for _, record := range doc.Records {
		switch record.Type {
		case gedcom.RecordTypeIndividual:
			v.validateIndividual(record)
		case gedcom.RecordTypeFamily:
			v.validateFamily(record)
		}
	}
}

// validateIndividual validates an individual record.
func (v *Validator) validateIndividual(record *gedcom.Record) {
	// Check for required NAME tag
	hasName := false
	for _, tag := range record.Tags {
		if tag.Tag == "NAME" {
			hasName = true
			break
		}
	}

	if !hasName {
		v.errors = append(v.errors, &ValidationError{
			Code:    "MISSING_REQUIRED_FIELD",
			Message: "Individual record missing required NAME tag",
			XRef:    record.XRef,
		})
	}
}

// validateFamily validates a family record.
func (v *Validator) validateFamily(record *gedcom.Record) {
	// Family records should have at least one spouse or child
	hasMembers := false
	for _, tag := range record.Tags {
		if tag.Tag == "HUSB" || tag.Tag == "WIFE" || tag.Tag == "CHIL" {
			hasMembers = true
			break
		}
	}

	if !hasMembers {
		v.errors = append(v.errors, &ValidationError{
			Code:    "EMPTY_FAMILY",
			Message: "Family record has no members (no HUSB, WIFE, or CHIL tags)",
			XRef:    record.XRef,
		})
	}
}
