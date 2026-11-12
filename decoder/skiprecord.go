package decoder

import (
	"fmt"
	"strings"

	"github.com/cacack/gedcom-go/v3/parser"
)

// skipOrphanedRecords removes the subordinate lines of every record whose
// level-0 line the lenient parser dropped, and reports each skipped record with
// a CodeSkippedRecord diagnostic naming its line range.
//
// Without this, the lines under a dropped `0 @I2@ INDI` (a stray mid-stream
// BOM, leading junk, a bare "0" with no tag) follow the previous record's lines and
// buildRecords attaches them to that record: the second individual's NAME
// becomes the first individual's second name, and the document is plausible
// and wrong. Skipping the block loses that record, but visibly.
//
// A record's block runs from its dropped level-0 line up to, not including,
// the next kept level-0 line. Both lines and parseErrors are in input order.
// The returned slice shares no backing array with lines when any line is
// removed; otherwise it is lines itself.
func skipOrphanedRecords(lines []*parser.Line, parseErrors []*parser.ParseError, collector *diagnosticCollector) []*parser.Line {
	var kept []*parser.Line
	modified := false
	i := 0
	for _, pe := range parseErrors {
		if !isDroppedLevelZero(pe) {
			continue
		}
		// Copy the lines before the dropped line. A kept line at the error's
		// own line number means the parser recovered it, so it is not dropped.
		for i < len(lines) && lines[i].LineNumber < pe.Line {
			kept = append(kept, lines[i])
			i++
		}
		if i < len(lines) && lines[i].LineNumber == pe.Line {
			continue
		}
		modified = true
		last := pe.Line
		for i < len(lines) && lines[i].Level != 0 {
			last = lines[i].LineNumber
			i++
		}
		collector.addSkippedRecord(pe, last)
	}
	if !modified {
		return lines
	}
	return append(kept, lines[i:]...)
}

// isDroppedLevelZero reports whether pe rejected a line that was meant as a
// level-0 line: its first field is "0" once any non-digit prefix is removed,
// as in "\uFEFF0 @I2@ INDI" or "x0 HEAD". A line that is only "0"
// (no tag) counts too. The test looks at the first field alone, so prose that
// happens to contain " 0 " further along ("page 0 of 3") is not mistaken for a
// record boundary.
func isDroppedLevelZero(pe *parser.ParseError) bool {
	fields := strings.Fields(pe.Context)
	if len(fields) == 0 {
		return false
	}
	level := strings.TrimLeftFunc(fields[0], func(r rune) bool { return r < '0' || r > '9' })
	return level == "0"
}

// addSkippedRecord records that the record starting at pe's line was skipped,
// through line last. Severity is SeverityError: the record's data is lost.
func (c *diagnosticCollector) addSkippedRecord(pe *parser.ParseError, last int) {
	if !c.collecting() {
		return
	}
	lineRange := fmt.Sprintf("line %d", pe.Line)
	if last > pe.Line {
		lineRange = fmt.Sprintf("lines %d-%d", pe.Line, last)
	}
	c.add(NewDiagnostic(
		pe.Line,
		SeverityError,
		CodeSkippedRecord,
		fmt.Sprintf("record skipped (%s): its level-0 line could not be parsed, so its subordinate lines were not attached to the preceding record", lineRange),
		pe.Context,
	))
}
