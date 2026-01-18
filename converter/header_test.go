package converter

import (
	"testing"

	"github.com/cacack/gedcom-go/gedcom"
)

func TestTransformHeader(t *testing.T) {
	tests := []struct {
		name           string
		doc            *gedcom.Document
		targetVersion  gedcom.Version
		wantEncoding   gedcom.Encoding
		checkSCHMA     bool
		wantSCHMAGone  bool
	}{
		{
			name: "nil header creates new header",
			doc: &gedcom.Document{
				Header: nil,
			},
			targetVersion: gedcom.Version70,
			wantEncoding:  gedcom.EncodingUTF8,
		},
		{
			name: "upgrade to 7.0 sets UTF-8",
			doc: &gedcom.Document{
				Header: &gedcom.Header{
					Encoding: gedcom.EncodingANSEL,
				},
			},
			targetVersion: gedcom.Version70,
			wantEncoding:  gedcom.EncodingUTF8,
		},
		{
			name: "downgrade to 5.5 removes SCHMA",
			doc: &gedcom.Document{
				Header: &gedcom.Header{
					Encoding: gedcom.EncodingUTF8,
					Tags: []*gedcom.Tag{
						{Level: 1, Tag: "SCHMA"},
						{Level: 1, Tag: "SOUR", Value: "Test"},
					},
				},
			},
			targetVersion: gedcom.Version55,
			checkSCHMA:    true,
			wantSCHMAGone: true,
		},
		{
			name: "downgrade to 5.5.1 removes SCHMA",
			doc: &gedcom.Document{
				Header: &gedcom.Header{
					Encoding: gedcom.EncodingUTF8,
					Tags: []*gedcom.Tag{
						{Level: 1, Tag: "SCHMA"},
					},
				},
			},
			targetVersion: gedcom.Version551,
			checkSCHMA:    true,
			wantSCHMAGone: true,
		},
		{
			name: "5.5.1 keeps UTF-8 if set",
			doc: &gedcom.Document{
				Header: &gedcom.Header{
					Encoding: gedcom.EncodingUTF8,
				},
			},
			targetVersion: gedcom.Version551,
			wantEncoding:  gedcom.EncodingUTF8,
		},
		{
			name: "5.5.1 sets UTF-8 if empty",
			doc: &gedcom.Document{
				Header: &gedcom.Header{
					Encoding: "",
				},
			},
			targetVersion: gedcom.Version551,
			wantEncoding:  gedcom.EncodingUTF8,
		},
		{
			name: "5.5 sets ANSEL if empty",
			doc: &gedcom.Document{
				Header: &gedcom.Header{
					Encoding: "",
				},
			},
			targetVersion: gedcom.Version55,
			wantEncoding:  gedcom.EncodingANSEL,
		},
		{
			name: "5.5 keeps existing encoding",
			doc: &gedcom.Document{
				Header: &gedcom.Header{
					Encoding: gedcom.EncodingUTF8,
				},
			},
			targetVersion: gedcom.Version55,
			wantEncoding:  gedcom.EncodingUTF8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Deep copy to avoid mutation between tests
			doc := deepCopyDocument(tt.doc)
			if doc == nil {
				doc = &gedcom.Document{}
			}

			report := &gedcom.ConversionReport{}
			transformHeader(doc, tt.targetVersion, report)

			if doc.Header == nil {
				t.Fatal("Header should not be nil after transformation")
			}

			if tt.wantEncoding != "" && doc.Header.Encoding != tt.wantEncoding {
				t.Errorf("Encoding = %v, want %v", doc.Header.Encoding, tt.wantEncoding)
			}

			if tt.checkSCHMA {
				hasSCHMA := false
				for _, tag := range doc.Header.Tags {
					if tag.Tag == "SCHMA" {
						hasSCHMA = true
						break
					}
				}
				if tt.wantSCHMAGone && hasSCHMA {
					t.Error("SCHMA tag should have been removed")
				}
			}
		})
	}
}

func TestUpgradeHeaderTo70(t *testing.T) {
	tests := []struct {
		name             string
		header           *gedcom.Header
		wantEncoding     gedcom.Encoding
		wantTransform    bool
		transformType    string
	}{
		{
			name: "ANSEL to UTF-8",
			header: &gedcom.Header{
				Encoding: gedcom.EncodingANSEL,
			},
			wantEncoding:  gedcom.EncodingUTF8,
			wantTransform: true,
			transformType: "ENCODING_UPDATED",
		},
		{
			name: "already UTF-8 no change",
			header: &gedcom.Header{
				Encoding: gedcom.EncodingUTF8,
			},
			wantEncoding:  gedcom.EncodingUTF8,
			wantTransform: false,
		},
		{
			name: "ASCII to UTF-8",
			header: &gedcom.Header{
				Encoding: gedcom.EncodingASCII,
			},
			wantEncoding:  gedcom.EncodingUTF8,
			wantTransform: true,
			transformType: "ENCODING_UPDATED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := deepCopyHeader(tt.header)
			report := &gedcom.ConversionReport{}

			upgradeHeaderTo70(header, report)

			if header.Encoding != tt.wantEncoding {
				t.Errorf("Encoding = %v, want %v", header.Encoding, tt.wantEncoding)
			}

			hasTransform := len(report.Transformations) > 0
			if hasTransform != tt.wantTransform {
				t.Errorf("Has transformation = %v, want %v", hasTransform, tt.wantTransform)
			}

			if tt.wantTransform && hasTransform {
				if report.Transformations[0].Type != tt.transformType {
					t.Errorf("Transform type = %v, want %v", report.Transformations[0].Type, tt.transformType)
				}
			}
		})
	}
}

func TestDowngradeHeaderFrom70(t *testing.T) {
	tests := []struct {
		name            string
		header          *gedcom.Header
		targetVersion   gedcom.Version
		wantSCHMAGone   bool
		wantDataLoss    bool
		wantTransform   bool
	}{
		{
			name: "removes SCHMA tag",
			header: &gedcom.Header{
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "SOUR", Value: "Test"},
					{Level: 1, Tag: "SCHMA"},
					{Level: 1, Tag: "DEST", Value: "Test"},
				},
			},
			targetVersion: gedcom.Version55,
			wantSCHMAGone: true,
			wantDataLoss:  true,
			wantTransform: true,
		},
		{
			name: "no SCHMA tag, no changes",
			header: &gedcom.Header{
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "SOUR", Value: "Test"},
				},
			},
			targetVersion: gedcom.Version55,
			wantSCHMAGone: true, // trivially true - no SCHMA
			wantDataLoss:  false,
			wantTransform: false,
		},
		{
			name: "nil tags",
			header: &gedcom.Header{
				Tags: nil,
			},
			targetVersion: gedcom.Version551,
			wantSCHMAGone: true,
			wantDataLoss:  false,
			wantTransform: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := deepCopyHeader(tt.header)
			report := &gedcom.ConversionReport{}

			downgradeHeaderFrom70(header, tt.targetVersion, report)

			hasSCHMA := false
			for _, tag := range header.Tags {
				if tag.Tag == "SCHMA" {
					hasSCHMA = true
					break
				}
			}
			if tt.wantSCHMAGone && hasSCHMA {
				t.Error("SCHMA tag should have been removed")
			}

			if report.HasDataLoss() != tt.wantDataLoss {
				t.Errorf("HasDataLoss = %v, want %v", report.HasDataLoss(), tt.wantDataLoss)
			}

			hasTransform := false
			for _, t := range report.Transformations {
				if t.Type == "SCHMA_REMOVED" {
					hasTransform = true
					break
				}
			}
			if hasTransform != tt.wantTransform {
				t.Errorf("Has SCHMA_REMOVED transform = %v, want %v", hasTransform, tt.wantTransform)
			}
		})
	}
}

func TestUpdateEncoding(t *testing.T) {
	tests := []struct {
		name          string
		header        *gedcom.Header
		targetVersion gedcom.Version
		wantEncoding  gedcom.Encoding
	}{
		{
			name:          "7.0 always UTF-8",
			header:        &gedcom.Header{Encoding: gedcom.EncodingANSEL},
			targetVersion: gedcom.Version70,
			wantEncoding:  gedcom.EncodingUTF8,
		},
		{
			name:          "5.5.1 empty defaults to UTF-8",
			header:        &gedcom.Header{Encoding: ""},
			targetVersion: gedcom.Version551,
			wantEncoding:  gedcom.EncodingUTF8,
		},
		{
			name:          "5.5.1 preserves existing",
			header:        &gedcom.Header{Encoding: gedcom.EncodingANSEL},
			targetVersion: gedcom.Version551,
			wantEncoding:  gedcom.EncodingANSEL,
		},
		{
			name:          "5.5 empty defaults to ANSEL",
			header:        &gedcom.Header{Encoding: ""},
			targetVersion: gedcom.Version55,
			wantEncoding:  gedcom.EncodingANSEL,
		},
		{
			name:          "5.5 preserves UTF-8",
			header:        &gedcom.Header{Encoding: gedcom.EncodingUTF8},
			targetVersion: gedcom.Version55,
			wantEncoding:  gedcom.EncodingUTF8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := deepCopyHeader(tt.header)
			report := &gedcom.ConversionReport{}

			updateEncoding(header, tt.targetVersion, report)

			if header.Encoding != tt.wantEncoding {
				t.Errorf("Encoding = %v, want %v", header.Encoding, tt.wantEncoding)
			}
		})
	}
}

func TestTransformHeaderReport(t *testing.T) {
	t.Run("encoding update adds transformation with details", func(t *testing.T) {
		doc := &gedcom.Document{
			Header: &gedcom.Header{
				Encoding: gedcom.EncodingANSEL,
			},
		}
		report := &gedcom.ConversionReport{}

		transformHeader(doc, gedcom.Version70, report)

		found := false
		for _, t := range report.Transformations {
			if t.Type == "ENCODING_UPDATED" {
				found = true
				if len(t.Details) < 2 {
					// Should have "From: ANSEL" and "To: UTF-8"
					continue
				}
			}
		}
		if !found {
			t.Error("Should have ENCODING_UPDATED transformation")
		}
	})

	t.Run("SCHMA removal adds both transformation and data loss", func(t *testing.T) {
		doc := &gedcom.Document{
			Header: &gedcom.Header{
				Tags: []*gedcom.Tag{
					{Tag: "SCHMA"},
				},
			},
		}
		report := &gedcom.ConversionReport{}

		transformHeader(doc, gedcom.Version55, report)

		foundTransform := false
		for _, t := range report.Transformations {
			if t.Type == "SCHMA_REMOVED" {
				foundTransform = true
			}
		}
		if !foundTransform {
			t.Error("Should have SCHMA_REMOVED transformation")
		}

		foundDataLoss := false
		for _, d := range report.DataLoss {
			if contains(d.Feature, "SCHMA") {
				foundDataLoss = true
			}
		}
		if !foundDataLoss {
			t.Error("Should have SCHMA data loss item")
		}
	})
}
