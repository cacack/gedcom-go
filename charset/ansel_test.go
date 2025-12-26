package charset

import (
	"bytes"
	"io"
	"testing"
)

func TestNewAnselReader_BasicASCII(t *testing.T) {
	input := []byte("Hello World\n")
	r := newAnselReader(bytes.NewReader(input))
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !bytes.Equal(got, input) {
		t.Errorf("newAnselReader() = %q, want %q", got, input)
	}
}

func TestNewAnselReader_ExtendedLatin(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "Polish L (uppercase)",
			input: []byte{0xA1},
			want:  "\u0141", // Ł
		},
		{
			name:  "Polish L (lowercase)",
			input: []byte{0xB1},
			want:  "\u0142", // ł
		},
		{
			name:  "Scandinavian O stroke (uppercase)",
			input: []byte{0xA2},
			want:  "\u00D8", // Ø
		},
		{
			name:  "AE ligature (lowercase)",
			input: []byte{0xB5},
			want:  "\u00E6", // æ
		},
		{
			name:  "Euro sign",
			input: []byte{0xC8},
			want:  "\u20AC", // €
		},
		{
			name:  "Copyright sign",
			input: []byte{0xC3},
			want:  "\u00A9", // ©
		},
		{
			name:  "Inverted question mark",
			input: []byte{0xC5},
			want:  "\u00BF", // ¿
		},
		// LDS Extensions (non-standard ANSEL, but common in GEDCOM files)
		{
			name:  "LDS empty box",
			input: []byte{0xBE},
			want:  "\u25A1", // White square
		},
		{
			name:  "LDS black box",
			input: []byte{0xBF},
			want:  "\u25A0", // Black square
		},
		{
			name:  "LDS midline e",
			input: []byte{0xCD},
			want:  "e", // Midline e rendered as lowercase e
		},
		{
			name:  "LDS midline o",
			input: []byte{0xCE},
			want:  "o", // Midline o rendered as lowercase o
		},
		{
			name:  "LDS alternate eszett",
			input: []byte{0xCF},
			want:  "\u00DF", // ß (same as 0xC7)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newAnselReader(bytes.NewReader(tt.input))
			got, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("ReadAll() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("newAnselReader() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewAnselReader_CombiningDiacriticals(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "Acute accent + e = é",
			input: []byte{0xE2, 'e'},
			want:  "e\u0301", // e + combining acute
		},
		{
			name:  "Grave accent + a = à",
			input: []byte{0xE1, 'a'},
			want:  "a\u0300", // a + combining grave
		},
		{
			name:  "Circumflex + o = ô",
			input: []byte{0xE3, 'o'},
			want:  "o\u0302", // o + combining circumflex
		},
		{
			name:  "Tilde + n = ñ",
			input: []byte{0xE4, 'n'},
			want:  "n\u0303", // n + combining tilde
		},
		{
			name:  "Umlaut + u = ü",
			input: []byte{0xE8, 'u'},
			want:  "u\u0308", // u + combining diaeresis
		},
		{
			name:  "Cedilla + c = ç",
			input: []byte{0xF0, 'c'},
			want:  "c\u0327", // c + combining cedilla
		},
		{
			name:  "Multiple combining marks",
			input: []byte{0xE2, 0xE8, 'a'}, // acute + diaeresis + a
			want:  "a\u0301\u0308",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newAnselReader(bytes.NewReader(tt.input))
			got, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("ReadAll() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("newAnselReader() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewAnselReader_GEDCOMText(t *testing.T) {
	// Simulates GEDCOM content with ANSEL characters
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name: "Name with accented character",
			// "José" where é is acute + e
			input: append([]byte("Jos"), append([]byte{0xE2, 'e'}, []byte(" /Smith/")...)...),
			want:  "Jose\u0301 /Smith/",
		},
		{
			name: "Place with Polish L",
			// "Łódź" partially encoded
			input: []byte{0xA1, 'o', 'd', 'z'},
			want:  "\u0141odz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newAnselReader(bytes.NewReader(tt.input))
			got, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("ReadAll() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("newAnselReader() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewAnselReader_SmallBufferReads(t *testing.T) {
	// Test reading with a small buffer to exercise buffering logic
	input := append([]byte("Hello "), append([]byte{0xE2, 'e'}, []byte(" World")...)...)
	r := newAnselReader(bytes.NewReader(input))

	// Read one byte at a time
	var result []byte
	buf := make([]byte, 1)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}
	}

	want := "Hello e\u0301 World"
	if string(result) != want {
		t.Errorf("Small buffer reads = %q, want %q", result, want)
	}
}

func TestNewAnselReader_EmptyInput(t *testing.T) {
	r := newAnselReader(bytes.NewReader([]byte{}))
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Expected empty output, got %d bytes", len(got))
	}
}

func TestNewAnselReader_InvalidBytes(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "C1 control character (0x80)",
			input: []byte{0x80},
		},
		{
			name:  "Undefined ANSEL byte (0x90)",
			input: []byte{0x90},
		},
		{
			name:  "Undefined ANSEL byte (0xD0)",
			input: []byte{0xD0},
		},
		{
			name:  "Undefined ANSEL byte (0xFF)",
			input: []byte{0xFF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newAnselReader(bytes.NewReader(tt.input))
			_, err := io.ReadAll(r)
			if err == nil {
				t.Error("Expected error for invalid ANSEL byte, got nil")
			}
			anselErr, ok := err.(*ErrInvalidANSEL)
			if !ok {
				t.Errorf("Expected *ErrInvalidANSEL, got %T", err)
			} else if anselErr.Byte != tt.input[0] {
				t.Errorf("Error byte = 0x%02X, want 0x%02X", anselErr.Byte, tt.input[0])
			}
		})
	}
}

func TestNewAnselReader_UndefinedCombining(t *testing.T) {
	// 0xFC and 0xFD are undefined in the combining range
	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "Undefined combining 0xFC",
			input: []byte{0xFC, 'a'},
		},
		{
			name:  "Undefined combining 0xFD",
			input: []byte{0xFD, 'a'},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newAnselReader(bytes.NewReader(tt.input))
			_, err := io.ReadAll(r)
			if err == nil {
				t.Error("Expected error for undefined combining mark, got nil")
			}
			_, ok := err.(*ErrInvalidANSEL)
			if !ok {
				t.Errorf("Expected *ErrInvalidANSEL, got %T", err)
			}
		})
	}
}

func TestNewAnselReader_TrailingCombining(t *testing.T) {
	// Combining marks at end of stream without a base character
	input := []byte{0xE2} // Just an acute accent, no following character
	r := newAnselReader(bytes.NewReader(input))
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	// Should output the combining character as-is
	want := "\u0301" // combining acute
	if string(got) != want {
		t.Errorf("Trailing combining = %q, want %q", got, want)
	}
}

func TestNewAnselReader_UndefinedExtendedLatin(t *testing.T) {
	// 0xAF, 0xBB are undefined in the extended Latin range
	// Note: 0xBE, 0xBF are LDS extensions now mapped to Unicode squares
	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "Undefined 0xAF",
			input: []byte{0xAF},
		},
		{
			name:  "Undefined 0xBB",
			input: []byte{0xBB},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newAnselReader(bytes.NewReader(tt.input))
			_, err := io.ReadAll(r)
			if err == nil {
				t.Error("Expected error for undefined ANSEL byte, got nil")
			}
			_, ok := err.(*ErrInvalidANSEL)
			if !ok {
				t.Errorf("Expected *ErrInvalidANSEL, got %T", err)
			}
		})
	}
}

func TestErrInvalidANSEL_Error(t *testing.T) {
	err := &ErrInvalidANSEL{Line: 5, Column: 10, Byte: 0x80}
	want := "invalid ANSEL byte 0x80 at line 5, column 10"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}

func TestIsCombiningDiacritical(t *testing.T) {
	tests := []struct {
		b    byte
		want bool
	}{
		{0xDF, false}, // Just before range
		{0xE0, true},  // Start of range
		{0xE5, true},  // Middle of range
		{0xFE, true},  // End of range
		{0xFF, false}, // Just after range
		{0x00, false}, // Zero
		{0x7F, false}, // ASCII
		{0xA0, false}, // Extended Latin range
	}

	for _, tt := range tests {
		got := IsCombiningDiacritical(tt.b)
		if got != tt.want {
			t.Errorf("IsCombiningDiacritical(0x%02X) = %v, want %v", tt.b, got, tt.want)
		}
	}
}

func TestNewAnselReader_LineColumnTracking(t *testing.T) {
	// Test that line/column tracking works for error reporting
	input := []byte("Line1\nLine2\n\x80") // Invalid byte on line 3
	r := newAnselReader(bytes.NewReader(input))
	_, err := io.ReadAll(r)

	if err == nil {
		t.Fatal("Expected error for invalid byte")
	}

	anselErr, ok := err.(*ErrInvalidANSEL)
	if !ok {
		t.Fatalf("Expected *ErrInvalidANSEL, got %T", err)
	}

	if anselErr.Line != 3 {
		t.Errorf("Line = %d, want 3", anselErr.Line)
	}
	if anselErr.Column != 1 {
		t.Errorf("Column = %d, want 1", anselErr.Column)
	}
}

func TestNewAnselReader_LargeInput(t *testing.T) {
	// Test with a larger input to exercise buffer handling
	var input bytes.Buffer
	for i := 0; i < 100; i++ {
		input.WriteString("Line ")
		input.Write([]byte{0xE2, 'e'}) // acute + e
		input.WriteString(" content\n")
	}

	r := newAnselReader(&input)
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	// Verify we got the expected number of lines
	lines := bytes.Count(got, []byte{'\n'})
	if lines != 100 {
		t.Errorf("Got %d lines, want 100", lines)
	}
}
