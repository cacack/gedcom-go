package encoder

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// atSignVersions are the output versions whose "@" escaping differs (#708).
var atSignVersions = []gedcom.Version{gedcom.Version551, gedcom.Version70}

// encodeDecode encodes doc for version v and decodes the result.
func encodeDecode(t *testing.T, doc *gedcom.Document, v gedcom.Version) (string, *gedcom.Document) {
	t.Helper()
	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, doc, &EncodeOptions{TargetVersion: v, LineEnding: "\n"}); err != nil {
		t.Fatalf("Encode(%s) error = %v", v, err)
	}
	out := buf.String()
	got, err := decoder.Decode(strings.NewReader(out))
	if err != nil {
		t.Fatalf("Decode(%s) error = %v\n%s", v, err, out)
	}
	return out, got
}

// TestInlineNotePointerShapedRoundTrip is the case #708 was filed for: inline
// note text shaped like a pointer is written escaped, so it re-decodes as the
// same text and not as a link to whatever record carries that identifier.
func TestInlineNotePointerShapedRoundTrip(t *testing.T) {
	wantLine := map[gedcom.Version]string{
		gedcom.Version551: "1 NOTE @@I1@@",
		gedcom.Version70:  "1 NOTE @@I1@",
	}
	for _, v := range atSignVersions {
		t.Run(string(v), func(t *testing.T) {
			doc := &gedcom.Document{Records: []*gedcom.Record{{
				XRef:   "@I1@",
				Type:   gedcom.RecordTypeIndividual,
				Entity: &gedcom.Individual{XRef: "@I1@", InlineNotes: []string{"@I1@"}},
			}}}
			out, got := encodeDecode(t, doc, v)
			if !strings.Contains(out, "\n"+wantLine[v]+"\n") {
				t.Errorf("output lacks %q:\n%s", wantLine[v], out)
			}
			indi := got.GetIndividual("@I1@")
			if indi == nil {
				t.Fatalf("individual missing:\n%s", out)
			}
			if !reflect.DeepEqual(indi.InlineNotes, []string{"@I1@"}) {
				t.Errorf("InlineNotes = %q, want [\"@I1@\"]", indi.InlineNotes)
			}
			if len(indi.NoteXRefs) != 0 {
				t.Errorf("NoteXRefs = %q, want empty", indi.NoteXRefs)
			}
		})
	}
}

// TestTypedTextAtSignRoundTrip covers the escaping rules field by field: a
// leading "@" in 7.0, every "@" in 5.5.1, a CONT line's own leading "@", and a
// 5.5.1 calendar escape, which is not doubled. Pointers are written verbatim.
func TestTypedTextAtSignRoundTrip(t *testing.T) {
	const page = "@@start, cost @ 5, a@b.com"
	tests := []struct {
		v     gedcom.Version
		lines []string
	}{
		{gedcom.Version551, []string{
			"2 PAGE @@@@start, cost @@ 5, a@@b.com",
			"1 NOTE first",
			"2 CONT @@second@@",
			"2 DATE @#DJULIAN@ 1 JAN 1700",
			"1 FAMC @F1@",
		}},
		{gedcom.Version70, []string{
			"2 PAGE @@@start, cost @ 5, a@b.com",
			"1 NOTE first",
			"2 CONT @@second@",
			"1 FAMC @F1@",
		}},
	}
	for _, tt := range tests {
		t.Run(string(tt.v), func(t *testing.T) {
			indi := &gedcom.Individual{
				XRef:            "@I1@",
				SourceCitations: []*gedcom.SourceCitation{{SourceXRef: "@S1@", Page: page}},
				InlineNotes:     []string{"first\n@second@"},
				ChildInFamilies: []gedcom.FamilyLink{{FamilyXRef: "@F1@"}},
			}
			if tt.v == gedcom.Version551 {
				indi.Events = []*gedcom.Event{{Type: gedcom.EventBirth, Date: "@#DJULIAN@ 1 JAN 1700"}}
			}
			doc := &gedcom.Document{Records: []*gedcom.Record{
				{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: indi},
			}}
			out, got := encodeDecode(t, doc, tt.v)
			for _, line := range tt.lines {
				if !strings.Contains(out, "\n"+line+"\n") {
					t.Errorf("output lacks %q:\n%s", line, out)
				}
			}
			again := got.GetIndividual("@I1@")
			if again == nil {
				t.Fatalf("individual missing:\n%s", out)
			}
			if p := again.SourceCitations[0].Page; p != page {
				t.Errorf("Page = %q, want %q", p, page)
			}
			if !reflect.DeepEqual(again.InlineNotes, indi.InlineNotes) {
				t.Errorf("InlineNotes = %q, want %q", again.InlineNotes, indi.InlineNotes)
			}
			if tt.v == gedcom.Version551 && again.Events[0].Date != indi.Events[0].Date {
				t.Errorf("Date = %q, want %q", again.Events[0].Date, indi.Events[0].Date)
			}
		})
	}
}

// TestHeaderFieldsAtSign covers the hand-built header path, whose SOUR, DATE
// and LANG are text like any typed field.
func TestHeaderFieldsAtSign(t *testing.T) {
	for _, v := range atSignVersions {
		t.Run(string(v), func(t *testing.T) {
			doc := &gedcom.Document{Header: &gedcom.Header{SourceSystem: "@APP@"}}
			out, got := encodeDecode(t, doc, v)
			if strings.Contains(out, "\n1 SOUR @APP@\n") {
				t.Errorf("SOUR written unescaped:\n%s", out)
			}
			if got.Header.SourceSystem != "@APP@" {
				t.Errorf("SourceSystem = %q, want %q", got.Header.SourceSystem, "@APP@")
			}
		})
	}
}

// textSentinel is pointer-shaped text. pointerSentinel replaces it in every
// pointer field, so after a round trip any visited pointer equal to
// textSentinel is text that came back as a link.
const (
	textSentinel    = "@T1@"
	pointerSentinel = "@P1@"
)

// TestTypedTextNeverDecodesAsPointer asserts, for every string field of every
// record entity the encoder writes, that a typed text value never re-decodes
// as a pointer (#708). Every plain string field is filled with pointer-shaped
// text by reflection, so a field added later is covered without listing it;
// gedcom.Apply then moves every pointer field to a real pointer, since a
// pointer field is the one place that shape is meant to be written as is.
func TestTypedTextNeverDecodesAsPointer(t *testing.T) {
	entities := map[gedcom.RecordType]any{
		gedcom.RecordTypeIndividual: &gedcom.Individual{},
		gedcom.RecordTypeFamily:     &gedcom.Family{},
		gedcom.RecordTypeSource:     &gedcom.Source{},
		gedcom.RecordTypeSubmitter:  &gedcom.Submitter{},
		gedcom.RecordTypeSubmission: &gedcom.Submission{},
		gedcom.RecordTypeRepository: &gedcom.Repository{},
		gedcom.RecordTypeNote:       &gedcom.Note{},
		gedcom.RecordTypeMedia:      &gedcom.MediaObject{},
		gedcom.RecordTypeSharedNote: &gedcom.SharedNote{},
	}
	for _, v := range atSignVersions {
		for recType, entity := range entities {
			t.Run(string(v)+"/"+string(recType), func(t *testing.T) {
				e := reflect.New(reflect.TypeOf(entity).Elem())
				fillStrings(e.Elem(), 0)
				doc := &gedcom.Document{Records: []*gedcom.Record{{XRef: "@R1@", Type: recType, Entity: e.Interface()}}}
				gedcom.Apply(doc, map[string]string{textSentinel: pointerSentinel})

				out, got := encodeDecode(t, doc, v)
				if !strings.Contains(out, textSentinel[1:len(textSentinel)-1]) {
					t.Fatalf("no text field was written:\n%s", out)
				}
				for _, r := range got.Records {
					gedcom.VisitForVersion(r, v, func(ref string) {
						if ref == textSentinel {
							t.Errorf("text %s re-decoded as a pointer:\n%s", textSentinel, out)
						}
					})
				}
			})
		}
	}
}

// fillStrings sets every plain string in v to textSentinel, allocating
// pointers and giving each slice one element so nested structures are reached.
// Named string types (gedcom.EventType and the like) are tag names, not text,
// and are left empty; so are Tags fields, which are raw and written verbatim.
func fillStrings(v reflect.Value, depth int) {
	if depth > 8 {
		return
	}
	switch v.Kind() {
	case reflect.String:
		switch {
		case v.Type() == reflect.TypeOf(""):
			v.SetString(textSentinel)
		// An event's, attribute's or ordinance's type is its tag name, which the encoder
		// rejects empty (#726).
		case v.Type() == reflect.TypeOf(gedcom.EventType("")):
			v.SetString(string(gedcom.EventBirth))
		case v.Type() == reflect.TypeOf(gedcom.AttributeType("")):
			v.SetString(string(gedcom.AttributeOccupation))
		case v.Type() == reflect.TypeOf(gedcom.LDSOrdinanceType("")):
			v.SetString(string(gedcom.LDSBaptism))
		}
	case reflect.Pointer:
		if v.Type().Elem().Kind() == reflect.Struct {
			v.Set(reflect.New(v.Type().Elem()))
			fillStrings(v.Elem(), depth+1)
		}
	case reflect.Slice:
		s := reflect.MakeSlice(v.Type(), 1, 1)
		fillStrings(s.Index(0), depth+1)
		v.Set(s)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			if !f.IsExported() || f.Name == "Tags" || f.Name == "XRef" {
				continue
			}
			fillStrings(v.Field(i), depth+1)
		}
	}
}

// TestAtSignWrapFitsLineLength asserts that 5.5.1 escaping, which doubles every
// "@", is measured before the CONC split: an "@"-dense note stays within the
// line limit, no line holds an odd "@" run, and the text round-trips.
func TestAtSignWrapFitsLineLength(t *testing.T) {
	for _, note := range []string{strings.Repeat("@", 300), strings.Repeat("a@", 200), strings.Repeat("x", 247) + "@@y"} {
		doc := &gedcom.Document{Records: []*gedcom.Record{{
			XRef:   "@I1@",
			Type:   gedcom.RecordTypeIndividual,
			Entity: &gedcom.Individual{XRef: "@I1@", InlineNotes: []string{note}},
		}}}
		out, got := encodeDecode(t, doc, gedcom.Version551)
		for _, line := range strings.Split(out, "\n") {
			// 255 is the 5.5.1 line limit; DefaultMaxLineLength bounds the value.
			if len(line) > 255 {
				t.Errorf("line is %d bytes, want at most 255: %q", len(line), line)
			}
			if _, value, ok := strings.Cut(line, " CONC "); ok && strings.HasPrefix(value, "@") && !strings.HasPrefix(value, "@@") {
				t.Errorf("CONC line begins with a lone @: %q", line)
			}
		}
		if indi := got.GetIndividual("@I1@"); indi == nil || !reflect.DeepEqual(indi.InlineNotes, []string{note}) {
			t.Errorf("note did not round-trip:\n%s", out)
		}
	}
}

// TestUndeclaredVersionRoundTrip covers a file with no GEDC.VERS: the decoder
// unescapes by the detected version, the one the encoder writes it as, so an
// unedited decode-encode keeps a mid-value "@@" as written.
func TestUndeclaredVersionRoundTrip(t *testing.T) {
	input := "0 HEAD\n1 SOUR a@@b\n0 @N1@ NOTE x@@y\n0 TRLR\n"
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got := doc.GetNote("@N1@").Text; got != "x@y" {
		t.Errorf("Note.Text = %q, want %q", got, "x@y")
	}
	if got := doc.Header.SourceSystem; got != "a@b" {
		t.Errorf("SourceSystem = %q, want %q", got, "a@b")
	}
	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, doc, &EncodeOptions{LineEnding: "\n"}); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	for _, want := range []string{"\n1 SOUR a@@b\n", "\n0 @N1@ NOTE x@@y\n"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("output lacks %q:\n%s", want, buf.String())
		}
	}
}

// TestMultiLineNoteReEscaped pins the limit of the NOTE record's verbatim
// path: only a one-line Text keeps Record.Value as written. A multi-line Text
// is written from the entity, so a lone "@" a 5.5.1 producer left undoubled is
// re-escaped -- the same text, in its spec form.
func TestMultiLineNoteReEscaped(t *testing.T) {
	input := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @N1@ NOTE a@b\n0 @N2@ NOTE a@b\n1 CONT c\n0 TRLR\n"
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	for _, r := range doc.Records {
		r.Tags = nil
	}
	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, doc, &EncodeOptions{LineEnding: "\n"}); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	for _, want := range []string{"\n0 @N1@ NOTE a@b\n", "\n0 @N2@ NOTE a@@b\n1 CONT c\n"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("output lacks %q:\n%s", want, buf.String())
		}
	}
}
