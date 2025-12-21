package gedcom

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Calendar represents the calendar system used for a date.
type Calendar int

const (
	// CalendarGregorian is the default Gregorian calendar
	CalendarGregorian Calendar = iota
	// CalendarJulian is the Julian calendar
	CalendarJulian
	// CalendarHebrew is the Hebrew calendar
	CalendarHebrew
	// CalendarFrenchRepublican is the French Republican calendar
	CalendarFrenchRepublican
)

// String returns the string representation of the calendar.
func (c Calendar) String() string {
	switch c {
	case CalendarGregorian:
		return "Gregorian"
	case CalendarJulian:
		return "Julian"
	case CalendarHebrew:
		return "Hebrew"
	case CalendarFrenchRepublican:
		return "French Republican"
	default:
		return "Unknown"
	}
}

// DateModifier represents modifiers for date precision and ranges.
type DateModifier int

const (
	// ModifierNone indicates an exact date with no modifier
	ModifierNone DateModifier = iota
	// ModifierAbout indicates an approximate date (ABT)
	ModifierAbout
	// ModifierCalculated indicates a calculated date (CAL)
	ModifierCalculated
	// ModifierEstimated indicates an estimated date (EST)
	ModifierEstimated
	// ModifierBefore indicates a date before the specified date (BEF)
	ModifierBefore
	// ModifierAfter indicates a date after the specified date (AFT)
	ModifierAfter
	// ModifierBetween indicates a date range (BET...AND)
	ModifierBetween
	// ModifierFrom indicates a period starting from a date (FROM)
	ModifierFrom
	// ModifierTo indicates a period ending at a date (TO)
	ModifierTo
	// ModifierFromTo indicates a period with start and end dates (FROM...TO)
	ModifierFromTo
)

// String returns the string representation of the date modifier.
func (m DateModifier) String() string {
	switch m {
	case ModifierNone:
		return ""
	case ModifierAbout:
		return "ABT"
	case ModifierCalculated:
		return "CAL"
	case ModifierEstimated:
		return "EST"
	case ModifierBefore:
		return "BEF"
	case ModifierAfter:
		return "AFT"
	case ModifierBetween:
		return "BET"
	case ModifierFrom:
		return "FROM"
	case ModifierTo:
		return "TO"
	case ModifierFromTo:
		return "FROM TO"
	default:
		return "Unknown"
	}
}

// Date represents a parsed GEDCOM date with support for multiple calendar systems,
// modifiers, and date ranges/periods.
type Date struct {
	// Original is the raw GEDCOM date string (preserved for round-trip)
	Original string

	// Day is the day of the month (0 if unknown, 1-31)
	Day int

	// Month is the month (0 if unknown, 1-12)
	Month int

	// Year is the year (0 if unknown)
	Year int

	// Modifier indicates the type of date (exact, approximate, range, etc.)
	Modifier DateModifier

	// EndDate is populated for ranges (BET...AND) and periods (FROM...TO)
	EndDate *Date

	// Calendar indicates the calendar system (Gregorian, Julian, Hebrew, French Republican)
	Calendar Calendar
}

// monthNames maps three-letter month abbreviations to month numbers.
// Case-insensitive matching is handled by converting input to uppercase.
var monthNames = map[string]int{
	"JAN": 1, "FEB": 2, "MAR": 3, "APR": 4,
	"MAY": 5, "JUN": 6, "JUL": 7, "AUG": 8,
	"SEP": 9, "OCT": 10, "NOV": 11, "DEC": 12,
}

// ParseDate parses a GEDCOM date string into a Date struct.
// Supports exact dates, partial dates, modifiers, ranges, and periods.
// Currently only supports Gregorian calendar dates (Phase 1).
//
// Examples:
//   - "25 DEC 2020" -> exact date
//   - "JAN 1900" -> partial date (month and year)
//   - "1850" -> partial date (year only)
//   - "ABT 1850" -> approximate date
//   - "BET 1850 AND 1860" -> date range
//   - "FROM 1880 TO 1920" -> date period
func ParseDate(s string) (*Date, error) {
	if s == "" {
		return nil, fmt.Errorf("empty date string")
	}

	// Preserve original string
	original := s

	// Trim and normalize whitespace
	s = strings.TrimSpace(s)
	s = normalizeWhitespace(s)

	// Create date with original string
	date := &Date{
		Original: original,
		Calendar: CalendarGregorian, // Default calendar
	}

	// Check for calendar escape sequence
	calendar, rest, found := parseCalendarEscape(s)
	if found {
		date.Calendar = calendar
		s = rest

		// Phase 1: Only support Gregorian calendar
		if calendar != CalendarGregorian {
			return nil, fmt.Errorf("calendar %s not supported in Phase 1 (only Gregorian supported)", calendar)
		}
	}

	// Check for date modifier
	modifier, rest, found := parseModifier(s)
	if found {
		date.Modifier = modifier
		s = rest

		// Handle range and period modifiers
		switch modifier {
		case ModifierBetween:
			// BET date1 AND date2
			return parseDateRange(s, original)
		case ModifierFrom, ModifierTo, ModifierFromTo:
			// FROM date, TO date, or FROM date TO date
			return parseDatePeriod(s, original, modifier)
		}
	}

	// Parse the date components (day, month, year)
	if err := parseDateComponents(s, date); err != nil {
		return nil, err
	}

	return date, nil
}

// parseCalendarEscape parses a calendar escape sequence like @#DJULIAN@
// and returns the calendar type and the remaining string.
func parseCalendarEscape(s string) (Calendar, string, bool) {
	if !strings.HasPrefix(s, "@#D") {
		return CalendarGregorian, s, false
	}

	// Find the closing @
	end := strings.Index(s[3:], "@")
	if end == -1 {
		return CalendarGregorian, s, false
	}

	calendarName := s[3 : 3+end]
	rest := strings.TrimSpace(s[3+end+1:])

	var calendar Calendar
	switch strings.ToUpper(calendarName) {
	case "GREGORIAN":
		calendar = CalendarGregorian
	case "JULIAN":
		calendar = CalendarJulian
	case "HEBREW":
		calendar = CalendarHebrew
	case "FRENCH R":
		calendar = CalendarFrenchRepublican
	default:
		return CalendarGregorian, s, false
	}

	return calendar, rest, true
}

// parseModifier parses a date modifier keyword and returns the modifier type
// and the remaining string.
func parseModifier(s string) (DateModifier, string, bool) {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ModifierNone, s, false
	}

	firstWord := strings.ToUpper(fields[0])
	var modifier DateModifier
	var found bool

	switch firstWord {
	case "ABT":
		modifier = ModifierAbout
		found = true
	case "CAL":
		modifier = ModifierCalculated
		found = true
	case "EST":
		modifier = ModifierEstimated
		found = true
	case "BEF":
		modifier = ModifierBefore
		found = true
	case "AFT":
		modifier = ModifierAfter
		found = true
	case "BET":
		modifier = ModifierBetween
		found = true
	case "FROM":
		modifier = ModifierFrom
		found = true
	case "TO":
		modifier = ModifierTo
		found = true
	default:
		return ModifierNone, s, false
	}

	// Return the rest of the string after the modifier
	rest := strings.TrimSpace(strings.TrimPrefix(s, fields[0]))
	return modifier, rest, found
}

// parseDateRange parses a date range in the format "date1 AND date2".
func parseDateRange(s, original string) (*Date, error) {
	// Find the AND keyword
	andIndex := strings.Index(strings.ToUpper(s), " AND ")
	if andIndex == -1 {
		return nil, fmt.Errorf("invalid date range: missing AND keyword in '%s'", original)
	}

	// Parse the first date
	date1Str := strings.TrimSpace(s[:andIndex])
	date1 := &Date{Original: original, Calendar: CalendarGregorian, Modifier: ModifierBetween}
	if err := parseDateComponents(date1Str, date1); err != nil {
		return nil, fmt.Errorf("invalid start date in range: %w", err)
	}

	// Parse the second date
	date2Str := strings.TrimSpace(s[andIndex+5:]) // Skip " AND "
	date2 := &Date{Original: "", Calendar: CalendarGregorian}
	if err := parseDateComponents(date2Str, date2); err != nil {
		return nil, fmt.Errorf("invalid end date in range: %w", err)
	}

	date1.EndDate = date2
	return date1, nil
}

// parseDatePeriod parses a date period (FROM, TO, or FROM...TO).
func parseDatePeriod(s, original string, modifier DateModifier) (*Date, error) {
	// Check if there's a TO keyword for FROM...TO format
	toIndex := strings.Index(strings.ToUpper(s), " TO ")

	if modifier == ModifierFrom && toIndex != -1 {
		// FROM date1 TO date2
		date1Str := strings.TrimSpace(s[:toIndex])
		date1 := &Date{Original: original, Calendar: CalendarGregorian, Modifier: ModifierFromTo}
		if err := parseDateComponents(date1Str, date1); err != nil {
			return nil, fmt.Errorf("invalid start date in period: %w", err)
		}

		date2Str := strings.TrimSpace(s[toIndex+4:]) // Skip " TO "
		date2 := &Date{Original: "", Calendar: CalendarGregorian}
		if err := parseDateComponents(date2Str, date2); err != nil {
			return nil, fmt.Errorf("invalid end date in period: %w", err)
		}

		date1.EndDate = date2
		return date1, nil
	}

	// Simple FROM or TO
	date := &Date{Original: original, Calendar: CalendarGregorian, Modifier: modifier}
	if err := parseDateComponents(s, date); err != nil {
		return nil, err
	}
	return date, nil
}

// parseDateComponents parses the date components (day, month, year) from a string.
func parseDateComponents(s string, date *Date) error {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return fmt.Errorf("empty date")
	}

	// Try to parse based on number of fields
	switch len(fields) {
	case 1:
		// Year only
		year, err := strconv.Atoi(fields[0])
		if err != nil {
			return fmt.Errorf("invalid year: %s", fields[0])
		}
		date.Year = year

	case 2:
		// Month and year (no day)
		month, err := parseMonth(fields[0])
		if err != nil {
			return err
		}
		year, err := strconv.Atoi(fields[1])
		if err != nil {
			return fmt.Errorf("invalid year: %s", fields[1])
		}
		date.Month = month
		date.Year = year

	case 3:
		// Day, month, and year
		day, err := strconv.Atoi(fields[0])
		if err != nil {
			return fmt.Errorf("invalid day: %s", fields[0])
		}
		month, err := parseMonth(fields[1])
		if err != nil {
			return err
		}
		year, err := strconv.Atoi(fields[2])
		if err != nil {
			return fmt.Errorf("invalid year: %s", fields[2])
		}
		date.Day = day
		date.Month = month
		date.Year = year

	default:
		return fmt.Errorf("invalid date format: too many components in '%s'", s)
	}

	return nil
}

// parseMonth parses a three-letter month abbreviation (case-insensitive).
func parseMonth(s string) (int, error) {
	month, ok := monthNames[strings.ToUpper(s)]
	if !ok {
		return 0, fmt.Errorf("invalid month abbreviation: %s", s)
	}
	return month, nil
}

// normalizeWhitespace normalizes multiple spaces/tabs to single spaces.
func normalizeWhitespace(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

// Compare compares two dates and returns -1 if d < other, 0 if d == other, 1 if d > other.
// For partial dates, missing components are treated as the earliest possible value
// (day=1, month=1). For ranges, the start dates are compared.
func (d *Date) Compare(other *Date) int {
	if d == nil && other == nil {
		return 0
	}
	if d == nil {
		return -1
	}
	if other == nil {
		return 1
	}

	// Compare years
	y1, y2 := d.Year, other.Year
	if y1 < y2 {
		return -1
	}
	if y1 > y2 {
		return 1
	}

	// Years are equal, compare months (treat 0 as 1)
	m1, m2 := d.Month, other.Month
	if m1 == 0 {
		m1 = 1
	}
	if m2 == 0 {
		m2 = 1
	}
	if m1 < m2 {
		return -1
	}
	if m1 > m2 {
		return 1
	}

	// Months are equal, compare days (treat 0 as 1)
	d1, d2 := d.Day, other.Day
	if d1 == 0 {
		d1 = 1
	}
	if d2 == 0 {
		d2 = 1
	}
	if d1 < d2 {
		return -1
	}
	if d1 > d2 {
		return 1
	}

	return 0
}

// ToTime converts the date to a time.Time value.
// Returns an error if the date is incomplete (missing day, month, or year)
// or if the calendar is not Gregorian.
func (d *Date) ToTime() (time.Time, error) {
	if d.Calendar != CalendarGregorian {
		return time.Time{}, fmt.Errorf("ToTime only supports Gregorian calendar, got %s", d.Calendar)
	}

	if d.Year == 0 {
		return time.Time{}, fmt.Errorf("incomplete date: year is missing")
	}
	if d.Month == 0 {
		return time.Time{}, fmt.Errorf("incomplete date: month is missing")
	}
	if d.Day == 0 {
		return time.Time{}, fmt.Errorf("incomplete date: day is missing")
	}

	return time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.UTC), nil
}

// String returns the original GEDCOM date string.
func (d *Date) String() string {
	return d.Original
}
