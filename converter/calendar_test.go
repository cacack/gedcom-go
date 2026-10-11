package converter_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/converter"
	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// calendarDoc builds a document whose single individual carries one BIRT event
// dated value, held both in Record.Tags and in the typed event.
func calendarDoc(version gedcom.Version, value string) *gedcom.Document {
	parsed, _ := gedcom.ParseDate(value)
	record := &gedcom.Record{
		XRef: "@I1@",
		Type: gedcom.RecordTypeIndividual,
		Tags: []*gedcom.Tag{
			{Level: 1, Tag: "BIRT"},
			{Level: 2, Tag: "DATE", Value: value},
		},
		Entity: &gedcom.Individual{
			XRef:   "@I1@",
			Events: []*gedcom.Event{{Type: gedcom.EventBirth, Date: value, ParsedDate: parsed}},
		},
	}
	return &gedcom.Document{
		Header:  &gedcom.Header{Version: version},
		Records: []*gedcom.Record{record},
		XRefMap: map[string]*gedcom.Record{"@I1@": record},
	}
}

// convertedBirthDate converts a calendarDoc and returns the raw DATE tag value
// and the typed event, failing the test if either is missing.
func convertedBirthDate(t *testing.T, doc *gedcom.Document, target gedcom.Version) (string, *gedcom.Event, *gedcom.ConversionReport) {
	t.Helper()
	got, report, err := converter.Convert(doc, target)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	record := got.GetRecord("@I1@")
	ind, _ := record.GetIndividual()
	return record.Tags[1].Value, ind.Events[0], report
}

func TestConvertDateCalendarSyntax(t *testing.T) {
	tests := []struct {
		name   string
		from   gedcom.Version
		to     gedcom.Version
		value  string
		expect string
	}{
		{"gregorian to 7.0", gedcom.Version551, gedcom.Version70, "@#DGREGORIAN@ 1 JAN 1700", "GREGORIAN 1 JAN 1700"},
		{"julian to 7.0", gedcom.Version551, gedcom.Version70, "@#DJULIAN@ 1 JAN 1700", "JULIAN 1 JAN 1700"},
		{"hebrew to 7.0", gedcom.Version551, gedcom.Version70, "@#DHEBREW@ 1 TSH 5500", "HEBREW 1 TSH 5500"},
		{"french to 7.0", gedcom.Version551, gedcom.Version70, "@#DFRENCH R@ 1 VEND 10", "FRENCH_R 1 VEND 10"},
		{"5.5 source to 7.0", gedcom.Version55, gedcom.Version70, "@#DJULIAN@ 1700", "JULIAN 1700"},
		{"range to 7.0", gedcom.Version551, gedcom.Version70, "BET @#DJULIAN@ 1700 AND @#DGREGORIAN@ 1710", "BET JULIAN 1700 AND GREGORIAN 1710"},
		{"period to 7.0", gedcom.Version551, gedcom.Version70, "FROM @#DHEBREW@ 5500 TO @#DFRENCH R@ 10", "FROM HEBREW 5500 TO FRENCH_R 10"},
		{"approximation to 7.0", gedcom.Version551, gedcom.Version70, "ABT @#DJULIAN@ MAR 1066", "ABT JULIAN MAR 1066"},
		{"phrase text untouched to 7.0", gedcom.Version551, gedcom.Version70, "INT @#DJULIAN@ 1700 (@#DJULIAN@ by the family)", "INT JULIAN 1700 (@#DJULIAN@ by the family)"},
		{"gregorian to 5.5.1", gedcom.Version70, gedcom.Version551, "GREGORIAN 1 JAN 1700", "@#DGREGORIAN@ 1 JAN 1700"},
		{"julian to 5.5.1", gedcom.Version70, gedcom.Version551, "JULIAN 1 JAN 1700", "@#DJULIAN@ 1 JAN 1700"},
		{"hebrew to 5.5", gedcom.Version70, gedcom.Version55, "HEBREW 1 TSH 5500", "@#DHEBREW@ 1 TSH 5500"},
		{"french to 5.5.1", gedcom.Version70, gedcom.Version551, "FRENCH_R 1 VEND 10", "@#DFRENCH R@ 1 VEND 10"},
		{"range to 5.5.1", gedcom.Version70, gedcom.Version551, "BET JULIAN 1700 AND GREGORIAN 1710", "BET @#DJULIAN@ 1700 AND @#DGREGORIAN@ 1710"},
		{"period to 5.5.1", gedcom.Version70, gedcom.Version551, "FROM HEBREW 5500 TO FRENCH_R 10", "FROM @#DHEBREW@ 5500 TO @#DFRENCH R@ 10"},
		{"approximation to 5.5.1", gedcom.Version70, gedcom.Version551, "BEF JULIAN 1066", "BEF @#DJULIAN@ 1066"},
		{"phrase text untouched to 5.5.1", gedcom.Version70, gedcom.Version551, "INT JULIAN 1700 (JULIAN by the family)", "INT @#DJULIAN@ 1700 (JULIAN by the family)"},
		{"no calendar unchanged", gedcom.Version551, gedcom.Version70, "BET 1700 AND 1710", "BET 1700 AND 1710"},
		{"unterminated escape unchanged", gedcom.Version551, gedcom.Version70, "@#DJULIAN 1700", "@#DJULIAN 1700"},
		{"unclosed phrase untouched", gedcom.Version70, gedcom.Version551, "INT JULIAN 1700 (JULIAN by", "INT @#DJULIAN@ 1700 (JULIAN by"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, event, report := convertedBirthDate(t, calendarDoc(tt.from, tt.value), tt.to)
			if raw != tt.expect {
				t.Errorf("raw DATE = %q, want %q", raw, tt.expect)
			}
			if event.Date != tt.expect {
				t.Errorf("Event.Date = %q, want %q", event.Date, tt.expect)
			}
			if event.ParsedDate != nil && event.ParsedDate.Original != tt.expect {
				t.Errorf("ParsedDate.Original out of step with Event.Date %q: %+v", tt.expect, event.ParsedDate)
			}
			changed := tt.value != tt.expect
			if hasTransformation(report, "DATE_CALENDAR_CONVERTED") != changed {
				t.Errorf("DATE_CALENDAR_CONVERTED reported = %v, want %v", !changed, changed)
			}
			if changed && !hasNote(report.Normalized, tt.value, tt.expect) {
				t.Errorf("no Normalized note %q -> %q in %+v", tt.value, tt.expect, report.Normalized)
			}
		})
	}
}

func TestConvertDateUnknownCalendarReported(t *testing.T) {
	tests := []struct {
		name  string
		from  gedcom.Version
		to    gedcom.Version
		value string
	}{
		{"roman escape to 7.0", gedcom.Version551, gedcom.Version70, "@#DROMAN@ 1 JAN 100"},
		{"unknown escape to 7.0", gedcom.Version551, gedcom.Version70, "@#DUNKNOWN@ 1700"},
		{"extension calendar to 5.5.1", gedcom.Version70, gedcom.Version551, "_MAYAN 1 JAN 100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, event, report := convertedBirthDate(t, calendarDoc(tt.from, tt.value), tt.to)
			if raw != tt.value || event.Date != tt.value {
				t.Errorf("unknown calendar rewritten: raw %q, Event.Date %q, want %q", raw, event.Date, tt.value)
			}
			if len(report.Preserved) != 1 || report.Preserved[0].Path != "Individual @I1@ > BIRT > DATE" ||
				!hasNote(report.Preserved, tt.value, tt.value) {
				t.Errorf("want one Preserved note for %q at the BIRT DATE, got %+v", tt.value, report.Preserved)
			}
			if hasTransformation(report, "DATE_CALENDAR_CONVERTED") {
				t.Error("unknown calendar counted as converted")
			}
		})
	}
}

// TestConvertDateCalendarEntityOnly covers a document built in code, whose
// typed event is the encoder's only write path.
func TestConvertDateCalendarEntityOnly(t *testing.T) {
	doc := calendarDoc(gedcom.Version551, "@#DJULIAN@ 1700")
	doc.Records[0].Tags = nil

	got, report, err := converter.Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	ind, _ := got.GetRecord("@I1@").GetIndividual()
	if ind.Events[0].Date != "JULIAN 1700" {
		t.Errorf("Event.Date = %q, want %q", ind.Events[0].Date, "JULIAN 1700")
	}
	if !hasNote(report.Normalized, "@#DJULIAN@ 1700", "JULIAN 1700") {
		t.Errorf("no Normalized note for the entity date in %+v", report.Normalized)
	}
	if orig, _ := doc.GetRecord("@I1@").GetIndividual(); orig.Events[0].Date != "@#DJULIAN@ 1700" {
		t.Errorf("source document mutated: %q", orig.Events[0].Date)
	}
}

// TestConvertDateCalendarFamilyEntity covers the family-side typed dates:
// events, attributes and LDS ordinances, with SDATE alongside DATE.
func TestConvertDateCalendarFamilyEntity(t *testing.T) {
	fam := &gedcom.Family{
		XRef:          "@F1@",
		Events:        []*gedcom.Event{{Type: gedcom.EventMarriage, Date: "JULIAN 1700", SortDate: "JULIAN 1700"}},
		Attributes:    []*gedcom.Attribute{{Type: gedcom.AttributeNumberOfChildren, Value: "2", Date: "HEBREW 5500"}},
		LDSOrdinances: []*gedcom.LDSOrdinance{{Type: gedcom.LDSSealingSpouse, Date: "FRENCH_R 10"}},
	}
	record := &gedcom.Record{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: fam}
	doc := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version70},
		Records: []*gedcom.Record{record},
		XRefMap: map[string]*gedcom.Record{"@F1@": record},
	}

	got, report, err := converter.Convert(doc, gedcom.Version551)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	out, _ := got.GetRecord("@F1@").GetFamily()
	for name, pair := range map[string][2]string{
		"event DATE":     {out.Events[0].Date, "@#DJULIAN@ 1700"},
		"event SDATE":    {out.Events[0].SortDate, "@#DJULIAN@ 1700"},
		"attribute DATE": {out.Attributes[0].Date, "@#DHEBREW@ 5500"},
		"ordinance DATE": {out.LDSOrdinances[0].Date, "@#DFRENCH R@ 10"},
	} {
		if pair[0] != pair[1] {
			t.Errorf("%s = %q, want %q", name, pair[0], pair[1])
		}
	}
	assertNotePaths(t, report.Normalized, []string{
		"Family @F1@ > MARR > DATE",
		"Family @F1@ > MARR > SDATE",
		"Family @F1@ > NCHI > DATE",
		"Family @F1@ > SLGS > DATE",
	})
}

// TestConvertDateCalendarCitationEntity covers the typed citation DATA.DATE
// values of a document built in code, which the encoder writes from the entity.
func TestConvertDateCalendarCitationEntity(t *testing.T) {
	cite := func() []*gedcom.SourceCitation {
		return []*gedcom.SourceCitation{{SourceXRef: "@S1@", Data: &gedcom.SourceCitationData{Date: "@#DJULIAN@ 1700"}}}
	}
	ind := &gedcom.Individual{
		XRef:            "@I1@",
		SourceCitations: cite(),
		Associations:    []*gedcom.Association{{IndividualXRef: "@I2@", SourceCitations: cite()}},
		Events:          []*gedcom.Event{{Type: gedcom.EventBirth, SourceCitations: cite()}},
		Attributes:      []*gedcom.Attribute{{Type: gedcom.AttributeOccupation, Value: "Smith", SourceCitations: cite()}},
	}
	fam := &gedcom.Family{XRef: "@F1@", SourceCitations: cite()}
	media := &gedcom.MediaObject{XRef: "@M1@", SourceCitations: cite()}
	snote := &gedcom.SharedNote{XRef: "@N1@", Text: "note", SourceCitations: cite()}
	records := []*gedcom.Record{
		{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: ind},
		{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: fam},
		{XRef: "@M1@", Type: gedcom.RecordTypeMedia, Entity: media},
		{XRef: "@N1@", Type: gedcom.RecordTypeSharedNote, Entity: snote},
	}
	doc := &gedcom.Document{Header: &gedcom.Header{Version: gedcom.Version551}, Records: records, XRefMap: map[string]*gedcom.Record{}}
	for _, r := range records {
		doc.XRefMap[r.XRef] = r
	}

	got, report, err := converter.Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	outInd, _ := got.GetRecord("@I1@").GetIndividual()
	outFam, _ := got.GetRecord("@F1@").GetFamily()
	outMedia, _ := got.GetRecord("@M1@").GetMediaObject()
	outNote, _ := got.GetRecord("@N1@").GetSharedNote()
	for name, c := range map[string]*gedcom.SourceCitation{
		"individual":  outInd.SourceCitations[0],
		"association": outInd.Associations[0].SourceCitations[0],
		"event":       outInd.Events[0].SourceCitations[0],
		"attribute":   outInd.Attributes[0].SourceCitations[0],
		"family":      outFam.SourceCitations[0],
		"media":       outMedia.SourceCitations[0],
		"shared note": outNote.SourceCitations[0],
	} {
		if c.Data.Date != "JULIAN 1700" {
			t.Errorf("%s citation DATA.DATE = %q, want %q", name, c.Data.Date, "JULIAN 1700")
		}
	}
	assertNotePaths(t, report.Normalized, []string{
		"Individual @I1@ > SOUR > DATA > DATE",
		"Individual @I1@ > ASSO > SOUR > DATA > DATE",
		"Individual @I1@ > BIRT > SOUR > DATA > DATE",
		"Individual @I1@ > OCCU > SOUR > DATA > DATE",
		"Family @F1@ > SOUR > DATA > DATE",
		"MediaObject @M1@ > SOUR > DATA > DATE",
		"SNOTE @N1@ > SOUR > DATA > DATE",
	})
	if text := encode(t, got); strings.Contains(text, "@#D") {
		t.Errorf("7.0 output still carries a 5.5.x calendar escape:\n%s", text)
	}
}

// assertNotePaths fails unless notes carry exactly the paths want, in order.
func assertNotePaths(t *testing.T, notes []gedcom.ConversionNote, want []string) {
	t.Helper()
	got := make([]string, len(notes))
	for i, n := range notes {
		got[i] = n.Path
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("note paths:\ngot  %q\nwant %q", got, want)
	}
}

// calendarRoundTripGED is a 5.5.1 file dating events in each standard
// calendar, plus a range whose endpoints use different calendars.
const calendarRoundTripGED = `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 BIRT
2 DATE @#DGREGORIAN@ 1 JAN 1700
1 CHR
2 DATE @#DJULIAN@ 12 FEB 1700
1 DEAT
2 DATE @#DHEBREW@ 1 TSH 5500
1 BURI
2 DATE @#DFRENCH R@ 1 VEND 10
1 RESI
2 DATE BET @#DJULIAN@ 1700 AND @#DGREGORIAN@ 1710
0 TRLR
`

func TestConvertDateCalendarRoundTrip(t *testing.T) {
	src, err := decoder.Decode(strings.NewReader(calendarRoundTripGED))
	if err != nil {
		t.Fatalf("decode 5.5.1: %v", err)
	}

	v7, _, err := converter.Convert(src, gedcom.Version70)
	if err != nil {
		t.Fatalf("Convert to 7.0: %v", err)
	}
	v7Text := encode(t, v7)
	for _, want := range []string{
		"2 DATE GREGORIAN 1 JAN 1700",
		"2 DATE JULIAN 12 FEB 1700",
		"2 DATE HEBREW 1 TSH 5500",
		"2 DATE FRENCH_R 1 VEND 10",
		"2 DATE BET JULIAN 1700 AND GREGORIAN 1710",
	} {
		if !strings.Contains(v7Text, want) {
			t.Errorf("7.0 output missing %q:\n%s", want, v7Text)
		}
	}
	if strings.Contains(v7Text, "@#D") {
		t.Errorf("7.0 output still carries a 5.5.x calendar escape:\n%s", v7Text)
	}

	v7Doc := decodeClean(t, v7Text, gedcom.Version70)

	back, _, err := converter.Convert(v7Doc, gedcom.Version551)
	if err != nil {
		t.Fatalf("Convert to 5.5.1: %v", err)
	}
	backText := encode(t, back)
	decodeClean(t, backText, gedcom.Version551)

	if got, want := dateLines(backText), dateLines(calendarRoundTripGED); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("round trip changed DATE lines:\ngot  %q\nwant %q", got, want)
	}

	// The typed dates keep their calendar across the round trip.
	ind := back.GetIndividual("@I1@")
	wantCals := []gedcom.Calendar{gedcom.CalendarGregorian, gedcom.CalendarJulian, gedcom.CalendarHebrew, gedcom.CalendarFrenchRepublican, gedcom.CalendarJulian}
	for i, event := range ind.Events {
		if event.ParsedDate == nil || event.ParsedDate.Calendar != wantCals[i] {
			t.Errorf("event %d (%s) calendar = %+v, want %v", i, event.Type, event.ParsedDate, wantCals[i])
		}
	}
	if end := ind.Events[4].ParsedDate.EndDate; end == nil || end.Calendar != gedcom.CalendarGregorian {
		t.Errorf("mixed range end calendar = %+v, want Gregorian", end)
	}
}

func encode(t *testing.T, doc *gedcom.Document) string {
	t.Helper()
	var buf bytes.Buffer
	if err := encoder.Encode(&buf, doc); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.String()
}

// decodeClean decodes text, requiring the declared version and no
// INVALID_VALUE diagnostics.
func decodeClean(t *testing.T, text string, version gedcom.Version) *gedcom.Document {
	t.Helper()
	result, err := decoder.DecodeWithDiagnostics(strings.NewReader(text), decoder.DefaultOptions())
	if err != nil {
		t.Fatalf("decode %s: %v", version, err)
	}
	if result.Document.Header.Version != version {
		t.Errorf("decoded version = %s, want %s", result.Document.Header.Version, version)
	}
	for _, d := range result.Diagnostics {
		if d.Code == decoder.CodeInvalidValue {
			t.Errorf("%s output has INVALID_VALUE diagnostic: %v", version, d)
		}
	}
	return result.Document
}

func dateLines(text string) []string {
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimRight(line, "\r"), "2 DATE ") {
			lines = append(lines, strings.TrimRight(line, "\r"))
		}
	}
	return lines
}

func hasTransformation(report *gedcom.ConversionReport, kind string) bool {
	for _, tr := range report.Transformations {
		if tr.Type == kind {
			return true
		}
	}
	return false
}

func hasNote(notes []gedcom.ConversionNote, original, result string) bool {
	for _, n := range notes {
		if n.Original == original && n.Result == result {
			return true
		}
	}
	return false
}

const calendarHeaderGED = `0 HEAD
1 SOUR X
2 DATA D
3 DATE @#DJULIAN@ 1 JAN 1700
1 DATE @#DGREGORIAN@ 2 FEB 2000
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 TRLR
`

// TestConvertDateCalendarHeader covers header dates, which the encoder writes
// from Header.Tags, converted to 7.0 and back.
func TestConvertDateCalendarHeader(t *testing.T) {
	src, err := decoder.Decode(strings.NewReader(calendarHeaderGED))
	if err != nil {
		t.Fatalf("decode 5.5.1: %v", err)
	}

	v7, report, err := converter.Convert(src, gedcom.Version70)
	if err != nil {
		t.Fatalf("Convert to 7.0: %v", err)
	}
	v7Text := encode(t, v7)
	for _, want := range []string{"3 DATE JULIAN 1 JAN 1700", "1 DATE GREGORIAN 2 FEB 2000"} {
		if !strings.Contains(v7Text, want) {
			t.Errorf("7.0 output missing %q:\n%s", want, v7Text)
		}
	}
	if strings.Contains(v7Text, "@#D") {
		t.Errorf("7.0 output still carries a 5.5.x calendar escape:\n%s", v7Text)
	}
	if v7.Header.Date != "GREGORIAN 2 FEB 2000" {
		t.Errorf("Header.Date = %q, want it kept in step with Tags", v7.Header.Date)
	}
	for _, n := range []gedcom.ConversionNote{
		{Path: "Header > SOUR > DATA > DATE", Original: "@#DJULIAN@ 1 JAN 1700", Result: "JULIAN 1 JAN 1700"},
		{Path: "Header > DATE", Original: "@#DGREGORIAN@ 2 FEB 2000", Result: "GREGORIAN 2 FEB 2000"},
	} {
		if !hasNoteAt(report.Normalized, n) {
			t.Errorf("missing Normalized note %+v in %+v", n, report.Normalized)
		}
	}
	if len(report.Normalized) != 2 || !hasTransformation(report, "DATE_CALENDAR_CONVERTED") {
		t.Errorf("want two Normalized notes and a DATE_CALENDAR_CONVERTED transformation, got %+v / %+v", report.Normalized, report.Transformations)
	}

	back, _, err := converter.Convert(decodeClean(t, v7Text, gedcom.Version70), gedcom.Version551)
	if err != nil {
		t.Fatalf("Convert to 5.5.1: %v", err)
	}
	backText := encode(t, back)
	for _, want := range []string{"3 DATE @#DJULIAN@ 1 JAN 1700", "1 DATE @#DGREGORIAN@ 2 FEB 2000"} {
		if !strings.Contains(backText, want) {
			t.Errorf("5.5.1 output missing %q:\n%s", want, backText)
		}
	}
}

// TestConvertDateCalendarHeaderTypedOnly covers a header built in code, whose
// typed Date is the encoder's only write path.
func TestConvertDateCalendarHeaderTypedOnly(t *testing.T) {
	doc := calendarDoc(gedcom.Version551, "1700")
	doc.Header.Date = "@#DJULIAN@ 1 JAN 1700"

	got, report, err := converter.Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if got.Header.Date != "JULIAN 1 JAN 1700" {
		t.Errorf("Header.Date = %q, want %q", got.Header.Date, "JULIAN 1 JAN 1700")
	}
	want := gedcom.ConversionNote{Path: "Header > DATE", Original: "@#DJULIAN@ 1 JAN 1700", Result: "JULIAN 1 JAN 1700"}
	if len(report.Normalized) != 1 || !hasNoteAt(report.Normalized, want) {
		t.Errorf("want one Normalized note %+v, got %+v", want, report.Normalized)
	}
}

func hasNoteAt(notes []gedcom.ConversionNote, want gedcom.ConversionNote) bool {
	for _, n := range notes {
		if n.Path == want.Path && n.Original == want.Original && n.Result == want.Result {
			return true
		}
	}
	return false
}
