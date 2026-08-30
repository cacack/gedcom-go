package validator

import (
	"fmt"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// PlaceConsistencyValidator reports events and attributes whose two place
// carriers disagree.
//
// A place can be recorded twice: on the legacy Place scalar and on
// PlaceDetail.Name. Decode fills both from the same line, so a decoded document
// never disagrees. A hand-built one can, and the disagreement is invisible at
// runtime: the read path ([gedcom.Event.PlaceName]) prefers PlaceDetail.Name
// while the encoder prefers the scalar, so a caller who updates only
// PlaceDetail.Name reads the new value everywhere and writes the old one to the
// file, with no error and no diagnostic.
//
// This validator is the missing signal. It is deliberately narrow: it fires
// only when both carriers are non-empty and differ, which is never a shape the
// library itself produces.
type PlaceConsistencyValidator struct{}

// NewPlaceConsistencyValidator creates a new PlaceConsistencyValidator.
func NewPlaceConsistencyValidator() *PlaceConsistencyValidator {
	return &PlaceConsistencyValidator{}
}

// Validate runs the place-carrier consistency check across every individual in
// the document and returns any issues found.
func (v *PlaceConsistencyValidator) Validate(doc *gedcom.Document) []Issue {
	if doc == nil {
		return nil
	}

	var issues []Issue
	for _, ind := range doc.Individuals() {
		issues = append(issues, v.ValidateIndividual(ind)...)
	}
	return issues
}

// ValidateIndividual runs the place-carrier consistency check on a single
// individual's events and attributes.
func (v *PlaceConsistencyValidator) ValidateIndividual(ind *gedcom.Individual) []Issue {
	if ind == nil {
		return nil
	}

	var issues []Issue

	for _, event := range ind.Events {
		if event == nil {
			continue
		}
		if issue := v.checkCarriers(ind.XRef, string(event.Type), event.Place, event.PlaceDetail); issue != nil {
			issues = append(issues, *issue)
		}
	}

	for _, attr := range ind.Attributes {
		if attr == nil {
			continue
		}
		if issue := v.checkCarriers(ind.XRef, attr.Type, attr.Place, attr.PlaceDetail); issue != nil {
			issues = append(issues, *issue)
		}
	}

	return issues
}

// checkCarriers reports a mismatch only when both carriers hold a value and the
// values differ. One carrier empty is the ordinary shape -- it is what a
// structured-only or scalar-only caller produces, and the encoder handles both.
func (v *PlaceConsistencyValidator) checkCarriers(xref, tag, place string, detail *gedcom.PlaceDetail) *Issue {
	if detail == nil || place == "" || detail.Name == "" || place == detail.Name {
		return nil
	}

	issue := NewIssue(
		SeverityWarning,
		CodePlaceCarrierMismatch,
		fmt.Sprintf("%s place disagrees between Place (%q) and PlaceDetail.Name (%q); the encoder writes Place, but PlaceName() reads PlaceDetail.Name", tag, place, detail.Name),
		xref,
	).
		WithDetail("tag", tag).
		WithDetail("place", place).
		WithDetail("place_detail_name", detail.Name)
	return &issue
}
