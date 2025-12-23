package gedcom

import (
	"testing"
)

// TestGregorianToJDN tests the Gregorian to JDN conversion with known reference dates.
func TestGregorianToJDN(t *testing.T) {
	tests := []struct {
		name  string
		year  int
		month int
		day   int
		want  int
	}{
		// Modern reference dates
		{
			name:  "January 1, 2000",
			year:  2000,
			month: 1,
			day:   1,
			want:  2451545,
		},
		{
			name:  "Unix epoch (January 1, 1970)",
			year:  1970,
			month: 1,
			day:   1,
			want:  2440588,
		},
		{
			name:  "Gregorian calendar adoption (October 15, 1582)",
			year:  1582,
			month: 10,
			day:   15,
			want:  2299161,
		},

		// Historical BC dates (using astronomical year numbering)
		// Note: These are proleptic Gregorian dates, not Julian calendar dates
		{
			name:  "Ides of March, 44 BC (astronomical year -43)",
			year:  -43,
			month: 3,
			day:   15,
			want:  1705428,
		},
		{
			name:  "January 1, 1 BC (astronomical year 0)",
			year:  0,
			month: 1,
			day:   1,
			want:  1721060,
		},
		{
			name:  "January 1, 1 AD",
			year:  1,
			month: 1,
			day:   1,
			want:  1721426,
		},

		// Year boundaries
		{
			name:  "December 31, 1999",
			year:  1999,
			month: 12,
			day:   31,
			want:  2451544,
		},
		{
			name:  "January 1, 2001",
			year:  2001,
			month: 1,
			day:   1,
			want:  2451911,
		},

		// Leap year dates
		{
			name:  "February 29, 2000 (leap year)",
			year:  2000,
			month: 2,
			day:   29,
			want:  2451604,
		},
		{
			name:  "February 29, 1600 (leap year)",
			year:  1600,
			month: 2,
			day:   29,
			want:  2305507,
		},

		// Other notable dates
		{
			name:  "July 4, 1776 (US Independence)",
			year:  1776,
			month: 7,
			day:   4,
			want:  2369916,
		},
		{
			name:  "November 24, 4714 BC (JDN epoch, astronomical year -4713)",
			year:  -4713,
			month: 11,
			day:   24,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GregorianToJDN(tt.year, tt.month, tt.day)
			if got != tt.want {
				t.Errorf("GregorianToJDN(%d, %d, %d) = %d, want %d",
					tt.year, tt.month, tt.day, got, tt.want)
			}
		})
	}
}

// TestJDNToGregorian tests the JDN to Gregorian conversion with known reference dates.
func TestJDNToGregorian(t *testing.T) {
	tests := []struct {
		name      string
		jdn       int
		wantYear  int
		wantMonth int
		wantDay   int
	}{
		// Modern reference dates
		{
			name:      "JDN 2451545 (January 1, 2000)",
			jdn:       2451545,
			wantYear:  2000,
			wantMonth: 1,
			wantDay:   1,
		},
		{
			name:      "JDN 2440588 (Unix epoch, January 1, 1970)",
			jdn:       2440588,
			wantYear:  1970,
			wantMonth: 1,
			wantDay:   1,
		},
		{
			name:      "JDN 2299161 (Gregorian adoption, October 15, 1582)",
			jdn:       2299161,
			wantYear:  1582,
			wantMonth: 10,
			wantDay:   15,
		},

		// Historical BC dates (astronomical year numbering)
		// Note: These are proleptic Gregorian dates, not Julian calendar dates
		{
			name:      "JDN 1705428 (Ides of March, 44 BC)",
			jdn:       1705428,
			wantYear:  -43,
			wantMonth: 3,
			wantDay:   15,
		},
		{
			name:      "JDN 1721060 (January 1, 1 BC / astronomical year 0)",
			jdn:       1721060,
			wantYear:  0,
			wantMonth: 1,
			wantDay:   1,
		},

		// Year boundaries
		{
			name:      "JDN 2451544 (December 31, 1999)",
			jdn:       2451544,
			wantYear:  1999,
			wantMonth: 12,
			wantDay:   31,
		},
		{
			name:      "JDN 2451911 (January 1, 2001)",
			jdn:       2451911,
			wantYear:  2001,
			wantMonth: 1,
			wantDay:   1,
		},

		// Leap year dates
		{
			name:      "JDN 2451604 (February 29, 2000)",
			jdn:       2451604,
			wantYear:  2000,
			wantMonth: 2,
			wantDay:   29,
		},

		// JDN epoch
		{
			name:      "JDN 0 (November 24, 4714 BC)",
			jdn:       0,
			wantYear:  -4713,
			wantMonth: 11,
			wantDay:   24,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotYear, gotMonth, gotDay := JDNToGregorian(tt.jdn)
			if gotYear != tt.wantYear || gotMonth != tt.wantMonth || gotDay != tt.wantDay {
				t.Errorf("JDNToGregorian(%d) = (%d, %d, %d), want (%d, %d, %d)",
					tt.jdn, gotYear, gotMonth, gotDay, tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

// TestGregorianJDNRoundTrip tests that converting Gregorian -> JDN -> Gregorian
// returns the same date (verifying the functions are inverse operations).
func TestGregorianJDNRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		year  int
		month int
		day   int
	}{
		{"Modern date", 2000, 6, 15},
		{"Leap year", 2000, 2, 29},
		{"Non-leap year", 1900, 2, 28},
		{"BC date (astronomical)", -43, 3, 15},
		{"Year 0 (1 BC)", 0, 1, 1},
		{"Year 1 AD", 1, 1, 1},
		{"Recent date", 2024, 12, 23},
		{"19th century", 1850, 7, 4},
		{"16th century", 1582, 10, 15},
		{"Deep BC (astronomical)", -4713, 11, 24},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to JDN and back
			jdn := GregorianToJDN(tt.year, tt.month, tt.day)
			gotYear, gotMonth, gotDay := JDNToGregorian(jdn)

			if gotYear != tt.year || gotMonth != tt.month || gotDay != tt.day {
				t.Errorf("Round trip failed for (%d, %d, %d): got (%d, %d, %d) via JDN %d",
					tt.year, tt.month, tt.day, gotYear, gotMonth, gotDay, jdn)
			}
		})
	}
}

// TestAstronomicalYear tests the conversion from GEDCOM year/BC format to astronomical year.
func TestAstronomicalYear(t *testing.T) {
	tests := []struct {
		name string
		year int
		isBC bool
		want int
	}{
		// AD years (unchanged)
		{
			name: "2000 AD",
			year: 2000,
			isBC: false,
			want: 2000,
		},
		{
			name: "1 AD",
			year: 1,
			isBC: false,
			want: 1,
		},

		// BC years (subtract 1 and negate)
		{
			name: "1 BC",
			year: 1,
			isBC: true,
			want: 0,
		},
		{
			name: "2 BC",
			year: 2,
			isBC: true,
			want: -1,
		},
		{
			name: "44 BC",
			year: 44,
			isBC: true,
			want: -43,
		},
		{
			name: "100 BC",
			year: 100,
			isBC: true,
			want: -99,
		},
		{
			name: "4714 BC (JDN epoch)",
			year: 4714,
			isBC: true,
			want: -4713,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AstronomicalYear(tt.year, tt.isBC)
			if got != tt.want {
				t.Errorf("AstronomicalYear(%d, %v) = %d, want %d",
					tt.year, tt.isBC, got, tt.want)
			}
		})
	}
}

// TestFromAstronomicalYear tests the conversion from astronomical year back to GEDCOM format.
func TestFromAstronomicalYear(t *testing.T) {
	tests := []struct {
		name      string
		astroYear int
		wantYear  int
		wantIsBC  bool
	}{
		// AD years (positive)
		{
			name:      "2000 AD",
			astroYear: 2000,
			wantYear:  2000,
			wantIsBC:  false,
		},
		{
			name:      "1 AD",
			astroYear: 1,
			wantYear:  1,
			wantIsBC:  false,
		},

		// BC years (zero and negative)
		{
			name:      "1 BC (astronomical 0)",
			astroYear: 0,
			wantYear:  1,
			wantIsBC:  true,
		},
		{
			name:      "2 BC (astronomical -1)",
			astroYear: -1,
			wantYear:  2,
			wantIsBC:  true,
		},
		{
			name:      "44 BC (astronomical -43)",
			astroYear: -43,
			wantYear:  44,
			wantIsBC:  true,
		},
		{
			name:      "100 BC (astronomical -99)",
			astroYear: -99,
			wantYear:  100,
			wantIsBC:  true,
		},
		{
			name:      "4714 BC (JDN epoch, astronomical -4713)",
			astroYear: -4713,
			wantYear:  4714,
			wantIsBC:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotYear, gotIsBC := FromAstronomicalYear(tt.astroYear)
			if gotYear != tt.wantYear || gotIsBC != tt.wantIsBC {
				t.Errorf("FromAstronomicalYear(%d) = (%d, %v), want (%d, %v)",
					tt.astroYear, gotYear, gotIsBC, tt.wantYear, tt.wantIsBC)
			}
		})
	}
}

// TestAstronomicalYearRoundTrip tests that converting GEDCOM year -> astronomical -> GEDCOM
// returns the same values.
func TestAstronomicalYearRoundTrip(t *testing.T) {
	tests := []struct {
		year int
		isBC bool
	}{
		{2000, false},
		{1, false},
		{1, true},
		{2, true},
		{44, true},
		{100, true},
		{4714, true},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			// Convert to astronomical and back
			astro := AstronomicalYear(tt.year, tt.isBC)
			gotYear, gotIsBC := FromAstronomicalYear(astro)

			if gotYear != tt.year || gotIsBC != tt.isBC {
				t.Errorf("Round trip failed for (%d, %v): got (%d, %v) via astro %d",
					tt.year, tt.isBC, gotYear, gotIsBC, astro)
			}
		})
	}
}

// TestGEDCOMDateToJDN tests converting a GEDCOM Date to JDN using the helper functions.
func TestGEDCOMDateToJDN(t *testing.T) {
	tests := []struct {
		name     string
		year     int
		month    int
		day      int
		isBC     bool
		wantJDN  int
		wantDesc string
	}{
		{
			name:     "Modern date",
			year:     2000,
			month:    1,
			day:      1,
			isBC:     false,
			wantJDN:  2451545,
			wantDesc: "January 1, 2000 AD",
		},
		{
			name:     "Ides of March 44 BC",
			year:     44,
			month:    3,
			day:      15,
			isBC:     true,
			wantJDN:  1705428,
			wantDesc: "March 15, 44 BC",
		},
		{
			name:     "Year 1 BC",
			year:     1,
			month:    1,
			day:      1,
			isBC:     true,
			wantJDN:  1721060,
			wantDesc: "January 1, 1 BC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert GEDCOM year to astronomical year
			astroYear := AstronomicalYear(tt.year, tt.isBC)

			// Convert to JDN
			jdn := GregorianToJDN(astroYear, tt.month, tt.day)

			if jdn != tt.wantJDN {
				t.Errorf("%s: got JDN %d, want %d", tt.wantDesc, jdn, tt.wantJDN)
			}
		})
	}
}

// TestJDNToGEDCOMDate tests converting a JDN back to GEDCOM Date format.
func TestJDNToGEDCOMDate(t *testing.T) {
	tests := []struct {
		name      string
		jdn       int
		wantYear  int
		wantMonth int
		wantDay   int
		wantIsBC  bool
	}{
		{
			name:      "Modern date",
			jdn:       2451545,
			wantYear:  2000,
			wantMonth: 1,
			wantDay:   1,
			wantIsBC:  false,
		},
		{
			name:      "Ides of March 44 BC",
			jdn:       1705428,
			wantYear:  44,
			wantMonth: 3,
			wantDay:   15,
			wantIsBC:  true,
		},
		{
			name:      "Year 1 BC",
			jdn:       1721060,
			wantYear:  1,
			wantMonth: 1,
			wantDay:   1,
			wantIsBC:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert JDN to astronomical year
			astroYear, month, day := JDNToGregorian(tt.jdn)

			// Convert astronomical year back to GEDCOM format
			year, isBC := FromAstronomicalYear(astroYear)

			if year != tt.wantYear || month != tt.wantMonth || day != tt.wantDay || isBC != tt.wantIsBC {
				t.Errorf("JDN %d: got (%d %s, %d, %d), want (%d %s, %d, %d)",
					tt.jdn,
					year, bcString(isBC), month, day,
					tt.wantYear, bcString(tt.wantIsBC), tt.wantMonth, tt.wantDay)
			}
		})
	}
}

// Helper function to format BC/AD string for error messages.
func bcString(isBC bool) string {
	if isBC {
		return "BC"
	}
	return "AD"
}

// TestJulianToJDN tests the Julian to JDN conversion with known reference dates.
func TestJulianToJDN(t *testing.T) {
	tests := []struct {
		name  string
		year  int
		month int
		day   int
		want  int
	}{
		// Historical reference dates
		{
			name:  "October 4, 1582 (last Julian day in most countries)",
			year:  1582,
			month: 10,
			day:   4,
			want:  2299160,
		},
		{
			name:  "Ides of March, 44 BC (astronomical year -43)",
			year:  -43,
			month: 3,
			day:   15,
			want:  1705426,
		},
		{
			name:  "January 1, 1 AD",
			year:  1,
			month: 1,
			day:   1,
			want:  1721424,
		},
		{
			name:  "January 1, 1 BC (astronomical year 0)",
			year:  0,
			month: 1,
			day:   1,
			want:  1721058,
		},

		// Test leap year behavior (every 4 years, no century exception)
		{
			name:  "February 29, 100 AD (Julian leap year)",
			year:  100,
			month: 2,
			day:   29,
			want:  1757642,
		},
		{
			name:  "February 29, 200 AD (Julian leap year)",
			year:  200,
			month: 2,
			day:   29,
			want:  1794167,
		},
		{
			name:  "February 29, 1900 (Julian leap year)",
			year:  1900,
			month: 2,
			day:   29,
			want:  2415092,
		},

		// Modern dates (for comparison with Gregorian)
		{
			name:  "January 1, 2000 (Julian calendar)",
			year:  2000,
			month: 1,
			day:   1,
			want:  2451558, // 13 days ahead of Gregorian
		},

		// Year boundaries
		{
			name:  "December 31, 1581",
			year:  1581,
			month: 12,
			day:   31,
			want:  2298883,
		},
		{
			name:  "January 1, 1583",
			year:  1583,
			month: 1,
			day:   1,
			want:  2299249,
		},

		// Deep BC dates
		{
			name:  "January 1, 4713 BC (JDN epoch, astronomical year -4712)",
			year:  -4712,
			month: 1,
			day:   1,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JulianToJDN(tt.year, tt.month, tt.day)
			if got != tt.want {
				t.Errorf("JulianToJDN(%d, %d, %d) = %d, want %d",
					tt.year, tt.month, tt.day, got, tt.want)
			}
		})
	}
}

// TestJDNToJulian tests the JDN to Julian conversion with known reference dates.
func TestJDNToJulian(t *testing.T) {
	tests := []struct {
		name      string
		jdn       int
		wantYear  int
		wantMonth int
		wantDay   int
	}{
		// Historical reference dates
		{
			name:      "JDN 2299160 (October 4, 1582)",
			jdn:       2299160,
			wantYear:  1582,
			wantMonth: 10,
			wantDay:   4,
		},
		{
			name:      "JDN 1705426 (Ides of March, 44 BC)",
			jdn:       1705426,
			wantYear:  -43,
			wantMonth: 3,
			wantDay:   15,
		},
		{
			name:      "JDN 1721424 (January 1, 1 AD)",
			jdn:       1721424,
			wantYear:  1,
			wantMonth: 1,
			wantDay:   1,
		},
		{
			name:      "JDN 1721058 (January 1, 1 BC / astronomical year 0)",
			jdn:       1721058,
			wantYear:  0,
			wantMonth: 1,
			wantDay:   1,
		},

		// Test leap year dates
		{
			name:      "JDN 1757642 (February 29, 100 AD)",
			jdn:       1757642,
			wantYear:  100,
			wantMonth: 2,
			wantDay:   29,
		},

		// Modern date
		{
			name:      "JDN 2451558 (January 1, 2000 Julian)",
			jdn:       2451558,
			wantYear:  2000,
			wantMonth: 1,
			wantDay:   1,
		},

		// Year boundaries
		{
			name:      "JDN 2298883 (December 31, 1581)",
			jdn:       2298883,
			wantYear:  1581,
			wantMonth: 12,
			wantDay:   31,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotYear, gotMonth, gotDay := JDNToJulian(tt.jdn)
			if gotYear != tt.wantYear || gotMonth != tt.wantMonth || gotDay != tt.wantDay {
				t.Errorf("JDNToJulian(%d) = (%d, %d, %d), want (%d, %d, %d)",
					tt.jdn, gotYear, gotMonth, gotDay, tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

// TestJulianJDNRoundTrip tests that converting Julian -> JDN -> Julian
// returns the same date (verifying the functions are inverse operations).
func TestJulianJDNRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		year  int
		month int
		day   int
	}{
		{"October 4, 1582", 1582, 10, 4},
		{"Ides of March 44 BC", -43, 3, 15},
		{"January 1, 1 AD", 1, 1, 1},
		{"Year 0 (1 BC)", 0, 1, 1},
		{"February 29, 100 AD", 100, 2, 29},
		{"February 29, 1900", 1900, 2, 29},
		{"January 1, 2000", 2000, 1, 1},
		{"December 31, 1999", 1999, 12, 31},
		{"July 4, 1776", 1776, 7, 4},
		{"Deep BC", -4712, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to JDN and back
			jdn := JulianToJDN(tt.year, tt.month, tt.day)
			gotYear, gotMonth, gotDay := JDNToJulian(jdn)

			if gotYear != tt.year || gotMonth != tt.month || gotDay != tt.day {
				t.Errorf("Round trip failed for (%d, %d, %d): got (%d, %d, %d) via JDN %d",
					tt.year, tt.month, tt.day, gotYear, gotMonth, gotDay, jdn)
			}
		})
	}
}

// TestJulianVsGregorian tests the difference between Julian and Gregorian calendars.
// This verifies that the two calendar systems produce different JDN values for the same date,
// and demonstrates the historical calendar transition.
func TestJulianVsGregorian(t *testing.T) {
	tests := []struct {
		name            string
		year            int
		month           int
		day             int
		wantDaysDiff    int // Expected difference in days (Julian JDN - Gregorian JDN)
		descriptionDiff string
	}{
		{
			name:            "October 1582 transition",
			year:            1582,
			month:           10,
			day:             4,
			wantDaysDiff:    10, // Oct 4 Julian = JDN 2299160, Oct 4 Gregorian = JDN 2299150
			descriptionDiff: "Oct 4, 1582 Julian is 10 days ahead of same date in Gregorian",
		},
		{
			name:            "Ides of March 44 BC",
			year:            -43,
			month:           3,
			day:             15,
			wantDaysDiff:    -2, // Julian = 1705426, Gregorian = 1705428 (2 days earlier)
			descriptionDiff: "In 44 BC, Julian calendar is 2 days behind proleptic Gregorian",
		},
		{
			name:            "January 1, 1 AD",
			year:            1,
			month:           1,
			day:             1,
			wantDaysDiff:    -2, // Julian = 1721424, Gregorian = 1721426
			descriptionDiff: "At year 1 AD, Julian is 2 days behind Gregorian",
		},
		{
			name:            "Modern era (2000)",
			year:            2000,
			month:           1,
			day:             1,
			wantDaysDiff:    13, // Julian = 2451558, Gregorian = 2451545
			descriptionDiff: "By year 2000, Julian calendar is 13 days ahead of Gregorian",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			julianJDN := JulianToJDN(tt.year, tt.month, tt.day)
			gregorianJDN := GregorianToJDN(tt.year, tt.month, tt.day)
			diff := julianJDN - gregorianJDN

			if diff != tt.wantDaysDiff {
				t.Errorf("%s: Julian-Gregorian difference = %d days, want %d days\n"+
					"Julian JDN: %d, Gregorian JDN: %d\n"+
					"Context: %s",
					tt.name, diff, tt.wantDaysDiff,
					julianJDN, gregorianJDN, tt.descriptionDiff)
			}
		})
	}
}

// TestCalendarTransition tests the specific dates of the Gregorian calendar transition.
// October 4, 1582 (Julian) was followed by October 15, 1582 (Gregorian),
// skipping 10 days. Both dates should have JDN values that differ by exactly 1.
func TestCalendarTransition(t *testing.T) {
	// Last day of Julian calendar in most countries
	lastJulianJDN := JulianToJDN(1582, 10, 4)

	// First day of Gregorian calendar
	firstGregorianJDN := GregorianToJDN(1582, 10, 15)

	// These should be consecutive JDN values
	if firstGregorianJDN-lastJulianJDN != 1 {
		t.Errorf("Calendar transition error: Oct 4, 1582 (Julian) JDN=%d should be followed by Oct 15, 1582 (Gregorian) JDN=%d, diff=%d",
			lastJulianJDN, firstGregorianJDN, firstGregorianJDN-lastJulianJDN)
	}

	// Verify the actual JDN values
	if lastJulianJDN != 2299160 {
		t.Errorf("Oct 4, 1582 (Julian) JDN = %d, want 2299160", lastJulianJDN)
	}
	if firstGregorianJDN != 2299161 {
		t.Errorf("Oct 15, 1582 (Gregorian) JDN = %d, want 2299161", firstGregorianJDN)
	}
}

// TestGEDCOMJulianDateToJDN tests converting a GEDCOM Julian Date to JDN.
func TestGEDCOMJulianDateToJDN(t *testing.T) {
	tests := []struct {
		name     string
		year     int
		month    int
		day      int
		isBC     bool
		wantJDN  int
		wantDesc string
	}{
		{
			name:     "October 4, 1582",
			year:     1582,
			month:    10,
			day:      4,
			isBC:     false,
			wantJDN:  2299160,
			wantDesc: "Last day of Julian calendar",
		},
		{
			name:     "Ides of March 44 BC",
			year:     44,
			month:    3,
			day:      15,
			isBC:     true,
			wantJDN:  1705426,
			wantDesc: "Julius Caesar assassination",
		},
		{
			name:     "January 1, 1 AD",
			year:     1,
			month:    1,
			day:      1,
			isBC:     false,
			wantJDN:  1721424,
			wantDesc: "Start of AD era",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert GEDCOM year to astronomical year
			astroYear := AstronomicalYear(tt.year, tt.isBC)

			// Convert to JDN using Julian calendar
			jdn := JulianToJDN(astroYear, tt.month, tt.day)

			if jdn != tt.wantJDN {
				t.Errorf("%s: got JDN %d, want %d", tt.wantDesc, jdn, tt.wantJDN)
			}
		})
	}
}

// TestIsFrenchLeapYear tests the French Republican leap year calculation.
func TestIsFrenchLeapYear(t *testing.T) {
	tests := []struct {
		year int
		want bool
	}{
		// Based on actual calendar: FR year N is leap if Gregorian year (1792+N) is leap
		{1, false},  // Year 1 (Sep 22, 1792 - Sep 21, 1793), Gregorian 1793 not leap
		{2, false},  // Year 2 (Sep 22, 1793 - Sep 21, 1794), Gregorian 1794 not leap
		{3, false},  // Year 3 (Sep 22, 1794 - Sep 21, 1795), Gregorian 1795 not leap
		{4, true},   // Year 4 (Sep 22, 1795 - Sep 21, 1796), Gregorian 1796 IS leap
		{5, false},  // Year 5 (Sep 22, 1796 - Sep 21, 1797), Gregorian 1797 not leap
		{6, false},  // Year 6 (Sep 22, 1797 - Sep 21, 1798), Gregorian 1798 not leap
		{7, false},  // Year 7 (Sep 22, 1798 - Sep 21, 1799), Gregorian 1799 not leap
		{8, false},  // Year 8 (Sep 22, 1799 - Sep 21, 1800), Gregorian 1800 not leap (century)
		{9, false},  // Year 9 (Sep 22, 1800 - Sep 21, 1801), Gregorian 1801 not leap
		{10, false}, // Year 10 (Sep 22, 1801 - Sep 21, 1802), Gregorian 1802 not leap
		{11, false}, // Year 11 (Sep 22, 1802 - Sep 21, 1803), Gregorian 1803 not leap
		{12, true},  // Year 12 (Sep 22, 1803 - Sep 21, 1804), Gregorian 1804 IS leap
		{13, false}, // Year 13 (Sep 22, 1804 - Sep 21, 1805), Gregorian 1805 not leap
		{14, false}, // Year 14 (Sep 22, 1805 - Sep 21, 1806), Gregorian 1806 not leap

		// Test pattern continues beyond historical range
		{15, false}, // Year 15 (Sep 22, 1806 - Sep 21, 1807), Gregorian 1807 not leap
		{16, true},  // Year 16 (Sep 22, 1807 - Sep 21, 1808), Gregorian 1808 IS leap
		{20, true},  // Year 20 (Sep 22, 1811 - Sep 21, 1812), Gregorian 1812 IS leap
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := IsFrenchLeapYear(tt.year)
			if got != tt.want {
				t.Errorf("IsFrenchLeapYear(%d) = %v, want %v", tt.year, got, tt.want)
			}
		})
	}
}

// TestFrenchToJDN tests the French Republican to JDN conversion with known reference dates.
func TestFrenchToJDN(t *testing.T) {
	tests := []struct {
		name  string
		year  int
		month int
		day   int
		want  int
	}{
		// Epoch and critical dates
		{
			name:  "1 Vendémiaire 1 (epoch)",
			year:  1,
			month: 1,
			day:   1,
			want:  2375840, // September 22, 1792
		},
		{
			name:  "1 Vendémiaire 8",
			year:  8,
			month: 1,
			day:   1,
			want:  2378396, // September 22, 1799
		},
		{
			name:  "1 Vendémiaire 14",
			year:  14,
			month: 1,
			day:   1,
			want:  2380587, // September 22, 1805
		},

		// Test each month type
		{
			name:  "1 Brumaire 1",
			year:  1,
			month: 2,
			day:   1,
			want:  2375870, // 30 days after epoch
		},
		{
			name:  "1 Frimaire 1",
			year:  1,
			month: 3,
			day:   1,
			want:  2375900, // 60 days after epoch
		},

		// Test last day of regular month
		{
			name:  "30 Vendémiaire 1",
			year:  1,
			month: 1,
			day:   30,
			want:  2375869,
		},

		// Test complementary days (month 13)
		{
			name:  "1 Complementary 1 (first complementary day)",
			year:  1,
			month: 13,
			day:   1,
			want:  2376200, // After 12 months of 30 days each
		},
		{
			name:  "5 Complementary 1 (last complementary day, non-leap)",
			year:  1,
			month: 13,
			day:   5,
			want:  2376204,
		},

		// Test leap year complementary days
		{
			name:  "6 Complementary 4 (leap year, 6th day)",
			year:  4,
			month: 13,
			day:   6,
			want:  2377300, // Year 4 is a leap year
		},

		// Test year boundaries
		{
			name:  "1 Vendémiaire 2",
			year:  2,
			month: 1,
			day:   1,
			want:  2376205, // Should be day after last complementary day of year 1
		},
		{
			name:  "1 Vendémiaire 3",
			year:  3,
			month: 1,
			day:   1,
			want:  2376570,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FrenchToJDN(tt.year, tt.month, tt.day)
			if got != tt.want {
				t.Errorf("FrenchToJDN(%d, %d, %d) = %d, want %d",
					tt.year, tt.month, tt.day, got, tt.want)
			}
		})
	}
}

// TestJDNToFrench tests the JDN to French Republican conversion with known reference dates.
func TestJDNToFrench(t *testing.T) {
	tests := []struct {
		name      string
		jdn       int
		wantYear  int
		wantMonth int
		wantDay   int
	}{
		// Epoch and critical dates
		{
			name:      "JDN 2375840 (1 Vendémiaire 1)",
			jdn:       2375840,
			wantYear:  1,
			wantMonth: 1,
			wantDay:   1,
		},
		{
			name:      "JDN 2378396 (1 Vendémiaire 8)",
			jdn:       2378396,
			wantYear:  8,
			wantMonth: 1,
			wantDay:   1,
		},
		{
			name:      "JDN 2380587 (1 Vendémiaire 14)",
			jdn:       2380587,
			wantYear:  14,
			wantMonth: 1,
			wantDay:   1,
		},

		// Test each month
		{
			name:      "JDN 2375870 (1 Brumaire 1)",
			jdn:       2375870,
			wantYear:  1,
			wantMonth: 2,
			wantDay:   1,
		},
		{
			name:      "JDN 2375900 (1 Frimaire 1)",
			jdn:       2375900,
			wantYear:  1,
			wantMonth: 3,
			wantDay:   1,
		},

		// Test last day of month
		{
			name:      "JDN 2375869 (30 Vendémiaire 1)",
			jdn:       2375869,
			wantYear:  1,
			wantMonth: 1,
			wantDay:   30,
		},

		// Test complementary days
		{
			name:      "JDN 2376200 (1 Complementary 1)",
			jdn:       2376200,
			wantYear:  1,
			wantMonth: 13,
			wantDay:   1,
		},
		{
			name:      "JDN 2376204 (5 Complementary 1)",
			jdn:       2376204,
			wantYear:  1,
			wantMonth: 13,
			wantDay:   5,
		},

		// Test leap year complementary days
		{
			name:      "JDN 2377300 (6 Complementary 4, leap year)",
			jdn:       2377300,
			wantYear:  4,
			wantMonth: 13,
			wantDay:   6,
		},

		// Test year boundaries
		{
			name:      "JDN 2376205 (1 Vendémiaire 2)",
			jdn:       2376205,
			wantYear:  2,
			wantMonth: 1,
			wantDay:   1,
		},
		{
			name:      "JDN 2376570 (1 Vendémiaire 3)",
			jdn:       2376570,
			wantYear:  3,
			wantMonth: 1,
			wantDay:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotYear, gotMonth, gotDay := JDNToFrench(tt.jdn)
			if gotYear != tt.wantYear || gotMonth != tt.wantMonth || gotDay != tt.wantDay {
				t.Errorf("JDNToFrench(%d) = (%d, %d, %d), want (%d, %d, %d)",
					tt.jdn, gotYear, gotMonth, gotDay, tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

// TestFrenchJDNRoundTrip tests that converting French -> JDN -> French
// returns the same date (verifying the functions are inverse operations).
func TestFrenchJDNRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		year  int
		month int
		day   int
	}{
		{"Epoch", 1, 1, 1},
		{"End of first month", 1, 1, 30},
		{"Start of second month", 1, 2, 1},
		{"Mid year", 1, 6, 15},
		{"Last regular month", 1, 12, 30},
		{"First complementary day", 1, 13, 1},
		{"Last complementary day (non-leap)", 1, 13, 5},
		{"Last complementary day (leap year 4)", 4, 13, 6},
		{"Year 2 start", 2, 1, 1},
		{"Year 3 start", 3, 1, 1},
		{"Year 8 start", 8, 1, 1},
		{"Year 14 start", 14, 1, 1},
		{"Year 10 mid", 10, 7, 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to JDN and back
			jdn := FrenchToJDN(tt.year, tt.month, tt.day)
			gotYear, gotMonth, gotDay := JDNToFrench(jdn)

			if gotYear != tt.year || gotMonth != tt.month || gotDay != tt.day {
				t.Errorf("Round trip failed for (%d, %d, %d): got (%d, %d, %d) via JDN %d",
					tt.year, tt.month, tt.day, gotYear, gotMonth, gotDay, jdn)
			}
		})
	}
}

// TestFrenchToGregorian tests conversion from French Republican to Gregorian calendar.
func TestFrenchToGregorian(t *testing.T) {
	tests := []struct {
		name          string
		frenchYear    int
		frenchMonth   int
		frenchDay     int
		gregorianYear int
		gregorianMon  int
		gregorianDay  int
	}{
		{
			name:          "1 Vendémiaire 1 = Sep 22, 1792",
			frenchYear:    1,
			frenchMonth:   1,
			frenchDay:     1,
			gregorianYear: 1792,
			gregorianMon:  9,
			gregorianDay:  22,
		},
		{
			name:          "1 Vendémiaire 8 = Sep 22, 1799",
			frenchYear:    8,
			frenchMonth:   1,
			frenchDay:     1,
			gregorianYear: 1799,
			gregorianMon:  9,
			gregorianDay:  22,
		},
		{
			name:          "1 Vendémiaire 14 = Sep 22, 1805",
			frenchYear:    14,
			frenchMonth:   1,
			frenchDay:     1,
			gregorianYear: 1805,
			gregorianMon:  9,
			gregorianDay:  22,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert French to JDN
			jdn := FrenchToJDN(tt.frenchYear, tt.frenchMonth, tt.frenchDay)

			// Convert JDN to Gregorian
			year, month, day := JDNToGregorian(jdn)

			if year != tt.gregorianYear || month != tt.gregorianMon || day != tt.gregorianDay {
				t.Errorf("French %d/%d/%d = JDN %d = Gregorian %d/%d/%d, want %d/%d/%d",
					tt.frenchYear, tt.frenchMonth, tt.frenchDay,
					jdn,
					year, month, day,
					tt.gregorianYear, tt.gregorianMon, tt.gregorianDay)
			}
		})
	}
}

// TestIsHebrewLeapYear tests the Hebrew leap year calculation using the 19-year Metonic cycle.
func TestIsHebrewLeapYear(t *testing.T) {
	tests := []struct {
		year int
		want bool
	}{
		// Known leap years in recent cycle (years where (7*year+1)%19 < 7)
		{5784, true}, // 2023-2024 (leap year) - (7*5784+1)%19 = 0
		{5787, true}, // 2026-2027 (leap year) - (7*5787+1)%19 = 2
		{5790, true}, // 2029-2030 (leap year) - (7*5790+1)%19 = 4
		{5793, true}, // 2032-2033 (leap year) - (7*5793+1)%19 = 6
		{5795, true}, // 2034-2035 (leap year) - (7*5795+1)%19 = 1
		{5798, true}, // 2037-2038 (leap year) - (7*5798+1)%19 = 3
		{5776, true}, // 2015-2016 (leap year) - (7*5776+1)%19 = 1

		// Known regular years in recent cycle
		{5785, false}, // 2024-2025 (not leap) - (7*5785+1)%19 = 7
		{5786, false}, // 2025-2026 (not leap) - (7*5786+1)%19 = 14
		{5788, false}, // 2027-2028 (not leap) - (7*5788+1)%19 = 9
		{5789, false}, // 2028-2029 (not leap) - (7*5789+1)%19 = 16
		{5791, false}, // 2030-2031 (not leap) - (7*5791+1)%19 = 11
		{5792, false}, // 2031-2032 (not leap) - (7*5792+1)%19 = 18
		{5794, false}, // 2033-2034 (not leap) - (7*5794+1)%19 = 12
		{5796, false}, // 2035-2036 (not leap) - (7*5796+1)%19 = 8
		{5797, false}, // 2036-2037 (not leap) - (7*5797+1)%19 = 15
		{5799, false}, // 2038-2039 (not leap) - (7*5799+1)%19 = 10
		{5800, false}, // 2039-2040 (not leap) - (7*5800+1)%19 = 17
		{5780, false}, // 2019-2020 (not leap) - (7*5780+1)%19 = 10

		// Very early years
		{1, false},
		{3, true},  // First leap year in the cycle
		{6, true},  // Second leap year
		{8, true},  // Third leap year
		{11, true}, // Fourth leap year
		{14, true}, // Fifth leap year
		{17, true}, // Sixth leap year
		{19, true}, // Seventh leap year
		{20, false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := IsHebrewLeapYear(tt.year)
			if got != tt.want {
				t.Errorf("IsHebrewLeapYear(%d) = %v, want %v", tt.year, got, tt.want)
			}
		})
	}
}

// TestHebrewMonthsInYear tests the number of months in Hebrew years.
func TestHebrewMonthsInYear(t *testing.T) {
	tests := []struct {
		year int
		want int
	}{
		{5784, 13}, // Leap year
		{5785, 12}, // Regular year
		{5787, 13}, // Leap year
		{5789, 12}, // Regular year
		{1, 12},    // First year, regular
		{3, 13},    // First leap year
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := HebrewMonthsInYear(tt.year)
			if got != tt.want {
				t.Errorf("HebrewMonthsInYear(%d) = %d, want %d", tt.year, got, tt.want)
			}
		})
	}
}

// TestHebrewDaysInYear tests the calculation of year length including the dehiyot.
func TestHebrewDaysInYear(t *testing.T) {
	tests := []struct {
		year int
		want int
	}{
		// Known year lengths (can be verified with hebcal.com or similar)
		{5785, 355}, // Regular complete year (not leap)
		{5784, 383}, // Leap deficient year
		{5780, 355}, // Regular complete year (not leap, not 385!)

		// Test that all year lengths are in valid range
		{1, 0},    // Will calculate
		{100, 0},  // Will calculate
		{5000, 0}, // Will calculate
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := HebrewDaysInYear(tt.year)

			// If expected value is 0, just verify it's in valid range
			if tt.want == 0 {
				isLeap := IsHebrewLeapYear(tt.year)
				minDays, maxDays := 353, 355
				if isLeap {
					minDays, maxDays = 383, 385
				}
				if got < minDays || got > maxDays {
					t.Errorf("HebrewDaysInYear(%d) = %d, want between %d and %d (leap=%v)",
						tt.year, got, minDays, maxDays, isLeap)
				}
			} else if got != tt.want {
				t.Errorf("HebrewDaysInYear(%d) = %d, want %d", tt.year, got, tt.want)
			}
		})
	}
}

// TestHebrewDaysInMonth tests the calculation of month lengths.
func TestHebrewDaysInMonth(t *testing.T) {
	tests := []struct {
		name  string
		year  int
		month int
		want  int
	}{
		// Fixed-length months
		{"Tishrei always 30", 5785, 1, 30},
		{"Tevet always 29", 5785, 4, 29},
		{"Shevat always 30", 5785, 5, 30},
		{"Adar 29 in regular year", 5785, 6, 29},
		{"Nisan always 30", 5785, 8, 30},
		{"Iyar always 29", 5785, 9, 29},
		{"Sivan always 30", 5785, 10, 30},
		{"Tammuz always 29", 5785, 11, 29},
		{"Av always 30", 5785, 12, 30},
		{"Elul always 29", 5785, 13, 29},

		// Variable months (Cheshvan and Kislev)
		{"Cheshvan in complete year", 5785, 2, 30},
		{"Kislev in complete year", 5785, 3, 30},

		// Leap year Adar I and Adar II
		{"Adar I in leap year", 5784, 6, 30},
		{"Adar II in leap year", 5784, 7, 29},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HebrewDaysInMonth(tt.year, tt.month)
			if got != tt.want {
				t.Errorf("HebrewDaysInMonth(%d, %d) = %d, want %d",
					tt.year, tt.month, got, tt.want)
			}
		})
	}
}

// TestHebrewToJDN tests the Hebrew to JDN conversion with known reference dates.
func TestHebrewToJDN(t *testing.T) {
	tests := []struct {
		name   string
		year   int
		month  int
		day    int
		want   int
		verify string // Description for manual verification
	}{
		// Reference dates from the prompt
		{
			name:   "Rosh Hashanah 5785",
			year:   5785,
			month:  1,
			day:    1,
			want:   2460587,
			verify: "Oct 3, 2024",
		},
		{
			name:   "Passover 5785",
			year:   5785,
			month:  8,
			day:    15,
			want:   2460779,
			verify: "Apr 13, 2025",
		},
		{
			name:   "Yom Kippur 5780",
			year:   5780,
			month:  1,
			day:    10,
			want:   2458766,
			verify: "Oct 9, 2019",
		},

		// Additional test points
		{
			name:   "Hebrew epoch",
			year:   1,
			month:  1,
			day:    1,
			want:   347998,
			verify: "Sep 7, 3761 BC (Julian)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HebrewToJDN(tt.year, tt.month, tt.day)
			if got != tt.want {
				t.Errorf("HebrewToJDN(%d, %d, %d) = %d, want %d (%s)",
					tt.year, tt.month, tt.day, got, tt.want, tt.verify)
			}
		})
	}
}

// TestJDNToHebrew tests the JDN to Hebrew conversion with known reference dates.
func TestJDNToHebrew(t *testing.T) {
	tests := []struct {
		name      string
		jdn       int
		wantYear  int
		wantMonth int
		wantDay   int
		verify    string
	}{
		{
			name:      "JDN 2460587 (Rosh Hashanah 5785)",
			jdn:       2460587,
			wantYear:  5785,
			wantMonth: 1,
			wantDay:   1,
			verify:    "Oct 3, 2024",
		},
		{
			name:      "JDN 2460779 (Passover 5785)",
			jdn:       2460779,
			wantYear:  5785,
			wantMonth: 8,
			wantDay:   15,
			verify:    "Apr 13, 2025",
		},
		{
			name:      "JDN 2458766 (Yom Kippur 5780)",
			jdn:       2458766,
			wantYear:  5780,
			wantMonth: 1,
			wantDay:   10,
			verify:    "Oct 9, 2019",
		},
		{
			name:      "JDN 347998 (Hebrew epoch)",
			jdn:       347998,
			wantYear:  1,
			wantMonth: 1,
			wantDay:   1,
			verify:    "Sep 7, 3761 BC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotYear, gotMonth, gotDay := JDNToHebrew(tt.jdn)
			if gotYear != tt.wantYear || gotMonth != tt.wantMonth || gotDay != tt.wantDay {
				t.Errorf("JDNToHebrew(%d) = (%d, %d, %d), want (%d, %d, %d) (%s)",
					tt.jdn, gotYear, gotMonth, gotDay,
					tt.wantYear, tt.wantMonth, tt.wantDay, tt.verify)
			}
		})
	}
}

// TestHebrewJDNRoundTrip tests that converting Hebrew -> JDN -> Hebrew returns the same date.
func TestHebrewJDNRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		year  int
		month int
		day   int
	}{
		{"Rosh Hashanah 5785", 5785, 1, 1},
		{"Passover 5785", 5785, 8, 15},
		{"Yom Kippur 5780", 5780, 1, 10},
		{"Hebrew epoch", 1, 1, 1},
		{"Mid year regular", 5785, 6, 15},
		{"End of Tishrei", 5785, 1, 30},
		{"Leap year Adar I", 5784, 6, 15},
		{"Leap year Adar II", 5784, 7, 15},
		{"Last month Elul", 5785, 13, 29},
		{"Cheshvan", 5785, 2, 15},
		{"Kislev", 5785, 3, 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to JDN and back
			jdn := HebrewToJDN(tt.year, tt.month, tt.day)
			gotYear, gotMonth, gotDay := JDNToHebrew(jdn)

			if gotYear != tt.year || gotMonth != tt.month || gotDay != tt.day {
				t.Errorf("Round trip failed for (%d, %d, %d): got (%d, %d, %d) via JDN %d",
					tt.year, tt.month, tt.day, gotYear, gotMonth, gotDay, jdn)
			}
		})
	}
}

// TestHebrewToGregorian tests conversion from Hebrew to Gregorian calendar.
func TestHebrewToGregorian(t *testing.T) {
	tests := []struct {
		name          string
		hebrewYear    int
		hebrewMonth   int
		hebrewDay     int
		gregorianYear int
		gregorianMon  int
		gregorianDay  int
	}{
		{
			name:          "Rosh Hashanah 5785 = Oct 3, 2024",
			hebrewYear:    5785,
			hebrewMonth:   1,
			hebrewDay:     1,
			gregorianYear: 2024,
			gregorianMon:  10,
			gregorianDay:  3,
		},
		{
			name:          "Passover 5785 = Apr 13, 2025",
			hebrewYear:    5785,
			hebrewMonth:   8,
			hebrewDay:     15,
			gregorianYear: 2025,
			gregorianMon:  4,
			gregorianDay:  13,
		},
		{
			name:          "Yom Kippur 5780 = Oct 9, 2019",
			hebrewYear:    5780,
			hebrewMonth:   1,
			hebrewDay:     10,
			gregorianYear: 2019,
			gregorianMon:  10,
			gregorianDay:  9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert Hebrew to JDN
			jdn := HebrewToJDN(tt.hebrewYear, tt.hebrewMonth, tt.hebrewDay)

			// Convert JDN to Gregorian
			year, month, day := JDNToGregorian(jdn)

			if year != tt.gregorianYear || month != tt.gregorianMon || day != tt.gregorianDay {
				t.Errorf("Hebrew %d/%d/%d = JDN %d = Gregorian %d/%d/%d, want %d/%d/%d",
					tt.hebrewYear, tt.hebrewMonth, tt.hebrewDay,
					jdn,
					year, month, day,
					tt.gregorianYear, tt.gregorianMon, tt.gregorianDay)
			}
		})
	}
}

// TestHebrewLeapYearAdarHandling tests that Adar I and Adar II are handled correctly in leap years.
func TestHebrewLeapYearAdarHandling(t *testing.T) {
	// Year 5784 is a leap year
	leapYear := 5784

	// Verify it's a leap year
	if !IsHebrewLeapYear(leapYear) {
		t.Fatalf("Year %d should be a leap year", leapYear)
	}

	// Verify 13 months
	if HebrewMonthsInYear(leapYear) != 13 {
		t.Fatalf("Leap year %d should have 13 months", leapYear)
	}

	// Verify Adar I (month 6) has 30 days
	adarIDays := HebrewDaysInMonth(leapYear, 6)
	if adarIDays != 30 {
		t.Errorf("Adar I in leap year should have 30 days, got %d", adarIDays)
	}

	// Verify Adar II (month 7) has 29 days
	adarIIDays := HebrewDaysInMonth(leapYear, 7)
	if adarIIDays != 29 {
		t.Errorf("Adar II in leap year should have 29 days, got %d", adarIIDays)
	}

	// Test round trip for both Adar months
	for month := 6; month <= 7; month++ {
		for day := 1; day <= HebrewDaysInMonth(leapYear, month); day++ {
			jdn := HebrewToJDN(leapYear, month, day)
			gotYear, gotMonth, gotDay := JDNToHebrew(jdn)
			if gotYear != leapYear || gotMonth != month || gotDay != day {
				t.Errorf("Adar round trip failed for (%d, %d, %d): got (%d, %d, %d)",
					leapYear, month, day, gotYear, gotMonth, gotDay)
			}
		}
	}
}
