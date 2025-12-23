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

	// IsBC is true for B.C./BCE dates
	IsBC bool

	// DualYear is the second year for dual dating (e.g., 1751 from "1750/51")
	DualYear int

	// Phrase contains the text for date phrases (e.g., "unknown" from "(unknown)")
	Phrase string

	// IsPhrase is true when the date is a phrase, not a parseable date
	IsPhrase bool
}

// monthNames maps three-letter month abbreviations to month numbers.
// Case-insensitive matching is handled by converting input to uppercase.
var monthNames = map[string]int{
	"JAN": 1, "FEB": 2, "MAR": 3, "APR": 4,
	"MAY": 5, "JUN": 6, "JUL": 7, "AUG": 8,
	"SEP": 9, "OCT": 10, "NOV": 11, "DEC": 12,
}

// hebrewMonthNames maps Hebrew month codes to month numbers.
// The Hebrew calendar has 13 months (Adar II only in leap years).
var hebrewMonthNames = map[string]int{
	"TSH": 1,  // Tishrei
	"CSH": 2,  // Cheshvan
	"KSL": 3,  // Kislev
	"TVT": 4,  // Tevet
	"SHV": 5,  // Shevat
	"ADR": 6,  // Adar
	"ADS": 7,  // Adar II (leap years only)
	"NSN": 8,  // Nisan
	"IYR": 9,  // Iyar
	"SVN": 10, // Sivan
	"TMZ": 11, // Tammuz
	"AAV": 12, // Av
	"ELL": 13, // Elul
}

// frenchMonthNames maps French Republican month codes to month numbers.
// The French Republican calendar has 12 months of 30 days plus complementary days.
//
//nolint:misspell // THER is the GEDCOM code for Thermidor, not a misspelling of "there"
var frenchMonthNames = map[string]int{
	"VEND": 1,  // Vendémiaire
	"BRUM": 2,  // Brumaire
	"FRIM": 3,  // Frimaire
	"NIVO": 4,  // Nivôse
	"PLUV": 5,  // Pluviôse
	"VENT": 6,  // Ventôse
	"GERM": 7,  // Germinal
	"FLOR": 8,  // Floréal
	"PRAI": 9,  // Prairial
	"MESS": 10, // Messidor
	"THER": 11, // Thermidor
	"FRUC": 12, // Fructidor
	"COMP": 13, // Complementary days (Sans-culottides)
}

// ParseDate parses a GEDCOM date string into a Date struct.
// Supports exact dates, partial dates, modifiers, ranges, periods, dual dating,
// B.C. dates, and date phrases.
//
// Examples:
//   - "25 DEC 2020" -> exact date
//   - "JAN 1900" -> partial date (month and year)
//   - "1850" -> partial date (year only)
//   - "ABT 1850" -> approximate date
//   - "BET 1850 AND 1860" -> date range
//   - "FROM 1880 TO 1920" -> date period
//   - "21 FEB 1750/51" -> dual dating
//   - "44 BC" -> B.C. date
//   - "(unknown)" -> date phrase
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

	// Check for date phrase first (starts with '(')
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		date.IsPhrase = true
		date.Phrase = s[1 : len(s)-1] // Remove parentheses
		return date, nil
	}

	// Check for calendar escape sequence
	calendar, rest, found := parseCalendarEscape(s)
	if found {
		date.Calendar = calendar
		s = rest
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

	// Parse the date components (day, month, year, BC, dual year)
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

	// Parse the second date (inherits calendar from first date)
	date2Str := strings.TrimSpace(s[andIndex+5:]) // Skip " AND "
	date2 := &Date{Original: "", Calendar: date1.Calendar}
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

		// Parse the second date (inherits calendar from first date)
		date2Str := strings.TrimSpace(s[toIndex+4:]) // Skip " TO "
		date2 := &Date{Original: "", Calendar: date1.Calendar}
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

// isBCSuffix checks if a string is a B.C./BCE suffix.
func isBCSuffix(s string) bool {
	upper := strings.ToUpper(s)
	return upper == "BC" || upper == "B.C." || upper == "BCE" || upper == "B.C.E."
}

// parseDateComponents parses the date components (day, month, year, BC, dual year) from a string.
func parseDateComponents(s string, date *Date) error {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return fmt.Errorf("empty date")
	}

	// Check for B.C./BCE suffix on last field
	if isBCSuffix(fields[len(fields)-1]) {
		date.IsBC = true
		fields = fields[:len(fields)-1] // Remove BC suffix
		if len(fields) == 0 {
			return fmt.Errorf("empty date after BC suffix")
		}
	}

	return parseDateFields(fields, date)
}

// parseDateFields parses date fields after BC suffix has been handled.
func parseDateFields(fields []string, date *Date) error {
	switch len(fields) {
	case 1:
		return parseYearOnly(fields[0], date)
	case 2:
		return parseMonthYear(fields, date)
	case 3:
		return parseDayMonthYear(fields, date)
	default:
		return fmt.Errorf("invalid date format: too many components in '%s'", strings.Join(fields, " "))
	}
}

// parseYearOnly parses a year-only date (possibly with dual year).
func parseYearOnly(yearStr string, date *Date) error {
	year, dualYear, err := parseYearWithDual(yearStr)
	if err != nil {
		return err
	}
	date.Year = year
	date.DualYear = dualYear
	return nil
}

// parseMonthYear parses a month-year date (no day).
func parseMonthYear(fields []string, date *Date) error {
	month, err := parseMonthForCalendar(fields[0], date.Calendar)
	if err != nil {
		return err
	}
	year, dualYear, err := parseYearWithDual(fields[1])
	if err != nil {
		return err
	}
	date.Month = month
	date.Year = year
	date.DualYear = dualYear
	return nil
}

// parseDayMonthYear parses a full day-month-year date.
func parseDayMonthYear(fields []string, date *Date) error {
	day, err := strconv.Atoi(fields[0])
	if err != nil {
		return fmt.Errorf("invalid day: %s", fields[0])
	}
	month, err := parseMonthForCalendar(fields[1], date.Calendar)
	if err != nil {
		return err
	}
	year, dualYear, err := parseYearWithDual(fields[2])
	if err != nil {
		return err
	}
	date.Day = day
	date.Month = month
	date.Year = year
	date.DualYear = dualYear
	return nil
}

// parseMonthForCalendar parses a month code for a specific calendar system.
// Returns the month number (1-13 depending on calendar) or an error if invalid.
// Month codes are case-insensitive.
func parseMonthForCalendar(s string, calendar Calendar) (int, error) {
	upperMonth := strings.ToUpper(s)

	var monthMap map[string]int
	switch calendar {
	case CalendarGregorian, CalendarJulian:
		monthMap = monthNames
	case CalendarHebrew:
		monthMap = hebrewMonthNames
	case CalendarFrenchRepublican:
		monthMap = frenchMonthNames
	default:
		return 0, fmt.Errorf("unsupported calendar type: %s", calendar)
	}

	month, ok := monthMap[upperMonth]
	if !ok {
		return 0, fmt.Errorf("invalid month code '%s' for %s calendar", s, calendar)
	}
	return month, nil
}

// parseYearWithDual parses a year field that may contain dual dating (e.g., "1750/51" or "1750/1751").
// Returns the primary year and dual year (0 if no dual year).
func parseYearWithDual(s string) (primaryYear, dualYear int, err error) {
	// Check for dual year format (year/year)
	if strings.Contains(s, "/") {
		parts := strings.Split(s, "/")
		if len(parts) != 2 {
			return 0, 0, fmt.Errorf("invalid dual year format: %s", s)
		}

		primaryYear, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, fmt.Errorf("invalid primary year in dual date: %s", parts[0])
		}

		// Parse secondary year - may be 2-digit or 4-digit
		secondaryStr := parts[1]
		secondaryYear, err := strconv.Atoi(secondaryStr)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid secondary year in dual date: %s", secondaryStr)
		}

		// If secondary year is 2 digits, infer century from primary year
		if len(secondaryStr) <= 2 {
			century := (primaryYear / 100) * 100
			secondaryYear = century + secondaryYear
		}

		return primaryYear, secondaryYear, nil
	}

	// No dual year
	year, err := strconv.Atoi(s)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid year: %s", s)
	}
	return year, 0, nil
}

// normalizeWhitespace normalizes multiple spaces/tabs to single spaces.
func normalizeWhitespace(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

// Validate checks if the date is semantically valid (e.g., no day overflow like Feb 30).
// Returns nil for partial dates (day, month, or year is 0) or non-Gregorian calendars.
// Uses stdlib time.Date() to detect invalid dates via normalization.
func (d *Date) Validate() error {
	// Skip validation for partial dates
	if d.Day == 0 || d.Month == 0 || d.Year == 0 {
		return nil
	}

	// Only validate Gregorian calendar for now
	if d.Calendar != CalendarGregorian {
		return nil
	}

	// Use time.Date to check if the date normalizes (indicating overflow)
	t := time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.UTC)

	// If the date normalized to a different day or month, it's invalid
	if t.Day() != d.Day || int(t.Month()) != d.Month {
		// Build informative error message
		monthName := getMonthName(d.Month)
		daysInMonth := getDaysInMonth(d.Month, d.Year)

		if d.Day > daysInMonth {
			return fmt.Errorf("invalid date: %s has %d days in %d, got day %d", monthName, daysInMonth, d.Year, d.Day)
		}
		return fmt.Errorf("invalid date: %d %s %d", d.Day, monthName, d.Year)
	}

	return nil
}

// getMonthName returns the full month name for a month number (1-12).
func getMonthName(month int) string {
	monthNames := []string{
		"", "January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December",
	}
	if month < 1 || month > 12 {
		return "Unknown"
	}
	return monthNames[month]
}

// getDaysInMonth returns the number of days in a month for a given year.
func getDaysInMonth(month, year int) int {
	// Use time.Date with day 0 of next month to get last day of current month
	t := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC)
	return t.Day()
}

// compareInts returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareInts(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// defaultToOne returns 1 if val is 0, otherwise returns val.
func defaultToOne(val int) int {
	if val == 0 {
		return 1
	}
	return val
}

// Compare compares two dates and returns -1 if d < other, 0 if d == other, 1 if d > other.
// For partial dates, missing components are treated as the earliest possible value
// (day=1, month=1). B.C. dates sort before all A.D. dates, and among B.C. dates,
// higher year numbers are earlier (100 BC < 200 BC).
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

	// Handle BC vs AD comparison
	if d.IsBC != other.IsBC {
		if d.IsBC {
			return -1 // BC dates come before AD dates
		}
		return 1 // AD dates come after BC dates
	}

	// Compare years (reversed for BC dates: 100 BC > 200 BC)
	y1, y2 := d.Year, other.Year
	if cmp := compareInts(y1, y2); cmp != 0 {
		if d.IsBC {
			return -cmp // Reverse for BC
		}
		return cmp
	}

	// Years are equal, compare months (treat 0 as 1)
	m1 := defaultToOne(d.Month)
	m2 := defaultToOne(other.Month)
	if cmp := compareInts(m1, m2); cmp != 0 {
		return cmp
	}

	// Months are equal, compare days (treat 0 as 1)
	d1 := defaultToOne(d.Day)
	d2 := defaultToOne(other.Day)
	return compareInts(d1, d2)
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
