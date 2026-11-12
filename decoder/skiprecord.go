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
// levelZero holds the line numbers of the dropped level-0 lines
// (diagnosticBudget.levelZero). A record's block runs from its dropped level-0
// line up to, not including, the next kept level-0 line. lines, levelZero and
// parseErrors are in input order. A dropped line whose error the budget left
// out still skips its record; its diagnostic, which could only fall past the
// bound, is then built without the line's context and counted as left out.
// The returned slice shares no backing array with lines when any line is
// removed; otherwise it is lines itself.
func skipOrphanedRecords(lines []*parser.Line, levelZero []int, parseErrors []*parser.ParseError, collector *diagnosticCollector) []*parser.Line {
	var kept []*parser.Line
	modified := false
	i, j := 0, 0
	for _, dropped := range levelZero {
		// Copy the lines before the dropped line. A kept line at the error's
		// own line number means the parser recovered it, so it is not dropped.
		for i < len(lines) && lines[i].LineNumber < dropped {
			kept = append(kept, lines[i])
			i++
		}
		if i < len(lines) && lines[i].LineNumber == dropped {
			continue
		}
		modified = true
		last := dropped
		for i < len(lines) && lines[i].Level != 0 {
			last = lines[i].LineNumber
			i++
		}
		for j < len(parseErrors) && parseErrors[j].Line < dropped {
			j++
		}
		context := ""
		if j < len(parseErrors) && parseErrors[j].Line == dropped {
			context = parseErrors[j].Context
		}
		collector.addSkippedRecord(dropped, last, context)
	}
	if !modified {
		return lines
	}
	return append(kept, lines[i:]...)
}

// isDroppedLevelZero reports whether text, a line the parser rejected, was
// meant as a level-0 line. Its first field must be "0" once any non-digit prefix is
// removed ("\uFEFF0 @I2@ INDI", "x0 HEAD"), and the rest must have the shape of
// a record line: an xref followed by an uppercase or extension tag
// ("@I2@ INDI"), or a bare HEAD or TRLR. A line that is exactly "0" counts too.
// The shape test keeps stray text inside a record ("£0 10s paid for burial",
// "page 0 of 3") from being taken for a record boundary, which would discard
// the rest of a correctly parented record.
func isDroppedLevelZero(text string) bool {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return false
	}
	if len(fields) == 1 {
		return fields[0] == "0"
	}
	level := strings.TrimLeftFunc(fields[0], func(r rune) bool { return r < '0' || r > '9' })
	if level != "0" {
		return false
	}
	if isXRef(fields[1]) {
		return len(fields) >= 3 && isTagShaped(fields[2])
	}
	return len(fields) == 2 && (fields[1] == "HEAD" || fields[1] == "TRLR")
}

// isXRef reports whether s has the shape of a cross-reference: "@" + at least
// one character + "@".
func isXRef(s string) bool {
	return len(s) >= 3 && s[0] == '@' && s[len(s)-1] == '@'
}

// isTagShaped reports whether s looks like a GEDCOM tag: an uppercase letter or
// underscore, then uppercase letters, digits or underscores.
func isTagShaped(s string) bool {
	for i, r := range s {
		switch {
		case r == '_', r >= 'A' && r <= 'Z':
		case i > 0 && r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return s != ""
}

// addSkippedRecord records that the record starting at line first was
// skipped, through line last; context is the dropped line's. Severity is
// SeverityError: the record's data is lost.
func (c *diagnosticCollector) addSkippedRecord(first, last int, context string) {
	if !c.collecting() {
		return
	}
	lineRange := fmt.Sprintf("line %d", first)
	if last > first {
		lineRange = fmt.Sprintf("lines %d-%d", first, last)
	}
	c.add(NewDiagnostic(
		first,
		SeverityError,
		CodeSkippedRecord,
		fmt.Sprintf("record skipped (%s): its level-0 line could not be parsed, so its subordinate lines were not attached to the preceding record", lineRange),
		context,
	))
}
