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

// Integration tests for ConversionNote population

func TestConversionNotes_CONCConsolidation(t *testing.T) {
	tests := []struct {
		name              string
		sourceVersion     gedcom.Version
		targetVersion     gedcom.Version
		tags              []*gedcom.Tag
		wantNormalizedLen int
		wantReason        string
	}{
		{
			name:          "CONC consolidation to 7.0 produces normalized note",
			sourceVersion: gedcom.Version55,
			targetVersion: gedcom.Version70,
			// NOTE at level 1 with CONC child at level 2
			tags: []*gedcom.Tag{
				{Level: 1, Tag: "NOTE", Value: "This is the first part"},
				{Level: 2, Tag: "CONC", Value: " and this is concatenated"},
			},
			wantNormalizedLen: 1,
			wantReason:        "GEDCOM 7.0 removes CONC tags",
		},
		{
			name:          "CONT conversion to 7.0 produces normalized note",
			sourceVersion: gedcom.Version55,
			targetVersion: gedcom.Version70,
			// NOTE at level 1 with CONT child at level 2
			tags: []*gedcom.Tag{
				{Level: 1, Tag: "NOTE", Value: "Line one"},
				{Level: 2, Tag: "CONT", Value: "Line two"},
			},
			wantNormalizedLen: 1,
			wantReason:        "GEDCOM 7.0 uses embedded newlines",
		},
		{
			name:          "mixed CONC and CONT to 7.0",
			sourceVersion: gedcom.Version55,
			targetVersion: gedcom.Version70,
			// NOTE with interleaved CONC and CONT
			tags: []*gedcom.Tag{
				{Level: 1, Tag: "NOTE", Value: "Start"},
				{Level: 2, Tag: "CONC", Value: " continued"},
				{Level: 2, Tag: "CONT", Value: "New line"},
			},
			wantNormalizedLen: 1,
			wantReason:        "GEDCOM 7.0 removes CONC tags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &gedcom.Document{
				Header: &gedcom.Header{Version: tt.sourceVersion},
				Records: []*gedcom.Record{
					{
						XRef: "@I1@",
						Type: gedcom.RecordTypeIndividual,
						Tags: tt.tags,
					},
				},
				XRefMap: map[string]*gedcom.Record{},
			}
			doc.XRefMap["@I1@"] = doc.Records[0]

			_, report, err := Convert(doc, tt.targetVersion)
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}

			if len(report.Normalized) < tt.wantNormalizedLen {
				t.Errorf("got %d normalized notes, want at least %d", len(report.Normalized), tt.wantNormalizedLen)
			}

			// Verify reason contains expected text
			foundReason := false
			for _, note := range report.Normalized {
				if contains(note.Reason, tt.wantReason) {
					foundReason = true
					break
				}
			}
			if !foundReason && tt.wantNormalizedLen > 0 {
				t.Errorf("no normalized note contains reason %q", tt.wantReason)
			}
		})
	}
}

func TestConversionNotes_XRefUppercase(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version55},
		Records: []*gedcom.Record{
			{
				XRef: "@i1@", // lowercase
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 0, Tag: "INDI", XRef: "@i1@"},
					{Level: 1, Tag: "NAME", Value: "John /Doe/"},
				},
			},
			{
				XRef: "@f1@", // lowercase
				Type: gedcom.RecordTypeFamily,
				Tags: []*gedcom.Tag{
					{Level: 0, Tag: "FAM", XRef: "@f1@"},
					{Level: 1, Tag: "HUSB", Value: "@i1@"},
				},
			},
		},
		XRefMap: map[string]*gedcom.Record{},
	}
	doc.XRefMap["@i1@"] = doc.Records[0]
	doc.XRefMap["@f1@"] = doc.Records[1]

	_, report, err := Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	// Should have normalized notes for XRef changes
	if len(report.Normalized) < 2 {
		t.Errorf("got %d normalized notes, want at least 2 for XRef uppercase", len(report.Normalized))
	}

	// Check for specific XRef normalized notes
	foundI1 := false
	foundF1 := false
	for _, note := range report.Normalized {
		if note.Original == "@i1@" && note.Result == "@I1@" {
			foundI1 = true
		}
		if note.Original == "@f1@" && note.Result == "@F1@" {
			foundF1 = true
		}
	}

	if !foundI1 {
		t.Error("missing normalized note for @i1@ -> @I1@")
	}
	if !foundF1 {
		t.Error("missing normalized note for @f1@ -> @F1@")
	}

	// Verify reason mentions GEDCOM 7.0
	for _, note := range report.Normalized {
		if note.Original == "@i1@" || note.Original == "@f1@" {
			if !contains(note.Reason, "GEDCOM 7.0") && !contains(note.Reason, "uppercase") {
				t.Errorf("XRef normalized note reason should mention GEDCOM 7.0 or uppercase: %q", note.Reason)
			}
		}
	}
}

func TestConversionNotes_MediaTypeMapping(t *testing.T) {
	tests := []struct {
		name           string
		sourceVersion  gedcom.Version
		targetVersion  gedcom.Version
		inputForm      string
		wantForm       string
		wantApproxNote bool
	}{
		{
			name:           "JPG to IANA media type",
			sourceVersion:  gedcom.Version55,
			targetVersion:  gedcom.Version70,
			inputForm:      "JPG",
			wantForm:       "image/jpeg",
			wantApproxNote: true,
		},
		{
			name:           "PNG to IANA media type",
			sourceVersion:  gedcom.Version551,
			targetVersion:  gedcom.Version70,
			inputForm:      "PNG",
			wantForm:       "image/png",
			wantApproxNote: true,
		},
		{
			name:           "IANA to legacy media type",
			sourceVersion:  gedcom.Version70,
			targetVersion:  gedcom.Version55,
			inputForm:      "image/jpeg",
			wantForm:       "JPG",
			wantApproxNote: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &gedcom.Document{
				Header: &gedcom.Header{Version: tt.sourceVersion},
				Records: []*gedcom.Record{
					{
						XRef: "@M1@",
						Type: gedcom.RecordTypeMedia,
						Tags: []*gedcom.Tag{
							{Level: 0, Tag: "OBJE", XRef: "@M1@"},
							{Level: 1, Tag: "FILE", Value: "photo.jpg"},
							{Level: 2, Tag: "FORM", Value: tt.inputForm},
						},
						Entity: &gedcom.MediaObject{
							XRef: "@M1@",
							Files: []*gedcom.MediaFile{
								{FileRef: "photo.jpg", Form: tt.inputForm},
							},
						},
					},
				},
				XRefMap: map[string]*gedcom.Record{},
			}
			doc.XRefMap["@M1@"] = doc.Records[0]

			_, report, err := Convert(doc, tt.targetVersion)
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}

			if tt.wantApproxNote {
				if len(report.Approximated) == 0 {
					t.Error("expected approximated notes for media type mapping")
				}

				// Find the specific note
				found := false
				for _, note := range report.Approximated {
					if note.Original == tt.inputForm && note.Result == tt.wantForm {
						found = true
						// Verify path includes OBJE and FILE[0]
						if !contains(note.Path, "MediaObject") && !contains(note.Path, "OBJE") {
							t.Errorf("approximated note path should mention MediaObject or OBJE: %q", note.Path)
						}
						// Verify reason mentions IANA or media types
						if !contains(note.Reason, "IANA") && !contains(note.Reason, "media type") {
							t.Errorf("approximated note reason should mention IANA or media type: %q", note.Reason)
						}
						break
					}
				}
				if !found {
					t.Errorf("missing approximated note for %q -> %q", tt.inputForm, tt.wantForm)
				}
			}
		})
	}
}

func TestConversionNotes_DroppedTags(t *testing.T) {
	tests := []struct {
		name          string
		sourceVersion gedcom.Version
		targetVersion gedcom.Version
		tagToAdd      string
		wantDropped   bool
	}{
		{
			name:          "EXID dropped in 7.0 to 5.5 conversion",
			sourceVersion: gedcom.Version70,
			targetVersion: gedcom.Version55,
			tagToAdd:      "EXID",
			wantDropped:   true,
		},
		{
			name:          "UID dropped in 7.0 to 5.5 conversion",
			sourceVersion: gedcom.Version70,
			targetVersion: gedcom.Version55,
			tagToAdd:      "UID",
			wantDropped:   true,
		},
		{
			name:          "CREA dropped in 7.0 to 5.5.1 conversion",
			sourceVersion: gedcom.Version70,
			targetVersion: gedcom.Version551,
			tagToAdd:      "CREA",
			wantDropped:   true,
		},
		{
			name:          "EMAIL dropped in 5.5.1 to 5.5 conversion",
			sourceVersion: gedcom.Version551,
			targetVersion: gedcom.Version55,
			tagToAdd:      "EMAIL",
			wantDropped:   true,
		},
		{
			name:          "WWW dropped in 5.5.1 to 5.5 conversion",
			sourceVersion: gedcom.Version551,
			targetVersion: gedcom.Version55,
			tagToAdd:      "WWW",
			wantDropped:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &gedcom.Document{
				Header: &gedcom.Header{Version: tt.sourceVersion},
				Records: []*gedcom.Record{
					{
						XRef: "@I1@",
						Type: gedcom.RecordTypeIndividual,
						Tags: []*gedcom.Tag{
							{Level: 0, Tag: "INDI", XRef: "@I1@"},
							{Level: 1, Tag: tt.tagToAdd, Value: "test-value"},
						},
					},
				},
				XRefMap: map[string]*gedcom.Record{},
			}
			doc.XRefMap["@I1@"] = doc.Records[0]

			_, report, err := Convert(doc, tt.targetVersion)
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}

			if tt.wantDropped {
				if len(report.Dropped) == 0 {
					t.Errorf("expected dropped notes for %s tag", tt.tagToAdd)
					return
				}

				// Find the specific dropped note
				found := false
				for _, note := range report.Dropped {
					if note.Original == tt.tagToAdd || contains(note.Path, tt.tagToAdd) {
						found = true
						// Verify reason mentions target version
						if !contains(note.Reason, tt.targetVersion.String()) && !contains(note.Reason, "not supported") {
							t.Errorf("dropped note reason should mention version: %q", note.Reason)
						}
						break
					}
				}
				if !found {
					t.Errorf("missing dropped note for %s tag", tt.tagToAdd)
				}
			}
		})
	}
}

func TestConversionNotes_PreservedUnknownTags(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version: gedcom.Version55,
			Tags: []*gedcom.Tag{
				{Level: 0, Tag: "HEAD"},
				{Level: 1, Tag: "_ROOTSMAGIC", Value: "custom header data"},
			},
		},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 0, Tag: "INDI", XRef: "@I1@"},
					{Level: 1, Tag: "_CUSTOM", Value: "custom value"},
					{Level: 1, Tag: "_FTDATA", Value: "family tree data"},
				},
			},
		},
		XRefMap: map[string]*gedcom.Record{},
	}
	doc.XRefMap["@I1@"] = doc.Records[0]

	opts := &ConvertOptions{
		PreserveUnknownTags: true,
	}
	_, report, err := ConvertWithOptions(doc, gedcom.Version70, opts)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	// Should have preserved notes for vendor extensions
	if len(report.Preserved) == 0 {
		t.Error("expected preserved notes for vendor extension tags")
		return
	}

	// Check for specific preserved tags
	foundCustom := false
	foundFTData := false
	foundRootsMagic := false
	for _, note := range report.Preserved {
		if note.Original == "_CUSTOM" {
			foundCustom = true
		}
		if note.Original == "_FTDATA" {
			foundFTData = true
		}
		if note.Original == "_ROOTSMAGIC" {
			foundRootsMagic = true
		}
	}

	if !foundCustom {
		t.Error("missing preserved note for _CUSTOM tag")
	}
	if !foundFTData {
		t.Error("missing preserved note for _FTDATA tag")
	}
	if !foundRootsMagic {
		t.Error("missing preserved note for _ROOTSMAGIC header tag")
	}

	// Verify preserved notes mention "vendor extension" or "preserved"
	for _, note := range report.Preserved {
		if !contains(note.Reason, "Vendor extension") && !contains(note.Reason, "preserved") {
			t.Errorf("preserved note reason should mention vendor extension: %q", note.Reason)
		}
	}
}

func TestConversionNotes_BackwardCompatibility(t *testing.T) {
	// Verify that Transformations[] is still populated alongside ConversionNotes
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version55},
		Records: []*gedcom.Record{
			{
				XRef: "@i1@", // lowercase for XRef transformation
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					// NOTE with CONC child for text transformation
					{Level: 1, Tag: "NOTE", Value: "Part 1"},
					{Level: 2, Tag: "CONC", Value: " Part 2"},
				},
			},
		},
		XRefMap: map[string]*gedcom.Record{},
	}
	doc.XRefMap["@i1@"] = doc.Records[0]

	_, report, err := Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	// Transformations should still be populated
	if len(report.Transformations) == 0 {
		t.Error("Transformations[] should be populated for backward compatibility")
	}

	// Both old and new reporting should work together
	foundXRefTransform := false
	foundCONCTransform := false
	for _, tr := range report.Transformations {
		if tr.Type == "XREF_UPPERCASE" {
			foundXRefTransform = true
		}
		if tr.Type == "CONC_REMOVED" {
			foundCONCTransform = true
		}
	}

	if !foundXRefTransform {
		t.Error("missing XREF_UPPERCASE transformation")
	}
	if !foundCONCTransform {
		t.Error("missing CONC_REMOVED transformation")
	}

	// ConversionNotes should also be populated
	if len(report.Normalized) == 0 {
		t.Error("Normalized notes should be populated")
	}
}

func TestConversionNotes_NewlinesToCONT(t *testing.T) {
	// Test downgrade from 7.0 where embedded newlines become CONT tags
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version70},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 0, Tag: "INDI", XRef: "@I1@"},
					{Level: 1, Tag: "NOTE", Value: "Line 1\nLine 2\nLine 3"},
				},
			},
		},
		XRefMap: map[string]*gedcom.Record{},
	}
	doc.XRefMap["@I1@"] = doc.Records[0]

	_, report, err := Convert(doc, gedcom.Version55)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	// Should have normalized notes for newline expansion
	foundExpansion := false
	for _, note := range report.Normalized {
		if contains(note.Reason, "CONT") || contains(note.Reason, "newline") {
			foundExpansion = true
			// Original should have newlines
			if !contains(note.Original, "\n") {
				t.Errorf("newline expansion note original should contain newline: %q", note.Original)
			}
			break
		}
	}

	if !foundExpansion {
		t.Error("expected normalized note for newline-to-CONT expansion")
	}
}
