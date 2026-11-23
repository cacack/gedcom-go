package gedcomgo

import (
	"bytes"
	"strings"
	"testing"
)

// TestBlankLinesLeaveDocumentUnchanged checks that a lenient decode skips
// blank lines between records without changing the document (#885): it
// encodes the same as the input without them.
func TestBlankLinesLeaveDocumentUnchanged(t *testing.T) {
	const plain = "0 HEAD\n1 GEDC\n2 VERS 5.5\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
		"0 @I1@ INDI\n1 NAME First /Example/\n1 NOTE A note\n2 CONT second line\n" +
		"0 @I2@ INDI\n1 NAME Second /Example/\n0 TRLR\n"
	blank := strings.NewReplacer("\n0 ", "\n\n0 ", "\n2 CONT", "\n \n2 CONT").Replace(plain)

	encode := func(input string) []byte {
		t.Helper()
		result, err := DecodeWithDiagnostics(strings.NewReader(input))
		if err != nil {
			t.Fatalf("DecodeWithDiagnostics() error = %v", err)
		}
		if result.Diagnostics.HasErrors() {
			t.Errorf("diagnostics have errors: %v", result.Diagnostics)
		}
		var buf bytes.Buffer
		if err := Encode(&buf, result.Document); err != nil {
			t.Fatalf("Encode() error = %v", err)
		}
		return buf.Bytes()
	}

	if got, want := encode(blank), encode(plain); !bytes.Equal(got, want) {
		t.Errorf("document with blank lines encodes as\n%s\nwant\n%s", got, want)
	}
}
