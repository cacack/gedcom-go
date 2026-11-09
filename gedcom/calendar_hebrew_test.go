package gedcom

import "testing"

// hebrewNewYearDaysReference is an independent implementation of the days
// from the Hebrew epoch to 1 Tishrei of year, following Dershowitz &
// Reingold, "Calendrical Calculations" (hebrew-calendar-elapsed-days plus
// hebrew-new-year-delay). It shares no code with hebrewDelay and is used to
// check it over a long range of years (#702).
func hebrewNewYearDaysReference(year int) int {
	elapsed := func(y int) int {
		months := (235*y - 234) / 19
		parts := 12084 + 13753*months
		days := 29*months + parts/25920
		if (3*(days+1))%7 < 3 {
			days++
		}
		return days
	}
	delay := func(y int) int {
		ny0, ny1, ny2 := elapsed(y-1), elapsed(y), elapsed(y+1)
		switch {
		case ny2-ny1 == 356:
			return 2
		case ny1-ny0 == 382:
			return 1
		}
		return 0
	}
	return elapsed(year) + delay(year) - (elapsed(1) + delay(1))
}

// TestHebrewDelayMatchesReference compares hebrewDelay with the reference
// for every year 1-8000. Before #702 the dehiyot tested the wrong weekday,
// and 488 of these years differed.
func TestHebrewDelayMatchesReference(t *testing.T) {
	for year := 1; year <= 8000; year++ {
		if got, want := hebrewDelay(year), hebrewNewYearDaysReference(year); got != want {
			t.Fatalf("hebrewDelay(%d) = %d, want %d", year, got, want)
		}
	}
}

// TestHebrewRoshHashanahKnownDates pins 1 Tishrei against published Gregorian
// dates of Rosh Hashanah, mixing years the #702 bug shifted (marked) with
// years it did not.
func TestHebrewRoshHashanahKnownDates(t *testing.T) {
	tests := []struct {
		year    int
		y, m, d int // Gregorian date of 1 Tishrei
		shifted bool
	}{
		{5661, 1900, 9, 24, true},
		{5700, 1939, 9, 14, false},
		{5718, 1957, 9, 26, true},
		{5739, 1978, 10, 2, true},
		{5745, 1984, 9, 27, true},
		{5749, 1988, 9, 12, false},
		{5759, 1998, 9, 21, true},
		{5760, 1999, 9, 11, false},
		{5776, 2015, 9, 14, false},
		{5780, 2019, 9, 30, false},
		{5781, 2020, 9, 19, false},
		{5782, 2021, 9, 7, false},
		{5784, 2023, 9, 16, false},
		{5785, 2024, 10, 3, false},
		{5786, 2025, 9, 23, false},
		{5787, 2026, 9, 12, false},
		{5789, 2028, 9, 21, true},
		{5796, 2035, 10, 4, true},
	}
	for _, tt := range tests {
		want := GregorianToJDN(tt.y, tt.m, tt.d)
		if got := HebrewToJDN(tt.year, 1, 1); got != want {
			t.Errorf("HebrewToJDN(%d, 1, 1) = %d, want %d (%04d-%02d-%02d, shifted before fix: %v)",
				tt.year, got, want, tt.y, tt.m, tt.d, tt.shifted)
		}
	}
}

// TestHebrewDaysInYearIsValidLength asserts that every year 1-8000 has one
// of the six lengths the calendar allows. Before #702, 5745 had 356 days.
func TestHebrewDaysInYearIsValidLength(t *testing.T) {
	for year := 1; year <= 8000; year++ {
		n := HebrewDaysInYear(year)
		switch n {
		case 353, 354, 355:
			if IsHebrewLeapYear(year) {
				t.Fatalf("HebrewDaysInYear(%d) = %d for a leap year", year, n)
			}
		case 383, 384, 385:
			if !IsHebrewLeapYear(year) {
				t.Fatalf("HebrewDaysInYear(%d) = %d for a common year", year, n)
			}
		default:
			t.Fatalf("HebrewDaysInYear(%d) = %d, not a Hebrew year length", year, n)
		}
	}
}

// TestHebrewJDNRoundTripAllDates round-trips every valid date in 5000-6000
// and checks that consecutive dates map to consecutive JDNs.
func TestHebrewJDNRoundTripAllDates(t *testing.T) {
	prev := HebrewToJDN(5000, 1, 1) - 1
	for year := 5000; year <= 6000; year++ {
		// GEDCOM numbers the months 1-13 with Adar II as 7, so in a common
		// year month 7 has no days and is skipped.
		for month := 1; month <= 13; month++ {
			days := HebrewDaysInMonth(year, month)
			for day := 1; day <= days; day++ {
				jdn := HebrewToJDN(year, month, day)
				if jdn != prev+1 {
					t.Fatalf("HebrewToJDN(%d, %d, %d) = %d, want %d (not consecutive)", year, month, day, jdn, prev+1)
				}
				prev = jdn
				y, m, d := JDNToHebrew(jdn)
				if y != year || m != month || d != day {
					t.Fatalf("JDNToHebrew(%d) = (%d, %d, %d), want (%d, %d, %d)", jdn, y, m, d, year, month, day)
				}
			}
		}
	}
}
