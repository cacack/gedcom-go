package decoder

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// recordsFragment is a run of records exercising what the version and
// identifier grammar decide: a spaced pointer (an identifier in 5.5 and
// 5.5.1, an error in 7.0), a leading "@@" escape, an over-jumped level, an
// extension tag and a continued note.
const recordsFragment = "0 @I1@ INDI\n" +
	"1 NAME Ann /Example/\n" +
	"1 BIRT\n" +
	"2 DATE 1 JAN 1900\n" +
	"2 PLAC Exampletown\n" +
	"4 _NOTE deep\n" +
	"1 NOTE @N 1@\n" +
	"1 FAMS @F1@\n" +
	"1 _CUSTOM value\n" +
	"0 @F1@ FAM\n" +
	"1 HUSB @I1@\n" +
	"1 NOTE @@handle on the record\n" +
	"0 @N 1@ NOTE First line\n" +
	"1 CONT second line\n" +
	"0 @S1@ SOUR\n" +
	"1 TITL Example register\n" +
	"1 TEXT @@ quoted\n"

func wholeFile(vers, char, records string) string {
	head := "0 HEAD\n1 GEDC\n2 VERS " + vers + "\n"
	if char != "" {
		head += "1 CHAR " + char + "\n"
	}
	return head + records + "0 TRLR\n"
}

// zeroLineNumbers clears the line numbers DecodeRecords clears, so records
// from a whole file can be compared with records from a fragment.
func zeroLineNumbers(records []*gedcom.Record) {
	for _, r := range records {
		r.LineNumber = 0
		for _, tag := range r.Tags {
			tag.LineNumber = 0
		}
		clearEntityLineNumbers(r.Entity)
	}
}

// TestDecodeRecords_MatchesWholeFile pins that a fragment decodes, under its
// target document's version, to exactly the records the whole-file decoder
// gives the same lines, typed entities included.
func TestDecodeRecords_MatchesWholeFile(t *testing.T) {
	for _, vers := range []string{"5.5", "5.5.1", "7.0"} {
		t.Run(vers, func(t *testing.T) {
			file := wholeFile(vers, "UTF-8", recordsFragment)
			target, err := Decode(strings.NewReader(file))
			if err != nil {
				t.Fatalf("decode whole file: %v", err)
			}
			want, err := Decode(strings.NewReader(file))
			if err != nil {
				t.Fatal(err)
			}
			zeroLineNumbers(want.Records)

			got, _, err := DecodeRecords(strings.NewReader(recordsFragment), target, nil)
			if err != nil {
				t.Fatalf("DecodeRecords: %v", err)
			}
			if len(got) != len(want.Records) {
				t.Fatalf("got %d records, want %d", len(got), len(want.Records))
			}
			for i := range got {
				if !reflect.DeepEqual(got[i], want.Records[i]) {
					t.Errorf("record %d (%s) differs from the whole-file decode:\n got %#v\nwant %#v",
						i, want.Records[i].XRef, got[i].Entity, want.Records[i].Entity)
				}
			}
			if len(target.Records) != len(want.Records) {
				t.Errorf("target gained records: %d", len(target.Records))
			}
		})
	}
}

// TestDecodeRecords_VersionGrammar checks that the identifier grammar is the
// target's: a spaced pointer is an error only in a 7.0 document.
func TestDecodeRecords_VersionGrammar(t *testing.T) {
	const fragment = "0 @I 1@ INDI\n1 NAME Ann /Example/\n"
	for _, tt := range []struct {
		version   gedcom.Version
		wantError bool
	}{
		{gedcom.Version551, false},
		{gedcom.Version70, true},
	} {
		_, diags, err := DecodeRecords(strings.NewReader(fragment), gedcom.NewDocument(tt.version), nil)
		if err != nil {
			t.Fatalf("%s: DecodeRecords: %v", tt.version, err)
		}
		if got := diags.HasErrors(); got != tt.wantError {
			t.Errorf("%s: HasErrors = %v, want %v (%v)", tt.version, got, tt.wantError, diags)
		}
		_, _, err = DecodeRecords(strings.NewReader(fragment), gedcom.NewDocument(tt.version), &DecodeOptions{StrictMode: true})
		if (err != nil) != tt.wantError {
			t.Errorf("%s strict: error = %v, want error %v", tt.version, err, tt.wantError)
		}
	}
}

// TestDecodeRecords_Encoding checks that a fragment of an ANSEL document is
// read as ANSEL when it is not UTF-8, and as UTF-8 when it is, as the
// encoder writes it.
func TestDecodeRecords_Encoding(t *testing.T) {
	// ANSEL 0xE2 is a combining acute accent written before its base letter.
	anselRecord := "0 @I1@ INDI\n1 NAME Ren\xE2e /Example/\n"
	target, err := Decode(strings.NewReader(wholeFile("5.5", "ANSEL", anselRecord)))
	if err != nil {
		t.Fatal(err)
	}
	if target.Header.DetectedEncoding != gedcom.EncodingANSEL {
		t.Fatalf("DetectedEncoding = %q, want ANSEL", target.Header.DetectedEncoding)
	}
	want := target.Records[0].Tags[0].Value

	for name, fragment := range map[string]string{
		"ANSEL": anselRecord,
		"UTF-8": "0 @I1@ INDI\n1 NAME " + want + "\n",
	} {
		got, diags, err := DecodeRecords(strings.NewReader(fragment), target, nil)
		if err != nil {
			t.Fatalf("%s: DecodeRecords: %v", name, err)
		}
		if len(diags) != 0 {
			t.Errorf("%s: diagnostics %v", name, diags)
		}
		if v := got[0].Tags[0].Value; v != want {
			t.Errorf("%s: NAME = %q, want %q", name, v, want)
		}
	}

	// A UTF-8 document reads an invalid byte as the whole-file decoder does:
	// replaced and reported.
	got, diags, err := DecodeRecords(strings.NewReader(anselRecord), gedcom.NewDocument(gedcom.Version551), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got[0].Tags[0].Value, "�") || len(diags) == 0 {
		t.Errorf("NAME = %q, diagnostics %v; want a replaced, reported byte", got[0].Tags[0].Value, diags)
	}
}

// TestDecodeRecords_LineNumbers checks that diagnostics count lines from the
// fragment's first line while the records carry none.
func TestDecodeRecords_LineNumbers(t *testing.T) {
	fragment := "0 @I1@ INDI\n1 NAME Ann /Example/\n0 @I1@ INDI\n1 BIRT\n3 DATE 1900\n"
	got, diags, err := DecodeRecords(strings.NewReader(fragment), gedcom.NewDocument(gedcom.Version551), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d records, want both, the redefinition kept", len(got))
	}
	lines := map[string]int{}
	for _, d := range diags {
		lines[d.Code] = d.Line
	}
	if lines[CodeDuplicateXRef] != 3 || lines[CodeBadLevelJump] != 5 {
		t.Errorf("diagnostic lines = %v, want DUPLICATE_XREF at 3 and BAD_LEVEL_JUMP at 5", lines)
	}
	for _, r := range got {
		if r.LineNumber != 0 {
			t.Errorf("record %s LineNumber = %d, want 0", r.XRef, r.LineNumber)
		}
		for _, tag := range r.Tags {
			if tag.LineNumber != 0 {
				t.Errorf("tag %s LineNumber = %d, want 0", tag.Tag, tag.LineNumber)
			}
		}
	}
}

func TestDecodeRecords_NotRecords(t *testing.T) {
	for name, fragment := range map[string]string{
		"head":        "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @I1@ INDI\n",
		"trailer":     "0 @I1@ INDI\n0 TRLR\n",
		"subordinate": "1 NAME Ann /Example/\n0 @I1@ INDI\n",
	} {
		for _, strict := range []bool{false, true} {
			records, _, err := DecodeRecords(strings.NewReader(fragment), gedcom.NewDocument(gedcom.Version551), &DecodeOptions{StrictMode: strict})
			if !errors.Is(err, ErrNotRecords) {
				t.Errorf("%s (strict %v): error = %v, want ErrNotRecords", name, strict, err)
			}
			if records != nil {
				t.Errorf("%s (strict %v): records = %v, want nil", name, strict, records)
			}
		}
	}

	// A HEAD carrying an identifier is not the header: it is kept as a record.
	records, _, err := DecodeRecords(strings.NewReader("0 @X1@ HEAD\n"), gedcom.NewDocument(gedcom.Version551), nil)
	if err != nil || len(records) != 1 {
		t.Errorf("0 @X1@ HEAD: records = %v, error = %v; want one record", records, err)
	}
}

// A malformed first record line is a skipped record in lenient mode, as in a
// whole file, not a fragment starting at its level-1 subordinate line.
func TestDecodeRecords_RejectedFirstLine(t *testing.T) {
	fragment := "0\n1 NAME Ann /Example/\n0 @I2@ INDI\n"
	records, diags, err := DecodeRecords(strings.NewReader(fragment), gedcom.NewDocument(gedcom.Version551), nil)
	if err != nil {
		t.Fatalf("lenient: error = %v, want none", err)
	}
	if len(records) != 1 || records[0].XRef != "@I2@" {
		t.Errorf("lenient: records = %v, want @I2@ alone", records)
	}
	if len(diags) == 0 {
		t.Error("lenient: no diagnostics, want the skipped record reported")
	}
	if _, _, err := DecodeRecords(strings.NewReader(fragment), gedcom.NewDocument(gedcom.Version551), &DecodeOptions{StrictMode: true}); err == nil {
		t.Error("strict: no error, want the rejected line's")
	}
}

func TestDecodeRecords_EmptyAndNil(t *testing.T) {
	records, diags, err := DecodeRecords(strings.NewReader(""), gedcom.NewDocument(gedcom.Version70), nil)
	if err != nil || len(records) != 0 || len(diags) != 0 {
		t.Errorf("empty fragment: records %v, diagnostics %v, error %v; want none", records, diags, err)
	}
	if _, _, err := DecodeRecords(strings.NewReader("0 @I1@ INDI\n"), nil, nil); err == nil {
		t.Error("nil document: no error")
	}
	// A document with no header reads the fragment as 5.5 with the 7.0
	// identifier grammar, as a whole file declaring no version is read.
	records, _, err = DecodeRecords(strings.NewReader("0 @I1@ INDI\n1 NAME Ann /Example/\n"), &gedcom.Document{}, nil)
	if err != nil || len(records) != 1 || records[0].Entity == nil {
		t.Errorf("headerless document: records %v, error %v", records, err)
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestDecodeRecords_ReadError(t *testing.T) {
	_, _, err := DecodeRecords(failingReader{}, gedcom.NewDocument(gedcom.Version70), nil)
	if err == nil || !strings.Contains(err.Error(), "read failed") {
		t.Errorf("error = %v, want the read error", err)
	}
}
