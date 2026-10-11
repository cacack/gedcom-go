package version

import (
	"strings"

	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/parser"
)

// DetectVersion detects the GEDCOM version from parsed lines.
// It first tries to find the version in the header (HEAD -> GEDC -> VERS).
// If not found, it falls back to tag-based heuristics.
// Returns Version55 as the default if detection fails.
// A GEDCOM 7 patch version (VERS 7.0.N) is reported as [gedcom.Version70];
// the exact VERS value stays in the decoded header's raw Tags.
func DetectVersion(lines []*parser.Line) gedcom.Version {
	// Try to detect from header first
	version := detectFromHeader(lines, false)
	if version != "" {
		return version
	}

	// Fallback to tag-based heuristics
	return detectFromTags(lines)
}

// DeclaredVersion returns the version the GEDCOM header declares
// (HEAD -> GEDC -> VERS), or "" when there is no such declaration or it names
// a version this package does not recognize. Unlike [DetectVersion] it never
// guesses, so a caller can apply a version-specific rule only to a document
// that asked for it. Only the real header counts: the GEDCOM grammar gives HEAD
// no identifier, so a `0 @X1@ HEAD` record is ignored, as the decoder does.
// A GEDCOM 7 patch version (VERS 7.0.N) is reported as [gedcom.Version70].
func DeclaredVersion(lines []*parser.Line) gedcom.Version {
	return detectFromHeader(lines, true)
}

// detectFromHeader looks for the version in the GEDCOM header.
// Header structure:
//
//	0 HEAD
//	1 GEDC
//	2 VERS 5.5 (or 5.5.1, or 7.0, or a 7.0.N patch version)
//
// When realHeaderOnly is true, a level-0 HEAD carrying an XRef is not a header.
func detectFromHeader(lines []*parser.Line, realHeaderOnly bool) gedcom.Version {
	inHead := false
	inGedc := false

	for _, line := range lines {
		if version := processHeaderLine(line, &inHead, &inGedc, realHeaderOnly); version != "" {
			return version
		}
	}

	return ""
}

func processHeaderLine(line *parser.Line, inHead, inGedc *bool, realHeaderOnly bool) gedcom.Version {
	// Handle level 0 tags
	if line.Level == 0 {
		return handleLevel0(line, inHead, realHeaderOnly)
	}

	// Handle level 1 tags within HEAD
	if *inHead && line.Level == 1 {
		return handleLevel1(line, inGedc)
	}

	// Handle level 2 VERS tag within GEDC
	if *inHead && *inGedc && line.Level == 2 && line.Tag == "VERS" {
		return parseVersionString(line.Value)
	}

	return ""
}

func handleLevel0(line *parser.Line, inHead *bool, realHeaderOnly bool) gedcom.Version {
	if line.Tag == "HEAD" && (!realHeaderOnly || line.XRef == "") {
		*inHead = true
	} else {
		*inHead = false
	}
	return ""
}

func handleLevel1(line *parser.Line, inGedc *bool) gedcom.Version {
	if line.Tag == "GEDC" {
		*inGedc = true
	} else {
		*inGedc = false
	}
	return ""
}

// parseVersionString maps a HEAD.GEDC.VERS value to a version, or "" when it
// is not recognized. GEDCOM 7 versions itself as 7.0.N and lets VERS carry the
// patch level, so 7.0 followed by "." and ASCII digits is 7.0; anything else
// (7.0., 7.0.x, 7.1) stays unrecognized rather than guessed.
func parseVersionString(value string) gedcom.Version {
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

// detectFromTags uses tag-based heuristics to guess the GEDCOM version.
// This is a fallback when the header doesn't contain version info.
func detectFromTags(lines []*parser.Line) gedcom.Version {
	// Count tags specific to different versions
	var has70Tags, has551Tags bool

	for _, line := range lines {
		tag := line.Tag

		// GEDCOM 7.0 specific tags
		switch tag {
		case "EXID", "PHRASE", "SCHMA", "SNOTE", "UID", "CREA", "MIME":
			has70Tags = true
		}

		// GEDCOM 5.5.1 specific tags
		switch tag {
		case "MAP", "LATI", "LONG", "EMAIL", "WWW", "FACT":
			has551Tags = true
		}
	}

	// Determine version based on tags found
	if has70Tags {
		return gedcom.Version70
	}
	if has551Tags {
		return gedcom.Version551
	}

	// Default to 5.5 (most common)
	return gedcom.Version55
}
