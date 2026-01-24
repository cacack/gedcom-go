package decoder

import (
	"testing"
)

// ============================================================================
// Severity Tests
// ============================================================================

func TestSeverity_String(t *testing.T) {
	tests := []struct {
		name     string
		severity Severity
		want     string
	}{
		{
			name:     "error severity",
			severity: SeverityError,
			want:     "ERROR",
		},
		{
			name:     "warning severity",
			severity: SeverityWarning,
			want:     "WARNING",
		},
		{
			name:     "info severity",
			severity: SeverityInfo,
			want:     "INFO",
		},
		{
			name:     "unknown severity",
			severity: Severity(99),
			want:     "UNKNOWN(99)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.severity.String()
			if got != tt.want {
				t.Errorf("Severity.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ============================================================================
// Diagnostic Tests
// ============================================================================

func TestDiagnostic_String(t *testing.T) {
	tests := []struct {
		name string
		diag Diagnostic
		want string
	}{
		{
			name: "error with context",
			diag: Diagnostic{
				Line:     10,
				Severity: SeverityError,
				Code:     CodeSyntaxError,
				Message:  "invalid line format",
				Context:  "BAD_LINE",
			},
			want: `[ERROR] line 10: SYNTAX_ERROR: invalid line format (context: "BAD_LINE")`,
		},
		{
			name: "warning without context",
			diag: Diagnostic{
				Line:     5,
				Severity: SeverityWarning,
				Code:     CodeUnknownTag,
				Message:  "unknown tag CUSTOMTAG",
				Context:  "",
			},
			want: "[WARNING] line 5: UNKNOWN_TAG: unknown tag CUSTOMTAG",
		},
		{
			name: "info diagnostic",
			diag: Diagnostic{
				Line:     1,
				Severity: SeverityInfo,
				Code:     "SUGGESTION",
				Message:  "consider using standard format",
				Context:  "",
			},
			want: "[INFO] line 1: SUGGESTION: consider using standard format",
		},
		{
			name: "empty line diagnostic",
			diag: Diagnostic{
				Line:     20,
				Severity: SeverityError,
				Code:     CodeEmptyLine,
				Message:  "empty line",
				Context:  "",
			},
			want: "[ERROR] line 20: EMPTY_LINE: empty line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.diag.String()
			if got != tt.want {
				t.Errorf("Diagnostic.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDiagnostic_Error(t *testing.T) {
	diag := Diagnostic{
		Line:     42,
		Severity: SeverityError,
		Code:     CodeInvalidLevel,
		Message:  "invalid level number",
		Context:  "XYZ TAG",
	}

	// Error() should return the same as String()
	if diag.Error() != diag.String() {
		t.Errorf("Diagnostic.Error() = %q, want %q", diag.Error(), diag.String())
	}

	// Verify the diagnostic satisfies the error interface
	var err error = diag
	if err.Error() != diag.String() {
		t.Errorf("error interface Error() = %q, want %q", err.Error(), diag.String())
	}
}

func TestNewDiagnostic(t *testing.T) {
	diag := NewDiagnostic(15, SeverityWarning, CodeUnknownTag, "unknown tag TEST", "1 TEST value")

	if diag.Line != 15 {
		t.Errorf("Line = %d, want 15", diag.Line)
	}
	if diag.Severity != SeverityWarning {
		t.Errorf("Severity = %v, want SeverityWarning", diag.Severity)
	}
	if diag.Code != CodeUnknownTag {
		t.Errorf("Code = %q, want %q", diag.Code, CodeUnknownTag)
	}
	if diag.Message != "unknown tag TEST" {
		t.Errorf("Message = %q, want %q", diag.Message, "unknown tag TEST")
	}
	if diag.Context != "1 TEST value" {
		t.Errorf("Context = %q, want %q", diag.Context, "1 TEST value")
	}
}

func TestNewParseError(t *testing.T) {
	diag := NewParseError(7, CodeEmptyLine, "empty line", "")

	if diag.Line != 7 {
		t.Errorf("Line = %d, want 7", diag.Line)
	}
	if diag.Severity != SeverityError {
		t.Errorf("Severity = %v, want SeverityError", diag.Severity)
	}
	if diag.Code != CodeEmptyLine {
		t.Errorf("Code = %q, want %q", diag.Code, CodeEmptyLine)
	}
}

// ============================================================================
// Diagnostics Collection Tests
// ============================================================================

func TestDiagnostics_HasErrors(t *testing.T) {
	tests := []struct {
		name string
		ds   Diagnostics
		want bool
	}{
		{
			name: "empty diagnostics",
			ds:   Diagnostics{},
			want: false,
		},
		{
			name: "nil diagnostics",
			ds:   nil,
			want: false,
		},
		{
			name: "only warnings",
			ds: Diagnostics{
				{Severity: SeverityWarning},
				{Severity: SeverityWarning},
			},
			want: false,
		},
		{
			name: "only info",
			ds: Diagnostics{
				{Severity: SeverityInfo},
			},
			want: false,
		},
		{
			name: "single error",
			ds: Diagnostics{
				{Severity: SeverityError},
			},
			want: true,
		},
		{
			name: "mixed with errors",
			ds: Diagnostics{
				{Severity: SeverityWarning},
				{Severity: SeverityError},
				{Severity: SeverityInfo},
			},
			want: true,
		},
		{
			name: "all errors",
			ds: Diagnostics{
				{Severity: SeverityError},
				{Severity: SeverityError},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ds.HasErrors()
			if got != tt.want {
				t.Errorf("HasErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDiagnostics_Errors(t *testing.T) {
	tests := []struct {
		name      string
		ds        Diagnostics
		wantCount int
	}{
		{
			name:      "empty diagnostics",
			ds:        Diagnostics{},
			wantCount: 0,
		},
		{
			name:      "nil diagnostics",
			ds:        nil,
			wantCount: 0,
		},
		{
			name: "only warnings",
			ds: Diagnostics{
				{Severity: SeverityWarning},
				{Severity: SeverityWarning},
			},
			wantCount: 0,
		},
		{
			name: "mixed severities",
			ds: Diagnostics{
				{Severity: SeverityWarning, Code: "W1"},
				{Severity: SeverityError, Code: "E1"},
				{Severity: SeverityInfo, Code: "I1"},
				{Severity: SeverityError, Code: "E2"},
			},
			wantCount: 2,
		},
		{
			name: "all errors",
			ds: Diagnostics{
				{Severity: SeverityError},
				{Severity: SeverityError},
				{Severity: SeverityError},
			},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ds.Errors()
			if len(got) != tt.wantCount {
				t.Errorf("Errors() returned %d items, want %d", len(got), tt.wantCount)
			}

			// Verify all returned items are errors
			for _, d := range got {
				if d.Severity != SeverityError {
					t.Errorf("Errors() returned non-error: %v", d.Severity)
				}
			}
		})
	}
}

func TestDiagnostics_Warnings(t *testing.T) {
	tests := []struct {
		name      string
		ds        Diagnostics
		wantCount int
	}{
		{
			name:      "empty diagnostics",
			ds:        Diagnostics{},
			wantCount: 0,
		},
		{
			name:      "nil diagnostics",
			ds:        nil,
			wantCount: 0,
		},
		{
			name: "only errors",
			ds: Diagnostics{
				{Severity: SeverityError},
				{Severity: SeverityError},
			},
			wantCount: 0,
		},
		{
			name: "mixed severities",
			ds: Diagnostics{
				{Severity: SeverityWarning, Code: "W1"},
				{Severity: SeverityError, Code: "E1"},
				{Severity: SeverityInfo, Code: "I1"},
				{Severity: SeverityWarning, Code: "W2"},
			},
			wantCount: 2,
		},
		{
			name: "all warnings",
			ds: Diagnostics{
				{Severity: SeverityWarning},
				{Severity: SeverityWarning},
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ds.Warnings()
			if len(got) != tt.wantCount {
				t.Errorf("Warnings() returned %d items, want %d", len(got), tt.wantCount)
			}

			// Verify all returned items are warnings
			for _, d := range got {
				if d.Severity != SeverityWarning {
					t.Errorf("Warnings() returned non-warning: %v", d.Severity)
				}
			}
		})
	}
}

func TestDiagnostics_String(t *testing.T) {
	tests := []struct {
		name string
		ds   Diagnostics
		want string
	}{
		{
			name: "empty diagnostics",
			ds:   Diagnostics{},
			want: "no diagnostics",
		},
		{
			name: "nil diagnostics",
			ds:   nil,
			want: "no diagnostics",
		},
		{
			name: "single diagnostic",
			ds: Diagnostics{
				{
					Line:     5,
					Severity: SeverityError,
					Code:     CodeSyntaxError,
					Message:  "test error",
					Context:  "",
				},
			},
			want: "1 diagnostic(s):\n  [ERROR] line 5: SYNTAX_ERROR: test error\n",
		},
		{
			name: "multiple diagnostics",
			ds: Diagnostics{
				{
					Line:     1,
					Severity: SeverityError,
					Code:     CodeEmptyLine,
					Message:  "empty line",
					Context:  "",
				},
				{
					Line:     2,
					Severity: SeverityWarning,
					Code:     CodeUnknownTag,
					Message:  "unknown tag",
					Context:  "",
				},
			},
			want: "2 diagnostic(s):\n  [ERROR] line 1: EMPTY_LINE: empty line\n  [WARNING] line 2: UNKNOWN_TAG: unknown tag\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ds.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ============================================================================
// Diagnostic Code Constants Tests
// ============================================================================

func TestDiagnosticCodes(t *testing.T) {
	// Verify parse-phase codes exist and are non-empty
	parseCodes := []string{
		CodeSyntaxError,
		CodeInvalidLevel,
		CodeInvalidXRef,
		CodeBadLevelJump,
		CodeEmptyLine,
	}

	for _, code := range parseCodes {
		if code == "" {
			t.Error("Parse-phase diagnostic code should not be empty")
		}
	}

	// Verify entity-level codes exist and are non-empty
	entityCodes := []string{
		CodeUnknownTag,
		CodeInvalidValue,
		CodeMissingRequired,
		CodeSkippedRecord,
	}

	for _, code := range entityCodes {
		if code == "" {
			t.Error("Entity-level diagnostic code should not be empty")
		}
	}
}
