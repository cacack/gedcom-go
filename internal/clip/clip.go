// Package clip bounds the input text a diagnostic or error echoes back, so
// that a hostile or corrupt line is not copied whole into every error string
// that reports it (issue #744).
//
// The parser clips ParseError.Context and the decoder clips each Diagnostic's
// Message and Context. Both must clip the same way, so the rule lives here
// once.
package clip

import "unicode/utf8"

// Ellipsis marks the end of clipped text.
const Ellipsis = "…"

// String returns s unchanged when it is at most maxBytes long. Otherwise it
// returns the longest prefix of s that is at most maxBytes long and does not
// end inside a UTF-8 character, followed by Ellipsis. The result is a fresh
// copy, so it does not keep s's backing memory alive.
func String(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	cut := max(maxBytes, 0)
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	// Concatenation allocates, so the prefix is copied rather than shared.
	return s[:cut] + Ellipsis
}
