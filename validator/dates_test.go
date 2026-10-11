package validator

import (
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

// TestInvalidDateNotReported pins that valid, partial and non-Gregorian dates,
// which Date.Validate skips by design, raise no INVALID_DATE.
func TestInvalidDateNotReported(t *testing.T) {
	for _, date := range []string{
		"29 FEB 2000",
		"ABT 1900",
		"FEB 1900",
		"BET 1900 AND 1901",
		"@#DJULIAN@ 31 FEB 1900",
		"29 FEB 1723/24",
		"BET 1 JAN 1723/24 AND 29 FEB 1723/24",
		"1 JAN 1699/00",
		"(sometime in spring)",
		"not a date at all",
	} {
		t.Run(date, func(t *testing.T) {
			if got := invalidDateIssues(t, "1 BIRT", "2 DATE "+date); len(got) != 0 {
				t.Errorf("INVALID_DATE issues = %v, want none", got)
			}
		})
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
