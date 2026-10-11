package gedcom

import "testing"

// Regression tests for #563: Compare, IsBefore, IsAfter, IsEqual and
// YearsBetween read a dual date in New Style, as Bounds does, so
// "11 FEB 1731/32" is 11 FEB 1732.

func TestDate_Compare_DualYear(t *testing.T) {
	tests := []struct {
		name         string
		date1, date2 string
		want         int
	}{
		{"Jan-Mar dual date after December of Year", "11 FEB 1731/32", "1 DEC 1731", 1},
		{"Jan-Mar dual date equals New Style day", "11 FEB 1731/32", "11 FEB 1732", 0},
		{"Jan-Mar dual date before later New Style day", "11 FEB 1731/32", "12 FEB 1732", -1},
		{"dual February equals New Style February", "FEB 1731/32", "FEB 1732", 0},
		{"24 March is New Style DualYear", "24 MAR 1731/32", "24 MAR 1732", 0},
		{"25 March is New Style Year", "25 MAR 1731/32", "25 MAR 1731", 0},
		{"after March is New Style Year", "1 MAY 1731/32", "1 MAY 1731", 0},
		{"two dual dates in New Style order", "1 MAY 1731/32", "11 FEB 1731/32", -1},
		{"year-only dual reads as its earliest year", "1731/32", "1731", 0},
		{"dual March without day reads as Year", "MAR 1731/32", "1 MAR 1731", 0},
		{"dual year not following Year reads as Year", "11 FEB 1750/53", "11 FEB 1750", 0},
		{"B.C. dual date in New Style", "10 FEB 100/99 BC", "1 DEC 100 BC", 1},
		{"Julian dual date across calendars", "@#DJULIAN@ 11 FEB 1731/32", "22 FEB 1732", 0},
		{"Julian dual date in same calendar", "@#DJULIAN@ 11 FEB 1731/32", "@#DJULIAN@ 1 DEC 1731", 1},
		{"reverse order", "1 DEC 1731", "11 FEB 1731/32", -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d1, d2 := mustParse(t, tt.date1), mustParse(t, tt.date2)
			if got := d1.Compare(d2); got != tt.want {
				t.Errorf("Compare(%q, %q) = %d, want %d", tt.date1, tt.date2, got, tt.want)
			}
			if got := d1.IsBefore(d2); got != (tt.want < 0) {
				t.Errorf("IsBefore(%q, %q) = %v", tt.date1, tt.date2, got)
			}
			if got := d1.IsAfter(d2); got != (tt.want > 0) {
				t.Errorf("IsAfter(%q, %q) = %v", tt.date1, tt.date2, got)
			}
			if got := d1.IsEqual(d2); got != (tt.want == 0) {
				t.Errorf("IsEqual(%q, %q) = %v", tt.date1, tt.date2, got)
			}
		})
	}
}

// Compare must agree with Bounds' earliest day on every unmodified dual date.
func TestDate_Compare_DualYearAgreesWithBounds(t *testing.T) {
	for _, s := range []string{
		"11 FEB 1731/32", "24 MAR 1731/32", "25 MAR 1731/32", "1 MAY 1731/32",
		"FEB 1731/32", "MAR 1731/32", "1731/32", "10 FEB 100/99 BC",
	} {
		d := mustParse(t, s)
		earliest, _, ok := d.Bounds()
		if !ok {
			t.Fatalf("Bounds(%q) not ok", s)
		}
		y, m, day := JDNToGregorian(earliest)
		year, isBC := FromAstronomicalYear(y)
		first := &Date{Year: year, Month: m, Day: day, IsBC: isBC, Calendar: CalendarGregorian}
		if got := d.Compare(first); got != 0 {
			t.Errorf("Compare(%q, its earliest day %d-%d-%d) = %d, want 0", s, y, m, day, got)
		}
	}
}

func TestYearsBetween_DualYear(t *testing.T) {
	tests := []struct {
		name         string
		date1, date2 string
		want         int
		wantExact    bool
	}{
		{"complete dual date", "11 FEB 1731/32", "11 FEB 1760", 28, true},
		{"complete dual date, reversed", "11 FEB 1760", "11 FEB 1731/32", 28, true},
		{"dual date after March", "1 MAY 1731/32", "1 MAY 1760", 29, true},
		{"partial dual date", "FEB 1731/32", "1760", 28, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, exact, err := YearsBetween(mustParse(t, tt.date1), mustParse(t, tt.date2))
			if err != nil {
				t.Fatalf("YearsBetween: %v", err)
			}
			if got != tt.want || exact != tt.wantExact {
				t.Errorf("YearsBetween(%q, %q) = %d, %v; want %d, %v", tt.date1, tt.date2, got, exact, tt.want, tt.wantExact)
			}
		})
	}
}

func TestValidateBirthBeforeDeath_DualYear(t *testing.T) {
	// Death 10 FEB 1731/32 is 10 FEB 1732, after birth 20 DEC 1731.
	if err := ValidateBirthBeforeDeath(mustParse(t, "20 DEC 1731"), mustParse(t, "10 FEB 1731/32")); err != nil {
		t.Errorf("false positive: %v", err)
	}
	// Birth 10 FEB 1731/32 is 10 FEB 1732, after death 20 DEC 1731.
	if err := ValidateBirthBeforeDeath(mustParse(t, "10 FEB 1731/32"), mustParse(t, "20 DEC 1731")); err == nil {
		t.Error("false negative: death before a New Style birth not reported")
	}
}
