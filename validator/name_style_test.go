package validator

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// styleDoc decodes a 5.5.1 document whose INDI @I1@ carries body (level-1
// lines and below). The INDI is on line 5, so body starts on line 6.
func styleDoc(t *testing.T, body ...string) *gedcom.Document {
	t.Helper()
	lines := append([]string{
		"0 HEAD",
		"1 GEDC",
		"2 VERS 5.5.1",
		"1 CHAR UTF-8",
		"0 @I1@ INDI",
	}, body...)
	lines = append(lines, "0 TRLR")
	doc, err := decoder.Decode(strings.NewReader(strings.Join(lines, "\n") + "\n"))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	return doc
}

// allStyleCodes returns every style check's code.
func allStyleCodes() []string {
	var codes []string
	for _, check := range AllStyleChecks() {
		codes = append(codes, check.Code)
	}
	return codes
}

// styleIssues returns the issues ValidateAll reports for doc with the given
// style checks on, at StrictnessStrict, keeping only style codes.
func styleIssues(doc *gedcom.Document, codes ...string) []Issue {
	v := NewWithOptions(&ValidateOptions{Strictness: StrictnessStrict, StyleChecks: codes})
	isStyle := make(map[string]bool)
	for _, code := range allStyleCodes() {
		isStyle[code] = true
	}
	var issues []Issue
	for _, issue := range v.ValidateAll(doc) {
		if isStyle[issue.Code] {
			issues = append(issues, issue)
		}
	}
	return issues
}

// TestNameStyleEachCode pins each style check: with every check on, the
// pattern is reported once on its NAME line, under its own code, with the
// record xref and the NAME line number.
func TestNameStyleEachCode(t *testing.T) {
	tests := []struct {
		name     string
		body     []string
		wantCode string
		wantTags string
	}{
		{"surname in capitals", []string{"1 NAME John /SMITH/"}, CodeNameAllCaps, ""},
		{"given in capitals", []string{"1 NAME JOHN /Smith/"}, CodeNameAllCaps, ""},
		{"capital word after apostrophe", []string{"1 NAME Sean /O'BRIEN/"}, CodeNameAllCaps, ""},
		{"text after surname", []string{"1 NAME John /Smith/ Jr"}, CodeNameTextAfterSurname, ""},
		{"leading space", []string{"1 NAME  John /Smith/"}, CodeNameSpacing, ""},
		{"trailing space", []string{"1 NAME John /Smith/ "}, CodeNameSpacing, ""},
		{"doubled space", []string{"1 NAME John  Paul /Smith/"}, CodeNameSpacing, ""},
		{"space inside slashes", []string{"1 NAME John / Smith/"}, CodeNameSpacing, ""},
		{"quoted nickname", []string{`1 NAME John "Jack" /Smith/`}, CodeNameQuotesOrParens, ""},
		{"curly quotes", []string{"1 NAME John “Jack” /Smith/"}, CodeNameQuotesOrParens, ""},
		{"parentheses", []string{"1 NAME Mary /Smith/", "1 NAME Mary (Polly) /Jones/"}, CodeNameQuotesOrParens, ""},
		{"question mark", []string{"1 NAME John /Smith?/"}, CodeNameUncertainMark, ""},
		{"asterisk", []string{"1 NAME *John /Smith/"}, CodeNameUncertainMark, ""},
		{"digit", []string{"1 NAME John2 /Smith/"}, CodeNameUncertainMark, ""},
		{"or between alternatives", []string{"1 NAME John /Smith or Smyth/"}, CodeNameUncertainMark, ""},
		{"NSFX with trailing period", []string{"1 NAME John /Smith/", "2 NSFX Jr."}, CodeNamePartPunctuation, "NSFX"},
		{"NSFX with leading comma", []string{"1 NAME John /Smith/", "2 NSFX , Jr"}, CodeNamePartPunctuation, "NSFX"},
		{"NICK and NSFX", []string{"1 NAME John /Smith/", "2 NICK (Jack)", "2 NSFX Jr "}, CodeNamePartPunctuation, "NICK,NSFX"},
		{"vendor tag", []string{"1 NAME John /Smith/", "2 _MARNM Jones", "2 _AKA Jack"}, CodeNameVendorTag, "_MARNM,_AKA"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := styleDoc(t, tt.body...)
			// The offending NAME is the last level-1 NAME in body.
			wantLine := 0
			for i, line := range tt.body {
				if strings.HasPrefix(line, "1 NAME") {
					wantLine = 6 + i
				}
			}
			issues := styleIssues(doc, allStyleCodes()...)
			if len(issues) != 1 {
				t.Fatalf("got %d style issues, want 1: %v", len(issues), issues)
			}
			got := issues[0]
			if got.Code != tt.wantCode || got.RecordXRef != "@I1@" || got.LineNumber != wantLine {
				t.Errorf("got %s %s line %d, want %s @I1@ line %d", got.Code, got.RecordXRef, got.LineNumber, tt.wantCode, wantLine)
			}
			if got.Details["tags"] != tt.wantTags {
				t.Errorf("Details[tags] = %q, want %q", got.Details["tags"], tt.wantTags)
			}
			if got.Severity == SeverityError {
				t.Errorf("severity = ERROR, want WARNING or INFO")
			}
		})
	}
}

// TestNameStyleOncePerLine pins that a pattern found several times on one
// NAME line is reported once, and that each check on a line reports
// separately.
func TestNameStyleOncePerLine(t *testing.T) {
	doc := styleDoc(t,
		"1 NAME JOHN /SMITH?/ Jr?",
		"2 _A x",
		"2 _B y",
	)
	issues := styleIssues(doc, allStyleCodes()...)
	var codes []string
	for _, issue := range issues {
		codes = append(codes, issue.Code)
		if issue.LineNumber != 6 {
			t.Errorf("%s line = %d, want 6", issue.Code, issue.LineNumber)
		}
	}
	want := []string{CodeNameAllCaps, CodeNameTextAfterSurname, CodeNameUncertainMark, CodeNameVendorTag}
	if !reflect.DeepEqual(codes, want) {
		t.Errorf("codes = %v, want %v", codes, want)
	}
}

// TestNameStyleOneAtATime pins that each check runs alone when only its code
// is listed, and that an unknown code is ignored.
func TestNameStyleOneAtATime(t *testing.T) {
	doc := styleDoc(t,
		`1 NAME  JOHN "Jack" /Smith or Smyth/ Jr`,
		"2 NSFX Jr.",
		"2 _AKA Jack",
	)
	for _, code := range allStyleCodes() {
		issues := styleIssues(doc, code, "NOT_A_STYLE_CHECK")
		if len(issues) != 1 || issues[0].Code != code {
			t.Errorf("StyleChecks{%s}: got %v, want one %s", code, issues, code)
		}
	}
	if issues := styleIssues(doc, "NOT_A_STYLE_CHECK"); len(issues) != 0 {
		t.Errorf("unknown code alone: got %v, want none", issues)
	}
}

// TestNameStyleUnsetUnchanged pins that with StyleChecks unset, ValidateAll
// reports exactly what it does without the option.
func TestNameStyleUnsetUnchanged(t *testing.T) {
	doc := styleDoc(t, `1 NAME  JOHN "Jack" /SMITH?/ Jr`, "2 _AKA Jack")
	for _, opts := range []*ValidateOptions{nil, {Strictness: StrictnessStrict}, {StyleChecks: []string{}}} {
		got := NewWithOptions(opts).ValidateAll(doc)
		want := NewWithOptions(nil).ValidateAll(doc)
		if opts != nil && opts.Strictness == StrictnessStrict {
			want = NewWithOptions(&ValidateOptions{Strictness: StrictnessStrict}).ValidateAll(doc)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("opts %+v: got %v, want %v", opts, got, want)
		}
		for _, issue := range got {
			for _, code := range allStyleCodes() {
				if issue.Code == code {
					t.Errorf("opts %+v: style issue %v reported with StyleChecks unset", opts, issue)
				}
			}
		}
	}
}

// TestNameStyleCleanNames pins that mixed-case "Given /Surname/" names with
// clean parts give no style issues, including initials and particles.
func TestNameStyleCleanNames(t *testing.T) {
	doc := styleDoc(t,
		"1 NAME John /Smith/",
		"2 GIVN John",
		"2 SURN Smith",
		"2 NSFX Jr",
		"2 NICK Jack",
		"1 NAME J. R. /McDonald/",
		"1 NAME Seán /O'Brien/",
		"1 NAME Anna /van der Berg/",
		"1 NAME Mary-Ann /Smith-Jones/",
		"1 NAME Oren /Ben Or/",
		"1 NAME /Smith/",
		"1 NAME John",
	)
	if issues := styleIssues(doc, allStyleCodes()...); len(issues) != 0 {
		t.Errorf("got %v, want no style issues", issues)
	}
}

// TestNameStyleScope pins what the checks read: only level-1 NAME lines of
// INDI records with raw Tags, and only tags directly under the NAME.
func TestNameStyleScope(t *testing.T) {
	doc := styleDoc(t,
		"1 NAME John /Smith/",
		"2 SOUR @S1@",
		"3 _APID 1,2",
		"1 _AKA JOHN /SMITH/",
		"1 EVEN",
		"2 _X y",
	)
	doc.Records = append(doc.Records,
		&gedcom.Record{XRef: "@R1@", Type: gedcom.RecordTypeRepository, Tags: []*gedcom.Tag{{Level: 1, Tag: "NAME", Value: "SMITH LIBRARY"}}},
		&gedcom.Record{XRef: "@I2@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: "@I2@"}},
		nil,
	)
	indi := doc.Records[0]
	indi.Tags = append([]*gedcom.Tag{indi.Tags[0], nil}, indi.Tags[1:]...)
	if issues := styleIssues(doc, allStyleCodes()...); len(issues) != 0 {
		t.Errorf("got %v, want no style issues", issues)
	}
}

// TestAllStyleChecks pins that the exported list covers every style code,
// once each, with a description and a non-error severity, and is a copy.
func TestAllStyleChecks(t *testing.T) {
	want := []string{
		CodeNameAllCaps, CodeNameTextAfterSurname, CodeNameSpacing, CodeNameQuotesOrParens,
		CodeNameUncertainMark, CodeNamePartPunctuation, CodeNameVendorTag,
	}
	checks := AllStyleChecks()
	if got := allStyleCodes(); !reflect.DeepEqual(got, want) {
		t.Errorf("codes = %v, want %v", got, want)
	}
	for _, check := range checks {
		if check.Description == "" {
			t.Errorf("%s has no description", check.Code)
		}
		if check.Severity == SeverityError {
			t.Errorf("%s has error severity", check.Code)
		}
	}
	checks[0].Code = "CHANGED"
	if AllStyleChecks()[0].Code != CodeNameAllCaps {
		t.Error("AllStyleChecks returned the package's own slice")
	}
}

// TestNameStyleUnknownCodeFinding covers the guard in nameLine.finding for a
// code it does not know.
func TestNameStyleUnknownCodeFinding(t *testing.T) {
	line := nameLine{value: "JOHN /SMITH/"}
	if _, found := line.finding("NOT_A_STYLE_CHECK"); found {
		t.Error("finding(unknown) = true, want false")
	}
	if hasEdgePunctuation("") {
		t.Error(`hasEdgePunctuation("") = true, want false`)
	}
}

// TestNameStyleStrictness pins that Strictness filters style issues like any
// other: at the default StrictnessNormal a listed Warning check reports and a
// listed Info check does not; at StrictnessStrict both do.
func TestNameStyleStrictness(t *testing.T) {
	doc := styleDoc(t, `1 NAME John "Jack" /SMITH/`)
	codes := []string{CodeNameAllCaps, CodeNameQuotesOrParens}
	for _, tt := range []struct {
		strictness Strictness
		want       []string
	}{
		{StrictnessNormal, []string{CodeNameQuotesOrParens}},
		{StrictnessStrict, []string{CodeNameAllCaps, CodeNameQuotesOrParens}},
	} {
		var got []string
		for _, issue := range NewWithOptions(&ValidateOptions{Strictness: tt.strictness, StyleChecks: codes}).ValidateAll(doc) {
			if slices.Contains(codes, issue.Code) {
				got = append(got, issue.Code)
			}
		}
		if !slices.Equal(got, tt.want) {
			t.Errorf("Strictness %v: got %v, want %v", tt.strictness, got, tt.want)
		}
	}
}
