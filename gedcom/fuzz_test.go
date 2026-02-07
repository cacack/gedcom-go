package gedcom

import "testing"

// FuzzParseDate fuzzes the ParseDate function with arbitrary string input.
// Seeds cover all supported date formats: exact, partial, modified, ranges,
// periods, dual dating, B.C. dates, calendar escapes, phrases, and interpreted dates.
func FuzzParseDate(f *testing.F) {
	seeds := []string{
		// Exact dates
		"25 DEC 2020",
		"1 JAN 1900",
		"14 FEB 1890",

		// Partial dates
		"JAN 1900",
		"1850",

		// Modifiers
		"ABT 1850",
		"CAL 1850",
		"EST 1850",
		"BEF 1850",
		"AFT 1850",

		// Ranges
		"BET 1850 AND 1860",
		"BET 1 JAN 1900 AND 31 DEC 1910",

		// Periods
		"FROM 1880",
		"TO 1920",
		"FROM 1880 TO 1920",
		"FROM JAN 1880 TO DEC 1920",

		// Dual dating
		"21 FEB 1750/51",
		"1 MAR 1750/1751",

		// B.C. dates
		"44 BC",
		"100 B.C.",
		"500 BCE",
		"1000 B.C.E.",

		// Calendar escapes
		"@#DJULIAN@ 25 DEC 1752",
		"@#DHEBREW@ 15 NSN 5784",
		"@#DFRENCH R@ 1 VEND 1",
		"@#DGREGORIAN@ 25 DEC 2020",

		// Date phrases
		"(unknown)",
		"(about 1850)",
		"(before the war)",

		// Interpreted dates
		"INT 1850 (about eighteen fifty)",
		"INT 25 DEC 1850 (Christmas day)",
		"INT 1850",

		// Edge cases
		"",
		"   ",
		"NOTADATE",
		"32 DEC 2020",
		"0 JAN 2020",
		"JAN",
		"BET AND",
		"FROM TO",
		"@#DINVALID@ 25 DEC 2020",

		// Hebrew months
		"@#DHEBREW@ 15 TSH 5784",
		"@#DHEBREW@ 1 ADR 5784",
		"@#DHEBREW@ 1 ADS 5784",

		// French Republican months
		"@#DFRENCH R@ 1 BRUM 1",
		"@#DFRENCH R@ 1 COMP 1",

		// Case variations
		"25 dec 2020",
		"abt 1850",
		"bet 1850 and 1860",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, s string) {
		// Errors are expected; panics are not.
		_, _ = ParseDate(s)
	})
}
