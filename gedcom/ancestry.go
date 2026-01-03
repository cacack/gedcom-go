package gedcom

import "strings"

// AncestryAPID represents an Ancestry Permanent Identifier from the _APID tag.
// This is a vendor-specific extension used by Ancestry.com to link GEDCOM data
// to their online databases.
//
// The APID format is typically "1,DATABASE::RECORD" where:
//   - The "1," prefix is a version/type indicator (usually ignored)
//   - DATABASE is the Ancestry database ID
//   - RECORD is the record ID within that database
//
// Example: "1,7602::2771226" where 7602 is the database and 2771226 is the record.
type AncestryAPID struct {
	// Raw is the original unparsed APID value
	Raw string

	// Database is the Ancestry database ID
	Database string

	// Record is the record ID within the database
	Record string
}

// ParseAPID parses an Ancestry APID string into an AncestryAPID struct.
// Returns nil if the value cannot be parsed.
//
// Supported formats:
//   - "1,DATABASE::RECORD" (standard format with prefix)
//   - "DATABASE::RECORD" (without prefix)
//
// Examples:
//
//	ParseAPID("1,7602::2771226") -> &AncestryAPID{Raw: "1,7602::2771226", Database: "7602", Record: "2771226"}
//	ParseAPID("7602::2771226") -> &AncestryAPID{Raw: "7602::2771226", Database: "7602", Record: "2771226"}
//	ParseAPID("invalid") -> nil
func ParseAPID(value string) *AncestryAPID {
	if value == "" {
		return nil
	}

	// Store the raw value
	apid := &AncestryAPID{Raw: value}

	// Find the :: separator
	sepIdx := strings.Index(value, "::")
	if sepIdx == -1 {
		return nil
	}

	// Extract the record (everything after ::)
	apid.Record = value[sepIdx+2:]
	if apid.Record == "" {
		return nil
	}

	// Extract the database part (before ::)
	dbPart := value[:sepIdx]

	// Check for "1," prefix (or any number prefix)
	if commaIdx := strings.Index(dbPart, ","); commaIdx != -1 {
		apid.Database = dbPart[commaIdx+1:]
	} else {
		apid.Database = dbPart
	}

	if apid.Database == "" {
		return nil
	}

	return apid
}

// URL returns the Ancestry.com URL for this record.
// The URL format is: https://www.ancestry.com/discoveryui-content/view/{record}:{database}
func (a *AncestryAPID) URL() string {
	if a == nil || a.Database == "" || a.Record == "" {
		return ""
	}
	return "https://www.ancestry.com/discoveryui-content/view/" + a.Record + ":" + a.Database
}
