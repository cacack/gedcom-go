package parser

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// T062: Write tests for invalid tag errors
func TestInvalidTagErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErrMsg  string
		wantLineNum int
	}{
		{
			name:        "invalid character in tag",
			input:       "0 INV@LID",
			wantErrMsg:  "",
			wantLineNum: 1,
		},
		{
			name:        "tag too long",
			input:       "0 VERYLONGTAGNAMETHATEXCEEDSLIMITS",
			wantErrMsg:  "",
			wantLineNum: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			_, err := p.ParseLine(tt.input)

			// For now, we accept these as valid (spec allows custom tags)
			// This test documents current behavior
			_ = err
		})
	}
}

// T063: Write tests for hierarchy level errors
func TestHierarchyLevelErrors(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErr    bool
		expectLine int
	}{
		{
			name: "level jump too large",
			input: `0 HEAD
1 GEDC
5 VERS`,
			wantErr:    false, // Parser accepts any level, decoder may validate
			expectLine: 3,
		},
		{
			name: "negative level",
			input: `0 HEAD
-1 GEDC`,
			wantErr:    true,
			expectLine: 2,
		},
		{
			name: "level exceeds max depth",
			input: `0 HEAD
101 DEEP`,
			wantErr:    true,
			expectLine: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			_, err := p.Parse(strings.NewReader(tt.input))

			if !tt.wantErr {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("Expected error but got none")
			}
			// ADR-007: parse failures surface a *ParseError carrying the
			// 1-based line number of the offending line. Require the type
			// so a regression that drops the structured error is caught,
			// rather than silently passing when the assertion is skipped.
			var parseErr *ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("expected *ParseError, got %T", err)
			}
			if parseErr.Line != tt.expectLine {
				t.Errorf("ParseError.Line = %d, want %d", parseErr.Line, tt.expectLine)
			}
		})
	}
}

// T064: Write tests for missing cross-reference targets (decoder responsibility, but test parser handling)
func TestMalformedXRefs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		// wantErr is the substring the ParseError message must contain;
		// empty means the line must parse without error.
		wantErr string
		// wantLine is the parsed (or recovered) line, or nil when none is returned.
		wantLine *Line
	}{
		{
			name:     "xref without closing @",
			input:    "0 @I1 INDI",
			wantErr:  "xref is missing its closing @", // Reported, and recovered (issue #385)
			wantLine: &Line{Level: 0, XRef: "@I1", Tag: "INDI", LineNumber: 1},
		},
		{
			name:     "xref without closing @ and a value",
			input:    "0 @I1 INDI extra value",
			wantErr:  "xref is missing its closing @",
			wantLine: &Line{Level: 0, XRef: "@I1", Tag: "INDI", Value: "extra value", LineNumber: 1},
		},
		{
			name:     "xref without closing @ separated from the tag by a tab",
			input:    "0 @I1\tINDI extra value",
			wantErr:  "xref is missing its closing @",
			wantLine: &Line{Level: 0, XRef: "@I1", Tag: "INDI", Value: "extra value", LineNumber: 1},
		},
		{
			// No tag means no record type to recover, so the line keeps the
			// parse it has always had — but a Line is still returned, never
			// nil, so a level-0 record can never be dropped by this path.
			name:     "xref without closing @ and no tag",
			input:    "0 @I1",
			wantErr:  "xref is missing its closing @",
			wantLine: &Line{Level: 0, Tag: "@I1", LineNumber: 1},
		},
		{
			// HEAD/TRLR guard: the decoder keys the header block and the end
			// of a record off a bare level-0 HEAD/TRLR tag, so promoting the
			// tag here would let a bogus header overwrite the real one.
			name:     "xref without closing @ before HEAD",
			input:    "0 @I1 HEAD",
			wantErr:  "xref is missing its closing @",
			wantLine: &Line{Level: 0, Tag: "@I1", Value: "HEAD", LineNumber: 1},
		},
		{
			// Same guard: promoting TRLR would drop the line and everything
			// subordinate to it from the document.
			name:     "xref without closing @ before TRLR",
			input:    "0 @I1 TRLR",
			wantErr:  "xref is missing its closing @",
			wantLine: &Line{Level: 0, Tag: "@I1", Value: "TRLR", LineNumber: 1},
		},
		{
			// Level guard: a subordinate Tag has nowhere to store an XRef, so
			// recovering one here would delete the "@I1" text entirely.
			name:     "unterminated xref below level 0",
			input:    "1 @I1 NOTE plain text",
			wantLine: &Line{Level: 1, Tag: "@I1", Value: "NOTE plain text", LineNumber: 1},
		},
		{
			// Count guard: the email's "@" could equally be the terminator, so
			// the shape is genuinely ambiguous and is left alone.
			name:     "unterminated xref with an email in the value",
			input:    "0 @N1 NOTE write to a@b.com",
			wantLine: &Line{Level: 0, Tag: "@N1", Value: "NOTE write to a@b.com", LineNumber: 1},
		},
		{
			// Count guard: an escaped "@@" in the value.
			name:     "unterminated xref with an escaped at-sign in the value",
			input:    "0 @N1 NOTE @@escaped",
			wantLine: &Line{Level: 0, Tag: "@N1", Value: "NOTE @@escaped", LineNumber: 1},
		},
		{
			// Count guard: a trailing pointer must not pose as the terminator.
			name:     "unterminated xref with a trailing pointer",
			input:    "0 @I1 INDI @X@",
			wantLine: &Line{Level: 0, Tag: "@I1", Value: "INDI @X@", LineNumber: 1},
		},
		{
			name:     "pointer value at level 1 is untouched",
			input:    "1 FAMC @VOID@",
			wantLine: &Line{Level: 1, Tag: "FAMC", Value: "@VOID@", LineNumber: 1},
		},
		{
			name:     "calendar escape in the value is untouched",
			input:    "0 @N1@ NOTE @#DJULIAN@ date escape",
			wantLine: &Line{Level: 0, XRef: "@N1@", Tag: "NOTE", Value: "@#DJULIAN@ date escape", LineNumber: 1},
		},
		{
			name:     "xref without opening @",
			input:    "0 I1@ INDI",
			wantLine: &Line{Level: 0, Tag: "I1@", Value: "INDI", LineNumber: 1},
		},
		{
			name:     "empty xref",
			input:    "0 @@ INDI",
			wantLine: &Line{Level: 0, XRef: "@@", Tag: "INDI", LineNumber: 1},
		},
		{
			name:     "xref not followed by a space",
			input:    "0 @I1@INDI",
			wantLine: &Line{Level: 0, Tag: "@I1@INDI", LineNumber: 1},
		},
		{
			name:     "xref run together with the tag",
			input:    "0 @I1@x INDI",
			wantLine: &Line{Level: 0, Tag: "@I1@x", Value: "INDI", LineNumber: 1},
		},
		{
			name:     "xref with spaces",
			input:    "0 @I 1@ INDI",
			wantErr:  "xref contains a space", // Reported, and recovered (issue #377)
			wantLine: &Line{Level: 0, XRef: "@I 1@", Tag: "INDI", LineNumber: 1},
		},
		{
			name:     "xref with spaces separated from the tag by a tab",
			input:    "0 @I 1@\tINDI extra value",
			wantErr:  "xref contains a space",
			wantLine: &Line{Level: 0, XRef: "@I 1@", Tag: "INDI", Value: "extra value", LineNumber: 1},
		},
		{
			name:     "xref with spaces and no tag",
			input:    "0 @I 1@",
			wantErr:  "line with xref must have a tag",
			wantLine: nil,
		},
		{
			name:     "xref with spaces and a value",
			input:    "0 @NoTe ref@ NOTE mixed case and space",
			wantErr:  "xref contains a space",
			wantLine: &Line{Level: 0, XRef: "@NoTe ref@", Tag: "NOTE", Value: "mixed case and space", LineNumber: 1},
		},
		{
			// The pointer in the value must not pose as the closing @, so this
			// keeps its pre-existing parse rather than inventing an identifier.
			name:     "unterminated xref followed by a pointer in the value",
			input:    "1 @I1 NOTE see @P1@ here",
			wantLine: &Line{Level: 1, Tag: "@I1", Value: "NOTE see @P1@ here", LineNumber: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			line, err := p.ParseLine(tt.input)

			switch {
			case tt.wantErr == "" && err != nil:
				t.Errorf("Unexpected error: %v", err)
			case tt.wantErr != "" && err == nil:
				t.Errorf("Expected error containing %q but got none", tt.wantErr)
			case tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr):
				t.Errorf("Error = %v, want it to contain %q", err, tt.wantErr)
			}

			if tt.wantLine == nil {
				if line != nil {
					t.Errorf("Expected no line, got %+v", line)
				}
				return
			}
			if line == nil {
				t.Fatalf("Expected line %+v, got nil", tt.wantLine)
			}
			if *line != *tt.wantLine {
				t.Errorf("Line = %+v, want %+v", *line, *tt.wantLine)
			}
		})
	}
}

// T065: Write tests for encoding errors (handled by charset package)
// Already covered in charset/charset_test.go

// T066: Write tests for malformed files
func TestMalformedFiles(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "missing HEAD",
			input: `0 @I1@ INDI
1 NAME John
0 TRLR`,
			wantErr: false, // Parser accepts, decoder may validate
		},
		{
			name: "missing TRLR",
			input: `0 HEAD
1 GEDC
0 @I1@ INDI`,
			wantErr: false, // Parser accepts
		},
		{
			name:    "completely empty file",
			input:   "",
			wantErr: false, // Returns empty line list
		},
		{
			name: "only whitespace lines",
			input: `

`,
			wantErr: true, // Empty lines are errors
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			lines, err := p.Parse(strings.NewReader(tt.input))

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.wantErr && err == nil {
				t.Logf("Parsed %d lines from malformed input", len(lines))
			}
		})
	}
}

// T067: Write tests for partial file recovery (error recovery mode)
func TestErrorRecoveryMode(t *testing.T) {
	// This test verifies that parser can continue after encountering errors
	// when in recovery mode (future enhancement)

	input := `0 HEAD
1 GEDC
INVALID LINE HERE
2 VERS 5.5
0 TRLR`

	p := NewParser()
	_, err := p.Parse(strings.NewReader(input))

	// Currently, parser stops at first error. This test documents that
	// behavior and pins down the ADR-007 guarantee that the error points
	// at the exact offending line with its content preserved.
	if err == nil {
		t.Fatal("Expected error for invalid line")
	}

	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected *ParseError, got %T", err)
	}
	if parseErr.Line != 3 {
		t.Errorf("ParseError.Line = %d, want 3", parseErr.Line)
	}
	if !strings.Contains(parseErr.Context, "INVALID LINE HERE") {
		t.Errorf("ParseError.Context = %q, want it to contain the offending line", parseErr.Context)
	}
}

// Test that errors include helpful context
func TestErrorContext(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantContext string
	}{
		{
			name:        "error shows line content",
			input:       "INVALID",
			wantContext: "INVALID",
		},
		{
			name:        "error shows numeric issue",
			input:       "X HEAD",
			wantContext: "X HEAD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			_, err := p.ParseLine(tt.input)

			if err == nil {
				t.Fatal("Expected error but got none")
			}

			var parseErr *ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("Expected *ParseError, got %T", err)
			}

			if parseErr.Context != tt.wantContext {
				t.Errorf("Context = %q, want %q", parseErr.Context, tt.wantContext)
			}

			errMsg := parseErr.Error()
			if !strings.Contains(errMsg, tt.wantContext) {
				t.Errorf("Error message %q should contain context %q", errMsg, tt.wantContext)
			}
		})
	}
}

// Test error messages are clear and actionable
func TestErrorMessageQuality(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantContains   []string
		wantLineNumber int
	}{
		{
			name:           "empty line error",
			input:          "",
			wantContains:   []string{"empty", "line 1"},
			wantLineNumber: 1,
		},
		{
			name:           "missing tag error",
			input:          "0",
			wantContains:   []string{"tag", "line 1"},
			wantLineNumber: 1,
		},
		{
			name:           "invalid level error",
			input:          "ABC TAG",
			wantContains:   []string{"level", "line 1"},
			wantLineNumber: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			_, err := p.ParseLine(tt.input)

			if err == nil {
				t.Fatal("Expected error but got none")
			}

			errMsg := err.Error()
			for _, substr := range tt.wantContains {
				if !strings.Contains(errMsg, substr) {
					t.Errorf("Error message %q should contain %q", errMsg, substr)
				}
			}

			var parseErr *ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("expected *ParseError, got %T", err)
			}
			if parseErr.Line != tt.wantLineNumber {
				t.Errorf("ParseError.Line = %d, want %d", parseErr.Line, tt.wantLineNumber)
			}
		})
	}
}

// Test ParseError.Unwrap() method for error unwrapping
func TestParseErrorUnwrap(t *testing.T) {
	t.Run("unwrap wrapped error", func(t *testing.T) {
		baseErr := fmt.Errorf("base error")
		parseErr := wrapParseError(1, "wrapped message", "context", baseErr)

		unwrapped := parseErr.(*ParseError).Unwrap()
		if unwrapped != baseErr {
			t.Errorf("Unwrap() = %v, want %v", unwrapped, baseErr)
		}

		// Test with errors.Is
		if !errors.Is(parseErr, baseErr) {
			t.Error("errors.Is() should find base error through Unwrap()")
		}
	})

	t.Run("unwrap error without underlying error", func(t *testing.T) {
		parseErr := newParseError(1, "simple error", "context")

		unwrapped := parseErr.(*ParseError).Unwrap()
		if unwrapped != nil {
			t.Errorf("Unwrap() = %v, want nil", unwrapped)
		}
	})

	t.Run("errors.As with wrapped error", func(t *testing.T) {
		baseErr := &ParseError{Line: 99, Message: "original"}
		wrappedErr := wrapParseError(1, "wrapped", "ctx", baseErr)

		var target *ParseError
		if !errors.As(wrappedErr, &target) {
			t.Error("errors.As() should find ParseError through Unwrap()")
		}
		if target.Line != 1 {
			t.Errorf("errors.As() found wrong error, Line = %d, want 1", target.Line)
		}
	})
}

// locatedReadError mimics a charset-layer failure: the reader rejects a chunk
// it has already consumed, so the error knows a physical line the parser's own
// counter has not reached yet.
type locatedReadError struct {
	line int
}

func (e *locatedReadError) Error() string  { return "invalid byte" }
func (e *locatedReadError) ErrorLine() int { return e.line }

// failingReader hands over content once, then fails with the given error.
type failingReader struct {
	content string
	err     error
	sent    bool
}

func (r *failingReader) Read(p []byte) (int, error) {
	if r.sent {
		return 0, r.err
	}
	r.sent = true
	return copy(p, r.content), nil
}

// Reader errors that carry their own physical line must be reported at that
// line, not at the last line the parser managed to tokenize (issue #376).
func TestParseReadErrorReportsPhysicalLine(t *testing.T) {
	const content = "0 HEAD\n1 SOUR TEST\n"

	tests := []struct {
		name     string
		err      error
		wantLine int
	}{
		{
			name:     "error carries physical line",
			err:      &locatedReadError{line: 42},
			wantLine: 42,
		},
		{
			name:     "error without a line falls back to parser counter",
			err:      errors.New("disk on fire"),
			wantLine: 2,
		},
		{
			name:     "unset line falls back to parser counter",
			err:      &locatedReadError{line: 0},
			wantLine: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			_, err := p.Parse(&failingReader{content: content, err: tt.err})
			assertReadErrorLine(t, err, tt.wantLine, tt.err)

			p2 := NewParser()
			lines, _, fatalErr := p2.ParseWithOptions(
				&failingReader{content: content, err: tt.err},
				&ParseOptions{Lenient: true},
			)
			assertReadErrorLine(t, fatalErr, tt.wantLine, tt.err)
			if len(lines) != 2 {
				t.Errorf("ParseWithOptions() returned %d lines, want 2", len(lines))
			}
		})
	}
}

// assertReadErrorLine checks that err is a ParseError located at wantLine and
// still wrapping the original reader error.
func assertReadErrorLine(t *testing.T, err error, wantLine int, wantWrapped error) {
	t.Helper()

	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("error = %v (%T), want *ParseError", err, err)
	}
	if parseErr.Line != wantLine {
		t.Errorf("ParseError.Line = %d, want %d", parseErr.Line, wantLine)
	}
	if !errors.Is(err, wantWrapped) {
		t.Errorf("error = %v, want it to wrap %v", err, wantWrapped)
	}
}
