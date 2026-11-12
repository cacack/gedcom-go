package validator

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// grammarCodes are the codes the grammar check reports.
var grammarCodes = []string{
	CodeUnexpectedSubstructure,
	CodeVersionForeignStructure,
	CodeMissingRequiredSubstructure,
	CodeCardinalityExceeded,
	CodeInvalidEnumValue,
	CodeInvalidPayload,
	CodeInvalidRecordXRef,
	CodeDeprecatedStructure,
}

func decodeGrammarDoc(t *testing.T, src string) *gedcom.Document {
	t.Helper()
	doc, err := decoder.Decode(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	return doc
}

// grammarFindings renders issues as "CODE L<line> <tag>", the tag being
// Details["tag"], so a test pins what was reported where.
func grammarFindings(issues []Issue) []string {
	out := make([]string, 0, len(issues))
	for _, i := range issues {
		out = append(out, fmt.Sprintf("%s L%d %s", i.Code, i.LineNumber, i.Details["tag"]))
	}
	return out
}

func assertGrammar(t *testing.T, src string, want ...string) []Issue {
	t.Helper()
	issues := validateGrammar(decodeGrammarDoc(t, src))
	if got := grammarFindings(issues); !slices.Equal(got, want) {
		t.Errorf("grammar findings\n got: %q\nwant: %q", got, want)
	}
	return issues
}

const head70 = "0 HEAD\n1 GEDC\n2 VERS 7.0\n"

// head70Lang is head70 with a valid HEAD.LANG.
const head70Lang = head70 + "1 LANG en-US\n"

const head551 = "0 HEAD\n1 SOUR test\n1 SUBM @U1@\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n0 @U1@ SUBM\n1 NAME Submitter\n"

func TestGrammar_ValidDocumentsAreClean(t *testing.T) {
	assertGrammar(t, head70Lang+
		"0 @I1@ INDI\n1 NAME John /Doe/\n1 SEX M\n1 FAMC @F1@\n2 PEDI BIRTH\n1 BIRT\n2 DATE 1 JAN 1900\n3 TIME 23:59:59.5Z\n2 AGE > 1y 2m 3w 4d\n"+
		"1 RESN CONFIDENTIAL, PRIVACY\n1 CHAN\n2 DATE 2 FEB 2020\n3 TIME 8:38\n1 _CUSTOM anything\n2 SEX Z\n"+
		"0 @F1@ FAM\n1 CHIL @I1@\n1 NCHI 2\n0 @S1@ SOUR\n1 TITL t\n1 EXID 12\n2 TYPE https://example.com\n"+
		"0 _EXT\n1 SEX Z\n0 TRLR\n")
	assertGrammar(t, head551+
		"0 @I1@ INDI\n1 NAME John /Doe/\n1 SEX u\n1 BIRT\n2 AGE <0y\n2 NOTE @N1@\n2 NOTE text\n3 CONC more\n1 FAMC @F1@\n2 PEDI BIRTH\n"+
		"1 ASSO @I2@\n2 RELA godfather\n0 @I2@ INDI\n1 CHR\n2 SOUR @S1@\n3 EVEN BIRT\n4 ROLE (Godparent)\n"+
		"0 @F1@ FAM\n1 CHIL @I1@\n0 @N1@ NOTE text\n0 @S1@ SOUR\n1 CHAN\n2 DATE 1 jan 1699/00\n3 TIME 12:00\n0 TRLR\n")
}

func TestGrammar_SubstructurePlacement(t *testing.T) {
	issues := assertGrammar(t, head70+
		"0 @I1@ INDI\n1 PEDI BIRTH\n1 NAME A /B/\n2 FOO bar\n0 FOO\n0 TRLR\n",
		"UNEXPECTED_SUBSTRUCTURE L5 PEDI",
		"UNEXPECTED_SUBSTRUCTURE L7 FOO",
		"UNEXPECTED_SUBSTRUCTURE L8 FOO")
	if got := issues[0].Details["superstructure"]; got != "INDI" {
		t.Errorf("superstructure = %q, want INDI", got)
	}
	if issues[0].Severity != SeverityError || issues[0].RecordXRef != "@I1@" {
		t.Errorf("issue = %+v, want an Error on @I1@", issues[0])
	}
	if _, ok := issues[2].Details["superstructure"]; ok {
		t.Errorf("a level-0 record has no superstructure: %v", issues[2].Details)
	}
}

func TestGrammar_VersionForeignStructures(t *testing.T) {
	issues := assertGrammar(t, head551+
		"0 @I1@ INDI\n1 EXID 123\n1 UID abc\n1 BIRT\n2 SDATE 1900\n1 CREA\n0 @N1@ SNOTE text\n0 TRLR\n",
		"VERSION_FOREIGN_STRUCTURE L11 EXID",
		"VERSION_FOREIGN_STRUCTURE L12 UID",
		"VERSION_FOREIGN_STRUCTURE L14 SDATE",
		"VERSION_FOREIGN_STRUCTURE L15 CREA",
		"VERSION_FOREIGN_STRUCTURE L16 SNOTE")
	if got := issues[0].Details["defined_in"]; got != "7.0" {
		t.Errorf("defined_in = %q, want 7.0", got)
	}
	// CONC is 5.5.x only: in 7.0 it is foreign, and CONT is accepted anywhere.
	issues = assertGrammar(t, head70+"0 @I1@ INDI\n1 NOTE a\n2 CONT b\n2 CONC c\n0 TRLR\n",
		"VERSION_FOREIGN_STRUCTURE L7 CONC")
	if got := issues[0].Details["defined_in"]; got != "5.5" {
		t.Errorf("defined_in = %q, want 5.5", got)
	}
}

func TestGrammar_Cardinality(t *testing.T) {
	issues := assertGrammar(t, head70+
		"0 @I1@ INDI\n1 SEX M\n1 SEX F\n1 SEX U\n1 BIRT\n2 DATE 1900\n2 DATE 1901\n0 TRLR\n",
		"CARDINALITY_EXCEEDED L6 SEX",
		"CARDINALITY_EXCEEDED L7 SEX",
		"CARDINALITY_EXCEEDED L10 DATE")
	if got := issues[0].Details["cardinality"]; got != "{0:1}" {
		t.Errorf("cardinality = %q, want {0:1}", got)
	}
	// A second GEDC in the header.
	assertGrammar(t, head70+"1 GEDC\n2 VERS 7.0\n0 TRLR\n", "CARDINALITY_EXCEEDED L4 GEDC")
}

func TestGrammar_RequiredSubstructures(t *testing.T) {
	issues := assertGrammar(t, "0 HEAD\n1 GEDC\n1 SCHMA\n0 @U1@ SUBM\n0 @I1@ INDI\n1 BAPL\n2 STAT CHILD\n1 EXID 1\n2 TYPE https://x\n0 TRLR\n",
		"MISSING_REQUIRED_SUBSTRUCTURE L2 VERS",
		"MISSING_REQUIRED_SUBSTRUCTURE L4 NAME",
		"MISSING_REQUIRED_SUBSTRUCTURE L7 DATE")
	if got := issues[0].Details["superstructure"]; got != "GEDC" {
		t.Errorf("superstructure = %q, want GEDC", got)
	}
	// 5.5.1 requires HEAD.SOUR, CHAR and GEDC.FORM; HEAD.SUBM is left to
	// MISSING_SUBM, and a missing header line is reported on line 0.
	assertGrammar(t, "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @I1@ INDI\n0 TRLR\n",
		"MISSING_REQUIRED_SUBSTRUCTURE L2 FORM",
		"MISSING_REQUIRED_SUBSTRUCTURE L0 CHAR",
		"MISSING_REQUIRED_SUBSTRUCTURE L0 SOUR")
}

func TestGrammar_Enumerations(t *testing.T) {
	issues := assertGrammar(t, head70+
		"0 @I1@ INDI\n1 SEX Z\n1 FAMC @F1@\n2 PEDI birth\n1 RESN CONFIDENTIAL, BOGUS\n1 NAME A /B/\n2 TYPE _NICK\n1 BIRT\n2 DATE 1900\n0 @F1@ FAM\n0 TRLR\n",
		"INVALID_ENUM_VALUE L5 SEX",
		"INVALID_ENUM_VALUE L7 PEDI",
		"INVALID_ENUM_VALUE L8 RESN")
	if got := issues[0].Details["value"]; got != "Z" {
		t.Errorf("value = %q, want Z", got)
	}
	if got := issues[0].Details["allowed"]; got != "F,M,U,X" {
		t.Errorf("allowed = %q, want F,M,U,X", got)
	}
	if got := issues[2].Details["value"]; got != "BOGUS" {
		t.Errorf("list value = %q, want BOGUS", got)
	}
	// 5.5.1: case-insensitive, and its own vocabulary (no X, no CLERGY).
	assertGrammar(t, head551+
		"0 @I1@ INDI\n1 SEX X\n1 FAMC @F1@\n2 PEDI Adopted\n2 STAT BOGUS\n1 CHR\n2 SOUR @S1@\n3 EVEN BIRT\n4 ROLE CLERGY\n0 @F1@ FAM\n0 @S1@ SOUR\n0 TRLR\n",
		"INVALID_ENUM_VALUE L11 SEX",
		"INVALID_ENUM_VALUE L14 STAT",
		"INVALID_ENUM_VALUE L18 ROLE")
}

func TestGrammar_Payloads(t *testing.T) {
	issues := assertGrammar(t, head70+
		"1 DATE 1 JAN 2020\n2 TIME 25:99\n1 LANG en_US\n"+
		"0 @I1@ INDI\n1 BIRT Yes\n2 AGE 5 years\n1 CHAN\n2 DATE ABT 2020\n1 FAMC I1\n1 NCHI two\n1 FAMS @f1@\n"+
		"1 DEAT\n2 DATE 32 JAN 2000\n2 AGE\n0 TRLR\n",
		"INVALID_PAYLOAD L5 TIME",
		"INVALID_PAYLOAD L6 LANG",
		"INVALID_PAYLOAD L8 BIRT",
		"INVALID_PAYLOAD L9 AGE",
		"INVALID_PAYLOAD L11 DATE",
		"INVALID_PAYLOAD L12 FAMC",
		"INVALID_PAYLOAD L13 NCHI",
		"INVALID_PAYLOAD L14 FAMS")
	want := map[string]string{"TIME": "time", "LANG": "language tag", "BIRT": "Y or empty", "AGE": "age",
		"DATE": "exact date", "FAMC": "pointer", "NCHI": "integer", "FAMS": "pointer"}
	for _, i := range issues {
		if got := i.Details["datatype"]; got != want[i.Details["tag"]] {
			t.Errorf("%s datatype = %q, want %q", i.Details["tag"], got, want[i.Details["tag"]])
		}
	}
	// An event DATE (not DateExact) is left to INVALID_DATE, and an empty
	// AGE is permitted.
	// 5.5.1: AGE_AT_EVENT writes the bound against the number; REPO may be
	// empty; a dual year is part of DATE_EXACT.
	assertGrammar(t, head551+
		"0 @I1@ INDI\n1 BIRT\n2 AGE < 8y\n2 DATE 32 FOO 1900\n1 DEAT y\n0 @S1@ SOUR\n1 REPO\n1 CHAN\n2 DATE 1 JAN 1900\n3 TIME 1:02:03Z\n0 TRLR\n",
		"INVALID_PAYLOAD L12 AGE",
		"INVALID_PAYLOAD L19 TIME")
}

func TestGrammar_RecordXRefs(t *testing.T) {
	assertGrammar(t, head70+
		"0 @i1@ INDI\n0 @I-1@ INDI\n0 @VOID@ INDI\n0 INDI\n0 @I_1@ INDI\n0 TRLR\n",
		"INVALID_RECORD_XREF L4 INDI",
		"INVALID_RECORD_XREF L5 INDI",
		"INVALID_RECORD_XREF L6 INDI")
	// 5.5.1 requires an identifier and has no alphabet or @VOID@ rule here.
	assertGrammar(t, head551+"0 INDI\n0 @i-1@ INDI\n0 TRLR\n", "INVALID_RECORD_XREF L10 INDI")
}

func TestGrammar_Deprecated(t *testing.T) {
	issues := assertGrammar(t, head70+
		"1 SOUR x\n2 DATA y\n0 @I1@ INDI\n1 RESI\n2 ADDR a\n3 ADR1 a\n1 BAPL\n2 STAT COMPLETED\n3 DATE 1 JAN 2000\n1 EXID 1\n0 TRLR\n",
		"DEPRECATED_STRUCTURE L5 DATA",
		"DEPRECATED_STRUCTURE L9 ADR1",
		"DEPRECATED_STRUCTURE L11 STAT",
		"DEPRECATED_STRUCTURE L13 EXID")
	for _, i := range issues {
		if i.Severity != SeverityWarning {
			t.Errorf("%s severity = %v, want Warning", i.Details["tag"], i.Severity)
		}
	}
	// 5.5.1 deprecates nothing: ADR1 is current there.
	assertGrammar(t, head551+"0 @R1@ REPO\n1 NAME r\n1 ADDR a\n2 ADR1 a\n0 TRLR\n")
}

func TestGrammar_Continuations(t *testing.T) {
	// 5.5 BLOB requires CONT, which counts toward its cardinality.
	head55 := "0 HEAD\n1 SOUR t\n1 SUBM @U1@\n1 GEDC\n2 VERS 5.5\n2 FORM LINEAGE-LINKED\n1 CHAR ASCII\n0 @U1@ SUBM\n1 NAME n\n"
	assertGrammar(t, head55+"0 @O1@ OBJE\n1 FORM bmp\n1 BLOB\n2 CONT abc\n0 TRLR\n")
	assertGrammar(t, head55+"0 @O1@ OBJE\n1 FORM bmp\n1 BLOB\n0 TRLR\n", "MISSING_REQUIRED_SUBSTRUCTURE L12 CONT")
	// CONC is accepted under any 5.5.x structure, listed or not.
	assertGrammar(t, head551+"0 @I1@ INDI\n1 NAME a\n2 CONC b\n2 CONT c\n0 TRLR\n")
}

func TestGrammar_NotChecked(t *testing.T) {
	if got := validateGrammar(nil); got != nil {
		t.Errorf("nil document: %v", got)
	}
	if got := validateGrammar(&gedcom.Document{}); got != nil {
		t.Errorf("no header: %v", got)
	}
	// A version the tables do not cover, and records built in memory with
	// no Tags (the encoder writes them from their entities).
	doc := &gedcom.Document{
		Header:  &gedcom.Header{Version: "4.0", Tags: []*gedcom.Tag{{Level: 1, Tag: "BOGUS"}}},
		Records: []*gedcom.Record{{XRef: "@I1@", Type: gedcom.RecordTypeIndividual}},
	}
	if got := validateGrammar(doc); got != nil {
		t.Errorf("unknown version: %v", got)
	}
	// A declared version the tables do not cover is not checked against the
	// decoder's guess.
	if got := validateGrammar(decodeGrammarDoc(t, "0 HEAD\n1 GEDC\n2 VERS 7.1\n1 TITL t\n0 @I1@ INDI\n1 SEX Q\n0 TRLR\n")); got != nil {
		t.Errorf("7.1 document: %v", got)
	}
	doc.Header = &gedcom.Header{Version: gedcom.Version70}
	doc.Records = append(doc.Records, nil, &gedcom.Record{XRef: "@I2@", Type: gedcom.RecordTypeIndividual,
		Tags: []*gedcom.Tag{nil, {Level: 1, Tag: "SEX", Value: "Q", LineNumber: 9}}})
	if got := grammarFindings(validateGrammar(doc)); !slices.Equal(got, []string{"INVALID_ENUM_VALUE L9 SEX"}) {
		t.Errorf("in-memory document: %q", got)
	}
}

func TestGrammar_PicksAlternativeByPayload(t *testing.T) {
	// 5.5.1 NOTE is a pointer or text: each is counted against its own
	// alternative, and a pointer-shaped OBJE link must not carry FILE.
	assertGrammar(t, head551+
		"0 @I1@ INDI\n1 OBJE @O1@\n2 FILE x\n1 OBJE\n2 FILE x.jpg\n3 FORM jpg\n0 @O1@ OBJE\n1 FILE y\n2 FORM bmp\n0 TRLR\n",
		"UNEXPECTED_SUBSTRUCTURE L12 FILE")
}

func TestGrammar_OptInThroughValidatorAndQualityReport(t *testing.T) {
	doc := decodeGrammarDoc(t, head70+"0 @I1@ INDI\n1 SEX Z\n0 TRLR\n")
	hasGrammar := func(issues []Issue) bool {
		return slices.ContainsFunc(issues, func(i Issue) bool { return slices.Contains(grammarCodes, i.Code) })
	}

	if hasGrammar(New().ValidateAll(doc)) {
		t.Error("ValidateAll ran the grammar check without CheckGrammar")
	}
	if r := New().QualityReport(doc); len(r.GrammarIssues) != 0 || hasGrammar(r.Errors) {
		t.Errorf("QualityReport ran the grammar check without CheckGrammar: %v", r.GrammarIssues)
	}

	v := NewWithOptions(&ValidateOptions{CheckGrammar: true})
	issues := v.ValidateAll(doc)
	if !slices.ContainsFunc(issues, func(i Issue) bool { return i.Code == CodeInvalidEnumValue && i.LineNumber == 5 }) {
		t.Errorf("ValidateAll with CheckGrammar: no INVALID_ENUM_VALUE on line 5 in %v", issues)
	}
	report := v.QualityReport(doc)
	if got := grammarFindings(report.GrammarIssues); !slices.Equal(got, []string{"INVALID_ENUM_VALUE L5 SEX"}) {
		t.Errorf("GrammarIssues = %q", got)
	}
	if report.ErrorCount == 0 || len(report.IssuesByCode(CodeInvalidEnumValue)) != 1 {
		t.Errorf("report does not count the grammar issue: errors=%d", report.ErrorCount)
	}
	if !strings.Contains(report.String(), CodeInvalidEnumValue) {
		t.Errorf("report summary omits the grammar code:\n%s", report.String())
	}

	skipped := NewWithOptions(&ValidateOptions{CheckGrammar: true, SkipRules: []string{CodeInvalidEnumValue}})
	if hasGrammar(skipped.ValidateAll(doc)) {
		t.Error("SkipRules did not drop the grammar code")
	}
}
