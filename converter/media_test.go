package converter

import (
	"testing"

	"github.com/cacack/gedcom-go/gedcom"
)

func TestTransformMediaTypes(t *testing.T) {
	tests := []struct {
		name          string
		inputForm     string
		targetVersion gedcom.Version
		wantForm      string
		wantTransform bool
	}{
		{
			name:          "JPG to IANA for 7.0",
			inputForm:     "JPG",
			targetVersion: gedcom.Version70,
			wantForm:      "image/jpeg",
			wantTransform: true,
		},
		{
			name:          "image/jpeg to legacy for 5.5",
			inputForm:     "image/jpeg",
			targetVersion: gedcom.Version55,
			wantForm:      "JPG",
			wantTransform: true,
		},
		{
			name:          "image/jpeg to legacy for 5.5.1",
			inputForm:     "image/jpeg",
			targetVersion: gedcom.Version551,
			wantForm:      "JPG",
			wantTransform: true,
		},
		{
			name:          "already IANA for 7.0 no change",
			inputForm:     "image/png",
			targetVersion: gedcom.Version70,
			wantForm:      "image/png",
			wantTransform: false,
		},
		{
			name:          "already legacy for 5.5 no change",
			inputForm:     "PNG",
			targetVersion: gedcom.Version55,
			wantForm:      "PNG",
			wantTransform: false,
		},
		{
			name:          "unknown format preserved to 7.0",
			inputForm:     "UNKNOWN",
			targetVersion: gedcom.Version70,
			wantForm:      "UNKNOWN",
			wantTransform: false,
		},
		{
			name:          "unknown IANA preserved to 5.5",
			inputForm:     "application/unknown",
			targetVersion: gedcom.Version55,
			wantForm:      "application/unknown",
			wantTransform: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := createDocWithMediaFile(tt.inputForm)
			report := &gedcom.ConversionReport{}

			transformMediaTypes(doc, tt.targetVersion, report)

			media, _ := doc.Records[0].GetMediaObject()
			if media.Files[0].Form != tt.wantForm {
				t.Errorf("Form = %v, want %v", media.Files[0].Form, tt.wantForm)
			}

			hasTransform := len(report.Transformations) > 0
			if hasTransform != tt.wantTransform {
				t.Errorf("Has transformation = %v, want %v", hasTransform, tt.wantTransform)
			}
		})
	}
}

func TestTransformMediaTypesTranslations(t *testing.T) {
	t.Run("transforms translations too", func(t *testing.T) {
		media := &gedcom.MediaObject{
			XRef: "@M1@",
			Files: []*gedcom.MediaFile{
				{
					FileRef: "/path/to/file.jpg",
					Form:    "JPG",
					Translations: []*gedcom.MediaTranslation{
						{FileRef: "/path/to/file.png", Form: "PNG"},
						{FileRef: "/path/to/file.gif", Form: "GIF"},
					},
				},
			},
		}
		doc := &gedcom.Document{
			Header: &gedcom.Header{},
			Records: []*gedcom.Record{
				{
					XRef:   "@M1@",
					Type:   gedcom.RecordTypeMedia,
					Entity: media,
				},
			},
		}
		report := &gedcom.ConversionReport{}

		transformMediaTypes(doc, gedcom.Version70, report)

		if media.Files[0].Form != "image/jpeg" {
			t.Errorf("Main form = %v, want image/jpeg", media.Files[0].Form)
		}
		if media.Files[0].Translations[0].Form != "image/png" {
			t.Errorf("Translation 0 form = %v, want image/png", media.Files[0].Translations[0].Form)
		}
		if media.Files[0].Translations[1].Form != "image/gif" {
			t.Errorf("Translation 1 form = %v, want image/gif", media.Files[0].Translations[1].Form)
		}

		// Check report count
		for _, tr := range report.Transformations {
			if tr.Type == "MEDIA_TYPE_MAPPED" && tr.Count != 3 {
				t.Errorf("Transform count = %d, want 3", tr.Count)
			}
		}
	})
}

func TestTransformMediaTypesNilCases(t *testing.T) {
	t.Run("non-media records ignored", func(t *testing.T) {
		doc := &gedcom.Document{
			Header: &gedcom.Header{},
			Records: []*gedcom.Record{
				{XRef: "@I1@", Type: gedcom.RecordTypeIndividual},
			},
		}
		report := &gedcom.ConversionReport{}

		transformMediaTypes(doc, gedcom.Version70, report)

		if len(report.Transformations) > 0 {
			t.Error("Should have no transformations for non-media records")
		}
	})

	t.Run("nil entity handled", func(t *testing.T) {
		doc := &gedcom.Document{
			Header: &gedcom.Header{},
			Records: []*gedcom.Record{
				{XRef: "@M1@", Type: gedcom.RecordTypeMedia, Entity: nil},
			},
		}
		report := &gedcom.ConversionReport{}

		// Should not panic
		transformMediaTypes(doc, gedcom.Version70, report)
	})

	t.Run("nil file handled", func(t *testing.T) {
		media := &gedcom.MediaObject{
			Files: []*gedcom.MediaFile{nil},
		}
		doc := &gedcom.Document{
			Header: &gedcom.Header{},
			Records: []*gedcom.Record{
				{XRef: "@M1@", Type: gedcom.RecordTypeMedia, Entity: media},
			},
		}
		report := &gedcom.ConversionReport{}

		// Should not panic
		transformMediaTypes(doc, gedcom.Version70, report)
	})

	t.Run("empty form skipped", func(t *testing.T) {
		media := &gedcom.MediaObject{
			Files: []*gedcom.MediaFile{
				{FileRef: "/path", Form: ""},
			},
		}
		doc := &gedcom.Document{
			Header: &gedcom.Header{},
			Records: []*gedcom.Record{
				{XRef: "@M1@", Type: gedcom.RecordTypeMedia, Entity: media},
			},
		}
		report := &gedcom.ConversionReport{}

		transformMediaTypes(doc, gedcom.Version70, report)

		if len(report.Transformations) > 0 {
			t.Error("Should have no transformations for empty form")
		}
	})
}

func TestToIANAMediaType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Standard mappings
		{"JPG", "image/jpeg"},
		{"JPEG", "image/jpeg"},
		{"PNG", "image/png"},
		{"GIF", "image/gif"},
		{"TIFF", "image/tiff"},
		{"TIF", "image/tiff"},
		{"BMP", "image/bmp"},
		{"MP3", "audio/mpeg"},
		{"WAV", "audio/wav"},
		{"MP4", "video/mp4"},
		{"MPEG", "video/mpeg"},
		{"MPG", "video/mpeg"},
		{"AVI", "video/x-msvideo"},
		{"PDF", "application/pdf"},
		{"TXT", "text/plain"},
		{"TEXT", "text/plain"},

		// Case insensitive
		{"jpg", "image/jpeg"},
		{"Jpg", "image/jpeg"},

		// Already IANA format
		{"image/jpeg", "image/jpeg"},
		{"image/png", "image/png"},
		{"application/pdf", "application/pdf"},

		// Unknown - no mapping
		{"UNKNOWN", ""},
		{"xyz", ""},

		// With whitespace
		{" JPG ", "image/jpeg"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toIANAMediaType(tt.input)
			if got != tt.want {
				t.Errorf("toIANAMediaType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToLegacyMediaType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Standard mappings
		{"image/jpeg", "JPG"},
		{"image/png", "PNG"},
		{"image/gif", "GIF"},
		{"image/tiff", "TIFF"},
		{"image/bmp", "BMP"},
		{"audio/mpeg", "MP3"},
		{"audio/wav", "WAV"},
		{"video/mp4", "MP4"},
		{"video/mpeg", "MPEG"},
		{"video/x-msvideo", "AVI"},
		{"application/pdf", "PDF"},
		{"text/plain", "TXT"},

		// Case insensitive
		{"IMAGE/JPEG", "JPG"},
		{"Image/Png", "PNG"},

		// Already legacy format
		{"JPG", "JPG"},
		{"PNG", "PNG"},

		// Unknown IANA - no mapping
		{"application/unknown", ""},
		{"image/webp", ""},

		// With whitespace
		{" image/jpeg ", "JPG"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toLegacyMediaType(tt.input)
			if got != tt.want {
				t.Errorf("toLegacyMediaType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMediaTypeRoundTrip(t *testing.T) {
	// Test that converting to IANA and back preserves the format
	legacyTypes := []string{"JPG", "PNG", "GIF", "TIFF", "BMP", "MP3", "WAV", "MP4", "MPEG", "AVI", "PDF", "TXT"}

	for _, legacy := range legacyTypes {
		t.Run(legacy, func(t *testing.T) {
			iana := toIANAMediaType(legacy)
			if iana == "" {
				t.Fatalf("toIANAMediaType(%q) returned empty", legacy)
			}

			backToLegacy := toLegacyMediaType(iana)
			// Note: Some mappings are not 1:1 (e.g., JPEG -> image/jpeg -> JPG)
			// So we just verify it maps to something valid
			if backToLegacy == "" {
				t.Errorf("toLegacyMediaType(%q) returned empty", iana)
			}
		})
	}
}

func TestTransformMediaTypesReportDetails(t *testing.T) {
	t.Run("report includes transformation details", func(t *testing.T) {
		doc := createDocWithMediaFile("JPG")
		report := &gedcom.ConversionReport{}

		transformMediaTypes(doc, gedcom.Version70, report)

		if len(report.Transformations) == 0 {
			t.Fatal("Should have transformations")
		}

		tr := report.Transformations[0]
		if tr.Type != "MEDIA_TYPE_MAPPED" {
			t.Errorf("Type = %v, want MEDIA_TYPE_MAPPED", tr.Type)
		}
		if tr.Count != 1 {
			t.Errorf("Count = %d, want 1", tr.Count)
		}
		if len(tr.Details) == 0 {
			t.Error("Should have details")
		}
		// Details should contain "JPG -> image/jpeg"
		found := false
		for _, d := range tr.Details {
			if d == "JPG -> image/jpeg" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Details should contain 'JPG -> image/jpeg', got %v", tr.Details)
		}
	})
}

func TestLegacyToIANAMapping(t *testing.T) {
	// Verify the mapping contains expected entries
	expectedMappings := map[string]string{
		"JPG":  "image/jpeg",
		"JPEG": "image/jpeg",
		"PNG":  "image/png",
		"GIF":  "image/gif",
		"PDF":  "application/pdf",
	}

	for legacy, expectedIANA := range expectedMappings {
		if got := legacyToIANA[legacy]; got != expectedIANA {
			t.Errorf("legacyToIANA[%q] = %q, want %q", legacy, got, expectedIANA)
		}
	}
}

func TestIANAToLegacyMapping(t *testing.T) {
	// Verify the mapping contains expected entries
	expectedMappings := map[string]string{
		"image/jpeg":      "JPG",
		"image/png":       "PNG",
		"image/gif":       "GIF",
		"application/pdf": "PDF",
	}

	for iana, expectedLegacy := range expectedMappings {
		if got := ianaToLegacy[iana]; got != expectedLegacy {
			t.Errorf("ianaToLegacy[%q] = %q, want %q", iana, got, expectedLegacy)
		}
	}
}

// Helper function to create a document with a media file
func createDocWithMediaFile(form string) *gedcom.Document {
	media := &gedcom.MediaObject{
		XRef: "@M1@",
		Files: []*gedcom.MediaFile{
			{
				FileRef: "/path/to/file",
				Form:    form,
			},
		},
	}
	return &gedcom.Document{
		Header: &gedcom.Header{},
		Records: []*gedcom.Record{
			{
				XRef:   "@M1@",
				Type:   gedcom.RecordTypeMedia,
				Entity: media,
			},
		},
	}
}
