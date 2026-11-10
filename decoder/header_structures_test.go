package decoder

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestDecodeHeaderStructures verifies that the standard header structures
// HEAD.SOUR.VERS/NAME/CORP, HEAD.DEST, HEAD.DATE.TIME and HEAD.PLAC.FORM
// reach typed fields on gedcom.Header (issue #676).
func TestDecodeHeaderStructures(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  gedcom.Header
	}{
		{
			name: "5.5.1 with CORP ADDR and CONT",
			input: `0 HEAD
1 SOUR FTW
2 VERS 9.0
2 NAME Family Tree Maker
2 CORP Acme Software
3 ADDR 123 Main St
4 CONT Suite 4
4 CITY Springfield
4 STAE IL
4 POST 62701
4 CTRY USA
3 PHON 555-1234
3 PHON 555-5678
3 EMAIL info@example.com
3 FAX 555-0000
3 WWW www.example.com
2 DATA Some Data
3 DATE 1 JAN 2000
3 COPR Data copyright
1 DEST ANSTFILE
1 DATE 7 AUG 2026
2 TIME 13:45:02
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
1 PLAC
2 FORM City, County, State, Country
0 TRLR`,
			want: gedcom.Header{
				SourceSystem:  "FTW",
				SourceVersion: "9.0",
				SourceName:    "Family Tree Maker",
				SourceCorporation: &gedcom.Corporation{
					Name: "Acme Software",
					Address: &gedcom.Address{
						Full:       "123 Main St\nSuite 4",
						City:       "Springfield",
						State:      "IL",
						PostalCode: "62701",
						Country:    "USA",
					},
					Phone:   []string{"555-1234", "555-5678"},
					Email:   []string{"info@example.com"},
					Fax:     []string{"555-0000"},
					Website: []string{"www.example.com"},
				},
				Destination: "ANSTFILE",
				Date:        "7 AUG 2026",
				Time:        "13:45:02",
				PlaceForm:   "City, County, State, Country",
			},
		},
		{
			name: "7.0",
			input: `0 HEAD
1 GEDC
2 VERS 7.0
1 SOUR https://example.com/app
2 VERS 2.1
2 NAME Example App
2 CORP Example Inc
3 ADDR
4 ADR1 1 Infinite Loop
4 CITY Cupertino
3 WWW https://example.com
1 DEST https://example.org
1 DATE 7 AUG 2026
2 TIME 01:02:03.5Z
1 PLAC
2 FORM Town, Region
0 TRLR`,
			want: gedcom.Header{
				SourceSystem:  "https://example.com/app",
				SourceVersion: "2.1",
				SourceName:    "Example App",
				SourceCorporation: &gedcom.Corporation{
					Name:    "Example Inc",
					Address: &gedcom.Address{Line1: "1 Infinite Loop", City: "Cupertino"},
					Website: []string{"https://example.com"},
				},
				Destination: "https://example.org",
				Date:        "7 AUG 2026",
				Time:        "01:02:03.5Z",
				PlaceForm:   "Town, Region",
			},
		},
		{
			name: "absent",
			input: `0 HEAD
1 GEDC
2 VERS 7.0
1 SOUR APP
1 DATE 7 AUG 2026
0 TRLR`,
			want: gedcom.Header{SourceSystem: "APP", Date: "7 AUG 2026"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := Decode(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			h := doc.Header
			got := gedcom.Header{
				SourceSystem:      h.SourceSystem,
				SourceVersion:     h.SourceVersion,
				SourceName:        h.SourceName,
				SourceCorporation: h.SourceCorporation,
				Destination:       h.Destination,
				Date:              h.Date,
				Time:              h.Time,
				PlaceForm:         h.PlaceForm,
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("header structures:\n got %+v\nwant %+v", got, tt.want)
				if got.SourceCorporation != nil && tt.want.SourceCorporation != nil {
					t.Errorf("corporation:\n got %+v %+v\nwant %+v %+v",
						got.SourceCorporation, got.SourceCorporation.Address,
						tt.want.SourceCorporation, tt.want.SourceCorporation.Address)
				}
			}
		})
	}
}

// TestDecodeHeaderStructuresIgnoreOtherLevels verifies that the typed fields
// read only their own position: a VERS under GEDC, a NAME or DATE elsewhere,
// or a FORM under GEDC do not leak into them.
func TestDecodeHeaderStructuresIgnoreOtherLevels(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
1 SOUR APP
2 DATA Data
3 DATE 1 JAN 2000
4 TIME 10:00
0 @I1@ INDI
1 NAME John /Doe/
1 DEST NOWHERE
0 TRLR`
	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	h := doc.Header
	if h.SourceVersion != "" || h.SourceName != "" || h.SourceCorporation != nil ||
		h.Destination != "" || h.Time != "" || h.PlaceForm != "" {
		t.Errorf("unexpected typed values: %+v", h)
	}
}
