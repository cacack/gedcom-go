package encoder

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// longNoteDoc builds a document whose one individual carries a note long
// enough to need CONC splitting at the default line length.
func longNoteDoc(version gedcom.Version, note string) *gedcom.Document {
	return &gedcom.Document{
		Header: &gedcom.Header{Version: version, Encoding: gedcom.EncodingUTF8},
		Records: []*gedcom.Record{{
			XRef:   "@I1@",
			Type:   gedcom.RecordTypeIndividual,
			Entity: &gedcom.Individual{XRef: "@I1@", InlineNotes: []string{note}},
		}},
	}
}

func longNote() string {
	return strings.Repeat("The quick brown fox jumps over the lazy dog. ", 20) + "End."
}

// TestEncodeV7NeverWritesCONC checks that 7.0 output carries no CONC, whether
// 7.0 comes from the header or from TargetVersion, and through both the batch
// and streaming encoders (issue #565). GEDCOM 7.0 removed CONC.
func TestEncodeV7NeverWritesCONC(t *testing.T) {
	note := longNote()

	cases := []struct {
		name    string
		version gedcom.Version
		opts    *EncodeOptions
	}{
		{"header 7.0", gedcom.Version70, DefaultOptions()},
		{"header 7.0 with patch level", "7.0.14", DefaultOptions()},
		{"header 7.0 short max length", gedcom.Version70, &EncodeOptions{MaxLineLength: 20}},
		{"TargetVersion 7.0 over 5.5.1 header", gedcom.Version551, &EncodeOptions{TargetVersion: gedcom.Version70}},
	}

	encoders := map[string]func(*bytes.Buffer, *gedcom.Document, *EncodeOptions) error{
		"batch": func(b *bytes.Buffer, d *gedcom.Document, o *EncodeOptions) error {
			return EncodeWithOptions(b, d, o)
		},
		"streaming": func(b *bytes.Buffer, d *gedcom.Document, o *EncodeOptions) error {
			return EncodeStreamingWithOptions(b, d, o)
		},
	}

	for _, tc := range cases {
		for encName, encode := range encoders {
			t.Run(tc.name+"/"+encName, func(t *testing.T) {
				var buf bytes.Buffer
				if err := encode(&buf, longNoteDoc(tc.version, note), tc.opts); err != nil {
					t.Fatalf("encode: %v", err)
				}
				if strings.Contains(buf.String(), " CONC") {
					t.Fatalf("7.0 output contains CONC:\n%s", buf.String())
				}
				if !strings.Contains(buf.String(), "1 NOTE "+note+"\n") {
					t.Errorf("note not written whole on one line:\n%s", buf.String())
				}
			})
		}
	}
}

// TestEncode5xCONCSplitsBetweenNonSpaces checks that 5.x CONC splits never
// leave a space at either edge of a line, and that the text decodes back
// unchanged (issue #565).
func TestEncode5xCONCSplitsBetweenNonSpaces(t *testing.T) {
	note := longNote()

	for _, v := range []gedcom.Version{gedcom.Version55, gedcom.Version551, ""} {
		t.Run("version "+string(v), func(t *testing.T) {
			var buf bytes.Buffer
			opts := &EncodeOptions{MaxLineLength: 100}
			if err := EncodeWithOptions(&buf, longNoteDoc(v, note), opts); err != nil {
				t.Fatalf("encode: %v", err)
			}

			concs := 0
			lines := strings.Split(buf.String(), "\n")
			for i, line := range lines {
				if !strings.HasPrefix(line, "2 CONC ") {
					continue
				}
				concs++
				if strings.HasPrefix(line, "2 CONC  ") {
					t.Errorf("CONC value starts with a space: %q", line)
				}
				if strings.HasSuffix(lines[i-1], " ") {
					t.Errorf("line before CONC ends with a space: %q", lines[i-1])
				}
			}
			if concs == 0 {
				t.Fatalf("expected CONC lines for a %d-byte note:\n%s", len(note), buf.String())
			}

			if opts.version != "" {
				t.Errorf("caller's options were modified: version = %q", opts.version)
			}

			if v == "" {
				return // no header version to decode with
			}
			doc, err := decoder.Decode(&buf)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			got := doc.GetIndividual("@I1@").InlineNotes
			if len(got) != 1 || got[0] != note {
				t.Errorf("note round trip = %q, want %q", got, note)
			}
		})
	}
}

// TestStreamEncoderDoesNotModifyOptions checks that recording the version on
// WriteHeader leaves the caller's options untouched, so one options value can
// be shared across encoders of different versions.
func TestStreamEncoderDoesNotModifyOptions(t *testing.T) {
	opts := DefaultOptions()
	var buf bytes.Buffer
	enc := NewStreamEncoderWithOptions(&buf, opts)
	if err := enc.WriteHeader(&gedcom.Header{Version: gedcom.Version70}); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	if opts.version != "" {
		t.Errorf("caller's options were modified: version = %q", opts.version)
	}
}
