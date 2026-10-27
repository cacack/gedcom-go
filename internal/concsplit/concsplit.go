// Package concsplit splits an over-length GEDCOM 5.x line value into the
// pieces written as CONC lines.
//
// The encoder splits values it writes from typed entities, and the converter
// splits the raw tags of a 7.0 document downgraded to 5.x (issue #671). Both
// must cut in the same places, so the rule lives here once.
package concsplit

import "unicode/utf8"

// MaxValueLength is the longest value, in bytes, written on one line. GEDCOM
// 5.x limits a line to 255 characters; 248 leaves room for the level, tag and
// delimiters.
const MaxValueLength = 248

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
// character and has a non-space on both sides.
//
// GEDCOM 5.5.1 requires a CONC split to fall between two non-space
// characters, because readers that trim leading or trailing whitespace on a
// line would otherwise lose the space (issue #565). A cut inside a multi-byte
// character would leave both lines holding invalid UTF-8.
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
		if line[i-1] != ' ' && line[i] != ' ' {
			return i
		}
	}
	if fallback > 0 {
		return fallback
	}
	_, size := utf8.DecodeRuneInString(line)
	return size
}
