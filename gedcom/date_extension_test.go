package gedcom

import (
	"errors"
	"strings"
	"testing"
)

// date_extension_test.go covers GEDCOM 7.0 extension calendars, months and
// epochs (ADR 0014, spec-interpretation policy SI-018).

func TestParseDate_ExtensionTags(t *testing.T) {
	tests := []struct {
		in                              string
		calendar                        Calendar
		calendarTag, monthTag, epochTag string
		day, month, year                int
	}{
		// gedcom.io's extensions.ged
		{"FRENCH_R 2 _JOUR 8", CalendarFrenchRepublican, "", "_JOUR", "", 2, 0, 8},
		{"_CALENDRIER 4 COMP 8", CalendarExtension, "_CALENDRIER", "COMP", "", 4, 0, 8},
		{"_CALENDAR 8 _MONTH 190 _EPOCH", CalendarExtension, "_CALENDAR", "_MONTH", "_EPOCH", 8, 0, 190},
		{"_CAL2 23 _MON2 88 _EP2", CalendarExtension, "_CAL2", "_MON2", "_EP2", 23, 0, 88},
		// the other shapes
		{"_MAYAN 5 _KIN 2000", CalendarExtension, "_MAYAN", "_KIN", "", 5, 0, 2000},
		{"_MAYAN 2000", CalendarExtension, "_MAYAN", "", "", 0, 0, 2000},
		{"_MAYAN 2000 BCE", CalendarExtension, "_MAYAN", "", "BCE", 0, 0, 2000},
		{"_MAYAN JAN 2000", CalendarExtension, "_MAYAN", "JAN", "", 0, 0, 2000},
		{"1 JAN 2000 _AD", CalendarGregorian, "", "", "_AD", 1, 1, 2000},
		{"_LEAP 2000", CalendarExtension, "_LEAP", "", "", 0, 0, 2000},
		{"5 _LEAP 2000", CalendarGregorian, "", "_LEAP", "", 5, 0, 2000},
		{"JULIAN _X 1700", CalendarJulian, "", "_X", "", 0, 0, 1700},
		{"ABT _MAYAN 5 _KIN 2000", CalendarExtension, "_MAYAN", "_KIN", "", 5, 0, 2000},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			d, err := ParseDate(tt.in)
			if err != nil {
				t.Fatalf("ParseDate(%q) error: %v", tt.in, err)
			}
			if d.Calendar != tt.calendar || d.CalendarTag != tt.calendarTag || d.MonthTag != tt.monthTag || d.EpochTag != tt.epochTag {
				t.Errorf("calendar, tags = %v %q %q %q; want %v %q %q %q",
					d.Calendar, d.CalendarTag, d.MonthTag, d.EpochTag, tt.calendar, tt.calendarTag, tt.monthTag, tt.epochTag)
			}
			if d.Day != tt.day || d.Month != tt.month || d.Year != tt.year || d.IsBC {
				t.Errorf("day, month, year, IsBC = %d %d %d %v; want %d %d %d false", d.Day, d.Month, d.Year, d.IsBC, tt.day, tt.month, tt.year)
			}
			if d.Original != tt.in || !d.HasExtension() {
				t.Errorf("Original %q, HasExtension %v; want %q, true", d.Original, d.HasExtension(), tt.in)
			}
			if err := d.Validate(); err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}

// TestParseDate_ExtensionRange is the BIRT.DATE of extensions.ged: each
// endpoint keeps its own tags.
func TestParseDate_ExtensionRange(t *testing.T) {
	d, err := ParseDate("BET FRENCH_R 2 _JOUR 8 AND _CALENDRIER 4 COMP 8")
	if err != nil {
		t.Fatal(err)
	}
	if d.Calendar != CalendarFrenchRepublican || d.MonthTag != "_JOUR" || d.Day != 2 || d.Year != 8 {
		t.Errorf("start = %+v", d)
	}
	end := d.EndDate
	if end == nil || end.Calendar != CalendarExtension || end.CalendarTag != "_CALENDRIER" || end.MonthTag != "COMP" || end.Day != 4 || end.Year != 8 {
		t.Errorf("end = %+v", end)
	}
	if _, _, ok := d.Bounds(); ok {
		t.Error("Bounds() ok for an extension range")
	}

	// A standard start with an extension end
	d, err = ParseDate("FROM 1 JAN 1700 TO _MAYAN 2000")
	if err != nil {
		t.Fatal(err)
	}
	if d.HasExtension() || !d.EndDate.HasExtension() {
		t.Errorf("HasExtension start %v end %v; want false, true", d.HasExtension(), d.EndDate.HasExtension())
	}
	if _, _, ok := d.Bounds(); ok {
		t.Error("Bounds() ok for a range with an extension end")
	}
	if _, err := d.ToGregorian(); err == nil {
		t.Error("ToGregorian() of a range with an extension end succeeded")
	}
}

func TestParseDate_ExtensionTagsRejected(t *testing.T) {
	for _, in := range []string{
		"_MAYAN",             // a calendar needs a date
		"_MAYAN _KIN",        // no year
		"FRENCH_R 0 _JOUR 8", // day 0 is Date's absent value (SI-018)
		"_MAYAN 0",           // year 0 likewise
		"_MAYAN 5 _kin 2000", // not a tag
		"_MAYAN 5 kin 2000",  // not a tag
		"_mayan 5 _KIN 2000", // not a calendar: too many fields
		"_MAYAN 2000 BC",     // an extension calendar's epoch is BCE or a tag
		"HEBREW 1 _ADAR 5000 BCE",
		"FRENCH_R 2 _JOUR 8 BC",
	} {
		if d, err := ParseDate(in); err == nil {
			t.Errorf("ParseDate(%q) = %+v, want an error", in, d)
		} else if !errors.Is(err, ErrInvalidDate) {
			t.Errorf("ParseDate(%q) error %v does not wrap ErrInvalidDate", in, err)
		}
	}
}

// TestDate_ExtensionArithmetic holds every API that places a date in time
// to its documented result for a date with an extension tag.
func TestDate_ExtensionArithmetic(t *testing.T) {
	gregorian := mustParseDate("1 JAN 2000")
	for _, in := range []string{"_MAYAN 5 _KIN 2000", "FRENCH_R 2 _JOUR 8", "1 JAN 2000 _AD", "_CAL 2000 BCE"} {
		t.Run(in, func(t *testing.T) {
			d := mustParseDate(in)
			if _, _, ok := d.Bounds(); ok {
				t.Error("Bounds() ok")
			}
			if _, _, ok := AgeAt(gregorian, d); ok {
				t.Error("AgeAt() ok")
			}
			if _, err := d.ToGregorian(); err == nil || !strings.Contains(err.Error(), "extension") {
				t.Errorf("ToGregorian() error = %v, want one naming the extension", err)
			}
			if _, err := d.ToTime(); err == nil || !strings.Contains(err.Error(), "extension") {
				t.Errorf("ToTime() error = %v, want one naming the extension", err)
			}
			if _, _, err := YearsBetween(gregorian, d); err == nil || !strings.Contains(err.Error(), "second date") {
				t.Errorf("YearsBetween(gregorian, d) error = %v, want one naming the second date", err)
			}
			if _, _, err := YearsBetween(d, d); err == nil || !strings.Contains(err.Error(), "first date") {
				t.Errorf("YearsBetween(d, d) error = %v, want one naming the first date", err)
			}
			if d.Compare(gregorian) != 1 || gregorian.Compare(d) != -1 {
				t.Errorf("Compare with a Gregorian date = %d, %d; want 1, -1", d.Compare(gregorian), gregorian.Compare(d))
			}
			if d.IsBefore(gregorian) || d.IsAfter(gregorian) || gregorian.IsBefore(d) || gregorian.IsAfter(d) {
				t.Error("IsBefore or IsAfter true against a Gregorian date")
			}
			if !d.IsEqual(mustParseDate(in)) || d.Compare(mustParseDate(in)) != 0 {
				t.Error("two parses of the same value are not equal")
			}
		})
	}
}

// TestDate_ExtensionCompareOrder checks that extension dates sort among
// themselves by Original, after every other date.
func TestDate_ExtensionCompareOrder(t *testing.T) {
	a, b := mustParseDate("_A 2000"), mustParseDate("_B 1000")
	if a.Compare(b) != -1 || b.Compare(a) != 1 {
		t.Errorf("Compare(_A 2000, _B 1000) = %d, want -1 by Original", a.Compare(b))
	}
	if a.IsBefore(b) || b.IsAfter(a) {
		t.Error("IsBefore or IsAfter claims an order between extension dates")
	}
	if got := mustParseDate("9999").Compare(a); got != -1 {
		t.Errorf("Compare(9999, _A 2000) = %d, want -1", got)
	}
}

// TestDate_ExtensionCompareWithoutOriginal checks that two extension dates
// with no Original, a range's end date or a Date built in code, are equal
// only when their calendar, day, month, year and epoch are.
func TestDate_ExtensionCompareWithoutOriginal(t *testing.T) {
	a := mustParseDate("BET 1 JAN 2000 AND _MAYAN 5 _KIN 2000").EndDate
	b := mustParseDate("BET 1 JAN 2000 AND _AZTEC 1 _XX 1").EndDate
	if a.IsEqual(b) || a.Compare(b) == 0 || a.Compare(b) != -b.Compare(a) {
		t.Errorf("end dates _MAYAN 5 _KIN 2000 and _AZTEC 1 _XX 1: IsEqual %v, Compare %d, %d", a.IsEqual(b), a.Compare(b), b.Compare(a))
	}
	if !a.IsEqual(mustParseDate("BET 2 JAN 2000 AND _MAYAN 5 _KIN 2000").EndDate) {
		t.Error("equal extension end dates compare unequal")
	}

	mayan := &Date{Calendar: CalendarExtension, CalendarTag: "_MAYAN", Year: 2000}
	tests := []*Date{
		{Calendar: CalendarExtension, CalendarTag: "_AZTEC", Year: 2000},
		{Calendar: CalendarExtension, CalendarTag: "_MAYAN", Year: 1},
		{Calendar: CalendarExtension, CalendarTag: "_MAYAN", Year: 2000, Day: 5, MonthTag: "_KIN"},
		{Calendar: CalendarExtension, CalendarTag: "_MAYAN", Year: 2000, EpochTag: "_X"},
		{Calendar: CalendarGregorian, Year: 2000, Month: 3, EpochTag: "_AD"},
	}
	for _, other := range tests {
		if mayan.IsEqual(other) || mayan.Compare(other) != -other.Compare(mayan) {
			t.Errorf("_MAYAN 2000 vs %+v: IsEqual %v, Compare %d, %d", other, mayan.IsEqual(other), mayan.Compare(other), other.Compare(mayan))
		}
	}
	if !mayan.IsEqual(&Date{Calendar: CalendarExtension, CalendarTag: "_MAYAN", Year: 2000}) {
		t.Error("identical hand-built extension dates compare unequal")
	}
}

// TestDate_ExtensionValidate checks that Validate still checks the year range
// and dual year of an extension date, but not its day.
func TestDate_ExtensionValidate(t *testing.T) {
	if err := mustParseDate("31 FEB 2000 _AD").Validate(); err != nil {
		t.Errorf("Validate(31 FEB 2000 _AD) = %v, want nil: the epoch's leap years are unknown", err)
	}
	if err := mustParseDate("_MAYAN 5 _KIN 2000000").Validate(); err == nil {
		t.Error("Validate() accepted a year beyond 1,000,000")
	}
	if err := mustParseDate("_MAYAN 1750/53").Validate(); err == nil {
		t.Error("Validate() accepted a dual year that does not follow")
	}
}

func TestDate_HasExtension(t *testing.T) {
	var nilDate *Date
	if nilDate.HasExtension() || mustParseDate("1 JAN 2000").HasExtension() {
		t.Error("HasExtension() true for a nil or standard date")
	}
	if !(&Date{Calendar: CalendarExtension, Year: 1}).HasExtension() {
		t.Error("HasExtension() false for CalendarExtension with no tag")
	}
	if got := (&Date{Calendar: CalendarExtension, Year: 1}).errExtension("x").Error(); !strings.Contains(got, "no tag recorded") {
		t.Errorf("errExtension with no tag = %q", got)
	}
	if CalendarExtension.String() != "Extension" {
		t.Errorf("CalendarExtension.String() = %q", CalendarExtension.String())
	}
}

func TestCloneDate_ExtensionTags(t *testing.T) {
	d := mustParseDate("_CALENDAR 8 _MONTH 190 _EPOCH")
	c := cloneDate(d)
	if c.CalendarTag != d.CalendarTag || c.MonthTag != d.MonthTag || c.EpochTag != d.EpochTag {
		t.Errorf("cloneDate lost tags: %+v", c)
	}
}
