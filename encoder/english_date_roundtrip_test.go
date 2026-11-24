package encoder

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
)

// TestDecodedEnglishDatesRoundTripByteIdentically covers issue #907: a DATE
// spelled out in English is parsed on decode but written back as it was read.
func TestDecodedEnglishDatesRoundTripByteIdentically(t *testing.T) {
	const input = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
		"0 @I1@ INDI\n1 NAME Mary /Sample/\n1 BIRT\n2 DATE June 1880\n1 DEAT\n2 DATE 12 December 1846\n" +
		"1 BAPL\n2 DATE Abt. 1881\n1 OCCU Weaver\n2 DATE between 1880 and  1885\n" +
		"0 @I2@ INDI\n1 NAME Anna /Sample/\n1 BIRT\n2 DATE about 1873\n" +
		"0 @F1@ FAM\n1 MARR\n2 DATE before 1900\n" +
		"0 TRLR\n"

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	birth := doc.GetIndividual("@I1@").BirthEvent()
	if birth == nil || birth.ParsedDate == nil || birth.ParsedDate.Month != 6 {
		t.Fatalf("birth = %+v, want a parsed June 1880", birth)
	}

	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, doc, &EncodeOptions{}); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	if got := buf.String(); got != input {
		t.Errorf("round trip not byte-identical:\ngot:\n%s\nwant:\n%s", got, input)
	}

	// With the raw tags gone the record is written from the typed entity, which
	// must carry the same raw value.
	for _, record := range doc.Records {
		record.Tags = nil
	}
	buf.Reset()
	if err := EncodeWithOptions(&buf, doc, &EncodeOptions{}); err != nil {
		t.Fatalf("Encode() from entities error = %v", err)
	}
	for _, line := range []string{
		"2 DATE June 1880\n", "2 DATE 12 December 1846\n", "2 DATE Abt. 1881\n",
		"2 DATE between 1880 and  1885\n", "2 DATE about 1873\n", "2 DATE before 1900\n",
	} {
		if !strings.Contains(buf.String(), line) {
			t.Errorf("entity-built output lacks %q:\n%s", line, buf.String())
		}
	}
}
