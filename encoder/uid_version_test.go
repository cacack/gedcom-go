package encoder

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Issue #715: a record written from its entity spells its UIDs for the output
// version, UID for 7.0 and the vendor _UID for 5.5/5.5.1, so a 5.5.x document
// carrying identifiers does not need 7.0 to be written losslessly.
func TestUIDsWrittenForOutputVersion(t *testing.T) {
	newDoc := func(headerVersion gedcom.Version) *gedcom.Document {
		return &gedcom.Document{
			Header: &gedcom.Header{Version: headerVersion},
			Records: []*gedcom.Record{{
				Type: gedcom.RecordTypeIndividual, XRef: "@I1@",
				Entity: &gedcom.Individual{
					XRef:   "@I1@",
					UIDs:   []string{"abc"},
					Events: []*gedcom.Event{{Type: "BIRT", UIDs: []string{"ev"}}},
				},
			}},
		}
	}
	tests := []struct {
		name    string
		header  gedcom.Version
		target  gedcom.Version
		wantTag string
	}{
		{"5.5.1 header", gedcom.Version551, "", "_UID"},
		{"5.5 header", gedcom.Version55, "", "_UID"},
		{"7.0 header", gedcom.Version70, "", "UID"},
		{"unknown version", "", "", "UID"},
		{"7.0 header retargeted to 5.5.1", gedcom.Version70, gedcom.Version551, "_UID"},
		{"5.5.1 header retargeted to 7.0", gedcom.Version551, gedcom.Version70, "UID"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := EncodeWithOptions(&buf, newDoc(tt.header), &EncodeOptions{TargetVersion: tt.target}); err != nil {
				t.Fatalf("Encode: %v", err)
			}
			out := buf.String()
			if !strings.Contains(out, "1 "+tt.wantTag+" abc\n") || !strings.Contains(out, "2 "+tt.wantTag+" ev\n") {
				t.Errorf("want record and event UIDs as %s:\n%s", tt.wantTag, out)
			}
			other := map[string]string{"UID": "_UID", "_UID": "UID"}[tt.wantTag]
			if strings.Contains(out, " "+other+" ") {
				t.Errorf("output also writes %s:\n%s", other, out)
			}
		})
	}
}

// TestMinimumVersionIgnoresUIDs checks the version detector agrees with the
// encoder: UIDs alone do not force 7.0.
func TestMinimumVersionIgnoresUIDs(t *testing.T) {
	doc := &gedcom.Document{Records: []*gedcom.Record{{
		Type: gedcom.RecordTypeIndividual, XRef: "@I1@",
		Entity: &gedcom.Individual{
			XRef:       "@I1@",
			UIDs:       []string{"abc"},
			Events:     []*gedcom.Event{{Type: "BIRT", UIDs: []string{"ev"}}},
			Attributes: []*gedcom.Attribute{{Type: "OCCU", UIDs: []string{"at"}}},
		},
	}}}
	if doc.RequiresGEDCOM7() {
		t.Errorf("RequiresGEDCOM7() = true for a document whose only 7.0-looking feature is UIDs")
	}
	if v := doc.MinimumVersion(); v != gedcom.Version551 {
		t.Errorf("MinimumVersion() = %v, want 5.5.1", v)
	}
}
