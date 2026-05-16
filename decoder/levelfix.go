package decoder

import (
	"fmt"

	"github.com/cacack/gedcom-go/parser"
)

// normalizeLevelJumps detects malformed indentation in real-world GEDCOM exports
// and recovers from it. When a line's level skips one or more steps from the
// previous line (e.g., 1 -> 4 instead of 1 -> 2), the level is clamped to
// prevLevel+1 so the subordinate tag attaches to its natural parent, and a
// CodeBadLevelJump diagnostic is emitted if a collector is provided.
//
// Going up the hierarchy (e.g., 3 -> 1) is always valid: it closes child scopes
// naturally and is not flagged. Cascading jumps (e.g., 1 -> 4 -> 7) are clamped
// progressively so each subordinate attaches one level below the previous, which
// preserves nested structure correctly.
//
// Lines are mutated in place.
func normalizeLevelJumps(lines []*parser.Line, collector *diagnosticCollector) {
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
func (c *diagnosticCollector) addBadLevelJump(lineNumber, origLevel, prevLevel int, tag string) {
	if c == nil {
		return
	}
	c.add(NewParseError(
		lineNumber,
		CodeBadLevelJump,
		fmt.Sprintf("level jumped from %d to %d (skipping %d level(s)); clamped to %d for recovery",
			prevLevel, origLevel, origLevel-prevLevel-1, prevLevel+1),
		tag,
	))
}
