package encoder

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestEventAndAttributePayloadRoundTrip checks that a multi-line or overlong
// Event.Description and Attribute.Value are written with CONT/CONC lines as
// the output version allows (CONC only before 7.0) and decode back unchanged.
// Ref: Issue #734
func TestEventAndAttributePayloadRoundTrip(t *testing.T) {
	description := "Age: 57; " + strings.Repeat("Occupation: Farmer; ", 15) + "\nBirthplace: Virginia"
	value := "Farmer\nand carpenter"

	for _, version := range []gedcom.Version{gedcom.Version551, gedcom.Version70} {
		t.Run(string(version), func(t *testing.T) {
			doc := &gedcom.Document{
				Header: &gedcom.Header{Version: version},
				Records: []*gedcom.Record{{
					XRef: "@I1@",
					Type: gedcom.RecordTypeIndividual,
					Entity: &gedcom.Individual{
						XRef:       "@I1@",
						Events:     []*gedcom.Event{{Type: gedcom.EventResidence, Description: description}},
						Attributes: []*gedcom.Attribute{{Type: gedcom.AttributeOccupation, Value: value}},
					},
				}},
			}

			var buf bytes.Buffer
			if err := Encode(&buf, doc); err != nil {
				t.Fatalf("Encode() error = %v", err)
			}
			out := buf.String()

			if !strings.Contains(out, "\n2 CONT Birthplace: Virginia") {
				t.Errorf("RESI description line break not written as CONT\n%s", out)
			}
			if !strings.Contains(out, "\n2 CONT and carpenter") {
				t.Errorf("OCCU value line break not written as CONT\n%s", out)
			}
			hasCONC := strings.Contains(out, "\n2 CONC ")
			if wantCONC := version != gedcom.Version70; hasCONC != wantCONC {
				t.Errorf("output has CONC = %v, want %v\n%s", hasCONC, wantCONC, out)
			}

			back, err := decoder.Decode(strings.NewReader(out))
			if err != nil {
				t.Fatalf("re-decode: %v\n%s", err, out)
			}
			indi := back.GetIndividual("@I1@")
			if indi == nil || len(indi.Events) != 1 || len(indi.Attributes) != 1 {
				t.Fatalf("re-decoded @I1@ = %+v\n%s", indi, out)
			}
			if got := indi.Events[0].Description; got != description {
				t.Errorf("Description = %q, want %q", got, description)
			}
			if got := indi.Attributes[0].Value; got != value {
				t.Errorf("Value = %q, want %q", got, value)
			}
		})
	}
}

// TestSyncTagsKeepsOccurredFlag checks that a decoded "Y" flag survives
// SyncTags after an edit to its record, which rewrites the record from the
// typed model; before Event.Occurred existed it came out as a bare line.
// Ref: Issue #865
func TestSyncTagsKeepsOccurredFlag(t *testing.T) {
	in := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n0 @I1@ INDI\n1 DEAT Y\n1 BURI Y\n0 TRLR\n"
	doc, err := decoder.Decode(strings.NewReader(in))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	doc.GetIndividual("@I1@").Events[0].Date = "1900"
	if err := SyncTags(doc); err != nil {
		t.Fatalf("SyncTags() error = %v", err)
	}
	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	out := buf.String()
	for _, line := range []string{"\n1 DEAT Y\n2 DATE 1900\n", "\n1 BURI Y\n"} {
		if !strings.Contains(out, line) {
			t.Errorf("output lacks %q\n%s", strings.TrimSpace(line), out)
		}
	}
}

// TestEventOccurredRoundTrip checks that Event.Occurred is written as the
// event line's "Y" payload in every version and decodes back with an empty
// Description; that a Description wins over the flag; and that a generic EVEN
// and a NO assertion never write it.
// Ref: Issue #865
func TestEventOccurredRoundTrip(t *testing.T) {
	for _, version := range []gedcom.Version{gedcom.Version55, gedcom.Version551, gedcom.Version70} {
		t.Run(string(version), func(t *testing.T) {
			indiEvents := []*gedcom.Event{
				{Type: gedcom.EventDeath, Occurred: true},
				{Type: gedcom.EventBurial, Occurred: true},
				{Type: gedcom.EventBirth},
				{Type: gedcom.EventResidence, Occurred: true, Description: "Farm"},
				{Type: "EVEN", Occurred: true},
			}
			if version == gedcom.Version70 {
				indiEvents = append(indiEvents, &gedcom.Event{Type: gedcom.EventChristening, Occurred: true, IsNegative: true})
			}
			doc := &gedcom.Document{
				Header: &gedcom.Header{Version: version},
				Records: []*gedcom.Record{
					{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: "@I1@", Events: indiEvents}},
					{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: &gedcom.Family{
						XRef: "@F1@", Events: []*gedcom.Event{{Type: gedcom.EventMarriage, Occurred: true}},
					}},
				},
			}

			var buf bytes.Buffer
			if err := Encode(&buf, doc); err != nil {
				t.Fatalf("Encode() error = %v", err)
			}
			out := buf.String()
			for _, line := range []string{"\n1 DEAT Y\n", "\n1 BURI Y\n", "\n1 BIRT\n", "\n1 RESI Farm\n", "\n1 EVEN\n", "\n1 MARR Y\n"} {
				if !strings.Contains(out, line) {
					t.Errorf("output lacks %q\n%s", strings.TrimSpace(line), out)
				}
			}
			if version == gedcom.Version70 && !strings.Contains(out, "\n1 NO CHR\n") {
				t.Errorf("output lacks NO CHR\n%s", out)
			}

			back, err := decoder.Decode(strings.NewReader(out))
			if err != nil {
				t.Fatalf("re-decode: %v\n%s", err, out)
			}
			indi := back.GetIndividual("@I1@")
			if indi == nil || len(indi.Events) != len(indiEvents) {
				t.Fatalf("re-decoded @I1@ = %+v\n%s", indi, out)
			}
			wantOccurred := []bool{true, true, false, false, false, false}
			wantDesc := []string{"", "", "", "Farm", "", ""}
			for i, ev := range indi.Events {
				if ev.Occurred != wantOccurred[i] || ev.Description != wantDesc[i] {
					t.Errorf("Events[%d] %s = {Occurred %v, Description %q}, want {%v %q}",
						i, ev.Type, ev.Occurred, ev.Description, wantOccurred[i], wantDesc[i])
				}
			}
			fam := back.GetFamily("@F1@")
			if fam == nil || len(fam.Events) != 1 || !fam.Events[0].Occurred || fam.Events[0].Description != "" {
				t.Errorf("re-decoded @F1@ events = %+v", fam)
			}
		})
	}
}
