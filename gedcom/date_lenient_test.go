package gedcom

import (
	"errors"
	"reflect"
	"testing"
)

func TestParseDateLenient_Recovered(t *testing.T) {
	tests := []struct {
		input  string
		readAs string
	}{
		// Full English month names, any case.
		{"June 1880", "JUN 1880"},
		{"12 December 1846", "12 DEC 1846"},
		{"june 1880", "JUN 1880"},
		{"JUNE 1880", "JUN 1880"},
		{"1 January 1900", "1 JAN 1900"},
		{"February 1900", "FEB 1900"},
		{"March 1900", "MAR 1900"},
		{"April 1900", "APR 1900"},
		{"July 1900", "JUL 1900"},
		{"August 1900", "AUG 1900"},
		{"September 1900", "SEP 1900"},
		{"October 1900", "OCT 1900"},
		{"November 1900", "NOV 1900"},
		// A month code with a trailing period.
		{"3 Jan. 1900", "3 JAN 1900"},
		{"May. 1900", "MAY 1900"},
		// English modifiers and dotted keywords.
		{"about 1873", "ABT 1873"},
		{"About 1873", "ABT 1873"},
		{"abt. 1873", "ABT 1873"},
		{"Abt. 1790", "ABT 1790"},
		{"before 1900", "BEF 1900"},
		{"Bef. 1900", "BEF 1900"},
		{"after 1900", "AFT 1900"},
		{"aft. 1900", "AFT 1900"},
		{"est. 1900", "EST 1900"},
		{"cal. 1900", "CAL 1900"},
		{"between 1880 and 1885", "BET 1880 AND 1885"},
		{"BETWEEN 1880 AND 1885", "BET 1880 AND 1885"},
		{"bet. 1880 and 1885", "BET 1880 AND 1885"},
		// Spellings combine, and surrounding whitespace is ignored.
		{"about  June 1880", "ABT JUN 1880"},
		{" between June 1880 and 4 July 1885 ", "BET JUN 1880 AND 4 JUL 1885"},
		{"FROM June 1880 TO July 1881", "FROM JUN 1880 TO JUL 1881"},
		{"@#DJULIAN@ 3 March 1700", "@#DJULIAN@ 3 MAR 1700"},
		{"12 December 44 BC", "12 DEC 44 BC"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, readAs, err := ParseDateLenient(tt.input)
			if err != nil {
				t.Fatalf("ParseDateLenient(%q) error = %v", tt.input, err)
			}
			if readAs != tt.readAs {
				t.Errorf("readAs = %q, want %q", readAs, tt.readAs)
			}
			if got.Original != tt.input {
				t.Errorf("Original = %q, want the input %q", got.Original, tt.input)
			}
			want, err := ParseDate(tt.readAs)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.readAs, err)
			}
			want.Original = tt.input
			if !reflect.DeepEqual(got, want) {
				t.Errorf("ParseDateLenient(%q) = %+v, want %+v", tt.input, got, want)
			}
			if _, err := ParseDate(tt.input); err == nil {
				t.Errorf("ParseDate(%q) succeeded; strict parsing must still reject it", tt.input)
			}
		})
	}
}

func TestParseDateLenient_Modifiers(t *testing.T) {
	tests := []struct {
		input    string
		modifier DateModifier
		year     int
		endYear  int
	}{
		{"about 1873", ModifierAbout, 1873, 0},
		{"abt. 1873", ModifierAbout, 1873, 0},
		{"before 1900", ModifierBefore, 1900, 0},
		{"after 1900", ModifierAfter, 1900, 0},
		{"between 1880 and 1885", ModifierBetween, 1880, 1885},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, _, err := ParseDateLenient(tt.input)
			if err != nil {
				t.Fatalf("ParseDateLenient(%q) error = %v", tt.input, err)
			}
			if got.Modifier != tt.modifier || got.Year != tt.year {
				t.Errorf("got modifier %v year %d, want %v %d", got.Modifier, got.Year, tt.modifier, tt.year)
			}
			if tt.endYear == 0 {
				if got.EndDate != nil {
					t.Errorf("EndDate = %+v, want nil", got.EndDate)
				}
				return
			}
			if got.EndDate == nil || got.EndDate.Year != tt.endYear {
				t.Errorf("EndDate = %+v, want year %d", got.EndDate, tt.endYear)
			}
		})
	}
}

// A date ParseDate accepts comes back as ParseDate gives it, with no readAs.
func TestParseDateLenient_Standard(t *testing.T) {
	for _, input := range []string{
		"JUN 1880", "jun 1880", "ABT 1873", "BET 1880 AND 1885", "bet 1880 and 1885",
		"(about June)", "INT 1850 (about June 1850)", "@#DHEBREW@ 15 NSN 5785", "1880",
	} {
		t.Run(input, func(t *testing.T) {
			got, readAs, err := ParseDateLenient(input)
			if err != nil {
				t.Fatalf("ParseDateLenient(%q) error = %v", input, err)
			}
			if readAs != "" {
				t.Errorf("readAs = %q, want empty for a standard date", readAs)
			}
			want, _ := ParseDate(input)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("ParseDateLenient(%q) = %+v, want %+v", input, got, want)
			}
		})
	}
}

// Anything outside the list fails with the error ParseDate returns.
func TestParseDateLenient_Rejected(t *testing.T) {
	for _, input := range []string{
		"summer",
		"Bane",
		"1935-1993",
		"December 12, 1846",
		"circa 1873",
		"Sept. 1880",
		"1880 about",
		"about",
		"about summer",
		"32nd June 1880",
		"1880 and 1885",
		"INT June 1880 (a guess)",
		"about 1873 (June)",
		"@#DHEBREW@ 1 June 5600",
		"HEBREW 1 June 5600",
		"@#DFRENCH R@ 1 June 5",
		"FRENCH_R about 5",
		"",
		"   ",
	} {
		t.Run(input, func(t *testing.T) {
			_, wantErr := ParseDate(input)
			if wantErr == nil {
				t.Fatalf("ParseDate(%q) succeeded; not a rejection case", input)
			}
			got, readAs, err := ParseDateLenient(input)
			if err == nil {
				t.Fatalf("ParseDateLenient(%q) = %+v, want an error", input, got)
			}
			if got != nil || readAs != "" {
				t.Errorf("ParseDateLenient(%q) = (%+v, %q) with an error, want (nil, \"\")", input, got, readAs)
			}
			if err.Error() != wantErr.Error() {
				t.Errorf("error = %q, want ParseDate's %q", err, wantErr)
			}
			if errors.Is(wantErr, ErrEmptyDate) != errors.Is(err, ErrEmptyDate) ||
				errors.Is(wantErr, ErrInvalidDate) != errors.Is(err, ErrInvalidDate) {
				t.Errorf("error %v does not wrap the sentinel ParseDate's does", err)
			}
		})
	}
}

// Strict parsing is unchanged: the converter and validator rely on ParseDate
// to tell whether a value is a GEDCOM date.
func TestParseDate_StillRejectsEnglish(t *testing.T) {
	for _, input := range []string{"June 1880", "12 December 1846", "about 1873", "abt. 1873", "between 1880 and 1885"} {
		if got, err := ParseDate(input); err == nil {
			t.Errorf("ParseDate(%q) = %+v, want an error", input, got)
		}
	}
}

// A person born "about 1873" is not presumed living once the date is read.
func TestIndividual_PresumedLiving_EnglishDate(t *testing.T) {
	date, _, err := ParseDateLenient("about 1873")
	if err != nil {
		t.Fatalf("ParseDateLenient error = %v", err)
	}
	ind := &Individual{Events: []*Event{{Type: EventBirth, Date: "about 1873", ParsedDate: date}}}
	living, reason := ind.PresumedLiving(nil, LivingConfig{ReferenceYear: 2026})
	if living || reason != LivingReasonOldEvent {
		t.Errorf("PresumedLiving = (%v, %q), want (false, %q)", living, reason, LivingReasonOldEvent)
	}
}
