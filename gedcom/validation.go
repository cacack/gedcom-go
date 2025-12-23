package gedcom

import "fmt"

// ValidateBirthBeforeDeath checks if a birth date is before or equal to a death date.
// Returns nil if the birth date is before or equal to the death date, or if either date is nil.
// Returns an error with descriptive message if the death date is before the birth date.
func ValidateBirthBeforeDeath(birth, death *Date) error {
	if birth == nil || death == nil {
		return nil // Can't validate without both dates
	}

	if death.IsBefore(birth) {
		return fmt.Errorf("death date (%s) is before birth date (%s)",
			death.Original, birth.Original)
	}

	return nil
}

// ValidateParentChildDates validates that a parent was at least minParentAge years old
// when a child was born. Returns nil if either date is nil, or if the parent meets
// the minimum age requirement. Returns an error if the parent would have been too young.
//
// The minParentAge parameter specifies the minimum biological age for parenthood.
// A typical value is 12 years (biological minimum).
func ValidateParentChildDates(parentBirth, childBirth *Date, minParentAge int) error {
	if parentBirth == nil || childBirth == nil {
		return nil // Can't validate without both dates
	}

	years, _, err := YearsBetween(parentBirth, childBirth)
	if err != nil {
		return nil // Insufficient data to validate
	}

	if years < minParentAge {
		return fmt.Errorf("parent would have been %d years old at child's birth (minimum: %d)",
			years, minParentAge)
	}

	return nil
}

// ValidateMarriageDates validates that both spouses were at least minMarriageAge years old
// at the time of marriage. Returns nil if there is insufficient date data to validate.
// Returns an error if either spouse would have been too young at the marriage date.
//
// The minMarriageAge parameter specifies the minimum age for marriage.
// A typical value is 12 years (historical minimum in some cultures).
func ValidateMarriageDates(marriage, spouseBirth1, spouseBirth2 *Date, minMarriageAge int) error {
	if marriage == nil {
		return nil // Can't validate without marriage date
	}

	// Validate first spouse if birth date is available
	if spouseBirth1 != nil {
		years, _, err := YearsBetween(spouseBirth1, marriage)
		if err == nil && years < minMarriageAge {
			return fmt.Errorf("first spouse would have been %d years old at marriage (minimum: %d)",
				years, minMarriageAge)
		}
	}

	// Validate second spouse if birth date is available
	if spouseBirth2 != nil {
		years, _, err := YearsBetween(spouseBirth2, marriage)
		if err == nil && years < minMarriageAge {
			return fmt.Errorf("second spouse would have been %d years old at marriage (minimum: %d)",
				years, minMarriageAge)
		}
	}

	return nil
}
