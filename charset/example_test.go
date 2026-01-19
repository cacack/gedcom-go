package charset_test

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/cacack/gedcom-go/charset"
)

// Example demonstrates basic usage of the charset package with UTF-8 data.
func Example() {
	// GEDCOM data is typically read from a file, but can be any io.Reader
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Hans /Mueller/
0 TRLR`

	// NewReader handles BOM detection and encoding conversion automatically
	reader := charset.NewReader(strings.NewReader(gedcomData))

	// Read the converted UTF-8 content
	content, err := io.ReadAll(reader)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Content is now guaranteed to be valid UTF-8
	fmt.Printf("Read %d bytes of UTF-8 content\n", len(content))
	fmt.Printf("Contains NAME tag: %v\n", strings.Contains(string(content), "NAME Hans /Mueller/"))

	// Output:
	// Read 78 bytes of UTF-8 content
	// Contains NAME tag: true
}

// ExampleNewReader shows how to create a charset-aware reader that handles
// BOM detection and encoding conversion automatically.
func ExampleNewReader() {
	// UTF-8 content with a BOM (common in GEDCOM exports)
	gedcomBytes := append([]byte{0xEF, 0xBB, 0xBF}, []byte("0 HEAD\n1 CHAR UTF-8\n0 TRLR\n")...)

	// NewReader automatically strips the BOM and validates UTF-8
	reader := charset.NewReader(bytes.NewReader(gedcomBytes))

	content, err := io.ReadAll(reader)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// The BOM is removed from the output
	fmt.Printf("First bytes are not BOM: %v\n", content[0] == '0')
	fmt.Printf("Content starts with: %s\n", string(content[:6]))

	// Output:
	// First bytes are not BOM: true
	// Content starts with: 0 HEAD
}

// ExampleNewReader_ansel demonstrates reading ANSEL-encoded GEDCOM data.
// ANSEL is a legacy encoding commonly found in older GEDCOM 5.5 files.
func ExampleNewReader_ansel() {
	// ANSEL-encoded data with a combining acute accent (0xE2) before 'e'
	// In ANSEL, combining marks precede the base character
	// 0xE2 = combining acute accent, followed by 'e' = e with acute (e)
	anselData := []byte("0 HEAD\n1 CHAR ANSEL\n0 @I1@ INDI\n1 NAME Ren")
	anselData = append(anselData, 0xE2, 'e') // acute + e
	anselData = append(anselData, []byte(" /Dupont/\n0 TRLR\n")...)

	reader := charset.NewReader(bytes.NewReader(anselData))

	content, err := io.ReadAll(reader)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// The ANSEL combining mark has been converted to Unicode
	// In Unicode, combining marks follow the base character
	fmt.Printf("Contains converted name: %v\n", strings.Contains(string(content), "Rene"))

	// Output:
	// Contains converted name: true
}

// ExampleDetectBOM shows how to detect the encoding from a Byte Order Mark.
func ExampleDetectBOM() {
	// UTF-16 LE BOM (0xFF 0xFE) followed by some data
	utf16LEData := []byte{0xFF, 0xFE, '0', 0x00, ' ', 0x00}

	reader, encoding, err := charset.DetectBOM(bytes.NewReader(utf16LEData))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// The encoding is detected from the BOM
	switch encoding {
	case charset.EncodingUTF16LE:
		fmt.Println("Detected: UTF-16 LE")
	case charset.EncodingUTF16BE:
		fmt.Println("Detected: UTF-16 BE")
	case charset.EncodingUTF8:
		fmt.Println("Detected: UTF-8")
	default:
		fmt.Println("Detected: Unknown (no BOM)")
	}

	// The returned reader has the BOM consumed but data preserved
	remaining, _ := io.ReadAll(reader)
	fmt.Printf("Remaining bytes: %d\n", len(remaining))

	// Output:
	// Detected: UTF-16 LE
	// Remaining bytes: 4
}

// ExampleDetectEncodingFromHeader shows how to detect encoding from the CHAR tag.
func ExampleDetectEncodingFromHeader() {
	// GEDCOM content with CHAR tag declaring ANSEL encoding
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
1 CHAR ANSEL
0 @I1@ INDI
1 NAME Test /Person/
0 TRLR`

	reader, encoding, err := charset.DetectEncodingFromHeader(strings.NewReader(gedcomData))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// The encoding is detected from the CHAR tag
	switch encoding {
	case charset.EncodingANSEL:
		fmt.Println("Detected: ANSEL")
	case charset.EncodingUTF8:
		fmt.Println("Detected: UTF-8")
	case charset.EncodingASCII:
		fmt.Println("Detected: ASCII")
	default:
		fmt.Println("Detected: Unknown")
	}

	// The returned reader contains all the original content
	content, _ := io.ReadAll(reader)
	fmt.Printf("Content preserved: %v\n", strings.Contains(string(content), "0 HEAD"))

	// Output:
	// Detected: ANSEL
	// Content preserved: true
}

// ExampleValidateString demonstrates validating UTF-8 strings.
func ExampleValidateString() {
	// Valid UTF-8 string with special characters
	validUTF8 := "Hans Muller from Munchen"
	fmt.Printf("Valid UTF-8: %v\n", charset.ValidateString(validUTF8))

	// Invalid UTF-8 (lone continuation byte)
	invalidUTF8 := string([]byte{0x80, 0x81})
	fmt.Printf("Invalid UTF-8: %v\n", charset.ValidateString(invalidUTF8))

	// Output:
	// Valid UTF-8: true
	// Invalid UTF-8: false
}
