package decoder

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/parser"
)

// locatedDiagnostic is what the location tests compare: where a diagnostic
// is and what it is, without its message.
type locatedDiagnostic struct {
	Line int
	Code string
	XRef string
	Path string
}

func locatedDiagnostics(ds Diagnostics) []locatedDiagnostic {
	out := make([]locatedDiagnostic, 0, len(ds))
	for _, d := range ds {
		out = append(out, locatedDiagnostic{d.Line, d.Code, d.XRef, d.Path})
	}
	return out
}

func assertLocated(t *testing.T, input string, opts *DecodeOptions, want []locatedDiagnostic) {
	t.Helper()
	result, err := DecodeWithDiagnostics(strings.NewReader(input), opts)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	got := locatedDiagnostics(result.Diagnostics)
	if len(got) != len(want) {
		t.Fatalf("got %d diagnostics, want %d:\n%s", len(got), len(want), result.Diagnostics)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("diagnostic %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestDiagnosticLocation pins issue #909: a decode diagnostic names the
// record its line is in and the tag path down to that line.
func TestDiagnosticLocation(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []locatedDiagnostic
	}{
		{
			name: "invalid value on a nested DATE",
			input: "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n" +
				"0 @I1@ INDI\n1 NAME Ann /Example/\n1 BURI\n2 DATE Example Memorial Gardens\n2 PLAC Springfield\n" +
				"0 TRLR\n",
			want: []locatedDiagnostic{{7, CodeInvalidValue, "@I1@", "INDI.BURI.DATE"}},
		},
		{
			name: "non-standard date",
			input: "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n" +
				"0 @I1@ INDI\n1 BIRT\n2 DATE June 1880\n" +
				"0 @F1@ FAM\n1 MARR\n2 DATE about 1873\n" +
				"0 TRLR\n",
			want: []locatedDiagnostic{
				{6, CodeNonStandardDate, "@I1@", "INDI.BIRT.DATE"},
				{9, CodeNonStandardDate, "@F1@", "FAM.MARR.DATE"},
			},
		},
		{
			name: "header",
			input: "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FOO bar\n" +
				"0 TRLR\n",
			want: []locatedDiagnostic{{4, CodeUnknownTag, "", "HEAD.GEDC.FOO"}},
		},
		{
			name: "level-0 lines",
			input: "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n" +
				"0 @I1@ INDI\n" +
				"0 @I1@ INDI\n" +
				"0 @X1@ FOO\n" +
				"0 TRLR\n",
			want: []locatedDiagnostic{
				{5, CodeDuplicateXRef, "@I1@", "INDI"},
				{6, CodeUnknownTag, "@X1@", "FOO"},
			},
		},
		{
			name: "record without an identifier",
			input: "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n" +
				"0 INDI\n1 FOO bar\n" +
				"0 TRLR\n",
			want: []locatedDiagnostic{{5, CodeUnknownTag, "", "INDI.FOO"}},
		},
		{
			name: "clamped level jump uses the recovered level",
			input: "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n" +
				"0 @I1@ INDI\n1 BIRT\n4 DATE 1 JAN 1900\n1 DEAT\n2 PLAC Springfield\n3 FOO bar\n" +
				"0 TRLR\n",
			want: []locatedDiagnostic{
				{6, CodeBadLevelJump, "@I1@", "INDI.BIRT.DATE"},
				{9, CodeUnknownTag, "@I1@", "INDI.DEAT.PLAC.FOO"},
			},
		},
		{
			name: "dropped line inside a record gets the record alone",
			input: "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n" +
				"0 @I1@ INDI\n1 BIRT\nx DATE 1900\n\n" +
				"0 TRLR\n",
			want: []locatedDiagnostic{
				{6, CodeInvalidLevel, "@I1@", "INDI"},
				{7, CodeEmptyLine, "@I1@", "INDI"},
			},
		},
		{
			name: "skipped record belongs to no record",
			input: "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n" +
				"0 @I1@ INDI\n1 NAME Ann /Example/\n" +
				"x0 @I2@ INDI\n1 NAME Bob /Example/\n\n" +
				"0 @I3@ INDI\n1 FOO bar\n" +
				"0 TRLR\n",
			want: []locatedDiagnostic{
				{6, CodeInvalidLevel, "", ""},
				{8, CodeEmptyLine, "", ""},
				{6, CodeSkippedRecord, "", ""},
				{10, CodeUnknownTag, "@I3@", "INDI.FOO"},
			},
		},
		{
			name:  "line before the first record",
			input: "3 FOO bar\n0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 TRLR\n\n",
			want: []locatedDiagnostic{
				{6, CodeEmptyLine, "", "TRLR"},
				{1, CodeBadLevelJump, "", ""},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertLocated(t, tt.input, nil, tt.want)
		})
	}
}

// TestDiagnosticLocation_Summaries pins that a diagnostic counting lines not
// reported individually is located at its Line, the first of them, and that
// the truncation notice, at Line 0, has no location.
func TestDiagnosticLocation_Summaries(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("0 HEAD\n1 GEDC\n2 VERS 5.5.1\n")
	for i := 1; i <= MaxUnknownTagDiagnosticsPerTag+2; i++ {
		fmt.Fprintf(&sb, "0 @I%d@ INDI\n1 BIRT\n2 FOO bar\n", i)
	}
	sb.WriteString("0 TRLR\n")

	result, err := DecodeWithDiagnostics(strings.NewReader(sb.String()), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	if len(result.Diagnostics) != MaxUnknownTagDiagnosticsPerTag+1 {
		t.Fatalf("got %d diagnostics, want %d:\n%s", len(result.Diagnostics), MaxUnknownTagDiagnosticsPerTag+1, result.Diagnostics)
	}
	for i, d := range result.Diagnostics {
		want := locatedDiagnostic{3 + 3*(i+1), CodeUnknownTag, fmt.Sprintf("@I%d@", i+1), "INDI.BIRT.FOO"}
		if got := locatedDiagnostics(Diagnostics{d})[0]; got != want {
			t.Errorf("diagnostic %d = %+v, want %+v", i, got, want)
		}
	}
	if last := result.Diagnostics[MaxUnknownTagDiagnosticsPerTag]; !strings.Contains(last.Message, "2 more lines") {
		t.Errorf("last diagnostic is not the summary: %s", last)
	}

	assertLocated(t, sb.String(), &DecodeOptions{MaxDiagnostics: 2}, []locatedDiagnostic{
		{6, CodeUnknownTag, "@I1@", "INDI.BIRT.FOO"},
		{9, CodeUnknownTag, "@I2@", "INDI.BIRT.FOO"},
		{0, CodeDiagnosticsTruncated, "", ""},
	})
}

// TestLocateDiagnostics_NonContiguousLevels covers lines whose levels were
// not normalized: a deeper line attaches to the deepest open structure.
func TestLocateDiagnostics_NonContiguousLevels(t *testing.T) {
	lines := []*parser.Line{
		{Level: 0, XRef: "@I1@", Tag: "INDI", LineNumber: 1},
		{Level: 3, Tag: "DATE", LineNumber: 2},
	}
	ds := Diagnostics{NewDiagnostic(2, SeverityWarning, CodeInvalidValue, "m", "")}
	locateDiagnostics(ds, lines, nil)
	if ds[0].XRef != "@I1@" || ds[0].Path != "INDI.DATE" {
		t.Errorf("got XRef %q Path %q, want @I1@ INDI.DATE", ds[0].XRef, ds[0].Path)
	}
}

// TestDiagnosticLocation_PathIsBounded pins that Path is clipped like Message
// and Context: long tags at depth must not give every diagnostic its own
// copy of an arbitrarily long path.
func TestDiagnosticLocation_PathIsBounded(t *testing.T) {
	const depth, repeats = 20, 50
	var sb strings.Builder
	sb.WriteString("0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @I1@ INDI\n")
	sb.WriteString("1 _A" + strings.Repeat("É", 100) + "\n")
	for level := 2; level <= depth; level++ {
		fmt.Fprintf(&sb, "%d _%s\n", level, strings.Repeat("É", 100))
	}
	for range repeats {
		fmt.Fprintf(&sb, "%d A\n%d B\n", depth, depth+2)
	}
	sb.WriteString("0 TRLR\n")

	result, err := DecodeWithDiagnostics(strings.NewReader(sb.String()), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	// The first two tags and 23 two-byte characters of the third make 255
	// bytes; the next character would straddle the bound, so the cut falls
	// back to the character boundary.
	want := "INDI._A" + strings.Repeat("É", 100) + "._" + strings.Repeat("É", 23) + "…"
	jumps := 0
	for _, d := range result.Diagnostics {
		if d.Code != CodeBadLevelJump {
			continue
		}
		jumps++
		if d.XRef != "@I1@" || d.Path != want {
			t.Fatalf("line %d: XRef %q, Path %q (%d bytes), want @I1@ and %q", d.Line, d.XRef, d.Path, len(d.Path), want)
		}
	}
	if jumps != repeats {
		t.Errorf("got %d %s diagnostics, want %d", jumps, CodeBadLevelJump, repeats)
	}
}

// TestJoinPath covers the clipping edges of a diagnostic's Path.
func TestJoinPath(t *testing.T) {
	long := strings.Repeat("X", parser.MaxContextBytes)
	tests := []struct {
		name string
		tags []string
		want string
	}{
		{"short", []string{"INDI", "BURI", "DATE"}, "INDI.BURI.DATE"},
		{"exactly the bound", []string{long}, long},
		{"one tag over the bound", []string{long + "Y"}, long + "…"},
		{"separator at the bound", []string{long, "DATE"}, long + "…"},
		{"cut inside a later tag", []string{long[:250], "ABCDEFGH"}, long[:250] + ".ABCDE…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := joinPath(tt.tags); got != tt.want {
				t.Errorf("joinPath() = %q, want %q", got, tt.want)
			}
		})
	}
}
