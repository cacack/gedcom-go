package gedcom

import (
	"math"
	"testing"
)

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

		// Calendar escapes combined with modifiers (the escape is part of <date>)
		"ABT @#DFRENCH R@ 1 VEND 1",
		"BET @#DHEBREW@ 1 NSN 5700 AND @#DHEBREW@ 30 ELL 5700",
		"BET 1700 AND @#DJULIAN@ 1750",
		"FROM @#DJULIAN@ 1700 TO @#DJULIAN@ 1750",
		"INT @#DJULIAN@ 1700 (about seventeen hundred)",

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

// coordinateSeeds are the single-component coordinate spellings shared by both
// coordinate fuzz targets: valid forms, the non-decimal spellings ParseFloat
// would otherwise accept (#504), whitespace variants, and degenerate input.
var coordinateSeeds = []string{
	// Valid forms, all four directions in both cases
	"N42.3601",
	"S33.8688",
	"E151.2093",
	"W71.0589",
	"n42.3601",
	"s33.8688",
	"e151.2093",
	"w71.0589",
	"N0",
	"E0",
	"N90",
	"W180",
	"N.5",
	"N5.",

	// Whitespace variants
	"  N42.3601  ",
	"\tW71.0589\n",
	"",
	"   ",

	// Degenerate input
	"N",
	"X42.3601",
	"42.3601",
	"Nabc",

	// Signed values (the direction carries the sign)
	"N-42.3601",
	"N+42.3601",
	"-Inf",
	"+Inf",

	// Non-decimal spellings strconv.ParseFloat would otherwise accept (#504)
	"NaN",
	"nan",
	"Inf",
	"Infinity",
	"0x1p3",
	"Nnan",
	"NNaN",
	"Ninf",
	"Einf",
	"Ninfinity",
	"Sinfinity",
	"N0x1p3",
	"S0X1P-2",
	"N1e2",
	"N1_0.5",
	"N1.2.3",
	"N.",

	// Out of range for either axis
	"N91",
	"E181",
	"S1000000",
}

// FuzzParseCoordinate fuzzes ParseCoordinate with arbitrary string input.
// Invariant: a nil error implies a finite result. Per ADR 0007 the library must
// never panic, which the fuzzer enforces inherently — a panic fails the run.
//
// ParseCoordinate deliberately does not range-check, because a single component
// carries no information about which axis it belongs to, so no range assertion
// is made here; Coordinates.AsDecimal owns that check.
func FuzzParseCoordinate(f *testing.F) {
	for _, seed := range coordinateSeeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, s string) {
		v, err := ParseCoordinate(s)
		if err != nil {
			return
		}
		if math.IsNaN(v) {
			t.Errorf("ParseCoordinate(%q) = NaN with nil error", s)
		}
		if math.IsInf(v, 0) {
			t.Errorf("ParseCoordinate(%q) = %v with nil error, want finite", s, v)
		}
	})
}

// FuzzCoordinatesAsDecimal fuzzes Coordinates.AsDecimal with arbitrary
// latitude/longitude pairs. Invariant: a nil error implies both results are
// finite and within their axis range. Per ADR 0007 the library must never
// panic, which the fuzzer enforces inherently — a panic fails the run.
func FuzzCoordinatesAsDecimal(f *testing.F) {
	for _, seed := range coordinateSeeds {
		f.Add(seed, seed)
	}
	// Mixed pairs: valid with invalid, wrong axis, and single-component.
	f.Add("N42.3601", "W71.0589")
	f.Add("S33.8688", "E151.2093")
	f.Add("N42.3601", "Enan")
	f.Add("Nnan", "W71.0589")
	f.Add("N42.3601", "N71.0589")
	f.Add("E42.3601", "W71.0589")
	f.Add("N42.3601", "")
	f.Add("", "W71.0589")
	f.Add("N0", "E0")

	f.Fuzz(func(t *testing.T, lat, long string) {
		c := &Coordinates{Latitude: lat, Longitude: long}
		gotLat, gotLong, err := c.AsDecimal()
		if err != nil {
			return
		}
		if math.IsNaN(gotLat) || math.IsInf(gotLat, 0) {
			t.Errorf("AsDecimal(%q, %q) lat = %v with nil error, want finite", lat, long, gotLat)
		}
		if math.IsNaN(gotLong) || math.IsInf(gotLong, 0) {
			t.Errorf("AsDecimal(%q, %q) long = %v with nil error, want finite", lat, long, gotLong)
		}
		if gotLat < -90 || gotLat > 90 {
			t.Errorf("AsDecimal(%q, %q) lat = %v with nil error, want within [-90, 90]", lat, long, gotLat)
		}
		if gotLong < -180 || gotLong > 180 {
			t.Errorf("AsDecimal(%q, %q) long = %v with nil error, want within [-180, 180]", lat, long, gotLong)
		}
	})
}
