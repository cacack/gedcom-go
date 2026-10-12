// date_logic.go provides date logic validation for detecting genealogically impossible
// or improbable date relationships in GEDCOM data.
//
// The DateLogicValidator detects issues such as:
//   - Death before birth
//   - Children born before parents
//   - Marriage before birth
//   - Impossible ages (e.g., >120 years)
//   - Unreasonable parent ages at child's birth

package validator

import (
	"fmt"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// DateLogicConfig contains configurable thresholds for date logic validation.
type DateLogicConfig struct {
	// MaxReasonableAge is the maximum reasonable lifespan in years.
	// Ages exceeding this generate a warning. Default: 120.
	MaxReasonableAge int

	// MinParentAge is the minimum reasonable age to become a parent.
	// Parents younger than this at child's birth generate a warning. Default: 12.
	MinParentAge int

	// MaxMotherAge is the maximum reasonable age for a mother at child's birth.
	// Mothers older than this generate a warning. Default: 55.
	MaxMotherAge int

	// MaxFatherAge is the maximum reasonable age for a father at child's birth.
	// Fathers older than this generate a warning. Default: 90.
	MaxFatherAge int
}

// DefaultDateLogicConfig returns a DateLogicConfig with reasonable defaults.
func DefaultDateLogicConfig() *DateLogicConfig {
	return &DateLogicConfig{
		MaxReasonableAge: 120,
		MinParentAge:     12,
		MaxMotherAge:     55,
		MaxFatherAge:     90,
	}
}

// DateLogicValidator validates date logic in GEDCOM documents.
type DateLogicValidator struct {
	config *DateLogicConfig
}

// NewDateLogicValidator creates a new DateLogicValidator with the given configuration.
// If config is nil, default values are used.
func NewDateLogicValidator(config *DateLogicConfig) *DateLogicValidator {
	if config == nil {
		config = DefaultDateLogicConfig()
	}
	// Apply defaults for any zero values
	if config.MaxReasonableAge == 0 {
		config.MaxReasonableAge = 120
	}
	if config.MinParentAge == 0 {
		config.MinParentAge = 12
	}
	if config.MaxMotherAge == 0 {
		config.MaxMotherAge = 55
	}
	if config.MaxFatherAge == 0 {
		config.MaxFatherAge = 90
	}
	return &DateLogicValidator{config: config}
}

// Validate runs all date logic validations on the document and returns any issues found.
func (v *DateLogicValidator) Validate(doc *gedcom.Document) []Issue {
	if doc == nil {
		return nil
	}

	var issues []Issue
	for _, ind := range doc.Individuals() {
		issues = append(issues, v.ValidateIndividual(doc, ind)...)
	}
	return issues
}

// ValidateIndividual runs all date logic validations on a single individual.
func (v *DateLogicValidator) ValidateIndividual(doc *gedcom.Document, ind *gedcom.Individual) []Issue {
	if ind == nil {
		return nil
	}

	var issues []Issue

	// Check death before birth
	if issue := v.checkDeathBeforeBirth(ind); issue != nil {
		issues = append(issues, *issue)
	}

	// Check child born before parent
	issues = append(issues, v.checkChildBeforeParent(doc, ind)...)

	// Check marriage before birth
	issues = append(issues, v.checkMarriageBeforeBirth(doc, ind)...)

	// Check reasonable age (lifespan)
	if issue := v.checkReasonableAge(ind); issue != nil {
		issues = append(issues, *issue)
	}

	// Check reasonable parent age
	issues = append(issues, v.checkReasonableParentAge(doc, ind)...)

	return issues
}

// checkDeathBeforeBirth checks if an individual's death date is before their birth date.
// Returns an Issue whose severity comes from orderSeverity if detected, nil otherwise.
func (v *DateLogicValidator) checkDeathBeforeBirth(ind *gedcom.Individual) *Issue {
	birthDate := ind.BirthDate()
	deathDate := ind.DeathDate()

	// Skip if either date is missing
	if birthDate == nil || deathDate == nil {
		return nil
	}

	// Skip if either date has no year (can't compare meaningfully)
	if birthDate.Year == 0 || deathDate.Year == 0 {
		return nil
	}

	// Check if death is before birth
	if severity, found := orderSeverity(deathDate, birthDate); found {
		issue := NewIssue(
			severity,
			CodeDeathBeforeBirth,
			fmt.Sprintf("death date (%s) is %s birth date (%s)", deathDate.Original, orderVerb(severity), birthDate.Original),
			ind.XRef,
		).
			WithDetail("birth_date", birthDate.Original).
			WithDetail("death_date", deathDate.Original)
		return &issue
	}

	return nil
}

// checkChildBeforeParent checks if an individual was born before any of their parents.
// Returns an Issue for each parent born after the child, with severity from orderSeverity.
func (v *DateLogicValidator) checkChildBeforeParent(doc *gedcom.Document, ind *gedcom.Individual) []Issue {
	if doc == nil {
		return nil
	}

	childBirth := ind.BirthDate()
	if childBirth == nil || childBirth.Year == 0 {
		return nil
	}

	var issues []Issue
	parents := ind.Parents(doc)

	for _, parent := range parents {
		parentBirth := parent.BirthDate()
		if parentBirth == nil || parentBirth.Year == 0 {
			continue
		}

		// Child born before parent is impossible
		if severity, found := orderSeverity(childBirth, parentBirth); found {
			issue := NewIssue(
				severity,
				CodeChildBeforeParent,
				fmt.Sprintf("child born (%s) %s parent born (%s)", childBirth.Original, orderVerb(severity), parentBirth.Original),
				ind.XRef,
			).
				WithRelatedXRef(parent.XRef).
				WithDetail("child_birth", childBirth.Original).
				WithDetail("parent_birth", parentBirth.Original)
			issues = append(issues, issue)
		}
	}

	return issues
}

// checkMarriageBeforeBirth checks if an individual was married before they were born.
// Returns an Issue for each marriage dated before birth, with severity from orderSeverity.
func (v *DateLogicValidator) checkMarriageBeforeBirth(doc *gedcom.Document, ind *gedcom.Individual) []Issue {
	if doc == nil {
		return nil
	}

	birthDate := ind.BirthDate()
	if birthDate == nil || birthDate.Year == 0 {
		return nil
	}

	var issues []Issue

	// Check all families where this individual is a spouse
	for _, link := range ind.SpouseInFamilies {
		fam := doc.GetFamily(link.FamilyXRef)
		if fam == nil {
			continue
		}

		// Look for marriage events. A negative assertion (NO MARR) records a
		// marriage that did not happen, so its date says nothing about birth.
		for _, event := range fam.OccurredEvents() {
			if event.Type != gedcom.EventMarriage {
				continue
			}

			marriageDate := event.ParsedDate
			if marriageDate == nil || marriageDate.Year == 0 {
				continue
			}

			// Marriage before birth is impossible
			if severity, found := orderSeverity(marriageDate, birthDate); found {
				issue := NewIssue(
					severity,
					CodeMarriageBeforeBirth,
					fmt.Sprintf("marriage date (%s) is %s birth date (%s)", marriageDate.Original, orderVerb(severity), birthDate.Original),
					ind.XRef,
				).
					WithRelatedXRef(fam.XRef).
					WithDetail("birth_date", birthDate.Original).
					WithDetail("marriage_date", marriageDate.Original)
				issues = append(issues, issue)
			}
		}
	}

	return issues
}

// orderSeverity reports whether date a may fall before date b, and how
// certain that is. It is SeverityError when the bounds prove it (a's latest day
// is before b's earliest; see gedcom.Date.Bounds), and SeverityWarning when
// Compare puts a first but the dates are too imprecise to prove it, as with
// "1503" against "2 FEB 1503". An approximate date (ABT, CAL or EST) on
// either side caps it at SeverityWarning even when the bounds are disjoint,
// since Bounds does not guess how far "about" reaches; INT names an exact
// date and is not capped. found is false when a is not before b by Compare,
// or the bounds prove it is not; severity is then SeverityInfo and must be
// ignored.
func orderSeverity(a, b *gedcom.Date) (severity Severity, found bool) {
	aEarliest, aLatest, aOK := a.Bounds()
	bEarliest, bLatest, bOK := b.Bounds()
	if aOK && bOK {
		if aLatest < bEarliest {
			if isApproximate(a) || isApproximate(b) {
				return SeverityWarning, true
			}
			return SeverityError, true
		}
		// The bounds prove a is not before b; checked ahead of IsBefore so
		// the bounds decide whenever they can, whatever Compare's sort order.
		if aEarliest >= bLatest {
			return SeverityInfo, false
		}
	}
	if a.IsBefore(b) {
		return SeverityWarning, true
	}
	return SeverityInfo, false
}

// isApproximate reports whether d carries an ABT, CAL or EST modifier. Only
// the top-level Modifier is checked: the date grammar does not let ABT, CAL or
// EST appear inside BET, FROM/TO, BEF or AFT, so ranges are never approximate.
func isApproximate(d *gedcom.Date) bool {
	switch d.Modifier {
	case gedcom.ModifierAbout, gedcom.ModifierCalculated, gedcom.ModifierEstimated:
		return true
	default:
		return false
	}
}

// orderVerb phrases an ordering issue's message for its severity.
func orderVerb(severity Severity) string {
	if severity == SeverityError {
		return "before"
	}
	return "possibly before"
}

// checkReasonableAge checks if an individual's lifespan exceeds the maximum reasonable age.
// Returns an Issue with Warning severity if the age exceeds the configured maximum.
func (v *DateLogicValidator) checkReasonableAge(ind *gedcom.Individual) *Issue {
	birthDate := ind.BirthDate()
	deathDate := ind.DeathDate()

	// Skip if either date is missing
	if birthDate == nil || deathDate == nil {
		return nil
	}

	// Calculate years between birth and death
	years, _, err := gedcom.YearsBetween(birthDate, deathDate)
	if err != nil {
		return nil
	}

	// Check if age exceeds maximum
	if years > v.config.MaxReasonableAge {
		issue := NewIssue(
			SeverityWarning,
			CodeImpossibleAge,
			fmt.Sprintf("age of %d years exceeds maximum reasonable age of %d", years, v.config.MaxReasonableAge),
			ind.XRef,
		).
			WithDetail("age", fmt.Sprintf("%d", years)).
			WithDetail("max_age", fmt.Sprintf("%d", v.config.MaxReasonableAge)).
			WithDetail("birth_date", birthDate.Original).
			WithDetail("death_date", deathDate.Original)
		return &issue
	}

	return nil
}

// checkReasonableParentAge checks if an individual was a reasonable age when their children were born.
// Returns Issues with Warning severity for each child where the parent's age was unreasonable.
func (v *DateLogicValidator) checkReasonableParentAge(doc *gedcom.Document, ind *gedcom.Individual) []Issue {
	if doc == nil {
		return nil
	}

	parentBirth := ind.BirthDate()
	if parentBirth == nil || parentBirth.Year == 0 {
		return nil
	}

	// Determine max age based on sex
	var maxParentAge int
	if ind.Sex == "F" {
		maxParentAge = v.config.MaxMotherAge
	} else {
		// Default to father's age for male or unknown sex
		maxParentAge = v.config.MaxFatherAge
	}

	var issues []Issue
	children := ind.Children(doc)

	for _, child := range children {
		childBirth := child.BirthDate()
		if childBirth == nil || childBirth.Year == 0 {
			continue
		}

		// Calculate parent's age when child was born
		parentAge, _, err := gedcom.YearsBetween(parentBirth, childBirth)
		if err != nil {
			continue
		}

		// Check if parent was too young
		if parentAge < v.config.MinParentAge {
			issue := NewIssue(
				SeverityWarning,
				CodeUnreasonableParentAge,
				fmt.Sprintf("parent was %d years old at child's birth (minimum: %d)", parentAge, v.config.MinParentAge),
				ind.XRef,
			).
				WithRelatedXRef(child.XRef).
				WithDetail("parent_age", fmt.Sprintf("%d", parentAge)).
				WithDetail("min_age", fmt.Sprintf("%d", v.config.MinParentAge)).
				WithDetail("parent_birth", parentBirth.Original).
				WithDetail("child_birth", childBirth.Original)
			issues = append(issues, issue)
		}

		// Check if parent was too old
		if parentAge > maxParentAge {
			parentType := "father"
			if ind.Sex == "F" {
				parentType = "mother"
			}
			issue := NewIssue(
				SeverityWarning,
				CodeUnreasonableParentAge,
				fmt.Sprintf("%s was %d years old at child's birth (maximum: %d)", parentType, parentAge, maxParentAge),
				ind.XRef,
			).
				WithRelatedXRef(child.XRef).
				WithDetail("parent_age", fmt.Sprintf("%d", parentAge)).
				WithDetail("max_age", fmt.Sprintf("%d", maxParentAge)).
				WithDetail("parent_birth", parentBirth.Original).
				WithDetail("child_birth", childBirth.Original)
			issues = append(issues, issue)
		}
	}

	return issues
}
