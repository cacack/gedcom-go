package gedcom

import (
	"fmt"
	"math"
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
	// ModifierInterpreted indicates an interpreted date where the user has clarified
	// an ambiguous original phrase (INT). The original phrase is stored in InterpretedFrom.
	ModifierInterpreted
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
	case ModifierInterpreted:
		return "INT"
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

	// DualYear is the second year for dual dating (e.g., 1751 from "1750/51"),
	// or 0 if none. A one- or two-digit suffix gives the last digits of the year
	// that follows Year: "1699/00" is 1700, and on a B.C. date, where the following
	// year is the smaller number, "100/99 BC" is 99. A longer suffix is taken as
	// written. ParseDate accepts a suffix that is not the following year (e.g.
	// "1750/53" gives 1753, "1750/50" gives 1850, "1750/1749" gives 1749) so the
	// value is preserved; Validate reports it. A B.C. dual year of 0 ("1/0 BC")
	// cannot be represented and reads as no dual year.
	DualYear int

	// Phrase contains the text for date phrases (e.g., "unknown" from "(unknown)")
	Phrase string

	// IsPhrase is true when the date is a phrase, not a parseable date
	IsPhrase bool

	// IsInterpreted is true when the date uses the INT modifier, indicating
	// the user has interpreted an ambiguous original phrase
	IsInterpreted bool

	// InterpretedFrom contains the original phrase that was interpreted,
	// without the surrounding parentheses (e.g., "about eighteen fifty" from
	// "INT 1850 (about eighteen fifty)")
	InterpretedFrom string
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
//   - "@#DJULIAN@ 1 JAN 1750" or "JULIAN 1 JAN 1750" -> Julian calendar (5.5.x escape or 7.0 keyword)
//   - "(unknown)" -> date phrase
//
// Both calendar forms are accepted regardless of the document's GEDCOM version;
// ParseDate does not check version conformance.
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

	// Check for a leading calendar escape sequence. The grammar puts the escape
	// inside each <date> production, so this only seeds the default calendar for
	// the components below; parseDateComponents strips a per-component escape of
	// its own and overrides the seed.
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
			return parseDateRange(s, original, date.Calendar)
		case ModifierFrom, ModifierTo, ModifierFromTo:
			// FROM date, TO date, or FROM date TO date
			return parseDatePeriod(s, original, modifier, date.Calendar)
		case ModifierInterpreted:
			// INT date (original phrase)
			return parseInterpretedDate(s, original, date.Calendar)
		}
	}

	// Parse the date components (day, month, year, BC, dual year)
	if err := parseDateComponents(s, date); err != nil {
		return nil, err
	}

	return date, nil
}

// gedcom7Calendars maps the GEDCOM 7 calendar keywords to calendars. The 7.0
// grammar spells them case-sensitively, so lookups use the field as written.
// Extension calendars (_XXX) are not recognized.
var gedcom7Calendars = map[string]Calendar{
	"GREGORIAN": CalendarGregorian,
	"JULIAN":    CalendarJulian,
	"HEBREW":    CalendarHebrew,
	"FRENCH_R":  CalendarFrenchRepublican,
}

// parseCalendarEscape parses a leading calendar marker and returns the calendar
// type and the remaining string. It accepts both the GEDCOM 5.5.x escape form
// (e.g. @#DJULIAN@) and the GEDCOM 7 keyword form (e.g. JULIAN).
func parseCalendarEscape(s string) (Calendar, string, bool) {
	if first, rest, _ := strings.Cut(s, " "); first != "" {
		if calendar, ok := gedcom7Calendars[first]; ok {
			return calendar, strings.TrimSpace(rest), true
		}
	}

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
	case "INT":
		modifier = ModifierInterpreted
		found = true
	default:
		return ModifierNone, s, false
	}

	// Return the rest of the string after the modifier
	rest := strings.TrimSpace(strings.TrimPrefix(s, fields[0]))
	return modifier, rest, found
}

// parseDateRange parses a date range in the format "date1 AND date2".
// defaultCal is the calendar each date starts from; either date may override it
// with a calendar escape of its own.
func parseDateRange(s, original string, defaultCal Calendar) (*Date, error) {
	// Find the AND keyword (case-insensitive, byte-safe)
	andIndex := indexFoldASCII(s, " AND ")
	if andIndex == -1 {
		return nil, fmt.Errorf("invalid date range: missing AND keyword in '%s'", original)
	}

	// Parse the first date
	date1Str := strings.TrimSpace(s[:andIndex])
	date1 := &Date{Original: original, Calendar: defaultCal, Modifier: ModifierBetween}
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
// defaultCal is the calendar each date starts from; either date may override it
// with a calendar escape of its own.
func parseDatePeriod(s, original string, modifier DateModifier, defaultCal Calendar) (*Date, error) {
	// Check if there's a TO keyword for FROM...TO format (case-insensitive, byte-safe)
	toIndex := indexFoldASCII(s, " TO ")

	if modifier == ModifierFrom && toIndex != -1 {
		// FROM date1 TO date2
		date1Str := strings.TrimSpace(s[:toIndex])
		date1 := &Date{Original: original, Calendar: defaultCal, Modifier: ModifierFromTo}
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
	date := &Date{Original: original, Calendar: defaultCal, Modifier: modifier}
	if err := parseDateComponents(s, date); err != nil {
		return nil, err
	}
	return date, nil
}

// parseInterpretedDate parses an interpreted date in the format "date (original phrase)".
// The date portion is parsed normally, and the phrase is stored in InterpretedFrom.
// Examples:
//   - "1850 (about eighteen fifty)" -> Year=1850, InterpretedFrom="about eighteen fifty"
//   - "25 DEC 1850 (Christmas day)" -> Day=25, Month=12, Year=1850, InterpretedFrom="Christmas day"
//   - "1850" -> Year=1850, InterpretedFrom="" (no phrase is valid)
//
// defaultCal is the calendar the date starts from; the date may override it with
// a calendar escape of its own.
func parseInterpretedDate(s, original string, defaultCal Calendar) (*Date, error) {
	date := &Date{
		Original:      original,
		Calendar:      defaultCal,
		Modifier:      ModifierInterpreted,
		IsInterpreted: true,
	}

	// Find the opening parenthesis to separate date from phrase
	parenIndex := strings.Index(s, "(")
	if parenIndex == -1 {
		// No parenthetical phrase - just parse the date
		if err := parseDateComponents(s, date); err != nil {
			return nil, err
		}
		return date, nil
	}

	// Extract the date portion (before the opening paren)
	dateStr := strings.TrimSpace(s[:parenIndex])
	if dateStr == "" {
		return nil, fmt.Errorf("empty date in interpreted date: '%s'", original)
	}

	// Parse the date portion
	if err := parseDateComponents(dateStr, date); err != nil {
		return nil, err
	}

	// Extract the phrase between parentheses
	// Find the last closing paren to handle nested parens
	phraseStart := parenIndex + 1
	lastParen := strings.LastIndex(s, ")")
	if lastParen == -1 {
		// No closing paren - treat rest of string as phrase
		date.InterpretedFrom = s[phraseStart:]
	} else {
		date.InterpretedFrom = s[phraseStart:lastParen]
	}

	return date, nil
}

// isBCSuffix checks if a string is a B.C./BCE suffix.
func isBCSuffix(s string) bool {
	upper := strings.ToUpper(s)
	return upper == "BC" || upper == "B.C." || upper == "BCE" || upper == "B.C.E."
}

// parseDateComponents parses the date components (day, month, year, BC, dual year) from a string.
// A calendar escape belongs to the <date> production, so one may appear here even after a
// modifier keyword has been consumed (e.g. "ABT @#DJULIAN@ MAR 1066"); it overrides the
// calendar the caller seeded on date. The strip must precede tokenization because
// "@#DFRENCH R@" contains a space that strings.Fields would split.
func parseDateComponents(s string, date *Date) error {
	if calendar, rest, found := parseCalendarEscape(strings.TrimSpace(s)); found {
		date.Calendar = calendar
		s = rest
	}

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
	year, dualYear, err := parseYearWithDual(yearStr, date.IsBC)
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
	year, dualYear, err := parseYearWithDual(fields[1], date.IsBC)
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
	year, dualYear, err := parseYearWithDual(fields[2], date.IsBC)
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
// Returns the primary year and dual year (0 if no dual year). isBC selects the
// direction in which a short suffix resolves; see Date.DualYear.
func parseYearWithDual(s string, isBC bool) (primaryYear, dualYear int, err error) {
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

		switch len(secondaryStr) {
		case 1:
			secondaryYear = followingYearEndingIn(primaryYear, secondaryYear, 10, isBC)
		case 2:
			secondaryYear = followingYearEndingIn(primaryYear, secondaryYear, 100, isBC)
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

// followingYearEndingIn returns the nearest year after primary, in the
// direction years run (downward for B.C.), that is congruent to suffix modulo
// modulus (10 for a one-digit suffix, 100 for two). For example
// (1699, 0, 100) is 1700 and (100, 99, 100, B.C.) is 99.
// The result is not guaranteed to be the year following primary: "1750/53"
// gives 1753, and a B.C. suffix with no earlier year above 0, such as 45/46 BC
// or 100/00 BC, keeps primary's leading digits (46, 100). Validate reports
// such a dual year; the value is kept so nothing written is lost.
func followingYearEndingIn(primary, suffix, modulus int, isBC bool) int {
	year := primary - primary%modulus + suffix
	if !isBC && year <= primary {
		year += modulus
	}
	if isBC && year >= primary && year-modulus > 0 {
		year -= modulus
	}
	return year
}

// normalizeWhitespace normalizes multiple spaces/tabs to single spaces.
func normalizeWhitespace(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

// indexFoldASCII returns the byte index of the first case-insensitive occurrence
// of the ASCII substr in s, or -1 if not found.
// Unlike strings.Index(strings.ToUpper(s), substr), this is safe for strings
// containing non-ASCII or invalid UTF-8 bytes because it does not change the
// byte length of s.
func indexFoldASCII(s, substr string) int {
	n := len(substr)
	if n == 0 {
		return 0
	}
	if n > len(s) {
		return -1
	}
	for i := 0; i <= len(s)-n; i++ {
		if strings.EqualFold(s[i:i+n], substr) {
			return i
		}
	}
	return -1
}

// Validate checks if the date is semantically valid (e.g., no day overflow like Feb 30).
// For ranges (BET...AND) and periods (FROM...TO), both the start date and EndDate
// are checked independently; the first error is returned, and an error in the end
// date is prefixed with "invalid end date". A partial or non-Gregorian endpoint is
// skipped without skipping the other one.
// Each endpoint gets two checks, in order: its dual year, then its day.
// The dual-year check runs on every date, partial or not, on any calendar: a
// DualYear that is not the year following Year (e.g. "1750/53") is an error.
// The day check is skipped for partial dates (day, month, or year is 0) and
// non-Gregorian calendars; the validator package relies on this, clearing Day
// to check only a dual year, so keep partial dates skipped.
// Gregorian dates are checked on the proleptic Gregorian calendar using astronomical
// years (1 BCE = year 0, a leap year), matching the calendar conversions; error
// messages report the era year as written.
// Uses stdlib time.Date() to detect invalid dates via normalization.
func (d *Date) Validate() error {
	if err := d.validateEndpoint(); err != nil {
		return err
	}
	if d.EndDate != nil {
		if err := d.EndDate.validateEndpoint(); err != nil {
			return fmt.Errorf("invalid end date: %w", err)
		}
	}
	return nil
}

// validateEndpoint checks a single date (ignoring EndDate): its dual year, then its day.
func (d *Date) validateEndpoint() error {
	if err := d.validateDualYear(); err != nil {
		return err
	}
	return d.validateDay()
}

// validateDualYear checks that a DualYear is the year following Year,
// which on a B.C. date is the smaller number.
func (d *Date) validateDualYear() error {
	if d.DualYear == 0 {
		return nil
	}
	following := d.Year + 1
	if d.IsBC {
		following = d.Year - 1
	}
	if d.DualYear != following {
		return fmt.Errorf("invalid date: dual year %d does not follow %d (a slash year is read as Old Style dual dating)", d.DualYear, d.Year)
	}
	return nil
}

// validateDay checks a single date (ignoring EndDate) for day overflow.
// ToTime also relies on it to reject dates that time.Date would roll over
// (e.g. 31 FEB 1900 becoming 3 MAR 1900), so loosening this check loosens ToTime.
// ToTime deliberately does not check the dual year: a time.Time has no use for it.
func (d *Date) validateDay() error {
	// Skip validation for partial dates
	if d.Day == 0 || d.Month == 0 || d.Year == 0 {
		return nil
	}

	// Only validate Gregorian calendar for now
	if d.Calendar != CalendarGregorian {
		return nil
	}

	// Use time.Date to check if the date normalizes (indicating overflow)
	year := AstronomicalYear(d.Year, d.IsBC)
	t := time.Date(year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.UTC)

	// If the date normalized to a different day or month, it's invalid
	if t.Day() != d.Day || int(t.Month()) != d.Month {
		// Build informative error message using the era year as written
		monthName := getMonthName(d.Month)
		daysInMonth := getDaysInMonth(d.Month, year)
		eraYear := strconv.Itoa(d.Year)
		if d.IsBC {
			eraYear += " BCE"
		}

		if d.Day > daysInMonth {
			return fmt.Errorf("invalid date: %s has %d days in %s, got day %d", monthName, daysInMonth, eraYear, d.Day)
		}
		return fmt.Errorf("invalid date: %d %s %s", d.Day, monthName, eraYear)
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

// getDaysInMonth returns the number of days in a month for a given astronomical year.
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
//
// When comparing dates with different calendars, both dates are converted to JDN
// (Julian Day Number) for accurate cross-calendar comparison.
//
// A dual date is read in New Style, as Bounds reads it: "11 FEB 1731/32"
// compares equal to 11 FEB 1732. A dual date whose New Style year is unknown
// is read in Year, its earliest reading: one that does not say which side of
// 25 March it falls on ("1731/32", "MAR 1731/32"), whose DualYear is not the
// following year, or that is not Gregorian or Julian. So "MAR 1731/32" sorts
// as March 1731 while "1 MAR 1731/32" sorts as March 1732.
//
// Compare ignores Modifier, so it gives a sort order, not proof that one date
// precedes another; see Bounds for that.
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
	d, other = d.inNewStyle(), other.inNewStyle()

	// If calendars differ, convert both to JDN for comparison
	if d.Calendar != other.Calendar {
		// Try to convert both to JDN
		jdn1, err1 := d.toJDN()
		jdn2, err2 := other.toJDN()

		// If both conversions succeed, compare JDN values
		if err1 == nil && err2 == nil {
			return compareInts(jdn1, jdn2)
		}

		// If conversion fails for one or both dates, fall back to same-calendar logic
		// This happens for dates with year=0 (too incomplete)
		// Fall through to the original comparison logic
	}

	// Same calendar (or cross-calendar conversion failed) - use original logic

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

// Bounds returns the earliest and latest day the date can denote, as Julian
// Day Numbers (see GregorianToJDN), so bounds from any calendar compare
// directly. Unlike Compare, which reads a partial date as its first day and
// ignores modifiers, Bounds covers every reading of the date:
//   - A partial date spans its whole month or year: "1503" runs from
//     1 JAN 1503 to 31 DEC 1503.
//   - ABT, CAL, EST and INT dates are bounded by the period they name; Bounds
//     does not guess how far "about" reaches.
//   - BEF and TO have no earliest day and AFT and FROM no latest; the open end
//     is math.MinInt or math.MaxInt. The named date itself is included, so
//     "BEF 1850" ends on 31 DEC 1850.
//   - BET...AND and FROM...TO run from the earlier endpoint's earliest day to
//     the later endpoint's latest.
//   - A dual date is read in New Style: "11 FEB 1731/32" is 11 FEB 1732, since
//     DualYear applies from 1 January to 24 March and Year from 25 March. A
//     dual date that does not say which side of 25 March it falls on ("1731/32",
//     "MAR 1731/32"), whose DualYear is not the year following Year (see
//     Validate), or that is not Gregorian or Julian, spans both readings.
//
// ok is false when the date or a range's end date has no year, the date is a
// phrase, its calendar is unknown, it names a month or day the calendar does
// not have that year (31 FEB 1850, Adar II in a common Hebrew year), or its
// year is beyond 1,000,000. earliest and latest are meaningless when ok is
// false.
//
// The open ends math.MinInt and math.MaxInt are sentinels: compare them, but
// do not add to or subtract from them, which overflows.
//
// One date is provably before another when its latest day is before the
// other's earliest: aLatest < bEarliest holds on every reading of both.
// Compare and IsBefore can disagree with Bounds on partial, modified and dual
// dates; use Bounds when the question is what the dates prove.
func (d *Date) Bounds() (earliest, latest int, ok bool) {
	if d == nil || d.IsPhrase {
		return 0, 0, false
	}
	earliest, latest, ok = d.endpointBounds()
	if !ok {
		return 0, 0, false
	}
	switch d.Modifier {
	case ModifierBefore, ModifierTo:
		return math.MinInt, latest, true
	case ModifierAfter, ModifierFrom:
		return earliest, math.MaxInt, true
	case ModifierBetween, ModifierFromTo:
		if d.EndDate == nil {
			return 0, 0, false
		}
		endEarliest, endLatest, ok := d.EndDate.endpointBounds()
		if !ok {
			return 0, 0, false
		}
		return min(earliest, endEarliest), max(latest, endLatest), true
	}
	return earliest, latest, true
}

// endpointBounds bounds a single date, ignoring Modifier and EndDate. A dual
// date whose New Style year is unknown spans both of its years.
func (d *Date) endpointBounds() (earliest, latest int, ok bool) {
	if d.DualYear == 0 {
		return d.boundsInYear(d.Year)
	}
	if year, known := d.newStyleYear(); known {
		return d.boundsInYear(year)
	}
	earliest, latest, ok = d.boundsInYear(d.Year)
	dualEarliest, dualLatest, dualOK := d.boundsInYear(d.DualYear)
	return min(earliest, dualEarliest), max(latest, dualLatest), ok && dualOK
}

// newStyleYear returns the year a dual date falls in when the year starts on
// 1 January: DualYear from 1 January to 24 March, Year from 25 March. known is
// false when the date does not say which side of 25 March it falls on, its
// DualYear is not the following year, or it is not Gregorian or Julian.
func (d *Date) newStyleYear() (year int, known bool) {
	if d.validateDualYear() != nil || (d.Calendar != CalendarGregorian && d.Calendar != CalendarJulian) {
		return 0, false
	}
	switch {
	case d.Month == 1 || d.Month == 2 || (d.Month == 3 && d.Day >= 1 && d.Day <= 24):
		return d.DualYear, true
	case d.Month > 3 || (d.Month == 3 && d.Day >= 25):
		return d.Year, true
	}
	return 0, false
}

// inNewStyle returns d with Year set to its New Style year and DualYear
// cleared, when d is a dual date whose New Style year is known (see
// newStyleYear); otherwise it returns d itself. Compare and YearsBetween read
// dates through it, so they order dual dates as Bounds does.
func (d *Date) inNewStyle() *Date {
	if d.DualYear == 0 {
		return d
	}
	year, known := d.newStyleYear()
	if !known {
		return d
	}
	result := *d
	result.Year, result.DualYear = year, 0
	return &result
}

// maxBoundsYear is the largest year, before or after the epoch, that Bounds
// places. Larger years are not genealogical dates, and keeping them out keeps
// the day arithmetic from overflowing.
const maxBoundsYear = 1_000_000

// boundsInYear bounds the date's day and month, or its whole month or year
// when those are 0, within year on the date's calendar. ok is false for a
// month or day the calendar does not have that year, such as 31 FEB or Adar II
// in a common Hebrew year.
func (d *Date) boundsInYear(year int) (earliest, latest int, ok bool) {
	if year == 0 || year < -maxBoundsYear || year > maxBoundsYear || d.Month < 0 {
		return 0, 0, false
	}
	lastMonth := 13 // Hebrew and French Republican
	if d.Calendar == CalendarGregorian || d.Calendar == CalendarJulian {
		year = AstronomicalYear(year, d.IsBC)
		lastMonth = 12
	}

	month, nextYear, nextMonth := d.Month, year, d.Month+1
	if d.Month == 0 {
		month, nextYear, nextMonth = 1, year+1, 1
	} else if nextMonth > lastMonth {
		nextYear, nextMonth = year+1, 1
	}
	start, err := calendarToJDN(d.Calendar, year, month, 1)
	if err != nil {
		return 0, 0, false
	}
	next, _ := calendarToJDN(d.Calendar, nextYear, nextMonth, 1)
	if next <= start {
		return 0, 0, false // a month the year does not have
	}
	if d.Month == 0 || d.Day == 0 {
		return start, next - 1, true
	}
	day := start + d.Day - 1
	if d.Day < 1 || day >= next {
		return 0, 0, false // a day the month does not have
	}
	return day, day, true
}

// ToTime converts the date to a time.Time value.
// Returns an error if the date is incomplete (missing day, month, or year),
// if the calendar is not Gregorian, or if the day does not exist in that
// month and year (e.g. 31 FEB 1900, 29 FEB 1900), using the same rule as
// Validate; invalid dates are never rolled over into the following month.
// B.C. dates use astronomical year numbering via AstronomicalYear, so the
// result's Year() is 0 for 1 BCE and -43 for 44 BCE (proleptic Gregorian).
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
	if err := d.validateDay(); err != nil {
		return time.Time{}, err
	}

	return time.Date(AstronomicalYear(d.Year, d.IsBC), time.Month(d.Month), d.Day, 0, 0, 0, 0, time.UTC), nil
}

// String returns the original GEDCOM date string.
func (d *Date) String() string {
	return d.Original
}

// toJDN converts a Date to Julian Day Number.
// Returns error if date is too incomplete (year=0).
// For partial dates, uses day=1 and month=1 as defaults.
func (d *Date) toJDN() (int, error) {
	if d.Year == 0 {
		return 0, fmt.Errorf("cannot convert date to JDN: year is missing")
	}

	// Use defaults for partial dates
	day := d.Day
	if day == 0 {
		day = 1
	}
	month := d.Month
	if month == 0 {
		month = 1
	}

	// Gregorian and Julian dates count B.C. years astronomically
	year := d.Year
	if d.Calendar == CalendarGregorian || d.Calendar == CalendarJulian {
		year = AstronomicalYear(d.Year, d.IsBC)
	}
	return calendarToJDN(d.Calendar, year, month, day)
}

// calendarToJDN converts a day on cal to a Julian Day Number. year is
// astronomical for the Gregorian and Julian calendars (see AstronomicalYear)
// and as written for the others. Hebrew months use GEDCOM numbering.
func calendarToJDN(cal Calendar, year, month, day int) (int, error) {
	switch cal {
	case CalendarGregorian:
		return GregorianToJDN(year, month, day), nil
	case CalendarJulian:
		return JulianToJDN(year, month, day), nil
	case CalendarHebrew:
		return HebrewToJDN(year, month, day), nil
	case CalendarFrenchRepublican:
		return FrenchToJDN(year, month, day), nil
	default:
		return 0, fmt.Errorf("unsupported calendar: %s", cal)
	}
}

// ToGregorian converts the date to the Gregorian calendar.
// Returns a new Date with Calendar set to CalendarGregorian.
//
// For dates already in Gregorian calendar, returns a copy of self.
// For partial dates (missing day or month), converts available components.
// Returns error if year is 0 (date too incomplete to convert).
//
// Ranges and periods are converted as a unit: the EndDate is converted too,
// and a failure there fails the whole conversion rather than returning a
// range that silently lost its endpoint.
//
// All non-calendar fields (Original, Modifier, DualYear, Phrase, IsPhrase,
// IsInterpreted, InterpretedFrom) are preserved from the source date.
// DualYear is copied as written, not converted, so when the conversion
// changes Year it may no longer follow it and Validate on the result
// reports it.
func (d *Date) ToGregorian() (*Date, error) {
	result, err := d.toGregorianComponents()
	if err != nil {
		return nil, err
	}

	// A range or period is one value: its endpoint may carry its own calendar,
	// so it needs converting even when the start date is already Gregorian.
	if d.EndDate != nil {
		end, err := d.EndDate.ToGregorian()
		if err != nil {
			return nil, fmt.Errorf("cannot convert end date to Gregorian: %w", err)
		}
		result.EndDate = end
	}

	return result, nil
}

// toGregorianComponents converts the date's own calendar components, ignoring
// EndDate. ToGregorian handles the endpoint.
func (d *Date) toGregorianComponents() (*Date, error) {
	// If already Gregorian, return a copy
	if d.Calendar == CalendarGregorian {
		result := *d
		return &result, nil
	}

	// Need at least a year to convert
	if d.Year == 0 {
		return nil, fmt.Errorf("cannot convert date to Gregorian: year is missing")
	}

	// Convert to JDN first
	jdn, err := d.toJDN()
	if err != nil {
		return nil, err
	}

	// Convert JDN to Gregorian
	astroYear, month, day := JDNToGregorian(jdn)

	// Convert astronomical year back to GEDCOM format
	year, isBC := FromAstronomicalYear(astroYear)

	// Create new date with Gregorian calendar. Only the calendar-dependent
	// fields are recomputed; everything else carries over so the conversion
	// stays lossless.
	result := &Date{
		Original:        d.Original, // Preserve original for lossless representation
		Year:            year,
		Calendar:        CalendarGregorian,
		IsBC:            isBC,
		Modifier:        d.Modifier,
		DualYear:        d.DualYear,
		Phrase:          d.Phrase,
		IsPhrase:        d.IsPhrase,
		IsInterpreted:   d.IsInterpreted,
		InterpretedFrom: d.InterpretedFrom,
	}

	// For partial dates, preserve the precision level
	switch {
	case d.Month == 0:
		// Year-only: don't set month or day
		result.Month = 0
		result.Day = 0
	case d.Day == 0:
		// Month+Year: set month but not day
		result.Month = month
		result.Day = 0
	default:
		// Complete date: set both month and day
		result.Month = month
		result.Day = day
	}

	return result, nil
}

// IsAfter returns true if d is after other.
// Returns false if either date is nil.
func (d *Date) IsAfter(other *Date) bool {
	if d == nil || other == nil {
		return false
	}
	return d.Compare(other) > 0
}

// IsBefore returns true if d is before other by Compare.
// Returns false if either date is nil. It reads a partial date as its first
// day and ignores modifiers; see Bounds to test whether d is provably before
// other.
func (d *Date) IsBefore(other *Date) bool {
	if d == nil || other == nil {
		return false
	}
	return d.Compare(other) < 0
}

// IsEqual returns true if d is equal to other.
// Returns true if both dates are nil, false if only one is nil.
func (d *Date) IsEqual(other *Date) bool {
	if d == nil && other == nil {
		return true
	}
	if d == nil || other == nil {
		return false
	}
	return d.Compare(other) == 0
}

// YearsBetween calculates the number of years between two dates.
// For complete Gregorian dates, uses exact calculation via time.Time.
// For partial dates, non-Gregorian calendars, or dates ToTime rejects (such as
// a nonexistent day like 31 FEB), uses simple year subtraction.
// Both paths count BCE years astronomically (see AstronomicalYear), so there is
// no year zero between 1 BCE and 1 CE.
// A dual date is first read in New Style, as Compare reads it, so
// "11 FEB 1731/32" counts from 1732; one whose New Style year is unknown
// counts from Year.
// Returns error if both dates don't have years (insufficient data).
// Always returns absolute value (no negative years).
func YearsBetween(d1, d2 *Date) (years int, exact bool, err error) {
	// Check both dates exist and have years
	if d1 == nil || d2 == nil || d1.Year == 0 || d2.Year == 0 {
		return 0, false, fmt.Errorf("insufficient date information")
	}
	d1, d2 = d1.inNewStyle(), d2.inNewStyle()

	// Try exact calculation for complete Gregorian dates
	t1, err1 := d1.ToTime()
	t2, err2 := d2.ToTime()
	if err1 == nil && err2 == nil {
		// Use time.Time for exact calculation
		// Calculate the difference in years, accounting for birthdays not yet reached
		var earlier, later time.Time
		if t1.Before(t2) {
			earlier = t1
			later = t2
		} else {
			earlier = t2
			later = t1
		}

		years = later.Year() - earlier.Year()
		// Adjust if birthday hasn't occurred yet this year
		if later.Month() < earlier.Month() ||
			(later.Month() == earlier.Month() && later.Day() < earlier.Day()) {
			years--
		}
		return years, true, nil
	}

	// Fall back to year subtraction for partial dates or non-Gregorian calendars
	yearDiff := AstronomicalYear(d1.Year, d1.IsBC) - AstronomicalYear(d2.Year, d2.IsBC)
	if yearDiff < 0 {
		yearDiff = -yearDiff
	}
	return yearDiff, false, nil
}
