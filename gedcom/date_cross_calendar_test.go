package gedcom

import "testing"

// Regression tests for #618: YearsBetween counts dates in different calendars
// on one scale, so a Hebrew birth against a Gregorian death is a plausible age.

func TestYearsBetween_CrossCalendar(t *testing.T) {
	tests := []struct {
		name         string
		date1, date2 string
		wantYears    int
		wantExact    bool
	}{
		{"Hebrew birth to Gregorian death", "@#DHEBREW@ 1 TSH 5660", "1 JAN 1950", 50, true},
		{"Gregorian death to Hebrew birth", "1 JAN 1950", "@#DHEBREW@ 1 TSH 5660", 50, true},
		{"Hebrew year-only birth", "@#DHEBREW@ 5710", "1950", 1, false},
		{"Julian birth before Gregorian anniversary", "@#DJULIAN@ 1 MAR 1700", "5 MAR 1750", 49, true},
		{"Julian birth on Gregorian anniversary", "@#DJULIAN@ 1 MAR 1700", "12 MAR 1750", 50, true},
		{"Julian dual date read in New Style", "@#DJULIAN@ 10 FEB 1700/01", "21 FEB 1750", 49, true},
		{"French Republican birth", "@#DFRENCH R@ 1 VEND 1", "1 JAN 1850", 57, true},
		{"French Republican year-only birth", "@#DFRENCH R@ 10", "1850", 49, false},
		{"modified Hebrew birth", "ABT @#DHEBREW@ 5660", "1950", 51, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			years, exact, err := YearsBetween(mustParse(t, tt.date1), mustParse(t, tt.date2))
			if err != nil {
				t.Fatalf("YearsBetween(%q, %q) error: %v", tt.date1, tt.date2, err)
			}
			if years != tt.wantYears || exact != tt.wantExact {
				t.Errorf("YearsBetween(%q, %q) = %d, %v; want %d, %v",
					tt.date1, tt.date2, years, exact, tt.wantYears, tt.wantExact)
			}
		})
	}
}

// Dates in the same calendar keep counting as before.
func TestYearsBetween_SameCalendarUnchanged(t *testing.T) {
	tests := []struct {
		name         string
		date1, date2 string
		wantYears    int
		wantExact    bool
	}{
		{"two Gregorian dates", "15 JUN 1900", "14 JUN 1950", 49, true},
		{"two Hebrew years", "@#DHEBREW@ 5660", "@#DHEBREW@ 5710", 50, false},
		{"two Julian dates", "@#DJULIAN@ 1 MAR 1700", "@#DJULIAN@ 5 MAR 1750", 50, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			years, exact, err := YearsBetween(mustParse(t, tt.date1), mustParse(t, tt.date2))
			if err != nil {
				t.Fatalf("YearsBetween(%q, %q) error: %v", tt.date1, tt.date2, err)
			}
			if years != tt.wantYears || exact != tt.wantExact {
				t.Errorf("YearsBetween(%q, %q) = %d, %v; want %d, %v",
					tt.date1, tt.date2, years, exact, tt.wantYears, tt.wantExact)
			}
		})
	}
}

// A date whose calendar cannot be converted gives an error, not a raw year
// difference.
func TestYearsBetween_CrossCalendarUnconvertible(t *testing.T) {
	unknown := &Date{Year: 1900, Calendar: Calendar(99)}
	if _, _, err := YearsBetween(unknown, mustParse(t, "1950")); err == nil {
		t.Error("YearsBetween(unknown calendar, Gregorian) error = nil, want error")
	}
	if _, _, err := YearsBetween(mustParse(t, "1950"), unknown); err == nil {
		t.Error("YearsBetween(Gregorian, unknown calendar) error = nil, want error")
	}
}
