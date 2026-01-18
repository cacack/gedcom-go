package converter

import (
	"testing"

	"github.com/cacack/gedcom-go/gedcom"
)

func TestConvert(t *testing.T) {
	tests := []struct {
		name          string
		doc           *gedcom.Document
		targetVersion gedcom.Version
		wantErr       bool
		errContains   string
	}{
		{
			name:          "nil document returns error",
			doc:           nil,
			targetVersion: gedcom.Version70,
			wantErr:       true,
			errContains:   "document is nil",
		},
		{
			name: "invalid version returns error",
			doc: &gedcom.Document{
				Header: &gedcom.Header{Version: gedcom.Version55},
			},
			targetVersion: "invalid",
			wantErr:       true,
			errContains:   "invalid target version",
		},
		{
			name: "same version is no-op",
			doc: &gedcom.Document{
				Header: &gedcom.Header{Version: gedcom.Version55},
			},
			targetVersion: gedcom.Version55,
			wantErr:       false,
		},
		{
			name: "5.5 to 5.5.1 conversion",
			doc: &gedcom.Document{
				Header: &gedcom.Header{Version: gedcom.Version55},
			},
			targetVersion: gedcom.Version551,
			wantErr:       false,
		},
		{
			name: "5.5 to 7.0 conversion",
			doc: &gedcom.Document{
				Header: &gedcom.Header{Version: gedcom.Version55},
			},
			targetVersion: gedcom.Version70,
			wantErr:       false,
		},
		{
			name: "5.5.1 to 5.5 conversion",
			doc: &gedcom.Document{
				Header: &gedcom.Header{Version: gedcom.Version551},
			},
			targetVersion: gedcom.Version55,
			wantErr:       false,
		},
		{
			name: "5.5.1 to 7.0 conversion",
			doc: &gedcom.Document{
				Header: &gedcom.Header{Version: gedcom.Version551},
			},
			targetVersion: gedcom.Version70,
			wantErr:       false,
		},
		{
			name: "7.0 to 5.5 conversion",
			doc: &gedcom.Document{
				Header: &gedcom.Header{Version: gedcom.Version70},
			},
			targetVersion: gedcom.Version55,
			wantErr:       false,
		},
		{
			name: "7.0 to 5.5.1 conversion",
			doc: &gedcom.Document{
				Header: &gedcom.Header{Version: gedcom.Version70},
			},
			targetVersion: gedcom.Version551,
			wantErr:       false,
		},
		{
			name: "empty source version defaults to 5.5",
			doc: &gedcom.Document{
				Header: &gedcom.Header{Version: ""},
			},
			targetVersion: gedcom.Version70,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, report, err := Convert(tt.doc, tt.targetVersion)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Convert() expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Convert() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Convert() unexpected error: %v", err)
				return
			}

			if report == nil {
				t.Error("Convert() report should not be nil")
				return
			}

			// Same version should return original doc, not a copy
			if tt.doc != nil && tt.doc.Header != nil && tt.doc.Header.Version == tt.targetVersion {
				if result != tt.doc {
					t.Error("Convert() same version should return original document")
				}
				if !report.Success {
					t.Error("Convert() same version should report success")
				}
				return
			}

			// For actual conversions, verify result
			if result == nil {
				t.Error("Convert() result should not be nil")
				return
			}

			if result.Header.Version != tt.targetVersion {
				t.Errorf("Convert() result version = %v, want %v", result.Header.Version, tt.targetVersion)
			}

			if !report.Success {
				t.Error("Convert() should report success")
			}
		})
	}
}

func TestConvertWithOptions(t *testing.T) {
	t.Run("nil options uses defaults", func(t *testing.T) {
		doc := &gedcom.Document{
			Header: &gedcom.Header{Version: gedcom.Version55},
		}
		result, report, err := ConvertWithOptions(doc, gedcom.Version70, nil)
		if err != nil {
			t.Errorf("ConvertWithOptions() unexpected error: %v", err)
		}
		if result == nil {
			t.Error("ConvertWithOptions() result should not be nil")
		}
		if report == nil {
			t.Error("ConvertWithOptions() report should not be nil")
		}
	})

	t.Run("StrictDataLoss fails on data loss", func(t *testing.T) {
		doc := &gedcom.Document{
			Header: &gedcom.Header{Version: gedcom.Version70},
			Records: []*gedcom.Record{
				{
					XRef: "@I1@",
					Tags: []*gedcom.Tag{
						{Level: 0, Tag: "INDI"},
						{Level: 1, Tag: "EXID", Value: "external-id"},
					},
				},
			},
		}
		opts := &ConvertOptions{
			StrictDataLoss: true,
			Validate:       false,
		}
		_, report, err := ConvertWithOptions(doc, gedcom.Version55, opts)
		if err == nil {
			t.Error("ConvertWithOptions() expected error with StrictDataLoss")
		}
		if report == nil {
			t.Error("ConvertWithOptions() report should not be nil even on error")
		}
		if report != nil && !report.HasDataLoss() {
			t.Error("ConvertWithOptions() should report data loss")
		}
	})

	t.Run("Validate option runs validation", func(t *testing.T) {
		doc := &gedcom.Document{
			Header:  &gedcom.Header{Version: gedcom.Version55},
			Records: []*gedcom.Record{},
		}
		opts := &ConvertOptions{
			Validate: true,
		}
		_, report, err := ConvertWithOptions(doc, gedcom.Version70, opts)
		if err != nil {
			t.Errorf("ConvertWithOptions() unexpected error: %v", err)
		}
		// Validation may add issues but shouldn't fail conversion
		if report == nil {
			t.Error("ConvertWithOptions() report should not be nil")
		}
	})
}

func TestConversionPaths(t *testing.T) {
	// Test all 6 conversion paths work correctly
	versions := []gedcom.Version{gedcom.Version55, gedcom.Version551, gedcom.Version70}

	for _, sourceVer := range versions {
		for _, targetVer := range versions {
			if sourceVer == targetVer {
				continue
			}
			t.Run(string(sourceVer)+"_to_"+string(targetVer), func(t *testing.T) {
				doc := createTestDocument(sourceVer)
				result, report, err := Convert(doc, targetVer)
				if err != nil {
					t.Errorf("Convert() error = %v", err)
					return
				}
				if result == nil {
					t.Error("Convert() result should not be nil")
					return
				}
				if result.Header.Version != targetVer {
					t.Errorf("Convert() version = %v, want %v", result.Header.Version, targetVer)
				}
				if !report.Success {
					t.Error("Convert() should report success")
				}
			})
		}
	}
}

func TestConvertDoesNotMutateOriginal(t *testing.T) {
	original := &gedcom.Document{
		Header: &gedcom.Header{
			Version:  gedcom.Version55,
			Encoding: gedcom.EncodingANSEL,
		},
		Records: []*gedcom.Record{
			{
				XRef: "@i1@", // lowercase
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 0, Tag: "INDI", XRef: "@i1@"},
					{Level: 1, Tag: "NAME", Value: "John /Doe/"},
				},
			},
		},
		XRefMap: map[string]*gedcom.Record{
			"@i1@": nil, // will be set below
		},
	}
	original.XRefMap["@i1@"] = original.Records[0]

	originalXRef := original.Records[0].XRef
	originalEncoding := original.Header.Encoding

	result, _, err := Convert(original, gedcom.Version70)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	// Original should be unchanged
	if original.Records[0].XRef != originalXRef {
		t.Errorf("Original XRef mutated: got %v, want %v", original.Records[0].XRef, originalXRef)
	}
	if original.Header.Encoding != originalEncoding {
		t.Errorf("Original encoding mutated: got %v, want %v", original.Header.Encoding, originalEncoding)
	}

	// Result should have uppercase XRef (for 7.0)
	if result.Records[0].XRef != "@I1@" {
		t.Errorf("Result XRef = %v, want @I1@", result.Records[0].XRef)
	}
}

func TestRecord551Tags(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version551},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Tags: []*gedcom.Tag{
					{Level: 0, Tag: "INDI"},
					{Level: 1, Tag: "EMAIL", Value: "test@example.com"},
					{Level: 1, Tag: "FAX", Value: "123-456-7890"},
					{Level: 1, Tag: "WWW", Value: "http://example.com"},
				},
			},
		},
	}

	_, report, err := Convert(doc, gedcom.Version55)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	// Should record data loss for 5.5.1 tags
	if !report.HasDataLoss() {
		t.Error("Convert() should report data loss for 5.5.1 tags")
	}

	foundEmailLoss := false
	foundFaxLoss := false
	foundWWWLoss := false
	for _, loss := range report.DataLoss {
		if contains(loss.Feature, "EMAIL") {
			foundEmailLoss = true
		}
		if contains(loss.Feature, "FAX") {
			foundFaxLoss = true
		}
		if contains(loss.Feature, "WWW") {
			foundWWWLoss = true
		}
	}
	if !foundEmailLoss {
		t.Error("Should report EMAIL tag data loss")
	}
	if !foundFaxLoss {
		t.Error("Should report FAX tag data loss")
	}
	if !foundWWWLoss {
		t.Error("Should report WWW tag data loss")
	}
}

func TestRecord70DataLoss(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version70},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Tags: []*gedcom.Tag{
					{Level: 0, Tag: "INDI"},
					{Level: 1, Tag: "EXID", Value: "external-id"},
					{Level: 1, Tag: "UID", Value: "unique-id"},
					{Level: 1, Tag: "CREA"},
				},
			},
		},
	}

	tests := []struct {
		name          string
		targetVersion gedcom.Version
	}{
		{"7.0 to 5.5", gedcom.Version55},
		{"7.0 to 5.5.1", gedcom.Version551},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, report, err := Convert(doc, tt.targetVersion)
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}

			if !report.HasDataLoss() {
				t.Error("Convert() should report data loss for 7.0 tags")
			}

			foundEXID := false
			foundUID := false
			foundCREA := false
			for _, loss := range report.DataLoss {
				if contains(loss.Feature, "EXID") {
					foundEXID = true
				}
				if contains(loss.Feature, "UID") {
					foundUID = true
				}
				if contains(loss.Feature, "CREA") {
					foundCREA = true
				}
			}
			if !foundEXID {
				t.Error("Should report EXID tag data loss")
			}
			if !foundUID {
				t.Error("Should report UID tag data loss")
			}
			if !foundCREA {
				t.Error("Should report CREA tag data loss")
			}
		})
	}
}

func TestCountTagsInRecord(t *testing.T) {
	tags := []*gedcom.Tag{
		{Tag: "NAME"},
		{Tag: "EMAIL"},
		{Tag: "EMAIL"},
		{Tag: "FAX"},
	}
	targetTags := []string{"EMAIL", "FAX", "WWW"}
	found := make(map[string]int)

	countTagsInRecord(tags, targetTags, found)

	if found["EMAIL"] != 2 {
		t.Errorf("EMAIL count = %d, want 2", found["EMAIL"])
	}
	if found["FAX"] != 1 {
		t.Errorf("FAX count = %d, want 1", found["FAX"])
	}
	if found["WWW"] != 0 {
		t.Errorf("WWW count = %d, want 0", found["WWW"])
	}
}

func TestFindTagsInRecord(t *testing.T) {
	tags := []*gedcom.Tag{
		{Tag: "NAME"},
		{Tag: "EXID"},
		{Tag: "EXID"}, // duplicate
		{Tag: "UID"},
	}
	targetTags := []string{"EXID", "UID", "CREA"}

	result := findTagsInRecord(tags, targetTags)

	// Should return unique tags only
	if len(result) != 2 {
		t.Errorf("findTagsInRecord() returned %d tags, want 2", len(result))
	}

	found := make(map[string]bool)
	for _, tag := range result {
		found[tag] = true
	}
	if !found["EXID"] {
		t.Error("Should find EXID")
	}
	if !found["UID"] {
		t.Error("Should find UID")
	}
	if found["CREA"] {
		t.Error("Should not find CREA (not in input)")
	}
}

// Helper functions

func createTestDocument(version gedcom.Version) *gedcom.Document {
	return &gedcom.Document{
		Header: &gedcom.Header{
			Version:  version,
			Encoding: gedcom.EncodingUTF8,
		},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 0, Tag: "INDI", XRef: "@I1@"},
					{Level: 1, Tag: "NAME", Value: "John /Doe/"},
				},
			},
		},
		XRefMap: map[string]*gedcom.Record{},
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || substr == "" ||
		(s != "" && substr != "" && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
