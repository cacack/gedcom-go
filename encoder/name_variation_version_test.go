package encoder

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Name variations are version-specific (issue #692): FONE and ROMN exist only
// in GEDCOM 5.5.1, TRAN only in 7.0. An encode from the typed model must write
// each only for the output version that defines it.

const nameVariations551 = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
	"0 @I1@ INDI\n" +
	"1 NAME Taro /Yamada/\n2 GIVN Taro\n2 SURN Yamada\n" +
	"2 FONE Yamada /Taro/\n3 TYPE kana\n" +
	"2 ROMN Yamada /Taro/\n3 TYPE romaji\n" +
	"0 TRLR\n"

const nameVariations70 = "0 HEAD\n1 GEDC\n2 VERS 7.0\n" +
	"0 @I1@ INDI\n" +
	"1 NAME Taro /Yamada/\n2 GIVN Taro\n2 SURN Yamada\n" +
	"2 TRAN Yamada /Taro/\n3 LANG ja-Latn\n" +
	"0 TRLR\n"

// decodeEntityOnly decodes input and clears every record's Tags, so the encoder
// writes each record from its typed entity.
func decodeEntityOnly(t *testing.T, input string) *gedcom.Document {
	t.Helper()
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	for _, rec := range doc.Records {
		rec.Tags = nil // force the entity path
	}
	return doc
}

type encodeFunc func(io.Writer, *gedcom.Document, *EncodeOptions) error

var nameVariationEncoders = map[string]encodeFunc{
	"batch":     EncodeWithOptions,
	"streaming": EncodeStreamingWithOptions,
}

func encodeWithFunc(t *testing.T, encode encodeFunc, doc *gedcom.Document, opts *EncodeOptions) string {
	t.Helper()
	var buf bytes.Buffer
	if err := encode(&buf, doc, opts); err != nil {
		t.Fatalf("encode error = %v", err)
	}
	return buf.String()
}

// hasTagLine reports whether out has a line whose tag is tag at any level.
func hasTagLine(out, tag string) bool {
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == tag {
			return true
		}
	}
	return false
}

func TestNameVariations_DroppedForOtherVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		target  gedcom.Version
		dropped []string
	}{
		{"5.5.1 to 7.0 drops FONE and ROMN", nameVariations551, gedcom.Version70, []string{"FONE", "ROMN"}},
		{"7.0 to 5.5.1 drops TRAN", nameVariations70, gedcom.Version551, []string{"TRAN"}},
		{"7.0 to 5.5 drops TRAN", nameVariations70, gedcom.Version55, []string{"TRAN"}},
	}
	for _, tt := range tests {
		for encName, encode := range nameVariationEncoders {
			t.Run(tt.name+"/"+encName, func(t *testing.T) {
				opts := &EncodeOptions{TargetVersion: tt.target}
				out := encodeWithFunc(t, encode, decodeEntityOnly(t, tt.input), opts)
				for _, tag := range tt.dropped {
					if hasTagLine(out, tag) {
						t.Errorf("output for %s has a %s line:\n%s", tt.target, tag, out)
					}
				}
				if !strings.Contains(out, "1 NAME Taro /Yamada/\n") {
					t.Errorf("output lost the NAME line:\n%s", out)
				}
				if *opts != (EncodeOptions{TargetVersion: tt.target}) {
					t.Errorf("caller's options were modified: %+v", *opts)
				}
			})
		}
	}
}

// TestNameVariations_KeptForOwnVersion pins that output for the document's
// own version -- no target, or a target equal to it -- is unchanged.
func TestNameVariations_KeptForOwnVersion(t *testing.T) {
	tests := []struct {
		name  string
		input string
		own   gedcom.Version
		want  string
	}{
		{"5.5.1 keeps FONE and ROMN", nameVariations551, gedcom.Version551, nameVariations551},
		{"7.0 keeps TRAN", nameVariations70, gedcom.Version70, nameVariations70},
	}
	for _, tt := range tests {
		for encName, encode := range nameVariationEncoders {
			for optName, opts := range map[string]*EncodeOptions{
				"no target":   {},
				"same target": {TargetVersion: tt.own},
			} {
				t.Run(tt.name+"/"+encName+"/"+optName, func(t *testing.T) {
					if got := encodeWithFunc(t, encode, decodeEntityOnly(t, tt.input), opts); got != tt.want {
						t.Errorf("output =\n%s\nwant:\n%s", got, tt.want)
					}
				})
			}
		}
	}
}
