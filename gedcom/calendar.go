package gedcom

// Calendar conversion functions using Julian Day Number (JDN) as the universal
// intermediate format. Implements standard algorithms from "Calendrical Calculations"
// by Dershowitz & Reingold.
//
// JDN is a continuous count of days since the beginning of the Julian Period
// (January 1, 4713 BC in the proleptic Julian calendar). It provides a
// calendar-independent way to perform date arithmetic and conversions.

// GregorianToJDN converts a Gregorian date to Julian Day Number.
// Uses astronomical year numbering (year 0 exists, negative years for BC).
//
// Reference: Dershowitz & Reingold "Calendrical Calculations"
//
// Parameters:
//   - year: astronomical year (0 = 1 BC, -1 = 2 BC, etc.)
//   - month: 1-12 (January = 1, December = 12)
//   - day: 1-31 depending on month
//
// Returns:
//   - jdn: Julian Day Number
//
// Example:
//
//	GregorianToJDN(2000, 1, 1) = 2451545  // January 1, 2000
//	GregorianToJDN(1970, 1, 1) = 2440588  // Unix epoch
func GregorianToJDN(year, month, day int) int {
	// Standard formula from Dershowitz & Reingold
	a := (14 - month) / 12
	y := year + 4800 - a
	m := month + 12*a - 3

	jdn := day + (153*m+2)/5 + 365*y + y/4 - y/100 + y/400 - 32045
	return jdn
}

// JDNToGregorian converts a Julian Day Number to Gregorian date.
// Returns year in astronomical numbering (0 = 1 BC, -1 = 2 BC, etc.)
//
// Reference: Dershowitz & Reingold "Calendrical Calculations"
//
// Parameters:
//   - jdn: Julian Day Number
//
// Returns:
//   - year: astronomical year (0 = 1 BC, -1 = 2 BC, etc.)
//   - month: 1-12 (January = 1, December = 12)
//   - day: 1-31 depending on month
//
// Example:
//
//	JDNToGregorian(2451545) = (2000, 1, 1)  // January 1, 2000
func JDNToGregorian(jdn int) (year, month, day int) {
	// Standard formula from Dershowitz & Reingold
	a := jdn + 32044
	b := (4*a + 3) / 146097
	c := a - (146097*b)/4
	d := (4*c + 3) / 1461
	e := c - (1461*d)/4
	m := (5*e + 2) / 153

	day = e - (153*m+2)/5 + 1
	month = m + 3 - 12*(m/10)
	year = 100*b + d - 4800 + m/10

	return year, month, day
}

// AstronomicalYear converts a GEDCOM year (with IsBC flag) to astronomical year.
// GEDCOM uses historical year numbering (1 BC, 2 BC, etc. with no year 0),
// while astronomical year numbering includes year 0 (0 = 1 BC, -1 = 2 BC, etc.).
//
// Conversion rules:
//   - AD years: unchanged (e.g., 2000 AD = 2000)
//   - BC years: year 0 = 1 BC, so n BC = -(n-1)
//     Examples:
//     1 BC = 0
//     2 BC = -1
//     44 BC = -43
//
// Parameters:
//   - year: GEDCOM year (always positive, e.g., 44 for "44 BC")
//   - isBC: true if the year is BC/BCE
//
// Returns:
//   - astronomical year (may be negative or zero)
//
// Example:
//
//	AstronomicalYear(44, true) = -43   // 44 BC
//	AstronomicalYear(2000, false) = 2000  // 2000 AD
func AstronomicalYear(year int, isBC bool) int {
	if !isBC {
		return year
	}
	// BC years: 1 BC = 0, 2 BC = -1, etc.
	return -(year - 1)
}

// FromAstronomicalYear converts astronomical year back to GEDCOM representation.
// Inverse of AstronomicalYear.
//
// Conversion rules:
//   - Positive years and zero: BC years
//     0 = 1 BC
//     -1 = 2 BC
//     -43 = 44 BC
//   - Positive years: AD years (unchanged)
//
// Parameters:
//   - astroYear: astronomical year (may be negative or zero)
//
// Returns:
//   - year: GEDCOM year (always positive)
//   - isBC: true if the year is BC/BCE
//
// Example:
//
//	FromAstronomicalYear(-43) = (44, true)   // 44 BC
//	FromAstronomicalYear(2000) = (2000, false)  // 2000 AD
func FromAstronomicalYear(astroYear int) (year int, isBC bool) {
	if astroYear > 0 {
		return astroYear, false
	}
	// Astronomical year 0 or negative = BC
	// 0 = 1 BC, -1 = 2 BC, etc.
	return -(astroYear - 1), true
}

// JulianToJDN converts a Julian calendar date to Julian Day Number.
// Uses astronomical year numbering (year 0 exists, negative years for BC).
//
// The Julian calendar was introduced by Julius Caesar in 45 BC and differs from
// the Gregorian calendar only in leap year rules:
//   - Julian: leap year every 4 years (no century exception)
//   - Gregorian: leap year every 4 years, except centuries unless divisible by 400
//
// Most countries switched from Julian to Gregorian in October 1582, though some
// regions continued using the Julian calendar until the 20th century. Historical
// dates before October 15, 1582 are typically expressed in Julian calendar.
//
// Reference: Dershowitz & Reingold "Calendrical Calculations"
//
// Parameters:
//   - year: astronomical year (0 = 1 BC, -1 = 2 BC, etc.)
//   - month: 1-12 (January = 1, December = 12)
//   - day: 1-31 depending on month
//
// Returns:
//   - jdn: Julian Day Number
//
// Example:
//
//	JulianToJDN(1582, 10, 4) = 2299160   // Oct 4, 1582 (last Julian day in most countries)
//	JulianToJDN(-43, 3, 15) = 1705426    // Ides of March, 44 BC
//	JulianToJDN(1, 1, 1) = 1721424       // January 1, 1 AD
func JulianToJDN(year, month, day int) int {
	// Standard formula from Dershowitz & Reingold
	// Same as Gregorian but without century leap year corrections
	a := (14 - month) / 12
	y := year + 4800 - a
	m := month + 12*a - 3

	jdn := day + (153*m+2)/5 + 365*y + y/4 - 32083
	return jdn
}

// JDNToJulian converts a Julian Day Number to Julian calendar date.
// Returns year in astronomical numbering (0 = 1 BC, -1 = 2 BC, etc.)
//
// Reference: Dershowitz & Reingold "Calendrical Calculations"
//
// Parameters:
//   - jdn: Julian Day Number
//
// Returns:
//   - year: astronomical year (0 = 1 BC, -1 = 2 BC, etc.)
//   - month: 1-12 (January = 1, December = 12)
//   - day: 1-31 depending on month
//
// Example:
//
//	JDNToJulian(2299160) = (1582, 10, 4)   // Oct 4, 1582 (last Julian day)
//	JDNToJulian(1705426) = (-43, 3, 15)    // Ides of March, 44 BC
func JDNToJulian(jdn int) (year, month, day int) {
	// Standard formula from Dershowitz & Reingold
	b := 0
	c := jdn + 32082
	d := (4*c + 3) / 1461
	e := c - (1461*d)/4
	m := (5*e + 2) / 153

	day = e - (153*m+2)/5 + 1
	month = m + 3 - 12*(m/10)
	year = 100*b + d - 4800 + m/10

	return year, month, day
}

// IsFrenchLeapYear returns true if the French Republican year is a leap year.
// Uses the "continuous" method: leap if following Gregorian year is divisible by 4,
// except centuries unless also divisible by 400.
//
// The French Republican calendar was historically based on astronomical observations,
// but for computational purposes we use the Gregorian-aligned rule based on the epoch.
// A French Republican year N spans from September 22 of Gregorian year (1791+N) to
// September 21 of Gregorian year (1792+N). The year is a leap year (366 days) when
// the following Gregorian year (1792+N) is a leap year.
//
// Parameters:
//   - year: French Republican year (1 = Sep 22, 1792 - Sep 21, 1793)
//
// Returns:
//   - true if the year has 6 complementary days (366 days total)
//
// Example:
//
//	IsFrenchLeapYear(4) = true   // Year 4 (Sep 22, 1795 - Sep 21, 1796), Gregorian 1796 is leap
//	IsFrenchLeapYear(12) = true  // Year 12 (Sep 22, 1803 - Sep 21, 1804), Gregorian 1804 is leap
//	IsFrenchLeapYear(16) = true  // Year 16 (Sep 22, 1807 - Sep 21, 1808), Gregorian 1808 is leap
func IsFrenchLeapYear(year int) bool {
	// The French Republican epoch is September 22, 1792 (Gregorian).
	// Year N of the calendar starts on Sep 22 of Gregorian year (1791 + N).
	// It is a leap year if the following Gregorian year (1792 + N) is a leap year.
	// This is because the leap day falls late in the year (in the complementary days).
	gregorianYear := 1791 + year + 1 // The year that follows the start year

	// Apply Gregorian leap year rules
	if gregorianYear%400 == 0 {
		return true
	}
	if gregorianYear%100 == 0 {
		return false
	}
	return gregorianYear%4 == 0
}

// FrenchToJDN converts a French Republican calendar date to Julian Day Number.
// Month 13 (COMP) represents the complementary days (1-5 or 1-6 in leap years).
//
// The French Republican calendar was used in France from 1792-1805. It has:
//   - 12 months of exactly 30 days each
//   - 5 complementary days (6 in leap years) at the end of the year
//   - Epoch: 1 Vendémiaire 1 = September 22, 1792 (Gregorian) = JDN 2375840
//
// Parameters:
//   - year: French Republican year (1 = 1792-1793)
//   - month: 1-13 (1-12 are regular months, 13 = complementary days)
//   - day: 1-30 for months 1-12, 1-5 (or 1-6 in leap years) for month 13
//
// Returns:
//   - jdn: Julian Day Number
//
// Example:
//
//	FrenchToJDN(1, 1, 1) = 2375840    // 1 Vendémiaire 1 (Sep 22, 1792)
//	FrenchToJDN(8, 1, 1) = 2378396    // 1 Vendémiaire 8 (Sep 22, 1799)
//	FrenchToJDN(14, 1, 1) = 2380587   // 1 Vendémiaire 14 (Sep 22, 1805)
func FrenchToJDN(year, month, day int) int {
	// French Republican epoch: 1 Vendémiaire 1 = September 22, 1792 (Gregorian) = JDN 2375840
	const epoch = 2375840

	// Calculate days before the start of this year
	// Count leap years before this year
	leapDays := 0
	for y := 1; y < year; y++ {
		if IsFrenchLeapYear(y) {
			leapDays++
		}
	}
	daysBeforeYear := (year-1)*365 + leapDays

	// Calculate days within this year
	daysInYear := 0
	if month <= 12 {
		// Regular months (1-12): each has exactly 30 days
		daysInYear = (month-1)*30 + day
	} else {
		// Month 13 (complementary days): after all 12 regular months
		daysInYear = 12*30 + day
	}

	// JDN = epoch + days before year + days in year - 1
	// (subtract 1 because day 1 of year 1 should equal the epoch)
	return epoch + daysBeforeYear + daysInYear - 1
}

// JDNToFrench converts a Julian Day Number to French Republican calendar date.
// Returns month=13 for complementary days.
//
// The French Republican calendar was used in France from 1792-1805. It has:
//   - 12 months of exactly 30 days each
//   - 5 complementary days (6 in leap years) at the end of the year
//   - Epoch: 1 Vendémiaire 1 = September 22, 1792 (Gregorian) = JDN 2375840
//
// Parameters:
//   - jdn: Julian Day Number
//
// Returns:
//   - year: French Republican year (1 = 1792-1793)
//   - month: 1-13 (1-12 are regular months, 13 = complementary days)
//   - day: 1-30 for months 1-12, 1-5 (or 1-6 in leap years) for month 13
//
// Example:
//
//	JDNToFrench(2375840) = (1, 1, 1)      // 1 Vendémiaire 1
//	JDNToFrench(2378396) = (8, 1, 1)      // 1 Vendémiaire 8
//	JDNToFrench(2380587) = (14, 1, 1)     // 1 Vendémiaire 14
func JDNToFrench(jdn int) (year, month, day int) {
	// French Republican epoch: 1 Vendémiaire 1 = September 22, 1792 (Gregorian) = JDN 2375840
	const epoch = 2375840

	// Days since epoch (0 = first day of year 1)
	daysSinceEpoch := jdn - epoch

	// Find the exact year by counting forward from epoch
	year = 1
	daysAccumulated := 0
	for {
		yearLength := 365
		if IsFrenchLeapYear(year) {
			yearLength = 366
		}

		if daysAccumulated+yearLength > daysSinceEpoch {
			// Found the year
			break
		}

		daysAccumulated += yearLength
		year++
	}

	// Days remaining in this year (0-based)
	dayInYear := daysSinceEpoch - daysAccumulated

	// Determine month and day
	if dayInYear < 360 {
		// Regular months (1-12): each has 30 days
		month = dayInYear/30 + 1
		day = dayInYear%30 + 1
	} else {
		// Complementary days (month 13)
		month = 13
		day = dayInYear - 360 + 1
	}

	return year, month, day
}

// IsHebrewLeapYear returns true if the Hebrew year has 13 months.
// Leap years occur in years 3, 6, 8, 11, 14, 17, 19 of each 19-year cycle.
//
// The Hebrew calendar uses a 19-year Metonic cycle where 7 out of every 19 years
// are leap years with an additional month (Adar II). The pattern ensures that
// the lunar calendar stays synchronized with the solar year.
//
// Parameters:
//   - year: Hebrew year
//
// Returns:
//   - true if the year has 13 months (leap year)
//
// Example:
//
//	IsHebrewLeapYear(5784) = true   // Year 5784 is a leap year
//	IsHebrewLeapYear(5785) = false  // Year 5785 is not a leap year
func IsHebrewLeapYear(year int) bool {
	return (7*year+1)%19 < 7
}

// HebrewMonthsInYear returns the number of months in a Hebrew year (12 or 13).
//
// Regular years have 12 months, leap years have 13 months (with Adar I and Adar II).
//
// Parameters:
//   - year: Hebrew year
//
// Returns:
//   - 12 for regular years, 13 for leap years
//
// Example:
//
//	HebrewMonthsInYear(5784) = 13  // Leap year
//	HebrewMonthsInYear(5785) = 12  // Regular year
func HebrewMonthsInYear(year int) int {
	if IsHebrewLeapYear(year) {
		return 13
	}
	return 12
}

// hebrewDelay calculates the number of days from Hebrew epoch to the start of a Hebrew year.
// This implements the complex Hebrew calendar arithmetic including the four dehiyot (postponement rules).
//
// The dehiyot ensure that:
//  1. Rosh Hashanah (1 Tishrei) doesn't fall on Sunday, Wednesday, or Friday
//  2. Yom Kippur (10 Tishrei) doesn't fall adjacent to Shabbat
//  3. Hoshana Rabbah (21 Tishrei) doesn't fall on Shabbat
//
// Reference: Dershowitz & Reingold "Calendrical Calculations"
func hebrewDelay(year int) int {
	// Calculate months elapsed since epoch (including leap years)
	monthsElapsed := 235*((year-1)/19) + // Complete 19-year cycles
		12*((year-1)%19) + // Regular months in incomplete cycle
		(7*((year-1)%19)+1)/19 // Leap months in incomplete cycle

	// Calculate parts (chelek) elapsed
	// 1 hour = 1080 parts, 1 day = 25920 parts
	// New moon calculation: 29 days 12 hours 793 parts per lunar month
	partsElapsed := 204 + 793*(monthsElapsed%1080)
	hoursElapsed := 5 + 12*monthsElapsed + 793*(monthsElapsed/1080) + partsElapsed/1080

	// Calculate day number since epoch
	// The molad of Tishrei 1 in year 1 was on Monday (day of week 1)
	// We count from the Sunday before as day 0, so the molad is on day 1
	// But since we later use day % 7 to get day of week, we need to account
	// for the fact that the epoch molad was on Monday (day 1 in 0-indexed week)
	//
	// Actually, we start counting from a base of day 1 (Monday), then add elapsed days
	day := 1 + 29*monthsElapsed + hoursElapsed/24
	parts := (hoursElapsed%24)*1080 + partsElapsed%1080

	// Apply dehiyot (postponement rules)
	// Rule 1 (Molad Zaken): If new moon is at or after noon (18 hours = 19440 parts), postpone
	if parts >= 19440 {
		day++
		parts = 0 // Reset to midnight
	}

	// Day of week (0 = Sunday, 1 = Monday, ..., 6 = Saturday)
	dayOfWeek := day % 7

	// Rule 2 (GaTaRaD): If Monday and parts >= 9 hours 204 parts, postpone to Tuesday
	// This only applies to regular (non-leap) years
	if dayOfWeek == 1 && parts >= 9*1080+204 && !IsHebrewLeapYear(year) {
		day += 1 // Tuesday
	} else if dayOfWeek == 0 && parts >= 15*1080+589 && IsHebrewLeapYear(year-1) {
		// Rule 3 (BeTuTaKFoT): If Tuesday and parts >= 15 hours 589 parts, postpone
		// This only applies when previous year was a leap year
		day += 1 // Wednesday, which will be postponed to Thursday by Rule 4
	}

	// Rule 4 (Lo ADU Rosh): Rosh Hashanah cannot fall on Sunday (0), Wednesday (3), or Friday (5)
	dayOfWeek = day % 7
	if dayOfWeek == 0 || dayOfWeek == 3 || dayOfWeek == 5 {
		day += 1
	}

	// Return the number of days SINCE the epoch
	// We started counting from day 1 (the molad of year 1), so subtract 1
	return day - 1
}

// HebrewDaysInYear returns the number of days in a Hebrew year (353-385).
//
// Hebrew year lengths:
//   - Regular years: 353 (deficient), 354 (regular), 355 (complete)
//   - Leap years: 383 (deficient), 384 (regular), 385 (complete)
//
// The variation comes from the dehiyot (postponement rules) which adjust
// the lengths of Cheshvan and Kislev to ensure proper calendar alignment.
//
// Parameters:
//   - year: Hebrew year
//
// Returns:
//   - Number of days in the year (353-385)
//
// Example:
//
//	HebrewDaysInYear(5785) = 355  // Regular complete year
//	HebrewDaysInYear(5784) = 385  // Leap complete year
func HebrewDaysInYear(year int) int {
	return hebrewDelay(year+1) - hebrewDelay(year)
}

// HebrewDaysInMonth returns the number of days in a Hebrew month (29 or 30).
//
// Month lengths vary based on the year type (deficient, regular, complete):
//   - Tishrei (1), Shevat (5), Nisan (8), Sivan (10), Av (12): always 30 days
//   - Tevet (4), Iyar (9), Tammuz (11), Elul (13): always 29 days
//   - Cheshvan (2): 29 (deficient/regular) or 30 (complete)
//   - Kislev (3): 29 (deficient) or 30 (regular/complete)
//   - Adar (6): 29 days in regular years, 30 days in leap years (as Adar I)
//   - Adar II (7): only exists in leap years, 29 days
//
// Parameters:
//   - year: Hebrew year
//   - month: Month number (1-13, using GEDCOM numbering where Tishrei=1)
//
// Returns:
//   - Number of days in the month (29 or 30)
//
// Example:
//
//	HebrewDaysInMonth(5785, 1) = 30   // Tishrei always has 30 days
//	HebrewDaysInMonth(5785, 2) = 29   // Cheshvan varies by year type
func HebrewDaysInMonth(year, month int) int {
	// Determine year type by calculating year length
	yearLength := HebrewDaysInYear(year)
	isLeap := IsHebrewLeapYear(year)

	// Base length for leap vs regular years
	var baseLength int
	if isLeap {
		baseLength = 383
	} else {
		baseLength = 353
	}

	// Determine year type: deficient (base), regular (base+1), complete (base+2)
	yearType := yearLength - baseLength // 0=deficient, 1=regular, 2=complete

	switch month {
	case 1, 5, 8, 10, 12: // Tishrei, Shevat, Nisan, Sivan, Av
		return 30
	case 4, 9, 11, 13: // Tevet, Iyar, Tammuz, Elul
		return 29
	case 2: // Cheshvan (Marcheshvan)
		// 29 days normally, 30 in complete years
		if yearType == 2 {
			return 30
		}
		return 29
	case 3: // Kislev
		// 30 days normally, 29 in deficient years
		if yearType == 0 {
			return 29
		}
		return 30
	case 6: // Adar (Adar I in leap years)
		if isLeap {
			return 30 // Adar I in leap years has 30 days
		}
		return 29 // Adar in non-leap years has 29 days
	case 7: // Adar II (only in leap years)
		if isLeap {
			return 29
		}
		// Month 7 doesn't exist in non-leap years, but return 0 for safety
		return 0
	default:
		return 0 // Invalid month
	}
}

// HebrewToJDN converts a Hebrew calendar date to Julian Day Number.
// Month numbering follows GEDCOM convention: Tishrei=1, ..., Elul=13.
// In leap years, Adar I=6 and Adar II=7.
//
// The Hebrew calendar is a lunisolar calendar with years of 12 or 13 months.
// The epoch is Tishrei 1, 1 = Monday, September 7, 3761 BC (Julian) = JDN 347998.
//
// Parameters:
//   - year: Hebrew year
//   - month: Month number (1-13, Tishrei=1 per GEDCOM convention)
//   - day: Day of month (1-30)
//
// Returns:
//   - jdn: Julian Day Number
//
// Example:
//
//	HebrewToJDN(5785, 1, 1) = 2460587   // 1 Tishrei 5785 (Rosh Hashanah)
//	HebrewToJDN(5785, 8, 15) = 2460779  // 15 Nisan 5785 (Passover)
func HebrewToJDN(year, month, day int) int {
	// Hebrew epoch: 1 Tishrei 1 = JDN 347998 (Monday, September 7, 3761 BC Julian)
	const epoch = 347998

	// Calculate days from epoch to the start of this year
	daysToYear := hebrewDelay(year)

	// Calculate days from start of year to start of this month
	daysToMonth := 0
	for m := 1; m < month; m++ {
		daysToMonth += HebrewDaysInMonth(year, m)
	}

	// JDN = epoch + days to year + days to month + day - 1
	// (subtract 1 because day 1 of year 1 should equal the epoch)
	return epoch + daysToYear + daysToMonth + day - 1
}

// JDNToHebrew converts a Julian Day Number to Hebrew calendar date.
// Returns month in GEDCOM numbering (Tishrei=1).
//
// The Hebrew calendar is a lunisolar calendar with years of 12 or 13 months.
// The epoch is Tishrei 1, 1 = Monday, September 7, 3761 BC (Julian) = JDN 347998.
//
// Parameters:
//   - jdn: Julian Day Number
//
// Returns:
//   - year: Hebrew year
//   - month: Month number (1-13, Tishrei=1 per GEDCOM convention)
//   - day: Day of month (1-30)
//
// Example:
//
//	JDNToHebrew(2460587) = (5785, 1, 1)   // 1 Tishrei 5785 (Rosh Hashanah)
//	JDNToHebrew(2460779) = (5785, 8, 15)  // 15 Nisan 5785 (Passover)
func JDNToHebrew(jdn int) (year, month, day int) {
	// Hebrew epoch: 1 Tishrei 1 = JDN 347998
	const epoch = 347998

	// Days since epoch
	daysSinceEpoch := jdn - epoch

	// Estimate the year (approximate)
	// Average Hebrew year is about 365.25 days
	estimatedYear := daysSinceEpoch/366 + 1
	if estimatedYear < 1 {
		estimatedYear = 1
	}

	// Find the exact year by checking when the next year starts
	year = estimatedYear
	for hebrewDelay(year+1) <= daysSinceEpoch {
		year++
	}
	for hebrewDelay(year) > daysSinceEpoch {
		year--
	}

	// Days from start of year
	daysIntoYear := daysSinceEpoch - hebrewDelay(year)

	// Find the month
	month = 1
	daysAccumulated := 0
	for month <= HebrewMonthsInYear(year) {
		monthLength := HebrewDaysInMonth(year, month)
		if daysAccumulated+monthLength > daysIntoYear {
			// Found the month
			break
		}
		daysAccumulated += monthLength
		month++
	}

	// Calculate day within month (1-based)
	day = daysIntoYear - daysAccumulated + 1

	return year, month, day
}
