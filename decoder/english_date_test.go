package decoder

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// englishDatesInput is the reproduction file of issue #907.
const englishDatesInput = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
	"0 @I2@ INDI\n1 NAME Mary /Sample/\n1 SEX F\n1 BIRT\n2 DATE June 1880\n" +
	"0 @I3@ INDI\n1 NAME Anna /Sample/\n1 SEX F\n1 BIRT\n2 DATE about 1873\n" +
	"0 TRLR\n"

// TestDecodeEnglishDates covers issue #907: dates spelled out in English are
// read as the GEDCOM dates they mean and reported as NONSTANDARD_DATE.
func TestDecodeEnglishDates(t *testing.T) {
	result, err := DecodeWithDiagnostics(strings.NewReader(englishDatesInput), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}

	want := []struct {
		line    int
		context string
		readAs  string
	}{
		{10, "June 1880", `"JUN 1880"`},
		{15, "about 1873", `"ABT 1873"`},
	}
	if len(result.Diagnostics) != len(want) {
		t.Fatalf("diagnostics = %v, want %d", result.Diagnostics, len(want))
	}
	for i, w := range want {
		d := result.Diagnostics[i]
		if d.Code != CodeNonStandardDate || d.Severity != SeverityWarning || d.Line != w.line || d.Context != w.context {
			t.Errorf("diagnostics[%d] = %+v, want a %s warning on line %d with context %q",
				i, d, CodeNonStandardDate, w.line, w.context)
		}
		if !strings.Contains(d.Message, w.readAs) {
			t.Errorf("diagnostics[%d].Message = %q, want it to name %s", i, d.Message, w.readAs)
		}
	}

	doc := result.Document
	mary := doc.GetIndividual("@I2@").BirthEvent()
	if mary.Date != "June 1880" {
		t.Errorf("Mary birth Date = %q, want the raw value", mary.Date)
	}
	if p := mary.ParsedDate; p == nil || p.Month != 6 || p.Year != 1880 || p.Original != "June 1880" {
		t.Errorf("Mary birth ParsedDate = %+v, want June 1880", p)
	}
	anna := doc.GetIndividual("@I3@")
	if p := anna.BirthEvent().ParsedDate; p == nil || p.Modifier != gedcom.ModifierAbout || p.Year != 1873 {
		t.Errorf("Anna birth ParsedDate = %+v, want ABT 1873", p)
	}
	if living, reason := anna.PresumedLiving(doc, gedcom.LivingConfig{ReferenceYear: 2026}); living {
		t.Errorf("Anna PresumedLiving = (true, %q), want false", reason)
	}
}

// A value outside the list of English spellings is still INVALID_VALUE, with
// the strict parser's message, and stays unparsed.
func TestDecodeUnrecognizedDateStillInvalid(t *testing.T) {
	for _, value := range []string{"summer", "Bane", "1935-1993"} {
		t.Run(value, func(t *testing.T) {
			input := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n" +
				"0 @I1@ INDI\n1 BIRT\n2 DATE " + value + "\n0 TRLR\n"
			result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
			if err != nil {
				t.Fatalf("DecodeWithDiagnostics() error = %v", err)
			}
			_, parseErr := gedcom.ParseDate(value)
			if parseErr == nil {
				t.Fatalf("ParseDate(%q) succeeded", value)
			}
			if len(result.Diagnostics) != 1 {
				t.Fatalf("diagnostics = %v, want 1", result.Diagnostics)
			}
			d := result.Diagnostics[0]
			wantMsg := "invalid value for DATE: " + parseErr.Error()
			if d.Code != CodeInvalidValue || d.Severity != SeverityWarning || d.Message != wantMsg || d.Context != value {
				t.Errorf("diagnostic = %+v, want %s %q with context %q", d, CodeInvalidValue, wantMsg, value)
			}
			birth := result.Document.GetIndividual("@I1@").BirthEvent()
			if birth.ParsedDate != nil || birth.Date != value {
				t.Errorf("birth = Date %q ParsedDate %+v, want the raw value unparsed", birth.Date, birth.ParsedDate)
			}
		})
	}
}

// An English date under a GEDCOM 7.0 PHRASE keeps both, and an LDS ordinance
// DATE is read the same way as an event's.
func TestDecodeEnglishDateWithPhraseAndOrdinance(t *testing.T) {
	const input = "0 HEAD\n1 GEDC\n2 VERS 7.0\n" +
		"0 @I1@ INDI\n1 BIRT\n2 DATE June 1880\n3 PHRASE early summer\n" +
		"1 BAPL\n2 DATE 12 December 1846\n0 TRLR\n"
	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	if hist := diagHistogram(result.Diagnostics); len(hist) != 1 || hist[CodeNonStandardDate] != 2 {
		t.Errorf("diagnostic codes = %v, want 2 %s", hist, CodeNonStandardDate)
	}
	ind := result.Document.GetIndividual("@I1@")
	if p := ind.BirthEvent().ParsedDate; p == nil || p.Month != 6 || p.Year != 1880 || p.Phrase != "early summer" || p.IsPhrase {
		t.Errorf("birth ParsedDate = %+v, want June 1880 with the PHRASE", p)
	}
	if len(ind.LDSOrdinances) != 1 {
		t.Fatalf("LDSOrdinances = %d, want 1", len(ind.LDSOrdinances))
	}
	if p := ind.LDSOrdinances[0].ParsedDate; p == nil || p.Day != 12 || p.Month != 12 || p.Year != 1846 {
		t.Errorf("BAPL ParsedDate = %+v, want 12 DEC 1846", p)
	}
}
