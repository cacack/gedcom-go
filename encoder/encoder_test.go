package encoder

import (
	"bytes"
	"strings"
	"testing"

	"github.com/elliotchance/go-gedcom/decoder"
)

func TestEncodeRoundtrip(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
0 TRLR
`

	// Decode
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// Encode
	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	output := buf.String()
	t.Logf("Encoded output:\n%s", output)

	// Verify it contains key elements
	if !strings.Contains(output, "0 HEAD") {
		t.Error("Output should contain HEAD")
	}
	if !strings.Contains(output, "0 @I1@ INDI") {
		t.Error("Output should contain INDI record")
	}
	if !strings.Contains(output, "1 NAME John /Smith/") {
		t.Error("Output should contain NAME tag")
	}
	if !strings.Contains(output, "0 TRLR") {
		t.Error("Output should contain TRLR")
	}

	// Decode the output to verify it's valid GEDCOM
	doc2, err := decoder.Decode(strings.NewReader(output))
	if err != nil {
		t.Fatalf("Failed to decode encoded output: %v", err)
	}

	if len(doc2.Records) != len(doc.Records) {
		t.Errorf("Record count mismatch: got %d, want %d", len(doc2.Records), len(doc.Records))
	}
}

func TestEncodeCRLF(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 TRLR
`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	opts := &EncodeOptions{
		LineEnding: "\r\n",
	}

	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, doc, opts); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "\r\n") {
		t.Error("Output should contain CRLF line endings")
	}
}
