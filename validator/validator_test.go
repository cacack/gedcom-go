package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// structureCodes are the three codes the record-structure checks report. The
// tests below that exercise those checks on 5.5 fixtures without a SUBM ignore
// every other code (notably MISSING_SUBM) rather than build full headers.
var structureCodes = map[string]bool{
	CodeBrokenXRef:           true,
	CodeMissingRequiredField: true,
	CodeEmptyFamily:          true,
}

// issuesFromErrors unwraps each Validate element to its *Issue, failing the
// test on any element that is not one.
func issuesFromErrors(t *testing.T, errs []error) []*Issue {
	t.Helper()
	out := make([]*Issue, 0, len(errs))
	for _, e := range errs {
		var issue *Issue
		if !errors.As(e, &issue) {
			t.Fatalf("Validate element %T (%v) is not a *Issue", e, e)
		}
		out = append(out, issue)
	}
	return out
}

// structureIssues returns the Validate results that carry a structure code.
func structureIssues(t *testing.T, errs []error) []*Issue {
	t.Helper()
	var out []*Issue
	for _, issue := range issuesFromErrors(t, errs) {
		if structureCodes[issue.Code] {
			out = append(out, issue)
		}
	}
	return out
}

func TestValidateBrokenXRef(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John Smith
1 ASSO @I999@
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	got := structureIssues(t, New().Validate(doc))
	if len(got) != 1 {
		t.Fatalf("structure issues = %v, want exactly one BROKEN_XREF", got)
	}
	issue := got[0]
	if issue.Code != CodeBrokenXRef || issue.Severity != SeverityError {
		t.Errorf("issue = %v, want SeverityError %s", issue, CodeBrokenXRef)
	}
	if issue.LineNumber != 6 {
		t.Errorf("LineNumber = %d, want 6 (the ASSO line)", issue.LineNumber)
	}
	if issue.RecordXRef != "@I1@" || issue.RelatedXRef != "@I999@" {
		t.Errorf("RecordXRef/RelatedXRef = %q/%q, want @I1@/@I999@", issue.RecordXRef, issue.RelatedXRef)
	}
	if issue.Details["tag"] != "ASSO" {
		t.Errorf("Details[tag] = %q, want ASSO", issue.Details["tag"])
	}
	// ADR 0007: an issue carrying both an XRef and a line prints both.
	if s := issue.Error(); !strings.Contains(s, "@I1@") || !strings.Contains(s, "[line 6]") {
		t.Errorf("Error() = %q, want both the XRef and the line", s)
	}
}

// TestValidateBrokenXRef_NotDoubleReported pins that a pointer the typed
// reference checks cover is reported once, under its ORPHANED_* code, and not
// again as BROKEN_XREF; while the same pointer on a record without a typed
// entity -- which the typed checks cannot see -- is reported as BROKEN_XREF.
func TestValidateBrokenXRef_NotDoubleReported(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John Smith
1 FAMS @F999@
1 FAMC @F998@
1 SOUR @S999@
0 @F1@ FAM
1 HUSB @I998@
1 WIFE @I997@
1 CHIL @I996@
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	all := New().ValidateAll(doc)
	if n := len(FilterByCode(all, CodeBrokenXRef)); n != 0 {
		t.Errorf("BROKEN_XREF count = %d, want 0 (typed ORPHANED_* checks cover these)", n)
	}
	for _, code := range []string{CodeOrphanedFAMS, CodeOrphanedFAMC, CodeOrphanedSOUR, CodeOrphanedHUSB, CodeOrphanedWIFE, CodeOrphanedCHIL} {
		if n := len(FilterByCode(all, code)); n != 1 {
			t.Errorf("%s count = %d, want 1", code, n)
		}
	}

	// Strip the typed entities: the same raw pointers are now uncovered.
	for _, r := range doc.Records {
		r.Entity = nil
	}
	broken := FilterByCode(New().ValidateAll(doc), CodeBrokenXRef)
	if len(broken) != 6 {
		t.Errorf("BROKEN_XREF count without entities = %d, want 6: %v", len(broken), broken)
	}
}

// TestValidateBrokenXRef_NonPointers pins the values the rule does not treat as
// pointers: the 7.0 @VOID@ sentinel, CONT/CONC text, escaped and
// whitespace-bearing values, and nested pointers that resolve.
func TestValidateBrokenXRef_NonPointers(t *testing.T) {
	note := &gedcom.Record{XRef: "@N1@", Type: gedcom.RecordTypeNote, LineNumber: 1, Tags: []*gedcom.Tag{
		nil,
		{Level: 1, Tag: "CONT", Value: "@X1@", LineNumber: 2},
		{Level: 1, Tag: "CONC", Value: "@X2@", LineNumber: 3},
		{Level: 1, Tag: "SOUR", Value: "@VOID@", LineNumber: 4},
		{Level: 1, Tag: "NOTE", Value: "@not a pointer@", LineNumber: 5},
		{Level: 1, Tag: "NOTE", Value: "@@X3@", LineNumber: 6},
		{Level: 2, Tag: "SOUR", Value: " @N1@ ", LineNumber: 7},
		{Level: 2, Tag: "SOUR", Value: " @S1@ ", LineNumber: 8},
	}}
	doc := &gedcom.Document{
		Records: []*gedcom.Record{nil, note},
		XRefMap: map[string]*gedcom.Record{"@N1@": note},
	}

	got := validateBrokenXRefs(doc)
	if len(got) != 1 || got[0].RelatedXRef != "@S1@" || got[0].LineNumber != 8 {
		t.Errorf("validateBrokenXRefs = %v, want only the trimmed @S1@ on line 8", got)
	}
}

// TestValidateEveryRepositoryLinkXRef guards #553 on both validation paths: a
// source whose middle REPO link of three dangles must be reported by the batch
// Validate and by the streaming validator. v2's typed model kept only the last
// REPO link, so the streaming path could not see the others.
func TestValidateEveryRepositoryLinkXRef(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @S1@ SOUR
1 TITL Parish Register
1 REPO @R1@
1 REPO @R998@
1 REPO @R1@
0 @R1@ REPO
1 NAME Archive
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	var broken []string
	for _, e := range New().Validate(doc) {
		var issue *Issue
		if errors.As(e, &issue) && issue.Code == CodeBrokenXRef {
			broken = append(broken, fmt.Sprintf("%s line %d", issue.RelatedXRef, issue.LineNumber))
		}
	}
	if len(broken) != 1 || broken[0] != "@R998@ line 7" {
		t.Errorf("batch BROKEN_XREF = %q, want one for @R998@ on line 7", broken)
	}

	sv := NewStreamingValidator(StreamingOptions{})
	for _, rec := range doc.Records {
		sv.ValidateRecord(rec)
	}
	var orphaned []string
	for _, iss := range sv.Finalize() {
		if iss.Details["reference_type"] == "REPO" {
			orphaned = append(orphaned, iss.RelatedXRef+" "+iss.Details["field"])
		}
	}
	if len(orphaned) != 1 || orphaned[0] != "@R998@ RepositoryLinks[1].XRef" {
		t.Errorf("streaming orphaned REPO = %q, want [@R998@ RepositoryLinks[1].XRef]", orphaned)
	}
}

func TestValidateMissingName(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 SEX M
1 ASSO @I1@
2 NAME Not a name of I1
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	got := structureIssues(t, New().Validate(doc))
	if len(got) != 1 {
		t.Fatalf("structure issues = %v, want exactly one MISSING_REQUIRED_FIELD", got)
	}
	issue := got[0]
	if issue.Code != CodeMissingRequiredField || issue.Severity != SeverityWarning {
		t.Errorf("issue = %v, want SeverityWarning %s", issue, CodeMissingRequiredField)
	}
	if issue.LineNumber != 4 || issue.RecordXRef != "@I1@" || issue.Details["field"] != "NAME" {
		t.Errorf("issue = %+v, want line 4, @I1@, field NAME", issue)
	}
	if s := issue.Error(); !strings.Contains(s, "@I1@") || !strings.Contains(s, "[line 4]") {
		t.Errorf("Error() = %q, want both the XRef and the line", s)
	}
}

// TestValidateRecordStructure_TypedEntityOnly pins that a record built in code
// with a typed entity and no raw Tags is judged by the entity, which is what
// the encoder writes for it.
func TestValidateRecordStructure_TypedEntityOnly(t *testing.T) {
	doc := &gedcom.Document{Records: []*gedcom.Record{
		nil,
		{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{
			XRef:  "@I1@",
			Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
		}},
		{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: &gedcom.Family{XRef: "@F1@", Husband: "@I1@"}},
		{XRef: "@F2@", Type: gedcom.RecordTypeFamily, Entity: &gedcom.Family{XRef: "@F2@", Children: []string{"@I1@"}}},
		{XRef: "@I2@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: "@I2@"}},
		{XRef: "@F3@", Type: gedcom.RecordTypeFamily, Entity: &gedcom.Family{XRef: "@F3@"}},
	}}

	got := validateRecordStructure(doc)
	if len(got) != 2 || got[0].RecordXRef != "@I2@" || got[1].RecordXRef != "@F3@" {
		t.Errorf("validateRecordStructure = %v, want only @I2@ and @F3@", got)
	}
}

// structureRulesFixture raises each record-structure code on a known line:
// BROKEN_XREF on lines 14 (ASSO) and 18 (nested SOUR), MISSING_REQUIRED_FIELD
// on line 19 (@I2@) and EMPTY_FAMILY on line 21 (@F1@), whose event-level
// "2 HUSB" is an age structure rather than a member.
const structureRulesFixture = `0 HEAD
1 SOUR gedcom-go
1 SUBM @U1@
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
1 NOTE Record-structure rule fixture: BROKEN_XREF, MISSING_REQUIRED_FIELD and EMPTY_FAMILY, each on a known line.
0 @U1@ SUBM
1 NAME Test Submitter
0 @I1@ INDI
1 NAME John /Doe/
1 SEX M
1 ASSO @I404@
2 RELA Godfather
1 BIRT
2 DATE 1 JAN 1900
2 SOUR @S404@
0 @I2@ INDI
1 SEX F
0 @F1@ FAM
1 MARR
2 HUSB
3 AGE 25
0 TRLR
`

// TestValidateStructureRules_LineNumbersFromFixture asserts, per ADR 0007,
// that every issue the three record-structure rules raise on a decoded
// fixture carries a non-zero line number, and that the rendered error shows
// it alongside the XRef.
func TestValidateStructureRules_LineNumbersFromFixture(t *testing.T) {
	doc, err := decoder.Decode(strings.NewReader(structureRulesFixture))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	got := structureIssues(t, New().Validate(doc))
	var lines []int
	for _, issue := range got {
		lines = append(lines, issue.LineNumber)
	}
	if want := []int{14, 18, 19, 21}; !slices.Equal(lines, want) {
		t.Errorf("structure issue lines = %v, want %v", lines, want)
	}
	seen := map[string]int{}
	for _, issue := range got {
		seen[issue.Code]++
		if issue.LineNumber <= 0 {
			t.Errorf("%s on %s has LineNumber %d, want > 0", issue.Code, issue.RecordXRef, issue.LineNumber)
			continue
		}
		want := fmt.Sprintf("[line %d]", issue.LineNumber)
		if s := issue.Error(); !strings.Contains(s, want) || !strings.Contains(s, issue.RecordXRef) {
			t.Errorf("Error() = %q, want both %s and %q", s, issue.RecordXRef, want)
		}
	}
	for code := range structureCodes {
		if seen[code] == 0 {
			t.Errorf("fixture raised no %s; the line-number assertion is vacuous for it", code)
		}
	}
}

// pointerLinesFixture raises every ORPHANED_* code on a known line, repeats a
// broken CHIL so each occurrence must get its own line, and carries two
// level-1 HUSB lines of which the typed Family.Husband keeps only the last.
const pointerLinesFixture = `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
1 FAMC @F404@
1 FAMS @F405@
1 SOUR @S404@
0 @F1@ FAM
1 HUSB @I8@
1 HUSB @I9@
1 WIFE @I7@
1 CHIL @I6@
1 CHIL @I6@
0 TRLR
`

// TestValidate_BrokenPointerLineNumbers asserts, per ADR 0007, that every
// broken level-1 pointer -- reported as ORPHANED_* through the typed entity or
// as BROKEN_XREF otherwise -- carries the line of the pointing tag, is
// reported exactly once, and renders that line in Error().
func TestValidate_BrokenPointerLineNumbers(t *testing.T) {
	doc, err := decoder.Decode(strings.NewReader(pointerLinesFixture))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	type finding struct {
		code, related string
		line          int
	}
	var got []finding
	for _, issue := range issuesFromErrors(t, New().Validate(doc)) {
		if issue.Code != CodeBrokenXRef && !strings.HasPrefix(issue.Code, "ORPHANED_") {
			continue
		}
		got = append(got, finding{issue.Code, issue.RelatedXRef, issue.LineNumber})
		if want := fmt.Sprintf("[line %d]", issue.LineNumber); !strings.Contains(issue.Error(), want) {
			t.Errorf("Error() = %q, want it to contain %q", issue.Error(), want)
		}
	}
	want := []finding{
		{CodeOrphanedFAMC, "@F404@", 6},
		{CodeOrphanedFAMS, "@F405@", 7},
		{CodeOrphanedSOUR, "@S404@", 8},
		{CodeBrokenXRef, "@I8@", 10},
		{CodeOrphanedHUSB, "@I9@", 11},
		{CodeOrphanedWIFE, "@I7@", 12},
		{CodeOrphanedCHIL, "@I6@", 13},
		{CodeOrphanedCHIL, "@I6@", 14},
	}
	byLine := func(a, b finding) int { return a.line - b.line }
	slices.SortFunc(got, byLine)
	if !slices.Equal(got, want) {
		t.Errorf("broken pointer findings =\n  %v\nwant\n  %v", got, want)
	}
}

// TestValidate_DuplicateXRefPointerLines pins that, when two records share
// an XRef (the decoder keeps both, and XRefMap only the last), each record's
// broken pointer carries its own record's line rather than the last
// duplicate's.
func TestValidate_DuplicateXRefPointerLines(t *testing.T) {
	const input = `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME A
1 FAMS @F9@
0 @I1@ INDI
1 NAME B
1 FAMS @F9@
0 TRLR
`
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	var lines []int
	for _, issue := range issuesFromErrors(t, New().Validate(doc)) {
		if issue.Code == CodeBrokenXRef || strings.HasPrefix(issue.Code, "ORPHANED_") {
			if issue.Code != CodeOrphanedFAMS || issue.RelatedXRef != "@F9@" {
				t.Errorf("unexpected pointer finding %v", issue)
				continue
			}
			lines = append(lines, issue.LineNumber)
		}
	}
	slices.Sort(lines)
	if want := []int{6, 9}; !slices.Equal(lines, want) {
		t.Errorf("ORPHANED_FAMS lines = %v, want %v", lines, want)
	}
}

// TestValidate_RepeatedBrokenHUSB pins that a raw pointer repeated more often
// than the typed entity holds it is still reported: Family.Husband holds @I9@
// once, so the first HUSB line is ORPHANED_HUSB and the second BROKEN_XREF.
func TestValidate_RepeatedBrokenHUSB(t *testing.T) {
	const input = `0 HEAD
1 GEDC
2 VERS 5.5
0 @F1@ FAM
1 HUSB @I9@
1 HUSB @I9@
0 TRLR
`
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	type finding struct {
		code string
		line int
	}
	var got []finding
	for _, issue := range issuesFromErrors(t, New().Validate(doc)) {
		if issue.Code == CodeBrokenXRef || strings.HasPrefix(issue.Code, "ORPHANED_") {
			got = append(got, finding{issue.Code, issue.LineNumber})
		}
	}
	slices.SortFunc(got, func(a, b finding) int { return a.line - b.line })
	want := []finding{{CodeOrphanedHUSB, 5}, {CodeBrokenXRef, 6}}
	if !slices.Equal(got, want) {
		t.Errorf("findings = %v, want %v", got, want)
	}
}

// TestReferenceValidator_LineNumberWithoutRawTags pins that an entity built
// in code, with no raw tags to look up, still reports its orphaned pointer,
// with LineNumber 0.
func TestReferenceValidator_LineNumberWithoutRawTags(t *testing.T) {
	fam := &gedcom.Family{XRef: "@F1@", Husband: "@I404@"}
	doc := &gedcom.Document{
		Records: []*gedcom.Record{{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: fam}},
		XRefMap: map[string]*gedcom.Record{},
	}
	doc.XRefMap["@F1@"] = doc.Records[0]

	got := NewReferenceValidator().Validate(doc)
	if len(got) != 1 || got[0].Code != CodeOrphanedHUSB || got[0].LineNumber != 0 {
		t.Errorf("Validate() = %v, want one ORPHANED_HUSB with LineNumber 0", got)
	}
}

func TestValidateValidFile(t *testing.T) {
	input := `0 HEAD
1 SUBM @U1@
1 GEDC
2 VERS 5.5
0 @U1@ SUBM
1 NAME Tester
0 @I1@ INDI
1 NAME John /Smith/
1 FAMS @F1@
0 @F1@ FAM
1 HUSB @I1@
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	v := New()
	errs := v.Validate(doc)

	if len(errs) != 0 {
		t.Errorf("Expected no validation errors for valid file, got %d errors:", len(errs))
		for _, err := range errs {
			t.Logf("  - %v", err)
		}
	}
	if errs == nil {
		t.Error("Validate returned nil, want an empty non-nil slice")
	}
}

// TestValidate_MatchesValidateAll pins that Validate is ValidateAll in the
// []error shape under every option that filters results.
func TestValidate_MatchesValidateAll(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 ASSO @I999@
1 BIRT
2 DATE 1 JAN 1900
1 DEAT
2 DATE 1 JAN 1800
0 @F1@ FAM
0 TRLR`
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	for _, opts := range []*ValidateOptions{
		nil,
		{Strictness: StrictnessRelaxed},
		{Strictness: StrictnessStrict},
		{MaxErrors: 2},
		{SkipRules: []string{CodeBrokenXRef, CodeMissingSUBM}},
	} {
		v := NewWithOptions(opts)
		issues := v.ValidateAll(doc)
		got := issuesFromErrors(t, v.Validate(doc))
		if len(got) != len(issues) || len(got) == 0 {
			t.Fatalf("opts %+v: Validate len = %d, ValidateAll len = %d", opts, len(got), len(issues))
		}
		for i := range issues {
			if got[i].Code != issues[i].Code || got[i].LineNumber != issues[i].LineNumber {
				t.Errorf("opts %+v [%d]: Validate %v, ValidateAll %v", opts, i, got[i], issues[i])
			}
		}
	}
}

// TestValidateFamilyEdgeCases tests edge cases in family validation
func TestValidateFamilyEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorCode   string
	}{
		{
			name: "empty family (no members)",
			input: `0 HEAD
1 GEDC
2 VERS 5.5
0 @F1@ FAM
0 TRLR`,
			expectError: true,
			errorCode:   CodeEmptyFamily,
		},
		{
			// "2 HUSB" under an event is the husband's age structure, not a
			// member of the family.
			name: "family with only event-level HUSB/WIFE",
			input: `0 HEAD
1 GEDC
2 VERS 5.5
0 @F1@ FAM
1 MARR
2 HUSB
3 AGE 25
2 WIFE
3 AGE 22
0 TRLR`,
			expectError: true,
			errorCode:   CodeEmptyFamily,
		},
		{
			name: "family with only children",
			input: `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME Child /One/
0 @F1@ FAM
1 CHIL @I1@
0 TRLR`,
			expectError: false,
		},
		{
			name: "family with only wife",
			input: `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME Jane /Doe/
0 @F1@ FAM
1 WIFE @I1@
0 TRLR`,
			expectError: false,
		},
		{
			name: "family with only husband",
			input: `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Doe/
0 @F1@ FAM
1 HUSB @I1@
0 TRLR`,
			expectError: false,
		},
		{
			name: "family with all members",
			input: `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Doe/
0 @I2@ INDI
1 NAME Jane /Doe/
0 @I3@ INDI
1 NAME Child /Doe/
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
0 TRLR`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := decoder.Decode(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}

			got := structureIssues(t, New().Validate(doc))

			if tt.expectError {
				if len(got) != 1 || got[0].Code != tt.errorCode {
					t.Fatalf("structure issues = %v, want exactly one %s", got, tt.errorCode)
				}
				if got[0].LineNumber <= 0 {
					t.Errorf("LineNumber = %d, want the FAM line", got[0].LineNumber)
				}
			} else if len(got) != 0 {
				t.Errorf("Expected no structure issues, got %v", got)
			}
		})
	}
}

// Test backward compatibility of New()
func TestNewBackwardCompatibility(t *testing.T) {
	v := New()
	if v == nil {
		t.Fatal("New() returned nil")
	}

	// Should have default config
	if v.config == nil {
		t.Fatal("New() should set default config")
	}
	if v.config.Strictness != StrictnessNormal {
		t.Errorf("Default strictness = %v, want StrictnessNormal", v.config.Strictness)
	}
}

// Test NewWithConfig
func TestNewWithConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *ValidatorConfig
		want   Strictness
	}{
		{
			name:   "nil config uses defaults",
			config: nil,
			want:   StrictnessNormal,
		},
		{
			name:   "relaxed strictness",
			config: &ValidatorConfig{Strictness: StrictnessRelaxed},
			want:   StrictnessRelaxed,
		},
		{
			name:   "normal strictness",
			config: &ValidatorConfig{Strictness: StrictnessNormal},
			want:   StrictnessNormal,
		},
		{
			name:   "strict strictness",
			config: &ValidatorConfig{Strictness: StrictnessStrict},
			want:   StrictnessStrict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewWithConfig(tt.config)
			if v == nil {
				t.Fatal("NewWithConfig() returned nil")
			}
			if v.config.Strictness != tt.want {
				t.Errorf("Strictness = %v, want %v", v.config.Strictness, tt.want)
			}
		})
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if opts == nil {
		t.Fatal("DefaultOptions() returned nil")
	}
	if opts.Strictness != StrictnessNormal {
		t.Errorf("Strictness = %v, want StrictnessNormal", opts.Strictness)
	}
}

func TestNewWithOptions(t *testing.T) {
	opts := &ValidateOptions{Strictness: StrictnessStrict}
	v := NewWithOptions(opts)
	if v == nil {
		t.Fatal("NewWithOptions() returned nil")
	}
	if v.config.Strictness != StrictnessStrict {
		t.Errorf("Strictness = %v, want StrictnessStrict", v.config.Strictness)
	}

	if v := NewWithOptions(nil); v == nil {
		t.Error("NewWithOptions(nil) returned nil")
	} else if v.config == nil || v.config.Strictness != StrictnessNormal {
		t.Errorf("NewWithOptions(nil) strictness = %v, want %v", v.config.Strictness, StrictnessNormal)
	}
}

// Test ValidateAll returns Issues
func TestValidateAllReturnsIssues(t *testing.T) {
	// Create document with date logic issue (death before birth)
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 1950
1 DEAT
2 DATE 1940
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	v := New()
	issues := v.ValidateAll(doc)

	// Should have at least one issue for death before birth
	if len(issues) == 0 {
		t.Fatal("Expected at least one issue")
	}

	found := false
	for _, issue := range issues {
		if issue.Code == CodeDeathBeforeBirth {
			found = true
			if issue.Severity != SeverityError {
				t.Errorf("DeathBeforeBirth should be SeverityError, got %v", issue.Severity)
			}
			if issue.RecordXRef != "@I1@" {
				t.Errorf("RecordXRef = %q, want @I1@", issue.RecordXRef)
			}
			break
		}
	}

	if !found {
		t.Error("Expected DEATH_BEFORE_BIRTH issue")
	}
}

// Test ValidateAll with nil document
func TestValidateAllNilDocument(t *testing.T) {
	v := New()
	issues := v.ValidateAll(nil)
	if issues != nil {
		t.Errorf("ValidateAll(nil) = %v, want nil", issues)
	}
}

// Test ValidateDateLogic
func TestValidateDateLogic(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME Old /Person/
1 BIRT
2 DATE 1800
1 DEAT
2 DATE 1950
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// With normal strictness, should get warning for impossible age
	v := New()
	issues := v.ValidateDateLogic(doc)

	found := false
	for _, issue := range issues {
		if issue.Code == CodeImpossibleAge {
			found = true
			if issue.Severity != SeverityWarning {
				t.Errorf("ImpossibleAge should be SeverityWarning, got %v", issue.Severity)
			}
			break
		}
	}

	if !found {
		t.Error("Expected IMPOSSIBLE_AGE warning")
	}
}

// Test ValidateDateLogic with nil document
func TestValidateDateLogicNilDocument(t *testing.T) {
	v := New()
	issues := v.ValidateDateLogic(nil)
	if issues != nil {
		t.Errorf("ValidateDateLogic(nil) = %v, want nil", issues)
	}
}

// Test FindOrphanedReferences
func TestFindOrphanedReferences(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Smith/
1 FAMS @F999@
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	v := New()
	issues := v.FindOrphanedReferences(doc)

	if len(issues) == 0 {
		t.Fatal("Expected orphaned reference issue")
	}

	found := false
	for _, issue := range issues {
		if issue.Code == CodeOrphanedFAMS {
			found = true
			if issue.Severity != SeverityError {
				t.Errorf("OrphanedFAMS should be SeverityError, got %v", issue.Severity)
			}
			if issue.RelatedXRef != "@F999@" {
				t.Errorf("RelatedXRef = %q, want @F999@", issue.RelatedXRef)
			}
			break
		}
	}

	if !found {
		t.Error("Expected ORPHANED_FAMS issue")
	}
}

// Test FindOrphanedReferences with nil document
func TestFindOrphanedReferencesNilDocument(t *testing.T) {
	v := New()
	issues := v.FindOrphanedReferences(nil)
	if issues != nil {
		t.Errorf("FindOrphanedReferences(nil) = %v, want nil", issues)
	}
}

// Test FindPotentialDuplicates
func TestFindPotentialDuplicates(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 1950
1 SEX M
0 @I2@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 1951
1 SEX M
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	v := New()
	duplicates := v.FindPotentialDuplicates(doc)

	if len(duplicates) == 0 {
		t.Fatal("Expected potential duplicate")
	}

	pair := duplicates[0]
	if pair.Individual1 == nil || pair.Individual2 == nil {
		t.Fatal("DuplicatePair individuals should not be nil")
	}
	if pair.Confidence < 0.7 {
		t.Errorf("Confidence = %v, want >= 0.7", pair.Confidence)
	}
	if len(pair.MatchReasons) == 0 {
		t.Error("MatchReasons should not be empty")
	}
}

// Test FindPotentialDuplicates with nil document
func TestFindPotentialDuplicatesNilDocument(t *testing.T) {
	v := New()
	duplicates := v.FindPotentialDuplicates(nil)
	if duplicates != nil {
		t.Errorf("FindPotentialDuplicates(nil) = %v, want nil", duplicates)
	}
}

// Test QualityReport
func TestQualityReport(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 1950
0 @I2@ INDI
1 NAME Jane /Doe/
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
0 @S1@ SOUR
1 TITL Test Source
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	v := New()
	report := v.QualityReport(doc)

	if report == nil {
		t.Fatal("QualityReport() returned nil")
	}

	// Check summary counts
	if report.TotalIndividuals != 2 {
		t.Errorf("TotalIndividuals = %d, want 2", report.TotalIndividuals)
	}
	if report.TotalFamilies != 1 {
		t.Errorf("TotalFamilies = %d, want 1", report.TotalFamilies)
	}
	if report.TotalSources != 1 {
		t.Errorf("TotalSources = %d, want 1", report.TotalSources)
	}

	// Check completeness metrics
	if report.IndividualsWithBirthDate != 1 {
		t.Errorf("IndividualsWithBirthDate = %d, want 1", report.IndividualsWithBirthDate)
	}
	if report.BirthDateCoverage != 0.5 {
		t.Errorf("BirthDateCoverage = %v, want 0.5", report.BirthDateCoverage)
	}

	// Check that completeness issues are generated
	if report.TotalIssues == 0 {
		t.Error("Expected completeness issues")
	}
}

// Test QualityReport with nil document
func TestQualityReportNilDocument(t *testing.T) {
	v := New()
	report := v.QualityReport(nil)

	if report == nil {
		t.Fatal("QualityReport(nil) should return empty report, not nil")
	}
	if report.TotalIndividuals != 0 {
		t.Errorf("TotalIndividuals = %d, want 0", report.TotalIndividuals)
	}
	if len(report.Errors) != 0 {
		t.Errorf("Errors = %v, want empty", report.Errors)
	}
}

// Test QualityReport JSON output
func TestQualityReportJSON(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Smith/
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	v := New()
	report := v.QualityReport(doc)

	jsonBytes, err := report.JSON()
	if err != nil {
		t.Fatalf("JSON() error = %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("JSON output not valid: %v", err)
	}

	// Check some expected keys
	if _, ok := parsed["total_individuals"]; !ok {
		t.Error("JSON missing total_individuals key")
	}
}

// Test Strictness filtering
func TestStrictnessFiltering(t *testing.T) {
	// Create document with issues of different severities
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME Old /Person/
1 BIRT
2 DATE 1800
1 DEAT
2 DATE 1950
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	tests := []struct {
		name       string
		strictness Strictness
		wantErrors bool
		wantWarns  bool
		wantInfo   bool
	}{
		{
			name:       "relaxed only shows errors",
			strictness: StrictnessRelaxed,
			wantErrors: true,
			wantWarns:  false,
			wantInfo:   false,
		},
		{
			name:       "normal shows errors and warnings",
			strictness: StrictnessNormal,
			wantErrors: true,
			wantWarns:  true,
			wantInfo:   false,
		},
		{
			name:       "strict shows all",
			strictness: StrictnessStrict,
			wantErrors: true,
			wantWarns:  true,
			wantInfo:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewWithConfig(&ValidatorConfig{Strictness: tt.strictness})

			// Use ValidateDateLogic which can return warnings (impossible age)
			issues := v.ValidateDateLogic(doc)

			hasWarnings := false
			for _, issue := range issues {
				if issue.Severity == SeverityWarning {
					hasWarnings = true
					break
				}
			}

			if hasWarnings != tt.wantWarns {
				t.Errorf("hasWarnings = %v, want %v", hasWarnings, tt.wantWarns)
			}
		})
	}
}

// Test configuration with custom DateLogic config
func TestConfigWithCustomDateLogic(t *testing.T) {
	// Create document with person who lived to 130 years
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME Old /Person/
1 BIRT
2 DATE 1800
1 DEAT
2 DATE 1930
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// With default config (max 120), should get warning
	v1 := New()
	issues1 := v1.ValidateDateLogic(doc)
	hasWarning1 := false
	for _, issue := range issues1 {
		if issue.Code == CodeImpossibleAge {
			hasWarning1 = true
			break
		}
	}
	if !hasWarning1 {
		t.Error("Default config should warn about 130 year lifespan")
	}

	// With custom config (max 140), should not get warning
	v2 := NewWithConfig(&ValidatorConfig{
		DateLogic: &DateLogicConfig{
			MaxReasonableAge: 140,
		},
	})
	issues2 := v2.ValidateDateLogic(doc)
	hasWarning2 := false
	for _, issue := range issues2 {
		if issue.Code == CodeImpossibleAge {
			hasWarning2 = true
			break
		}
	}
	if hasWarning2 {
		t.Error("Custom config with max 140 should not warn about 130 year lifespan")
	}
}

// Test configuration with custom Duplicates config
func TestConfigWithCustomDuplicates(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 1950
1 SEX M
0 @I2@ INDI
1 NAME Jon /Smith/
1 BIRT
2 DATE 1951
1 SEX M
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// With high similarity threshold, should not match
	config := &ValidatorConfig{
		Duplicates: &DuplicateConfig{
			MinNameSimilarity: 0.99, // Very high threshold
			MaxBirthYearDiff:  2,
			MinConfidence:     0.9,
		},
	}
	v := NewWithConfig(config)
	duplicates := v.FindPotentialDuplicates(doc)

	// John and Jon should not match with such high similarity threshold
	if len(duplicates) != 0 {
		t.Errorf("With high similarity threshold, expected no duplicates, got %d", len(duplicates))
	}
}

// Test lazy initialization of sub-validators
func TestLazyInitialization(t *testing.T) {
	v := New()

	// Initially, sub-validators should be nil
	if v.dateLogic != nil {
		t.Error("dateLogic should be nil initially")
	}
	if v.references != nil {
		t.Error("references should be nil initially")
	}
	if v.duplicates != nil {
		t.Error("duplicates should be nil initially")
	}
	if v.quality != nil {
		t.Error("quality should be nil initially")
	}

	// Create a minimal document
	doc := &gedcom.Document{}

	// After calling ValidateDateLogic, dateLogic should be initialized
	v.ValidateDateLogic(doc)
	if v.dateLogic == nil {
		t.Error("dateLogic should be initialized after ValidateDateLogic")
	}

	// Other validators should still be nil
	if v.references != nil {
		t.Error("references should still be nil")
	}
}

// Test that Validate() still works (backward compatibility)
func TestValidateBackwardCompatibility(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME John /Smith/
0 @F1@ FAM
1 HUSB @I1@
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	v := New()
	errs := v.Validate(doc)

	// This should still return []error, not []Issue
	if len(errs) != 0 {
		t.Errorf("Expected no errors for valid file, got %d", len(errs))
	}

	// Test that returned errors implement error interface
	for _, err := range errs {
		_ = err.Error() // Should not panic
	}
}

// Test ValidateCustomTags method
func TestValidateCustomTags(t *testing.T) {
	t.Run("returns nil when no registry configured", func(t *testing.T) {
		v := New() // No registry configured
		doc := &gedcom.Document{
			Records: []*gedcom.Record{
				{
					XRef: "@I1@",
					Type: gedcom.RecordTypeIndividual,
					Tags: []*gedcom.Tag{
						{Level: 1, Tag: "_CUSTOM", Value: "value"},
					},
				},
			},
		}

		issues := v.ValidateCustomTags(doc)
		if issues != nil {
			t.Errorf("expected nil when no registry configured, got %d issues", len(issues))
		}
	})

	t.Run("returns nil for nil document", func(t *testing.T) {
		registry := NewTagRegistry()
		v := NewWithConfig(&ValidatorConfig{
			TagRegistry: registry,
		})

		issues := v.ValidateCustomTags(nil)
		if issues != nil {
			t.Errorf("expected nil for nil document, got %v", issues)
		}
	})

	t.Run("validates custom tags with registry", func(t *testing.T) {
		registry := NewTagRegistry()
		_ = registry.Register("_MILT", TagDefinition{
			Tag:            "_MILT",
			AllowedParents: []string{"INDI"},
		})

		v := NewWithConfig(&ValidatorConfig{
			TagRegistry:        registry,
			ValidateCustomTags: true,
		})

		doc := &gedcom.Document{
			Records: []*gedcom.Record{
				{
					XRef: "@F1@",
					Type: gedcom.RecordTypeFamily,
					Tags: []*gedcom.Tag{
						{Level: 1, Tag: "_MILT", Value: "Army"}, // Invalid: FAM is not INDI
					},
				},
			},
		}

		issues := v.ValidateCustomTags(doc)
		if len(issues) != 1 {
			t.Errorf("expected 1 issue, got %d", len(issues))
		}
		if len(issues) > 0 && issues[0].Code != CodeInvalidTagParent {
			t.Errorf("expected code %s, got %s", CodeInvalidTagParent, issues[0].Code)
		}
	})
}

// Test ValidateAll includes custom tag validation
func TestValidateAllWithCustomTags(t *testing.T) {
	registry := NewTagRegistry()
	_ = registry.Register("_MILT", TagDefinition{
		Tag:            "_MILT",
		AllowedParents: []string{"INDI"},
	})

	v := NewWithConfig(&ValidatorConfig{
		TagRegistry:        registry,
		ValidateCustomTags: true,
		Strictness:         StrictnessStrict,
	})

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{
				XRef: "@F1@",
				Type: gedcom.RecordTypeFamily,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "_MILT", Value: "Army"}, // Invalid parent
				},
			},
		},
	}

	issues := v.ValidateAll(doc)

	// Should find the invalid parent issue
	found := false
	for _, issue := range issues {
		if issue.Code == CodeInvalidTagParent {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected ValidateAll to include custom tag validation issues")
	}
}

// Test ValidatorConfig with TagRegistry
func TestValidatorConfigWithTagRegistry(t *testing.T) {
	registry := NewTagRegistry()

	config := &ValidatorConfig{
		TagRegistry:        registry,
		ValidateCustomTags: true,
	}

	v := NewWithConfig(config)
	if v.config.TagRegistry != registry {
		t.Error("expected TagRegistry to be set in config")
	}
	if !v.config.ValidateCustomTags {
		t.Error("expected ValidateCustomTags to be true")
	}
}

// TestMaxErrors tests the MaxErrors configuration option.
func TestMaxErrors(t *testing.T) {
	// Create document with multiple issues (orphaned references)
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Smith/
1 FAMS @F999@
1 FAMC @F998@
0 @I2@ INDI
1 NAME Jane /Doe/
1 FAMS @F997@
1 FAMC @F996@
0 @I3@ INDI
1 NAME Bob /Brown/
1 FAMS @F995@
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	t.Run("unlimited errors (default)", func(t *testing.T) {
		v := New() // MaxErrors = 0 (unlimited)
		issues := v.ValidateAll(doc)

		// Should have all orphaned reference issues
		if len(issues) < 5 {
			t.Errorf("Expected at least 5 issues with unlimited, got %d", len(issues))
		}
	})

	t.Run("limit to 2 errors", func(t *testing.T) {
		v := NewWithConfig(&ValidatorConfig{
			MaxErrors: 2,
		})
		issues := v.ValidateAll(doc)

		if len(issues) != 2 {
			t.Errorf("Expected exactly 2 issues with MaxErrors=2, got %d", len(issues))
		}
	})

	t.Run("limit to 1 error", func(t *testing.T) {
		v := NewWithConfig(&ValidatorConfig{
			MaxErrors: 1,
		})
		issues := v.ValidateAll(doc)

		if len(issues) != 1 {
			t.Errorf("Expected exactly 1 issue with MaxErrors=1, got %d", len(issues))
		}
	})

	t.Run("limit higher than actual issues", func(t *testing.T) {
		v := NewWithConfig(&ValidatorConfig{
			MaxErrors: 100,
		})
		issues := v.ValidateAll(doc)

		// Should return all issues since limit is higher
		if len(issues) < 5 {
			t.Errorf("Expected at least 5 issues, got %d", len(issues))
		}
	})
}

// TestSkipRules tests the SkipRules configuration option.
func TestSkipRules(t *testing.T) {
	// Create document with orphaned references and date logic issues
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Smith/
1 FAMS @F999@
1 BIRT
2 DATE 1950
1 DEAT
2 DATE 1940
0 @I2@ INDI
1 NAME Jane /Doe/
1 FAMC @F998@
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	t.Run("no rules skipped (default)", func(t *testing.T) {
		v := New()
		issues := v.ValidateAll(doc)

		// Should have both types of issues
		hasOrphaned := false
		hasDateLogic := false
		for _, issue := range issues {
			if issue.Code == CodeOrphanedFAMS || issue.Code == CodeOrphanedFAMC {
				hasOrphaned = true
			}
			if issue.Code == CodeDeathBeforeBirth {
				hasDateLogic = true
			}
		}

		if !hasOrphaned {
			t.Error("Expected orphaned reference issues")
		}
		if !hasDateLogic {
			t.Error("Expected death before birth issue")
		}
	})

	t.Run("skip ORPHANED_FAMS", func(t *testing.T) {
		v := NewWithConfig(&ValidatorConfig{
			SkipRules: []string{CodeOrphanedFAMS},
		})
		issues := v.ValidateAll(doc)

		for _, issue := range issues {
			if issue.Code == CodeOrphanedFAMS {
				t.Error("Expected ORPHANED_FAMS to be skipped")
			}
		}

		// Other issues should still be present
		hasOther := false
		for _, issue := range issues {
			if issue.Code != CodeOrphanedFAMS {
				hasOther = true
				break
			}
		}
		if !hasOther {
			t.Error("Expected other issues to remain")
		}
	})

	t.Run("skip multiple rules", func(t *testing.T) {
		v := NewWithConfig(&ValidatorConfig{
			SkipRules: []string{CodeOrphanedFAMS, CodeOrphanedFAMC, CodeDeathBeforeBirth},
		})
		issues := v.ValidateAll(doc)

		for _, issue := range issues {
			if issue.Code == CodeOrphanedFAMS || issue.Code == CodeOrphanedFAMC || issue.Code == CodeDeathBeforeBirth {
				t.Errorf("Expected %s to be skipped", issue.Code)
			}
		}
	})

	t.Run("skip nonexistent rule has no effect", func(t *testing.T) {
		v := NewWithConfig(&ValidatorConfig{
			SkipRules: []string{"NONEXISTENT_RULE"},
		})
		issues := v.ValidateAll(doc)

		// Should still have all actual issues
		if len(issues) == 0 {
			t.Error("Expected issues to remain when skipping nonexistent rule")
		}
	})
}

// TestSkipRulesAndMaxErrorsCombined tests that both options work together.
func TestSkipRulesAndMaxErrorsCombined(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Smith/
1 FAMS @F999@
1 FAMC @F998@
1 BIRT
2 DATE 1950
1 DEAT
2 DATE 1940
0 @I2@ INDI
1 NAME Jane /Doe/
1 FAMS @F997@
0 TRLR`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	v := NewWithConfig(&ValidatorConfig{
		SkipRules: []string{CodeDeathBeforeBirth}, // Skip date logic
		MaxErrors: 2,                              // Limit to 2
	})
	issues := v.ValidateAll(doc)

	// Should have exactly 2 issues
	if len(issues) != 2 {
		t.Errorf("Expected 2 issues, got %d", len(issues))
	}

	// None should be DEATH_BEFORE_BIRTH
	for _, issue := range issues {
		if issue.Code == CodeDeathBeforeBirth {
			t.Error("Expected DEATH_BEFORE_BIRTH to be skipped")
		}
	}
}

// TestFilterBySkipRulesHelper tests the filterBySkipRules helper function.
func TestFilterBySkipRulesHelper(t *testing.T) {
	issues := []Issue{
		{Code: "A001", Message: "Issue A1"},
		{Code: "B001", Message: "Issue B1"},
		{Code: "A001", Message: "Issue A2"},
		{Code: "C001", Message: "Issue C1"},
	}

	t.Run("nil config returns all issues", func(t *testing.T) {
		v := &Validator{config: nil}
		result := v.filterBySkipRules(issues)
		if len(result) != len(issues) {
			t.Errorf("Expected %d issues, got %d", len(issues), len(result))
		}
	})

	t.Run("empty SkipRules returns all issues", func(t *testing.T) {
		v := &Validator{config: &ValidatorConfig{SkipRules: []string{}}}
		result := v.filterBySkipRules(issues)
		if len(result) != len(issues) {
			t.Errorf("Expected %d issues, got %d", len(issues), len(result))
		}
	})

	t.Run("filters matching codes", func(t *testing.T) {
		v := &Validator{config: &ValidatorConfig{SkipRules: []string{"A001"}}}
		result := v.filterBySkipRules(issues)

		// Should have 2 issues (B001 and C001)
		if len(result) != 2 {
			t.Errorf("Expected 2 issues, got %d", len(result))
		}
		for _, issue := range result {
			if issue.Code == "A001" {
				t.Error("A001 should have been filtered")
			}
		}
	})
}

// TestApplyMaxErrorsHelper tests the applyMaxErrors helper function.
func TestApplyMaxErrorsHelper(t *testing.T) {
	issues := []Issue{
		{Code: "A001", Message: "Issue 1"},
		{Code: "A002", Message: "Issue 2"},
		{Code: "A003", Message: "Issue 3"},
		{Code: "A004", Message: "Issue 4"},
	}

	t.Run("nil config returns all issues", func(t *testing.T) {
		v := &Validator{config: nil}
		result := v.applyMaxErrors(issues)
		if len(result) != len(issues) {
			t.Errorf("Expected %d issues, got %d", len(issues), len(result))
		}
	})

	t.Run("zero MaxErrors returns all issues", func(t *testing.T) {
		v := &Validator{config: &ValidatorConfig{MaxErrors: 0}}
		result := v.applyMaxErrors(issues)
		if len(result) != len(issues) {
			t.Errorf("Expected %d issues, got %d", len(issues), len(result))
		}
	})

	t.Run("negative MaxErrors returns all issues", func(t *testing.T) {
		v := &Validator{config: &ValidatorConfig{MaxErrors: -1}}
		result := v.applyMaxErrors(issues)
		if len(result) != len(issues) {
			t.Errorf("Expected %d issues, got %d", len(issues), len(result))
		}
	})

	t.Run("MaxErrors limits results", func(t *testing.T) {
		v := &Validator{config: &ValidatorConfig{MaxErrors: 2}}
		result := v.applyMaxErrors(issues)
		if len(result) != 2 {
			t.Errorf("Expected 2 issues, got %d", len(result))
		}
		// First 2 issues should be preserved
		if result[0].Code != "A001" || result[1].Code != "A002" {
			t.Error("First 2 issues should be preserved")
		}
	})

	t.Run("MaxErrors higher than count returns all", func(t *testing.T) {
		v := &Validator{config: &ValidatorConfig{MaxErrors: 100}}
		result := v.applyMaxErrors(issues)
		if len(result) != len(issues) {
			t.Errorf("Expected %d issues, got %d", len(issues), len(result))
		}
	})
}

// --- Bounded duplicate detection wiring (#530) -------------------------------

// duplicateLimitCap is the MaxGroupSize the tests below configure. It is small
// so that duplicateAndLimitDocument can cross it with a handful of records
// instead of a thousand.
const duplicateLimitCap = 3

// duplicateAndLimitDocument builds a document carrying all three of the signals
// the SkipDuplicateDetection tests have to tell apart:
//
//   - a matching duplicate pair (POTENTIAL_DUPLICATE, Info),
//   - a surname group past duplicateLimitCap (DUPLICATE_DETECTION_LIMITED,
//     Warning),
//   - a death before birth, which belongs to an entirely different validator
//     (DEATH_BEFORE_BIRTH, Error) and must survive the duplicate opt-out. It is
//     what distinguishes "duplicate detection was skipped" from "ValidateAll
//     stopped working".
func duplicateAndLimitDocument() *gedcom.Document {
	var individuals []*gedcom.Individual

	// The duplicate pair: identical names, same birth year, same sex.
	for _, xref := range []string{"@D1@", "@D2@"} {
		ind := makeIndividual(xref, 1900, 0)
		ind.Names = []*gedcom.PersonalName{{Given: "John", Surname: "Ashworth"}}
		ind.Sex = "M"
		individuals = append(individuals, ind)
	}

	// The oversized group. Given names are distinct, so it is the skip that
	// removes these from the results rather than a failed comparison.
	for i := 0; i <= duplicateLimitCap; i++ {
		individuals = append(individuals, &gedcom.Individual{
			XRef:  fmt.Sprintf("@B%d@", i),
			Names: []*gedcom.PersonalName{{Given: "P" + strconv.Itoa(i), Surname: "Bellwether"}},
			Sex:   "F",
		})
	}

	other := makeIndividual("@X1@", 1950, 1940)
	other.Names = []*gedcom.PersonalName{{Given: "Mallory", Surname: "Quintrell"}}
	individuals = append(individuals, other)

	return makeDocument(individuals, nil)
}

// duplicateLimitOptions returns validator options whose duplicate cap is small
// enough for duplicateAndLimitDocument's Bellwether group to cross it.
func duplicateLimitOptions(strictness Strictness, skipDuplicates bool) *ValidateOptions {
	duplicates := DefaultDuplicateConfig()
	duplicates.MaxGroupSize = duplicateLimitCap

	return &ValidateOptions{
		Strictness:             strictness,
		Duplicates:             duplicates,
		SkipDuplicateDetection: skipDuplicates,
	}
}

// codeCounts tallies issues by code so a test can assert on presence and
// absence in one place.
func codeCounts(issues []Issue) map[string]int {
	counts := make(map[string]int, len(issues))
	for _, issue := range issues {
		counts[issue.Code]++
	}
	return counts
}

// TestValidateAll_SkipDuplicateDetection covers the opt-out from the most
// expensive validator. Skipping must remove BOTH duplicate signals -- the pairs
// and the notice that some groups were not swept -- because a limit warning
// from an analysis the caller deliberately turned off would be noise. Every
// other validator has to keep running, which the death-before-birth error pins.
//
// StrictnessStrict is required to see POTENTIAL_DUPLICATE at all: it is Info.
func TestValidateAll_SkipDuplicateDetection(t *testing.T) {
	doc := duplicateAndLimitDocument()

	// Baseline first, so the absences below are the option's doing and not a
	// fixture that never produced the signals.
	baseline := codeCounts(NewWithOptions(duplicateLimitOptions(StrictnessStrict, false)).ValidateAll(doc))
	if baseline[CodePotentialDuplicate] == 0 {
		t.Fatal("fixture produced no POTENTIAL_DUPLICATE; the skip assertion would be vacuous")
	}
	if baseline[CodeDuplicateDetectionLimited] != 1 {
		t.Fatalf("fixture produced %d limit issues, want 1; the skip assertion would be vacuous",
			baseline[CodeDuplicateDetectionLimited])
	}
	if baseline[CodeDeathBeforeBirth] == 0 {
		t.Fatal("fixture produced no DEATH_BEFORE_BIRTH; the 'other validators still run' " +
			"assertion would be vacuous")
	}

	skipped := codeCounts(NewWithOptions(duplicateLimitOptions(StrictnessStrict, true)).ValidateAll(doc))
	if got := skipped[CodePotentialDuplicate]; got != 0 {
		t.Errorf("SkipDuplicateDetection still emitted %d POTENTIAL_DUPLICATE issue(s)", got)
	}
	if got := skipped[CodeDuplicateDetectionLimited]; got != 0 {
		t.Errorf("SkipDuplicateDetection still emitted %d DUPLICATE_DETECTION_LIMITED issue(s): "+
			"a limit notice for an analysis the caller turned off is noise", got)
	}
	if got, want := skipped[CodeDeathBeforeBirth], baseline[CodeDeathBeforeBirth]; got != want {
		t.Errorf("DEATH_BEFORE_BIRTH count = %d with the duplicate opt-out, want %d: "+
			"the option must not disturb any other validator", got, want)
	}
}

// TestFindPotentialDuplicates_IgnoresSkipOption pins the deliberate asymmetry:
// SkipDuplicateDetection removes duplicate detection from the ValidateAll
// sweep, but calling FindPotentialDuplicates IS the request for duplicates, and
// returning none would be surprising rather than helpful.
func TestFindPotentialDuplicates_IgnoresSkipOption(t *testing.T) {
	doc := duplicateAndLimitDocument()
	v := NewWithOptions(duplicateLimitOptions(StrictnessStrict, true))

	if pairs := v.FindPotentialDuplicates(doc); len(pairs) == 0 {
		t.Error("FindPotentialDuplicates returned no pairs under SkipDuplicateDetection; " +
			"the option governs ValidateAll, not an explicit request")
	}
}

// TestValidateAll_LimitIssueSeverityRouting guards the Warning-not-Info
// decision on DUPLICATE_DETECTION_LIMITED.
//
// The default strictness reports errors and warnings, so a Warning reaches a
// caller who configured nothing -- which is the point: silence there would let
// "no duplicates found" be read as "none exist". Demoting it to Info would make
// the StrictnessNormal half of this test fail. StrictnessRelaxed asks for errors
// only and is expected to drop it, which is what proves the issue is travelling
// through the severity filter rather than bypassing it.
func TestValidateAll_LimitIssueSeverityRouting(t *testing.T) {
	doc := duplicateAndLimitDocument()

	normal := codeCounts(NewWithOptions(duplicateLimitOptions(StrictnessNormal, false)).ValidateAll(doc))
	if got := normal[CodeDuplicateDetectionLimited]; got != 1 {
		t.Errorf("StrictnessNormal reported %d limit issues, want 1: an incomplete sweep must "+
			"reach a caller who configured nothing", got)
	}
	// Info-level duplicate pairs are filtered out at this strictness, so the
	// warning above is genuinely the only duplicate signal that survives.
	if got := normal[CodePotentialDuplicate]; got != 0 {
		t.Errorf("StrictnessNormal reported %d POTENTIAL_DUPLICATE issues, want 0 (Info)", got)
	}

	relaxed := codeCounts(NewWithOptions(duplicateLimitOptions(StrictnessRelaxed, false)).ValidateAll(doc))
	if got := relaxed[CodeDuplicateDetectionLimited]; got != 0 {
		t.Errorf("StrictnessRelaxed reported %d limit issues, want 0: relaxed means errors only", got)
	}
}

// TestValidateAll_LimitIssueSurvivesMaxErrors guards the ordering that makes
// the completeness notice trustworthy.
//
// filterByStrictness ends in applyMaxErrors, which truncates by POSITION, not
// by severity -- it returns issues[:MaxErrors]. While the limit issue was
// appended after every other validator's output it was therefore the first
// issue dropped, and the caller most likely to set MaxErrors is precisely the
// hardened caller the group cap exists for. Losing it hands them "no
// duplicates" when the truth is "not everything was compared", which is the
// exact misreading the Warning severity was chosen to prevent.
//
// MaxErrors is set to 1 so the assertion is unambiguous: exactly one issue
// comes back, and it has to be this one.
func TestValidateAll_LimitIssueSurvivesMaxErrors(t *testing.T) {
	doc := duplicateAndLimitDocument()

	opts := duplicateLimitOptions(StrictnessStrict, false)
	opts.MaxErrors = 1

	issues := NewWithOptions(opts).ValidateAll(doc)
	if len(issues) != 1 {
		t.Fatalf("MaxErrors=1 returned %d issues, want 1", len(issues))
	}
	if issues[0].Code != CodeDuplicateDetectionLimited {
		t.Errorf("MaxErrors=1 kept %s, want %s: a meta-issue about the completeness of "+
			"the results must outrank the results themselves",
			issues[0].Code, CodeDuplicateDetectionLimited)
	}

	// Without a cap the same document yields more than one issue, so the
	// assertion above is really testing truncation order and not a document
	// that happens to produce a single issue.
	uncapped := NewWithOptions(duplicateLimitOptions(StrictnessStrict, false)).ValidateAll(doc)
	if len(uncapped) < 2 {
		t.Fatalf("fixture produced %d issues without MaxErrors; the truncation test is vacuous",
			len(uncapped))
	}
}

// TestFindPotentialDuplicatesReport covers the Validator-level report accessor.
//
// Before it existed, FindPotentialDuplicates' doc comment pointed callers at
// DuplicateDetector.FindDuplicatesReport to discover a suppressed sweep -- but
// Validator exposes no detector, so following that advice meant rebuilding one
// from the same options by hand. This asserts the report is reachable without
// leaving the abstraction, and that it agrees with the slice-returning method.
func TestFindPotentialDuplicatesReport(t *testing.T) {
	doc := duplicateAndLimitDocument()
	v := NewWithOptions(duplicateLimitOptions(StrictnessNormal, false))

	report := v.FindPotentialDuplicatesReport(doc)

	if len(report.LimitIssues) != 1 {
		t.Fatalf("got %d limit issues, want 1: the oversized group must be reported",
			len(report.LimitIssues))
	}
	if code := report.LimitIssues[0].Code; code != CodeDuplicateDetectionLimited {
		t.Errorf("limit issue code = %s, want %s", code, CodeDuplicateDetectionLimited)
	}
	if got, want := len(report.Pairs), len(v.FindPotentialDuplicates(doc)); got != want {
		t.Errorf("report carried %d pairs, FindPotentialDuplicates returned %d: the two "+
			"must describe the same sweep", got, want)
	}
	if len(report.Pairs) == 0 {
		t.Error("fixture produced no pairs; the agreement check above is vacuous")
	}

	// The skip option is deliberately ignored here, exactly as it is by
	// FindPotentialDuplicates -- asking for duplicates is an explicit request.
	skipped := NewWithOptions(duplicateLimitOptions(StrictnessNormal, true)).
		FindPotentialDuplicatesReport(doc)
	if len(skipped.Pairs) != len(report.Pairs) {
		t.Errorf("SkipDuplicateDetection changed the report (%d pairs vs %d); it must apply "+
			"to the ValidateAll sweep only", len(skipped.Pairs), len(report.Pairs))
	}

	if got := New().FindPotentialDuplicatesReport(nil); got.Pairs != nil || got.LimitIssues != nil {
		t.Error("nil document must yield a zero report")
	}
}
