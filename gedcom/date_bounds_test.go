package gedcom

import (
	"math"
	"testing"
)

func TestDate_Bounds(t *testing.T) {
	g := GregorianToJDN
	tests := []struct {
		name         string
		input        string
		wantEarliest int
		wantLatest   int
	}{
		// Exact and partial dates span the period they name.
		{"exact date", "2 FEB 1503", g(1503, 2, 2), g(1503, 2, 2)},
		{"year only", "1503", g(1503, 1, 1), g(1503, 12, 31)},
		{"month and year", "FEB 1900", g(1900, 2, 1), g(1900, 2, 28)},
		{"leap February", "FEB 2000", g(2000, 2, 1), g(2000, 2, 29)},
		{"December", "DEC 1900", g(1900, 12, 1), g(1900, 12, 31)},
		{"B.C. year", "44 BC", g(-43, 1, 1), g(-43, 12, 31)},

		// Approximate dates are bounded by the period they name.
		{"about", "ABT 1090", g(1090, 1, 1), g(1090, 12, 31)},
		{"calculated", "CAL MAR 1850", g(1850, 3, 1), g(1850, 3, 31)},
		{"estimated", "EST 1850", g(1850, 1, 1), g(1850, 12, 31)},
		{"interpreted", "INT 1850 (about eighteen fifty)", g(1850, 1, 1), g(1850, 12, 31)},

		// BEF/AFT and single-ended periods are open-ended.
		{"before", "BEF 1850", math.MinInt, g(1850, 12, 31)},
		{"after", "AFT 1078", g(1078, 1, 1), math.MaxInt},
		{"from", "FROM 1880", g(1880, 1, 1), math.MaxInt},
		{"to", "TO 1920", math.MinInt, g(1920, 12, 31)},

		// Ranges and periods span both endpoints.
		{"between", "BET 1850 AND JUN 1860", g(1850, 1, 1), g(1860, 6, 30)},
		{"between reversed", "BET 1860 AND 1850", g(1850, 1, 1), g(1860, 12, 31)},
		{"from to", "FROM 1880 TO 1920", g(1880, 1, 1), g(1920, 12, 31)},
		{"between mixed calendars", "BET @#DJULIAN@ 1 JAN 1700 AND @#DGREGORIAN@ 1 JAN 1700",
			g(1700, 1, 1), JulianToJDN(1700, 1, 1)},

		// Other calendars.
		{"Julian year", "@#DJULIAN@ 1700", JulianToJDN(1700, 1, 1), JulianToJDN(1701, 1, 1) - 1},
		{"Julian December", "@#DJULIAN@ DEC 1700", JulianToJDN(1700, 12, 1), JulianToJDN(1700, 12, 31)},
		{"Hebrew year", "@#DHEBREW@ 5785", HebrewToJDN(5785, 1, 1), HebrewToJDN(5786, 1, 1) - 1},
		{"Hebrew Adar in a common year", "@#DHEBREW@ ADR 5785", HebrewToJDN(5785, 6, 1), HebrewToJDN(5785, 6, 29)},
		{"Hebrew Elul", "@#DHEBREW@ ELL 5785", HebrewToJDN(5785, 13, 1), HebrewToJDN(5786, 1, 1) - 1},
		{"French year", "@#DFRENCH R@ 8", FrenchToJDN(8, 1, 1), FrenchToJDN(9, 1, 1) - 1},
		{"French Fructidor", "@#DFRENCH R@ FRUC 8", FrenchToJDN(8, 12, 1), FrenchToJDN(8, 12, 30)},
		{"French complementary days", "@#DFRENCH R@ COMP 8", FrenchToJDN(8, 13, 1), FrenchToJDN(9, 1, 1) - 1},

		// A dual date is read in New Style: the dual year from 1 January to
		// 24 March, the first year from 25 March. Where the date does not say
		// which side of 25 March it falls, it spans both readings.
		{"dual Jan-Mar day", "11 FEB 1731/32", g(1732, 2, 11), g(1732, 2, 11)},
		{"dual February", "FEB 1731/32", g(1732, 2, 1), g(1732, 2, 29)},
		{"dual 24 March", "24 MAR 1731/32", g(1732, 3, 24), g(1732, 3, 24)},
		{"dual 25 March", "25 MAR 1731/32", g(1731, 3, 25), g(1731, 3, 25)},
		{"dual after March", "1 MAY 1731/32", g(1731, 5, 1), g(1731, 5, 1)},
		{"dual March, no day", "MAR 1731/32", g(1731, 3, 1), g(1732, 3, 31)},
		{"dual year only", "1731/32", g(1731, 1, 1), g(1732, 12, 31)},
		{"dual century", "1 JAN 1699/00", g(1700, 1, 1), g(1700, 1, 1)},
		{"dual not following", "11 FEB 1750/53", g(1750, 2, 11), g(1753, 2, 11)},
		{"dual B.C.", "1 JAN 100/99 BC", g(-98, 1, 1), g(-98, 1, 1)},
		{"dual Julian", "@#DJULIAN@ 11 FEB 1731/32", JulianToJDN(1732, 2, 11), JulianToJDN(1732, 2, 11)},
		{"dual range end", "BET 1730 AND 11 FEB 1731/32", g(1730, 1, 1), g(1732, 2, 11)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("ParseDate(%q) error: %v", tt.input, err)
			}
			earliest, latest, ok := d.Bounds()
			if !ok {
				t.Fatalf("Bounds(%q) ok = false, want true", tt.input)
			}
			if earliest != tt.wantEarliest || latest != tt.wantLatest {
				t.Errorf("Bounds(%q) = (%d, %d), want (%d, %d)",
					tt.input, earliest, latest, tt.wantEarliest, tt.wantLatest)
			}
		})
	}
}

func mustParse(t *testing.T, s string) *Date {
	t.Helper()
	d, err := ParseDate(s)
	if err != nil {
		t.Fatalf("ParseDate(%q) error: %v", s, err)
	}
	return d
}

func TestDate_Bounds_NotOK(t *testing.T) {
	phrase, err := ParseDate("(unknown)")
	if err != nil {
		t.Fatalf("ParseDate error: %v", err)
	}
	tests := []struct {
		name string
		date *Date
	}{
		{"nil", nil},
		{"zero value", &Date{}},
		{"phrase", phrase},
		{"phrase with a year", &Date{Year: 1850, IsPhrase: true}},
		{"unknown calendar", &Date{Year: 1850, Calendar: Calendar(99)}},
		{"unknown calendar with month", &Date{Year: 1850, Month: 3, Calendar: Calendar(99)}},
		{"unknown calendar with day", &Date{Year: 1850, Month: 3, Day: 2, Calendar: Calendar(99)}},
		{"between without end date", &Date{Year: 1850, Modifier: ModifierBetween}},
		{"range with yearless end date", &Date{Year: 1850, Modifier: ModifierFromTo, EndDate: &Date{}}},
		{"day past the end of the month", mustParse(t, "31 FEB 1850")},
		{"sixth complementary day in a common year", mustParse(t, "@#DFRENCH R@ 6 COMP 8")},
		{"Adar II in a common Hebrew year", mustParse(t, "@#DHEBREW@ ADS 5785")},
		{"day in Adar II in a common Hebrew year", mustParse(t, "@#DHEBREW@ 1 ADS 5785")},
		{"negative day", &Date{Year: 1850, Month: 3, Day: -1}},
		{"negative month", &Date{Year: 1850, Month: -1}},
		{"negative Julian month", &Date{Year: 1850, Month: -1, Calendar: CalendarJulian}},
		{"year beyond the range", &Date{Year: maxBoundsYear + 1}},
		{"B.C. year beyond the range", &Date{Year: -maxBoundsYear - 1}},
		{"huge French year", &Date{Year: math.MaxInt, Calendar: CalendarFrenchRepublican}},
		{"huge dual year", &Date{Year: 1850, DualYear: math.MaxInt}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, ok := tt.date.Bounds(); ok {
				t.Errorf("Bounds() ok = true, want false")
			}
		})
	}
}

// TestDate_Bounds_ProvesOrder covers the orderings from issue #566: Compare
// puts the first date before the second, but the bounds overlap, so the order
// is not provable.
func TestDate_Bounds_ProvesOrder(t *testing.T) {
	tests := []struct {
		a, b       string
		wantProven bool
	}{
		{"1503", "2 FEB 1503", false},
		{"AFT 1078", "ABT 1090", false},
		{"10 FEB 1731/32", "20 DEC 1731", false},
		{"1940", "1950", true},
		{"BEF 1950", "1960", true},
		{"20 DEC 1731", "10 FEB 1731/32", true},
	}
	for _, tt := range tests {
		t.Run(tt.a+" before "+tt.b, func(t *testing.T) {
			a, _ := ParseDate(tt.a)
			b, _ := ParseDate(tt.b)
			_, aLatest, _ := a.Bounds()
			bEarliest, _, _ := b.Bounds()
			if got := aLatest < bEarliest; got != tt.wantProven {
				t.Errorf("latest(%q) < earliest(%q) = %v, want %v", tt.a, tt.b, got, tt.wantProven)
			}
		})
	}
}
