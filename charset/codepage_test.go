package charset

import (
	"errors"
	"io"
	"strings"
	"testing"
	"unicode/utf16"
)

// TestDetectEncodingFromHeader_CHARValues covers every CHAR value #735 added,
// and that the whole value is read rather than its first word.
func TestDetectEncodingFromHeader_CHARValues(t *testing.T) {
	tests := []struct {
		value string
		want  Encoding
	}{
		{"UTF8", EncodingUTF8},
		{"utf8", EncodingUTF8},
		{"WINDOWS-1252", EncodingLATIN1},
		{"windows-1252", EncodingLATIN1},
		{"CP1252", EncodingLATIN1},
		{"WIN-1252", EncodingLATIN1},
		{"IBMPC", EncodingCP437},
		{"IBM PC", EncodingCP437},
		{"IBM   PC", EncodingCP437},
		{"ibm pc", EncodingCP437},
		{"CP437", EncodingCP437},
		{"IBM-437", EncodingCP437},
		{"CP850", EncodingCP850},
		{"IBM-850", EncodingCP850},
		{"MACINTOSH", EncodingMacRoman},
		{"MAC", EncodingMacRoman},
		{"MACROMAN", EncodingMacRoman},
		{"ANSEL ", EncodingANSEL},
		{"UTF-8\t", EncodingUTF8},
		// Whole value, not first word: "IBM WINDOWS" is no recognised value
		// and must not be read as one of its words.
		{"IBM WINDOWS", EncodingUnknown},
		{"ANSEL EXTRA", EncodingUnknown},
	}
	for _, eol := range []string{"\n", "\r", "\r\n"} {
		for _, tt := range tests {
			input := "0 HEAD" + eol + "1 CHAR " + tt.value + eol + "0 TRLR" + eol
			t.Run(strings.TrimSpace(tt.value)+"/"+strings.ReplaceAll(strings.ReplaceAll(eol, "\r", "CR"), "\n", "LF"), func(t *testing.T) {
				r, got, err := DetectEncodingFromHeader(strings.NewReader(input))
				if err != nil {
					t.Fatalf("DetectEncodingFromHeader() error = %v", err)
				}
				if got != tt.want {
					t.Errorf("CHAR %q: encoding = %v, want %v", tt.value, got, tt.want)
				}
				data, err := io.ReadAll(r)
				if err != nil || string(data) != input {
					t.Errorf("replayed %q, %v; want %q", data, err, input)
				}
			})
		}
	}
}

// TestDetectEncodingFromHeader_CHARWithoutValue checks that an empty CHAR does
// not borrow the next line's text as its value.
func TestDetectEncodingFromHeader_CHARWithoutValue(t *testing.T) {
	_, got, err := DetectEncodingFromHeader(strings.NewReader("0 HEAD\n1 CHAR\nMAC\n0 TRLR\n"))
	if err != nil {
		t.Fatalf("DetectEncodingFromHeader() error = %v", err)
	}
	if got != EncodingUnknown {
		t.Errorf("encoding = %v, want EncodingUnknown", got)
	}
}

// TestNewReader_CodePages decodes a high byte under each code page CHAR value,
// choosing bytes whose character differs between the code pages so a wrong
// table is caught.
func TestNewReader_CodePages(t *testing.T) {
	tests := []struct {
		char string
		b    byte
		want string
	}{
		{"IBMPC", 0x82, "é"},
		{"IBM PC", 0xD0, "╨"},
		{"CP437", 0x9C, "£"},
		{"CP850", 0xD0, "ð"},
		{"IBM-850", 0x9B, "ø"},
		{"MACINTOSH", 0x8E, "é"},
		{"MACROMAN", 0xD2, "“"},
		{"WINDOWS-1252", 0x80, "€"},
		{"CP1252", 0x8E, "Ž"},
	}
	for _, tt := range tests {
		t.Run(tt.char, func(t *testing.T) {
			input := "0 HEAD\n1 CHAR " + tt.char + "\n0 @N1@ NOTE Fr" + string([]byte{tt.b}) + "mont\n0 TRLR\n"
			got, err := io.ReadAll(NewReader(strings.NewReader(input)))
			if err != nil {
				t.Fatalf("ReadAll() error = %v", err)
			}
			want := strings.Replace(input, string([]byte{tt.b}), tt.want, 1)
			if string(got) != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

// TestNewReader_UTF8Explicit checks that CHAR UTF8 (no hyphen) reads UTF-8.
func TestNewReader_UTF8Explicit(t *testing.T) {
	input := "0 HEAD\n1 CHAR UTF8\n0 @N1@ NOTE Frémont\n0 TRLR\n"
	got, err := io.ReadAll(NewReader(strings.NewReader(input)))
	if err != nil || string(got) != input {
		t.Errorf("got %q, %v; want %q", got, err, input)
	}
}

func encodeUTF16(s string, bigEndian bool) []byte {
	var out []byte
	for _, u := range utf16.Encode([]rune(s)) {
		if bigEndian {
			out = append(out, byte(u>>8), byte(u))
		} else {
			out = append(out, byte(u), byte(u>>8))
		}
	}
	return out
}

// TestNewReader_BOMlessUTF16 checks that UTF-16 without a BOM is recognised
// by its first line, as GEDCOM 5.5 requires for CHAR UNICODE.
func TestNewReader_BOMlessUTF16(t *testing.T) {
	const text = "0 HEAD\n1 CHAR UNICODE\n0 @N1@ NOTE Frémont ∑\n0 TRLR\n"
	for _, tt := range []struct {
		name      string
		bigEndian bool
	}{{"LE", false}, {"BE", true}} {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReader(strings.NewReader(string(encodeUTF16(text, tt.bigEndian))))
			got, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("ReadAll() error = %v", err)
			}
			if string(got) != text {
				t.Errorf("got %q, want %q", got, text)
			}
		})
	}
}

// TestNewReader_NotUTF16 checks that inputs too short or unlike a UTF-16
// "0 " are read as before.
func TestNewReader_NotUTF16(t *testing.T) {
	for _, input := range []string{"", "0", "0\x00", "0\x00 ", "0 HEAD\n0 TRLR\n", "0\x00\x00\x00"} {
		got, err := io.ReadAll(NewReader(strings.NewReader(input)))
		if err != nil || string(got) != input {
			t.Errorf("NewReader(%q) = %q, %v; want the input unchanged", input, got, err)
		}
	}
}

// TestNewReader_UTF16SniffReadError checks that a read error while sniffing
// for UTF-16 is reported after the bytes already read, none lost.
func TestNewReader_UTF16SniffReadError(t *testing.T) {
	boom := errors.New("boom")
	// DetectBOM reads three bytes cleanly; the error arrives on the fourth.
	r := NewReader(io.MultiReader(strings.NewReader("0 H"), &errorReader{err: boom}))
	got, err := io.ReadAll(r)
	if !errors.Is(err, boom) {
		t.Errorf("ReadAll() error = %v, want %v", err, boom)
	}
	if string(got) != "0 H" {
		t.Errorf("ReadAll() = %q, want %q", got, "0 H")
	}
}
