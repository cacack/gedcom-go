package testing

import (
	"fmt"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// compareDocuments compares two documents and returns differences.
// It compares headers, record counts, and all record tags.
func compareDocuments(before, after *gedcom.Document, report *RoundTripReport) {
	// Compare headers
	compareHeaders(before.Header, after.Header, report)

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
func compareHeaders(before, after *gedcom.Header, report *RoundTripReport) {
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

	// Compare Encoding.
	//
	// The encoder always writes UTF-8 bytes and declares that charset, so a
	// document decoded from ANSEL or CP1252 is expected to come back as UTF-8.
	// That is asserted positively rather than exempted: an encoder that started
	// declaring something else would still fail here, where skipping the field
	// would hide it. compatibility.md records the normalization.
	switch {
	case before.Encoding == "":
		if after.Encoding != "" {
			report.AddDifference("Header.Encoding", "", string(after.Encoding))
		}
	case after.Encoding != gedcom.EncodingUTF8:
		report.AddDifference(
			"Header.Encoding",
			string(before.Encoding)+" (expected normalization to UTF-8)",
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

	// Header tags are compared unconditionally. They were gated behind an
	// off-by-default option while the encoder rebuilt HEAD from four scalars
	// and discarded the rest, which meant the round-trip guard could not see
	// the largest source of loss in the library (issues #429, #410).
	compareTags(normalizeCharTag(before.Tags), after.Tags, "Header.Tags", report)
}

// normalizeCharTag returns tags with the level-1 CHAR value set to the charset
// the encoder actually writes, so the expected normalization does not register
// as a difference while every other header tag is still compared verbatim.
//
// The input is not mutated: the comparison must not edit the document it was
// handed.
func normalizeCharTag(tags []*gedcom.Tag) []*gedcom.Tag {
	normalized := make([]*gedcom.Tag, len(tags))
	for i, tag := range tags {
		if tag != nil && tag.Level == 1 && tag.Tag == "CHAR" && tag.Value != string(gedcom.EncodingUTF8) {
			replaced := *tag
			replaced.Value = string(gedcom.EncodingUTF8)
			normalized[i] = &replaced
			continue
		}
		normalized[i] = tag
	}
	return normalized
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
