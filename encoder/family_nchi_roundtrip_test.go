package encoder

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/decoder"
)

// TestDecodedNCHIRoundTripsByteIdentically is the regression test issue #485
// asks for: a decoded family carrying `1 NCHI 4` reads back through the
// accessor and re-encodes byte-for-byte, with no second NCHI line from the
// removed dual store.
func TestDecodedNCHIRoundTripsByteIdentically(t *testing.T) {
	const input = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n" +
		"0 @F1@ FAM\n1 HUSB @I1@\n1 NCHI 4\n2 SOUR @S1@\n0 TRLR\n"

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	fam := doc.GetFamily("@F1@")
	if fam == nil {
		t.Fatal("GetFamily(@F1@) returned nil")
	}
	if got := fam.NumberOfChildren(); got != "4" {
		t.Errorf("NumberOfChildren() = %q, want %q", got, "4")
	}

	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, doc, &EncodeOptions{}); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	if got := buf.String(); got != input {
		t.Errorf("round trip not byte-identical:\ngot:\n%s\nwant:\n%s", got, input)
	}
	if n := strings.Count(buf.String(), "NCHI"); n != 1 {
		t.Errorf("NCHI appears %d times, want 1", n)
	}
}
