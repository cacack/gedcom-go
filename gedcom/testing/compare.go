package testing

import (
	"fmt"

	"github.com/cacack/gedcom-go/gedcom"
)

// compareDocuments compares two documents and returns differences.
// It compares headers, record counts, and all record tags.
func compareDocuments(before, after *gedcom.Document, report *RoundTripReport, cfg *roundTripConfig) {
	// Compare headers
	compareHeaders(before.Header, after.Header, report, cfg)

	// Compare record counts
	if len(before.Records) != len(after.Records) {
		report.AddDifference(
			"Records.Count",
			fmt.Sprintf("%d", len(before.Records)),
			fmt.Sprintf("%d", len(after.Records)),
		)
		// Still try to compare what we can
	}

	// Compare records by position
	minRecords := len(before.Records)
	if len(after.Records) < minRecords {
		minRecords = len(after.Records)
	}

	for i := 0; i < minRecords; i++ {
		compareRecords(before.Records[i], after.Records[i], i, report)
	}

	// Report missing records in after
	for i := minRecords; i < len(before.Records); i++ {
		xref := before.Records[i].XRef
		if xref == "" {
			xref = fmt.Sprintf("index:%d", i)
		}
		report.AddDifference(
			fmt.Sprintf("Record[%s]", xref),
			fmt.Sprintf("present (%s)", before.Records[i].Type),
			"missing",
		)
	}

	// Report extra records in after
	for i := minRecords; i < len(after.Records); i++ {
		xref := after.Records[i].XRef
		if xref == "" {
			xref = fmt.Sprintf("index:%d", i)
		}
		report.AddDifference(
			fmt.Sprintf("Record[%s]", xref),
			"missing",
			fmt.Sprintf("present (%s)", after.Records[i].Type),
		)
	}
}

// compareHeaders compares two header structs.
func compareHeaders(before, after *gedcom.Header, report *RoundTripReport, cfg *roundTripConfig) {
	// Handle nil headers
	if before == nil && after == nil {
		return
	}
	if before == nil {
		report.AddDifference("Header", "nil", "present")
		return
	}
	if after == nil {
		report.AddDifference("Header", "present", "nil")
		return
	}

	// Compare Version
	if before.Version != after.Version {
		report.AddDifference(
			"Header.Version",
			string(before.Version),
			string(after.Version),
		)
	}

	// Compare Encoding
	if before.Encoding != after.Encoding {
		report.AddDifference(
			"Header.Encoding",
			string(before.Encoding),
			string(after.Encoding),
		)
	}

	// Compare SourceSystem
	if before.SourceSystem != after.SourceSystem {
		report.AddDifference(
			"Header.SourceSystem",
			before.SourceSystem,
			after.SourceSystem,
		)
	}

	// Compare Language
	if before.Language != after.Language {
		report.AddDifference(
			"Header.Language",
			before.Language,
			after.Language,
		)
	}

	// Compare Header.Tags if enabled
	// By default, header tags are not compared because the encoder
	// reconstructs the header from Header fields.
	if cfg != nil && cfg.compareHeaderTags {
		compareTags(before.Tags, after.Tags, "Header.Tags", report)
	}
}

// compareRecords compares two records at the given index.
func compareRecords(before, after *gedcom.Record, index int, report *RoundTripReport) {
	// Build record path prefix
	xref := before.XRef
	if xref == "" {
		xref = fmt.Sprintf("index:%d", index)
	}
	pathPrefix := fmt.Sprintf("Record[%s]", xref)

	// Compare XRef
	if before.XRef != after.XRef {
		report.AddDifference(
			pathPrefix+".XRef",
			before.XRef,
			after.XRef,
		)
	}

	// Compare Type
	if before.Type != after.Type {
		report.AddDifference(
			pathPrefix+".Type",
			string(before.Type),
			string(after.Type),
		)
	}

	// Compare Value (record-level value, used for NOTE records)
	if before.Value != after.Value {
		report.AddDifference(
			pathPrefix+".Value",
			before.Value,
			after.Value,
		)
	}

	// Compare tags
	compareTags(before.Tags, after.Tags, pathPrefix+".Tags", report)
}

// compareTags compares two tag slices recursively.
func compareTags(before, after []*gedcom.Tag, pathPrefix string, report *RoundTripReport) {
	// Compare tag counts
	if len(before) != len(after) {
		report.AddDifference(
			pathPrefix+".Count",
			fmt.Sprintf("%d", len(before)),
			fmt.Sprintf("%d", len(after)),
		)
		// Still try to compare what we can
	}

	// Compare tags by position
	minTags := len(before)
	if len(after) < minTags {
		minTags = len(after)
	}

	for i := 0; i < minTags; i++ {
		compareTag(before[i], after[i], fmt.Sprintf("%s[%d]", pathPrefix, i), report)
	}

	// Report missing tags in after
	for i := minTags; i < len(before); i++ {
		report.AddDifference(
			fmt.Sprintf("%s[%d]", pathPrefix, i),
			fmt.Sprintf("present (%s)", before[i].Tag),
			"missing",
		)
	}

	// Report extra tags in after
	for i := minTags; i < len(after); i++ {
		report.AddDifference(
			fmt.Sprintf("%s[%d]", pathPrefix, i),
			"missing",
			fmt.Sprintf("present (%s)", after[i].Tag),
		)
	}
}

// compareTag compares two individual tags.
// LineNumber is intentionally not compared as it may change during round-trip.
func compareTag(before, after *gedcom.Tag, path string, report *RoundTripReport) {
	// Compare Level
	if before.Level != after.Level {
		report.AddDifference(
			path+".Level",
			fmt.Sprintf("%d", before.Level),
			fmt.Sprintf("%d", after.Level),
		)
	}

	// Compare Tag name
	if before.Tag != after.Tag {
		report.AddDifference(
			path+".Tag",
			before.Tag,
			after.Tag,
		)
	}

	// Compare Value
	if before.Value != after.Value {
		report.AddDifference(
			path+".Value",
			before.Value,
			after.Value,
		)
	}

	// Compare XRef
	if before.XRef != after.XRef {
		report.AddDifference(
			path+".XRef",
			before.XRef,
			after.XRef,
		)
	}

	// Note: LineNumber is intentionally NOT compared as it is expected
	// to change during round-trip due to header reconstruction.
}
