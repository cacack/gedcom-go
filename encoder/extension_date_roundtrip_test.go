package encoder

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestExtensionDatesRoundTrip decodes gedcom.io's extensions.ged, whose dates
// use extension calendars, months and epochs (ADR 0014): each parses with its
// tags kept, and each DATE is written back unchanged, from the raw tags and
// from the typed entity.
func TestExtensionDatesRoundTrip(t *testing.T) {
	input, err := os.ReadFile("../testdata/gedcom-7.0/familysearch-examples/extensions.ged")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := decoder.Decode(bytes.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	dates := map[gedcom.EventType]*gedcom.Date{}
	for _, ind := range doc.Individuals() {
		for _, e := range ind.Events {
			dates[e.Type] = e.ParsedDate
		}
	}
	birth := dates[gedcom.EventBirth]
	if birth == nil || birth.Calendar != gedcom.CalendarFrenchRepublican || birth.MonthTag != "_JOUR" ||
		birth.EndDate == nil || birth.EndDate.CalendarTag != "_CALENDRIER" || birth.EndDate.MonthTag != "COMP" {
		t.Errorf("BIRT ParsedDate = %+v, want FRENCH_R 2 _JOUR 8 to _CALENDRIER 4 COMP 8", birth)
	}
	death := dates[gedcom.EventDeath]
	if death == nil || death.CalendarTag != "_CALENDAR" || death.MonthTag != "_MONTH" || death.EpochTag != "_EPOCH" {
		t.Errorf("DEAT ParsedDate = %+v, want _CALENDAR _MONTH _EPOCH", death)
	}
	grad := dates[gedcom.EventGraduation]
	if grad == nil || grad.CalendarTag != "_CAL2" || grad.MonthTag != "_MON2" || grad.EpochTag != "_EP2" {
		t.Errorf("GRAD ParsedDate = %+v, want _CAL2 _MON2 _EP2", grad)
	}

	lines := []string{
		"2 DATE BET FRENCH_R 2 _JOUR 8 AND _CALENDRIER 4 COMP 8\n",
		"2 DATE _CALENDAR 8 _MONTH 190 _EPOCH\n",
		"2 DATE _CAL2 23 _MON2 88 _EP2\n",
	}
	encode := func() string {
		t.Helper()
		var buf bytes.Buffer
		if err := EncodeWithOptions(&buf, doc, &EncodeOptions{}); err != nil {
			t.Fatalf("Encode() error = %v", err)
		}
		return buf.String()
	}
	if got := encode(); got != string(input) {
		t.Errorf("round trip not byte-identical:\ngot:\n%s\nwant:\n%s", got, input)
	}
	for _, record := range doc.Records {
		if record.Type == gedcom.RecordTypeIndividual {
			record.Tags = nil
		}
	}
	got := encode()
	for _, line := range lines {
		if !strings.Contains(got, line) {
			t.Errorf("entity-built output lacks %q:\n%s", line, got)
		}
	}
}
