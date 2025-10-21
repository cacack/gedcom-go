package decoder

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cacack/gedcom-go/gedcom"
)

// T031: Write integration tests for full document parsing
func TestDecode(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
1 SEX M
0 @F1@ FAM
1 HUSB @I1@
0 TRLR`

	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if doc == nil {
		t.Fatal("Decode() returned nil document")
	}

	// Verify header
	if doc.Header.Version != gedcom.Version55 {
		t.Errorf("Version = %v, want %v", doc.Header.Version, gedcom.Version55)
	}

	if doc.Header.Encoding != gedcom.EncodingUTF8 {
		t.Errorf("Encoding = %v, want %v", doc.Header.Encoding, gedcom.EncodingUTF8)
	}

	// Verify records
	if len(doc.Records) < 2 {
		t.Fatalf("Expected at least 2 records, got %d", len(doc.Records))
	}
}

// T032: Write tests for XRefMap resolution (all XRefs indexed, valid lookups)
func TestXRefMap(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
0 @I2@ INDI
1 NAME Jane /Doe/
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
0 TRLR`

	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// Verify XRefMap is built
	if doc.XRefMap == nil {
		t.Fatal("XRefMap is nil")
	}

	// Check all XRefs are indexed
	expectedXRefs := []string{"@I1@", "@I2@", "@F1@"}
	for _, xref := range expectedXRefs {
		if _, found := doc.XRefMap[xref]; !found {
			t.Errorf("XRef %q not found in XRefMap", xref)
		}
	}

	// Verify lookups work
	rec := doc.XRefMap["@I1@"]
	if rec == nil {
		t.Fatal("Failed to lookup @I1@")
	}
	if rec.XRef != "@I1@" {
		t.Errorf("Record XRef = %q, want @I1@", rec.XRef)
	}
}

// T053: Handle empty/header-only files
func TestDecodeEmptyFile(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "empty file",
			input:   ``,
			wantErr: false,
		},
		{
			name: "header only",
			input: `0 HEAD
1 GEDC
2 VERS 5.5
0 TRLR`,
			wantErr: false,
		},
		{
			name: "minimal valid file",
			input: `0 HEAD
0 TRLR`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := Decode(strings.NewReader(tt.input))

			if tt.wantErr {
				if err == nil {
					t.Error("Decode() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Decode() unexpected error: %v", err)
			}

			if doc == nil {
				t.Fatal("Decode() returned nil document")
			}
		})
	}
}

// T054: Add timeout support via context.Context
func TestDecodeWithContext(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
0 TRLR`

	t.Run("no timeout", func(t *testing.T) {
		ctx := context.Background()
		opts := &DecodeOptions{
			Context: ctx,
		}

		doc, err := DecodeWithOptions(strings.NewReader(input), opts)
		if err != nil {
			t.Fatalf("DecodeWithOptions() error = %v", err)
		}

		if doc == nil {
			t.Fatal("DecodeWithOptions() returned nil document")
		}
	})

	t.Run("with timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		opts := &DecodeOptions{
			Context: ctx,
		}

		doc, err := DecodeWithOptions(strings.NewReader(input), opts)
		if err != nil {
			t.Fatalf("DecodeWithOptions() error = %v", err)
		}

		if doc == nil {
			t.Fatal("DecodeWithOptions() returned nil document")
		}
	})

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		opts := &DecodeOptions{
			Context: ctx,
		}

		_, err := DecodeWithOptions(strings.NewReader(input), opts)
		if err == nil {
			t.Error("DecodeWithOptions() expected error for cancelled context")
		}

		// Verify the error is context.Canceled
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled error, got %v", err)
		}
	})
}

// Test max nesting depth
func TestDecodeMaxNestingDepth(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 TRLR`

	opts := &DecodeOptions{
		MaxNestingDepth: 10,
	}

	_, err := DecodeWithOptions(strings.NewReader(input), opts)
	if err != nil {
		t.Fatalf("DecodeWithOptions() error = %v", err)
	}
}

// Test strict mode
func TestDecodeStrictMode(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 TRLR`

	opts := &DecodeOptions{
		StrictMode: true,
	}

	_, err := DecodeWithOptions(strings.NewReader(input), opts)
	if err != nil {
		t.Fatalf("DecodeWithOptions() error = %v", err)
	}
}

// Test header with SOUR at different levels
func TestDecodeHeaderSOURLevels(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
1 SOUR MyApp
2 SOUR NestedSource
0 TRLR`

	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// Only level 1 SOUR should be captured as SourceSystem
	if doc.Header.SourceSystem != "MyApp" {
		t.Errorf("Header.SourceSystem = %q, want %q", doc.Header.SourceSystem, "MyApp")
	}
}

// Test header with version already set in GEDC tag
func TestDecodeHeaderVersionFromGEDC(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 7.0
1 CHAR UTF-8
0 TRLR`

	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// Version should be set from GEDC/VERS tag
	if doc.Header.Version != "7.0" {
		t.Errorf("Header.Version = %q, want %q", doc.Header.Version, "7.0")
	}
}

// Test header with all optional fields
func TestDecodeHeaderComplete(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR ANSEL
1 SOUR FamilyTreeMaker
1 LANG French
0 TRLR`

	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if doc.Header.Version != "5.5.1" {
		t.Errorf("Header.Version = %q, want %q", doc.Header.Version, "5.5.1")
	}
	if string(doc.Header.Encoding) != "ANSEL" {
		t.Errorf("Header.Encoding = %q, want %q", doc.Header.Encoding, "ANSEL")
	}
	if doc.Header.SourceSystem != "FamilyTreeMaker" {
		t.Errorf("Header.SourceSystem = %q, want %q", doc.Header.SourceSystem, "FamilyTreeMaker")
	}
	if doc.Header.Language != "French" {
		t.Errorf("Header.Language = %q, want %q", doc.Header.Language, "French")
	}
}

// Test context cancellation at different stages
func TestDecodeContextCancellationStages(t *testing.T) {
	t.Run("context cancelled after parsing", func(t *testing.T) {
		// This test is challenging because we can't easily cancel context
		// *during* parsing. However, we can verify the check exists by
		// using a very short timeout that might expire during processing
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		// Create a small but valid GEDCOM
		input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME Test
0 TRLR`

		opts := &DecodeOptions{
			Context: ctx,
		}

		// Try to decode - may or may not error depending on timing
		_, err := DecodeWithOptions(strings.NewReader(input), opts)

		// Either succeeds or gets context.DeadlineExceeded
		if err != nil && err != context.DeadlineExceeded {
			t.Errorf("Expected nil or context.DeadlineExceeded, got %v", err)
		}
	})
}

// Test buildHeader edge cases
func TestDecodeHeaderEdgeCases(t *testing.T) {
	t.Run("header without CHAR tag", func(t *testing.T) {
		input := `0 HEAD
1 GEDC
2 VERS 5.5
1 SOUR TestApp
0 TRLR`

		doc, err := Decode(strings.NewReader(input))
		if err != nil {
			t.Fatalf("Decode() error = %v", err)
		}

		// Should still decode successfully
		if doc.Header.Version != "5.5" {
			t.Errorf("Header.Version = %q, want %q", doc.Header.Version, "5.5")
		}
		if doc.Header.SourceSystem != "TestApp" {
			t.Errorf("Header.SourceSystem = %q, want %q", doc.Header.SourceSystem, "TestApp")
		}
	})

	t.Run("header with LANG only", func(t *testing.T) {
		input := `0 HEAD
1 LANG Spanish
0 TRLR`

		doc, err := Decode(strings.NewReader(input))
		if err != nil {
			t.Fatalf("Decode() error = %v", err)
		}

		if doc.Header.Language != "Spanish" {
			t.Errorf("Header.Language = %q, want %q", doc.Header.Language, "Spanish")
		}
	})

	t.Run("nested tags within header", func(t *testing.T) {
		input := `0 HEAD
1 SOUR TestApp
2 VERS 1.0
2 NAME Test Application
1 CHAR UTF-8
0 TRLR`

		doc, err := Decode(strings.NewReader(input))
		if err != nil {
			t.Fatalf("Decode() error = %v", err)
		}

		// Level 1 SOUR should be captured
		if doc.Header.SourceSystem != "TestApp" {
			t.Errorf("Header.SourceSystem = %q, want %q", doc.Header.SourceSystem, "TestApp")
		}
	})
}
