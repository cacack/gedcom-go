package decoder

import (
	"testing"

	"github.com/cacack/gedcom-go/v2/parser"
)

func TestNormalizeLevelJumps_NoJumps(t *testing.T) {
	lines := []*parser.Line{
		{Level: 0, Tag: "HEAD", LineNumber: 1},
		{Level: 1, Tag: "GEDC", LineNumber: 2},
		{Level: 2, Tag: "VERS", Value: "5.5.1", LineNumber: 3},
		{Level: 0, Tag: "INDI", XRef: "@I1@", LineNumber: 4},
		{Level: 1, Tag: "NAME", Value: "Test", LineNumber: 5},
		{Level: 0, Tag: "TRLR", LineNumber: 6},
	}
	collector := &diagnosticCollector{lenient: true}

	normalizeLevelJumps(lines, collector)

	if len(collector.diagnostics) != 0 {
		t.Errorf("expected no diagnostics, got %d: %v", len(collector.diagnostics), collector.diagnostics)
	}
	// Levels unchanged
	for i, l := range lines {
		want := []int{0, 1, 2, 0, 1, 0}[i]
		if l.Level != want {
			t.Errorf("line %d: Level = %d, want %d", l.LineNumber, l.Level, want)
		}
	}
}

func TestNormalizeLevelJumps_PatternA_Skip(t *testing.T) {
	// 1 BIRT then 4 DATE: jump from 1 to 4
	lines := []*parser.Line{
		{Level: 0, Tag: "INDI", XRef: "@I1@", LineNumber: 1},
		{Level: 1, Tag: "BIRT", LineNumber: 2},
		{Level: 4, Tag: "DATE", Value: "1 JAN 1900", LineNumber: 3},
	}
	collector := &diagnosticCollector{lenient: true}

	normalizeLevelJumps(lines, collector)

	if got := lines[2].Level; got != 2 {
		t.Errorf("DATE level = %d, want 2 (clamped from 4)", got)
	}
	if len(collector.diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(collector.diagnostics))
	}
	d := collector.diagnostics[0]
	if d.Code != CodeBadLevelJump {
		t.Errorf("Code = %q, want %q", d.Code, CodeBadLevelJump)
	}
	if d.Line != 3 {
		t.Errorf("Line = %d, want 3", d.Line)
	}
	if d.Severity != SeverityWarning {
		t.Errorf("Severity = %v, want %v (recovered condition; not an error)", d.Severity, SeverityWarning)
	}
}

func TestNormalizeLevelJumps_PatternB_Subordinate(t *testing.T) {
	// 1 BIRT/2 DATE/1 DEAT/3 PLAC: PLAC jumps from 1 to 3
	lines := []*parser.Line{
		{Level: 0, Tag: "INDI", XRef: "@I1@", LineNumber: 1},
		{Level: 1, Tag: "BIRT", LineNumber: 2},
		{Level: 2, Tag: "DATE", Value: "1 JAN 1900", LineNumber: 3},
		{Level: 1, Tag: "DEAT", LineNumber: 4},
		{Level: 3, Tag: "PLAC", Value: "London", LineNumber: 5},
	}
	collector := &diagnosticCollector{lenient: true}

	normalizeLevelJumps(lines, collector)

	if got := lines[4].Level; got != 2 {
		t.Errorf("PLAC level = %d, want 2 (clamped from 3)", got)
	}
	if len(collector.diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(collector.diagnostics))
	}
	if d := collector.diagnostics[0]; d.Code != CodeBadLevelJump || d.Line != 5 {
		t.Errorf("got %+v; want CodeBadLevelJump on line 5", d)
	}
}

func TestNormalizeLevelJumps_CascadingJumps(t *testing.T) {
	// 1 -> 4 -> 7: each jump clamps progressively (4->2, 7->3)
	lines := []*parser.Line{
		{Level: 0, Tag: "INDI", XRef: "@I1@", LineNumber: 1},
		{Level: 1, Tag: "BIRT", LineNumber: 2},
		{Level: 4, Tag: "DATE", LineNumber: 3},
		{Level: 7, Tag: "TIME", LineNumber: 4},
	}
	collector := &diagnosticCollector{lenient: true}

	normalizeLevelJumps(lines, collector)

	if lines[2].Level != 2 {
		t.Errorf("DATE level = %d, want 2", lines[2].Level)
	}
	if lines[3].Level != 3 {
		t.Errorf("TIME level = %d, want 3", lines[3].Level)
	}
	if len(collector.diagnostics) != 2 {
		t.Errorf("expected 2 diagnostics, got %d", len(collector.diagnostics))
	}
}

func TestNormalizeLevelJumps_GoingUpIsValid(t *testing.T) {
	// Going from level 3 back to level 1 must not be flagged.
	lines := []*parser.Line{
		{Level: 0, Tag: "INDI", XRef: "@I1@", LineNumber: 1},
		{Level: 1, Tag: "BIRT", LineNumber: 2},
		{Level: 2, Tag: "DATE", LineNumber: 3},
		{Level: 3, Tag: "TIME", LineNumber: 4},
		{Level: 1, Tag: "DEAT", LineNumber: 5},
	}
	collector := &diagnosticCollector{lenient: true}

	normalizeLevelJumps(lines, collector)

	if len(collector.diagnostics) != 0 {
		t.Errorf("expected no diagnostics, got %d: %v", len(collector.diagnostics), collector.diagnostics)
	}
}

func TestNormalizeLevelJumps_NilCollector(t *testing.T) {
	// Must not panic and must still clamp levels.
	lines := []*parser.Line{
		{Level: 0, Tag: "INDI", LineNumber: 1},
		{Level: 1, Tag: "BIRT", LineNumber: 2},
		{Level: 4, Tag: "DATE", LineNumber: 3},
	}

	normalizeLevelJumps(lines, nil)

	if lines[2].Level != 2 {
		t.Errorf("DATE level = %d, want 2", lines[2].Level)
	}
}

func TestNormalizeLevelJumps_EmptyInput(t *testing.T) {
	normalizeLevelJumps(nil, nil)
	normalizeLevelJumps([]*parser.Line{}, &diagnosticCollector{lenient: true})
}

func TestNormalizeLevelJumps_LeadingJump(t *testing.T) {
	// First non-trivial line at level 3 (no preceding HEAD/INDI): clamp to 1.
	lines := []*parser.Line{
		{Level: 3, Tag: "STRAY", LineNumber: 1},
	}
	collector := &diagnosticCollector{lenient: true}

	normalizeLevelJumps(lines, collector)

	if lines[0].Level != 1 {
		t.Errorf("STRAY level = %d, want 1", lines[0].Level)
	}
	if len(collector.diagnostics) != 1 {
		t.Errorf("expected 1 diagnostic, got %d", len(collector.diagnostics))
	}
}
