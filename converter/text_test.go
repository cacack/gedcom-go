package converter

import (
	"testing"

	"github.com/cacack/gedcom-go/gedcom"
)

func TestTransformTextForVersion(t *testing.T) {
	tests := []struct {
		name          string
		inputTags     []*gedcom.Tag
		targetVersion gedcom.Version
		wantValue     string
		wantTagCount  int
	}{
		{
			name: "7.0 consolidates CONC",
			inputTags: []*gedcom.Tag{
				{Level: 1, Tag: "NOTE", Value: "First part"},
				{Level: 2, Tag: "CONC", Value: " continued"},
			},
			targetVersion: gedcom.Version70,
			wantValue:     "First part continued",
			wantTagCount:  1,
		},
		{
			name: "7.0 converts CONT to newlines",
			inputTags: []*gedcom.Tag{
				{Level: 1, Tag: "NOTE", Value: "Line 1"},
				{Level: 2, Tag: "CONT", Value: "Line 2"},
			},
			targetVersion: gedcom.Version70,
			wantValue:     "Line 1\nLine 2",
			wantTagCount:  1,
		},
		{
			name: "5.5 expands newlines to CONT",
			inputTags: []*gedcom.Tag{
				{Level: 1, Tag: "NOTE", Value: "Line 1\nLine 2\nLine 3"},
			},
			targetVersion: gedcom.Version55,
			wantValue:     "Line 1",
			wantTagCount:  3, // NOTE + 2 CONT
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &gedcom.Document{
				Header: &gedcom.Header{},
				Records: []*gedcom.Record{
					{Tags: deepCopyTags(tt.inputTags)},
				},
			}
			report := &gedcom.ConversionReport{}

			transformTextForVersion(doc, tt.targetVersion, report)

			if len(doc.Records[0].Tags) != tt.wantTagCount {
				t.Errorf("Tag count = %d, want %d", len(doc.Records[0].Tags), tt.wantTagCount)
			}
			if doc.Records[0].Tags[0].Value != tt.wantValue {
				t.Errorf("Value = %q, want %q", doc.Records[0].Tags[0].Value, tt.wantValue)
			}
		})
	}
}

func TestConsolidateCONCAndCONT(t *testing.T) {
	tests := []struct {
		name          string
		inputTags     []*gedcom.Tag
		wantValue     string
		wantCONCCount int
		wantCONTCount int
	}{
		{
			name: "CONC only",
			inputTags: []*gedcom.Tag{
				{Level: 1, Tag: "NOTE", Value: "Hello"},
				{Level: 2, Tag: "CONC", Value: " World"},
			},
			wantValue:     "Hello World",
			wantCONCCount: 1,
			wantCONTCount: 0,
		},
		{
			name: "CONT only",
			inputTags: []*gedcom.Tag{
				{Level: 1, Tag: "NOTE", Value: "Line 1"},
				{Level: 2, Tag: "CONT", Value: "Line 2"},
			},
			wantValue:     "Line 1\nLine 2",
			wantCONCCount: 0,
			wantCONTCount: 1,
		},
		{
			name: "interleaved CONC and CONT",
			inputTags: []*gedcom.Tag{
				{Level: 1, Tag: "NOTE", Value: "Start"},
				{Level: 2, Tag: "CONC", Value: " continued"},
				{Level: 2, Tag: "CONT", Value: "New line"},
				{Level: 2, Tag: "CONC", Value: " more"},
			},
			wantValue:     "Start continued\nNew line more",
			wantCONCCount: 2,
			wantCONTCount: 1,
		},
		{
			name: "multiple CONT tags",
			inputTags: []*gedcom.Tag{
				{Level: 1, Tag: "NOTE", Value: "Line 1"},
				{Level: 2, Tag: "CONT", Value: "Line 2"},
				{Level: 2, Tag: "CONT", Value: "Line 3"},
				{Level: 2, Tag: "CONT", Value: "Line 4"},
			},
			wantValue:     "Line 1\nLine 2\nLine 3\nLine 4",
			wantCONCCount: 0,
			wantCONTCount: 3,
		},
		{
			name: "empty CONT",
			inputTags: []*gedcom.Tag{
				{Level: 1, Tag: "NOTE", Value: "Before"},
				{Level: 2, Tag: "CONT", Value: ""},
				{Level: 2, Tag: "CONT", Value: "After"},
			},
			wantValue:     "Before\n\nAfter",
			wantCONCCount: 0,
			wantCONTCount: 2,
		},
		{
			name: "preserves non-CONC/CONT children",
			inputTags: []*gedcom.Tag{
				{Level: 1, Tag: "NOTE", Value: "Text"},
				{Level: 2, Tag: "CONC", Value: " more"},
				{Level: 2, Tag: "SOUR", Value: "@S1@"},
			},
			wantValue:     "Text more",
			wantCONCCount: 1,
			wantCONTCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &gedcom.Document{
				Header: &gedcom.Header{},
				Records: []*gedcom.Record{
					{Tags: deepCopyTags(tt.inputTags)},
				},
			}
			report := &gedcom.ConversionReport{}

			consolidateCONCAndCONT(doc, report)

			if doc.Records[0].Tags[0].Value != tt.wantValue {
				t.Errorf("Value = %q, want %q", doc.Records[0].Tags[0].Value, tt.wantValue)
			}

			// Check report counts
			var concCount, contCount int
			for _, tr := range report.Transformations {
				if tr.Type == "CONC_REMOVED" {
					concCount = tr.Count
				}
				if tr.Type == "CONT_CONVERTED" {
					contCount = tr.Count
				}
			}
			if concCount != tt.wantCONCCount {
				t.Errorf("CONC count = %d, want %d", concCount, tt.wantCONCCount)
			}
			if contCount != tt.wantCONTCount {
				t.Errorf("CONT count = %d, want %d", contCount, tt.wantCONTCount)
			}
		})
	}
}

func TestConsolidateCONCAndCONTInTags(t *testing.T) {
	tests := []struct {
		name          string
		inputTags     []*gedcom.Tag
		wantCount     int
		wantCONCCount int
		wantCONTCount int
	}{
		{
			name:          "empty tags",
			inputTags:     []*gedcom.Tag{},
			wantCount:     0,
			wantCONCCount: 0,
			wantCONTCount: 0,
		},
		{
			name:          "nil tags",
			inputTags:     nil,
			wantCount:     0,
			wantCONCCount: 0,
			wantCONTCount: 0,
		},
		{
			name: "no continuation tags",
			inputTags: []*gedcom.Tag{
				{Level: 1, Tag: "NOTE", Value: "Text"},
				{Level: 1, Tag: "NAME", Value: "John"},
			},
			wantCount:     2,
			wantCONCCount: 0,
			wantCONTCount: 0,
		},
		{
			name: "nested structure preserved",
			inputTags: []*gedcom.Tag{
				{Level: 0, Tag: "INDI"},
				{Level: 1, Tag: "NAME", Value: "John /Doe/"},
				{Level: 2, Tag: "GIVN", Value: "John"},
				{Level: 2, Tag: "SURN", Value: "Doe"},
			},
			wantCount:     4,
			wantCONCCount: 0,
			wantCONTCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, concCount, contCount := consolidateCONCAndCONTInTags(tt.inputTags)

			if len(result) != tt.wantCount {
				t.Errorf("Result count = %d, want %d", len(result), tt.wantCount)
			}
			if concCount != tt.wantCONCCount {
				t.Errorf("CONC count = %d, want %d", concCount, tt.wantCONCCount)
			}
			if contCount != tt.wantCONTCount {
				t.Errorf("CONT count = %d, want %d", contCount, tt.wantCONTCount)
			}
		})
	}
}

func TestConsolidateCONC(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Tags: []*gedcom.Tag{
				{Level: 1, Tag: "NOTE", Value: "Header"},
				{Level: 2, Tag: "CONC", Value: " note"},
			},
		},
		Records: []*gedcom.Record{
			{
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "TEXT", Value: "Record"},
					{Level: 2, Tag: "CONC", Value: " text"},
				},
			},
		},
	}
	report := &gedcom.ConversionReport{}

	consolidateCONC(doc, report)

	if doc.Header.Tags[0].Value != "Header note" {
		t.Errorf("Header value = %q, want %q", doc.Header.Tags[0].Value, "Header note")
	}
	if doc.Records[0].Tags[0].Value != "Record text" {
		t.Errorf("Record value = %q, want %q", doc.Records[0].Tags[0].Value, "Record text")
	}

	// Check report
	found := false
	for _, tr := range report.Transformations {
		if tr.Type == "CONC_REMOVED" {
			found = true
			if tr.Count != 2 {
				t.Errorf("CONC count = %d, want 2", tr.Count)
			}
		}
	}
	if !found {
		t.Error("Should have CONC_REMOVED transformation")
	}
}

func TestConsolidateCONCOnlyInTags(t *testing.T) {
	tests := []struct {
		name      string
		inputTags []*gedcom.Tag
		wantValue string
		wantCount int
	}{
		{
			name:      "empty",
			inputTags: []*gedcom.Tag{},
			wantValue: "",
			wantCount: 0,
		},
		{
			name: "CONC consolidated, CONT preserved",
			inputTags: []*gedcom.Tag{
				{Level: 1, Tag: "NOTE", Value: "Start"},
				{Level: 2, Tag: "CONC", Value: " middle"},
				{Level: 2, Tag: "CONT", Value: "newline"},
			},
			wantValue: "Start middle",
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, count := consolidateCONCOnlyInTags(tt.inputTags)

			if count != tt.wantCount {
				t.Errorf("Count = %d, want %d", count, tt.wantCount)
			}
			if tt.wantValue != "" && len(result) > 0 && result[0].Value != tt.wantValue {
				t.Errorf("Value = %q, want %q", result[0].Value, tt.wantValue)
			}

			// CONT should still be present if it was in input
			if tt.name == "CONC consolidated, CONT preserved" {
				hasCONT := false
				for _, tag := range result {
					if tag.Tag == "CONT" {
						hasCONT = true
						break
					}
				}
				if !hasCONT {
					t.Error("CONT tag should be preserved")
				}
			}
		})
	}
}

func TestConvertCONTToNewlines(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Tags: []*gedcom.Tag{
				{Level: 1, Tag: "NOTE", Value: "Line 1"},
				{Level: 2, Tag: "CONT", Value: "Line 2"},
			},
		},
		Records: []*gedcom.Record{
			{
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "TEXT", Value: "A"},
					{Level: 2, Tag: "CONT", Value: "B"},
					{Level: 2, Tag: "CONT", Value: "C"},
				},
			},
		},
	}
	report := &gedcom.ConversionReport{}

	convertCONTToNewlines(doc, report)

	if doc.Header.Tags[0].Value != "Line 1\nLine 2" {
		t.Errorf("Header value = %q, want %q", doc.Header.Tags[0].Value, "Line 1\nLine 2")
	}
	if doc.Records[0].Tags[0].Value != "A\nB\nC" {
		t.Errorf("Record value = %q, want %q", doc.Records[0].Tags[0].Value, "A\nB\nC")
	}

	// Check report
	found := false
	for _, tr := range report.Transformations {
		if tr.Type == "CONT_CONVERTED" {
			found = true
			if tr.Count != 3 {
				t.Errorf("CONT count = %d, want 3", tr.Count)
			}
		}
	}
	if !found {
		t.Error("Should have CONT_CONVERTED transformation")
	}
}

func TestConvertCONTOnlyInTags(t *testing.T) {
	tests := []struct {
		name      string
		inputTags []*gedcom.Tag
		wantValue string
		wantCount int
	}{
		{
			name:      "empty",
			inputTags: []*gedcom.Tag{},
			wantValue: "",
			wantCount: 0,
		},
		{
			name: "converts CONT to newlines",
			inputTags: []*gedcom.Tag{
				{Level: 1, Tag: "NOTE", Value: "First"},
				{Level: 2, Tag: "CONT", Value: "Second"},
			},
			wantValue: "First\nSecond",
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, count := convertCONTOnlyInTags(tt.inputTags)

			if count != tt.wantCount {
				t.Errorf("Count = %d, want %d", count, tt.wantCount)
			}
			if tt.wantValue != "" && len(result) > 0 && result[0].Value != tt.wantValue {
				t.Errorf("Value = %q, want %q", result[0].Value, tt.wantValue)
			}
		})
	}
}

func TestExpandNewlinesToCONT(t *testing.T) {
	tests := []struct {
		name         string
		inputValue   string
		wantTagCount int
		wantCONTTags int
	}{
		{
			name:         "no newlines",
			inputValue:   "Single line",
			wantTagCount: 1,
			wantCONTTags: 0,
		},
		{
			name:         "one newline",
			inputValue:   "Line 1\nLine 2",
			wantTagCount: 2,
			wantCONTTags: 1,
		},
		{
			name:         "multiple newlines",
			inputValue:   "Line 1\nLine 2\nLine 3",
			wantTagCount: 3,
			wantCONTTags: 2,
		},
		{
			name:         "empty lines",
			inputValue:   "Before\n\nAfter",
			wantTagCount: 3,
			wantCONTTags: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &gedcom.Document{
				Header: &gedcom.Header{},
				Records: []*gedcom.Record{
					{
						Tags: []*gedcom.Tag{
							{Level: 1, Tag: "NOTE", Value: tt.inputValue},
						},
					},
				},
			}
			report := &gedcom.ConversionReport{}

			expandNewlinesToCONT(doc, report)

			if len(doc.Records[0].Tags) != tt.wantTagCount {
				t.Errorf("Tag count = %d, want %d", len(doc.Records[0].Tags), tt.wantTagCount)
			}

			contCount := 0
			for _, tag := range doc.Records[0].Tags {
				if tag.Tag == "CONT" {
					contCount++
				}
			}
			if contCount != tt.wantCONTTags {
				t.Errorf("CONT tags = %d, want %d", contCount, tt.wantCONTTags)
			}
		})
	}
}

func TestExpandNewlinesInTags(t *testing.T) {
	tests := []struct {
		name      string
		inputTags []*gedcom.Tag
		wantCount int
		wantCONT  int
	}{
		{
			name:      "empty",
			inputTags: []*gedcom.Tag{},
			wantCount: 0,
			wantCONT:  0,
		},
		{
			name: "no newlines",
			inputTags: []*gedcom.Tag{
				{Level: 1, Tag: "NOTE", Value: "No newlines"},
			},
			wantCount: 1,
			wantCONT:  0,
		},
		{
			name: "creates CONT at correct level",
			inputTags: []*gedcom.Tag{
				{Level: 2, Tag: "NOTE", Value: "First\nSecond"},
			},
			wantCount: 2,
			wantCONT:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, count := expandNewlinesInTags(tt.inputTags)

			if len(result) != tt.wantCount {
				t.Errorf("Result count = %d, want %d", len(result), tt.wantCount)
			}
			if count != tt.wantCONT {
				t.Errorf("CONT count = %d, want %d", count, tt.wantCONT)
			}

			// Check CONT tags have correct level
			if tt.wantCONT > 0 && len(result) > 1 {
				parentLevel := tt.inputTags[0].Level
				for _, tag := range result[1:] {
					if tag.Tag == "CONT" && tag.Level != parentLevel+1 {
						t.Errorf("CONT level = %d, want %d", tag.Level, parentLevel+1)
					}
				}
			}
		})
	}
}

func TestTextTransformationHeaderTags(t *testing.T) {
	t.Run("header tags are processed", func(t *testing.T) {
		doc := &gedcom.Document{
			Header: &gedcom.Header{
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "NOTE", Value: "Header note"},
					{Level: 2, Tag: "CONC", Value: " continued"},
				},
			},
			Records: []*gedcom.Record{},
		}
		report := &gedcom.ConversionReport{}

		consolidateCONCAndCONT(doc, report)

		if doc.Header.Tags[0].Value != "Header note continued" {
			t.Errorf("Value = %q, want %q", doc.Header.Tags[0].Value, "Header note continued")
		}
	})

	t.Run("nil header is handled", func(t *testing.T) {
		doc := &gedcom.Document{
			Header:  nil,
			Records: []*gedcom.Record{},
		}
		report := &gedcom.ConversionReport{}

		// Should not panic
		consolidateCONCAndCONT(doc, report)
	})
}

func TestTextTransformationReportCounts(t *testing.T) {
	t.Run("no transformations when no continuation tags", func(t *testing.T) {
		doc := &gedcom.Document{
			Header: &gedcom.Header{},
			Records: []*gedcom.Record{
				{
					Tags: []*gedcom.Tag{
						{Level: 1, Tag: "NOTE", Value: "Simple note"},
					},
				},
			},
		}
		report := &gedcom.ConversionReport{}

		consolidateCONCAndCONT(doc, report)

		if len(report.Transformations) > 0 {
			t.Errorf("Expected no transformations, got %d", len(report.Transformations))
		}
	})
}
