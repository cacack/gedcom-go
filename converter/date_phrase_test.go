package converter_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/converter"
	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// phraseDoc is calendarDoc with a 7.0 PHRASE beneath the BIRT DATE, in the
// raw tags and in the typed ParsedDate, as decoding gives it.
func phraseDoc(value, phrase string) *gedcom.Document {
	doc := calendarDoc(gedcom.Version70, value)
	record := doc.Records[0]
	record.Tags = append(record.Tags, &gedcom.Tag{Level: 3, Tag: "PHRASE", Value: phrase})
	ind, _ := record.GetIndividual()
	parsed, err := gedcom.ParseDate(value)
	if err != nil {
		parsed = &gedcom.Date{Original: value, Calendar: gedcom.CalendarGregorian, IsPhrase: true}
	}
	parsed.Phrase = phrase
	ind.Events[0].ParsedDate = parsed
	return doc
}

// birthPhrase returns the value of the PHRASE beneath the converted BIRT
// DATE, and whether there is one.
func birthPhrase(doc *gedcom.Document) (string, bool) {
	for _, tag := range doc.GetRecord("@I1@").Tags {
		if tag.Tag == "PHRASE" {
			return tag.Value, true
		}
	}
	return "", false
}

func dataLossFeatures(report *gedcom.ConversionReport) []string {
	var features []string
	for _, item := range report.DataLoss {
		features = append(features, item.Feature)
	}
	return features
}

// TestConvertDateEpochAndPhraseTo70 covers issue #722 upgrading: each 5.x
// epoch spelling becomes BCE, and INT and parenthesized phrases move to a
// PHRASE, with nothing reported as lost.
func TestConvertDateEpochAndPhraseTo70(t *testing.T) {
	tests := []struct {
		name       string
		from       gedcom.Version
		value      string
		wantValue  string
		wantPhrase string // "" for no PHRASE
		wantTypes  []string
	}{
		{"B.C.", gedcom.Version551, "44 B.C.", "44 BCE", "", []string{"DATE_EPOCH_CONVERTED"}},
		{"BC", gedcom.Version551, "15 MAR 44 BC", "15 MAR 44 BCE", "", []string{"DATE_EPOCH_CONVERTED"}},
		{"B.C.E.", gedcom.Version551, "1 JAN 2000 B.C.E.", "1 JAN 2000 BCE", "", []string{"DATE_EPOCH_CONVERTED"}},
		{"lower case bc", gedcom.Version55, "44 bc", "44 BCE", "", []string{"DATE_EPOCH_CONVERTED"}},
		{"range endpoints", gedcom.Version551, "BET 100 B.C. AND 44 BC", "BET 100 BCE AND 44 BCE", "", []string{"DATE_EPOCH_CONVERTED"}},
		{"INT with phrase", gedcom.Version551, "INT 1850 (about eighteen fifty)", "1850", "about eighteen fifty", []string{"DATE_PHRASE_CONVERTED"}},
		{"INT from 5.5", gedcom.Version55, "INT 2 JAN 1850 (the day after)", "2 JAN 1850", "the day after", []string{"DATE_PHRASE_CONVERTED"}},
		{"INT without phrase", gedcom.Version551, "INT 1850", "1850", "", []string{"DATE_PHRASE_CONVERTED"}},
		{"phrase only", gedcom.Version551, "(unknown)", "", "unknown", []string{"DATE_PHRASE_CONVERTED"}},
		{
			"INT with calendar and epoch", gedcom.Version551, "INT @#DJULIAN@ 44 B.C. (Ides of March B.C.)", "JULIAN 44 BCE", "Ides of March B.C.",
			[]string{"DATE_CALENDAR_CONVERTED", "DATE_EPOCH_CONVERTED", "DATE_PHRASE_CONVERTED"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, report, err := converter.Convert(calendarDoc(tt.from, tt.value), gedcom.Version70)
			if err != nil {
				t.Fatalf("Convert: %v", err)
			}
			raw := got.GetRecord("@I1@").Tags[1].Value
			if raw != tt.wantValue {
				t.Errorf("raw DATE = %q, want %q", raw, tt.wantValue)
			}
			phrase, hasPhrase := birthPhrase(got)
			if phrase != tt.wantPhrase || hasPhrase != (tt.wantPhrase != "") {
				t.Errorf("PHRASE = %q (present %v), want %q", phrase, hasPhrase, tt.wantPhrase)
			}
			ind, _ := got.GetRecord("@I1@").GetIndividual()
			event := ind.Events[0]
			if event.Date != tt.wantValue {
				t.Errorf("Event.Date = %q, want %q", event.Date, tt.wantValue)
			}
			if event.ParsedDate == nil || event.ParsedDate.Phrase != tt.wantPhrase {
				t.Errorf("ParsedDate = %+v, want Phrase %q", event.ParsedDate, tt.wantPhrase)
			}
			if tt.wantValue != "" {
				if _, err := gedcom.ParseDate(raw); err != nil {
					t.Errorf("converted DATE %q does not parse: %v", raw, err)
				}
			}
			for _, kind := range tt.wantTypes {
				if !hasTransformation(report, kind) {
					t.Errorf("no %s transformation in %+v", kind, report.Transformations)
				}
			}
			if len(report.DataLoss) != 0 || len(report.Preserved) != 0 {
				t.Errorf("lossless mapping reported: DataLoss %+v, Preserved %+v", report.DataLoss, report.Preserved)
			}
			wantResult := tt.wantValue
			if tt.wantPhrase != "" {
				wantResult = strings.TrimPrefix(tt.wantValue+" + PHRASE "+tt.wantPhrase, " + ")
			}
			if !hasNote(report.Normalized, tt.value, wantResult) {
				t.Errorf("no Normalized note %q -> %q in %+v", tt.value, wantResult, report.Normalized)
			}
		})
	}
}

// TestConvertDatePhraseUnmappedTo70 covers 5.x phrases with no 7.0 form:
// each is left in the value and reported, so the conversion does not pass
// off invalid 7.0 output as clean.
func TestConvertDatePhraseUnmappedTo70(t *testing.T) {
	for _, value := range []string{
		"ABT 1850 (circa)",
		"INT (no date)",
		"INT 1850 (unclosed",
		"()",
	} {
		t.Run(value, func(t *testing.T) {
			got, report, err := converter.Convert(calendarDoc(gedcom.Version551, value), gedcom.Version70)
			if err != nil {
				t.Fatalf("Convert: %v", err)
			}
			if raw := got.GetRecord("@I1@").Tags[1].Value; raw != value {
				t.Errorf("raw DATE = %q, want it unchanged %q", raw, value)
			}
			if _, ok := birthPhrase(got); ok {
				t.Error("PHRASE added for an unmapped phrase")
			}
			want := gedcom.ConversionNote{Path: "Individual @I1@ > BIRT > DATE", Original: value, Result: value}
			found := false
			for _, n := range report.Preserved {
				if n.Path == want.Path && n.Original == want.Original && n.Result == want.Result && strings.Contains(n.Reason, "PHRASE") {
					found = true
				}
			}
			if !found {
				t.Errorf("no Preserved note %+v in %+v", want, report.Preserved)
			}
			if len(report.DataLoss) != 1 || report.DataLoss[0].Feature != "Date phrases" ||
				!slices.Equal(report.DataLoss[0].AffectedRecords, []string{"@I1@"}) {
				t.Errorf("DataLoss = %+v, want one Date phrases item naming @I1@", report.DataLoss)
			}
		})
	}
}

// TestConvertDatePhraseExactDateTo70 covers a phrase in a CHAN.DATE, which
// 7.0 types as DateExact with no PHRASE: it is reported, not moved.
func TestConvertDatePhraseExactDateTo70(t *testing.T) {
	doc := calendarDoc(gedcom.Version551, "1850")
	doc.Records[0].Tags = append(doc.Records[0].Tags,
		&gedcom.Tag{Level: 1, Tag: "CHAN"},
		&gedcom.Tag{Level: 2, Tag: "DATE", Value: "(sometime)"},
	)
	got, report, err := converter.Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if _, ok := birthPhrase(got); ok {
		t.Error("PHRASE added under CHAN.DATE")
	}
	if !slices.Contains(dataLossFeatures(report), "Date phrases") {
		t.Errorf("DataLoss = %+v, want a Date phrases item", report.DataLoss)
	}
}

// TestConvertDateEpochAndPhraseTo5x covers issue #722 downgrading: BCE
// becomes B.C., and a DATE with a PHRASE folds into INT or a parenthesized
// phrase with no data loss.
func TestConvertDateEpochAndPhraseTo5x(t *testing.T) {
	tests := []struct {
		name   string
		to     gedcom.Version
		value  string
		phrase string // "" for no PHRASE
		want   string
	}{
		{"BCE", gedcom.Version551, "44 BCE", "", "44 B.C."},
		{"BCE to 5.5", gedcom.Version55, "15 MAR 44 BCE", "", "15 MAR 44 B.C."},
		{"BCE range", gedcom.Version551, "FROM 100 BCE TO 44 BCE", "", "FROM 100 B.C. TO 44 B.C."},
		{"date and phrase", gedcom.Version551, "1850", "about eighteen fifty", "INT 1850 (about eighteen fifty)"},
		{"phrase only", gedcom.Version551, "", "unknown", "(unknown)"},
		{"phrase only to 5.5", gedcom.Version55, "", "unknown", "(unknown)"},
		{"calendar, epoch and phrase", gedcom.Version551, "JULIAN 15 MAR 44 BCE", "Ides of March", "INT @#DJULIAN@ 15 MAR 44 B.C. (Ides of March)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := calendarDoc(gedcom.Version70, tt.value)
			if tt.phrase != "" {
				doc = phraseDoc(tt.value, tt.phrase)
			}
			got, report, err := converter.Convert(doc, tt.to)
			if err != nil {
				t.Fatalf("Convert: %v", err)
			}
			if raw := got.GetRecord("@I1@").Tags[1].Value; raw != tt.want {
				t.Errorf("raw DATE = %q, want %q", raw, tt.want)
			}
			if p, ok := birthPhrase(got); ok {
				t.Errorf("PHRASE %q left in 5.x output", p)
			}
			if len(report.DataLoss) != 0 {
				t.Errorf("lossless mapping reported as DataLoss: %+v", report.DataLoss)
			}
			ind, _ := got.GetRecord("@I1@").GetIndividual()
			if ind.Events[0].Date != tt.want {
				t.Errorf("Event.Date = %q, want %q", ind.Events[0].Date, tt.want)
			}
			if _, err := gedcom.ParseDate(tt.want); err != nil {
				t.Errorf("converted DATE %q does not parse: %v", tt.want, err)
			}
			if tt.phrase != "" {
				original := strings.TrimPrefix(tt.value+" + PHRASE "+tt.phrase, " + ")
				if !hasNote(report.Normalized, original, tt.want) || !hasTransformation(report, "DATE_PHRASE_CONVERTED") {
					t.Errorf("no Normalized note %q -> %q or DATE_PHRASE_CONVERTED in %+v / %+v", original, tt.want, report.Normalized, report.Transformations)
				}
			}
		})
	}
}

// TestConvertDatePhraseUnfoldableTo5x covers PHRASEs 5.x has no form for:
// they are dropped and reported, and the DATE value is kept.
func TestConvertDatePhraseUnfoldableTo5x(t *testing.T) {
	tests := []struct {
		name, value, phrase string
	}{
		{"approximation", "ABT 1850", "circa"},
		{"range", "BET 1850 AND 1860", "the 1850s"},
		{"parenthesis in phrase", "1850", "about (or so)"},
		{"too long", "1850", strings.Repeat("x", 250)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, report, err := converter.Convert(phraseDoc(tt.value, tt.phrase), gedcom.Version551)
			if err != nil {
				t.Fatalf("Convert: %v", err)
			}
			if raw := got.GetRecord("@I1@").Tags[1].Value; raw != tt.value {
				t.Errorf("raw DATE = %q, want %q", raw, tt.value)
			}
			if _, ok := birthPhrase(got); ok {
				t.Error("PHRASE left in 5.x output")
			}
			if !slices.Contains(dataLossFeatures(report), "PHRASE tags") {
				t.Errorf("DataLoss = %+v, want a PHRASE tags item", report.DataLoss)
			}
		})
	}
}

// TestConvertDatePhraseEntityOnly covers documents built in code, whose
// typed event is the encoder's only write path, in both directions.
func TestConvertDatePhraseEntityOnly(t *testing.T) {
	t.Run("to 7.0", func(t *testing.T) {
		doc := calendarDoc(gedcom.Version551, "INT 44 B.C. (Ides of March)")
		doc.Records[0].Tags = nil
		got, report, err := converter.Convert(doc, gedcom.Version70)
		if err != nil {
			t.Fatalf("Convert: %v", err)
		}
		ind, _ := got.GetRecord("@I1@").GetIndividual()
		if ind.Events[0].Date != "44 BCE" || ind.Events[0].ParsedDate == nil || ind.Events[0].ParsedDate.Phrase != "Ides of March" {
			t.Errorf("event = %q / %+v, want 44 BCE with Phrase", ind.Events[0].Date, ind.Events[0].ParsedDate)
		}
		if text := encode(t, got); !strings.Contains(text, "2 DATE 44 BCE\n3 PHRASE Ides of March\n") {
			t.Errorf("7.0 output lacks DATE and PHRASE:\n%s", text)
		}
		if len(report.DataLoss) != 0 {
			t.Errorf("DataLoss = %+v", report.DataLoss)
		}
	})
	t.Run("to 5.5.1", func(t *testing.T) {
		doc := phraseDoc("44 BCE", "Ides of March")
		doc.Records[0].Tags = nil
		got, report, err := converter.Convert(doc, gedcom.Version551)
		if err != nil {
			t.Fatalf("Convert: %v", err)
		}
		ind, _ := got.GetRecord("@I1@").GetIndividual()
		if want := "INT 44 B.C. (Ides of March)"; ind.Events[0].Date != want {
			t.Errorf("Event.Date = %q, want %q", ind.Events[0].Date, want)
		}
		text := encode(t, got)
		if !strings.Contains(text, "2 DATE INT 44 B.C. (Ides of March)\n") || strings.Contains(text, "PHRASE") {
			t.Errorf("5.5.1 output lacks the folded DATE or keeps PHRASE:\n%s", text)
		}
		if len(report.DataLoss) != 0 {
			t.Errorf("DataLoss = %+v", report.DataLoss)
		}
	})
}

// datePhraseRoundTripGED is a 5.5.1 file with every epoch spelling and both
// lossless date phrase forms.
const datePhraseRoundTripGED = `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Julius /Caesar/
1 BIRT
2 DATE 100 B.C.
1 DEAT
2 DATE 15 MAR 44 BC
1 BURI
2 DATE INT 44 B.C.E. (after the funeral)
1 CHR
2 DATE (unknown)
1 RESI
2 DATE INT 1850 (about eighteen fifty)
0 TRLR
`

// TestConvertDatePhraseRoundTrip converts a 5.5.1 file to 7.0 and back
// through the encoder and decoder: every 7.0 DATE is valid 7.0 date syntax,
// and the 5.5.1 result keeps every phrase, with B.C. as the epoch.
func TestConvertDatePhraseRoundTrip(t *testing.T) {
	src, err := decoder.Decode(strings.NewReader(datePhraseRoundTripGED))
	if err != nil {
		t.Fatalf("decode 5.5.1: %v", err)
	}
	v7, report, err := converter.Convert(src, gedcom.Version70)
	if err != nil {
		t.Fatalf("Convert to 7.0: %v", err)
	}
	if len(report.DataLoss) != 0 {
		t.Errorf("upgrade DataLoss = %+v", report.DataLoss)
	}
	v7Text := encode(t, v7)
	for _, want := range []string{
		"2 DATE 100 BCE\n",
		"2 DATE 15 MAR 44 BCE\n",
		"2 DATE 44 BCE\n3 PHRASE after the funeral\n",
		"2 DATE\n3 PHRASE unknown\n",
		"2 DATE 1850\n3 PHRASE about eighteen fifty\n",
	} {
		if !strings.Contains(v7Text, want) {
			t.Errorf("7.0 output missing %q:\n%s", want, v7Text)
		}
	}
	for _, line := range dateLines(v7Text) {
		value := strings.TrimPrefix(line, "2 DATE ")
		if strings.ContainsAny(value, "()") || strings.Contains(value, "INT") || strings.Contains(value, "B.C") {
			t.Errorf("7.0 DATE %q keeps 5.x syntax", value)
		}
		if _, err := gedcom.ParseDate(value); err != nil {
			t.Errorf("7.0 DATE %q does not parse: %v", value, err)
		}
	}

	back, report, err := converter.Convert(decodeClean(t, v7Text, gedcom.Version70), gedcom.Version551)
	if err != nil {
		t.Fatalf("Convert to 5.5.1: %v", err)
	}
	if len(report.DataLoss) != 0 {
		t.Errorf("downgrade DataLoss = %+v", report.DataLoss)
	}
	backText := encode(t, back)
	decodeClean(t, backText, gedcom.Version551)
	want := []string{
		"2 DATE 100 B.C.",
		"2 DATE 15 MAR 44 B.C.",
		"2 DATE INT 44 B.C. (after the funeral)",
		"2 DATE (unknown)",
		"2 DATE INT 1850 (about eighteen fifty)",
	}
	if got := dateLines(backText); !slices.Equal(got, want) {
		t.Errorf("round trip DATE lines:\ngot  %q\nwant %q", got, want)
	}
	if ind := back.GetIndividual("@I1@"); ind.Events[2].ParsedDate == nil || ind.Events[2].ParsedDate.InterpretedFrom != "after the funeral" {
		t.Errorf("BURI ParsedDate = %+v, want InterpretedFrom kept", ind.Events[2].ParsedDate)
	}
}

// TestConvertDatePhraseSourceDataEntity covers the typed DATA.EVEN.DATE of a
// source record built in code: its phrase moves to DatePhrase for 7.0 and
// folds back into the value for 5.x, and a phrase-only value, which the
// encoder cannot write as an empty DATE, stays in the value.
func TestConvertDatePhraseSourceDataEntity(t *testing.T) {
	tests := []struct {
		name                 string
		from, to             gedcom.Version
		date, phrase         string
		wantDate, wantPhrase string
	}{
		{"to 7.0", gedcom.Version551, gedcom.Version70, "INT 1700 (about 1700)", "", "1700", "about 1700"},
		{"to 7.0 phrase only", gedcom.Version551, gedcom.Version70, "(unknown)", "", "(unknown)", ""},
		{"to 5.5.1", gedcom.Version70, gedcom.Version551, "1700", "about 1700", "INT 1700 (about 1700)", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := &gedcom.Source{
				XRef: "@S1@",
				Data: &gedcom.SourceData{Events: []*gedcom.SourceDataEvent{
					{Types: "BIRT", Date: tt.date, DatePhrase: tt.phrase},
				}},
			}
			record := &gedcom.Record{XRef: "@S1@", Type: gedcom.RecordTypeSource, Entity: src}
			doc := &gedcom.Document{
				Header:  &gedcom.Header{Version: tt.from},
				Records: []*gedcom.Record{record},
				XRefMap: map[string]*gedcom.Record{"@S1@": record},
			}

			got, _, err := converter.Convert(doc, tt.to)
			if err != nil {
				t.Fatalf("Convert: %v", err)
			}
			out, _ := got.GetRecord("@S1@").GetSource()
			e := out.Data.Events[0]
			if e.Date != tt.wantDate || e.DatePhrase != tt.wantPhrase {
				t.Errorf("Date, DatePhrase = %q, %q, want %q, %q", e.Date, e.DatePhrase, tt.wantDate, tt.wantPhrase)
			}
		})
	}
}
