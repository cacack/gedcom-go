package gedcom

import (
	"strings"
	"testing"
)

func TestConversionReport_AddTransformation(t *testing.T) {
	report := &ConversionReport{}

	report.AddTransformation(Transformation{
		Type:        "XREF_UPPERCASE",
		Description: "Converted XRefs to uppercase",
		Count:       5,
	})

	if len(report.Transformations) != 1 {
		t.Errorf("Expected 1 transformation, got %d", len(report.Transformations))
	}

	if report.Transformations[0].Type != "XREF_UPPERCASE" {
		t.Errorf("Expected type XREF_UPPERCASE, got %s", report.Transformations[0].Type)
	}

	if report.Transformations[0].Count != 5 {
		t.Errorf("Expected count 5, got %d", report.Transformations[0].Count)
	}
}

func TestConversionReport_AddDataLoss(t *testing.T) {
	report := &ConversionReport{}

	report.AddDataLoss(DataLossItem{
		Feature:         "SCHMA tag",
		Reason:          "Not supported in GEDCOM 5.5",
		AffectedRecords: []string{"@I1@", "@I2@"},
	})

	if len(report.DataLoss) != 1 {
		t.Errorf("Expected 1 data loss item, got %d", len(report.DataLoss))
	}

	if report.DataLoss[0].Feature != "SCHMA tag" {
		t.Errorf("Expected feature 'SCHMA tag', got %s", report.DataLoss[0].Feature)
	}

	if len(report.DataLoss[0].AffectedRecords) != 2 {
		t.Errorf("Expected 2 affected records, got %d", len(report.DataLoss[0].AffectedRecords))
	}
}

func TestConversionReport_HasDataLoss(t *testing.T) {
	t.Run("no data loss", func(t *testing.T) {
		report := &ConversionReport{}
		if report.HasDataLoss() {
			t.Error("Expected HasDataLoss to return false")
		}
	})

	t.Run("with data loss", func(t *testing.T) {
		report := &ConversionReport{
			DataLoss: []DataLossItem{
				{Feature: "test", Reason: "test"},
			},
		}
		if !report.HasDataLoss() {
			t.Error("Expected HasDataLoss to return true")
		}
	})
}

func TestConversionReport_String(t *testing.T) {
	t.Run("minimal report", func(t *testing.T) {
		report := &ConversionReport{
			SourceVersion: Version55,
			TargetVersion: Version70,
			Success:       true,
		}

		str := report.String()

		if !strings.Contains(str, "5.5") {
			t.Error("Expected string to contain source version")
		}
		if !strings.Contains(str, "7.0") {
			t.Error("Expected string to contain target version")
		}
		if !strings.Contains(str, "Success: true") {
			t.Error("Expected string to contain success status")
		}
	})

	t.Run("report with transformations", func(t *testing.T) {
		report := &ConversionReport{
			SourceVersion: Version55,
			TargetVersion: Version70,
			Success:       true,
			Transformations: []Transformation{
				{
					Type:        "XREF_UPPERCASE",
					Description: "Converted XRefs to uppercase",
					Count:       10,
				},
			},
		}

		str := report.String()

		if !strings.Contains(str, "Transformations: 1") {
			t.Error("Expected string to contain transformation count")
		}
		if !strings.Contains(str, "XREF_UPPERCASE") {
			t.Error("Expected string to contain transformation type")
		}
		if !strings.Contains(str, "10 instances") {
			t.Error("Expected string to contain instance count")
		}
	})

	t.Run("report with data loss", func(t *testing.T) {
		report := &ConversionReport{
			SourceVersion: Version70,
			TargetVersion: Version55,
			Success:       true,
			DataLoss: []DataLossItem{
				{
					Feature: "SCHMA tag",
					Reason:  "Not supported in GEDCOM 5.5",
				},
			},
		}

		str := report.String()

		if !strings.Contains(str, "Data Loss: 1") {
			t.Error("Expected string to contain data loss count")
		}
		if !strings.Contains(str, "SCHMA tag") {
			t.Error("Expected string to contain data loss feature")
		}
	})

	t.Run("report with validation issues", func(t *testing.T) {
		report := &ConversionReport{
			SourceVersion:    Version55,
			TargetVersion:    Version70,
			Success:          true,
			ValidationIssues: []string{"issue1", "issue2"},
		}

		str := report.String()

		if !strings.Contains(str, "Validation Issues: 2") {
			t.Error("Expected string to contain validation issues count")
		}
	})
}
