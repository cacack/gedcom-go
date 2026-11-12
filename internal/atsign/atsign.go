// Package atsign escapes and unescapes the "@" in a GEDCOM line value
// (issue #708).
//
// An "@" at the start of a line value opens a pointer ("1 NOTE @N1@"), so a
// literal one has to be written doubled. The versions disagree on how far that
// goes:
//
//   - GEDCOM 5.5 and 5.5.1 double every literal "@" ("a@@b.com"), except the
//     "@" of an escape sequence such as the "@#DJULIAN@" calendar escape.
//   - GEDCOM 7.0 doubles only a leading "@" ("@@start"); an "@" elsewhere in
//     the value is written as is.
//
// Any other version, or none, takes the 7.0 rule: doubling a leading "@" is
// what keeps text from being read as a pointer in every version. The decoder
// never applies it to a file declaring no version; it unescapes by the version
// it detects, which is the one the encoder escapes a decoded document by.
//
// The encoder escapes the values it writes from typed fields and the decoder
// unescapes the values it reads into them, so the two must agree, which is why
// the rule lives here once. Pointer fields and raw [gedcom.Tag] values go
// through neither: a pointer is not text, and raw tags are the wire form. The
// converter moves raw values from one version's wire form to another's with
// [Reescape] (issue #773).
package atsign

import (
	"strings"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// doublesEvery reports whether version v doubles every literal "@", rather
// than only a leading one.
func doublesEvery(v gedcom.Version) bool {
	return v == gedcom.Version55 || v == gedcom.Version551
}

// Escape returns s as it is written in a line value of GEDCOM version v. A
// line break in s starts a new line, written as CONT, so in 7.0 an "@" after
// one is leading too and is doubled.
//
// The result is never a pointer, under any version's grammar: a leading "@"
// is always doubled -- in 5.5 and 5.5.1 unless it opens an escape sequence,
// which is never the whole value. The 7.0 rule, which 7.0 has no escape
// sequences for, doubles a leading "@#DJULIAN@" too, so a calendar escape in
// typed text survives only as 5.5 or 5.5.1 output.
func Escape(s string, v gedcom.Version) string {
	if strings.IndexByte(s, '@') < 0 {
		return s
	}
	if doublesEvery(v) {
		return escapeEvery(s)
	}
	return escapeLeading(s)
}

// Unescape reverses [Escape]: it returns the text a line value of GEDCOM
// version v holds. In 5.5 and 5.5.1 every "@@" collapses to "@"; in 7.0 only
// a leading one does, at the start of s and after each line break. Anything
// else, a lone "@" a producer did not double included, is returned as is.
func Unescape(s string, v gedcom.Version) string {
	if !strings.Contains(s, "@@") {
		return s
	}
	if doublesEvery(v) {
		return strings.ReplaceAll(s, "@@", "@")
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if atLineStart(s, i) && strings.HasPrefix(s[i:], "@@") {
			i++
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// Reescape returns s, a line value as written in GEDCOM version from, as it is
// written in version to: Escape(Unescape(s, from), to), so it holds the same
// text in both. Only a change between the 5.5/5.5.1 rule and the 7.0 one
// changes s.
//
// One thing is not text and is kept as written: a 5.5/5.5.1 escape sequence,
// such as an "@#DROMAN@ 1700" calendar escape the converter has no 7.0
// keyword for, is not doubled when it starts a line of 7.0 output, nor is the
// "@#" that opens an unterminated one ("@#DJULIAN 1700"). s must not be a
// pointer, which is never text either.
func Reescape(s string, from, to gedcom.Version) string {
	if doublesEvery(from) == doublesEvery(to) || strings.IndexByte(s, '@') < 0 {
		return s
	}
	if doublesEvery(to) {
		return escapeEvery(Unescape(s, from))
	}
	// 5.5/5.5.1 to 7.0: a literal "@" ("@@", or a lone "@" a producer did
	// not double) is doubled only at the start of a line.
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '@' {
			b.WriteByte(s[i])
			continue
		}
		if n := escapeSequenceLen(s[i:]); n > 0 {
			b.WriteString(s[i : i+n])
			i += n - 1
			continue
		}
		if strings.HasPrefix(s[i:], "@#") {
			b.WriteByte('@')
			continue
		}
		if atLineStart(s, i) {
			b.WriteByte('@')
		}
		b.WriteByte('@')
		if strings.HasPrefix(s[i:], "@@") {
			i++
		}
	}
	return b.String()
}

// escapeLeading doubles each "@" that begins a line of s.
func escapeLeading(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 1)
	for i := 0; i < len(s); i++ {
		if s[i] == '@' && atLineStart(s, i) {
			b.WriteByte('@')
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// escapeEvery doubles each "@" of s that is not part of an escape sequence.
func escapeEvery(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	for i := 0; i < len(s); i++ {
		if s[i] != '@' {
			b.WriteByte(s[i])
			continue
		}
		if n := escapeSequenceLen(s[i:]); n > 0 {
			b.WriteString(s[i : i+n])
			i += n - 1
			continue
		}
		b.WriteString("@@")
	}
	return b.String()
}

// escapeSequenceLen returns the length of the 5.5.1 escape sequence s begins
// with, or 0 if it begins with none. The grammar's escape is "@#", at least one
// character of escape text, and a closing "@" followed by a character other
// than "@" on the same line -- so "@#DJULIAN@ 1700" begins with one, while a
// whole value (or line) of "@#X@" does not and is doubled like any other text.
func escapeSequenceLen(s string) int {
	if !strings.HasPrefix(s, "@#") {
		return 0
	}
	end := strings.IndexAny(s[2:], "@\n\r")
	if end <= 0 || s[2+end] != '@' {
		return 0
	}
	n := 2 + end + 1
	if n >= len(s) || s[n] == '@' || s[n] == '\n' || s[n] == '\r' {
		return 0
	}
	return n
}

// atLineStart reports whether byte i of s begins a line: it is the first byte,
// or follows a CR or LF.
func atLineStart(s string, i int) bool {
	return i == 0 || s[i-1] == '\n' || s[i-1] == '\r'
}
