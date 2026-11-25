package decoder

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// headerFragment returns a header declaring vers, with the structures the
// decoder types (SOUR with VERS, NAME and CORP, DEST, DATE with TIME, SUBM,
// COPR, LANG, PLAC.FORM, a continued NOTE) and an extension tag.
func headerFragment(vers string) string {
	return "0 HEAD\n" +
		"1 GEDC\n2 VERS " + vers + "\n" +
		"1 CHAR UTF-8\n" +
		"1 SOUR RootsMagic\n2 VERS 9.0\n2 NAME RootsMagic\n2 CORP RootsMagic, Inc.\n" +
		"1 DEST ANSTFILE\n" +
		"1 DATE 1 JAN 2024\n2 TIME 12:00:00\n" +
		"1 SUBM @U1@\n" +
		"1 COPR Example\n" +
		"1 LANG English\n" +
		"1 PLAC\n2 FORM City, County, State\n" +
		"1 NOTE First line\n2 CONT second line\n" +
		"1 _CUSTOM value\n"
}

// TestDecodeHeader_MatchesWholeFile pins that a header fragment decodes,
// under its target document's version, to exactly the header the whole-file
// decoder gives the same lines.
func TestDecodeHeader_MatchesWholeFile(t *testing.T) {
	for _, vers := range []string{"5.5", "5.5.1", "7.0"} {
		t.Run(vers, func(t *testing.T) {
			fragment := headerFragment(vers)
			file := fragment + "0 @U1@ SUBM\n1 NAME Ann\n0 TRLR\n"
			target, err := Decode(strings.NewReader(file))
			if err != nil {
				t.Fatal(err)
			}
			want, err := Decode(strings.NewReader(file))
			if err != nil {
				t.Fatal(err)
			}
			for _, tag := range want.Header.Tags {
				tag.LineNumber = 0
			}

			got, diags, err := DecodeHeader(strings.NewReader(fragment), target, nil)
			if err != nil {
				t.Fatalf("DecodeHeader: %v", err)
			}
			if len(diags) != 0 {
				t.Errorf("diagnostics: %v", diags)
			}
			if !reflect.DeepEqual(got, want.Header) {
				t.Errorf("header differs from the whole-file decode:\n got %#v\nwant %#v", got, want.Header)
			}
			if got == target.Header {
				t.Error("the target's header was returned")
			}
		})
	}
}

// TestDecodeHeader_TargetVersion checks that a fragment is read as the target
// document's version, not the version its GEDC.VERS declares.
func TestDecodeHeader_TargetVersion(t *testing.T) {
	fragment := "0 HEAD\n1 GEDC\n2 VERS 7.0\n1 SCHMA\n2 TAG _A https://example.com/a\n"
	for _, v := range []gedcom.Version{gedcom.Version55, gedcom.Version551, gedcom.Version70} {
		doc := gedcom.NewDocument(v)
		h, _, err := DecodeHeader(strings.NewReader(fragment), doc, nil)
		if err != nil {
			t.Fatalf("%s: %v", v, err)
		}
		if h.Version != v || h.DeclaredVersion != v {
			t.Errorf("%s: Version %q, DeclaredVersion %q; want the target's", v, h.Version, h.DeclaredVersion)
		}
		if err := doc.ReplaceHeader(h); err != nil {
			t.Fatal(err)
		}
		if gotSchema := doc.Schema != nil; gotSchema != (v == gedcom.Version70) {
			t.Errorf("%s: Schema = %+v; a schema only for 7.0", v, doc.Schema)
		}
	}
}

// TestDecodeHeader_Diagnostics checks that diagnostics count lines from the
// fragment's first line while the header's tags carry none.
func TestDecodeHeader_Diagnostics(t *testing.T) {
	fragment := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 FOO bar\n1 NOTE x\n3 CONT deep\n"
	h, diags, err := DecodeHeader(strings.NewReader(fragment), gedcom.NewDocument(gedcom.Version551), nil)
	if err != nil {
		t.Fatal(err)
	}
	lines := map[string]int{}
	for _, d := range diags {
		lines[d.Code] = d.Line
	}
	if lines[CodeUnknownTag] != 4 || lines[CodeBadLevelJump] != 6 {
		t.Errorf("diagnostic lines = %v, want UNKNOWN_TAG at 4 and BAD_LEVEL_JUMP at 6", lines)
	}
	if len(h.Tags) != 5 {
		t.Fatalf("got %d tags, want 5", len(h.Tags))
	}
	for _, tag := range h.Tags {
		if tag.LineNumber != 0 {
			t.Errorf("tag %s LineNumber = %d, want 0", tag.Tag, tag.LineNumber)
		}
	}
	if _, _, err := DecodeHeader(strings.NewReader("0 HEAD\nnot a line\n"), gedcom.NewDocument(gedcom.Version551), &DecodeOptions{StrictMode: true}); err == nil {
		t.Error("strict: no error, want the malformed line's")
	}
}

func TestDecodeHeader_NotHeader(t *testing.T) {
	for _, tt := range []struct {
		name, fragment, line string
	}{
		{"empty", "", "no 0 HEAD"},
		{"record", "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @I1@ INDI\n1 NAME Ann\n", "line 4 is INDI"},
		{"trailer", "0 HEAD\n1 CHAR UTF-8\n0 TRLR\n", "line 3 is TRLR"},
		{"second head", "0 HEAD\n1 CHAR UTF-8\n0 HEAD\n1 NOTE merged\n", "line 3 is HEAD"},
		{"record first", "0 @I1@ INDI\n", "line 1 is not 0 HEAD"},
		{"subordinate first", "1 CHAR UTF-8\n0 HEAD\n", "line 1 is not 0 HEAD"},
		{"identified head", "0 @X1@ HEAD\n", "line 1 is not 0 HEAD"},
	} {
		for _, strict := range []bool{false, true} {
			h, _, err := DecodeHeader(strings.NewReader(tt.fragment), gedcom.NewDocument(gedcom.Version551), &DecodeOptions{StrictMode: strict})
			if !errors.Is(err, ErrNotHeader) || !strings.Contains(err.Error(), tt.line) {
				t.Errorf("%s (strict %v): error = %v, want ErrNotHeader naming %q", tt.name, strict, err, tt.line)
			}
			if h != nil {
				t.Errorf("%s (strict %v): header = %v, want nil", tt.name, strict, h)
			}
		}
	}
}

// TestDecodeHeader_Encoding checks that a fragment of an ANSEL document is
// read as ANSEL when it is not UTF-8, and as UTF-8 when it is, whatever its
// own CHAR line says, and that the header keeps the document's
// DetectedEncoding.
func TestDecodeHeader_Encoding(t *testing.T) {
	// ANSEL 0xE2 is a combining acute accent written before its base letter.
	anselHeader := "0 HEAD\n1 GEDC\n2 VERS 5.5\n1 CHAR ANSEL\n1 COPR Ren\xE2e\n"
	target, err := Decode(strings.NewReader(anselHeader + "0 TRLR\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := target.Header.Copyright
	for name, fragment := range map[string]string{
		"ANSEL": anselHeader,
		"UTF-8": "0 HEAD\n1 GEDC\n2 VERS 5.5\n1 CHAR ANSEL\n1 COPR " + want + "\n",
	} {
		h, diags, err := DecodeHeader(strings.NewReader(fragment), target, nil)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(diags) != 0 {
			t.Errorf("%s: diagnostics %v", name, diags)
		}
		if h.Copyright != want || h.DetectedEncoding != gedcom.EncodingANSEL {
			t.Errorf("%s: COPR %q, DetectedEncoding %q; want %q, ANSEL", name, h.Copyright, h.DetectedEncoding, want)
		}
	}
}

func TestDecodeHeader_NilAndHeaderless(t *testing.T) {
	if _, _, err := DecodeHeader(strings.NewReader("0 HEAD\n"), nil, nil); err == nil {
		t.Error("nil document: no error")
	}
	h, _, err := DecodeHeader(strings.NewReader("0 HEAD\n"), &gedcom.Document{}, nil)
	if err != nil || h == nil || len(h.Tags) != 0 || h.DetectedEncoding != "" {
		t.Errorf("headerless document: header %+v, error %v", h, err)
	}
	if _, _, err := DecodeHeader(failingReader{}, gedcom.NewDocument(gedcom.Version70), nil); err == nil || !strings.Contains(err.Error(), "read failed") {
		t.Errorf("read error: error = %v, want the read error", err)
	}
}
