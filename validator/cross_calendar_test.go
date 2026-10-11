package validator

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Regression tests for #618: the age checks and duplicate detection compare
// dates in different calendars on one scale.

func crossCalendarIndividual(t *testing.T, xref, birth, death string) *gedcom.Individual {
	t.Helper()
	ind := &gedcom.Individual{
		XRef:  xref,
		Names: []*gedcom.PersonalName{{Full: "Sarah /Levi/"}},
		Sex:   "F",
	}
	for _, ev := range []struct {
		typ  gedcom.EventType
		date string
	}{{gedcom.EventBirth, birth}, {gedcom.EventDeath, death}} {
		if ev.date == "" {
			continue
		}
		parsed, err := gedcom.ParseDate(ev.date)
		if err != nil {
			t.Fatalf("ParseDate(%q): %v", ev.date, err)
		}
		ind.Events = append(ind.Events, &gedcom.Event{Type: ev.typ, Date: ev.date, ParsedDate: parsed})
	}
	return ind
}

func TestCheckReasonableAge_CrossCalendar(t *testing.T) {
	v := NewDateLogicValidator(nil)
	tests := []struct {
		name      string
		birth     string
		death     string
		wantIssue bool
	}{
		{"Hebrew birth, Gregorian death", "@#DHEBREW@ 1 TSH 5660", "1 JAN 1950", false},
		{"Hebrew year-only birth, Gregorian death", "@#DHEBREW@ 5710", "1950", false},
		{"Julian birth, Gregorian death", "@#DJULIAN@ 1 MAR 1700", "5 MAR 1750", false},
		{"French Republican birth, Gregorian death", "@#DFRENCH R@ 1 VEND 1", "1 JAN 1850", false},
		{"Hebrew birth 150 years before Gregorian death", "@#DHEBREW@ 1 TSH 5660", "1 JAN 2050", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := v.checkReasonableAge(crossCalendarIndividual(t, "@I1@", tt.birth, tt.death))
			if tt.wantIssue != (issue != nil) {
				t.Errorf("checkReasonableAge(%q, %q) = %v, want issue %v", tt.birth, tt.death, issue, tt.wantIssue)
			}
		})
	}
}

func TestFindDuplicates_CrossCalendarBirthYears(t *testing.T) {
	tests := []struct {
		name         string
		date1, date2 string
		wantReason   string // the birth-year reason expected; "" means none
	}{
		{"Hebrew and Gregorian with close year numbers", "@#DHEBREW@ 1900", "1900", ""},
		{"Hebrew and Gregorian in the same year", "@#DHEBREW@ 1 TSH 5660", "1899", "same birth year"},
		{"Julian late December is the next Gregorian year", "@#DJULIAN@ 25 DEC 1700", "1701", "same birth year"},
		{"two Gregorian dates as before", "1900", "1901", "birth year within 1 years"},
		{"two Hebrew dates compare their own years", "@#DHEBREW@ 5660", "@#DHEBREW@ 5661", "birth year within 1 years"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ind1 := crossCalendarIndividual(t, "@I1@", tt.date1, "")
			ind2 := crossCalendarIndividual(t, "@I2@", tt.date2, "")
			doc := &gedcom.Document{
				Records: []*gedcom.Record{
					{XRef: ind1.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind1},
					{XRef: ind2.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind2},
				},
			}
			duplicates := NewDuplicateDetector(nil).FindDuplicates(doc)
			if len(duplicates) != 1 {
				t.Fatalf("Expected 1 duplicate pair, got %d", len(duplicates))
			}
			gotReason := ""
			for _, reason := range duplicates[0].MatchReasons {
				if reason == "same birth year" || strings.HasPrefix(reason, "birth year within") {
					gotReason = reason
				}
			}
			if gotReason != tt.wantReason {
				t.Errorf("birth-year reason = %q, want %q (reasons %v)", gotReason, tt.wantReason, duplicates[0].MatchReasons)
			}
		})
	}
}

func TestBirthYearsApart_Unconvertible(t *testing.T) {
	unknown := &gedcom.Date{Year: 1900, Calendar: gedcom.Calendar(99)}
	if _, ok := birthYearsApart(unknown, &gedcom.Date{Year: 1900}); ok {
		t.Error("birthYearsApart(unknown calendar, Gregorian) ok = true, want false")
	}
}

func TestBirthYearsApart_Years(t *testing.T) {
	tests := []struct {
		name         string
		date1, date2 string
		want         int
	}{
		{"Hebrew year-only placed on 1 Tishrei", "@#DHEBREW@ 5660", "1899", 0},
		{"Julian late December is the next Gregorian year", "@#DJULIAN@ 25 DEC 1700", "1701", 0},
		{"Julian dual date read in New Style", "@#DJULIAN@ 11 FEB 1731/32", "1732", 0},
		{"Gregorian dual date read in New Style against Julian", "11 FEB 1731/32", "@#DJULIAN@ 1 JAN 1732", 0},
		{"same-calendar dual date compares Year as written", "11 FEB 1731/32", "1732", 1},
		{"modifier is ignored", "BEF @#DHEBREW@ 5660", "1899", 0},
		{"range compares its start", "BET @#DHEBREW@ 5660 AND @#DHEBREW@ 5670", "1899", 0},
		{"BCE Julian against Gregorian", "@#DJULIAN@ 15 MAR 44 B.C.", "44 B.C.", 0},
		{"BCE and CE across calendars", "@#DJULIAN@ 15 MAR 44 B.C.", "44", 87},
		{"invalid day placed on its month", "31 FEB 1850", "@#DJULIAN@ 1850", 0},
		{"invalid Hebrew month placed on its year", "@#DHEBREW@ 1 ADS 5661", "1900", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d1, err := gedcom.ParseDate(tt.date1)
			if err != nil {
				t.Fatalf("ParseDate(%q): %v", tt.date1, err)
			}
			d2, err := gedcom.ParseDate(tt.date2)
			if err != nil {
				t.Fatalf("ParseDate(%q): %v", tt.date2, err)
			}
			for _, pair := range [][2]*gedcom.Date{{d1, d2}, {d2, d1}} {
				got, ok := birthYearsApart(pair[0], pair[1])
				if !ok || got != tt.want {
					t.Errorf("birthYearsApart(%q, %q) = %d, %v; want %d, true", pair[0].Original, pair[1].Original, got, ok, tt.want)
				}
			}
		})
	}
}

// A range end that cannot be placed does not discard the start's year.
func TestBirthYearsApart_UnplaceableRangeEnd(t *testing.T) {
	start, err := gedcom.ParseDate("@#DHEBREW@ 5660")
	if err != nil {
		t.Fatal(err)
	}
	start.Modifier = gedcom.ModifierBetween
	start.EndDate = &gedcom.Date{Year: 5670, Calendar: gedcom.Calendar(99)}
	if got, ok := birthYearsApart(start, &gedcom.Date{Year: 1899}); !ok || got != 0 {
		t.Errorf("birthYearsApart(range with unplaceable end, 1899) = %d, %v; want 0, true", got, ok)
	}
}
