package gedcom

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateBirthBeforeDeath(t *testing.T) {
	tests := []struct {
		name      string
		birth     *Date
		death     *Date
		wantError bool
		errMsg    string
	}{
		{
			name:      "valid: birth before death",
			birth:     mustParseDate("15 MAR 1850"),
			death:     mustParseDate("20 DEC 1920"),
			wantError: false,
		},
		{
			name:      "valid: birth equals death (same day)",
			birth:     mustParseDate("1 JAN 1850"),
			death:     mustParseDate("1 JAN 1850"),
			wantError: false,
		},
		{
			name:      "valid: partial dates (years only)",
			birth:     mustParseDate("1850"),
			death:     mustParseDate("1920"),
			wantError: false,
		},
		{
			name:      "valid: partial dates (month and year)",
			birth:     mustParseDate("MAR 1850"),
			death:     mustParseDate("DEC 1920"),
			wantError: false,
		},
		{
			name:      "invalid: death before birth",
			birth:     mustParseDate("15 MAR 1920"),
			death:     mustParseDate("20 DEC 1850"),
			wantError: true,
			errMsg:    "death date",
		},
		{
			name:      "valid: nil birth date",
			birth:     nil,
			death:     mustParseDate("20 DEC 1920"),
			wantError: false,
		},
		{
			name:      "valid: nil death date",
			birth:     mustParseDate("15 MAR 1850"),
			death:     nil,
			wantError: false,
		},
		{
			name:      "valid: both nil",
			birth:     nil,
			death:     nil,
			wantError: false,
		},
		{
			name:      "valid: partial dates edge case",
			birth:     mustParseDate("1850"),
			death:     mustParseDate("JAN 1851"),
			wantError: false,
		},
		{
			name:      "invalid: partial dates (same year, death month before birth month)",
			birth:     mustParseDate("DEC 1850"),
			death:     mustParseDate("JAN 1850"),
			wantError: true,
			errMsg:    "death date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBirthBeforeDeath(tt.birth, tt.death)
			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateBirthBeforeDeath() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateBirthBeforeDeath() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateBirthBeforeDeath() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateParentChildDates(t *testing.T) {
	tests := []struct {
		name         string
		parentBirth  *Date
		childBirth   *Date
		minParentAge int
		wantError    bool
		errMsg       string
	}{
		{
			name:         "valid: parent 25 years old",
			parentBirth:  mustParseDate("1850"),
			childBirth:   mustParseDate("1875"),
			minParentAge: 12,
			wantError:    false,
		},
		{
			name:         "valid: parent exactly minimum age",
			parentBirth:  mustParseDate("1850"),
			childBirth:   mustParseDate("1862"),
			minParentAge: 12,
			wantError:    false,
		},
		{
			name:         "valid: parent 13 years old (above minimum)",
			parentBirth:  mustParseDate("15 JAN 1850"),
			childBirth:   mustParseDate("20 FEB 1863"),
			minParentAge: 12,
			wantError:    false,
		},
		{
			name:         "invalid: parent too young (11 years)",
			parentBirth:  mustParseDate("1850"),
			childBirth:   mustParseDate("1861"),
			minParentAge: 12,
			wantError:    true,
			errMsg:       "11 years old",
		},
		{
			name:         "invalid: parent 5 years old",
			parentBirth:  mustParseDate("1850"),
			childBirth:   mustParseDate("1855"),
			minParentAge: 12,
			wantError:    true,
			errMsg:       "5 years old",
		},
		{
			name:         "valid: nil parent birth date",
			parentBirth:  nil,
			childBirth:   mustParseDate("1875"),
			minParentAge: 12,
			wantError:    false,
		},
		{
			name:         "valid: nil child birth date",
			parentBirth:  mustParseDate("1850"),
			childBirth:   nil,
			minParentAge: 12,
			wantError:    false,
		},
		{
			name:         "valid: both nil",
			parentBirth:  nil,
			childBirth:   nil,
			minParentAge: 12,
			wantError:    false,
		},
		{
			name:         "valid: partial dates",
			parentBirth:  mustParseDate("MAR 1850"),
			childBirth:   mustParseDate("DEC 1875"),
			minParentAge: 12,
			wantError:    false,
		},
		{
			name:         "valid: different minimum age (15 years)",
			parentBirth:  mustParseDate("1850"),
			childBirth:   mustParseDate("1865"),
			minParentAge: 15,
			wantError:    false,
		},
		{
			name:         "invalid: different minimum age (15 years, parent 14)",
			parentBirth:  mustParseDate("1850"),
			childBirth:   mustParseDate("1864"),
			minParentAge: 15,
			wantError:    true,
			errMsg:       "14 years old",
		},
		{
			// #817: the absolute year difference read this as a parent of 5.
			name:         "invalid: child born before parent",
			parentBirth:  mustParseDate("1910"),
			childBirth:   mustParseDate("1905"),
			minParentAge: 12,
			wantError:    true,
			errMsg:       "child's birth date (1905) is before parent's birth date (1910)",
		},
		{
			name:         "invalid: year-only dates give an age range",
			parentBirth:  mustParseDate("1850"),
			childBirth:   mustParseDate("1861"),
			minParentAge: 12,
			wantError:    true,
			errMsg:       "parent would have been at most 11 years old at child's birth (minimum: 12)",
		},
		{
			name:         "invalid: complete dates give an exact age",
			parentBirth:  mustParseDate("15 JAN 1850"),
			childBirth:   mustParseDate("14 JAN 1862"),
			minParentAge: 12,
			wantError:    true,
			errMsg:       "parent would have been 11 years old at child's birth (minimum: 12)",
		},
		{
			// The first-day reading counted 12 years; the parent was at most 11.
			name:         "invalid: month dates read over their whole month",
			parentBirth:  mustParseDate("DEC 1850"),
			childBirth:   mustParseDate("JAN 1862"),
			minParentAge: 12,
			wantError:    true,
			errMsg:       "parent would have been 11 years old",
		},
		{
			name:         "valid: open-ended parent birth may be old enough",
			parentBirth:  mustParseDate("BEF 1850"),
			childBirth:   mustParseDate("1855"),
			minParentAge: 12,
			wantError:    false,
		},
		{
			name:         "valid: child birth may fall before parent birth",
			parentBirth:  mustParseDate("1850"),
			childBirth:   mustParseDate("MAR 1850"),
			minParentAge: 0,
			wantError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateParentChildDates(tt.parentBirth, tt.childBirth, tt.minParentAge)
			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateParentChildDates() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateParentChildDates() error = %v, want error containing %q", err, tt.errMsg)
				}
				wantKind := ErrBelowMinimumAge
				if strings.Contains(tt.errMsg, "is before") {
					wantKind = ErrEventBeforeBirth
				}
				if err != nil && !errors.Is(err, wantKind) {
					t.Errorf("ValidateParentChildDates() error = %v, want errors.Is %v", err, wantKind)
				}
			} else if err != nil {
				t.Errorf("ValidateParentChildDates() unexpected error = %v", err)
			}
		})
	}
}

func TestValidateMarriageDates(t *testing.T) {
	tests := []struct {
		name           string
		marriage       *Date
		spouseBirth1   *Date
		spouseBirth2   *Date
		minMarriageAge int
		wantError      bool
		errMsg         string
	}{
		{
			name:           "valid: both spouses old enough",
			marriage:       mustParseDate("1875"),
			spouseBirth1:   mustParseDate("1850"),
			spouseBirth2:   mustParseDate("1852"),
			minMarriageAge: 12,
			wantError:      false,
		},
		{
			name:           "valid: first spouse exactly minimum age",
			marriage:       mustParseDate("1862"),
			spouseBirth1:   mustParseDate("1850"),
			spouseBirth2:   mustParseDate("1848"),
			minMarriageAge: 12,
			wantError:      false,
		},
		{
			name:           "valid: second spouse exactly minimum age",
			marriage:       mustParseDate("1864"),
			spouseBirth1:   mustParseDate("1848"),
			spouseBirth2:   mustParseDate("1852"),
			minMarriageAge: 12,
			wantError:      false,
		},
		{
			name:           "invalid: first spouse too young",
			marriage:       mustParseDate("1861"),
			spouseBirth1:   mustParseDate("1850"),
			spouseBirth2:   mustParseDate("1848"),
			minMarriageAge: 12,
			wantError:      true,
			errMsg:         "first spouse",
		},
		{
			name:           "invalid: second spouse too young",
			marriage:       mustParseDate("1861"),
			spouseBirth1:   mustParseDate("1848"),
			spouseBirth2:   mustParseDate("1850"),
			minMarriageAge: 12,
			wantError:      true,
			errMsg:         "second spouse",
		},
		{
			name:           "invalid: both spouses too young",
			marriage:       mustParseDate("1860"),
			spouseBirth1:   mustParseDate("1850"),
			spouseBirth2:   mustParseDate("1851"),
			minMarriageAge: 12,
			wantError:      true,
			errMsg:         "spouse",
		},
		{
			name:           "valid: nil marriage date",
			marriage:       nil,
			spouseBirth1:   mustParseDate("1850"),
			spouseBirth2:   mustParseDate("1852"),
			minMarriageAge: 12,
			wantError:      false,
		},
		{
			name:           "valid: nil first spouse birth",
			marriage:       mustParseDate("1875"),
			spouseBirth1:   nil,
			spouseBirth2:   mustParseDate("1852"),
			minMarriageAge: 12,
			wantError:      false,
		},
		{
			name:           "valid: nil second spouse birth",
			marriage:       mustParseDate("1875"),
			spouseBirth1:   mustParseDate("1850"),
			spouseBirth2:   nil,
			minMarriageAge: 12,
			wantError:      false,
		},
		{
			name:           "valid: all dates nil",
			marriage:       nil,
			spouseBirth1:   nil,
			spouseBirth2:   nil,
			minMarriageAge: 12,
			wantError:      false,
		},
		{
			name:           "valid: partial dates",
			marriage:       mustParseDate("JUN 1875"),
			spouseBirth1:   mustParseDate("MAR 1850"),
			spouseBirth2:   mustParseDate("1852"),
			minMarriageAge: 12,
			wantError:      false,
		},
		{
			name:           "valid: only first spouse birth known",
			marriage:       mustParseDate("1875"),
			spouseBirth1:   mustParseDate("1850"),
			spouseBirth2:   nil,
			minMarriageAge: 12,
			wantError:      false,
		},
		{
			name:           "invalid: only first spouse birth known, too young",
			marriage:       mustParseDate("1860"),
			spouseBirth1:   mustParseDate("1850"),
			spouseBirth2:   nil,
			minMarriageAge: 12,
			wantError:      true,
			errMsg:         "first spouse",
		},
		{
			name:           "valid: different minimum age (16 years)",
			marriage:       mustParseDate("1875"),
			spouseBirth1:   mustParseDate("1850"),
			spouseBirth2:   mustParseDate("1852"),
			minMarriageAge: 16,
			wantError:      false,
		},
		{
			name:           "invalid: different minimum age (16 years, first spouse 15)",
			marriage:       mustParseDate("1865"),
			spouseBirth1:   mustParseDate("1850"),
			spouseBirth2:   mustParseDate("1848"),
			minMarriageAge: 16,
			wantError:      true,
			errMsg:         "first spouse",
		},
		{
			// #817: the absolute year difference read this as a spouse of 5.
			name:           "invalid: marriage before first spouse's birth",
			marriage:       mustParseDate("1905"),
			spouseBirth1:   mustParseDate("1910"),
			spouseBirth2:   mustParseDate("1880"),
			minMarriageAge: 12,
			wantError:      true,
			errMsg:         "marriage date (1905) is before first spouse's birth date (1910)",
		},
		{
			name:           "invalid: marriage before second spouse's birth",
			marriage:       mustParseDate("1905"),
			spouseBirth1:   mustParseDate("1880"),
			spouseBirth2:   mustParseDate("1910"),
			minMarriageAge: 12,
			wantError:      true,
			errMsg:         "marriage date (1905) is before second spouse's birth date (1910)",
		},
		{
			name:           "invalid: second spouse too young, with age range",
			marriage:       mustParseDate("1861"),
			spouseBirth1:   mustParseDate("1840"),
			spouseBirth2:   mustParseDate("1850"),
			minMarriageAge: 12,
			wantError:      true,
			errMsg:         "second spouse would have been at most 11 years old at marriage (minimum: 12)",
		},
		{
			name:           "valid: unparseable spouse birth is skipped",
			marriage:       mustParseDate("1861"),
			spouseBirth1:   &Date{Original: "unknown"},
			spouseBirth2:   mustParseDate("1840"),
			minMarriageAge: 12,
			wantError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMarriageDates(tt.marriage, tt.spouseBirth1, tt.spouseBirth2, tt.minMarriageAge)
			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateMarriageDates() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateMarriageDates() error = %v, want error containing %q", err, tt.errMsg)
				}
				wantKind := ErrBelowMinimumAge
				if strings.Contains(tt.errMsg, "is before") {
					wantKind = ErrEventBeforeBirth
				}
				if err != nil && !errors.Is(err, wantKind) {
					t.Errorf("ValidateMarriageDates() error = %v, want errors.Is %v", err, wantKind)
				}
			} else if err != nil {
				t.Errorf("ValidateMarriageDates() unexpected error = %v", err)
			}
		})
	}
}

// mustParseDate is a helper that panics on parse error (for test data only)
func mustParseDate(s string) *Date {
	d, err := ParseDate(s)
	if err != nil {
		panic("mustParseDate: " + err.Error())
	}
	return d
}
