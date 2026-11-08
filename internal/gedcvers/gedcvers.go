// Package gedcvers maps a GEDC.VERS value to the GEDCOM version it names.
//
// The version package reads the version a file declares, and the encoder
// resolves the version it was asked to write; both must agree on what a value
// such as 7.0.14 means, so the rule lives here once (issues #587, #696).
package gedcvers

import (
	"strings"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Parse maps a GEDC.VERS value to a version, or "" when it is not recognized.
// GEDCOM 7 versions itself as 7.0.N and lets VERS carry the patch level, so 7.0
// followed by "." and ASCII digits is 7.0; anything else (7.0., 7.0.x, 7.1)
// stays unrecognized rather than guessed.
func Parse(value string) gedcom.Version {
	version := strings.TrimSpace(value)
	switch version {
	case "5.5":
		return gedcom.Version55
	case "5.5.1":
		return gedcom.Version551
	case "7.0":
		return gedcom.Version70
	}
	if patch, ok := strings.CutPrefix(version, "7.0."); ok && isDigits(patch) {
		return gedcom.Version70
	}
	return ""
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
