package decoder

import (
	"fmt"
	"strings"
	"testing"
)

// unknownTagMessages returns the message of every CodeUnknownTag diagnostic.
func unknownTagMessages(ds Diagnostics) []string {
	var out []string
	for _, d := range ds {
		if d.Code == CodeUnknownTag {
			out = append(out, fmt.Sprintf("line %d: %s", d.Line, d.Message))
		}
	}
	return out
}

// TestStandardRecordStructuresNotUnknown pins the record substructures the
// GEDCOM 5.5, 5.5.1 and 7.0 grammars define that the decoder used to report as
// UNKNOWN_TAG (#730): RIN on every 5.x record, 7.0 CREA/REFN/UID on REPO, SUBM
// and SNOTE, SUBM RFN, and the 5.5 OBJE record and PLAC layouts. Each line must
// be kept in the record's raw Tags (ADR 0003).
func TestStandardRecordStructuresNotUnknown(t *testing.T) {
	tests := []struct {
		name  string
		vers  string
		body  string
		xref  string
		lines []string // "TAG value" lines that must survive in Record.Tags
	}{
		{
			name: "5.5.1 RIN on FAM, SOUR, REPO, NOTE, OBJE",
			vers: "5.5.1",
			body: `0 @F1@ FAM
1 RIN MH:F1
0 @S1@ SOUR
1 TITL Census
1 RIN MH:S1
0 @R1@ REPO
1 NAME Archive
1 RIN MH:R1
0 @N1@ NOTE Text
1 RIN MH:N1
0 @O1@ OBJE
1 FILE a.jpg
2 FORM jpg
1 RIN MH:O1`,
			xref:  "@F1@",
			lines: []string{"RIN MH:F1"},
		},
		{
			name: "5.5.1 SUBM RFN",
			vers: "5.5.1",
			body: `0 @U1@ SUBM
1 NAME Me
1 RFN 123`,
			xref:  "@U1@",
			lines: []string{"RFN 123"},
		},
		{
			name: "7.0 REPO CREA",
			vers: "7.0",
			body: `0 @R1@ REPO
1 NAME Archive
1 CREA
2 DATE 1 JAN 2000`,
			xref:  "@R1@",
			lines: []string{"CREA "},
		},
		{
			name: "7.0 SUBM CREA and REFN",
			vers: "7.0",
			body: `0 @U1@ SUBM
1 NAME Me
1 CREA
2 DATE 1 JAN 2000
1 REFN 42
2 TYPE user`,
			xref:  "@U1@",
			lines: []string{"CREA ", "REFN 42"},
		},
		{
			name: "7.0 SNOTE CREA, REFN, UID",
			vers: "7.0",
			body: `0 @N1@ SNOTE Shared
1 CREA
2 DATE 1 JAN 2000
1 REFN 42
1 UID 0b4b7e4e-0f9a-4b5e-9e2b-1f5b0c3d2a10`,
			xref:  "@N1@",
			lines: []string{"CREA ", "REFN 42", "UID 0b4b7e4e-0f9a-4b5e-9e2b-1f5b0c3d2a10"},
		},
		{
			name: "5.5 OBJE record FORM, TITL, BLOB, OBJE",
			vers: "5.5",
			body: `0 @O1@ OBJE
1 FORM bmp
1 TITL Portrait
1 BLOB
2 CONT .HM.......
1 OBJE @O2@
0 @O2@ OBJE
1 FORM bmp
1 BLOB
2 CONT .HM.......`,
			xref:  "@O1@",
			lines: []string{"FORM bmp", "TITL Portrait", "BLOB ", "OBJE @O2@"},
		},
		{
			name: "5.5 PLAC SOUR",
			vers: "5.5",
			body: `0 @I1@ INDI
1 BIRT
2 PLAC Boston
3 SOUR @S1@
0 @S1@ SOUR
1 TITL Census`,
			xref:  "@I1@",
			lines: []string{"SOUR @S1@"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := "0 HEAD\n1 GEDC\n2 VERS " + tt.vers + "\n" + tt.body + "\n0 TRLR"
			result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
			if err != nil {
				t.Fatalf("DecodeWithDiagnostics() error = %v", err)
			}
			if got := unknownTagMessages(result.Diagnostics); len(got) != 0 {
				t.Errorf("standard structures reported as UNKNOWN_TAG:\n  %s", strings.Join(got, "\n  "))
			}

			record := result.Document.GetRecord(tt.xref)
			if record == nil {
				t.Fatalf("GetRecord(%s) = nil", tt.xref)
			}
			var kept []string
			for _, tag := range record.Tags {
				kept = append(kept, tag.Tag+" "+tag.Value)
			}
			for _, want := range tt.lines {
				found := false
				for _, k := range kept {
					if k == want {
						found = true
					}
				}
				if !found {
					t.Errorf("Record.Tags of %s lacks %q; has %q", tt.xref, want, kept)
				}
			}
		})
	}
}

// TestFiveFiveOnlyStructuresReportedInOtherVersions checks that the 5.5 OBJE
// record layout and PLAC.SOUR, which 5.5.1 and 7.0 do not define, are still
// reported in a document declaring 5.5.1, as ASSO.TYPE is (#730).
func TestFiveFiveOnlyStructuresReportedInOtherVersions(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @O1@ OBJE
1 FILE a.bmp
2 FORM bmp
1 FORM bmp
1 TITL Portrait
0 @I1@ INDI
1 BIRT
2 PLAC Boston
3 SOUR @S1@
0 @S1@ SOUR
1 TITL Census
0 TRLR`
	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	got := strings.Join(unknownTagMessages(result.Diagnostics), "\n")
	want := strings.Join([]string{
		"line 7: unknown tag: FORM",
		"line 8: unknown tag: TITL",
		"line 12: unknown tag: SOUR",
	}, "\n")
	if got != want {
		t.Errorf("UNKNOWN_TAG diagnostics =\n%s\nwant\n%s", got, want)
	}
}

// TestUnknownTagDiagnosticsCappedPerTag checks that a nonstandard tag repeated
// on many lines is reported line by line MaxUnknownTagDiagnosticsPerTag times,
// then once more counting the rest from the first line not reported (#730),
// that another tag has its own allowance, and that every line stays in the raw
// tags.
func TestUnknownTagDiagnosticsCappedPerTag(t *testing.T) {
	const events = MaxUnknownTagDiagnosticsPerTag + 5

	var b strings.Builder
	b.WriteString("0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @I1@ INDI\n")
	line := 5
	var rinLines []int
	for i := 0; i < events; i++ {
		fmt.Fprintf(&b, "1 BIRT\n2 RIN R%d\n", i)
		rinLines = append(rinLines, line+1)
		line += 2
	}
	b.WriteString("1 ZZZZ once\n0 TRLR\n")
	zzzzLine := line

	result, err := DecodeWithDiagnostics(strings.NewReader(b.String()), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}

	var want []string
	for _, l := range rinLines[:MaxUnknownTagDiagnosticsPerTag] {
		want = append(want, fmt.Sprintf("line %d: unknown tag: RIN", l))
	}
	want = append(want,
		fmt.Sprintf("line %d: unknown tag: ZZZZ", zzzzLine),
		fmt.Sprintf(
			"line %d: unknown tag: RIN (5 more lines with this tag, from this line on, not reported individually)",
			rinLines[MaxUnknownTagDiagnosticsPerTag]))
	got := unknownTagMessages(result.Diagnostics)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("UNKNOWN_TAG diagnostics =\n  %s\nwant\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}

	rins := 0
	for _, tag := range result.Document.GetRecord("@I1@").Tags {
		if tag.Tag == "RIN" {
			rins++
		}
	}
	if rins != events {
		t.Errorf("Record.Tags holds %d RIN lines, want all %d", rins, events)
	}

	// The corpus tests count lines through unknownTagLines; tie its parse to
	// the summary format written by unknownTagSummaries.
	perTag := map[string]int{}
	for _, d := range result.Diagnostics {
		tag, lines, ok := unknownTagLines(d)
		if d.Code == CodeUnknownTag && !ok {
			t.Errorf("unknownTagLines cannot parse %q", d.Message)
		}
		perTag[tag] += lines
	}
	if perTag["RIN"] != events || perTag["ZZZZ"] != 1 {
		t.Errorf("unknownTagLines totals = %v, want RIN:%d ZZZZ:1", perTag, events)
	}
}
