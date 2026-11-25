// Package concsplit splits a GEDCOM line value into the pieces written as
// CONT and CONC lines.
//
// The encoder splits values it writes from typed entities, gedcom's
// Record.SetFullValue splits a raw tag's value, and the converter splits the
// raw tags of a 7.0 document downgraded to 5.x (issue #671). They must cut in
// the same places, so the rule lives here once.
package concsplit

import (
	"strings"
	"unicode/utf8"
)

// MaxValueLength is the longest value, in bytes, written on one line. GEDCOM
// 5.x limits a line to 255 characters; 248 leaves room for the level, tag and
// delimiters.
const MaxValueLength = 248

// Segment is one physical line of a value written across continuation lines.
// Tag is "" for the value's first line, which belongs to the tag that owns the
// value, and "CONT" or "CONC" for each later one.
type Segment struct {
	Tag   string
	Value string
}

// Fold splits value into the lines it is written as: a CONT for each line
// break, and, when maxLen is positive, a CONC for each piece of a line longer
// than maxLen bytes (cut by [Split]). A maxLen of 0 or less cuts no CONC, as
// for GEDCOM 7.0, which has none. escape, when non-nil, is applied to each line
// before it is cut, so the escaped line is what has to fit. Fold returns at
// least one segment, the first with an empty Tag.
func Fold(value string, maxLen int, escape func(string) string) []Segment {
	lines := Lines(value)
	segments := make([]Segment, 0, len(lines))
	for i, line := range lines {
		if escape != nil {
			line = escape(line)
		}
		pieces := []string{line}
		if maxLen > 0 {
			pieces = Split(line, maxLen)
		}
		tag := ""
		if i > 0 {
			tag = "CONT"
		}
		segments = append(segments, Segment{Tag: tag, Value: pieces[0]})
		for _, piece := range pieces[1:] {
			segments = append(segments, Segment{Tag: "CONC", Value: piece})
		}
	}
	return segments
}

// Lines splits a tag value on line breaks, returning at least one element.
// All three terminators the parser recognizes are normalized first -- CRLF,
// bare LF and bare CR -- so that re-encoding is the exact inverse of reading:
// whichever form the source used, the parser saw a line break there, and a
// CONT is how that line break is written back. Callers on the hot path should
// skip it entirely for a value with no line break rather than rely on it
// returning a single element, so that the common case allocates nothing.
func Lines(value string) []string {
	if !strings.ContainsAny(value, "\n\r") {
		return []string{value}
	}
	normalized := strings.ReplaceAll(value, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	return strings.Split(normalized, "\n")
}

// Split returns line cut into pieces of at most maxLen bytes, each later
// piece to be written as a CONC line. It returns at least one element, and
// the pieces concatenate back to line. A maxLen below 1 is treated as 1, so
// every piece holds at least one character and the split always ends.
func Split(line string, maxLen int) []string {
	maxLen = max(maxLen, 1)
	if len(line) <= maxLen {
		return []string{line}
	}

	var segments []string
	remaining := line
	for len(remaining) > maxLen {
		splitAt := splitPoint(remaining, maxLen)
		segments = append(segments, remaining[:splitAt])
		remaining = remaining[splitAt:]
	}
	if remaining != "" {
		segments = append(segments, remaining)
	}
	return segments
}

// splitPoint returns where to cut line so the first piece is at most
// maxLen bytes: the last offset at or before maxLen that starts a UTF-8
// character, has a non-space on both sides and is not inside an "@@" pair.
//
// GEDCOM 5.5.1 requires a CONC split to fall between two non-space
// characters, because readers that trim leading or trailing whitespace on a
// line would otherwise lose the space (issue #565). A cut inside a multi-byte
// character would leave both lines holding invalid UTF-8. A cut inside an
// "@@" pair, the escape for a literal "@" (issue #708), would leave a lone
// "@" on each line, which a reader escaping line by line misreads.
//
// A line with no such offset in range (a run of spaces longer than maxLen)
// falls back to the last character boundary, and a first character longer
// than maxLen is kept whole, so the caller's loop always makes progress.
func splitPoint(line string, maxLen int) int {
	if len(line) <= maxLen {
		return len(line)
	}

	fallback := 0
	for i := maxLen; i > 0; i-- {
		if !utf8.RuneStart(line[i]) {
			continue
		}
		if fallback == 0 {
			fallback = i
		}
		if line[i-1] != ' ' && line[i] != ' ' && !splitsAtPair(line, i) {
			return i
		}
	}
	if fallback > 0 {
		return fallback
	}
	_, size := utf8.DecodeRuneInString(line)
	return size
}

// splitsAtPair reports whether offset i of line falls inside an "@@" pair: an
// odd number of bytes into a run of "@", whose bytes pair off from its start.
func splitsAtPair(line string, i int) bool {
	if line[i-1] != '@' || line[i] != '@' {
		return false
	}
	start := i - 1
	for start > 0 && line[start-1] == '@' {
		start--
	}
	return (i-start)%2 == 1
}
