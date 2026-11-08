package encoder

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// datePhraseInput70 carries a GEDCOM 7.0 DATE PHRASE on an event, on a
// phrase-only event date (empty DATE value) and on an LDS ordinance date.
const datePhraseInput70 = "0 HEAD\n1 GEDC\n2 VERS 7.0\n" +
	"0 @I1@ INDI\n" +
	"1 BIRT\n2 DATE 1850\n3 PHRASE about 1850\n" +
	"1 RESI\n2 DATE\n3 PHRASE while young\n" +
	"1 BAPL\n2 DATE 1860\n3 PHRASE as a child\n" +
	"0 TRLR\n"

// encodeTypedOnly decodes input, clears every Record.Tags so the typed
// entities are the only source, and encodes the result with opts.
func encodeTypedOnly(t *testing.T, input string, opts *EncodeOptions) string {
	t.Helper()
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	for _, rec := range doc.Records {
		rec.Tags = nil // force the entity path
	}
	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, doc, opts); err != nil {
		t.Fatalf("EncodeWithOptions() error = %v", err)
	}
	return buf.String()
}

// TestDatePhrase70RoundTrip pins issue #691: a 7.0 DATE PHRASE is written
// beneath its DATE from ParsedDate.Phrase and survives a re-decode.
func TestDatePhrase70RoundTrip(t *testing.T) {
	out := encodeTypedOnly(t, datePhraseInput70, nil)
	for _, want := range []string{
		"1 BIRT\n2 DATE 1850\n3 PHRASE about 1850\n",
		"1 RESI\n2 DATE\n3 PHRASE while young\n",
		"1 BAPL\n2 DATE 1860\n3 PHRASE as a child\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}

	doc, err := decoder.Decode(strings.NewReader(out))
	if err != nil {
		t.Fatalf("re-Decode() error = %v", err)
	}
	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatalf("re-decoded document has no @I1@:\n%s", out)
	}
	phrases := map[string]string{}
	for _, ev := range indi.Events {
		if ev.ParsedDate != nil {
			phrases[string(ev.Type)] = ev.ParsedDate.Phrase
		}
	}
	for _, ord := range indi.LDSOrdinances {
		if ord.ParsedDate != nil {
			phrases[string(ord.Type)] = ord.ParsedDate.Phrase
		}
	}
	for tag, want := range map[string]string{"BIRT": "about 1850", "RESI": "while young", "BAPL": "as a child"} {
		if got := phrases[tag]; got != want {
			t.Errorf("%s ParsedDate.Phrase = %q, want %q", tag, got, want)
		}
	}
}

// TestDatePhrase551ValueNotWrittenAsPhrase pins that a 5.5.1 "(text)" date
// phrase stays in the DATE value, whatever the output version.
func TestDatePhrase551ValueNotWrittenAsPhrase(t *testing.T) {
	const input = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n" +
		"0 @I1@ INDI\n1 BIRT\n2 DATE (unknown)\n" +
		"0 TRLR\n"
	for _, tc := range []struct {
		name string
		opts *EncodeOptions
	}{
		{name: "header version", opts: nil},
		{name: "target 7.0", opts: &EncodeOptions{TargetVersion: gedcom.Version70}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := encodeTypedOnly(t, input, tc.opts)
			if !strings.Contains(out, "2 DATE (unknown)\n") {
				t.Errorf("output missing DATE (unknown):\n%s", out)
			}
			if strings.Contains(out, "PHRASE") {
				t.Errorf("output has a PHRASE line:\n%s", out)
			}
		})
	}
}

// TestDatePhraseDroppedFor5x pins that 5.x output never writes a DATE PHRASE:
// the phrase is dropped, as the converter does on a 7.0 downgrade.
func TestDatePhraseDroppedFor5x(t *testing.T) {
	out := encodeTypedOnly(t, datePhraseInput70, &EncodeOptions{TargetVersion: gedcom.Version551})
	if strings.Contains(out, "PHRASE") {
		t.Errorf("5.5.1 output has a PHRASE line:\n%s", out)
	}
	if !strings.Contains(out, "2 DATE 1850\n") {
		t.Errorf("5.5.1 output missing the DATE value:\n%s", out)
	}
}

// TestDatePhraseRawOptionsNotWritten pins that writers called with options
// forOutput has not resolved (or nil) do not write a DATE PHRASE.
func TestDatePhraseRawOptionsNotWritten(t *testing.T) {
	ev := &gedcom.Event{Type: gedcom.EventBirth, ParsedDate: &gedcom.Date{Phrase: "x", IsPhrase: true}}
	for _, tag := range eventToTags(ev, 1, nil) {
		if tag.Tag == "DATE" || tag.Tag == "PHRASE" {
			t.Errorf("unexpected %s with nil options", tag.Tag)
		}
	}
	opts := (&EncodeOptions{TargetVersion: gedcom.Version70}).forOutput(nil)
	tags := eventToTags(ev, 1, opts)
	if len(tags) != 3 || tags[1].Tag != "DATE" || tags[1].Value != "" || tags[2].Tag != "PHRASE" || tags[2].Level != 3 {
		t.Errorf("7.0 tags = %+v, want BIRT, empty DATE, PHRASE at level 3", tags)
	}
}

// TestDatePhraseHandBuiltValuePhraseNotDoubled pins that a hand-built event
// whose DATE value is a "(text)" phrase, with a ParsedDate that never set
// Original, is not also given a PHRASE line: the written value decides.
func TestDatePhraseHandBuiltValuePhraseNotDoubled(t *testing.T) {
	ev := &gedcom.Event{
		Type:       gedcom.EventBirth,
		Date:       "(unknown)",
		ParsedDate: &gedcom.Date{Phrase: "unknown", IsPhrase: true},
	}
	opts := (&EncodeOptions{TargetVersion: gedcom.Version70}).forOutput(nil)
	for _, tag := range eventToTags(ev, 1, opts) {
		if tag.Tag == "PHRASE" {
			t.Errorf("7.0 output doubled the value phrase as a PHRASE line: %+v", tag)
		}
	}
}
