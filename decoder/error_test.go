package decoder

import (
	"os"
	"strings"
	"testing"
)

// T064: Test missing cross-reference targets
func TestMissingXRefTargets(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John Smith
1 FAMS @F999@
0 TRLR`

	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// Verify that @F999@ is NOT in XRefMap (broken reference)
	if doc.XRefMap["@F999@"] != nil {
		t.Error("Expected @F999@ to not be in XRefMap (broken reference)")
	}

	// Verify that @I1@ IS in XRefMap (valid reference)
	if doc.XRefMap["@I1@"] == nil {
		t.Error("Expected @I1@ to be in XRefMap")
	}
}

// Test malformed files from testdata
func TestMalformedFilesFromTestData(t *testing.T) {
	testFiles := []struct {
		name        string
		path        string
		shouldError bool
		description string
	}{
		{
			name:        "invalid level",
			path:        "../testdata/malformed/invalid-level.ged",
			shouldError: false, // Parser accepts any level < 100
			description: "File with unusually deep nesting (level 99)",
		},
		{
			name:        "missing xref",
			path:        "../testdata/malformed/missing-xref.ged",
			shouldError: false, // Decoder accepts, validation would catch
			description: "File with reference to non-existent record",
		},
	}

	for _, tt := range testFiles {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(tt.path)
			if err != nil {
				t.Skipf("Test file not found: %s", tt.path)
				return
			}
			defer f.Close()

			doc, err := Decode(f)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for %s but got none", tt.description)
				} else {
					t.Logf("Got expected error: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.description, err)
				} else {
					t.Logf("Successfully parsed %s: %d records", tt.description, len(doc.Records))
				}
			}
		})
	}
}

// Test that decoder provides helpful error messages
func TestDecoderErrorMessages(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantErr      bool
		errSubstring string
	}{
		{
			name: "invalid UTF-8",
			input: "0 HEAD\n1 NAME \xFF\xFE Invalid UTF-8\n0 TRLR",
			wantErr:      true,
			errSubstring: "error",
		},
		{
			name:         "completely invalid format",
			input:        "This is not GEDCOM at all!",
			wantErr:      true,
			errSubstring: "level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decode(strings.NewReader(tt.input))

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.wantErr && err != nil {
				errMsg := err.Error()
				if !strings.Contains(errMsg, tt.errSubstring) {
					t.Errorf("Error message %q should contain %q", errMsg, tt.errSubstring)
				}
				t.Logf("Got expected error: %v", err)
			}
		})
	}
}

// Test graceful handling of truncated files
func TestTruncatedFiles(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "truncated mid-record",
			input: `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John`,
			wantErr: false, // Should parse what's available
		},
		{
			name: "truncated in header",
			input: `0 HEAD
1 GE`,
			wantErr: false, // Should parse partial content
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := Decode(strings.NewReader(tt.input))

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.wantErr && doc != nil {
				t.Logf("Successfully parsed truncated file: %d records", len(doc.Records))
			}
		})
	}
}
