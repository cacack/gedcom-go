package converter

import (
	"slices"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
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

// formTagValues returns the values of the record's FORM tags at level, in order.
func formTagValues(record *gedcom.Record, level int) []string {
	var out []string
	for _, tag := range record.Tags {
		if tag != nil && tag.Tag == "FORM" && tag.Level == level {
			out = append(out, tag.Value)
		}
	}
	return out
}

// TestConvertDecodedMediaTypes is issue #777: a decoded document's FORM values
// are mapped in Record.Tags, so the encoder writes them and the re-derived
// entity agrees, and a value is reported only when it changed.
func TestConvertDecodedMediaTypes(t *testing.T) {
	t.Run("5.5.1 to 7.0", func(t *testing.T) {
		const in = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
			"0 @O1@ OBJE\n1 FILE photo.jpg\n2 FORM JPG\n" +
			"1 FILE scan.png\n2 FORM image/png\n0 TRLR\n"
		converted, report, out := convertText(t, in, gedcom.Version70)

		if !strings.Contains(out, "1 FILE photo.jpg\n2 FORM image/jpeg\n") {
			t.Errorf("output lacks the mapped FORM:\n%s", out)
		}
		record := converted.XRefMap["@O1@"]
		if got := formTagValues(record, 2); !slices.Equal(got, []string{"image/jpeg", "image/png"}) {
			t.Errorf("Tags FORM = %q, want [image/jpeg image/png]", got)
		}
		media, _ := record.GetMediaObject()
		if media.Files[0].Form != "image/jpeg" {
			t.Errorf("entity Form = %q, want image/jpeg", media.Files[0].Form)
		}

		path := BuildNestedPath("OBJE", "@O1@", "FILE[0]", "FORM")
		if !hasApproximated(report, path, "JPG", "image/jpeg") {
			t.Errorf("no Approximated note at %q: %+v", path, report.Approximated)
		}
		for _, n := range report.Approximated {
			if n.Original == "image/png" {
				t.Errorf("unchanged value reported: %+v", n)
			}
		}
		for _, tr := range report.Transformations {
			if tr.Type == "MEDIA_TYPE_MAPPED" && !slices.Equal(tr.Details, []string{"JPG -> image/jpeg"}) {
				t.Errorf("MEDIA_TYPE_MAPPED details = %q, want [JPG -> image/jpeg]", tr.Details)
			}
		}
	})

	t.Run("7.0 to 5.5.1", func(t *testing.T) {
		const in = "0 HEAD\n1 GEDC\n2 VERS 7.0\n" +
			"0 @O1@ OBJE\n1 FILE photo.jpg\n2 FORM image/jpeg\n" +
			"2 TRAN photo.pdf\n3 FORM application/pdf\n0 TRLR\n"
		converted, report, out := convertText(t, in, gedcom.Version551)

		if !strings.Contains(out, "1 FILE photo.jpg\n2 FORM JPG\n") {
			t.Errorf("output lacks the mapped FORM:\n%s", out)
		}
		record := converted.XRefMap["@O1@"]
		if got := formTagValues(record, 2); !slices.Equal(got, []string{"JPG"}) {
			t.Errorf("Tags FILE.FORM = %q, want [JPG]", got)
		}
		if strings.Contains(out, "TRAN") {
			t.Errorf("output keeps the 7.0-only TRAN:\n%s", out)
		}
		if !hasApproximated(report, BuildNestedPath("OBJE", "@O1@", "FILE[0]", "FORM"), "image/jpeg", "JPG") {
			t.Errorf("no Approximated note for FILE.FORM: %+v", report.Approximated)
		}
		// The TRAN is dropped, not mapped, so only its drop is reported.
		for _, n := range report.Approximated {
			if n.Original == "application/pdf" {
				t.Errorf("dropped TRAN's FORM reported as mapped: %+v", n)
			}
		}
	})

	t.Run("nothing to map reports nothing", func(t *testing.T) {
		const in = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
			"0 @O1@ OBJE\n1 FILE scan.png\n2 FORM image/png\n0 TRLR\n"
		_, report, _ := convertText(t, in, gedcom.Version70)
		if hasTransformation(report, "MEDIA_TYPE_MAPPED") {
			t.Errorf("MEDIA_TYPE_MAPPED reported with nothing changed: %+v", report.Transformations)
		}
		if len(report.Approximated) > 0 {
			t.Errorf("Approximated notes with nothing changed: %+v", report.Approximated)
		}
	})
}

// TestTransformMediaTypesTaggedTranslation: a tagged record's FILE.TRAN.FORM
// is mapped on a conversion to 7.0.
func TestTransformMediaTypesTaggedTranslation(t *testing.T) {
	form := &gedcom.Tag{Level: 3, Tag: "FORM", Value: "PNG"}
	doc := &gedcom.Document{Records: []*gedcom.Record{{
		XRef: "@O1@",
		Type: gedcom.RecordTypeMedia,
		Tags: []*gedcom.Tag{
			{Level: 1, Tag: "FILE", Value: "a.jpg"},
			{Level: 2, Tag: "FORM", Value: "image/jpeg"},
			{Level: 2, Tag: "TRAN", Value: "a.png"},
			form,
		},
	}}}
	report := &gedcom.ConversionReport{}
	transformMediaTypes(doc, gedcom.Version70, report)
	if form.Value != "image/png" {
		t.Errorf("TRAN FORM = %q, want image/png", form.Value)
	}
	if !hasApproximated(report, BuildNestedPath("OBJE", "@O1@", "FILE[0]", "TRAN[0]", "FORM"), "PNG", "image/png") {
		t.Errorf("no Approximated note for FILE.TRAN.FORM: %+v", report.Approximated)
	}
}

// TestTransformMediaTypesInlineLinks: to 5.x, an inline link's FORM is mapped
// in both the 5.5 (OBJE.FORM) and 5.5.1 (OBJE.FILE.FORM) layouts, and a FORM
// under a pointer or another structure is not.
func TestTransformMediaTypesInlineLinks(t *testing.T) {
	form55 := &gedcom.Tag{Level: 2, Tag: "FORM", Value: "image/jpeg"}
	form551 := &gedcom.Tag{Level: 4, Tag: "FORM", Value: "image/png"}
	pointerForm := &gedcom.Tag{Level: 2, Tag: "FORM", Value: "image/gif"}
	otherForm := &gedcom.Tag{Level: 2, Tag: "FORM", Value: "image/bmp"}
	doc := &gedcom.Document{Records: []*gedcom.Record{{
		XRef: "@I1@",
		Type: gedcom.RecordTypeIndividual,
		Tags: []*gedcom.Tag{
			{Level: 1, Tag: "OBJE"},
			form55,
			{Level: 2, Tag: "FILE", Value: "a.jpg"},
			{Level: 1, Tag: "BIRT"},
			{Level: 2, Tag: "OBJE"},
			{Level: 3, Tag: "FILE", Value: "b.png"},
			form551,
			{Level: 1, Tag: "OBJE", Value: "@O1@"},
			pointerForm,
			{Level: 1, Tag: "_X"},
			otherForm,
		},
	}}}
	report := &gedcom.ConversionReport{}
	transformMediaTypes(doc, gedcom.Version551, report)
	if form55.Value != "JPG" || form551.Value != "PNG" {
		t.Errorf("inline FORMs = %q, %q, want JPG, PNG", form55.Value, form551.Value)
	}
	if pointerForm.Value != "image/gif" || otherForm.Value != "image/bmp" {
		t.Errorf("non-link FORMs mapped: %q, %q", pointerForm.Value, otherForm.Value)
	}
	if !hasApproximated(report, "Individual @I1@ > BIRT > OBJE > FILE > FORM", "image/png", "PNG") {
		t.Errorf("no Approximated note at the link's path: %+v", report.Approximated)
	}
	if len(report.Approximated) != 2 {
		t.Errorf("Approximated = %+v, want 2 notes", report.Approximated)
	}
}
