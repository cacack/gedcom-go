package gedcom

import "fmt"

// ValidateBirthBeforeDeath checks if a birth date is before or equal to a death date.
// Returns nil if the birth date is before or equal to the death date, or if either date is nil.
// Returns an error with descriptive message if the death date is before the birth date.
//
// The order is IsBefore's, which reads a partial date as its first day and
// ignores modifiers, so imprecise dates can fail without proving anything;
// see Date.Bounds for what the dates prove.
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
// when a child was born. Returns nil if either date is nil or has no Bounds, or if
// the parent may have met the minimum age. Returns an error if the parent was too
// young, or if the child's birth is provably before the parent's.
//
// The age is AgeAt's, the rule the validator's UNREASONABLE_PARENT_AGE check
// applies: a partial date spans its whole period, and the parent is too young
// only when even the oldest age the dates allow is below the minimum. A parent
// born in 1850 and a child born in 1861 give "at most 11 years old".
//
// The minParentAge parameter specifies the minimum biological age for parenthood.
// A typical value is 12 years (biological minimum).
func ValidateParentChildDates(parentBirth, childBirth *Date, minParentAge int) error {
	if parentBirth == nil || childBirth == nil {
		return nil // Can't validate without both dates
	}
	return checkMinimumAge("parent", "child's birth", parentBirth, childBirth, minParentAge)
}

// ValidateMarriageDates validates that both spouses were at least minMarriageAge years old
// at the time of marriage. Returns nil if there is insufficient date data to validate.
// Returns an error if either spouse was too young at the marriage date, or if the
// marriage is provably before that spouse's birth.
//
// Ages are computed as ValidateParentChildDates computes them, with AgeAt: a
// spouse is too young only when even the oldest age the dates allow is below
// the minimum.
//
// The minMarriageAge parameter specifies the minimum age for marriage.
// A typical value is 12 years (historical minimum in some cultures).
func ValidateMarriageDates(marriage, spouseBirth1, spouseBirth2 *Date, minMarriageAge int) error {
	if marriage == nil {
		return nil // Can't validate without marriage date
	}

	if spouseBirth1 != nil {
		if err := checkMinimumAge("first spouse", "marriage", spouseBirth1, marriage, minMarriageAge); err != nil {
			return err
		}
	}

	if spouseBirth2 != nil {
		if err := checkMinimumAge("second spouse", "marriage", spouseBirth2, marriage, minMarriageAge); err != nil {
			return err
		}
	}

	return nil
}

// checkMinimumAge reports who as too young at event when even the oldest age
// AgeAt allows is below minAge, and reports an event provably before the birth.
func checkMinimumAge(who, event string, birth, eventDate *Date, minAge int) error {
	youngest, oldest, ok := AgeAt(birth, eventDate)
	if !ok {
		_, _, birthOK := birth.Bounds()
		_, _, eventOK := eventDate.Bounds()
		if birthOK && eventOK {
			return fmt.Errorf("%s date (%s) is before %s birth date (%s)",
				event, eventDate.Original, who+"'s", birth.Original)
		}
		return nil // Insufficient data to validate
	}

	if oldest < minAge {
		prefix := ""
		if youngest != oldest {
			prefix = "at most "
		}
		return fmt.Errorf("%s would have been %s%d years old at %s (minimum: %d)",
			who, prefix, oldest, event, minAge)
	}

	return nil
}
