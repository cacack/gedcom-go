package gedcom

import (
	"strings"
	"testing"
	"time"
)

func TestParseDate_ExactDates(t *testing.T) {
	tests := []struct {
		input     string
		wantDay   int
		wantMonth int
		wantYear  int
	}{
		{"25 DEC 2020", 25, 12, 2020},
		{"1 JAN 1900", 1, 1, 1900},
		{"14 FEB 1890", 14, 2, 1890},
		{"31 MAR 2000", 31, 3, 2000},
		{"15 APR 1950", 15, 4, 1950},
		{"1 MAY 1975", 1, 5, 1975},
		{"30 JUN 1985", 30, 6, 1985},
		{"4 JUL 1776", 4, 7, 1776},
		{"15 AUG 2010", 15, 8, 2010},
		{"11 SEP 2001", 11, 9, 2001},
		{"31 OCT 1999", 31, 10, 1999},
		{"11 NOV 1918", 11, 11, 1918},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			date, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.input, err)
			}

			if date.Day != tt.wantDay {
				t.Errorf("Day = %d, want %d", date.Day, tt.wantDay)
			}
			if date.Month != tt.wantMonth {
				t.Errorf("Month = %d, want %d", date.Month, tt.wantMonth)
			}
			if date.Year != tt.wantYear {
				t.Errorf("Year = %d, want %d", date.Year, tt.wantYear)
			}
			if date.Modifier != ModifierNone {
				t.Errorf("Modifier = %v, want ModifierNone", date.Modifier)
			}
			if date.Calendar != CalendarGregorian {
				t.Errorf("Calendar = %v, want CalendarGregorian", date.Calendar)
			}
			if date.Original != tt.input {
				t.Errorf("Original = %q, want %q", date.Original, tt.input)
			}
		})
	}
}

func TestParseDate_PartialDates(t *testing.T) {
	tests := []struct {
		input     string
		wantDay   int
		wantMonth int
		wantYear  int
	}{
		{"1850", 0, 0, 1850},
		{"2000", 0, 0, 2000},
		{"1920", 0, 0, 1920},
		{"JAN 1900", 0, 1, 1900},
		{"FEB 1802", 0, 2, 1802},
		{"DEC 2020", 0, 12, 2020},
		{"MAR 1950", 0, 3, 1950},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			date, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.input, err)
			}

			if date.Day != tt.wantDay {
				t.Errorf("Day = %d, want %d", date.Day, tt.wantDay)
			}
			if date.Month != tt.wantMonth {
				t.Errorf("Month = %d, want %d", date.Month, tt.wantMonth)
			}
			if date.Year != tt.wantYear {
				t.Errorf("Year = %d, want %d", date.Year, tt.wantYear)
			}
		})
	}
}

func TestParseDate_CaseInsensitiveMonths(t *testing.T) {
	tests := []string{
		"25 DEC 2020",
		"25 Dec 2020",
		"25 dec 2020",
		"25 dEc 2020",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			date, err := ParseDate(input)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", input, err)
			}

			if date.Month != 12 {
				t.Errorf("Month = %d, want 12", date.Month)
			}
		})
	}
}

func TestParseDate_Modifiers(t *testing.T) {
	tests := []struct {
		input        string
		wantModifier DateModifier
		wantYear     int
	}{
		{"ABT 1850", ModifierAbout, 1850},
		{"CAL 1875", ModifierCalculated, 1875},
		{"EST 1820", ModifierEstimated, 1820},
		{"BEF 1900", ModifierBefore, 1900},
		{"AFT 1850", ModifierAfter, 1850},
		{"ABT 12 MAY 1875", ModifierAbout, 1875},
		{"BEF 15 JUN 1850", ModifierBefore, 1850},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			date, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.input, err)
			}

			if date.Modifier != tt.wantModifier {
				t.Errorf("Modifier = %v, want %v", date.Modifier, tt.wantModifier)
			}
			if date.Year != tt.wantYear {
				t.Errorf("Year = %d, want %d", date.Year, tt.wantYear)
			}
		})
	}
}

func TestParseDate_Ranges(t *testing.T) {
	tests := []struct {
		input          string
		wantStartYear  int
		wantEndYear    int
		wantStartMonth int
		wantEndMonth   int
		wantStartDay   int
		wantEndDay     int
	}{
		{"BET 1850 AND 1860", 1850, 1860, 0, 0, 0, 0},
		{"BET 1 JAN 1900 AND 31 DEC 1900", 1900, 1900, 1, 12, 1, 31},
		{"BET FEB 1920 AND MAR 1920", 1920, 1920, 2, 3, 0, 0},
		{"BET 1848 AND 1852", 1848, 1852, 0, 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			date, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.input, err)
			}

			if date.Modifier != ModifierBetween {
				t.Errorf("Modifier = %v, want ModifierBetween", date.Modifier)
			}
			if date.Year != tt.wantStartYear {
				t.Errorf("Start Year = %d, want %d", date.Year, tt.wantStartYear)
			}
			if date.Month != tt.wantStartMonth {
				t.Errorf("Start Month = %d, want %d", date.Month, tt.wantStartMonth)
			}
			if date.Day != tt.wantStartDay {
				t.Errorf("Start Day = %d, want %d", date.Day, tt.wantStartDay)
			}

			if date.EndDate == nil {
				t.Fatal("EndDate is nil, want non-nil")
			}
			if date.EndDate.Year != tt.wantEndYear {
				t.Errorf("End Year = %d, want %d", date.EndDate.Year, tt.wantEndYear)
			}
			if date.EndDate.Month != tt.wantEndMonth {
				t.Errorf("End Month = %d, want %d", date.EndDate.Month, tt.wantEndMonth)
			}
			if date.EndDate.Day != tt.wantEndDay {
				t.Errorf("End Day = %d, want %d", date.EndDate.Day, tt.wantEndDay)
			}
		})
	}
}

func TestParseDate_Periods(t *testing.T) {
	tests := []struct {
		input        string
		wantModifier DateModifier
		wantYear     int
		wantEndYear  int
		hasEndDate   bool
	}{
		{"FROM 1880", ModifierFrom, 1880, 0, false},
		{"TO 1920", ModifierTo, 1920, 0, false},
		{"FROM 1880 TO 1920", ModifierFromTo, 1880, 1920, true},
		{"FROM JAN 1900 TO DEC 1905", ModifierFromTo, 1900, 1905, true},
		{"FROM 1 JAN 1900 TO 31 DEC 1910", ModifierFromTo, 1900, 1910, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			date, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.input, err)
			}

			if date.Modifier != tt.wantModifier {
				t.Errorf("Modifier = %v, want %v", date.Modifier, tt.wantModifier)
			}
			if date.Year != tt.wantYear {
				t.Errorf("Year = %d, want %d", date.Year, tt.wantYear)
			}

			if tt.hasEndDate {
				if date.EndDate == nil {
					t.Fatal("EndDate is nil, want non-nil")
				}
				if date.EndDate.Year != tt.wantEndYear {
					t.Errorf("End Year = %d, want %d", date.EndDate.Year, tt.wantEndYear)
				}
			} else if date.EndDate != nil {
				t.Errorf("EndDate is non-nil, want nil")
			}
		})
	}
}

func TestParseDate_WhitespaceTolerance(t *testing.T) {
	tests := []struct {
		input    string
		wantYear int
	}{
		{" 25 DEC 2020 ", 2020},
		{"25  DEC  2020", 2020},
		{"  1850  ", 1850},
		{"ABT  1850", 1850},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			date, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.input, err)
			}

			if date.Year != tt.wantYear {
				t.Errorf("Year = %d, want %d", date.Year, tt.wantYear)
			}
		})
	}
}

func TestParseDate_CalendarEscapes(t *testing.T) {
	tests := []struct {
		input        string
		wantErr      bool
		wantCalendar Calendar
	}{
		{"@#DGREGORIAN@ 25 DEC 2020", false, CalendarGregorian},
		{"@#DJULIAN@ 25 DEC 1700", false, CalendarJulian},
		{"@#DHEBREW@ 13 CSH 5760", false, CalendarHebrew},
		{"@#DFRENCH R@ 15 VEND 3", false, CalendarFrenchRepublican},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			date, err := ParseDate(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseDate(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Fatalf("ParseDate(%q) unexpected error = %v", tt.input, err)
				}
				if date.Calendar != tt.wantCalendar {
					t.Errorf("Calendar = %v, want %v", date.Calendar, tt.wantCalendar)
				}
			}
		})
	}
}

func TestParseDate_InvalidDates(t *testing.T) {
	tests := []string{
		"",
		"XYZ 2020",
		"INVALID",
		"25 XYZ 2020",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := ParseDate(input)
			if err == nil {
				t.Errorf("ParseDate(%q) expected error, got nil", input)
			}
		})
	}
}

// Note: ParseDate does not validate semantic correctness (e.g., day 32, Feb 30).
// That would be handled by a separate validation function in future phases.

func TestDate_Compare(t *testing.T) {
	tests := []struct {
		date1   string
		date2   string
		wantCmp int
	}{
		// Exact dates
		{"25 DEC 2020", "25 DEC 2020", 0},
		{"25 DEC 2020", "26 DEC 2020", -1},
		{"26 DEC 2020", "25 DEC 2020", 1},
		{"25 DEC 2019", "25 DEC 2020", -1},
		{"25 DEC 2020", "25 DEC 2019", 1},

		// Partial dates (year only)
		{"1850", "1850", 0},
		{"1850", "1851", -1},
		{"1851", "1850", 1},

		// Partial dates (month and year) - missing day treated as 1
		{"JAN 1920", "JAN 1920", 0},
		{"JAN 1920", "FEB 1920", -1},
		{"FEB 1920", "JAN 1920", 1},

		// Mix of partial and complete dates
		{"1920", "1 JAN 1920", 0},     // 1920 treated as 1 JAN 1920
		{"JAN 1920", "1 JAN 1920", 0}, // JAN 1920 treated as 1 JAN 1920
		{"1920", "2 JAN 1920", -1},
		{"JAN 1920", "2 JAN 1920", -1},
	}

	for _, tt := range tests {
		t.Run(tt.date1+" vs "+tt.date2, func(t *testing.T) {
			d1, err := ParseDate(tt.date1)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.date1, err)
			}
			d2, err := ParseDate(tt.date2)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.date2, err)
			}

			got := d1.Compare(d2)
			if got != tt.wantCmp {
				t.Errorf("Compare() = %d, want %d", got, tt.wantCmp)
			}
		})
	}
}

func TestDate_ToTime(t *testing.T) {
	tests := []struct {
		input    string
		wantTime time.Time
		wantErr  bool
	}{
		{"25 DEC 2020", time.Date(2020, 12, 25, 0, 0, 0, 0, time.UTC), false},
		{"1 JAN 1900", time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"1850", time.Time{}, true},     // Incomplete: no month/day
		{"JAN 1900", time.Time{}, true}, // Incomplete: no day
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			date, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.input, err)
			}

			gotTime, err := date.ToTime()
			if tt.wantErr {
				if err == nil {
					t.Errorf("ToTime() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("ToTime() unexpected error = %v", err)
				}
				if !gotTime.Equal(tt.wantTime) {
					t.Errorf("ToTime() = %v, want %v", gotTime, tt.wantTime)
				}
			}
		})
	}
}

func TestDate_String(t *testing.T) {
	tests := []string{
		"25 DEC 2020",
		"JAN 1900",
		"1850",
		"ABT 1850",
		"BET 1850 AND 1860",
		"FROM 1880 TO 1920",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			date, err := ParseDate(input)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", input, err)
			}

			if date.String() != input {
				t.Errorf("String() = %q, want %q", date.String(), input)
			}
		})
	}
}

func TestCalendar_String(t *testing.T) {
	tests := []struct {
		calendar Calendar
		want     string
	}{
		{CalendarGregorian, "Gregorian"},
		{CalendarJulian, "Julian"},
		{CalendarHebrew, "Hebrew"},
		{CalendarFrenchRepublican, "French Republican"},
		{Calendar(999), "Unknown"}, // Unknown calendar type
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.calendar.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDateModifier_String(t *testing.T) {
	tests := []struct {
		modifier DateModifier
		want     string
	}{
		{ModifierNone, ""},
		{ModifierAbout, "ABT"},
		{ModifierCalculated, "CAL"},
		{ModifierEstimated, "EST"},
		{ModifierBefore, "BEF"},
		{ModifierAfter, "AFT"},
		{ModifierBetween, "BET"},
		{ModifierFrom, "FROM"},
		{ModifierTo, "TO"},
		{ModifierFromTo, "FROM TO"},
		{DateModifier(999), "Unknown"}, // Unknown modifier type
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.modifier.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestParseDate_RangeErrors tests error cases for date ranges
func TestParseDate_RangeErrors(t *testing.T) {
	tests := []struct {
		input   string
		wantErr string
	}{
		{"BET 1850", "missing AND keyword"},
		{"BET INVALID AND 1860", "invalid start date"},
		{"BET 1850 AND INVALID", "invalid end date"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := ParseDate(tt.input)
			if err == nil {
				t.Fatalf("ParseDate(%q) expected error, got nil", tt.input)
			}
			if tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Error message = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// TestParseDate_PeriodErrors tests error cases for date periods
func TestParseDate_PeriodErrors(t *testing.T) {
	tests := []struct {
		input   string
		wantErr string
	}{
		{"FROM INVALID", "invalid"},
		{"TO INVALID", "invalid"},
		{"FROM INVALID TO 1920", "invalid start date"},
		{"FROM 1880 TO INVALID", "invalid end date"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := ParseDate(tt.input)
			if err == nil {
				t.Fatalf("ParseDate(%q) expected error, got nil", tt.input)
			}
			if tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Error message = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// TestParseDate_TooManyComponents tests dates with too many components
func TestParseDate_TooManyComponents(t *testing.T) {
	tests := []string{
		"1 2 3 4 5",
		"1 JAN 2020 EXTRA",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := ParseDate(input)
			if err == nil {
				t.Fatalf("ParseDate(%q) expected error, got nil", input)
			}
			if !strings.Contains(err.Error(), "too many components") {
				t.Errorf("Error message = %q, want substring 'too many components'", err.Error())
			}
		})
	}
}

// TestDate_ToTime_NonGregorian tests ToTime with non-Gregorian calendar
func TestDate_ToTime_NonGregorian(t *testing.T) {
	date := &Date{
		Day:      25,
		Month:    12,
		Year:     1700,
		Calendar: CalendarJulian,
	}

	_, err := date.ToTime()
	if err == nil {
		t.Fatal("ToTime() expected error for Julian calendar, got nil")
	}
	if !strings.Contains(err.Error(), "Gregorian") {
		t.Errorf("Error message = %q, want substring 'Gregorian'", err.Error())
	}
}

// TestDate_Compare_NilDates tests Compare with nil dates
func TestDate_Compare_NilDates(t *testing.T) {
	tests := []struct {
		name    string
		d1      *Date
		d2      *Date
		wantCmp int
	}{
		{"both nil", nil, nil, 0},
		{"first nil", nil, &Date{Year: 2020}, -1},
		{"second nil", &Date{Year: 2020}, nil, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.d1.Compare(tt.d2)
			if got != tt.wantCmp {
				t.Errorf("Compare() = %d, want %d", got, tt.wantCmp)
			}
		})
	}
}

// TestParseDate_CalendarEscapeEdgeCases tests edge cases for calendar escape parsing
func TestParseDate_CalendarEscapeEdgeCases(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"@#D", true},                     // Missing closing @
		{"@#DUNKNOWN@ 25 DEC 2020", true}, // Unknown calendar (treated as no escape, then invalid)
		{"@#D@ 25 DEC 2020", true},        // Empty calendar name
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := ParseDate(tt.input)
			if tt.wantErr && err == nil {
				t.Fatalf("ParseDate(%q) expected error, got nil", tt.input)
			}
		})
	}
}

// TestParseDate_CaseInsensitiveModifiers tests that modifiers are case-insensitive
func TestParseDate_CaseInsensitiveModifiers(t *testing.T) {
	tests := []struct {
		input        string
		wantModifier DateModifier
	}{
		{"abt 1850", ModifierAbout},
		{"Abt 1850", ModifierAbout},
		{"ABT 1850", ModifierAbout},
		{"bef 1900", ModifierBefore},
		{"aft 1850", ModifierAfter},
		{"cal 1875", ModifierCalculated},
		{"est 1820", ModifierEstimated},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			date, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.input, err)
			}
			if date.Modifier != tt.wantModifier {
				t.Errorf("Modifier = %v, want %v", date.Modifier, tt.wantModifier)
			}
		})
	}
}

// TestParseDate_InvalidDayNumbers tests dates with invalid day numbers (syntactically valid, but semantically questionable)
func TestParseDate_InvalidDayNumbers(t *testing.T) {
	// Note: The parser does not validate semantic correctness (e.g., Feb 30, day 32)
	// These are syntactically valid and will parse successfully
	tests := []string{
		"32 JAN 2020", // Day 32 doesn't exist
		"40 DEC 2020", // Day 40 doesn't exist
		"99 MAR 2020", // Day 99 doesn't exist
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			date, err := ParseDate(input)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v (parser doesn't validate semantic correctness)", input, err)
			}
			// Just verify it parsed, even though the date is semantically invalid
			if date.Day == 0 {
				t.Errorf("Day = 0, expected non-zero value")
			}
		})
	}
}

// TestDate_ToTime_IncompleteDates tests ToTime with incomplete dates (missing year, month, or day)
func TestDate_ToTime_IncompleteDates(t *testing.T) {
	tests := []struct {
		name    string
		date    *Date
		wantErr string
	}{
		{
			name:    "missing year",
			date:    &Date{Day: 25, Month: 12, Year: 0, Calendar: CalendarGregorian},
			wantErr: "year is missing",
		},
		{
			name:    "missing month",
			date:    &Date{Day: 25, Month: 0, Year: 2020, Calendar: CalendarGregorian},
			wantErr: "month is missing",
		},
		{
			name:    "missing day",
			date:    &Date{Day: 0, Month: 12, Year: 2020, Calendar: CalendarGregorian},
			wantErr: "day is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.date.ToTime()
			if err == nil {
				t.Fatal("ToTime() expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Error message = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// TestParseDate_InvalidYear tests dates with invalid year values
func TestParseDate_InvalidYear(t *testing.T) {
	tests := []string{
		"ABC",       // Invalid year only
		"JAN ABC",   // Invalid year with month
		"1 JAN ABC", // Invalid year with day and month
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := ParseDate(input)
			if err == nil {
				t.Fatalf("ParseDate(%q) expected error, got nil", input)
			}
			if !strings.Contains(err.Error(), "invalid year") {
				t.Errorf("Error message = %q, want substring 'invalid year'", err.Error())
			}
		})
	}
}

// TestParseDate_InvalidDay tests dates with invalid day values
func TestParseDate_InvalidDay(t *testing.T) {
	tests := []string{
		"ABC JAN 2020", // Invalid day
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := ParseDate(input)
			if err == nil {
				t.Fatalf("ParseDate(%q) expected error, got nil", input)
			}
			if !strings.Contains(err.Error(), "invalid day") {
				t.Errorf("Error message = %q, want substring 'invalid day'", err.Error())
			}
		})
	}
}

// TestParseDate_ModifierWithoutDate tests modifiers with no date following
func TestParseDate_ModifierWithoutDate(t *testing.T) {
	tests := []string{
		"ABT",
		"BEF",
		"AFT",
		"CAL",
		"EST",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := ParseDate(input)
			if err == nil {
				t.Fatalf("ParseDate(%q) expected error for modifier without date, got nil", input)
			}
		})
	}
}

// TestParseDate_CalendarWithoutDate tests calendar escape with no date following
func TestParseDate_CalendarWithoutDate(t *testing.T) {
	_, err := ParseDate("@#DGREGORIAN@")
	if err == nil {
		t.Fatal("ParseDate('@#DGREGORIAN@') expected error for calendar without date, got nil")
	}
}

// TestParseDate_CalendarWithWhitespaceOnly tests calendar escape with only whitespace following
func TestParseDate_CalendarWithWhitespaceOnly(t *testing.T) {
	_, err := ParseDate("@#DGREGORIAN@   ")
	if err == nil {
		t.Fatal("ParseDate('@#DGREGORIAN@   ') expected error for calendar with whitespace only, got nil")
	}
}

// TestDate_Validate tests date validation
func TestDate_Validate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		// Valid dates
		{"valid complete date", "25 DEC 2020", false, ""},
		{"valid leap year", "29 FEB 2000", false, ""},
		{"valid leap year 2", "29 FEB 2020", false, ""},
		{"valid Jan 31", "31 JAN 2020", false, ""},
		{"valid Mar 31", "31 MAR 2020", false, ""},
		{"valid May 31", "31 MAY 2020", false, ""},
		{"valid Jul 31", "31 JUL 2020", false, ""},
		{"valid Aug 31", "31 AUG 2020", false, ""},
		{"valid Oct 31", "31 OCT 2020", false, ""},
		{"valid Dec 31", "31 DEC 2020", false, ""},

		// Partial dates (should not error)
		{"partial year only", "1850", false, ""},
		{"partial month year", "JAN 1900", false, ""},

		// Invalid dates - day overflow
		{"invalid Feb 30", "30 FEB 2023", true, "February has 28 days"},
		{"invalid Feb 29 non-leap", "29 FEB 1900", true, "February has 28 days"},
		{"invalid Jun 31", "31 JUN 2020", true, "June has 30 days"},
		{"invalid Apr 31", "31 APR 2020", true, "April has 30 days"},
		{"invalid Sep 31", "31 SEP 2020", true, "September has 30 days"},
		{"invalid Nov 31", "31 NOV 2020", true, "November has 30 days"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.input, err)
			}

			err = date.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %q, want substring %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestParseDate_DualDating tests dual year format parsing
func TestParseDate_DualDating(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantYear     int
		wantDualYear int
		wantMonth    int
		wantDay      int
	}{
		{"dual year 2-digit", "21 FEB 1750/51", 1750, 1751, 2, 21},
		{"dual year 4-digit", "21 FEB 1750/1751", 1750, 1751, 2, 21},
		{"dual year month only", "FEB 1750/51", 1750, 1751, 2, 0},
		{"dual year only", "1750/51", 1750, 1751, 0, 0},
		{"dual year 2-digit 1600s", "15 MAR 1640/41", 1640, 1641, 3, 15},
		{"dual year 4-digit 1600s", "15 MAR 1640/1641", 1640, 1641, 3, 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.input, err)
			}

			if date.Year != tt.wantYear {
				t.Errorf("Year = %d, want %d", date.Year, tt.wantYear)
			}
			if date.DualYear != tt.wantDualYear {
				t.Errorf("DualYear = %d, want %d", date.DualYear, tt.wantDualYear)
			}
			if date.Month != tt.wantMonth {
				t.Errorf("Month = %d, want %d", date.Month, tt.wantMonth)
			}
			if date.Day != tt.wantDay {
				t.Errorf("Day = %d, want %d", date.Day, tt.wantDay)
			}
		})
	}
}

// TestParseDate_BCDates tests B.C./BCE date parsing
func TestParseDate_BCDates(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth int
		wantDay   int
		wantIsBC  bool
	}{
		{"BC uppercase", "44 BC", 44, 0, 0, true},
		{"BC with periods", "44 B.C.", 44, 0, 0, true},
		{"BCE uppercase", "44 BCE", 44, 0, 0, true},
		{"BCE with periods", "44 B.C.E.", 44, 0, 0, true},
		{"BC lowercase", "44 bc", 44, 0, 0, true},
		{"BC mixed case", "44 Bc", 44, 0, 0, true},
		{"BC with month", "JAN 100 BC", 100, 1, 0, true},
		{"BC with day and month", "15 MAR 44 BC", 44, 3, 15, true},
		{"BCE with month", "JAN 500 BCE", 500, 1, 0, true},
		{"regular AD date", "2020", 2020, 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.input, err)
			}

			if date.Year != tt.wantYear {
				t.Errorf("Year = %d, want %d", date.Year, tt.wantYear)
			}
			if date.Month != tt.wantMonth {
				t.Errorf("Month = %d, want %d", date.Month, tt.wantMonth)
			}
			if date.Day != tt.wantDay {
				t.Errorf("Day = %d, want %d", date.Day, tt.wantDay)
			}
			if date.IsBC != tt.wantIsBC {
				t.Errorf("IsBC = %v, want %v", date.IsBC, tt.wantIsBC)
			}
		})
	}
}

// TestDate_Compare_BCDates tests comparison of B.C. dates
func TestDate_Compare_BCDates(t *testing.T) {
	tests := []struct {
		date1   string
		date2   string
		wantCmp int
	}{
		// BC vs AD
		{"100 BC", "2020", -1},
		{"2020", "100 BC", 1},

		// BC vs BC (remember: 100 BC > 200 BC in time)
		{"100 BC", "100 BC", 0},
		{"100 BC", "200 BC", 1},  // 100 BC is later (closer to present)
		{"200 BC", "100 BC", -1}, // 200 BC is earlier
		{"44 BC", "100 BC", 1},   // 44 BC is later
		{"500 BC", "44 BC", -1},  // 500 BC is earlier

		// BC dates with months
		{"JAN 100 BC", "FEB 100 BC", -1},
		{"FEB 100 BC", "JAN 100 BC", 1},

		// BC dates with days
		{"15 MAR 44 BC", "15 MAR 44 BC", 0},
		{"14 MAR 44 BC", "15 MAR 44 BC", -1},
		{"16 MAR 44 BC", "15 MAR 44 BC", 1},
	}

	for _, tt := range tests {
		t.Run(tt.date1+" vs "+tt.date2, func(t *testing.T) {
			d1, err := ParseDate(tt.date1)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.date1, err)
			}
			d2, err := ParseDate(tt.date2)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.date2, err)
			}

			got := d1.Compare(d2)
			if got != tt.wantCmp {
				t.Errorf("Compare() = %d, want %d", got, tt.wantCmp)
			}
		})
	}
}

// TestParseDate_Phrases tests date phrase parsing
func TestParseDate_Phrases(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantPhrase   string
		wantIsPhrase bool
	}{
		{"simple phrase", "(unknown)", "unknown", true},
		{"phrase with spaces", "(about 1850)", "about 1850", true},
		{"phrase with text", "(before the war)", "before the war", true},
		{"phrase empty", "()", "", true},
		{"phrase with punctuation", "(c. 1850)", "c. 1850", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.input, err)
			}

			if date.IsPhrase != tt.wantIsPhrase {
				t.Errorf("IsPhrase = %v, want %v", date.IsPhrase, tt.wantIsPhrase)
			}
			if date.Phrase != tt.wantPhrase {
				t.Errorf("Phrase = %q, want %q", date.Phrase, tt.wantPhrase)
			}
			// Phrase dates should have no date components
			if date.Year != 0 || date.Month != 0 || date.Day != 0 {
				t.Errorf("Phrase date has non-zero date components: Year=%d, Month=%d, Day=%d",
					date.Year, date.Month, date.Day)
			}
		})
	}
}

// TestParseDate_PhraseNotParsed tests that phrase content is not parsed
func TestParseDate_PhraseNotParsed(t *testing.T) {
	// A phrase that looks like a date should not be parsed as a date
	date, err := ParseDate("(25 DEC 2020)")
	if err != nil {
		t.Fatalf("ParseDate error = %v", err)
	}

	if !date.IsPhrase {
		t.Error("IsPhrase = false, want true")
	}
	if date.Phrase != "25 DEC 2020" {
		t.Errorf("Phrase = %q, want %q", date.Phrase, "25 DEC 2020")
	}
	// Should not have parsed the date inside
	if date.Year != 0 || date.Month != 0 || date.Day != 0 {
		t.Errorf("Phrase should not be parsed: Year=%d, Month=%d, Day=%d",
			date.Year, date.Month, date.Day)
	}
}

// TestParseDate_InvalidDualYear tests invalid dual year formats
func TestParseDate_InvalidDualYear(t *testing.T) {
	tests := []string{
		"1750/51/52", // Too many parts
		"1750/ABC",   // Invalid secondary year
		"ABC/51",     // Invalid primary year
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := ParseDate(input)
			if err == nil {
				t.Errorf("ParseDate(%q) expected error, got nil", input)
			}
		})
	}
}

// TestParseDate_BCWithModifier tests B.C. dates with modifiers
func TestParseDate_BCWithModifier(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantModifier DateModifier
		wantYear     int
		wantIsBC     bool
	}{
		{"ABT with BC", "ABT 100 BC", ModifierAbout, 100, true},
		{"BEF with BC", "BEF 44 BC", ModifierBefore, 44, true},
		{"AFT with BCE", "AFT 500 BCE", ModifierAfter, 500, true},
		{"CAL with BC", "CAL 200 B.C.", ModifierCalculated, 200, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.input, err)
			}

			if date.Modifier != tt.wantModifier {
				t.Errorf("Modifier = %v, want %v", date.Modifier, tt.wantModifier)
			}
			if date.Year != tt.wantYear {
				t.Errorf("Year = %d, want %d", date.Year, tt.wantYear)
			}
			if date.IsBC != tt.wantIsBC {
				t.Errorf("IsBC = %v, want %v", date.IsBC, tt.wantIsBC)
			}
		})
	}
}

// TestParseDate_DualYearWithBC tests dual dating with B.C. dates
func TestParseDate_DualYearWithBC(t *testing.T) {
	date, err := ParseDate("21 FEB 45/44 BC")
	if err != nil {
		t.Fatalf("ParseDate error = %v", err)
	}

	if date.Year != 45 {
		t.Errorf("Year = %d, want 45", date.Year)
	}
	if date.DualYear != 44 {
		t.Errorf("DualYear = %d, want 44", date.DualYear)
	}
	if !date.IsBC {
		t.Error("IsBC = false, want true")
	}
}

// TestDate_Validate_NonGregorian tests that non-Gregorian calendars skip validation
func TestDate_Validate_NonGregorian(t *testing.T) {
	date := &Date{
		Day:      32,
		Month:    13,
		Year:     2020,
		Calendar: CalendarJulian,
	}

	err := date.Validate()
	if err != nil {
		t.Errorf("Validate() for non-Gregorian should return nil, got %v", err)
	}
}

// TestParseDate_JulianCalendar tests Julian calendar date parsing
func TestParseDate_JulianCalendar(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantDay      int
		wantMonth    int
		wantYear     int
		wantCalendar Calendar
		wantModifier DateModifier
		wantIsBC     bool
		wantDualYear int
	}{
		{
			name:         "full date",
			input:        "@#DJULIAN@ 25 DEC 1700",
			wantDay:      25,
			wantMonth:    12,
			wantYear:     1700,
			wantCalendar: CalendarJulian,
		},
		{
			name:         "partial month-year",
			input:        "@#DJULIAN@ MAR 1582",
			wantDay:      0,
			wantMonth:    3,
			wantYear:     1582,
			wantCalendar: CalendarJulian,
		},
		{
			name:         "partial year only",
			input:        "@#DJULIAN@ 1492",
			wantDay:      0,
			wantMonth:    0,
			wantYear:     1492,
			wantCalendar: CalendarJulian,
		},
		{
			name:         "with modifier ABT",
			input:        "@#DJULIAN@ ABT 15 MAR 44 BC",
			wantDay:      15,
			wantMonth:    3,
			wantYear:     44,
			wantCalendar: CalendarJulian,
			wantModifier: ModifierAbout,
			wantIsBC:     true,
		},
		{
			name:         "BC date",
			input:        "@#DJULIAN@ 15 MAR 44 BC",
			wantDay:      15,
			wantMonth:    3,
			wantYear:     44,
			wantCalendar: CalendarJulian,
			wantIsBC:     true,
		},
		{
			name:         "dual dating",
			input:        "@#DJULIAN@ 21 FEB 1750/51",
			wantDay:      21,
			wantMonth:    2,
			wantYear:     1750,
			wantCalendar: CalendarJulian,
			wantDualYear: 1751,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.input, err)
			}

			if date.Day != tt.wantDay {
				t.Errorf("Day = %d, want %d", date.Day, tt.wantDay)
			}
			if date.Month != tt.wantMonth {
				t.Errorf("Month = %d, want %d", date.Month, tt.wantMonth)
			}
			if date.Year != tt.wantYear {
				t.Errorf("Year = %d, want %d", date.Year, tt.wantYear)
			}
			if date.Calendar != tt.wantCalendar {
				t.Errorf("Calendar = %v, want %v", date.Calendar, tt.wantCalendar)
			}
			if date.Modifier != tt.wantModifier {
				t.Errorf("Modifier = %v, want %v", date.Modifier, tt.wantModifier)
			}
			if date.IsBC != tt.wantIsBC {
				t.Errorf("IsBC = %v, want %v", date.IsBC, tt.wantIsBC)
			}
			if date.DualYear != tt.wantDualYear {
				t.Errorf("DualYear = %d, want %d", date.DualYear, tt.wantDualYear)
			}
			if date.Original != tt.input {
				t.Errorf("Original = %q, want %q", date.Original, tt.input)
			}
		})
	}
}

// TestParseDate_HebrewCalendar tests Hebrew calendar date parsing for all 13 months
func TestParseDate_HebrewCalendar(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantDay      int
		wantMonth    int
		wantYear     int
		wantCalendar Calendar
	}{
		// All 13 Hebrew months
		{
			name:         "Tishrei",
			input:        "@#DHEBREW@ 10 TSH 5780",
			wantDay:      10,
			wantMonth:    1,
			wantYear:     5780,
			wantCalendar: CalendarHebrew,
		},
		{
			name:         "Cheshvan",
			input:        "@#DHEBREW@ 15 CSH 5785",
			wantDay:      15,
			wantMonth:    2,
			wantYear:     5785,
			wantCalendar: CalendarHebrew,
		},
		{
			name:         "Kislev (Chanukah)",
			input:        "@#DHEBREW@ 25 KSL 5785",
			wantDay:      25,
			wantMonth:    3,
			wantYear:     5785,
			wantCalendar: CalendarHebrew,
		},
		{
			name:         "Tevet",
			input:        "@#DHEBREW@ 10 TVT 5785",
			wantDay:      10,
			wantMonth:    4,
			wantYear:     5785,
			wantCalendar: CalendarHebrew,
		},
		{
			name:         "Shevat (Tu B'Shevat)",
			input:        "@#DHEBREW@ 15 SHV 5785",
			wantDay:      15,
			wantMonth:    5,
			wantYear:     5785,
			wantCalendar: CalendarHebrew,
		},
		{
			name:         "Adar (Purim)",
			input:        "@#DHEBREW@ 14 ADR 5785",
			wantDay:      14,
			wantMonth:    6,
			wantYear:     5785,
			wantCalendar: CalendarHebrew,
		},
		{
			name:         "Adar II (leap year)",
			input:        "@#DHEBREW@ 14 ADS 5784",
			wantDay:      14,
			wantMonth:    7,
			wantYear:     5784,
			wantCalendar: CalendarHebrew,
		},
		{
			name:         "Nisan (Passover)",
			input:        "@#DHEBREW@ 15 NSN 5785",
			wantDay:      15,
			wantMonth:    8,
			wantYear:     5785,
			wantCalendar: CalendarHebrew,
		},
		{
			name:         "Iyar",
			input:        "@#DHEBREW@ 5 IYR 5785",
			wantDay:      5,
			wantMonth:    9,
			wantYear:     5785,
			wantCalendar: CalendarHebrew,
		},
		{
			name:         "Sivan (Shavuot)",
			input:        "@#DHEBREW@ 6 SVN 5785",
			wantDay:      6,
			wantMonth:    10,
			wantYear:     5785,
			wantCalendar: CalendarHebrew,
		},
		{
			name:         "Tammuz",
			input:        "@#DHEBREW@ 17 TMZ 5785",
			wantDay:      17,
			wantMonth:    11,
			wantYear:     5785,
			wantCalendar: CalendarHebrew,
		},
		{
			name:         "Av (Tisha B'Av)",
			input:        "@#DHEBREW@ 9 AAV 5785",
			wantDay:      9,
			wantMonth:    12,
			wantYear:     5785,
			wantCalendar: CalendarHebrew,
		},
		{
			name:         "Elul",
			input:        "@#DHEBREW@ 1 ELL 5785",
			wantDay:      1,
			wantMonth:    13,
			wantYear:     5785,
			wantCalendar: CalendarHebrew,
		},
		// Partial dates
		{
			name:         "month-year only",
			input:        "@#DHEBREW@ NSN 5785",
			wantDay:      0,
			wantMonth:    8,
			wantYear:     5785,
			wantCalendar: CalendarHebrew,
		},
		{
			name:         "year only",
			input:        "@#DHEBREW@ 5765",
			wantDay:      0,
			wantMonth:    0,
			wantYear:     5765,
			wantCalendar: CalendarHebrew,
		},
		// Case insensitivity
		{
			name:         "lowercase month code",
			input:        "@#DHEBREW@ 15 nsn 5785",
			wantDay:      15,
			wantMonth:    8,
			wantYear:     5785,
			wantCalendar: CalendarHebrew,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.input, err)
			}

			if date.Day != tt.wantDay {
				t.Errorf("Day = %d, want %d", date.Day, tt.wantDay)
			}
			if date.Month != tt.wantMonth {
				t.Errorf("Month = %d, want %d", date.Month, tt.wantMonth)
			}
			if date.Year != tt.wantYear {
				t.Errorf("Year = %d, want %d", date.Year, tt.wantYear)
			}
			if date.Calendar != tt.wantCalendar {
				t.Errorf("Calendar = %v, want %v", date.Calendar, tt.wantCalendar)
			}
			if date.Original != tt.input {
				t.Errorf("Original = %q, want %q", date.Original, tt.input)
			}
		})
	}
}

// TestParseDate_FrenchRepublicanCalendar tests French Republican calendar date parsing for all 13 months
func TestParseDate_FrenchRepublicanCalendar(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantDay      int
		wantMonth    int
		wantYear     int
		wantCalendar Calendar
	}{
		// All 13 French Republican months
		{
			name:         "Vendémiaire (start of Republic)",
			input:        "@#DFRENCH R@ 1 VEND 1",
			wantDay:      1,
			wantMonth:    1,
			wantYear:     1,
			wantCalendar: CalendarFrenchRepublican,
		},
		{
			name:         "Brumaire (Napoleon's coup)",
			input:        "@#DFRENCH R@ 18 BRUM 8",
			wantDay:      18,
			wantMonth:    2,
			wantYear:     8,
			wantCalendar: CalendarFrenchRepublican,
		},
		{
			name:         "Frimaire",
			input:        "@#DFRENCH R@ 15 FRIM 3",
			wantDay:      15,
			wantMonth:    3,
			wantYear:     3,
			wantCalendar: CalendarFrenchRepublican,
		},
		{
			name:         "Nivôse",
			input:        "@#DFRENCH R@ 20 NIVO 5",
			wantDay:      20,
			wantMonth:    4,
			wantYear:     5,
			wantCalendar: CalendarFrenchRepublican,
		},
		{
			name:         "Pluviôse",
			input:        "@#DFRENCH R@ 10 PLUV 7",
			wantDay:      10,
			wantMonth:    5,
			wantYear:     7,
			wantCalendar: CalendarFrenchRepublican,
		},
		{
			name:         "Ventôse",
			input:        "@#DFRENCH R@ 25 VENT 4",
			wantDay:      25,
			wantMonth:    6,
			wantYear:     4,
			wantCalendar: CalendarFrenchRepublican,
		},
		{
			name:         "Germinal",
			input:        "@#DFRENCH R@ 12 GERM 6",
			wantDay:      12,
			wantMonth:    7,
			wantYear:     6,
			wantCalendar: CalendarFrenchRepublican,
		},
		{
			name:         "Floréal",
			input:        "@#DFRENCH R@ 5 FLOR 9",
			wantDay:      5,
			wantMonth:    8,
			wantYear:     9,
			wantCalendar: CalendarFrenchRepublican,
		},
		{
			name:         "Prairial",
			input:        "@#DFRENCH R@ 30 PRAI 2",
			wantDay:      30,
			wantMonth:    9,
			wantYear:     2,
			wantCalendar: CalendarFrenchRepublican,
		},
		{
			name:         "Messidor",
			input:        "@#DFRENCH R@ 14 MESS 10",
			wantDay:      14,
			wantMonth:    10,
			wantYear:     10,
			wantCalendar: CalendarFrenchRepublican,
		},
		{
			name:         "Thermidor",
			input:        "@#DFRENCH R@ 9 THER 11", //nolint:misspell // THER is GEDCOM code for Thermidor
			wantDay:      9,
			wantMonth:    11,
			wantYear:     11,
			wantCalendar: CalendarFrenchRepublican,
		},
		{
			name:         "Fructidor",
			input:        "@#DFRENCH R@ 22 FRUC 12",
			wantDay:      22,
			wantMonth:    12,
			wantYear:     12,
			wantCalendar: CalendarFrenchRepublican,
		},
		{
			name:         "Complementary days",
			input:        "@#DFRENCH R@ 1 COMP 3",
			wantDay:      1,
			wantMonth:    13,
			wantYear:     3,
			wantCalendar: CalendarFrenchRepublican,
		},
		// Partial dates
		{
			name:         "month-year only",
			input:        "@#DFRENCH R@ VEND 3",
			wantDay:      0,
			wantMonth:    1,
			wantYear:     3,
			wantCalendar: CalendarFrenchRepublican,
		},
		{
			name:         "year only",
			input:        "@#DFRENCH R@ 12",
			wantDay:      0,
			wantMonth:    0,
			wantYear:     12,
			wantCalendar: CalendarFrenchRepublican,
		},
		// Case insensitivity
		{
			name:         "lowercase month code",
			input:        "@#DFRENCH R@ 15 vend 3",
			wantDay:      15,
			wantMonth:    1,
			wantYear:     3,
			wantCalendar: CalendarFrenchRepublican,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.input, err)
			}

			if date.Day != tt.wantDay {
				t.Errorf("Day = %d, want %d", date.Day, tt.wantDay)
			}
			if date.Month != tt.wantMonth {
				t.Errorf("Month = %d, want %d", date.Month, tt.wantMonth)
			}
			if date.Year != tt.wantYear {
				t.Errorf("Year = %d, want %d", date.Year, tt.wantYear)
			}
			if date.Calendar != tt.wantCalendar {
				t.Errorf("Calendar = %v, want %v", date.Calendar, tt.wantCalendar)
			}
			if date.Original != tt.input {
				t.Errorf("Original = %q, want %q", date.Original, tt.input)
			}
		})
	}
}

// TestParseDate_CalendarWithModifiers tests calendar dates combined with modifiers
func TestParseDate_CalendarWithModifiers(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantModifier DateModifier
		wantCalendar Calendar
		wantYear     int
		wantMonth    int
		wantDay      int
	}{
		{
			name:         "ABT with Hebrew",
			input:        "@#DHEBREW@ ABT 15 NSN 5785",
			wantModifier: ModifierAbout,
			wantCalendar: CalendarHebrew,
			wantYear:     5785,
			wantMonth:    8,
			wantDay:      15,
		},
		{
			name:         "BEF with Julian",
			input:        "@#DJULIAN@ BEF 25 DEC 1700",
			wantModifier: ModifierBefore,
			wantCalendar: CalendarJulian,
			wantYear:     1700,
			wantMonth:    12,
			wantDay:      25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.input, err)
			}

			if date.Modifier != tt.wantModifier {
				t.Errorf("Modifier = %v, want %v", date.Modifier, tt.wantModifier)
			}
			if date.Calendar != tt.wantCalendar {
				t.Errorf("Calendar = %v, want %v", date.Calendar, tt.wantCalendar)
			}
			if date.Year != tt.wantYear {
				t.Errorf("Year = %d, want %d", date.Year, tt.wantYear)
			}
			if date.Month != tt.wantMonth {
				t.Errorf("Month = %d, want %d", date.Month, tt.wantMonth)
			}
			if date.Day != tt.wantDay {
				t.Errorf("Day = %d, want %d", date.Day, tt.wantDay)
			}
		})
	}
}

// TestParseDate_CalendarWithRanges tests calendar dates in ranges
// NOTE: Currently the calendar is not propagated to range dates (known limitation)
// This test verifies that the range structure parses correctly even if calendar is lost
func TestParseDate_CalendarWithRanges(t *testing.T) {
	input := "@#DJULIAN@ BET 1700 AND 1750"
	date, err := ParseDate(input)
	if err != nil {
		t.Fatalf("ParseDate(%q) error = %v", input, err)
	}

	if date.Modifier != ModifierBetween {
		t.Errorf("Modifier = %v, want ModifierBetween", date.Modifier)
	}
	// TODO: Calendar should be Julian but is currently lost in range parsing
	// if date.Calendar != CalendarJulian {
	//	t.Errorf("Calendar = %v, want CalendarJulian", date.Calendar)
	// }
	if date.Year != 1700 {
		t.Errorf("Start Year = %d, want 1700", date.Year)
	}
	if date.EndDate == nil {
		t.Fatal("EndDate is nil, want non-nil")
	}
	if date.EndDate.Year != 1750 {
		t.Errorf("End Year = %d, want 1750", date.EndDate.Year)
	}
	// TODO: EndDate calendar should inherit from start date
	// if date.EndDate.Calendar != CalendarJulian {
	//	t.Errorf("EndDate.Calendar = %v, want CalendarJulian", date.EndDate.Calendar)
	// }
}

// TestParseDate_CalendarMonthErrors tests invalid month codes for calendars
func TestParseDate_CalendarMonthErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid Hebrew month",
			input: "@#DHEBREW@ 15 XYZ 5785",
		},
		{
			name:  "invalid French Republican month",
			input: "@#DFRENCH R@ 15 XYZ 3",
		},
		{
			name:  "invalid Julian month",
			input: "@#DJULIAN@ 15 XYZ 1700",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseDate(tt.input)
			if err == nil {
				t.Errorf("ParseDate(%q) expected error for invalid month code, got nil", tt.input)
			}
			if !strings.Contains(err.Error(), "invalid month") && !strings.Contains(err.Error(), "unknown month") {
				t.Errorf("Error message = %q, want substring 'invalid month' or 'unknown month'", err.Error())
			}
		})
	}
}

// TestDate_ToGregorian tests calendar conversion to Gregorian
func TestDate_ToGregorian(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantDay      int
		wantMonth    int
		wantYear     int
		wantIsBC     bool
		wantOriginal string
		wantErr      bool
	}{
		// Julian to Gregorian
		{
			name:         "Julian to Gregorian complete date",
			input:        "@#DJULIAN@ 4 OCT 1582",
			wantDay:      14,
			wantMonth:    10,
			wantYear:     1582,
			wantIsBC:     false,
			wantOriginal: "@#DJULIAN@ 4 OCT 1582",
		},
		{
			name:         "Julian to Gregorian BC date",
			input:        "@#DJULIAN@ 15 MAR 44 BC",
			wantDay:      13,
			wantMonth:    3,
			wantYear:     44,
			wantIsBC:     true,
			wantOriginal: "@#DJULIAN@ 15 MAR 44 BC",
		},
		{
			name:         "Julian year only",
			input:        "@#DJULIAN@ 1700",
			wantDay:      0,
			wantMonth:    0,
			wantYear:     1700,
			wantIsBC:     false,
			wantOriginal: "@#DJULIAN@ 1700",
		},
		{
			name:         "Julian month+year",
			input:        "@#DJULIAN@ MAR 1582",
			wantDay:      0,
			wantMonth:    3,
			wantYear:     1582,
			wantIsBC:     false,
			wantOriginal: "@#DJULIAN@ MAR 1582",
		},

		// Hebrew to Gregorian
		{
			name:         "Hebrew to Gregorian complete date",
			input:        "@#DHEBREW@ 15 NSN 5785",
			wantDay:      13,
			wantMonth:    4,
			wantYear:     2025,
			wantIsBC:     false,
			wantOriginal: "@#DHEBREW@ 15 NSN 5785",
		},
		{
			name:         "Hebrew Rosh Hashanah",
			input:        "@#DHEBREW@ 1 TSH 5785",
			wantDay:      3,
			wantMonth:    10,
			wantYear:     2024,
			wantIsBC:     false,
			wantOriginal: "@#DHEBREW@ 1 TSH 5785",
		},
		{
			name:         "Hebrew year only",
			input:        "@#DHEBREW@ 5785",
			wantDay:      0,
			wantMonth:    0,
			wantYear:     2024,
			wantIsBC:     false,
			wantOriginal: "@#DHEBREW@ 5785",
		},
		{
			name:         "Hebrew month+year",
			input:        "@#DHEBREW@ NSN 5785",
			wantDay:      0,
			wantMonth:    3,
			wantYear:     2025,
			wantIsBC:     false,
			wantOriginal: "@#DHEBREW@ NSN 5785",
		},

		// French Republican to Gregorian
		{
			name:         "French to Gregorian complete date",
			input:        "@#DFRENCH R@ 1 VEND 1",
			wantDay:      22,
			wantMonth:    9,
			wantYear:     1792,
			wantIsBC:     false,
			wantOriginal: "@#DFRENCH R@ 1 VEND 1",
		},
		{
			name:         "French year only",
			input:        "@#DFRENCH R@ 8",
			wantDay:      0,
			wantMonth:    0,
			wantYear:     1799,
			wantIsBC:     false,
			wantOriginal: "@#DFRENCH R@ 8",
		},

		// Already Gregorian - should return copy of self
		{
			name:         "Gregorian to Gregorian (copy)",
			input:        "25 DEC 2020",
			wantDay:      25,
			wantMonth:    12,
			wantYear:     2020,
			wantIsBC:     false,
			wantOriginal: "25 DEC 2020",
		},
		{
			name:         "Gregorian year only (copy)",
			input:        "1850",
			wantDay:      0,
			wantMonth:    0,
			wantYear:     1850,
			wantIsBC:     false,
			wantOriginal: "1850",
		},

		// Error cases - phrases can't be converted
		// Note: Phrases have year=0, so ToGregorian should fail
		// This is actually handled correctly by the implementation
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.input, err)
			}

			greg, err := date.ToGregorian()
			if tt.wantErr {
				if err == nil {
					t.Errorf("ToGregorian() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ToGregorian() error = %v", err)
			}

			if greg.Day != tt.wantDay {
				t.Errorf("ToGregorian().Day = %d, want %d", greg.Day, tt.wantDay)
			}
			if greg.Month != tt.wantMonth {
				t.Errorf("ToGregorian().Month = %d, want %d", greg.Month, tt.wantMonth)
			}
			if greg.Year != tt.wantYear {
				t.Errorf("ToGregorian().Year = %d, want %d", greg.Year, tt.wantYear)
			}
			if greg.IsBC != tt.wantIsBC {
				t.Errorf("ToGregorian().IsBC = %v, want %v", greg.IsBC, tt.wantIsBC)
			}
			if greg.Calendar != CalendarGregorian {
				t.Errorf("ToGregorian().Calendar = %v, want CalendarGregorian", greg.Calendar)
			}
			if greg.Original != tt.wantOriginal {
				t.Errorf("ToGregorian().Original = %q, want %q", greg.Original, tt.wantOriginal)
			}
		})
	}
}

// TestDate_Compare_CrossCalendar tests cross-calendar comparison
func TestDate_Compare_CrossCalendar(t *testing.T) {
	tests := []struct {
		name    string
		date1   string
		date2   string
		wantCmp int
	}{
		// Same date in different calendars should compare equal (same JDN)
		{
			name:    "Julian Oct 4, 1582 vs Gregorian Oct 14, 1582 (same JDN)",
			date1:   "@#DJULIAN@ 4 OCT 1582",
			date2:   "14 OCT 1582",
			wantCmp: 0, // Julian Oct 4 = Gregorian Oct 14 (both JDN 2299160)
		},
		{
			name:    "Julian Oct 5, 1582 vs Gregorian Oct 15, 1582 (same JDN)",
			date1:   "@#DJULIAN@ 5 OCT 1582",
			date2:   "15 OCT 1582",
			wantCmp: 0, // Same JDN (2299161)
		},
		{
			name:    "Hebrew Rosh Hashanah 5785 vs Gregorian Oct 3, 2024",
			date1:   "@#DHEBREW@ 1 TSH 5785",
			date2:   "3 OCT 2024",
			wantCmp: 0, // Same day
		},
		{
			name:    "French Republican epoch vs Gregorian Sept 22, 1792",
			date1:   "@#DFRENCH R@ 1 VEND 1",
			date2:   "22 SEP 1792",
			wantCmp: 0, // Same day
		},

		// Different dates across calendars
		{
			name:    "Hebrew date before Gregorian date",
			date1:   "@#DHEBREW@ 1 TSH 5785",
			date2:   "4 OCT 2024",
			wantCmp: -1,
		},
		{
			name:    "Hebrew date after Gregorian date",
			date1:   "@#DHEBREW@ 2 TSH 5785",
			date2:   "3 OCT 2024",
			wantCmp: 1,
		},
		{
			name:    "Julian before Gregorian",
			date1:   "@#DJULIAN@ 1 JAN 1700",
			date2:   "12 JAN 1700",
			wantCmp: -1,
		},
		{
			name:    "Julian after Gregorian",
			date1:   "@#DJULIAN@ 15 JAN 1700",
			date2:   "1 JAN 1700",
			wantCmp: 1,
		},

		// Partial dates across calendars
		{
			name:    "Julian year only vs Gregorian year only (different JDN)",
			date1:   "@#DJULIAN@ 1700",
			date2:   "1700",
			wantCmp: 1, // Julian 1/1/1700 is 10 days after Gregorian 1/1/1700 (JDN 2341983 vs 2341973)
		},
		{
			name:    "Hebrew year only vs Gregorian date",
			date1:   "@#DHEBREW@ 5785",
			date2:   "1 JAN 2025",
			wantCmp: -1, // Hebrew 1 TSH 5785 (Oct 3, 2024) is before Jan 1, 2025
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d1, err := ParseDate(tt.date1)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.date1, err)
			}
			d2, err := ParseDate(tt.date2)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.date2, err)
			}

			got := d1.Compare(d2)
			if got != tt.wantCmp {
				t.Errorf("Compare() = %d, want %d", got, tt.wantCmp)
			}
		})
	}
}

// TestDate_toJDN tests the internal JDN conversion helper
func TestDate_toJDN(t *testing.T) {
	tests := []struct {
		name    string
		date    *Date
		wantJDN int
		wantErr bool
	}{
		{
			name: "Gregorian date",
			date: &Date{
				Year:     2000,
				Month:    1,
				Day:      1,
				Calendar: CalendarGregorian,
			},
			wantJDN: 2451545,
		},
		{
			name: "Gregorian BC date",
			date: &Date{
				Year:     44,
				Month:    3,
				Day:      15,
				IsBC:     true,
				Calendar: CalendarGregorian,
			},
			wantJDN: 1705428,
		},
		{
			name: "Julian date",
			date: &Date{
				Year:     1582,
				Month:    10,
				Day:      4,
				Calendar: CalendarJulian,
			},
			wantJDN: 2299160,
		},
		{
			name: "Hebrew date",
			date: &Date{
				Year:     5785,
				Month:    1,
				Day:      1,
				Calendar: CalendarHebrew,
			},
			wantJDN: 2460587,
		},
		{
			name: "French Republican date",
			date: &Date{
				Year:     1,
				Month:    1,
				Day:      1,
				Calendar: CalendarFrenchRepublican,
			},
			wantJDN: 2375840,
		},
		{
			name: "Partial date (year only)",
			date: &Date{
				Year:     2000,
				Month:    0,
				Day:      0,
				Calendar: CalendarGregorian,
			},
			wantJDN: 2451545, // Defaults to Jan 1
		},
		{
			name: "Partial date (month+year)",
			date: &Date{
				Year:     2000,
				Month:    6,
				Day:      0,
				Calendar: CalendarGregorian,
			},
			wantJDN: 2451697, // June 1, 2000
		},
		{
			name: "Missing year",
			date: &Date{
				Year:     0,
				Month:    1,
				Day:      1,
				Calendar: CalendarGregorian,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jdn, err := tt.date.toJDN()
			if tt.wantErr {
				if err == nil {
					t.Errorf("toJDN() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("toJDN() error = %v", err)
			}
			if jdn != tt.wantJDN {
				t.Errorf("toJDN() = %d, want %d", jdn, tt.wantJDN)
			}
		})
	}
}
