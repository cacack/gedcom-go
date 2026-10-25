package encoder

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// longNoteText is a single line well past DefaultMaxLineLength, with spaces
// wherever a word-boundary splitter would have cut.
var longNoteText = strings.Repeat("A long note about the family history. ", 12)

// lineWrapDoc returns a document with one individual, built from its entity,
// whose inline note is longNoteText.
func lineWrapDoc(version gedcom.Version) *gedcom.Document {
	return &gedcom.Document{
		Header: &gedcom.Header{Version: version},
		Records: []*gedcom.Record{{
			XRef:   "@I1@",
			Type:   gedcom.RecordTypeIndividual,
			Entity: &gedcom.Individual{XRef: "@I1@", InlineNotes: []string{longNoteText}},
		}},
	}
}

// decodedNote re-decodes output and returns @I1@'s first inline note.
func decodedNote(t *testing.T, output string) string {
	t.Helper()
	doc, err := decoder.Decode(strings.NewReader(output))
	if err != nil {
		t.Fatalf("re-decode: %v\n%s", err, output)
	}
	indi := doc.GetIndividual("@I1@")
	if indi == nil || len(indi.InlineNotes) == 0 {
		t.Fatalf("re-decoded @I1@ has no inline note\n%s", output)
	}
	return indi.InlineNotes[0]
}

// TestEncode70WritesNoCONC asserts GEDCOM 7.0 output never carries CONC, which
// 7.0 removed (issue #565), whichever way the output version is chosen, and
// that the caller's options are left as they were.
func TestEncode70WritesNoCONC(t *testing.T) {
	encodeWith := func(opts *EncodeOptions) func(*bytes.Buffer, *gedcom.Document) error {
		return func(buf *bytes.Buffer, doc *gedcom.Document) error {
			return EncodeWithOptions(buf, doc, opts)
		}
	}
	streamWith := func(opts *EncodeOptions) func(*bytes.Buffer, *gedcom.Document) error {
		return func(buf *bytes.Buffer, doc *gedcom.Document) error {
			enc := NewStreamEncoderWithOptions(buf, opts)
			if err := enc.WriteHeader(doc.Header); err != nil {
				return err
			}
			for _, r := range doc.Records {
				if err := enc.WriteRecord(r); err != nil {
					return err
				}
			}
			if err := enc.WriteTrailer(); err != nil {
				return err
			}
			return enc.Close()
		}
	}

	tests := []struct {
		name   string
		header gedcom.Version
		opts   *EncodeOptions
		encode func(*EncodeOptions) func(*bytes.Buffer, *gedcom.Document) error
	}{
		{"header version", gedcom.Version70, DefaultOptions(), encodeWith},
		{"target version", gedcom.Version551, &EncodeOptions{TargetVersion: gedcom.Version70}, encodeWith},
		{"short max line length", gedcom.Version70, &EncodeOptions{MaxLineLength: 20}, encodeWith},
		{"streaming header version", gedcom.Version70, DefaultOptions(), streamWith},
		{"streaming target version", gedcom.Version55, &EncodeOptions{TargetVersion: gedcom.Version70}, streamWith},
		{"EncodeStreamingWithOptions", gedcom.Version70, DefaultOptions(), func(opts *EncodeOptions) func(*bytes.Buffer, *gedcom.Document) error {
			return func(buf *bytes.Buffer, doc *gedcom.Document) error {
				return EncodeStreamingWithOptions(buf, doc, opts)
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := *tt.opts
			var buf bytes.Buffer
			if err := tt.encode(tt.opts)(&buf, lineWrapDoc(tt.header)); err != nil {
				t.Fatalf("encode: %v", err)
			}
			if strings.Contains(buf.String(), " CONC") {
				t.Errorf("7.0 output contains CONC:\n%s", buf.String())
			}
			if *tt.opts != before {
				t.Errorf("options changed by encoding: got %+v, want %+v", *tt.opts, before)
			}
			if got := decodedNote(t, buf.String()); got != longNoteText {
				t.Errorf("note round-trips as %q, want %q", got, longNoteText)
			}
		})
	}
}

// TestEncode5xCONCSplitsBetweenNonSpaces asserts 5.x output still splits long
// values with CONC, with no CONC line or the line before it ending or starting
// in a space, and that the value round-trips (issue #565).
func TestEncode5xCONCSplitsBetweenNonSpaces(t *testing.T) {
	for _, version := range []gedcom.Version{gedcom.Version55, gedcom.Version551} {
		t.Run(string(version), func(t *testing.T) {
			var buf bytes.Buffer
			if err := EncodeWithOptions(&buf, lineWrapDoc(version), &EncodeOptions{MaxLineLength: 100}); err != nil {
				t.Fatalf("encode: %v", err)
			}
			lines := strings.Split(buf.String(), "\n")
			concs := 0
			for i, line := range lines {
				if !strings.HasPrefix(line, "2 CONC ") {
					continue
				}
				concs++
				if strings.HasPrefix(strings.TrimPrefix(line, "2 CONC "), " ") {
					t.Errorf("CONC value starts with a space: %q", line)
				}
				if strings.HasSuffix(lines[i-1], " ") {
					t.Errorf("line before CONC ends with a space: %q", lines[i-1])
				}
			}
			if concs == 0 {
				t.Fatalf("expected CONC lines in %s output:\n%s", version, buf.String())
			}
			if got := decodedNote(t, buf.String()); got != longNoteText {
				t.Errorf("note round-trips as %q, want %q", got, longNoteText)
			}
		})
	}
}
