package decoder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/gedcom"
)

// ============================================================================
// Integration tests for lenient parsing with real malformed files
// ============================================================================

// TestDecodeWithDiagnostics_MalformedFiles tests lenient parsing with files from testdata/malformed/
func TestDecodeWithDiagnostics_MalformedFiles(t *testing.T) {
	tests := []struct {
		name                 string
		path                 string
		description          string
		expectDiagnostics    bool   // Should have any diagnostics
		expectError          bool   // Should return an error
		minRecords           int    // Minimum expected records (partial parsing)
		expectDiagnosticCode string // Expected diagnostic code (if any)
	}{
		{
			name:                 "invalid-level.ged",
			path:                 "../testdata/malformed/invalid-level.ged",
			description:          "File with a level-99 jump (level itself is < MaxNestingDepth, but the +98 jump is malformed indentation)",
			expectDiagnostics:    true,
			expectError:          false,
			minRecords:           1, // At least one INDI record
			expectDiagnosticCode: CodeBadLevelJump,
		},
		{
			name:              "invalid-xref.ged",
			path:              "../testdata/malformed/invalid-xref.ged",
			description:       "File with reference to non-existent family",
			expectDiagnostics: false, // Parser accepts, broken XRef is semantic
			expectError:       false,
			minRecords:        1,
		},
		{
			name:              "missing-header.ged",
			path:              "../testdata/malformed/missing-header.ged",
			description:       "File missing HEAD record",
			expectDiagnostics: false, // Missing header is semantic, not syntactic
			expectError:       false,
			minRecords:        1,
		},
		{
			name:              "missing-xref.ged",
			path:              "../testdata/malformed/missing-xref.ged",
			description:       "File with reference to non-existent family",
			expectDiagnostics: false, // Parser accepts, broken XRef is semantic
			expectError:       false,
			minRecords:        1,
		},
		{
			name:              "circular-reference.ged",
			path:              "../testdata/malformed/circular-reference.ged",
			description:       "File with circular family relationships",
			expectDiagnostics: false, // Circular refs are semantic, not syntactic
			expectError:       false,
			minRecords:        3, // Multiple individuals and families
		},
		{
			name:              "duplicate-xref.ged",
			path:              "../testdata/malformed/duplicate-xref.ged",
			description:       "File with duplicate XRef identifiers",
			expectDiagnostics: false, // Duplicate XRefs are handled (last wins)
			expectError:       false,
			minRecords:        1, // At least one record
		},
		{
			name:                 "level-jump-skip.ged",
			path:                 "../testdata/malformed/level-jump-skip.ged",
			description:          "File with level jump 1 -> 4 (real-world Ancestry-style export)",
			expectDiagnostics:    true,
			expectError:          false,
			minRecords:           1,
			expectDiagnosticCode: CodeBadLevelJump,
		},
		{
			name:                 "level-jump-subordinate.ged",
			path:                 "../testdata/malformed/level-jump-subordinate.ged",
			description:          "File where a subordinate skips a level (PLAC jumps from 1 to 3)",
			expectDiagnostics:    true,
			expectError:          false,
			minRecords:           1,
			expectDiagnosticCode: CodeBadLevelJump,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(tt.path)
			if err != nil {
				t.Skipf("Test file not found: %s", tt.path)
				return
			}
			defer f.Close()

			result, err := DecodeWithDiagnostics(f, nil)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s but got none", tt.description)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			}

			if result == nil {
				t.Fatal("DecodeWithDiagnostics returned nil result")
			}

			if result.Document == nil {
				t.Fatal("DecodeWithDiagnostics returned nil document")
			}

			// Check diagnostics
			if tt.expectDiagnostics && len(result.Diagnostics) == 0 {
				t.Error("Expected diagnostics but got none")
			}

			// Check specific diagnostic code if expected
			if tt.expectDiagnosticCode != "" {
				found := false
				for _, diag := range result.Diagnostics {
					if diag.Code == tt.expectDiagnosticCode {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected diagnostic with code %s not found", tt.expectDiagnosticCode)
				}
			}

			// Check minimum records (partial parsing)
			if len(result.Document.Records) < tt.minRecords {
				t.Errorf("Expected at least %d records, got %d",
					tt.minRecords, len(result.Document.Records))
			}

			t.Logf("Parsed %s: %d records, %d diagnostics",
				tt.description, len(result.Document.Records), len(result.Diagnostics))
		})
	}
}

// TestDecodeWithDiagnostics_SyntheticMalformedInput tests with synthetic malformed input
func TestDecodeWithDiagnostics_SyntheticMalformedInput(t *testing.T) {
	tests := []struct {
		name                 string
		input                string
		expectDiagnostics    int    // Expected number of diagnostics
		expectRecords        int    // Expected number of records
		expectError          bool   // Should return an error
		expectDiagnosticCode string // Expected diagnostic code
	}{
		{
			name: "empty lines mixed with valid",
			input: `0 HEAD
1 GEDC
2 VERS 5.5

0 @I1@ INDI
1 NAME John /Smith/

0 TRLR`,
			expectDiagnostics:    2, // Two empty lines
			expectRecords:        1,
			expectError:          false,
			expectDiagnosticCode: CodeEmptyLine,
		},
		{
			name: "invalid level mixed with valid",
			input: `0 HEAD
1 GEDC
2 VERS 5.5
XYZ INVALID LINE
0 @I1@ INDI
1 NAME Jane /Doe/
ANOTHER BAD LINE
0 TRLR`,
			expectDiagnostics:    2, // Two invalid lines
			expectRecords:        1,
			expectError:          false,
			expectDiagnosticCode: CodeInvalidLevel,
		},
		{
			name: "missing tag after level",
			input: `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1
0 TRLR`,
			expectDiagnostics:    1,
			expectRecords:        1,
			expectError:          false,
			expectDiagnosticCode: CodeSyntaxError,
		},
		{
			name: "xref without tag",
			input: `0 HEAD
1 GEDC
2 VERS 5.5
0 @BADXREF@
0 @I1@ INDI
1 NAME Valid /Person/
0 TRLR`,
			expectDiagnostics:    1,
			expectRecords:        1,
			expectError:          false,
			expectDiagnosticCode: CodeInvalidXRef,
		},
		{
			name: "all lines invalid",
			input: `invalid1
invalid2
invalid3`,
			expectDiagnostics: 3,
			expectRecords:     0,
			expectError:       true, // Error when no valid lines
		},
		{
			name:              "valid file no errors",
			input:             "0 HEAD\n1 GEDC\n2 VERS 5.5\n0 @I1@ INDI\n1 NAME Test\n0 TRLR",
			expectDiagnostics: 0,
			expectRecords:     1,
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DecodeWithDiagnostics(strings.NewReader(tt.input), nil)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			}

			if result == nil {
				if tt.expectError {
					// If error expected and result is nil, that's acceptable
					return
				}
				t.Fatal("DecodeWithDiagnostics returned nil result")
			}

			if len(result.Diagnostics) != tt.expectDiagnostics {
				t.Errorf("Expected %d diagnostics, got %d",
					tt.expectDiagnostics, len(result.Diagnostics))
				for _, d := range result.Diagnostics {
					t.Logf("  Diagnostic: %s", d.String())
				}
			}

			if result.Document != nil && len(result.Document.Records) != tt.expectRecords {
				t.Errorf("Expected %d records, got %d",
					tt.expectRecords, len(result.Document.Records))
			}

			if tt.expectDiagnosticCode != "" && len(result.Diagnostics) > 0 {
				found := false
				for _, diag := range result.Diagnostics {
					if diag.Code == tt.expectDiagnosticCode {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected diagnostic with code %s not found",
						tt.expectDiagnosticCode)
				}
			}
		})
	}
}

// TestDecodeWithDiagnostics_RecoveredRecordsUsable verifies recovered records are fully usable
func TestDecodeWithDiagnostics_RecoveredRecordsUsable(t *testing.T) {
	// Input with errors but recoverable individual record
	input := `0 HEAD
1 GEDC
2 VERS 5.5

0 @I1@ INDI
1 NAME John /Smith/
1 SEX M
1 BIRT
2 DATE 1 JAN 1950
2 PLAC New York, NY

0 @F1@ FAM
1 HUSB @I1@
0 TRLR`

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}

	if result == nil || result.Document == nil {
		t.Fatal("Result or document is nil")
	}

	// Should have two diagnostics for empty lines (one before INDI, one before FAM)
	if len(result.Diagnostics) != 2 {
		t.Errorf("Expected 2 diagnostics, got %d", len(result.Diagnostics))
	}

	// Verify individual record is usable
	individual := result.Document.GetIndividual("@I1@")
	if individual == nil {
		t.Fatal("GetIndividual(@I1@) returned nil")
	}

	// Check name
	if len(individual.Names) != 1 {
		t.Errorf("Expected 1 name, got %d", len(individual.Names))
	}
	if individual.Names[0].Full != "John /Smith/" {
		t.Errorf("Name = %q, want %q", individual.Names[0].Full, "John /Smith/")
	}

	// Check sex
	if individual.Sex != "M" {
		t.Errorf("Sex = %q, want %q", individual.Sex, "M")
	}

	// Check birth event
	if len(individual.Events) == 0 {
		t.Fatal("No events found")
	}

	// Verify family record is usable
	family := result.Document.GetFamily("@F1@")
	if family == nil {
		t.Fatal("GetFamily(@F1@) returned nil")
	}

	if family.Husband != "@I1@" {
		t.Errorf("Husband = %q, want %q", family.Husband, "@I1@")
	}

	// Verify XRefMap is populated
	if result.Document.XRefMap["@I1@"] == nil {
		t.Error("XRefMap[@I1@] is nil")
	}
	if result.Document.XRefMap["@F1@"] == nil {
		t.Error("XRefMap[@F1@] is nil")
	}
}

// TestDecodeWithDiagnostics_StrictVsLenient compares strict and lenient mode
func TestDecodeWithDiagnostics_StrictVsLenient(t *testing.T) {
	// Input with an error in the middle
	input := `0 HEAD
1 GEDC
2 VERS 5.5
INVALID LINE
0 @I1@ INDI
1 NAME Test
0 TRLR`

	t.Run("strict mode fails on error", func(t *testing.T) {
		opts := &DecodeOptions{StrictMode: true}
		result, err := DecodeWithDiagnostics(strings.NewReader(input), opts)

		if err == nil {
			t.Error("Expected error in strict mode")
		}
		if result != nil {
			t.Error("Expected nil result in strict mode on error")
		}
	})

	t.Run("lenient mode continues after error", func(t *testing.T) {
		opts := &DecodeOptions{StrictMode: false}
		result, err := DecodeWithDiagnostics(strings.NewReader(input), opts)

		if err != nil {
			t.Fatalf("Unexpected error in lenient mode: %v", err)
		}
		if result == nil {
			t.Fatal("Expected non-nil result in lenient mode")
		}
		if len(result.Diagnostics) == 0 {
			t.Error("Expected diagnostics in lenient mode")
		}
		if len(result.Document.Records) != 1 {
			t.Errorf("Expected 1 record in lenient mode, got %d",
				len(result.Document.Records))
		}
	})
}

// TestDecodeWithDiagnostics_AllMalformedFilesIntegration runs through all malformed test files
func TestDecodeWithDiagnostics_AllMalformedFilesIntegration(t *testing.T) {
	malformedDir := "../testdata/malformed"

	entries, err := os.ReadDir(malformedDir)
	if err != nil {
		t.Skipf("Could not read malformed directory: %v", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".ged") {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			path := filepath.Join(malformedDir, entry.Name())
			f, err := os.Open(path)
			if err != nil {
				t.Fatalf("Could not open file: %v", err)
			}
			defer f.Close()

			result, err := DecodeWithDiagnostics(f, nil)

			// In lenient mode, we should get a result even with malformed files
			// (unless the file is completely unparseable)
			if result == nil && err == nil {
				t.Error("Expected either result or error")
			}

			if result != nil {
				t.Logf("File %s: %d records, %d diagnostics, error=%v",
					entry.Name(),
					len(result.Document.Records),
					len(result.Diagnostics),
					err)

				// Log diagnostics for debugging
				for _, diag := range result.Diagnostics {
					t.Logf("  Diagnostic: %s", diag.String())
				}
			}
		})
	}
}

// TestDecodeWithDiagnostics_DiagnosticsHelpersWithRealData tests helper methods
func TestDecodeWithDiagnostics_DiagnosticsHelpersWithRealData(t *testing.T) {
	// Input with both parser errors and entity warnings
	input := `0 HEAD
1 GEDC
2 VERS 5.5

0 @I1@ INDI
1 NAME John /Smith/
1 UNKNOWNTAG custom value
INVALID LINE
0 TRLR`

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}

	// Should have diagnostics
	if len(result.Diagnostics) == 0 {
		t.Fatal("Expected diagnostics")
	}

	// Test HasErrors - should be true (parser error is SeverityError)
	if !result.Diagnostics.HasErrors() {
		t.Error("HasErrors() should return true")
	}

	// Test Errors() - should return error-level diagnostics
	errors := result.Diagnostics.Errors()
	if len(errors) == 0 {
		t.Error("Errors() should return at least one error")
	}
	for _, e := range errors {
		if e.Severity != SeverityError {
			t.Errorf("Errors() returned non-error: %v", e.Severity)
		}
	}

	// Test Warnings() - should return warning-level diagnostics (unknown tag)
	warnings := result.Diagnostics.Warnings()
	// Unknown tags generate warnings
	for _, w := range warnings {
		if w.Severity != SeverityWarning {
			t.Errorf("Warnings() returned non-warning: %v", w.Severity)
		}
	}

	// Test String() output
	output := result.Diagnostics.String()
	if !strings.Contains(output, "diagnostic(s)") {
		t.Errorf("String() should contain 'diagnostic(s)', got: %s", output)
	}

	t.Logf("Diagnostics output:\n%s", output)
}

// TestDecodeWithDiagnostics_LevelJumpRecovery_SingleSubordinateSkip verifies
// that a single subordinate tag with a level jump (1 -> 4) is recovered: a
// CodeBadLevelJump diagnostic is emitted and the DATE value lands on the BIRT
// event rather than being silently dropped.
func TestDecodeWithDiagnostics_LevelJumpRecovery_SingleSubordinateSkip(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
4 DATE 1 JAN 1900
0 TRLR`

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}

	// One BAD_LEVEL_JUMP for the 4 DATE line.
	jumps := 0
	for _, d := range result.Diagnostics {
		if d.Code == CodeBadLevelJump {
			jumps++
		}
	}
	if jumps != 1 {
		t.Errorf("expected 1 BAD_LEVEL_JUMP diagnostic, got %d (all: %v)", jumps, result.Diagnostics)
	}

	indi := result.Document.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("GetIndividual(@I1@) returned nil")
	}
	if len(indi.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(indi.Events))
	}
	birt := indi.Events[0]
	if birt.Date != "1 JAN 1900" {
		t.Errorf("BIRT.Date = %q, want %q (DATE was orphaned by level jump)", birt.Date, "1 JAN 1900")
	}
}

// TestDecodeWithDiagnostics_LevelJumpRecovery_MidRecordSubordinateSkip verifies
// the trickier case where a subordinate tag (PLAC) jumps from level 1 to 3
// mid-record. The clamped PLAC must attach to the immediately preceding level-1
// event (DEAT), not to the prior event (BIRT) or be silently dropped.
func TestDecodeWithDiagnostics_LevelJumpRecovery_MidRecordSubordinateSkip(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Jane /Doe/
1 BIRT
2 DATE 1 JAN 1900
1 DEAT
3 PLAC London, England
0 TRLR`

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}

	jumps := 0
	for _, d := range result.Diagnostics {
		if d.Code == CodeBadLevelJump {
			jumps++
		}
	}
	if jumps != 1 {
		t.Errorf("expected 1 BAD_LEVEL_JUMP diagnostic, got %d (all: %v)", jumps, result.Diagnostics)
	}

	indi := result.Document.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("GetIndividual(@I1@) returned nil")
	}
	if len(indi.Events) != 2 {
		t.Fatalf("expected 2 events (BIRT + DEAT), got %d", len(indi.Events))
	}

	var birt, deat *gedcom.Event
	for _, e := range indi.Events {
		switch e.Type {
		case gedcom.EventBirth:
			birt = e
		case gedcom.EventDeath:
			deat = e
		}
	}
	if birt == nil || deat == nil {
		t.Fatalf("missing BIRT or DEAT event; got events: %+v", indi.Events)
	}

	// PLAC must attach to DEAT (its preceding level-1 sibling), not BIRT.
	if deat.Place != "London, England" {
		t.Errorf("DEAT.Place = %q, want %q (PLAC was attached to the wrong event after clamping)",
			deat.Place, "London, England")
	}
	if birt.Place != "" {
		t.Errorf("BIRT.Place = %q, expected empty (PLAC should not have leaked to BIRT)", birt.Place)
	}
	// BIRT.Date should still parse normally.
	if birt.Date != "1 JAN 1900" {
		t.Errorf("BIRT.Date = %q, want %q", birt.Date, "1 JAN 1900")
	}
}
