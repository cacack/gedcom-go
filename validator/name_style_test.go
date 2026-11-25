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
		{"GIVN mismatch", []string{"1 NAME John /Smith/", "2 GIVN Jon"}, CodeNamePartMismatch, "GIVN"},
		{"SURN mismatch", []string{"1 NAME John /Smith/", "2 SURN Smyth"}, CodeNamePartMismatch, "SURN"},
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
			if got.Severity != SeverityWarning {
				t.Errorf("severity = %s, want WARNING", got.Severity)
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
		"2 GIVN Jon",
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
// once each, with a description and SeverityWarning, and is a copy.
func TestAllStyleChecks(t *testing.T) {
	want := []string{
		CodeNameAllCaps, CodeNameTextAfterSurname, CodeNameSpacing, CodeNameQuotesOrParens,
		CodeNameUncertainMark, CodeNamePartPunctuation, CodeNameVendorTag, CodeNamePartMismatch,
	}
	checks := AllStyleChecks()
	if got := allStyleCodes(); !reflect.DeepEqual(got, want) {
		t.Errorf("codes = %v, want %v", got, want)
	}
	for _, check := range checks {
		if check.Description == "" {
			t.Errorf("%s has no description", check.Code)
		}
		if check.Severity != SeverityWarning {
			t.Errorf("%s has severity %s, want WARNING", check.Code, check.Severity)
		}
	}
	checks[0].Code = "CHANGED"
	if AllStyleChecks()[0].Code != CodeNameAllCaps {
		t.Error("AllStyleChecks returned the package's own slice")
	}
}

// TestNamePartMismatch pins which GIVN and SURN lines NAME_PART_MISMATCH
// reports (spec-interpretation policy SI-022): the disagreeing tags under
// Details["tags"], each one's text and the value's, and nothing for a piece
// that agrees once NPFX, SPFX, NSFX, repeated lines, spacing, commas, case,
// quoted or parenthesized text and a value with no slashes are allowed for,
// or for a NAME with no GIVN or SURN.
func TestNamePartMismatch(t *testing.T) {
	tests := []struct {
		name string
		body []string
		want map[string]string // nil: not reported
	}{
		{"GIVN disagrees", []string{"1 NAME John /Smith/", "2 GIVN Jon", "2 SURN Smith"},
			map[string]string{"tags": "GIVN", "GIVN": "Jon", "given": "John"}},
		{"SURN disagrees", []string{"1 NAME John /Smith/", "2 GIVN John", "2 SURN Smyth"},
			map[string]string{"tags": "SURN", "SURN": "Smyth", "surname": "Smith"}},
		{"both disagree", []string{"1 NAME John /Smith/", "2 SURN Smyth", "2 GIVN Jon"},
			map[string]string{"tags": "GIVN,SURN", "GIVN": "Jon", "given": "John", "SURN": "Smyth", "surname": "Smith"}},
		{"GIVN missing a word", []string{"1 NAME John Paul /Smith/", "2 GIVN John"},
			map[string]string{"tags": "GIVN", "GIVN": "John", "given": "John Paul"}},
		{"no slashes", []string{"1 NAME John Smith", "2 GIVN John", "2 SURN Smith"}, nil},
		{"no slashes, GIVN disagrees", []string{"1 NAME John Smith", "2 GIVN Jon", "2 SURN Smith"},
			map[string]string{"tags": "GIVN", "GIVN": "Jon", "given": "John Smith"}},
		{"no slashes, SURN words out of order", []string{"1 NAME Ana Garcia Lopez", "2 SURN Lopez Garcia"},
			map[string]string{"tags": "SURN", "SURN": "Lopez Garcia", "surname": ""}},
		{"quoted nickname", []string{`1 NAME John "Jack" /Smith/`, "2 GIVN John", "2 NICK Jack"}, nil},
		{"parenthesized nickname", []string{"1 NAME John (Big Jim) /Smith/", "2 GIVN John"}, nil},
		{"curly-quoted nickname in GIVN", []string{"1 NAME John /Smith/", "2 GIVN John “Jack”"}, nil},
		{"unclosed quote", []string{`1 NAME John "Jack /Smith/`, "2 GIVN John"},
			map[string]string{"tags": "GIVN", "GIVN": "John", "given": `John "Jack`}},
		{"NPFX not in the value", []string{"1 NAME John /Smith/", "2 NPFX Dr.", "2 GIVN Dr. John"},
			map[string]string{"tags": "GIVN", "GIVN": "Dr. John", "given": "John"}},
		{"NPFX in the value", []string{"1 NAME Dr. John /Smith/", "2 NPFX Dr.", "2 GIVN John"}, nil},
		{"NPFX in the value and GIVN", []string{"1 NAME Dr. John /Smith/", "2 NPFX Dr.", "2 GIVN Dr. John"}, nil},
		{"SPFX in the value", []string{"1 NAME John /van der Berg/", "2 SPFX van der", "2 SURN Berg"}, nil},
		{"SPFX in the value and SURN", []string{"1 NAME John /van der Berg/", "2 SPFX van der", "2 SURN van der Berg"}, nil},
		{"NSFX in the given name", []string{"1 NAME John Jr", "2 GIVN John", "2 NSFX Jr"}, nil},
		{"NSFX after the surname", []string{"1 NAME John /Smith/ Jr", "2 GIVN John", "2 NSFX Jr"}, nil},
		{"several GIVN", []string{"1 NAME John Paul /Smith/", "2 GIVN John", "2 GIVN Paul"}, nil},
		{"several GIVN out of order", []string{"1 NAME John Paul /Smith/", "2 GIVN Paul", "2 GIVN John"},
			map[string]string{"tags": "GIVN", "GIVN": "Paul, John", "given": "John Paul"}},
		{"comma-separated GIVN", []string{"1 NAME John Paul /Smith/", "2 GIVN John, Paul"}, nil},
		{"several SURN", []string{"1 NAME Ana /Garcia Lopez/", "2 SURN Garcia", "2 SURN Lopez"}, nil},
		{"case and spacing", []string{"1 NAME John  Paul /SMITH/", "2 GIVN john paul", "2 SURN Smith"}, nil},
		{"no GIVN or SURN", []string{"1 NAME John /Smith/", "2 NPFX Dr.", "2 NICK Jack"}, nil},
		{"pieces of another NAME", []string{"1 NAME John /Smith/", "1 NAME Jon /Smyth/", "2 GIVN Jon", "2 SURN Smyth"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := styleIssues(styleDoc(t, tt.body...), CodeNamePartMismatch)
			if tt.want == nil {
				if len(issues) != 0 {
					t.Errorf("got %v, want none", issues)
				}
				return
			}
			if len(issues) != 1 {
				t.Fatalf("got %d issues, want 1: %v", len(issues), issues)
			}
			got := issues[0].Details
			for key, value := range tt.want {
				if got[key] != value {
					t.Errorf("Details[%s] = %q, want %q", key, got[key], value)
				}
			}
			if len(got) != len(tt.want)+1 || got["value"] == "" {
				t.Errorf("Details = %v, want %v and value", got, tt.want)
			}
		})
	}
}

// TestHasEdgePunctuation pins the test the NAME_PART_PUNCTUATION check
// applies to an NSFX or NICK value, an empty one included.
func TestHasEdgePunctuation(t *testing.T) {
	for value, want := range map[string]bool{"": false, "Jr": false, "Jr.": true, ", Jr": true, "Jr ": true, "(Jack)": true} {
		if got := hasEdgePunctuation(value); got != want {
			t.Errorf("hasEdgePunctuation(%q) = %v, want %v", value, got, want)
		}
	}
}

// TestNameStyleStrictness pins that every listed style check, NAME_ALL_CAPS
// included, reports at the default StrictnessNormal, and that
// StrictnessRelaxed, which keeps errors only, filters them.
func TestNameStyleStrictness(t *testing.T) {
	doc := styleDoc(t, `1 NAME John "Jack" /SMITH/`)
	codes := []string{CodeNameAllCaps, CodeNameQuotesOrParens}
	for _, tt := range []struct {
		strictness Strictness
		want       []string
	}{
		{StrictnessRelaxed, nil},
		{StrictnessNormal, []string{CodeNameAllCaps, CodeNameQuotesOrParens}},
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
