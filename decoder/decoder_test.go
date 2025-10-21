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
