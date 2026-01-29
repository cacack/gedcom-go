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

	t.Run("report with dropped notes", func(t *testing.T) {
		report := &ConversionReport{
			SourceVersion: Version70,
			TargetVersion: Version55,
			Success:       true,
			Dropped: []ConversionNote{
				{
					Path:     "Individual @I1@ > EXID",
					Original: "ext-id-value",
					Reason:   "Tag not supported in GEDCOM 5.5",
				},
			},
		}

		str := report.String()

		if !strings.Contains(str, "Dropped: 1") {
			t.Error("Expected string to contain dropped count")
		}
		if !strings.Contains(str, "Individual @I1@ > EXID") {
			t.Error("Expected string to contain dropped path")
		}
	})

	t.Run("report with normalized notes", func(t *testing.T) {
		report := &ConversionReport{
			SourceVersion: Version55,
			TargetVersion: Version70,
			Success:       true,
			Normalized: []ConversionNote{
				{
					Path:     "Individual @i1@",
					Original: "@i1@",
					Result:   "@I1@",
					Reason:   "GEDCOM 7.0 requires uppercase XRefs",
				},
			},
		}

		str := report.String()

		if !strings.Contains(str, "Normalized: 1") {
			t.Error("Expected string to contain normalized count")
		}
		if !strings.Contains(str, "@i1@") {
			t.Error("Expected string to contain original value")
		}
		if !strings.Contains(str, "@I1@") {
			t.Error("Expected string to contain result value")
		}
	})

	t.Run("report with approximated notes", func(t *testing.T) {
		report := &ConversionReport{
			SourceVersion: Version55,
			TargetVersion: Version70,
			Success:       true,
			Approximated: []ConversionNote{
				{
					Path:     "MediaObject @M1@ > FILE[0] > FORM",
					Original: "JPG",
					Result:   "image/jpeg",
					Reason:   "GEDCOM 7.0 uses IANA media types",
				},
			},
		}

		str := report.String()

		if !strings.Contains(str, "Approximated: 1") {
			t.Error("Expected string to contain approximated count")
		}
		if !strings.Contains(str, "JPG") {
			t.Error("Expected string to contain original media type")
		}
		if !strings.Contains(str, "image/jpeg") {
			t.Error("Expected string to contain result media type")
		}
	})

	t.Run("report with preserved notes", func(t *testing.T) {
		report := &ConversionReport{
			SourceVersion: Version55,
			TargetVersion: Version70,
			Success:       true,
			Preserved: []ConversionNote{
				{
					Path:     "Individual @I1@ > _CUSTOM",
					Original: "_CUSTOM",
					Result:   "_CUSTOM",
					Reason:   "Vendor extension preserved through conversion",
				},
			},
		}

		str := report.String()

		if !strings.Contains(str, "Preserved: 1") {
			t.Error("Expected string to contain preserved count")
		}
		if !strings.Contains(str, "_CUSTOM") {
			t.Error("Expected string to contain preserved tag")
		}
	})
}

func TestConversionNote_Fields(t *testing.T) {
	tests := []struct {
		name     string
		note     ConversionNote
		wantPath string
		wantOrig string
		wantRes  string
		wantReas string
	}{
		{
			name: "all fields populated",
			note: ConversionNote{
				Path:     "Individual @I1@ > NAME",
				Original: "John /Doe/",
				Result:   "JOHN /DOE/",
				Reason:   "Normalized to uppercase",
			},
			wantPath: "Individual @I1@ > NAME",
			wantOrig: "John /Doe/",
			wantRes:  "JOHN /DOE/",
			wantReas: "Normalized to uppercase",
		},
		{
			name: "empty original for new elements",
			note: ConversionNote{
				Path:     "Header > GEDC > VERS",
				Original: "",
				Result:   "7.0",
				Reason:   "Version set during conversion",
			},
			wantPath: "Header > GEDC > VERS",
			wantOrig: "",
			wantRes:  "7.0",
			wantReas: "Version set during conversion",
		},
		{
			name: "empty result for dropped elements",
			note: ConversionNote{
				Path:     "Individual @I1@ > EXID",
				Original: "external-id-123",
				Result:   "",
				Reason:   "Tag not supported in GEDCOM 5.5",
			},
			wantPath: "Individual @I1@ > EXID",
			wantOrig: "external-id-123",
			wantRes:  "",
			wantReas: "Tag not supported in GEDCOM 5.5",
		},
		{
			name: "deeply nested path",
			note: ConversionNote{
				Path:     "Individual @I1@ > BIRT > PLAC > FORM",
				Original: "place format",
				Result:   "updated format",
				Reason:   "Format changed",
			},
			wantPath: "Individual @I1@ > BIRT > PLAC > FORM",
			wantOrig: "place format",
			wantRes:  "updated format",
			wantReas: "Format changed",
		},
		{
			name: "header element without XRef",
			note: ConversionNote{
				Path:     "Header > CHAR",
				Original: "ANSEL",
				Result:   "UTF-8",
				Reason:   "GEDCOM 7.0 requires UTF-8 encoding",
			},
			wantPath: "Header > CHAR",
			wantOrig: "ANSEL",
			wantRes:  "UTF-8",
			wantReas: "GEDCOM 7.0 requires UTF-8 encoding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.note.Path != tt.wantPath {
				t.Errorf("Path = %q, want %q", tt.note.Path, tt.wantPath)
			}
			if tt.note.Original != tt.wantOrig {
				t.Errorf("Original = %q, want %q", tt.note.Original, tt.wantOrig)
			}
			if tt.note.Result != tt.wantRes {
				t.Errorf("Result = %q, want %q", tt.note.Result, tt.wantRes)
			}
			if tt.note.Reason != tt.wantReas {
				t.Errorf("Reason = %q, want %q", tt.note.Reason, tt.wantReas)
			}
		})
	}
}

func TestConversionReport_AddNormalized(t *testing.T) {
	tests := []struct {
		name    string
		notes   []ConversionNote
		wantLen int
	}{
		{
			name:    "empty report",
			notes:   []ConversionNote{},
			wantLen: 0,
		},
		{
			name: "single note",
			notes: []ConversionNote{
				{Path: "Individual @I1@ > NAME", Original: "old", Result: "new", Reason: "test"},
			},
			wantLen: 1,
		},
		{
			name: "multiple notes",
			notes: []ConversionNote{
				{Path: "Individual @I1@", Original: "@i1@", Result: "@I1@", Reason: "uppercase"},
				{Path: "Family @F1@", Original: "@f1@", Result: "@F1@", Reason: "uppercase"},
				{Path: "Source @S1@", Original: "@s1@", Result: "@S1@", Reason: "uppercase"},
			},
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &ConversionReport{}
			for _, note := range tt.notes {
				report.AddNormalized(note)
			}
			if len(report.Normalized) != tt.wantLen {
				t.Errorf("got %d notes, want %d", len(report.Normalized), tt.wantLen)
			}
			// Verify all notes are stored correctly
			for i, note := range tt.notes {
				if i < len(report.Normalized) {
					if report.Normalized[i].Path != note.Path {
						t.Errorf("note[%d].Path = %q, want %q", i, report.Normalized[i].Path, note.Path)
					}
				}
			}
		})
	}
}

func TestConversionReport_AddDropped(t *testing.T) {
	tests := []struct {
		name    string
		notes   []ConversionNote
		wantLen int
	}{
		{
			name:    "empty report",
			notes:   []ConversionNote{},
			wantLen: 0,
		},
		{
			name: "single note",
			notes: []ConversionNote{
				{Path: "Individual @I1@ > EXID", Original: "ext-id", Result: "", Reason: "not supported"},
			},
			wantLen: 1,
		},
		{
			name: "multiple notes",
			notes: []ConversionNote{
				{Path: "Individual @I1@ > EXID", Original: "ext-id-1", Result: "", Reason: "not supported"},
				{Path: "Individual @I2@ > UID", Original: "uid-value", Result: "", Reason: "not supported"},
			},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &ConversionReport{}
			for _, note := range tt.notes {
				report.AddDropped(note)
			}
			if len(report.Dropped) != tt.wantLen {
				t.Errorf("got %d notes, want %d", len(report.Dropped), tt.wantLen)
			}
		})
	}
}

func TestConversionReport_AddApproximated(t *testing.T) {
	tests := []struct {
		name    string
		notes   []ConversionNote
		wantLen int
	}{
		{
			name:    "empty report",
			notes:   []ConversionNote{},
			wantLen: 0,
		},
		{
			name: "single note",
			notes: []ConversionNote{
				{Path: "MediaObject @M1@ > FILE[0] > FORM", Original: "JPG", Result: "image/jpeg", Reason: "IANA media type"},
			},
			wantLen: 1,
		},
		{
			name: "multiple notes",
			notes: []ConversionNote{
				{Path: "MediaObject @M1@ > FILE[0] > FORM", Original: "JPG", Result: "image/jpeg", Reason: "IANA"},
				{Path: "MediaObject @M2@ > FILE[0] > FORM", Original: "PNG", Result: "image/png", Reason: "IANA"},
				{Path: "MediaObject @M3@ > FILE[0] > FORM", Original: "PDF", Result: "application/pdf", Reason: "IANA"},
			},
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &ConversionReport{}
			for _, note := range tt.notes {
				report.AddApproximated(note)
			}
			if len(report.Approximated) != tt.wantLen {
				t.Errorf("got %d notes, want %d", len(report.Approximated), tt.wantLen)
			}
		})
	}
}

func TestConversionReport_AddPreserved(t *testing.T) {
	tests := []struct {
		name    string
		notes   []ConversionNote
		wantLen int
	}{
		{
			name:    "empty report",
			notes:   []ConversionNote{},
			wantLen: 0,
		},
		{
			name: "single note",
			notes: []ConversionNote{
				{Path: "Individual @I1@ > _CUSTOM", Original: "_CUSTOM", Result: "_CUSTOM", Reason: "preserved"},
			},
			wantLen: 1,
		},
		{
			name: "multiple vendor extensions",
			notes: []ConversionNote{
				{Path: "Individual @I1@ > _CUSTOM1", Original: "_CUSTOM1", Result: "_CUSTOM1", Reason: "preserved"},
				{Path: "Individual @I1@ > _CUSTOM2", Original: "_CUSTOM2", Result: "_CUSTOM2", Reason: "preserved"},
				{Path: "Family @F1@ > _FTDATA", Original: "_FTDATA", Result: "_FTDATA", Reason: "preserved"},
			},
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &ConversionReport{}
			for _, note := range tt.notes {
				report.AddPreserved(note)
			}
			if len(report.Preserved) != tt.wantLen {
				t.Errorf("got %d notes, want %d", len(report.Preserved), tt.wantLen)
			}
		})
	}
}

func TestConversionReport_AllNotes(t *testing.T) {
	tests := []struct {
		name         string
		dropped      []ConversionNote
		normalized   []ConversionNote
		approximated []ConversionNote
		preserved    []ConversionNote
		wantLen      int
		wantOrder    []string // expected path order
	}{
		{
			name:    "all empty returns nil",
			wantLen: 0,
		},
		{
			name:    "only dropped",
			dropped: []ConversionNote{{Path: "D1"}},
			wantLen: 1,
		},
		{
			name:       "only normalized",
			normalized: []ConversionNote{{Path: "N1"}},
			wantLen:    1,
		},
		{
			name:         "only approximated",
			approximated: []ConversionNote{{Path: "A1"}},
			wantLen:      1,
		},
		{
			name:      "only preserved",
			preserved: []ConversionNote{{Path: "P1"}},
			wantLen:   1,
		},
		{
			name:         "all categories populated",
			dropped:      []ConversionNote{{Path: "D1"}, {Path: "D2"}},
			normalized:   []ConversionNote{{Path: "N1"}},
			approximated: []ConversionNote{{Path: "A1"}, {Path: "A2"}},
			preserved:    []ConversionNote{{Path: "P1"}},
			wantLen:      6,
			wantOrder:    []string{"D1", "D2", "N1", "A1", "A2", "P1"},
		},
		{
			name:       "order preserved: dropped, normalized, approximated, preserved",
			dropped:    []ConversionNote{{Path: "Dropped"}},
			normalized: []ConversionNote{{Path: "Normalized"}},
			preserved:  []ConversionNote{{Path: "Preserved"}},
			wantLen:    3,
			wantOrder:  []string{"Dropped", "Normalized", "Preserved"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &ConversionReport{
				Dropped:      tt.dropped,
				Normalized:   tt.normalized,
				Approximated: tt.approximated,
				Preserved:    tt.preserved,
			}

			all := report.AllNotes()

			if tt.wantLen == 0 {
				if all != nil {
					t.Errorf("AllNotes() = %v, want nil for empty categories", all)
				}
				return
			}

			if len(all) != tt.wantLen {
				t.Errorf("AllNotes() returned %d notes, want %d", len(all), tt.wantLen)
			}

			// Verify order if specified
			if tt.wantOrder != nil {
				for i, want := range tt.wantOrder {
					if i < len(all) && all[i].Path != want {
						t.Errorf("AllNotes()[%d].Path = %q, want %q", i, all[i].Path, want)
					}
				}
			}
		})
	}
}

func TestFormatConversionNote(t *testing.T) {
	tests := []struct {
		name        string
		note        ConversionNote
		wantStrings []string
	}{
		{
			name: "all fields",
			note: ConversionNote{
				Path:     "Individual @I1@ > NAME",
				Original: "old value",
				Result:   "new value",
				Reason:   "test reason",
			},
			wantStrings: []string{
				"Individual @I1@ > NAME",
				"Original: old value",
				"Result: new value",
				"Reason: test reason",
			},
		},
		{
			name: "empty original",
			note: ConversionNote{
				Path:   "Header > VERS",
				Result: "7.0",
				Reason: "version set",
			},
			wantStrings: []string{
				"Header > VERS",
				"Result: 7.0",
			},
		},
		{
			name: "empty result",
			note: ConversionNote{
				Path:     "Individual @I1@ > EXID",
				Original: "ext-id",
				Reason:   "not supported",
			},
			wantStrings: []string{
				"Individual @I1@ > EXID",
				"Original: ext-id",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := formatConversionNote(tt.note)
			for _, want := range tt.wantStrings {
				if !strings.Contains(formatted, want) {
					t.Errorf("formatConversionNote() missing %q in output:\n%s", want, formatted)
				}
			}
		})
	}
}
