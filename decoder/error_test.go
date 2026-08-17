package decoder

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/charset"
	"github.com/cacack/gedcom-go/v2/parser"
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

// Test that decoder surfaces the parser's structured *ParseError so
// callers get ADR-007's guarantees — a line number, the offending
// content, and the underlying cause — rather than an opaque string.
func TestDecoderErrorMessages(t *testing.T) {
	tests := []struct {
		name string
		// input is the raw GEDCOM to decode.
		input string
		// wantErrSubstring must appear in the rendered error message.
		wantErrSubstring string
		// wantLine is the ParseError.Line the failure should report.
		wantLine int
		// wantContextSubstr, when non-empty, must appear in
		// ParseError.Context (the preserved offending line content).
		wantContextSubstr string
		// wantWrapped asserts an underlying cause is reachable via
		// Unwrap (e.g. the charset decode error behind a read failure).
		wantWrapped bool
	}{
		{
			// The invalid bytes sit on physical line 2. The parser's own
			// counter reaches only line 1 — the charset reader delivers the
			// valid prefix and then stops — so the reported location has to
			// come from the charset error (issue #376).
			name:             "invalid UTF-8",
			input:            "0 HEAD\n1 NAME \xFF\xFE Invalid UTF-8\n0 TRLR",
			wantErrSubstring: "reading input",
			wantLine:         2,
			wantWrapped:      true,
		},
		{
			// Same failure over CRLF: the charset reader must not count the
			// CR and the LF as two separate line breaks.
			name:             "invalid UTF-8 with CRLF line endings",
			input:            "0 HEAD\r\n1 NAME \xFF\xFE Invalid UTF-8\r\n0 TRLR",
			wantErrSubstring: "reading input",
			wantLine:         2,
			wantWrapped:      true,
		},
		{
			// Same failure over bare CR (old Macintosh), which the parser
			// splits on but a naive LF-only counter would miss entirely.
			name:             "invalid UTF-8 with CR line endings",
			input:            "0 HEAD\r1 NAME \xFF\xFE Invalid UTF-8\r0 TRLR",
			wantErrSubstring: "reading input",
			wantLine:         2,
			wantWrapped:      true,
		},
		{
			name:              "completely invalid format",
			input:             "This is not GEDCOM at all!",
			wantErrSubstring:  "level",
			wantLine:          1,
			wantContextSubstr: "This is not GEDCOM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decode(strings.NewReader(tt.input))
			if err == nil {
				t.Fatal("Expected error but got none")
			}

			var parseErr *parser.ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("expected *parser.ParseError, got %T (%v)", err, err)
			}
			if parseErr.Line != tt.wantLine {
				t.Errorf("ParseError.Line = %d, want %d", parseErr.Line, tt.wantLine)
			}
			if !strings.Contains(err.Error(), tt.wantErrSubstring) {
				t.Errorf("error message %q should contain %q", err.Error(), tt.wantErrSubstring)
			}
			if tt.wantContextSubstr != "" && !strings.Contains(parseErr.Context, tt.wantContextSubstr) {
				t.Errorf("ParseError.Context = %q, want it to contain %q", parseErr.Context, tt.wantContextSubstr)
			}
			if tt.wantWrapped && parseErr.Unwrap() == nil {
				t.Error("expected ParseError to wrap an underlying cause (ADR-007), got nil")
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

// End-to-end guard for both halves of issue #382. The charset reader delivers
// the bytes ahead of the offending one, so the lenient partial document
// reaches the last complete line before it; the parser drops the truncated
// tail those bytes end in, so nothing shortened enters the document and the
// reader error is what gets reported.
//
// Deliberately independent of any buffer size: the assertion is "every
// complete line before the bad byte survives", which holds at whatever offset
// the chunk boundary happens to fall.
func TestDecodePartialRecoveryUpToInvalidByte(t *testing.T) {
	// Four complete lines, then a fifth cut short by an invalid byte.
	const input = "0 HEAD\n" +
		"1 SOUR TEST\n" +
		"0 @I1@ INDI\n" +
		"1 NAME John /Smith/\n" +
		"1 NOTE Fr\xFF"

	result, err := DecodeWithDiagnostics(strings.NewReader(input), DefaultOptions())
	if err == nil {
		t.Fatal("DecodeWithDiagnostics() error = nil, want the read failure")
	}

	var utf8Err *charset.ErrInvalidUTF8
	if !errors.As(err, &utf8Err) {
		t.Fatalf("error = %v (%T), want a wrapped *charset.ErrInvalidUTF8", err, err)
	}
	if utf8Err.Line != 5 || utf8Err.Column != 10 {
		t.Errorf("error at line %d, column %d; want line 5, column 10", utf8Err.Line, utf8Err.Column)
	}
	if strings.Contains(err.Error(), "at least level and tag") {
		t.Errorf("error = %v, want the read failure, not a syntax error about the truncated line", err)
	}

	if result == nil || result.Document == nil {
		t.Fatal("DecodeWithDiagnostics() returned no document; want the lines ahead of the bad byte")
	}
	indis := result.Document.Individuals()
	if len(indis) != 1 {
		t.Fatalf("partial document has %d individuals, want 1", len(indis))
	}
	if len(indis[0].Names) != 1 || indis[0].Names[0].Full != "John /Smith/" {
		t.Errorf("individual names = %+v, want the complete NAME line", indis[0].Names)
	}
	// The truncated NOTE is dropped rather than kept as "Fr".
	if got := len(indis[0].Notes); got != 0 {
		t.Errorf("individual has %d notes, want 0 (the truncated line must not be kept)", got)
	}
	for _, d := range result.Diagnostics {
		if strings.Contains(d.Message, "at least level and tag") {
			t.Errorf("diagnostic %+v reports a syntax error the input never contained", d)
		}
	}
}

// The no-partial-data branch stays reachable: when the very first byte is
// invalid there is nothing to recover, so the result is nil and the read
// failure is returned on its own.
func TestDecodeInvalidFirstByteHasNoPartialDocument(t *testing.T) {
	result, err := DecodeWithDiagnostics(strings.NewReader("\xFF0 HEAD\n0 TRLR\n"), DefaultOptions())
	if err == nil {
		t.Fatal("DecodeWithDiagnostics() error = nil, want the read failure")
	}
	if result != nil {
		t.Errorf("DecodeWithDiagnostics() result = %+v, want nil when no line could be parsed", result)
	}
	var utf8Err *charset.ErrInvalidUTF8
	if !errors.As(err, &utf8Err) {
		t.Fatalf("error = %v (%T), want a wrapped *charset.ErrInvalidUTF8", err, err)
	}
}

// An invalid ANSEL byte must be reported at its own physical line, end to end
// through charset -> parser -> decoder. This is the ADR 0007 ErrorLine()
// contract: without it the parser falls back to its own counter, which cannot
// reach the line the bad byte is on -- nothing after that byte arrives, and the
// truncated head of its line is dropped rather than tokenized (#382). So the
// fallback lands one line short, and reports the impossible "line 0" when the
// bad byte is the first in the file. Both are pinned here (issue #403).
func TestANSELErrorReportsPhysicalLine(t *testing.T) {
	separators := []struct {
		name string
		sep  string
	}{
		{"LF", "\n"},
		{"CRLF", "\r\n"},
		{"CR", "\r"},
	}

	tests := []struct {
		name     string
		lastLine string
		wantLine int
	}{
		{"bad byte in column 1", "\x80 NAME x", 5},
		{"bad byte mid-line", "1 NAME \x80x", 5},
	}

	for _, sepCase := range separators {
		for _, tt := range tests {
			t.Run(sepCase.name+"/"+tt.name, func(t *testing.T) {
				sep := sepCase.sep
				input := "0 HEAD" + sep + "1 CHAR ANSEL" + sep + "1 GEDC" + sep +
					"2 VERS 5.5.1" + sep + tt.lastLine + sep + "0 TRLR" + sep

				_, err := Decode(strings.NewReader(input))

				var parseErr *parser.ParseError
				if !errors.As(err, &parseErr) {
					t.Fatalf("Decode() error = %v (%T), want *parser.ParseError", err, err)
				}
				if parseErr.Line != tt.wantLine {
					t.Errorf("ParseError.Line = %d, want %d", parseErr.Line, tt.wantLine)
				}
			})
		}
	}

	// A bad first byte reported "line 0" before #403.
	t.Run("bad byte at offset 0", func(t *testing.T) {
		_, err := Decode(strings.NewReader("\x800 HEAD\n1 CHAR ANSEL\n0 TRLR\n"))

		var parseErr *parser.ParseError
		if !errors.As(err, &parseErr) {
			t.Fatalf("Decode() error = %v (%T), want *parser.ParseError", err, err)
		}
		if parseErr.Line != 1 {
			t.Errorf("ParseError.Line = %d, want 1", parseErr.Line)
		}
	})
}
