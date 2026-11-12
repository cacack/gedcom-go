package version

import (
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/internal/gedcvers"
	"github.com/cacack/gedcom-go/v3/parser"
)

// DetectVersion detects the GEDCOM version from parsed lines.
// It first tries to find the version in the header (HEAD -> GEDC -> VERS).
// If not found, it falls back to tag-based heuristics.
// Only the real header counts: a GEDC/VERS beneath a `0 @X1@ HEAD` record is
// not a declaration, as in the decoder, so it never stands in for a silent
// header. Its lines still feed the tag heuristics, like any other record's.
// Returns Version55 as the default if detection fails.
// Every GEDCOM 7 minor and patch version (VERS 7.0.N, 7.1, 7.1.N) is reported
// as [gedcom.Version70], and a vendor spelling of a known version (5.51, 551,
// 5.5.5, or a version followed by other text) as that version; the exact VERS
// value stays in the decoded header's raw Tags.
func DetectVersion(lines []*parser.Line) gedcom.Version {
	// Try to detect from header first
	if version := detectFromHeader(lines).Version; version != "" {
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
// A GEDCOM 7 patch version (VERS 7.0.N) is reported as [gedcom.Version70]; a
// later GEDCOM 7 minor keeps its major.minor, so VERS 7.1 and 7.1.0 are "7.1",
// a version this package has no constant for, whose rules are 7.0's (see
// [DetectVersion]). A vendor spelling of a known version, such as 5.51, is
// reported as that version.
func DeclaredVersion(lines []*parser.Line) gedcom.Version {
	return detectFromHeader(lines).Declared
}

// detectFromHeader looks for the version in the GEDCOM header.
// Header structure:
//
//	0 HEAD
//	1 GEDC
//	2 VERS 5.5 (or 5.5.1, or 7.0, or a 7.N or 7.N.M version)
//
// A level-0 HEAD carrying an XRef is not a header: the GEDCOM grammar gives
// HEAD no identifier.
func detectFromHeader(lines []*parser.Line) gedcvers.Resolution {
	inHead := false
	inGedc := false

	for _, line := range lines {
		if r := processHeaderLine(line, &inHead, &inGedc); r.Version != "" {
			return r
		}
	}

	return gedcvers.Resolution{}
}

func processHeaderLine(line *parser.Line, inHead, inGedc *bool) gedcvers.Resolution {
	// Handle level 0 tags
	if line.Level == 0 {
		return handleLevel0(line, inHead)
	}

	// Handle level 1 tags within HEAD
	if *inHead && line.Level == 1 {
		return handleLevel1(line, inGedc)
	}

	// Handle level 2 VERS tag within GEDC; 7.1 is 7.0 (see gedcvers.Resolve).
	if *inHead && *inGedc && line.Level == 2 && line.Tag == "VERS" {
		return gedcvers.Resolve(line.Value)
	}

	return gedcvers.Resolution{}
}

func handleLevel0(line *parser.Line, inHead *bool) gedcvers.Resolution {
	*inHead = line.Tag == "HEAD" && line.XRef == ""
	return gedcvers.Resolution{}
}

func handleLevel1(line *parser.Line, inGedc *bool) gedcvers.Resolution {
	*inGedc = line.Tag == "GEDC"
	return gedcvers.Resolution{}
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
