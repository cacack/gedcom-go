package converter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// A 7.0 → 5.x downgrade splits over-length values with CONC (#671). 7.0 has
// no line limit and no CONC, so a decoded 7.0 note keeps its whole text on one
// line, and the encoder writes raw tags verbatim: without the split, the 5.x
// output carried lines past the 255-character limit.

// longNoteText is text over the 248-byte split limit with spaces where a cut
// could fall, so a split that lands beside a space would lose it to a reader
// that trims lines.
func longNoteText(prefix string) string {
	return prefix + strings.Repeat("word ", 60) + strings.Repeat("日本語 ", 30) + "end "
}

func TestDowngradeSplitsLongValuesWithCONC(t *testing.T) {
	inline := longNoteText("Inline")
	inlineCont := longNoteText("Second line")
	shared := longNoteText("Shared")
	sharedCont := longNoteText("Shared second")
	nested := longNoteText("Nested")
	// A one-line shared note decodes with no subordinate tags, only an entity.
	single := longNoteText("Single")

	// A hand-built 7.0 document, encoded and decoded so the downgrade sees
	// what a caller reading a 7.0 file sees.
	src := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version70},
		Records: []*gedcom.Record{
			{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{
				XRef:        "@I1@",
				InlineNotes: []string{inline + "\n" + inlineCont},
				Names: []*gedcom.PersonalName{{
					Full:        "John /Doe/",
					InlineNotes: []string{nested},
				}},
			}},
			{XRef: "@N1@", Type: gedcom.RecordTypeSharedNote, Entity: &gedcom.SharedNote{
				XRef: "@N1@",
				Text: shared + "\n" + sharedCont,
			}},
			{XRef: "@N2@", Type: gedcom.RecordTypeSharedNote, Entity: &gedcom.SharedNote{
				XRef: "@N2@",
				Text: single,
			}},
		},
	}
	var v7 bytes.Buffer
	if err := encoder.Encode(&v7, src); err != nil {
		t.Fatalf("encode 7.0: %v", err)
	}
	if conc, _ := countContinuations(v7.String()); conc != 0 {
		t.Fatalf("7.0 output has %d CONC lines, want 0", conc)
	}

	for _, target := range []gedcom.Version{gedcom.Version55, gedcom.Version551} {
		t.Run(target.String(), func(t *testing.T) {
			doc, err := decoder.Decode(bytes.NewReader(v7.Bytes()))
			if err != nil {
				t.Fatalf("decode 7.0: %v", err)
			}
			converted, report, err := Convert(doc, target)
			if err != nil {
				t.Fatalf("convert: %v", err)
			}
			var out bytes.Buffer
			if err := encoder.Encode(&out, converted); err != nil {
				t.Fatalf("encode %s: %v", target, err)
			}

			for i, line := range strings.Split(out.String(), "\n") {
				if n := len([]rune(line)); n > 255 {
					t.Errorf("line %d has %d characters, want at most 255: %.60q...", i+1, n, line)
				}
			}
			outCONC, _ := countContinuations(out.String())
			if outCONC == 0 {
				t.Fatalf("output has no CONC lines:\n%s", out.String())
			}
			var reported int
			for _, tr := range report.Transformations {
				if tr.Type == "CONC_ADDED" {
					reported = tr.Count
				}
			}
			if reported != outCONC {
				t.Errorf("CONC_ADDED count = %d, output has %d CONC lines", reported, outCONC)
			}

			back, err := decoder.Decode(bytes.NewReader(out.Bytes()))
			if err != nil {
				t.Fatalf("decode %s: %v", target, err)
			}
			indi := back.GetIndividual("@I1@")
			if indi == nil {
				t.Fatal("individual @I1@ missing")
			}
			if want := inline + "\n" + inlineCont; len(indi.InlineNotes) != 1 || indi.InlineNotes[0] != want {
				t.Errorf("individual note = %q, want %q", indi.InlineNotes, want)
			}
			if len(indi.Names) != 1 || len(indi.Names[0].InlineNotes) != 1 || indi.Names[0].InlineNotes[0] != nested {
				t.Errorf("name note did not round-trip: %+v", indi.Names)
			}
			note := back.GetNote("@N1@")
			if note == nil {
				t.Fatal("note record @N1@ missing")
			}
			if want := shared + "\n" + sharedCont; note.Text != want {
				t.Errorf("note record text = %q, want %q", note.Text, want)
			}
			if note := back.GetNote("@N2@"); note == nil || note.Text != single {
				t.Errorf("one-line note record @N2@ did not round-trip: %+v", note)
			}
		})
	}
}

func TestExpandNewlinesInTagsSplitsContinuationAsSibling(t *testing.T) {
	long := strings.Repeat("a", 300)
	tags := []*gedcom.Tag{
		{Level: 1, Tag: "NOTE", Value: "short"},
		{Level: 2, Tag: "CONT", Value: long},
	}
	result, cont, conc, notes := expandNewlinesInTagsWithNotes(tags)
	if cont != 0 || conc != 1 || len(notes) != 1 || notes[0].reason != concSplitReason {
		t.Fatalf("cont=%d conc=%d notes=%+v, want 0, 1 and one CONC note", cont, conc, notes)
	}
	if len(result) != 3 || result[2].Tag != "CONC" || result[2].Level != 2 {
		t.Fatalf("CONC continuing a level-2 CONT must be a level-2 sibling, got %+v", result[2])
	}
	if got := result[1].Value + result[2].Value; got != long {
		t.Errorf("split pieces rebuild %q, want %q", got, long)
	}
}

func TestSplitLongRecordValue(t *testing.T) {
	long := strings.Repeat("b", 300)

	t.Run("short value is left for the encoder", func(t *testing.T) {
		record := &gedcom.Record{Type: gedcom.RecordTypeNote, Value: "one\ntwo"}
		if cont, conc := splitLongRecordValue(record); cont != 0 || conc != 0 {
			t.Errorf("cont=%d conc=%d, want 0, 0", cont, conc)
		}
		if record.Value != "one\ntwo" || len(record.Tags) != 0 {
			t.Errorf("record changed: %+v", record)
		}
	})

	t.Run("entity-only record is left alone", func(t *testing.T) {
		record := &gedcom.Record{Type: gedcom.RecordTypeSource, Value: long, Entity: &gedcom.Source{Title: "T"}}
		if _, conc := splitLongRecordValue(record); conc != 0 || record.Value != long {
			t.Errorf("conc=%d value len %d, want record untouched", conc, len(record.Value))
		}
	})

	t.Run("entity-only note is split", func(t *testing.T) {
		record := &gedcom.Record{Type: gedcom.RecordTypeNote, Value: long, Entity: &gedcom.Note{Text: long}}
		if _, conc := splitLongRecordValue(record); conc == 0 || record.Value+record.Tags[0].Value != long {
			t.Errorf("conc=%d tags=%d, want the value split onto CONC", conc, len(record.Tags))
		}
	})

	t.Run("continuations go before existing tags", func(t *testing.T) {
		record := &gedcom.Record{
			Type:  gedcom.RecordTypeNote,
			Value: long + "\nnext",
			Tags:  []*gedcom.Tag{{Level: 1, Tag: "CONT", Value: "last"}},
		}
		cont, conc := splitLongRecordValue(record)
		if cont != 1 || conc != 1 {
			t.Fatalf("cont=%d conc=%d, want 1, 1", cont, conc)
		}
		var got []string
		for _, tag := range record.Tags {
			got = append(got, tag.Tag)
		}
		if strings.Join(got, " ") != "CONC CONT CONT" {
			t.Errorf("tags = %v, want CONC CONT CONT", got)
		}
		if record.Value+record.Tags[0].Value != long {
			t.Error("value and CONC do not rebuild the first line")
		}
	})
}

// An XRef or tag that alone fills a 5.x line leaves no room for a value. The
// limit floors at 1 rather than going negative, which looped forever.
func TestMaxValueLengthFloorsAtOne(t *testing.T) {
	xref := "@" + strings.Repeat("A", 260) + "@"
	if got := maxValueLength(0, xref, "NOTE", 1); got != 1 {
		t.Errorf("maxValueLength = %d, want 1", got)
	}
	record := &gedcom.Record{XRef: xref, Type: gedcom.RecordTypeNote, Value: "ab"}
	if _, conc := splitLongRecordValue(record); conc != 1 || record.Value+record.Tags[0].Value != "ab" {
		t.Errorf("conc=%d, want the value split one byte per line", conc)
	}
	tags, _, _, _ := expandNewlinesInTagsWithNotes([]*gedcom.Tag{{Level: 1, XRef: xref, Tag: "NOTE"}})
	if len(tags) != 1 {
		t.Errorf("empty value gave %d tags, want 1", len(tags))
	}
}
