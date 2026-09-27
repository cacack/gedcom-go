package converter_test

import (
	"bytes"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/converter"
	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// GEDCOM 5.5/5.5.1 has no negative assertion (n NO <EVENT>). A downgrade has
// always reported NO as data loss; these tests pin that it is also actually
// dropped, so the output is valid 5.x and the report tells the truth.

const negativeAssertion70 = `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME John /Doe/
1 BIRT
2 DATE 1 JAN 1920
1 NO DEAT
2 DATE FROM 1900 TO 1910
3 PHRASE Not in the 1900s
2 NOTE Checked the parish burials
1 FAMS @F1@
0 @I2@ INDI
1 NAME Jane /Doe/
1 FAMS @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 NO MARR
2 DATE FROM 1800 TO 1850
1 DIV
0 TRLR
`

var noLine = regexp.MustCompile(`(?m)^\d+ NO( |$)`)

func decodeString(t *testing.T, text string) *gedcom.Document {
	t.Helper()
	doc, err := decoder.Decode(strings.NewReader(text))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	return doc
}

func encodeString(t *testing.T, doc *gedcom.Document) string {
	t.Helper()
	var buf bytes.Buffer
	if err := encoder.Encode(&buf, doc); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	return buf.String()
}

func noDataLoss(report *gedcom.ConversionReport) *gedcom.DataLossItem {
	for i := range report.DataLoss {
		if report.DataLoss[i].Feature == "NO tags" {
			return &report.DataLoss[i]
		}
	}
	return nil
}

// assertNegativeAssertionsDropped converts doc to target and checks the output
// carries no NO line, re-decodes without negated events, and that the report
// names exactly wantXRefs as losing a NO.
func assertNegativeAssertionsDropped(t *testing.T, doc *gedcom.Document, target gedcom.Version, wantXRefs []string, allowOtherDiagnostics bool) *gedcom.Document {
	t.Helper()
	converted, report, err := converter.Convert(doc, target)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	for _, ind := range converted.Individuals() {
		if n := len(ind.NegativeAssertions()); n != 0 {
			t.Errorf("converted %s still has %d negative assertion(s)", ind.XRef, n)
		}
	}
	for _, fam := range converted.Families() {
		if n := len(fam.NegativeAssertions()); n != 0 {
			t.Errorf("converted %s still has %d negative assertion(s)", fam.XRef, n)
		}
	}

	out := encodeString(t, converted)
	if m := noLine.FindAllString(out, -1); len(m) != 0 {
		t.Errorf("GEDCOM %s output contains NO lines %q:\n%s", target, m, out)
	}
	for _, orphan := range []string{"Not in the 1900s", "Checked the parish burials", "FROM 1900 TO 1910", "FROM 1800 TO 1850"} {
		if strings.Contains(out, orphan) {
			t.Errorf("output kept %q from a dropped NO block", orphan)
		}
	}

	result, err := decoder.DecodeWithDiagnostics(strings.NewReader(out), nil)
	if err != nil {
		t.Fatalf("re-Decode() error = %v\n%s", err, out)
	}
	// maximal70 carries other 7.0-only structures the converter reports but
	// keeps (UID, CREA, ...), so only the snippets must re-decode without any
	// diagnostic; every document must be free of one about NO.
	for _, d := range result.Diagnostics {
		if !allowOtherDiagnostics || strings.Contains(d.Message, "tag: NO") {
			t.Errorf("re-Decode() diagnostic: %s", d.Message)
		}
	}
	redecoded := result.Document
	if redecoded.Header.Version != target {
		t.Errorf("re-decoded version = %s, want %s", redecoded.Header.Version, target)
	}
	for _, ind := range redecoded.Individuals() {
		for _, e := range ind.Events {
			if e.IsNegative {
				t.Errorf("re-decoded %s has negated %s", ind.XRef, e.Type)
			}
		}
	}
	for _, fam := range redecoded.Families() {
		for _, e := range fam.Events {
			if e.IsNegative {
				t.Errorf("re-decoded %s has negated %s", fam.XRef, e.Type)
			}
		}
	}

	loss := noDataLoss(report)
	if loss == nil {
		t.Fatalf("report has no \"NO tags\" data loss; DataLoss = %+v", report.DataLoss)
	}
	got := slices.Sorted(slices.Values(loss.AffectedRecords))
	want := slices.Sorted(slices.Values(wantXRefs))
	if !slices.Equal(got, want) {
		t.Errorf("NO tags AffectedRecords = %v, want %v", got, want)
	}
	var dropped []string
	for _, note := range report.Dropped {
		if note.Original == "NO" {
			dropped = append(dropped, note.Path)
		}
	}
	if len(dropped) != len(wantXRefs) {
		t.Errorf("Dropped NO notes = %v, want one per record in %v", dropped, wantXRefs)
	}
	return redecoded
}

func TestConvert_DropsNegativeAssertions_TagsPath(t *testing.T) {
	for _, target := range []gedcom.Version{gedcom.Version551, gedcom.Version55} {
		t.Run(target.String(), func(t *testing.T) {
			doc := decodeString(t, negativeAssertion70)
			redecoded := assertNegativeAssertionsDropped(t, doc, target, []string{"@I1@", "@F1@"}, false)

			// Everything around the NO blocks survives.
			john := redecoded.GetIndividual("@I1@")
			if john == nil || john.BirthDate() == nil || john.BirthDate().Year != 1920 {
				t.Errorf("@I1@ lost its BIRT")
			}
			if john != nil && len(john.SpouseInFamilies) != 1 {
				t.Errorf("@I1@ FAMS = %v, want 1 link", john.SpouseInFamilies)
			}
			fam := redecoded.GetFamily("@F1@")
			if fam == nil || fam.Husband != "@I1@" || fam.Wife != "@I2@" {
				t.Fatalf("@F1@ lost its spouses: %+v", fam)
			}
			if len(fam.Events) != 1 || fam.Events[0].Type != gedcom.EventDivorce {
				t.Errorf("@F1@ events = %v, want just the DIV", fam.Events)
			}

			// The caller's document is untouched.
			if n := len(doc.GetIndividual("@I1@").NegativeAssertions()); n != 1 {
				t.Errorf("source @I1@ negative assertions = %d, want 1 (Convert must not mutate its input)", n)
			}
		})
	}
}

// TestConvert_DropsNegativeAssertions_EntityPath covers a document built in
// code: no raw Tags, so the encoder writes from Individual/Family.Events.
func TestConvert_DropsNegativeAssertions_EntityPath(t *testing.T) {
	ind := &gedcom.Individual{
		XRef: "@I1@",
		Events: []*gedcom.Event{
			{Type: gedcom.EventBirth, Date: "1 JAN 1920"},
			nil,
			{Type: gedcom.EventDeath, Date: "FROM 1900 TO 1910", IsNegative: true},
		},
	}
	fam := &gedcom.Family{
		XRef:    "@F1@",
		Husband: "@I1@",
		Events:  []*gedcom.Event{{Type: gedcom.EventMarriage, Date: "FROM 1800 TO 1850", IsNegative: true}},
	}
	plain := &gedcom.Individual{XRef: "@I2@", Events: []*gedcom.Event{{Type: gedcom.EventBirth}}}
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version70},
		Records: []*gedcom.Record{
			{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: ind},
			{XRef: "@I2@", Type: gedcom.RecordTypeIndividual, Entity: plain},
			{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: fam},
			{XRef: "@N1@", Type: gedcom.RecordTypeNote, Entity: &gedcom.Note{XRef: "@N1@", Text: "n"}},
		},
	}
	doc.XRefMap = map[string]*gedcom.Record{}
	for _, r := range doc.Records {
		doc.XRefMap[r.XRef] = r
	}

	redecoded := assertNegativeAssertionsDropped(t, doc, gedcom.Version551, []string{"@I1@", "@F1@"}, false)
	if got := redecoded.GetIndividual("@I1@"); got == nil || got.BirthEvent() == nil || got.DeathEvent() != nil {
		t.Errorf("re-decoded @I1@ = %+v, want its BIRT and no DEAT", got)
	}

	converted, _, err := converter.Convert(doc, gedcom.Version551)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	// A nil entry is the caller's element, not a negative assertion.
	if got := converted.GetIndividual("@I1@").Events; len(got) != 2 || got[1] != nil {
		t.Errorf("converted @I1@ Events = %v, want [BIRT, nil]", got)
	}
}

// TestConvert_DropsNegativeAssertions_KeepsNilTags pins that a nil inside a
// dropped NO block stays, matching how the EXID rewrite treats one.
func TestConvert_DropsNegativeAssertions_KeepsNilTags(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version70},
		Records: []*gedcom.Record{{
			XRef: "@I1@",
			Type: gedcom.RecordTypeIndividual,
			Tags: []*gedcom.Tag{
				{Level: 1, Tag: "NO", Value: "DEAT"},
				nil,
				{Level: 2, Tag: "DATE", Value: "FROM 1900 TO 1910"},
				{Level: 1, Tag: "SEX", Value: "M"},
			},
		}},
	}
	doc.XRefMap = map[string]*gedcom.Record{"@I1@": doc.Records[0]}

	converted, report, err := converter.Convert(doc, gedcom.Version551)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	tags := converted.Records[0].Tags
	if len(tags) != 2 || tags[0] != nil || tags[1] == nil || tags[1].Tag != "SEX" {
		t.Errorf("Tags after drop = %v, want [nil, SEX]", tags)
	}
	if noDataLoss(report) == nil {
		t.Error("NO block dropped from Tags was not reported as data loss")
	}
}

func TestConvert_DropsNegativeAssertions_Maximal70(t *testing.T) {
	data, err := os.ReadFile("../testdata/gedcom-7.0/maximal70.ged")
	if err != nil {
		t.Fatalf("read maximal70.ged: %v", err)
	}
	for _, target := range []gedcom.Version{gedcom.Version551, gedcom.Version55} {
		t.Run(target.String(), func(t *testing.T) {
			doc := decodeString(t, string(data))
			redecoded := assertNegativeAssertionsDropped(t, doc, target, []string{"@I1@", "@F1@"}, true)

			// maximal70's positive NATU and EMIG on @I1@ are unaffected.
			var types []gedcom.EventType
			for _, e := range redecoded.GetIndividual("@I1@").Events {
				if e.Type == gedcom.EventNaturalization || e.Type == gedcom.EventEmigration {
					types = append(types, e.Type)
				}
			}
			if len(types) != 2 {
				t.Errorf("@I1@ positive NATU/EMIG after conversion = %v, want both", types)
			}
		})
	}
}

// TestConvert_NoNegativeAssertions_NoReport pins that a 7.0 document without
// any NO reports none.
func TestConvert_NoNegativeAssertions_NoReport(t *testing.T) {
	doc := decodeString(t, strings.ReplaceAll(negativeAssertion70, "1 NO ", "1 "))
	_, report, err := converter.Convert(doc, gedcom.Version551)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if loss := noDataLoss(report); loss != nil {
		t.Errorf("unexpected NO data loss %+v", loss)
	}
}
