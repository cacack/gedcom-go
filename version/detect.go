// Package version provides GEDCOM version detection and validation.
//
// This package helps identify which GEDCOM specification version (5.5, 5.5.1, or 7.0)
// a file conforms to. It can detect the version from the header or use tag-based
// heuristics to make an educated guess.
//
// Example usage:
//
//	lines, _ := parser.Parse(reader)
//	version, err := version.DetectVersion(lines)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Detected GEDCOM version: %s\n", version)
package version

import (
	"strings"

	"github.com/cacack/gedcom-go/gedcom"
	"github.com/cacack/gedcom-go/parser"
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
//
//	0 HEAD
//	1 GEDC
//	2 VERS 5.5 (or 5.5.1, or 7.0)
func detectFromHeader(lines []*parser.Line) gedcom.Version {
	inHead := false
	inGedc := false

	for _, line := range lines {
		if version := processHeaderLine(line, &inHead, &inGedc); version != "" {
			return version
		}
	}

	return ""
}

func processHeaderLine(line *parser.Line, inHead, inGedc *bool) gedcom.Version {
	// Handle level 0 tags
	if line.Level == 0 {
		return handleLevel0(line, inHead)
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

func handleLevel0(line *parser.Line, inHead *bool) gedcom.Version {
	if line.Tag == "HEAD" {
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

func parseVersionString(value string) gedcom.Version {
	version := strings.TrimSpace(value)
	switch version {
	case "5.5":
		return gedcom.Version55
	case "5.5.1":
		return gedcom.Version551
	case "7.0", "7.0.0":
		return gedcom.Version70
	default:
		return ""
	}
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
