package decoder

import (
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/parser"
)

// buildLayout returns the layout of src, the decoded text the parser read,
// given the lines it parsed from it (see DecodeOptions.PreserveLayout).
//
// src is split into physical lines by the parser's own rule -- LF, CRLF or a
// bare CR ends a line -- so line n here is the line the parser numbered n. A
// line the parser returned is recorded as it parsed; one it skipped is carried
// in the next parsed line's Before, or in Tail at the end. Every string is a
// substring of src, so the layout costs one copy of the input plus a record
// per line.
//
// lines must be the parser's output, before lenient recovery edits them: a
// recorded line is written back only where it still reads as what its text
// says, and a clamped level is not what its text says.
func buildLayout(src string, lines []*parser.Line, bom bool) *gedcom.Layout {
	layout := &gedcom.Layout{BOM: bom}

	for _, line := range lines {
		if line.Level != 0 || line.XRef != "" {
			continue
		}
		switch line.Tag {
		case "HEAD":
			if layout.Head == 0 {
				layout.Head = line.LineNumber
			}
		case "TRLR":
			layout.Trailer = line.LineNumber
		}
	}

	// lines are in line-number order, so one pass over both finds each.
	next := 0        // index into lines of the first not yet reached
	skippedFrom := 0 // start of the skipped lines not yet attached to a line
	for start, n := 0, 1; start < len(src); n++ {
		textEnd, lineEnd := splitLine(src, start)
		layout.Lines = append(layout.Lines, gedcom.LayoutLine{})
		for next < len(lines) && lines[next].LineNumber < n {
			next++
		}
		if next < len(lines) && lines[next].LineNumber == n {
			line := lines[next]
			layout.Lines[n-1] = gedcom.LayoutLine{
				Level:      line.Level,
				XRef:       line.XRef,
				Tag:        line.Tag,
				Value:      line.Value,
				Before:     src[skippedFrom:start],
				Text:       src[start:textEnd],
				Terminator: src[textEnd:lineEnd],
			}
			skippedFrom = lineEnd
		}
		start = lineEnd
	}
	layout.Tail = src[skippedFrom:]
	return layout
}

// splitLine returns where the line starting at start ends, and where the next
// one starts: the two differ by its terminator, which is LF, CRLF, a bare CR,
// or nothing at the end of src. It is the split parser.lineScanner applies.
func splitLine(src string, start int) (textEnd, next int) {
	for i := start; i < len(src); i++ {
		switch src[i] {
		case '\n':
			return i, i + 1
		case '\r':
			if i+1 < len(src) && src[i+1] == '\n' {
				return i, i + 2
			}
			return i, i + 1
		}
	}
	return len(src), len(src)
}
