package gedcom

import "strings"

// SplitPersonalName splits a NAME value, "Given /Surname/ Suffix" in the
// 5.5.1 NAME_PERSONAL grammar, into its given name, surname and suffix. The
// decoder seeds PersonalName.Given, Surname and Suffix with it before GIVN,
// SURN and NSFX override them, so calling it on PersonalName.Full gives what
// the value alone says, to compare with the name piece lines.
//
// The surname is the text between the last two slashes, kept as written; the
// given name is the text before the first slash and the suffix the text after
// the last, both trimmed. A value with one slash has no closing slash, so the
// rest of it is the surname; one with none is all given name. Text between
// earlier slashes ("John /de la /Smith/") is malformed and is kept only in
// PersonalName.Full. A slash inside the suffix cannot be told from a closing
// one, so "John /Smith/ Jr./Sr" reads as surname " Jr." and suffix "Sr". An
// unclosed surname is trimmed, as it has no closing slash to keep it.
//
// Every package that reads a surname out of a NAME value uses it
// (spec-interpretation policy SI-009). It splits a value the PersonalName
// grammar rejects, one with a single slash or three, the same way, and no
// package reports such a value (SI-020).
func SplitPersonalName(full string) (given, surname, suffix string) {
	first := strings.Index(full, "/")
	if first < 0 {
		return strings.TrimSpace(full), "", ""
	}
	given = strings.TrimSpace(full[:first])
	last := strings.LastIndex(full, "/")
	if last == first {
		return given, strings.TrimSpace(full[first+1:]), ""
	}
	open := strings.LastIndex(full[:last], "/")
	return given, full[open+1 : last], strings.TrimSpace(full[last+1:])
}
