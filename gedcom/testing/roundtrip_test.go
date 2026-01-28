package testing

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/gedcom"
)

// validMinimalGEDCOM is a minimal valid GEDCOM 5.5.1 file.
const validMinimalGEDCOM = `0 HEAD
1 SOUR TestSystem
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
2 GIVN John
2 SURN Doe
1 SEX M
1 BIRT
2 DATE 1 JAN 1950
2 PLAC New York, USA
0 TRLR
`

// TestAssertRoundTrip_ValidGEDCOM tests that valid GEDCOM files pass round-trip.
func TestAssertRoundTrip_ValidGEDCOM(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "minimal GEDCOM",
			input: validMinimalGEDCOM,
		},
		{
			name: "multiple individuals",
			input: `0 HEAD
1 SOUR TestSystem
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 SEX M
0 @I2@ INDI
1 NAME Jane /Doe/
1 SEX F
0 TRLR
`,
		},
		{
			name: "family record",
			input: `0 HEAD
1 SOUR TestSystem
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 SEX M
1 FAMS @F1@
0 @I2@ INDI
1 NAME Jane /Smith/
1 SEX F
1 FAMS @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
0 TRLR
`,
		},
		{
			name: "vendor tags",
			input: `0 HEAD
1 SOUR TestSystem
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 _CUSTOM Value
2 _NESTED SubValue
0 TRLR
`,
		},
		{
			name: "note record with value",
			input: `0 HEAD
1 SOUR TestSystem
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @N1@ NOTE This is a note
0 TRLR
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AssertRoundTrip(t, []byte(tt.input))
		})
	}
}

// TestCheckRoundTrip_ReturnsReport tests that CheckRoundTrip returns proper reports.
func TestCheckRoundTrip_ReturnsReport(t *testing.T) {
	report, err := CheckRoundTrip(strings.NewReader(validMinimalGEDCOM))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !report.Equal {
		t.Errorf("expected Equal=true, got false with differences: %s", report.String())
	}

	if len(report.Differences) != 0 {
		t.Errorf("expected no differences, got %d", len(report.Differences))
	}
}

// TestCheckRoundTrip_InvalidGEDCOM tests error handling for invalid input.
func TestCheckRoundTrip_InvalidGEDCOM(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty input",
			input: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Empty input should decode to empty document, which should round-trip
			report, err := CheckRoundTrip(strings.NewReader(tt.input))
			// The decoder handles empty input gracefully
			if err != nil {
				t.Logf("got expected error: %v", err)
				return
			}
			t.Logf("report: %s", report.String())
		})
	}
}

// TestRoundTripReport_String tests the String() method formatting.
func TestRoundTripReport_String(t *testing.T) {
	t.Run("equal documents", func(t *testing.T) {
		report := &RoundTripReport{Equal: true}
		result := report.String()

		if !strings.Contains(result, "PASSED") {
			t.Errorf("expected 'PASSED' in output, got: %s", result)
		}
	})

	t.Run("documents with differences", func(t *testing.T) {
		report := &RoundTripReport{
			Equal: false,
			Differences: []Difference{
				{
					Path:   "Record[@I1@].Tags[0].Value",
					Before: "John",
					After:  "Jane",
				},
				{
					Path:   "Header.Version",
					Before: "5.5.1",
					After:  "5.5",
				},
			},
		}
		result := report.String()

		if !strings.Contains(result, "FAILED") {
			t.Errorf("expected 'FAILED' in output, got: %s", result)
		}
		if !strings.Contains(result, "2 differences") {
			t.Errorf("expected '2 differences' in output, got: %s", result)
		}
		if !strings.Contains(result, "Record[@I1@].Tags[0].Value") {
			t.Errorf("expected path in output, got: %s", result)
		}
		if !strings.Contains(result, "John") || !strings.Contains(result, "Jane") {
			t.Errorf("expected before/after values in output, got: %s", result)
		}
	})
}

// TestRoundTripReport_AddDifference tests the AddDifference helper.
func TestRoundTripReport_AddDifference(t *testing.T) {
	report := &RoundTripReport{Equal: true}

	report.AddDifference("test.path", "before", "after")

	if report.Equal {
		t.Error("expected Equal=false after adding difference")
	}
	if len(report.Differences) != 1 {
		t.Errorf("expected 1 difference, got %d", len(report.Differences))
	}

	diff := report.Differences[0]
	if diff.Path != "test.path" {
		t.Errorf("expected path 'test.path', got %q", diff.Path)
	}
	if diff.Before != "before" {
		t.Errorf("expected Before 'before', got %q", diff.Before)
	}
	if diff.After != "after" {
		t.Errorf("expected After 'after', got %q", diff.After)
	}
}

// TestCompareDocuments_DifferentRecordCounts tests detection of record count differences.
func TestCompareDocuments_DifferentRecordCounts(t *testing.T) {
	before := &gedcom.Document{
		Header: &gedcom.Header{Version: "5.5.1"},
		Records: []*gedcom.Record{
			{XRef: "@I1@", Type: gedcom.RecordTypeIndividual},
			{XRef: "@I2@", Type: gedcom.RecordTypeIndividual},
		},
	}
	after := &gedcom.Document{
		Header: &gedcom.Header{Version: "5.5.1"},
		Records: []*gedcom.Record{
			{XRef: "@I1@", Type: gedcom.RecordTypeIndividual},
		},
	}

	report := &RoundTripReport{Equal: true}
	compareDocuments(before, after, report)

	if report.Equal {
		t.Error("expected Equal=false for different record counts")
	}

	// Should have count difference and missing record
	foundCountDiff := false
	foundMissingRecord := false
	for _, diff := range report.Differences {
		if diff.Path == "Records.Count" {
			foundCountDiff = true
			if diff.Before != "2" || diff.After != "1" {
				t.Errorf("unexpected count diff: %s -> %s", diff.Before, diff.After)
			}
		}
		if strings.Contains(diff.Path, "@I2@") {
			foundMissingRecord = true
		}
	}

	if !foundCountDiff {
		t.Error("expected Records.Count difference")
	}
	if !foundMissingRecord {
		t.Error("expected missing @I2@ record difference")
	}
}

// TestCompareHeaders tests header comparison.
func TestCompareHeaders(t *testing.T) {
	tests := []struct {
		name          string
		before        *gedcom.Header
		after         *gedcom.Header
		expectDiffs   int
		expectedPaths []string
	}{
		{
			name:        "both nil",
			before:      nil,
			after:       nil,
			expectDiffs: 0,
		},
		{
			name:          "before nil",
			before:        nil,
			after:         &gedcom.Header{Version: "5.5.1"},
			expectDiffs:   1,
			expectedPaths: []string{"Header"},
		},
		{
			name:          "after nil",
			before:        &gedcom.Header{Version: "5.5.1"},
			after:         nil,
			expectDiffs:   1,
			expectedPaths: []string{"Header"},
		},
		{
			name:        "identical headers",
			before:      &gedcom.Header{Version: "5.5.1", Encoding: "UTF-8"},
			after:       &gedcom.Header{Version: "5.5.1", Encoding: "UTF-8"},
			expectDiffs: 0,
		},
		{
			name:          "different version",
			before:        &gedcom.Header{Version: "5.5.1"},
			after:         &gedcom.Header{Version: "5.5"},
			expectDiffs:   1,
			expectedPaths: []string{"Header.Version"},
		},
		{
			name:          "different encoding",
			before:        &gedcom.Header{Encoding: "UTF-8"},
			after:         &gedcom.Header{Encoding: "ANSEL"},
			expectDiffs:   1,
			expectedPaths: []string{"Header.Encoding"},
		},
		{
			name:          "different source system",
			before:        &gedcom.Header{SourceSystem: "System1"},
			after:         &gedcom.Header{SourceSystem: "System2"},
			expectDiffs:   1,
			expectedPaths: []string{"Header.SourceSystem"},
		},
		{
			name:          "different language",
			before:        &gedcom.Header{Language: "English"},
			after:         &gedcom.Header{Language: "French"},
			expectDiffs:   1,
			expectedPaths: []string{"Header.Language"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &RoundTripReport{Equal: true}
			compareHeaders(tt.before, tt.after, report)

			if len(report.Differences) != tt.expectDiffs {
				t.Errorf("expected %d differences, got %d: %v",
					tt.expectDiffs, len(report.Differences), report.Differences)
			}

			for _, expectedPath := range tt.expectedPaths {
				found := false
				for _, diff := range report.Differences {
					if diff.Path == expectedPath {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected difference at path %q not found", expectedPath)
				}
			}
		})
	}
}

// TestCompareTags tests tag comparison.
func TestCompareTags(t *testing.T) {
	tests := []struct {
		name          string
		before        []*gedcom.Tag
		after         []*gedcom.Tag
		expectDiffs   int
		expectedPaths []string
	}{
		{
			name:        "both empty",
			before:      nil,
			after:       nil,
			expectDiffs: 0,
		},
		{
			name: "identical tags",
			before: []*gedcom.Tag{
				{Level: 1, Tag: "NAME", Value: "John"},
			},
			after: []*gedcom.Tag{
				{Level: 1, Tag: "NAME", Value: "John"},
			},
			expectDiffs: 0,
		},
		{
			name: "different values",
			before: []*gedcom.Tag{
				{Level: 1, Tag: "NAME", Value: "John"},
			},
			after: []*gedcom.Tag{
				{Level: 1, Tag: "NAME", Value: "Jane"},
			},
			expectDiffs:   1,
			expectedPaths: []string{"prefix[0].Value"},
		},
		{
			name: "different levels",
			before: []*gedcom.Tag{
				{Level: 1, Tag: "NAME", Value: "John"},
			},
			after: []*gedcom.Tag{
				{Level: 2, Tag: "NAME", Value: "John"},
			},
			expectDiffs:   1,
			expectedPaths: []string{"prefix[0].Level"},
		},
		{
			name: "different tag names",
			before: []*gedcom.Tag{
				{Level: 1, Tag: "NAME", Value: "John"},
			},
			after: []*gedcom.Tag{
				{Level: 1, Tag: "GIVN", Value: "John"},
			},
			expectDiffs:   1,
			expectedPaths: []string{"prefix[0].Tag"},
		},
		{
			name: "different xrefs",
			before: []*gedcom.Tag{
				{Level: 1, Tag: "FAMS", XRef: "@F1@"},
			},
			after: []*gedcom.Tag{
				{Level: 1, Tag: "FAMS", XRef: "@F2@"},
			},
			expectDiffs:   1,
			expectedPaths: []string{"prefix[0].XRef"},
		},
		{
			name: "different counts - more before",
			before: []*gedcom.Tag{
				{Level: 1, Tag: "NAME", Value: "John"},
				{Level: 1, Tag: "SEX", Value: "M"},
			},
			after: []*gedcom.Tag{
				{Level: 1, Tag: "NAME", Value: "John"},
			},
			expectDiffs:   2, // count diff + missing tag
			expectedPaths: []string{"prefix.Count", "prefix[1]"},
		},
		{
			name: "different counts - more after",
			before: []*gedcom.Tag{
				{Level: 1, Tag: "NAME", Value: "John"},
			},
			after: []*gedcom.Tag{
				{Level: 1, Tag: "NAME", Value: "John"},
				{Level: 1, Tag: "SEX", Value: "M"},
			},
			expectDiffs:   2, // count diff + extra tag
			expectedPaths: []string{"prefix.Count", "prefix[1]"},
		},
		{
			name: "line number ignored",
			before: []*gedcom.Tag{
				{Level: 1, Tag: "NAME", Value: "John", LineNumber: 5},
			},
			after: []*gedcom.Tag{
				{Level: 1, Tag: "NAME", Value: "John", LineNumber: 10},
			},
			expectDiffs: 0, // LineNumber should be ignored
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &RoundTripReport{Equal: true}
			compareTags(tt.before, tt.after, "prefix", report)

			if len(report.Differences) != tt.expectDiffs {
				t.Errorf("expected %d differences, got %d: %v",
					tt.expectDiffs, len(report.Differences), report.Differences)
			}

			for _, expectedPath := range tt.expectedPaths {
				found := false
				for _, diff := range report.Differences {
					if diff.Path == expectedPath {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected difference at path %q not found in %v",
						expectedPath, report.Differences)
				}
			}
		})
	}
}

// TestCompareRecords tests record comparison.
func TestCompareRecords(t *testing.T) {
	tests := []struct {
		name          string
		before        *gedcom.Record
		after         *gedcom.Record
		expectDiffs   int
		expectedPaths []string
	}{
		{
			name: "identical records",
			before: &gedcom.Record{
				XRef:  "@I1@",
				Type:  gedcom.RecordTypeIndividual,
				Value: "",
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "NAME", Value: "John"},
				},
			},
			after: &gedcom.Record{
				XRef:  "@I1@",
				Type:  gedcom.RecordTypeIndividual,
				Value: "",
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "NAME", Value: "John"},
				},
			},
			expectDiffs: 0,
		},
		{
			name: "different xref",
			before: &gedcom.Record{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
			},
			after: &gedcom.Record{
				XRef: "@I2@",
				Type: gedcom.RecordTypeIndividual,
			},
			expectDiffs:   1,
			expectedPaths: []string{"Record[@I1@].XRef"},
		},
		{
			name: "different type",
			before: &gedcom.Record{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
			},
			after: &gedcom.Record{
				XRef: "@I1@",
				Type: gedcom.RecordTypeFamily,
			},
			expectDiffs:   1,
			expectedPaths: []string{"Record[@I1@].Type"},
		},
		{
			name: "different value",
			before: &gedcom.Record{
				XRef:  "@N1@",
				Type:  gedcom.RecordTypeNote,
				Value: "Note 1",
			},
			after: &gedcom.Record{
				XRef:  "@N1@",
				Type:  gedcom.RecordTypeNote,
				Value: "Note 2",
			},
			expectDiffs:   1,
			expectedPaths: []string{"Record[@N1@].Value"},
		},
		{
			name: "record without xref uses index",
			before: &gedcom.Record{
				Type: gedcom.RecordTypeIndividual,
			},
			after: &gedcom.Record{
				Type: gedcom.RecordTypeFamily,
			},
			expectDiffs:   1,
			expectedPaths: []string{"Record[index:0].Type"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &RoundTripReport{Equal: true}
			compareRecords(tt.before, tt.after, 0, report)

			if len(report.Differences) != tt.expectDiffs {
				t.Errorf("expected %d differences, got %d: %v",
					tt.expectDiffs, len(report.Differences), report.Differences)
			}

			for _, expectedPath := range tt.expectedPaths {
				found := false
				for _, diff := range report.Differences {
					if diff.Path == expectedPath {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected difference at path %q not found in %v",
						expectedPath, report.Differences)
				}
			}
		})
	}
}

// TestOptions tests functional options.
func TestOptions(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		cfg := defaultConfig()
		if cfg.compareHeaderTags {
			t.Error("expected compareHeaderTags=false by default")
		}
	})

	t.Run("WithHeaderTagComparison", func(t *testing.T) {
		cfg := applyOptions(WithHeaderTagComparison())
		if !cfg.compareHeaderTags {
			t.Error("expected compareHeaderTags=true after WithHeaderTagComparison")
		}
	})
}

// TestCheckRoundTrip_WithRealFiles tests with actual GEDCOM files.
func TestCheckRoundTrip_WithRealFiles(t *testing.T) {
	tests := []struct {
		name string
		file string
	}{
		{
			name: "GEDCOM 5.5.1 minimal",
			file: "../../testdata/gedcom-5.5.1/minimal.ged",
		},
		{
			name: "GEDCOM 5.5.1 comprehensive",
			file: "../../testdata/gedcom-5.5.1/comprehensive.ged",
		},
		{
			name: "GEDCOM 5.5 minimal",
			file: "../../testdata/gedcom-5.5/minimal.ged",
		},
		{
			name: "vendor tags (Ancestry)",
			file: "../../testdata/edge-cases/ancestry-extensions.ged",
		},
		{
			name: "vendor tags (FamilySearch)",
			file: "../../testdata/edge-cases/familysearch-extensions.ged",
		},
		{
			name: "vendor tags (Gramps)",
			file: "../../testdata/edge-cases/vendor-gramps.ged",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := os.ReadFile(tt.file)
			if err != nil {
				t.Skipf("test file not available: %v", err)
				return
			}

			report, err := CheckRoundTrip(bytes.NewReader(input))
			if err != nil {
				t.Fatalf("round-trip failed: %v", err)
			}

			if !report.Equal {
				t.Errorf("round-trip produced differences:\n%s", report.String())
			}
		})
	}
}

// TestCompareDocuments_ExtraRecordsInAfter tests detection of extra records.
func TestCompareDocuments_ExtraRecordsInAfter(t *testing.T) {
	before := &gedcom.Document{
		Header: &gedcom.Header{Version: "5.5.1"},
		Records: []*gedcom.Record{
			{XRef: "@I1@", Type: gedcom.RecordTypeIndividual},
		},
	}
	after := &gedcom.Document{
		Header: &gedcom.Header{Version: "5.5.1"},
		Records: []*gedcom.Record{
			{XRef: "@I1@", Type: gedcom.RecordTypeIndividual},
			{XRef: "@I2@", Type: gedcom.RecordTypeIndividual},
		},
	}

	report := &RoundTripReport{Equal: true}
	compareDocuments(before, after, report)

	if report.Equal {
		t.Error("expected Equal=false for different record counts")
	}

	// Should have count difference and extra record
	foundExtraRecord := false
	for _, diff := range report.Differences {
		if strings.Contains(diff.Path, "@I2@") && diff.Before == "missing" {
			foundExtraRecord = true
		}
	}

	if !foundExtraRecord {
		t.Error("expected extra @I2@ record difference")
	}
}
