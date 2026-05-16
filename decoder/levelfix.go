package decoder

import (
	"fmt"

	"github.com/cacack/gedcom-go/parser"
)

// normalizeLevelJumps detects malformed indentation in real-world GEDCOM exports
// and clamps over-jumped levels so subordinate tags attach to a valid parent
// where one exists. When a line's level skips one or more steps from the
// previous line (e.g., 1 -> 4 instead of 1 -> 2), the level is clamped to
// prevLevel+1 and a CodeBadLevelJump diagnostic is emitted if a collector is
// provided.
//
// Recovery is best-effort: when a leading line jumps from the implicit level-0
// start (e.g., the first parseable line is at level 3), clamping to level 1
// produces a tag that has no level-0 record to attach to, and buildRecords
// will drop it. The diagnostic is still emitted in that case so the caller can
// see the bad input.
//
// Going up the hierarchy (e.g., 3 -> 1) is always valid: it closes child scopes
// naturally and is not flagged. Cascading jumps (e.g., 1 -> 4 -> 7) are clamped
// progressively so each subordinate attaches one level below the previous, which
// preserves nested structure correctly.
//
// Lines are mutated in place.
func normalizeLevelJumps(lines []*parser.Line, collector *diagnosticCollector) {
	// GEDCOM level 0 is the document root; the first valid line in any record
	// starts at level 0. Initializing prev to 0 means any leading line above
	// level 1 is treated as a forward jump and clamped.
	prev := 0
	for _, line := range lines {
		if line.Level > prev+1 {
			if collector != nil {
				collector.addBadLevelJump(line.LineNumber, line.Level, prev, line.Tag)
			}
			line.Level = prev + 1
		}
		prev = line.Level
	}
}

// addBadLevelJump records a diagnostic for a malformed indentation jump.
// The level is clamped to prevLevel+1 by the caller; this method only reports.
//
// Severity is SeverityWarning, not SeverityError: clamping is a successful
// recovery — the document is built correctly and remains usable. Reserve
// SeverityError for diagnostics where data was lost.
func (c *diagnosticCollector) addBadLevelJump(lineNumber, origLevel, prevLevel int, tag string) {
	if c == nil {
		return
	}
	c.add(NewDiagnostic(
		lineNumber,
		SeverityWarning,
		CodeBadLevelJump,
		fmt.Sprintf("level jumped from %d to %d (skipping %d level(s)); clamped to %d for recovery",
			prevLevel, origLevel, origLevel-prevLevel-1, prevLevel+1),
		tag,
	))
}
