package decoder

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/charset"
	"github.com/cacack/gedcom-go/v3/parser"
)

// invalidUTF8Probe builds the #710 probe: 100 individuals, with one raw CP1252
// 'é' (0xE9) in a note on physical line 25, under the given CHAR line ("" for
// none).
func invalidUTF8Probe(charLine string) (input string, badLine int) {
	var sb strings.Builder
	line := 0
	w := func(s string) {
		sb.WriteString(s)
		sb.WriteString("\n")
		line++
	}
	w("0 HEAD")
	w("1 GEDC")
	w("2 VERS 5.5.1")
	w("2 FORM LINEAGE-LINKED")
	if charLine != "" {
		w(charLine)
	}
	for i := 1; i <= 100; i++ {
		w(fmt.Sprintf("0 @I%d@ INDI", i))
		w(fmt.Sprintf("1 NAME Person%d /Test/", i))
		if badLine == 0 && line >= 23 {
			if line == 23 {
				w("1 SEX U") // keeps the note on line 25 when CHAR adds a line
			}
			w("1 NOTE Caf\xE9 au lait")
			badLine = line
		}
	}
	w("0 TRLR")
	return sb.String(), badLine
}

// TestDecodeLenientReplacesInvalidUTF8 is the #710 acceptance probe. Lenient
// mode used to stop at the bad byte with "error reading input" and return 10
// of the 100 individuals; it now replaces the byte with U+FFFD, decodes all
// 100, and reports exactly one diagnostic at the byte's line and column.
func TestDecodeLenientReplacesInvalidUTF8(t *testing.T) {
	for _, charLine := range []string{"1 CHAR UTF-8", "", "1 CHAR ASCII", "1 CHAR UTF8"} {
		t.Run(fmt.Sprintf("%q", charLine), func(t *testing.T) {
			input, badLine := invalidUTF8Probe(charLine)
			if badLine != 25 {
				t.Fatalf("probe put the bad byte on line %d, want 25", badLine)
			}

			result, err := DecodeWithDiagnostics(strings.NewReader(input), DefaultOptions())
			if err != nil {
				t.Fatalf("DecodeWithDiagnostics() error = %v, want the bad byte recovered", err)
			}
			if got := len(result.Document.Individuals()); got != 100 {
				t.Errorf("Individuals() = %d, want 100", got)
			}

			want := Diagnostic{
				Line:     25,
				Severity: SeverityError,
				Code:     CodeInvalidEncoding,
				Message:  "invalid UTF-8 byte 0xE9 at column 11 replaced with U+FFFD",
			}
			if len(result.Diagnostics) != 1 || result.Diagnostics[0] != want {
				t.Errorf("Diagnostics = %v, want exactly [%v]", result.Diagnostics, want)
			}

			// The line holding the byte is kept, with only the byte replaced.
			var notes []string
			for _, indi := range result.Document.Individuals() {
				notes = append(notes, indi.InlineNotes...)
			}
			if wantNotes := []string{"Caf\uFFFD au lait"}; len(notes) != 1 || notes[0] != wantNotes[0] {
				t.Errorf("inline notes = %q, want %q", notes, wantNotes)
			}

			// Decode and DecodeWithOptions recover the same way.
			doc, err := Decode(strings.NewReader(input))
			if err != nil || len(doc.Individuals()) != 100 {
				t.Errorf("Decode() = %v individuals, error %v; want 100, nil", len(doc.Individuals()), err)
			}
		})
	}
}

// Strict mode is unchanged by #710: the first invalid byte fails the decode
// with no document, naming the byte's line.
func TestDecodeStrictRejectsInvalidUTF8(t *testing.T) {
	input, _ := invalidUTF8Probe("1 CHAR UTF-8")

	result, err := DecodeWithDiagnostics(strings.NewReader(input), &DecodeOptions{StrictMode: true})
	if result != nil {
		t.Errorf("result = %+v, want nil in strict mode", result)
	}
	var parseErr *parser.ParseError
	if !errors.As(err, &parseErr) || parseErr.Line != 25 || !strings.Contains(err.Error(), "error reading input") {
		t.Fatalf("error = %v, want an \"error reading input\" *parser.ParseError at line 25", err)
	}
	var utf8Err *charset.ErrInvalidUTF8
	if !errors.As(err, &utf8Err) || *utf8Err != (charset.ErrInvalidUTF8{Line: 25, Column: 11, Byte: 0xE9}) {
		t.Errorf("error = %v, want a wrapped *charset.ErrInvalidUTF8 at line 25, column 11, byte 0xE9", err)
	}
}

// Each invalid byte gets its own diagnostic, in input order, ahead of the
// parser's: encoding is repaired before lines are parsed.
func TestDecodeLenientInvalidUTF8DiagnosticsPerByte(t *testing.T) {
	const input = "0 HEAD\n" +
		"0 @I1@ INDI\n" +
		"1 NAME Jos\xE9 /Garc\xEDa/\n" +
		"bogus line\n" +
		"1 NOTE \x92quoted\x92\n" +
		"0 TRLR\n"

	result, err := DecodeWithDiagnostics(strings.NewReader(input), DefaultOptions())
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	var got []string
	for _, d := range result.Diagnostics {
		got = append(got, fmt.Sprintf("%d %s %s", d.Line, d.Code, d.Message))
	}
	want := []string{
		"3 INVALID_ENCODING invalid UTF-8 byte 0xE9 at column 11 replaced with U+FFFD",
		"3 INVALID_ENCODING invalid UTF-8 byte 0xED at column 18 replaced with U+FFFD",
		"5 INVALID_ENCODING invalid UTF-8 byte 0x92 at column 8 replaced with U+FFFD",
		"5 INVALID_ENCODING invalid UTF-8 byte 0x92 at column 15 replaced with U+FFFD",
	}
	if len(got) != len(want)+1 {
		t.Fatalf("diagnostics = %q, want %q followed by the parse error for line 4", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("diagnostic %d = %q, want %q", i, got[i], want[i])
		}
	}
	if !strings.HasPrefix(got[4], "4 ") {
		t.Errorf("last diagnostic = %q, want the parse error for line 4", got[4])
	}

	indi := result.Document.GetIndividual("@I1@")
	if indi == nil || len(indi.Names) != 1 || indi.Names[0].Full != "Jos\uFFFD /Garc\uFFFDa/" {
		t.Errorf("individual = %+v, want the NAME kept with both bytes replaced", indi)
	}
}

// Input that is mostly not UTF-8 must not grow the diagnostics with the file:
// past MaxInvalidEncodingDiagnostics the bytes are still replaced, but are
// counted in one closing diagnostic instead of reported one by one.
func TestDecodeLenientInvalidUTF8DiagnosticsCapped(t *testing.T) {
	const extra = 50
	var sb strings.Builder
	sb.WriteString("0 HEAD\n0 @I1@ INDI\n")
	for i := 0; i < MaxInvalidEncodingDiagnostics; i++ {
		sb.WriteString("1 NOTE \xE9\n") // lines 3 .. MaxInvalidEncodingDiagnostics+2
	}
	sb.WriteString("1 NOTE " + strings.Repeat("\xFF", extra) + "\n")
	sb.WriteString("0 TRLR\n")
	overLine := MaxInvalidEncodingDiagnostics + 3

	result, err := DecodeWithDiagnostics(strings.NewReader(sb.String()), DefaultOptions())
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	var enc Diagnostics
	for _, d := range result.Diagnostics {
		if d.Code == CodeInvalidEncoding {
			enc = append(enc, d)
		}
	}
	if len(enc) != MaxInvalidEncodingDiagnostics+1 {
		t.Fatalf("got %d INVALID_ENCODING diagnostics, want %d per-byte plus one summary", len(enc), MaxInvalidEncodingDiagnostics+1)
	}
	lastByte := enc[MaxInvalidEncodingDiagnostics-1]
	if lastByte.Line != MaxInvalidEncodingDiagnostics+2 || lastByte.Message != "invalid UTF-8 byte 0xE9 at column 8 replaced with U+FFFD" {
		t.Errorf("last per-byte diagnostic = line %d %q, want the 0xE9 on line %d", lastByte.Line, lastByte.Message, MaxInvalidEncodingDiagnostics+2)
	}
	summary := enc[MaxInvalidEncodingDiagnostics]
	wantMsg := fmt.Sprintf("%d further invalid UTF-8 bytes, from this line on, replaced with U+FFFD and not reported individually", extra)
	if summary.Line != overLine || summary.Message != wantMsg || summary.Severity != SeverityError {
		t.Errorf("summary = line %d %v %q, want line %d error %q", summary.Line, summary.Severity, summary.Message, overLine, wantMsg)
	}

	notes := result.Document.GetIndividual("@I1@").InlineNotes
	if len(notes) != MaxInvalidEncodingDiagnostics+1 || notes[len(notes)-1] != strings.Repeat("�", extra) {
		t.Errorf("notes = %q, want every byte past the cap still replaced", notes)
	}
}

// Decode discards diagnostics, so it must not build them for invalid bytes,
// yet must still recover from those bytes.
func TestDecodeLenientInvalidUTF8NotCollectedWhenDiscarded(t *testing.T) {
	input, _ := invalidUTF8Probe("1 CHAR UTF-8")
	result, err := decode(strings.NewReader(input), DefaultOptions(), false)
	if err != nil {
		t.Fatalf("decode() error = %v, want the bad byte recovered", err)
	}
	for _, d := range result.Diagnostics {
		if d.Code == CodeInvalidEncoding {
			t.Errorf("decode(collect=false) kept %v, want encoding diagnostics discarded", d)
		}
	}
	if n := len(result.Document.Individuals()); n != 100 {
		t.Errorf("decoded %d individuals, want 100", n)
	}
}
