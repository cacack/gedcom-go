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
