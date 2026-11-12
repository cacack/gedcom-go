package decoder

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/parser"
)

// TestDecodeWithDiagnostics_DroppedLevelZeroSkipsRecord pins issue #723: when
// lenient parsing drops a level-0 line, the lines under it are skipped and
// reported, rather than attached to the preceding record.
func TestDecodeWithDiagnostics_DroppedLevelZeroSkipsRecord(t *testing.T) {
	tests := []struct {
		name    string
		bad     string // the level-0 line the parser drops
		context string // the diagnostic context it reports
	}{
		{"mid-stream BOM (concatenated exports)", "\uFEFF0 @I2@ INDI", "\uFEFF0 @I2@ INDI"},
		{"leading junk", "xx0 @I2@ INDI", "xx0 @I2@ INDI"},
		{"level-0 line with no tag", "0", "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n" +
				"0 @I1@ INDI\n1 NAME John /Smith/\n" +
				tt.bad + "\n1 NAME Jane /Doe/\n2 GIVN Jane\n1 SEX F\n" +
				"0 @I3@ INDI\n1 NAME Ann /Lee/\n0 TRLR\n"

			t.Run("strict mode fails at the line", func(t *testing.T) {
				_, err := DecodeWithOptions(strings.NewReader(input), &DecodeOptions{StrictMode: true})
				if err == nil || !strings.Contains(err.Error(), "line 6") {
					t.Fatalf("strict decode error = %v, want a failure at line 6", err)
				}
			})

			result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			doc := result.Document

			john := doc.GetIndividual("@I1@")
			if john == nil {
				t.Fatal("@I1@ should exist")
			}
			if len(john.Names) != 1 || john.Names[0].Full != "John /Smith/" {
				t.Errorf("@I1@ Names = %+v, want only \"John /Smith/\" (Jane must not be merged in)", john.Names)
			}
			if john.Sex != "" {
				t.Errorf("@I1@ Sex = %q, want empty (the skipped record's SEX must not attach)", john.Sex)
			}
			for _, tag := range doc.XRefMap["@I1@"].Tags {
				if tag.LineNumber > 5 {
					t.Errorf("@I1@ holds line %d (%s %s) from the skipped record", tag.LineNumber, tag.Tag, tag.Value)
				}
			}
			ann := doc.GetIndividual("@I3@")
			if ann == nil || len(ann.Names) != 1 || ann.Names[0].Full != "Ann /Lee/" {
				t.Errorf("@I3@ = %+v, want the record after the skipped block intact", ann)
			}

			var skipped []Diagnostic
			for _, d := range result.Diagnostics {
				if d.Code == CodeSkippedRecord {
					skipped = append(skipped, d)
				}
			}
			if len(skipped) != 1 {
				t.Fatalf("SKIPPED_RECORD diagnostics = %v, want exactly one", skipped)
			}
			d := skipped[0]
			if d.Line != 6 || d.Severity != SeverityError || d.Context != tt.context {
				t.Errorf("diagnostic = %s, want line 6 at ERROR with context %q", d.String(), tt.context)
			}
			if !strings.Contains(d.Message, "lines 6-9") {
				t.Errorf("message = %q, want it to name the skipped range \"lines 6-9\"", d.Message)
			}
		})
	}
}

func TestSkipOrphanedRecords(t *testing.T) {
	lines := []*parser.Line{
		{Level: 0, Tag: "INDI", XRef: "@I1@", LineNumber: 1},
		{Level: 1, Tag: "NAME", Value: "A", LineNumber: 2},
		// line 3 dropped: "\uFEFF0 @I2@ INDI"
		{Level: 1, Tag: "NAME", Value: "B", LineNumber: 4},
		// line 5 dropped, not a level-0 line: "page 0 of 3"
		{Level: 1, Tag: "SEX", Value: "F", LineNumber: 6},
		// line 7 dropped with nothing under it: "x0 @I3@ INDI"
		{Level: 0, Tag: "INDI", XRef: "@I4@", LineNumber: 8},
		{Level: 0, Tag: "INDI", XRef: "@I5", LineNumber: 9}, // recovered by the parser
		{Level: 1, Tag: "NAME", Value: "E", LineNumber: 10},
	}
	errs := []*parser.ParseError{
		{Line: 3, Message: "invalid level number", Context: "\uFEFF0 @I2@ INDI"},
		{Line: 5, Message: "invalid level number", Context: "page 0 of 3"},
		{Line: 7, Message: "invalid level number", Context: "x0 @I3@ INDI"},
		{Line: 9, Message: "xref is missing its closing @", Context: "0 @I5 INDI"},
	}
	collector := &diagnosticCollector{}

	got := skipOrphanedRecords(lines, errs, collector)

	var nums []int
	for _, l := range got {
		nums = append(nums, l.LineNumber)
	}
	want := []int{1, 2, 8, 9, 10}
	if len(nums) != len(want) {
		t.Fatalf("kept lines %v, want %v", nums, want)
	}
	for i := range want {
		if nums[i] != want[i] {
			t.Fatalf("kept lines %v, want %v", nums, want)
		}
	}
	if len(collector.diagnostics) != 2 {
		t.Fatalf("diagnostics = %v, want two SKIPPED_RECORD", collector.diagnostics)
	}
	if d := collector.diagnostics[0]; d.Line != 3 || !strings.Contains(d.Message, "lines 3-6") {
		t.Errorf("first diagnostic = %s, want line 3 naming lines 3-6", d.String())
	}
	if d := collector.diagnostics[1]; d.Line != 7 || !strings.Contains(d.Message, "(line 7)") {
		t.Errorf("second diagnostic = %s, want line 7 naming only line 7", d.String())
	}
}

func TestSkipOrphanedRecords_NothingDropped(t *testing.T) {
	lines := []*parser.Line{
		{Level: 0, Tag: "INDI", XRef: "@I1@", LineNumber: 1},
		{Level: 1, Tag: "NAME", Value: "A", LineNumber: 3},
	}
	errs := []*parser.ParseError{{Line: 2, Message: "empty line", Context: ""}}

	got := skipOrphanedRecords(lines, errs, nil)

	if len(got) != 2 || got[0] != lines[0] || got[1] != lines[1] {
		t.Errorf("got %v, want lines unchanged", got)
	}
}
