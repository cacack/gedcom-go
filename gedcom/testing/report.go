package testing

import (
	"fmt"
	"strings"
)

// RoundTripReport contains the results of a round-trip comparison.
// It tracks whether documents are semantically equivalent and lists
// all differences found.
type RoundTripReport struct {
	// Equal is true if the documents are semantically equivalent.
	// When true, Differences will be empty.
	Equal bool

	// Differences lists all semantic differences found between
	// the original and round-tripped documents.
	Differences []Difference
}

// Difference represents a single semantic difference between
// the original and round-tripped documents.
type Difference struct {
	// Path describes the location of the difference using a
	// dot-notation path, e.g., "Record[@I1@].Tags[2].Value"
	// or "Header.Version".
	Path string

	// Before is the value in the original document.
	// Empty string for missing elements.
	Before string

	// After is the value after the round-trip.
	// Empty string for missing elements.
	After string
}

// String returns a human-readable summary of the round-trip report.
// This output is suitable for test failure messages.
func (r *RoundTripReport) String() string {
	var sb strings.Builder

	if r.Equal {
		sb.WriteString("Round-trip: PASSED (documents are semantically equivalent)\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("Round-trip: FAILED (%d differences found)\n", len(r.Differences)))
	sb.WriteString("\n")

	for i, diff := range r.Differences {
		sb.WriteString(fmt.Sprintf("  [%d] %s\n", i+1, diff.Path))
		sb.WriteString(fmt.Sprintf("      Before: %q\n", diff.Before))
		sb.WriteString(fmt.Sprintf("      After:  %q\n", diff.After))
	}

	return sb.String()
}

// AddDifference adds a difference to the report and sets Equal to false.
func (r *RoundTripReport) AddDifference(path, before, after string) {
	r.Equal = false
	r.Differences = append(r.Differences, Difference{
		Path:   path,
		Before: before,
		After:  after,
	})
}
