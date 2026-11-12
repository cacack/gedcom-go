package decoder

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// charsetProbe builds a minimal file declaring char, whose @I1@ name and
// @I2@ note are written as given (UTF-8 in the source), CRLF-free.
func charsetProbe(bom bool, char, name, note string) []byte {
	var b bytes.Buffer
	if bom {
		b.WriteString("\xEF\xBB\xBF")
	}
	b.WriteString("0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR " + char + "\n")
	b.WriteString("0 @I1@ INDI\n1 NAME " + name + "\n")
	b.WriteString("0 @I2@ INDI\n1 NAME Second /Person/\n")
	if note != "" {
		b.WriteString("1 NOTE " + note + "\n")
	}
	b.WriteString("0 @I3@ INDI\n1 NAME Third /Person/\n0 TRLR\n")
	return b.Bytes()
}

// TestDecode_CharsetMismatch pins #709: a legacy CHAR declaration over UTF-8
// bytes, and a UTF-8 BOM over any CHAR, decode as UTF-8 with one
// CHARSET_MISMATCH diagnostic on the CHAR line, keeping the declared value in
// Header.Encoding.
func TestDecode_CharsetMismatch(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		declared gedcom.Encoding
		wantName string
		wantNote string
	}{
		{"ANSEL over UTF-8", charsetProbe(false, "ANSEL", "José /Muñoz/", ""), gedcom.EncodingANSEL, "José /Muñoz/", ""},
		{"UTF-8 BOM over ANSEL", charsetProbe(true, "ANSEL", "José /Muñoz/", ""), gedcom.EncodingANSEL, "José /Muñoz/", ""},
		{"ANSI over UTF-8", charsetProbe(false, "ANSI", "Müller /François/", ""), "ANSI", "Müller /François/", ""},
		{"ANSEL over UTF-8 curly quotes", charsetProbe(false, "ANSEL", "José /Muñoz/", "“quoted”"), gedcom.EncodingANSEL, "José /Muñoz/", "“quoted”"},
		{"ASCII over UTF-8", charsetProbe(false, "ASCII", "Müller /François/", ""), gedcom.EncodingASCII, "Müller /François/", ""},
		{"UTF-8 BOM over UTF-16", charsetProbe(true, "UTF-16", "José /Muñoz/", ""), "UTF-16", "José /Muñoz/", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := DecodeWithDiagnostics(bytes.NewReader(tt.input), nil)
			if err != nil {
				t.Fatalf("DecodeWithDiagnostics() error = %v", err)
			}
			doc := res.Document
			if got := doc.GetIndividual("@I1@").Names[0].Full; got != tt.wantName {
				t.Errorf("@I1@ name = %q, want %q", got, tt.wantName)
			}
			if tt.wantNote != "" {
				i2 := doc.GetIndividual("@I2@")
				if len(i2.InlineNotes) != 1 || i2.InlineNotes[0] != tt.wantNote {
					t.Errorf("@I2@ notes = %q, want [%q]", i2.InlineNotes, tt.wantNote)
				}
				if doc.GetIndividual("@I3@") == nil {
					t.Error("@I3@ lost after the curly quotes")
				}
			}
			if doc.Header.Encoding != tt.declared {
				t.Errorf("Header.Encoding = %q, want declared %q", doc.Header.Encoding, tt.declared)
			}
			if doc.Header.DetectedEncoding != gedcom.EncodingUTF8 {
				t.Errorf("Header.DetectedEncoding = %q, want UTF-8", doc.Header.DetectedEncoding)
			}

			if len(res.Diagnostics) != 1 {
				t.Fatalf("diagnostics = %v, want exactly one CHARSET_MISMATCH", res.Diagnostics)
			}
			d := res.Diagnostics[0]
			wantMsg := "header declares CHAR " + string(tt.declared) + " but the input was decoded as UTF-8"
			if d.Code != CodeCharsetMismatch || d.Severity != SeverityWarning || d.Line != 4 ||
				d.Message != wantMsg || d.Context != "1 CHAR "+string(tt.declared) {
				t.Errorf("diagnostic = %+v, want warning %s on line 4 %q", d, CodeCharsetMismatch, wantMsg)
			}

			// Strict mode decodes the same text instead of failing on it.
			strict, err := DecodeWithOptions(bytes.NewReader(tt.input), &DecodeOptions{StrictMode: true})
			if err != nil {
				t.Fatalf("strict decode error = %v", err)
			}
			if got := strict.GetIndividual("@I1@").Names[0].Full; got != tt.wantName {
				t.Errorf("strict @I1@ name = %q, want %q", got, tt.wantName)
			}
		})
	}
}

// TestDecode_CharsetAgreement checks the cases #709 must leave alone: a
// declaration the bytes agree with keeps its encoding and reports nothing.
func TestDecode_CharsetAgreement(t *testing.T) {
	tests := []struct {
		name         string
		input        []byte
		wantName     string
		wantDetected gedcom.Encoding
	}{
		{"genuine ANSEL", charsetProbe(false, "ANSEL", "Jos\xE2e /Mu\xE4noz/", ""), "Jose\u0301 /Mun\u0303oz/", gedcom.EncodingANSEL},
		{"genuine Windows-1252", charsetProbe(false, "ANSI", "M\xFCller /Fran\xE7ois/", ""), "Müller /François/", gedcom.EncodingLATIN1},
		{"ASCII text", charsetProbe(false, "ASCII", "Plain /Name/", ""), "Plain /Name/", gedcom.EncodingASCII},
		{"UTF-8 BOM and CHAR UTF-8", charsetProbe(true, "UTF-8", "José /Muñoz/", ""), "José /Muñoz/", gedcom.EncodingUTF8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := DecodeWithDiagnostics(bytes.NewReader(tt.input), nil)
			if err != nil {
				t.Fatalf("DecodeWithDiagnostics() error = %v", err)
			}
			if got := res.Document.GetIndividual("@I1@").Names[0].Full; got != tt.wantName {
				t.Errorf("@I1@ name = %q, want %q", got, tt.wantName)
			}
			if got := res.Document.Header.DetectedEncoding; got != tt.wantDetected {
				t.Errorf("Header.DetectedEncoding = %q, want %q", got, tt.wantDetected)
			}
			if len(res.Diagnostics) != 0 {
				t.Errorf("diagnostics = %v, want none", res.Diagnostics)
			}
		})
	}
}

// TestDecode_EncodingFixturesHaveNoCharsetMismatch pins that UTF-8 sniffing
// (#709) changes nothing for the encoding fixtures: none of them is UTF-8
// under a legacy label, so each keeps the encoding it was decoded as before.
func TestDecode_EncodingFixturesHaveNoCharsetMismatch(t *testing.T) {
	want := map[string]gedcom.Encoding{
		"ansel-lf.ged":                    gedcom.EncodingANSEL,
		"ansel-trailing-combining.ged":    gedcom.EncodingANSEL,
		"ansi-cp1252-c1-range.ged":        gedcom.EncodingLATIN1,
		"ansi-cp1252-ftm17.ged":           gedcom.EncodingLATIN1,
		"brotherskeeper-5.5.1-w1250.ged":  gedcom.EncodingLATIN1,
		"familyhistorian-5.5.1-w1250.ged": gedcom.EncodingLATIN1,
		"geneanet-5.5.1-w1250.ged":        gedcom.EncodingLATIN1,
		"ibm-windows-easytree.ged":        gedcom.EncodingUTF8,
		"ibmpc-cp437-broskeep.ged":        gedcom.EncodingCP437,
		"utf16be.ged":                     gedcom.EncodingUTF16BE,
		"utf16le.ged":                     gedcom.EncodingUTF16LE,
		"utf8-bom.ged":                    gedcom.EncodingUTF8,
		"utf8-nobom-lf.ged":               gedcom.EncodingUTF8,
		"utf8-unicode.ged":                gedcom.EncodingUTF8,
	}
	paths, err := filepath.Glob("../testdata/encoding/*.ged")
	if err != nil || len(paths) == 0 {
		t.Fatalf("no encoding fixtures found: %v", err)
	}
	for _, path := range paths {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			wantEnc, ok := want[name]
			if !ok {
				t.Fatalf("fixture %s has no expected encoding; add it to this table", name)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			res, _ := DecodeWithDiagnostics(bytes.NewReader(data), nil)
			if res == nil {
				t.Fatal("DecodeWithDiagnostics() returned no result")
			}
			if got := res.Document.Header.DetectedEncoding; got != wantEnc {
				t.Errorf("Header.DetectedEncoding = %q, want %q", got, wantEnc)
			}
			for _, d := range res.Diagnostics {
				if d.Code == CodeCharsetMismatch {
					t.Errorf("unexpected diagnostic: %v", d)
				}
			}
		})
	}
}

// TestDecode_CharsetMismatchDetectedEncodingWithoutDiagnostics checks that
// DecodeWithOptions, which discards diagnostics, still applies the override.
func TestDecode_CharsetMismatchDetectedEncodingWithoutDiagnostics(t *testing.T) {
	doc, err := Decode(strings.NewReader(string(charsetProbe(false, "ANSEL", "José /Muñoz/", ""))))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Header.Encoding != gedcom.EncodingANSEL || doc.Header.DetectedEncoding != gedcom.EncodingUTF8 {
		t.Errorf("Encoding/DetectedEncoding = %q/%q, want ANSEL/UTF-8", doc.Header.Encoding, doc.Header.DetectedEncoding)
	}
}
