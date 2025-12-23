package gedcom

import (
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
			} else {
				if err != nil {
					t.Errorf("ValidateParentChildDates() unexpected error = %v", err)
				}
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
			} else {
				if err != nil {
					t.Errorf("ValidateMarriageDates() unexpected error = %v", err)
				}
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
