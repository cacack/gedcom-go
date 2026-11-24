package decoder

import (
	"sort"
	"strings"

	"github.com/cacack/gedcom-go/v3/internal/clip"
	"github.com/cacack/gedcom-go/v3/parser"
)

// locateDiagnostics sets XRef and Path on every diagnostic in ds from the
// line its Line names (#909). Doing it here, once the list is complete, covers
// every diagnostic whatever built it: parse errors, encoding repairs, entity
// diagnostics and the ones that count lines not reported individually.
//
// lines are the decoded lines, in input order, with their recovered levels.
// droppedLevelZero holds the line numbers of the level-0 lines the parser
// dropped (diagnosticBudget.levelZero): each ends the record before it, and
// the lines up to the next kept level-0 line belong to no record.
func locateDiagnostics(ds Diagnostics, lines []*parser.Line, droppedLevelZero []int) {
	wanted := make([]int, 0, len(ds))
	for i := range ds {
		if ds[i].Line > 0 {
			wanted = append(wanted, ds[i].Line)
		}
	}
	sort.Ints(wanted)

	type location struct{ xref, path string }
	found := make(map[int]location, len(wanted))
	l := lineLocator{lines: lines, dropped: droppedLevelZero}
	for _, w := range wanted {
		if _, ok := found[w]; !ok {
			xref, path := l.locate(w)
			found[w] = location{xref, path}
		}
	}
	for i := range ds {
		if loc, ok := found[ds[i].Line]; ok {
			ds[i].XRef, ds[i].Path = loc.xref, loc.path
		}
	}
}

// lineLocator walks a decode's lines once, answering locate for ascending
// line numbers.
type lineLocator struct {
	lines   []*parser.Line
	dropped []int // line numbers of the dropped level-0 lines

	i, k int      // next of lines and of dropped
	xref string   // of the record lines[i-1] is in
	path []string // tags from that record down to lines[i-1]; empty before the first record
}

// locate returns the XRef and Path of line number n, which must not be lower
// than the one last asked for.
func (l *lineLocator) locate(n int) (xref, path string) {
	for ; l.i < len(l.lines) && l.lines[l.i].LineNumber <= n; l.i++ {
		line := l.lines[l.i]
		switch {
		case line.Level == 0:
			l.path, l.xref = append(l.path[:0], line.Tag), line.XRef
		case len(l.path) > 0:
			// Levels are contiguous once normalizeLevelJumps has run.
			l.path = append(l.path[:min(line.Level, len(l.path))], line.Tag)
		}
	}
	last := 0
	if l.i > 0 {
		last = l.lines[l.i-1].LineNumber
	}
	for l.k < len(l.dropped) && l.dropped[l.k] <= last {
		l.k++
	}
	switch {
	case l.k < len(l.dropped) && l.dropped[l.k] <= n, len(l.path) == 0:
		// In a skipped record, or before the first record.
		return "", ""
	case last == n:
		return l.xref, joinPath(l.path)
	default:
		// A line the parser dropped: its own tag is not known.
		return l.xref, joinPath(l.path[:1])
	}
}

// joinPath joins tags with "." into a Diagnostic.Path, clipped like Message
// to parser.MaxContextBytes. A tag can be as long as a line and a path 100
// tags deep, so no more than the bound is ever copied: every diagnostic gets
// its own Path.
func joinPath(tags []string) string {
	var b strings.Builder
	for i, tag := range tags {
		if i > 0 {
			b.WriteByte('.')
		}
		// One byte past the bound is enough for clip.String to see the excess.
		if room := parser.MaxContextBytes + 1 - b.Len(); len(tag) >= room {
			b.WriteString(tag[:room])
			break
		}
		b.WriteString(tag)
	}
	return clip.String(b.String(), parser.MaxContextBytes)
}
