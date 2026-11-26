package validator

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// invalidDateIssues decodes a 5.5.1 document whose INDI @I1@ carries body
// (level-1 lines and below) and returns the INVALID_DATE issues ValidateAll
// reports for it.
func invalidDateIssues(t *testing.T, body ...string) []Issue {
	t.Helper()
	lines := append([]string{
		"0 HEAD",
		"1 GEDC",
		"2 VERS 5.5.1",
		"1 CHAR UTF-8",
		"0 @I1@ INDI",
		"1 NAME John /Smith/",
	}, body...)
	lines = append(lines, "0 TRLR")
	doc, err := decoder.Decode(strings.NewReader(strings.Join(lines, "\n") + "\n"))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	return FilterByCode(New().ValidateAll(doc), CodeInvalidDate)
}

// TestInvalidDateReported pins issue #613: a date that does not exist on the
// calendar is reported, with the record, the tag holding the date and the
// DATE line, including when only the end of a range is impossible.
func TestInvalidDateReported(t *testing.T) {
	tests := []struct {
		name     string
		body     []string
		wantTag  string
		wantPath string
		wantLine int
	}{
		{"single date", []string{"1 BIRT", "2 DATE 31 FEB 1900"}, "BIRT", "BIRT.DATE", 8},
		{"range end", []string{"1 DEAT", "2 DATE BET 1900 AND 31 FEB 1901"}, "DEAT", "DEAT.DATE", 8},
		{"period end", []string{"1 RESI", "2 DATE FROM 1 JAN 1900 TO 30 FEB 1901"}, "RESI", "RESI.DATE", 8},
		{"non-leap 29 FEB", []string{"1 OCCU Smith", "2 DATE 29 FEB 1900"}, "OCCU", "OCCU.DATE", 8},
		{"LDS ordinance", []string{"1 BAPL", "2 DATE 31 APR 1900"}, "BAPL", "BAPL.DATE", 8},
		{"nested citation", []string{"1 BIRT", "2 SOUR @S1@", "3 DATA", "4 DATE 31 JUN 1900"}, "DATA", "BIRT.SOUR.DATA.DATE", 10},
		{"range end after dual-dated start", []string{"1 BIRT", "2 DATE BET 29 FEB 1723/24 AND 31 FEB 1725"}, "BIRT", "BIRT.DATE", 8},
		{"dual-dated 30 FEB", []string{"1 BIRT", "2 DATE 30 FEB 1723/24"}, "BIRT", "BIRT.DATE", 8},
		{"dual-dated 31 APR", []string{"1 BIRT", "2 DATE 31 APR 1723/24"}, "BIRT", "BIRT.DATE", 8},
		{"dual-dated range end", []string{"1 BIRT", "2 DATE BET 1 JAN 1723/24 AND 32 JAN 1723/24"}, "BIRT", "BIRT.DATE", 8},
		{"dual year not following", []string{"1 BIRT", "2 DATE 1750/53"}, "BIRT", "BIRT.DATE", 8},
		{"dual-dated day with dual year not following", []string{"1 BIRT", "2 DATE 29 FEB 1723/1722"}, "BIRT", "BIRT.DATE", 8},
		{"reversed range", []string{"1 BIRT", "2 DATE BET 1860 AND 1850"}, "BIRT", "BIRT.DATE", 8},
		{"reversed period", []string{"1 RESI", "2 DATE FROM 1860 TO 1850"}, "RESI", "RESI.DATE", 8},
		{"reversed range with dual-dated end", []string{"1 BIRT", "2 DATE BET 1 MAR 1724 AND 29 FEB 1723/24"}, "BIRT", "BIRT.DATE", 8},
		{"Julian 30 FEB", []string{"1 BIRT", "2 DATE @#DJULIAN@ 30 FEB 1700"}, "BIRT", "BIRT.DATE", 8},
		{"Julian dual-dated 30 FEB", []string{"1 BIRT", "2 DATE @#DJULIAN@ 30 FEB 1723/24"}, "BIRT", "BIRT.DATE", 8},
		{"Julian 29 FEB in a common year", []string{"1 BIRT", "2 DATE @#DJULIAN@ 29 FEB 1723"}, "BIRT", "BIRT.DATE", 8},
		{"Hebrew 30 Adar in a common year", []string{"1 BIRT", "2 DATE @#DHEBREW@ 30 ADR 5785"}, "BIRT", "BIRT.DATE", 8},
		{"Hebrew Adar II in a common year", []string{"1 BIRT", "2 DATE @#DHEBREW@ 1 ADS 5785"}, "BIRT", "BIRT.DATE", 8},
		{"French Republican 31 VEND", []string{"1 BIRT", "2 DATE @#DFRENCH R@ 31 VEND 8"}, "BIRT", "BIRT.DATE", 8},
		{"French Republican 6 COMP in a common year", []string{"1 BIRT", "2 DATE @#DFRENCH R@ 6 COMP 5"}, "BIRT", "BIRT.DATE", 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := invalidDateIssues(t, tt.body...)
			if len(got) != 1 {
				t.Fatalf("INVALID_DATE issues = %v, want exactly one", got)
			}
			issue := got[0]
			if issue.Severity != SeverityWarning {
				t.Errorf("Severity = %v, want WARNING", issue.Severity)
			}
			if issue.RecordXRef != "@I1@" {
				t.Errorf("RecordXRef = %q, want @I1@", issue.RecordXRef)
			}
			if issue.LineNumber != tt.wantLine {
				t.Errorf("LineNumber = %d, want %d", issue.LineNumber, tt.wantLine)
			}
			if issue.Details["tag"] != tt.wantTag || issue.Details["path"] != tt.wantPath {
				t.Errorf("Details = %v, want tag %q and path %q", issue.Details, tt.wantTag, tt.wantPath)
			}
		})
	}
}

// TestInvalidDateNotReported pins that valid dates on any calendar, and partial
// dates, which Date.Validate skips by design, raise no INVALID_DATE.
func TestInvalidDateNotReported(t *testing.T) {
	for _, date := range []string{
		"29 FEB 2000",
		"ABT 1900",
		"FEB 1900",
		"BET 1900 AND 1901",
		"@#DJULIAN@ 29 FEB 1900",
		"@#DJULIAN@ 29 FEB 1723/24",
		"@#DHEBREW@ 1 ADS 5784",
		"@#DHEBREW@ ADS 5785",
		"@#DFRENCH R@ 5 COMP 3",
		"@#DFRENCH R@ 6 COMP 3",
		"29 FEB 1723/24",
		"BET 1 JAN 1723/24 AND 29 FEB 1723/24",
		"BET JAN 1850 AND 1850",
		"BET 15 JAN 1850 AND JAN 1850",
		"1 JAN 1699/00",
		"(sometime in spring)",
	} {
		t.Run(date, func(t *testing.T) {
			if got := invalidDateIssues(t, "1 BIRT", "2 DATE "+date); len(got) != 0 {
				t.Errorf("INVALID_DATE issues = %v, want none", got)
			}
		})
	}
}

// TestUnparseableDateReported pins issue #966: with default options, a
// non-empty DATE that gedcom.ParseDate rejects is reported as INVALID_DATE,
// with the value as written, wherever the DATE sits; an empty DATE, which 7.0
// permits beside a PHRASE, is not.
func TestUnparseableDateReported(t *testing.T) {
	tests := []struct {
		name     string
		body     []string
		wantTag  string
		wantPath string
		wantLine int
		value    string
	}{
		{"garbage", []string{"1 BIRT", "2 DATE FOO BAR"}, "BIRT", "BIRT.DATE", 8, "FOO BAR"},
		{"BCE on Hebrew", []string{"1 BIRT", "2 DATE HEBREW 5000 BCE"}, "BIRT", "BIRT.DATE", 8, "HEBREW 5000 BCE"},
		{"day zero", []string{"1 DEAT", "2 DATE 0 JAN 1850"}, "DEAT", "DEAT.DATE", 8, "0 JAN 1850"},
		{"English", []string{"1 BIRT", "2 DATE June 1880"}, "BIRT", "BIRT.DATE", 8, "June 1880"},
		{"with PHRASE", []string{"1 BIRT", "2 DATE FOO", "3 PHRASE spring"}, "BIRT", "BIRT.DATE", 8, "FOO"},
		{"attribute", []string{"1 OCCU Smith", "2 DATE summer"}, "OCCU", "OCCU.DATE", 8, "summer"},
		{"nested citation", []string{"1 BIRT", "2 SOUR @S1@", "3 DATA", "4 DATE nope"}, "DATA", "BIRT.SOUR.DATA.DATE", 10, "nope"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := invalidDateIssues(t, tt.body...)
			if len(got) != 1 {
				t.Fatalf("INVALID_DATE issues = %v, want exactly one", got)
			}
			issue := got[0]
			if issue.Severity != SeverityWarning || issue.RecordXRef != "@I1@" || issue.LineNumber != tt.wantLine {
				t.Errorf("issue = %v, want a WARNING on @I1@ line %d", issue, tt.wantLine)
			}
			if issue.Details["tag"] != tt.wantTag || issue.Details["path"] != tt.wantPath || issue.Details["value"] != tt.value {
				t.Errorf("Details = %v, want tag %q, path %q, value %q", issue.Details, tt.wantTag, tt.wantPath, tt.value)
			}
			if !strings.Contains(issue.Message, "is not a GEDCOM date") {
				t.Errorf("Message = %q, want it to say the value is not a GEDCOM date", issue.Message)
			}
		})
	}
	for _, body := range [][]string{
		{"1 BIRT", "2 DATE"},
		{"1 BIRT", "2 DATE", "3 PHRASE spring"},
		// Extension calendars, months and epochs are dates (ADR 0014)
		{"1 BIRT", "2 DATE BET FRENCH_R 2 _JOUR 8 AND _CALENDRIER 4 COMP 8"},
		{"1 DEAT", "2 DATE _CALENDAR 8 _MONTH 190 _EPOCH"},
		{"1 BIRT", "2 DATE 31 FEB 2000 _AD"},
	} {
		if got := invalidDateIssues(t, body...); len(got) != 0 {
			t.Errorf("%v: INVALID_DATE issues = %v, want none", body, got)
		}
	}
}

// TestUnparseableDateTypedEntity pins that a record built in code with a Date
// string but no ParsedDate has the string checked, as the encoder writes it.
func TestUnparseableDateTypedEntity(t *testing.T) {
	ind := &gedcom.Individual{
		XRef: "@I1@",
		Events: []*gedcom.Event{
			{Type: gedcom.EventBirth, Date: "FOO BAR"},
			{Type: gedcom.EventDeath, Date: "31 FEB 1900"},
			{Type: gedcom.EventBurial, Date: "1 MAR 1900"},
			{Type: gedcom.EventChristening},
		},
	}
	doc := &gedcom.Document{
		Header:  &gedcom.Header{Date: "not a date"},
		Records: []*gedcom.Record{{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: ind}},
	}
	got := validateDates(doc)
	want := []struct{ xref, tag, value string }{
		{"", "HEAD", "not a date"}, {"@I1@", "BIRT", "FOO BAR"}, {"@I1@", "DEAT", "31 FEB 1900"},
	}
	if len(got) != len(want) {
		t.Fatalf("issues = %v, want %d", got, len(want))
	}
	for i, w := range want {
		if got[i].RecordXRef != w.xref || got[i].Details["tag"] != w.tag || got[i].Details["value"] != w.value {
			t.Errorf("issue %d = %v, want %s %s %q", i, got[i], w.xref, w.tag, w.value)
		}
	}
}

// TestDateIssuesAgreeAcrossValidators pins issue #966: Validate, ValidateAll
// and the StreamingValidator report the same INVALID_DATE issues, for a date
// that does not parse and for one that does not exist, on any record type.
func TestDateIssuesAgreeAcrossValidators(t *testing.T) {
	doc, err := decoder.Decode(strings.NewReader(strings.Join([]string{
		"0 HEAD",
		"1 GEDC",
		"2 VERS 5.5.1",
		"1 CHAR UTF-8",
		"0 @I1@ INDI",
		"1 BIRT",
		"2 DATE FOO BAR",
		"1 DEAT",
		"2 DATE 32 JAN 1900",
		"0 @F1@ FAM",
		"1 MARR",
		"2 DATE HEBREW 5000 BCE",
		"0 @S1@ SOUR",
		"1 DATA",
		"2 EVEN BIRT",
		"3 DATE whenever",
		"0 TRLR",
	}, "\n") + "\n"))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	key := func(i Issue) string {
		return fmt.Sprintf("%s line %d %s %q: %s", i.RecordXRef, i.LineNumber, i.Details["path"], i.Details["value"], i.Message)
	}
	keys := func(issues []Issue) []string {
		var out []string
		for _, i := range FilterByCode(issues, CodeInvalidDate) {
			out = append(out, key(i))
		}
		return out
	}

	all := keys(New().ValidateAll(doc))
	var viaValidate []Issue
	for _, err := range New().Validate(doc) {
		var issue *Issue
		if errors.As(err, &issue) {
			viaValidate = append(viaValidate, *issue)
		}
	}
	sv := NewStreamingValidator(StreamingOptions{})
	var streamed []Issue
	for _, record := range doc.Records {
		streamed = append(streamed, sv.ValidateRecord(record)...)
	}
	streamed = append(streamed, sv.Finalize()...)

	if len(all) != 4 {
		t.Fatalf("ValidateAll INVALID_DATE = %v, want 4", all)
	}
	if got := keys(viaValidate); !slices.Equal(got, all) {
		t.Errorf("Validate INVALID_DATE = %v, want %v", got, all)
	}
	if got := keys(streamed); !slices.Equal(got, all) {
		t.Errorf("StreamingValidator INVALID_DATE = %v, want %v", got, all)
	}
}

// TestInvalidDateHeader pins that HEAD.DATE is checked too, with no record XRef.
func TestInvalidDateHeader(t *testing.T) {
	doc, err := decoder.Decode(strings.NewReader(strings.Join([]string{
		"0 HEAD",
		"1 GEDC",
		"2 VERS 5.5.1",
		"1 CHAR UTF-8",
		"1 DATE 30 FEB 2020",
		"0 TRLR",
	}, "\n") + "\n"))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	got := FilterByCode(New().ValidateAll(doc), CodeInvalidDate)
	if len(got) != 1 || got[0].RecordXRef != "" || got[0].Details["path"] != "DATE" || got[0].LineNumber != 5 {
		t.Errorf("INVALID_DATE issues = %v, want one for HEAD.DATE on line 5", got)
	}
}

// TestInvalidDateTypedEntity pins that a record built in code, with a typed
// entity and no raw Tags, has its ParsedDates checked.
func TestInvalidDateTypedEntity(t *testing.T) {
	mustParse := func(s string) *gedcom.Date {
		d, err := gedcom.ParseDate(s)
		if err != nil {
			t.Fatalf("ParseDate(%q) error = %v", s, err)
		}
		return d
	}
	ind := &gedcom.Individual{
		XRef:   "@I1@",
		Events: []*gedcom.Event{{Type: gedcom.EventBirth, Date: "31 FEB 1900", ParsedDate: mustParse("31 FEB 1900")}},
		Attributes: []*gedcom.Attribute{
			{Type: gedcom.AttributeOccupation, Date: "29 FEB 2000", ParsedDate: mustParse("29 FEB 2000")},
			{Type: gedcom.AttributeOccupation, Date: "31 JUN 1900", ParsedDate: mustParse("31 JUN 1900")},
		},
		LDSOrdinances: []*gedcom.LDSOrdinance{{Type: gedcom.LDSBaptism, Date: "BET 1900 AND 31 APR 1901", ParsedDate: mustParse("BET 1900 AND 31 APR 1901")}},
	}
	fam := &gedcom.Family{
		XRef:          "@F1@",
		Events:        []*gedcom.Event{{Type: gedcom.EventMarriage, Date: "31 NOV 1920", ParsedDate: mustParse("31 NOV 1920")}},
		Attributes:    []*gedcom.Attribute{{Type: gedcom.AttributeCaste, Date: "30 FEB 1921", ParsedDate: mustParse("30 FEB 1921")}},
		LDSOrdinances: []*gedcom.LDSOrdinance{{Type: gedcom.LDSSealingSpouse, Date: "31 SEP 1922", ParsedDate: mustParse("31 SEP 1922")}},
	}
	doc := &gedcom.Document{
		Header: &gedcom.Header{Date: "31 SEP 2020"},
		Records: []*gedcom.Record{
			{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: ind},
			{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: fam},
			nil,
		},
	}
	got := validateDates(doc)
	want := []struct{ xref, tag string }{
		{"", "HEAD"}, {"@I1@", "BIRT"}, {"@I1@", "OCCU"}, {"@I1@", "BAPL"},
		{"@F1@", "MARR"}, {"@F1@", "CAST"}, {"@F1@", "SLGS"},
	}
	if len(got) != len(want) {
		t.Fatalf("issues = %v, want %d", got, len(want))
	}
	for i, w := range want {
		wantPath := w.tag + ".DATE"
		if w.tag == "HEAD" {
			wantPath = "DATE"
		}
		if got[i].RecordXRef != w.xref || got[i].Details["tag"] != w.tag ||
			got[i].Details["path"] != wantPath || got[i].LineNumber != 0 {
			t.Errorf("issue %d = %v, want %s %s at path %s, line 0", i, got[i], w.xref, w.tag, wantPath)
		}
	}
}

// TestDateErrorReversedRangeNoOriginal pins that a hand-built reversed range
// with no Original is reported with the same message gedcom.Date.Validate
// gives, not with an empty quoted value.
func TestDateErrorReversedRangeNoOriginal(t *testing.T) {
	date := &gedcom.Date{
		Modifier: gedcom.ModifierBetween, Year: 1860,
		EndDate: &gedcom.Date{Year: 1850},
	}
	want := "invalid date range: the end date is before the start date"
	if err := date.Validate(); err == nil || err.Error() != want {
		t.Fatalf("Validate() error = %v, want %q", err, want)
	}
	if err := dateError(date); err == nil || err.Error() != want {
		t.Errorf("dateError() error = %v, want %q", err, want)
	}
}
