package converter

import "strings"

// PathSeparator is the delimiter used between path segments.
// The format " > " provides good readability in reports.
const PathSeparator = " > "

// recordTypeLabels maps GEDCOM record type tags to human-readable labels.
var recordTypeLabels = map[string]string{
	"INDI": "Individual",
	"FAM":  "Family",
	"SOUR": "Source",
	"REPO": "Repository",
	"NOTE": "Note",
	"OBJE": "MediaObject",
	"SUBM": "Submitter",
	"HEAD": "Header",
	"TRLR": "Trailer",
}

// RecordTypeLabel returns the human-readable label for a GEDCOM record type tag.
// Unknown tags are returned as-is.
//
// Example:
//
//	RecordTypeLabel("INDI") // returns "Individual"
//	RecordTypeLabel("FAM")  // returns "Family"
//	RecordTypeLabel("ZZZZ") // returns "ZZZZ" (unknown tag)
func RecordTypeLabel(tag string) string {
	if label, ok := recordTypeLabels[tag]; ok {
		return label
	}
	return tag
}

// BuildRecordPath creates a path string for a GEDCOM record.
// The path consists of a human-readable record type and optional XRef.
//
// Example:
//
//	BuildRecordPath("INDI", "@I1@") // returns "Individual @I1@"
//	BuildRecordPath("HEAD", "")     // returns "Header"
//	BuildRecordPath("FAM", "@F1@")  // returns "Family @F1@"
func BuildRecordPath(recordType, xref string) string {
	label := RecordTypeLabel(recordType)
	if xref == "" {
		return label
	}
	return label + " " + xref
}

// AppendToPath appends a segment to an existing path using the standard separator.
// If the base path is empty, returns only the segment.
//
// Example:
//
//	AppendToPath("Individual @I1@", "BIRT") // returns "Individual @I1@ > BIRT"
//	AppendToPath("", "HEAD")                // returns "HEAD"
func AppendToPath(base, segment string) string {
	if base == "" {
		return segment
	}
	return base + PathSeparator + segment
}

// BuildPath constructs a path from multiple segments.
// Empty segments are skipped.
//
// Example:
//
//	BuildPath("Individual @I1@", "BIRT", "DATE") // returns "Individual @I1@ > BIRT > DATE"
//	BuildPath("Header", "CHAR")                   // returns "Header > CHAR"
func BuildPath(segments ...string) string {
	var nonEmpty []string
	for _, s := range segments {
		if s != "" {
			nonEmpty = append(nonEmpty, s)
		}
	}
	return strings.Join(nonEmpty, PathSeparator)
}

// BuildNestedPath creates a path for a nested element within a record.
// This is a convenience function combining BuildRecordPath and BuildPath.
//
// Example:
//
//	BuildNestedPath("INDI", "@I1@", "BIRT", "DATE")
//	// returns "Individual @I1@ > BIRT > DATE"
//
//	BuildNestedPath("HEAD", "", "CHAR")
//	// returns "Header > CHAR"
func BuildNestedPath(recordType, xref string, tags ...string) string {
	basePath := BuildRecordPath(recordType, xref)
	if len(tags) == 0 {
		return basePath
	}
	return BuildPath(append([]string{basePath}, tags...)...)
}
