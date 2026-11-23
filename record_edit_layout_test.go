package gedcomgo

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
)

// recordEditSource has the layout a normalized encode would rewrite: CRLF
// terminators, indentation by level, blank lines between records and a
// trailing space on a valueless line.
const recordEditSource = "0 HEAD\r\n" +
	"  1 GEDC\r\n" +
	"    2 VERS 5.5.1\r\n" +
	"  1 CHAR UTF-8\r\n" +
	"\r\n" +
	"0 @I1@ INDI\r\n" +
	"  1 NAME Ann /Example/\r\n" +
	"  1 FAMS @F1@\r\n" +
	"\r\n" +
	"0 @I2@ INDI\r\n" +
	"  1 NAME Ben /Example/\r\n" +
	"  1 BIRT \r\n" +
	"    2 DATE 1 JAN 1900\r\n" +
	"\r\n" +
	"0 @F1@ FAM\r\n" +
	"  1 HUSB @I1@\r\n" +
	"  1 CHIL @I2@\r\n" +
	"\r\n" +
	"0 TRLR\r\n"

func decodeForEdit(t *testing.T) *Document {
	t.Helper()
	doc, err := decoder.DecodeWithOptions(strings.NewReader(recordEditSource), &decoder.DecodeOptions{PreserveLayout: true})
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func encodeLayout(t *testing.T, doc *Document) string {
	t.Helper()
	var out bytes.Buffer
	if err := encoder.EncodeWithOptions(&out, doc, &encoder.EncodeOptions{PreserveLayout: true}); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

// TestReplaceRecord_PreservesOtherRecords checks that, with a layout-preserving
// encode, replacing a record with one decoded from a fragment changes only
// that record's lines: every other record is written byte for byte as read.
//
// The fragment's record starts on line 10 (after blank lines, as an editor
// might leave them), the line number of @I2@ in the source, so a decoded
// record that kept fragment line numbers would take @I2@'s layout.
func TestReplaceRecord_PreservesOtherRecords(t *testing.T) {
	doc := decodeForEdit(t)
	fragment := strings.Repeat("\n", 9) + "0 @I1@ INDI\n1 NAME Ann /Changed/\n1 FAMS @F1@\n"
	records, _, err := decoder.DecodeRecords(strings.NewReader(fragment), doc, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.ReplaceRecord(records[0]); err != nil {
		t.Fatal(err)
	}

	got := encodeLayout(t, doc)
	oldRecord := "\r\n0 @I1@ INDI\r\n  1 NAME Ann /Example/\r\n  1 FAMS @F1@\r\n"
	newRecord := "0 @I1@ INDI\r\n1 NAME Ann /Changed/\r\n1 FAMS @F1@\r\n"
	want := strings.Replace(recordEditSource, oldRecord, newRecord, 1)
	if got != want {
		t.Errorf("encoded:\n%q\nwant:\n%q", got, want)
	}
}

// TestRemoveRecord_PreservesOtherRecords checks that removing a record, and
// the pointer to it, leaves every other line as read.
func TestRemoveRecord_PreservesOtherRecords(t *testing.T) {
	doc := decodeForEdit(t)
	refs, err := doc.RemoveRecord("@I2@")
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 || refs[0].XRef != "@F1@" || strings.Join(refs[0].Path, ".") != "FAM.CHIL" {
		t.Fatalf("references = %+v, want the family's CHIL", refs)
	}
	fam := doc.GetRecord("@F1@")
	fam.Tags = fam.Tags[:1]

	got := encodeLayout(t, doc)
	want := strings.Replace(recordEditSource,
		"\r\n0 @I2@ INDI\r\n  1 NAME Ben /Example/\r\n  1 BIRT \r\n    2 DATE 1 JAN 1900\r\n", "", 1)
	want = strings.Replace(want, "  1 CHIL @I2@\r\n", "", 1)
	if got != want {
		t.Errorf("encoded:\n%q\nwant:\n%q", got, want)
	}
}
