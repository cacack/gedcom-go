package decoder

import (
	"reflect"
	"strings"
	"testing"
)

// headerUnknownTagLines decodes input and returns the line numbers of its
// UNKNOWN_TAG diagnostics.
func headerUnknownTagLines(t *testing.T, input string) []int {
	t.Helper()
	res, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	var lines []int
	for _, d := range res.Diagnostics {
		if d.Code == CodeUnknownTag {
			lines = append(lines, d.Line)
		}
	}
	return lines
}

// TestHeaderUnknownTags verifies that buildHeader reports unknown header tags
// like every other context (issue #449). It used to be given no collector, so
// a nonstandard header tag was never reported.
func TestHeaderUnknownTags(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []int // lines reported as UNKNOWN_TAG
	}{
		{
			name: "unknown tag at HEAD level, subordinates not reported",
			input: `0 HEAD
1 GEDC
2 VERS 5.5.1
1 ZZZZ vendor value
2 YYYY nested
1 CHAR UTF-8
0 TRLR`,
			want: []int{4},
		},
		{
			name: "unknown tags in nested header contexts",
			input: `0 HEAD
1 SOUR APP
2 ZZZZ under SOUR
2 CORP Vendor
3 ZZZZ under CORP
3 ADDR 1 Main St
4 ZZZZ under ADDR
2 DATA Data
3 ZZZZ under DATA
1 DATE 1 JAN 2020
2 ZZZZ under DATE
1 GEDC
2 VERS 5.5.1
2 ZZZZ under GEDC
1 PLAC
2 ZZZZ under PLAC
0 TRLR`,
			want: []int{3, 5, 7, 9, 11, 14, 16},
		},
		{
			name: "misplaced standard tag",
			input: `0 HEAD
1 GEDC
2 VERS 5.5
1 DATE 6 APR 1996
1 TIME 14:54
0 TRLR`,
			want: []int{5},
		},
		{
			name: "underscore tags are extensions, not reported",
			input: `0 HEAD
1 SOUR Ancestry.com
2 _TREE My Tree
1 GEDC
2 VERS 5.5.1
1 _PROJECT_GUID 1234
2 ZZZZ under an extension
0 TRLR`,
			want: nil,
		},
		{
			name: "every standard 5.5.1 header structure",
			input: `0 HEAD
1 SOUR APP
2 VERS 1.0
2 NAME Application
2 CORP Vendor
3 ADDR 1 Main St
4 CONT Suite 2
4 ADR1 1 Main St
4 ADR2 Suite 2
4 ADR3 Floor 3
4 CITY Town
4 STAE State
4 POST 12345
4 CTRY Country
3 PHON 555-0100
3 EMAIL a@example.com
3 FAX 555-0101
3 WWW example.com
2 DATA Source Data
3 DATE 1 JAN 2020
3 COPR Copyright
4 CONT more
4 CONC more
1 DEST OTHER
1 DATE 1 JAN 2020
2 TIME 12:00:00
1 SUBM @U1@
1 SUBN @N1@
1 FILE family.ged
1 COPR Copyright
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
2 VERS 1.0
1 LANG English
1 PLAC
2 FORM City, County, State, Country
1 NOTE Header note
2 CONT continued
2 CONC concatenated
0 @U1@ SUBM
1 NAME Submitter
0 @N1@ SUBN
0 TRLR`,
			want: nil,
		},
		{
			name: "every standard 7.0 header structure",
			input: `0 HEAD
1 GEDC
2 VERS 7.0
1 SCHMA
2 TAG _SKYPEID http://xmlns.com/foaf/0.1/skypeID
1 SOUR APP
2 DATA Source Data
3 DATE 1 JAN 2020
4 TIME 12:00
1 NOTE Header note
2 CONT continued
2 MIME text/plain
2 LANG en
2 TRAN Translated
3 LANG fr
2 SOUR @S1@
3 PAGE 1
1 SNOTE @N1@
0 @S1@ SOUR
1 TITL Title
0 @N1@ SNOTE Shared note
0 TRLR`,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := headerUnknownTagLines(t, tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UNKNOWN_TAG lines = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestHeaderUnknownTagKeptInTags verifies that a reported header tag is still
// kept in Header.Tags: reporting it does not drop it (ADR 0003).
func TestHeaderUnknownTagKeptInTags(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
1 ZZZZ vendor value
0 TRLR`
	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	for _, tag := range doc.Header.Tags {
		if tag.Tag == "ZZZZ" && tag.Value == "vendor value" {
			return
		}
	}
	t.Errorf("Header.Tags lost the unknown tag: %+v", doc.Header.Tags)
}
