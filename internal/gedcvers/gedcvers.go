// Package gedcvers maps a GEDC.VERS value to the GEDCOM version it names.
//
// The version package reads the version a file declares, and the encoder
// resolves the version it was asked to write; both must agree on what a value
// such as 7.0.14 or 7.1 means, so the rule lives here once (issues #587, #696,
// #741).
package gedcvers

import (
	"strings"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Resolution is what a GEDC.VERS value names. The zero value means the value
// is not recognized.
type Resolution struct {
	// Version is the version whose rules the file is read by: 5.5, 5.5.1 or
	// 7.0. Every GEDCOM 7 minor and patch version is 7.0, as the GEDCOM 7
	// version guide asks: a later minor is read as the same major, its new
	// structures as extensions.
	Version gedcom.Version

	// Declared is the version the value declares, without a patch level:
	// Version, except for a GEDCOM 7 minor after 7.0, which keeps its own
	// major.minor ("7.1" for VERS 7.1 or 7.1.0).
	Declared gedcom.Version

	// Normalized reports that the value is not a conforming version string
	// (5.51, 551, 5.5.5, 7, or a version followed by other text) and was read
	// as the version it plainly means.
	Normalized bool
}

// Parse maps a GEDC.VERS value to the version whose rules apply, or "" when it
// is not recognized; see [Resolve].
func Parse(value string) gedcom.Version {
	return Resolve(value).Version
}

// Resolve maps a GEDC.VERS value, ignoring surrounding whitespace, to what it
// names. Conforming values are 5.5, 5.5.1, and 7.N or 7.N.M with N and M ASCII
// digits (leading zeros accepted). Spellings vendors write for a known version
// are read as that version with Normalized set: 5.51 and 551 are 5.5.1, 5.5.5
// (a later 5.5.1 profile) is 5.5.1, 55 is 5.5, and 7 is 7.0; so is any of
// these or a conforming value followed by whitespace and other text, such as
// "5.5.1 (GEDCOM)". Anything else (7.0., 7.0.x, 7.0.1.2, 8.0) is unrecognized
// rather than guessed.
func Resolve(value string) Resolution {
	trimmed := strings.TrimSpace(value)
	end := strings.IndexAny(trimmed, " \t")
	if end < 0 {
		return resolveToken(trimmed)
	}
	r := resolveToken(trimmed[:end])
	r.Normalized = r.Version != ""
	return r
}

// resolveToken resolves a value holding no whitespace.
func resolveToken(token string) Resolution {
	switch token {
	case "5.5":
		return Resolution{Version: gedcom.Version55, Declared: gedcom.Version55}
	case "5.5.1":
		return Resolution{Version: gedcom.Version551, Declared: gedcom.Version551}
	case "55":
		return Resolution{Version: gedcom.Version55, Declared: gedcom.Version55, Normalized: true}
	case "5.51", "551", "5.5.5":
		return Resolution{Version: gedcom.Version551, Declared: gedcom.Version551, Normalized: true}
	case "7":
		return Resolution{Version: gedcom.Version70, Declared: gedcom.Version70, Normalized: true}
	}
	rest, ok := strings.CutPrefix(token, "7.")
	if !ok {
		return Resolution{}
	}
	minor, patch, hasPatch := strings.Cut(rest, ".")
	if !isDigits(minor) || (hasPatch && !isDigits(patch)) {
		return Resolution{}
	}
	return Resolution{Version: gedcom.Version70, Declared: gedcom.Version("7." + trimZeros(minor))}
}

// trimZeros returns the digits s without leading zeros, keeping one digit.
func trimZeros(s string) string {
	t := strings.TrimLeft(s, "0")
	if t == "" {
		return "0"
	}
	return t
}

// isDigits reports whether s is non-empty and only ASCII digits.
func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
