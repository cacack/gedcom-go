package version

import (
	"strings"

	"github.com/elliotchance/go-gedcom/gedcom"
	"github.com/elliotchance/go-gedcom/parser"
)

// DetectVersion detects the GEDCOM version from parsed lines.
// It first tries to find the version in the header (HEAD -> GEDC -> VERS).
// If not found, it falls back to tag-based heuristics.
// Returns Version55 as the default if detection fails.
func DetectVersion(lines []*parser.Line) (gedcom.Version, error) {
	// Try to detect from header first
	version := detectFromHeader(lines)
	if version != "" {
		return version, nil
	}

	// Fallback to tag-based heuristics
	version = detectFromTags(lines)
	return version, nil
}

// detectFromHeader looks for the version in the GEDCOM header.
// Header structure:
//   0 HEAD
//   1 GEDC
//   2 VERS 5.5 (or 5.5.1, or 7.0)
func detectFromHeader(lines []*parser.Line) gedcom.Version {
	inHead := false
	inGedc := false

	for _, line := range lines {
		// Look for HEAD tag at level 0
		if line.Level == 0 && line.Tag == "HEAD" {
			inHead = true
			continue
		}

		// Stop when we hit the next level 0 tag
		if line.Level == 0 && line.Tag != "HEAD" {
			inHead = false
		}

		// Look for GEDC tag at level 1 within HEAD
		if inHead && line.Level == 1 && line.Tag == "GEDC" {
			inGedc = true
			continue
		}

		// Look for VERS tag at level 2 within GEDC
		if inHead && inGedc && line.Level == 2 && line.Tag == "VERS" {
			version := strings.TrimSpace(line.Value)
			switch version {
			case "5.5":
				return gedcom.Version55
			case "5.5.1":
				return gedcom.Version551
			case "7.0", "7.0.0":
				return gedcom.Version70
			}
		}

		// Reset inGedc if we go back to level 1 with different tag
		if inHead && line.Level == 1 && line.Tag != "GEDC" {
			inGedc = false
		}
	}

	return ""
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

// IsValidVersion checks if a version string is a valid GEDCOM version.
func IsValidVersion(version gedcom.Version) bool {
	switch version {
	case gedcom.Version55, gedcom.Version551, gedcom.Version70:
		return true
	default:
		return false
	}
}
