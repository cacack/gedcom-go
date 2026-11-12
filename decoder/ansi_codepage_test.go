package decoder

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/charset"
)

// ansiCodePageGEDCOM declares CHAR ANSI over the Windows-1250 bytes of
// "Đurić Čač": 0xD0, 0xE6, 0xC8 and 0xE8, which Windows-1252 reads as
// "Ð", "æ", "È" and "è".
const ansiCodePageGEDCOM = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR ANSI\n0 @I1@ INDI\n1 NAME \xD0uri\xE6 /\xC8a\xE8/\n0 TRLR\n"

// TestDecodeANSIEncoding checks DecodeOptions.ANSIEncoding in both modes
// (#759): naming Windows-1250 decodes the letters, and the default keeps the
// Windows-1252 reading.
func TestDecodeANSIEncoding(t *testing.T) {
	tests := []struct {
		name string
		opts *DecodeOptions
		want string
	}{
		{"default", DefaultOptions(), "Ðuriæ /Èaè/"},
		{"Windows-1250", &DecodeOptions{ANSIEncoding: charset.EncodingWindows1250}, "Đurić /Čač/"},
		{"Windows-1250 strict", &DecodeOptions{StrictMode: true, ANSIEncoding: charset.EncodingWindows1250}, "Đurić /Čač/"},
		{"Windows-1252 named", &DecodeOptions{ANSIEncoding: charset.EncodingLATIN1}, "Ðuriæ /Èaè/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := DecodeWithOptions(strings.NewReader(ansiCodePageGEDCOM), tt.opts)
			if err != nil {
				t.Fatalf("DecodeWithOptions() error = %v", err)
			}
			if got := doc.GetIndividual("@I1@").Names[0].Full; got != tt.want {
				t.Errorf("name = %q, want %q", got, tt.want)
			}
			if doc.Header.Encoding != "ANSI" {
				t.Errorf("Header.Encoding = %q, want the declared ANSI kept", doc.Header.Encoding)
			}
		})
	}
}

// TestDecodeAmbiguousCodePageDiagnostic checks that a lenient decode of CHAR
// ANSI reports the first ambiguous byte once, and that naming the code page or
// decoding strictly reports nothing.
func TestDecodeAmbiguousCodePageDiagnostic(t *testing.T) {
	result, err := DecodeWithDiagnostics(strings.NewReader(ansiCodePageGEDCOM), DefaultOptions())
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %v, want one AMBIGUOUS_CODEPAGE", result.Diagnostics)
	}
	d := result.Diagnostics[0]
	wantMsg := "CHAR ANSI read as Windows-1252: byte 0xD0 at column 8 is 'Ð', but 'Đ' in Windows-1250; set DecodeOptions.ANSIEncoding to name the code page"
	if d.Code != CodeAmbiguousCodePage || d.Severity != SeverityWarning || d.Line != 6 || d.Message != wantMsg {
		t.Errorf("diagnostic = %s line %d %v %q; want %s line 6 WARNING %q",
			d.Code, d.Line, d.Severity, d.Message, CodeAmbiguousCodePage, wantMsg)
	}
	if result.Diagnostics.HasErrors() {
		t.Error("HasErrors() = true, want the warning not to count as an error")
	}

	for name, opts := range map[string]*DecodeOptions{
		"Windows-1250 named": {ANSIEncoding: charset.EncodingWindows1250},
		"Windows-1252 named": {ANSIEncoding: charset.EncodingLATIN1},
		"strict":             {StrictMode: true},
	} {
		t.Run(name, func(t *testing.T) {
			result, err := DecodeWithDiagnostics(strings.NewReader(ansiCodePageGEDCOM), opts)
			if err != nil {
				t.Fatalf("DecodeWithDiagnostics() error = %v", err)
			}
			if len(result.Diagnostics) != 0 {
				t.Errorf("diagnostics = %v, want none", result.Diagnostics)
			}
		})
	}
}
