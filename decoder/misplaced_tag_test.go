package decoder

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// tagFindings decodes input and returns its UNKNOWN_TAG and MISPLACED_TAG
// diagnostics as "line CODE message".
func tagFindings(t *testing.T, input string) []string {
	t.Helper()
	res, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	var got []string
	for _, d := range res.Diagnostics {
		if d.Code == CodeUnknownTag || d.Code == CodeMisplacedTag {
			got = append(got, fmt.Sprintf("%d %s %s", d.Line, d.Code, d.Message))
		}
	}
	return got
}

// TestMisplacedTag tells a standard tag in the wrong place from a tag the
// version does not define (issue #884).
func TestMisplacedTag(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name: "standard tags under the wrong parent",
			input: `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME First /Example/
1 MARR
2 DATE 1 JAN 1900
1 EMAIL first@example.com
0 @O1@ OBJE
1 FILE photo.jpg
2 FORM jpg
1 DATE 1 JAN 1950
0 TRLR`,
			want: []string{
				"8 MISPLACED_TAG MARR is not allowed under INDI",
				"10 MISPLACED_TAG EMAIL is not allowed under INDI",
				"14 MISPLACED_TAG DATE is not allowed under OBJE",
			},
		},
		{
			name: "undefined tags stay unknown",
			input: `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 ZZZZ value
1 BIRT
2 ZZZZ nested
0 TRLR`,
			want: []string{
				"6 UNKNOWN_TAG unknown tag: ZZZZ",
				"8 UNKNOWN_TAG unknown tag: ZZZZ",
			},
		},
		{
			name: "nested parent is named",
			input: `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 BIRT
2 HUSB
0 TRLR`,
			want: []string{"7 MISPLACED_TAG HUSB is not allowed under BIRT"},
		},
		{
			name: "a tag defined only by another version is unknown",
			input: `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 NAME First /Example/
1 MIME text/plain
0 TRLR`,
			want: []string{"7 UNKNOWN_TAG unknown tag: MIME"},
		},
		{
			name: "header tags",
			input: `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
1 DATE 1 JAN 2020
1 TIME 12:00
0 TRLR`,
			want: []string{"6 MISPLACED_TAG TIME is not allowed under HEAD"},
		},
		{
			name: "under an unknown parent nothing is misplaced",
			input: `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @F1@ FAM
1 MARR
2 SOUR @S1@
3 ZZZZ
0 TRLR`,
			want: []string{"8 UNKNOWN_TAG unknown tag: ZZZZ"},
		},
		{
			// The decoder does not yet read CONT under NAME or PLAC, so it
			// still reports these lines, but never as misplaced.
			name: "7.0 CONT is never misplaced",
			input: `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME First /Example/
2 CONT more
1 BIRT
2 PLAC Example Town
3 CONT more
0 TRLR`,
			want: []string{"6 UNKNOWN_TAG unknown tag: CONT", "9 UNKNOWN_TAG unknown tag: CONT"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tagFindings(t, tt.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findings =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(tt.want, "\n"))
			}
		})
	}
}

// TestMisplacedTagSummary checks that repeats of a misplaced tag are capped
// and counted as unknown tags are, with an allowance of their own.
func TestMisplacedTagSummary(t *testing.T) {
	var b strings.Builder
	b.WriteString("0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n")
	for i := 1; i <= MaxUnknownTagDiagnosticsPerTag+3; i++ {
		fmt.Fprintf(&b, "0 @I%d@ INDI\n1 EMAIL p%d@example.com\n1 ZZZZ x\n", i, i)
	}
	b.WriteString("0 TRLR\n")

	counts := map[string]int{}
	var summaries []string
	for _, f := range tagFindings(t, b.String()) {
		if strings.Contains(f, "more lines") {
			summaries = append(summaries, f)
			continue
		}
		counts[strings.Fields(f)[1]]++
	}
	if counts[CodeMisplacedTag] != MaxUnknownTagDiagnosticsPerTag || counts[CodeUnknownTag] != MaxUnknownTagDiagnosticsPerTag {
		t.Errorf("per-line counts = %v, want %d of each", counts, MaxUnknownTagDiagnosticsPerTag)
	}
	firstOver := 4 + 3*MaxUnknownTagDiagnosticsPerTag + 2 // the 11th EMAIL line
	want := []string{
		fmt.Sprintf("%d MISPLACED_TAG misplaced tag: EMAIL (3 more lines with this tag, from this line on, not reported individually)", firstOver),
		fmt.Sprintf("%d UNKNOWN_TAG unknown tag: ZZZZ (3 more lines with this tag, from this line on, not reported individually)", firstOver+1),
	}
	if !reflect.DeepEqual(summaries, want) {
		t.Errorf("summaries =\n%s\nwant\n%s", strings.Join(summaries, "\n"), strings.Join(want, "\n"))
	}
}

func TestTagPlacementMisplaced(t *testing.T) {
	if placementFor("6.0") != nil {
		t.Error("placementFor(6.0) is not nil")
	}
	var none *tagPlacement
	if none.misplaced("INDI", true, "MARR") {
		t.Error("a nil placement reports a tag misplaced")
	}
	p := placement70
	for _, tc := range []struct {
		parent string
		top    bool
		tag    string
		want   bool
	}{
		{"INDI", true, "MARR", true},
		{"INDI", true, "BIRT", false},
		{"_EXT", false, "DATE", false}, // extension parent
		{"INDI", true, "_EXT", false},  // extension tag
		{"SEX", false, "DATE", true},   // a parent that takes no substructures
		// A citation SOUR allows PAGE. Parents are matched by tag, so PAGE
		// under HEAD's SOUR is accepted too: a known false negative.
		{"SOUR", false, "PAGE", false},
		{"SOUR", true, "PAGE", true},   // the SOUR record
		{"ZZZZ", false, "DATE", false}, // unknown parent
		{"HEAD", true, "DATE", false},  // header
		{"DATE", false, "PHRASE", false},
		{"NAME", false, "CONT", false}, // 7.0 CONT continues any line
	} {
		if got := p.misplaced(tc.parent, tc.top, tc.tag); got != tc.want {
			t.Errorf("misplaced(%q, %v, %q) = %v, want %v", tc.parent, tc.top, tc.tag, got, tc.want)
		}
	}
}
