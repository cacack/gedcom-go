package converter_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/converter"
	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// downgradeFrom70 decodes input, converts it to target and encodes the result.
func downgradeFrom70(t *testing.T, input []byte, target gedcom.Version) (*gedcom.Document, *gedcom.ConversionReport, string) {
	t.Helper()
	doc, err := decoder.Decode(bytes.NewReader(input))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	converted, report, err := converter.Convert(doc, target)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	var buf bytes.Buffer
	if err := encoder.EncodeWithOptions(&buf, converted, &encoder.EncodeOptions{TargetVersion: target}); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return converted, report, buf.String()
}

// recordLineTags returns the tag of every line after the header, i.e. the
// lines written from records.
func recordLineTags(out string) map[string]bool {
	tags := make(map[string]bool)
	inHeader := false
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		tag := fields[1]
		if strings.HasPrefix(tag, "@") && len(fields) > 2 {
			tag = fields[2]
		}
		if fields[0] == "0" {
			inHeader = tag == "HEAD"
		}
		if !inHeader {
			tags[tag] = true
		}
	}
	return tags
}

// TestDowngrade70OutputMatchesReport pins issue #570: every 7.0-only tag the
// report calls dropped is absent from the 5.x output, and SNOTE is rewritten
// as NOTE rather than reported as lost.
func TestDowngrade70OutputMatchesReport(t *testing.T) {
	input, err := os.ReadFile("../testdata/gedcom-7.0/maximal70.ged")
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []gedcom.Version{gedcom.Version55, gedcom.Version551} {
		t.Run(target.String(), func(t *testing.T) {
			converted, report, out := downgradeFrom70(t, input, target)
			written := recordLineTags(out)

			for _, loss := range report.DataLoss {
				tag, ok := strings.CutSuffix(loss.Feature, " tags")
				if ok && written[tag] {
					t.Errorf("report drops %s but the output still writes it", tag)
				}
				if loss.Feature == "SNOTE tags" {
					t.Error("SNOTE reported as data loss; it should be rewritten as NOTE")
				}
			}
			for _, tag := range []string{"EXID", "TRAN", "PHRASE", "UID", "CREA", "SNOTE"} {
				if written[tag] {
					t.Errorf("output writes 7.0-only tag %s", tag)
				}
			}

			if !strings.Contains(out, "0 @N1@ NOTE Shared note 1\n") {
				t.Error("output lacks the shared note as a NOTE record")
			}
			if !strings.Contains(out, "1 NOTE @N1@\n") {
				t.Error("output lacks a NOTE pointer to the shared note")
			}
			record := converted.XRefMap["@N1@"]
			if record == nil || record.Type != gedcom.RecordTypeNote {
				t.Fatalf("@N1@ record = %+v, want NOTE", record)
			}
			if note, ok := record.Entity.(*gedcom.Note); !ok || note.Text != "Shared note 1" {
				t.Errorf("@N1@ entity = %#v, want *gedcom.Note with its text", record.Entity)
			}
			if transformationCount(report, "SNOTE_RECORD_TO_NOTE") != 2 {
				t.Errorf("SNOTE_RECORD_TO_NOTE count = %d, want 2", transformationCount(report, "SNOTE_RECORD_TO_NOTE"))
			}
			if transformationCount(report, "SNOTE_POINTER_TO_NOTE") == 0 {
				t.Error("SNOTE_POINTER_TO_NOTE not reported")
			}
		})
	}
}

// TestDowngrade70DropsTagBlocks checks that a dropped tag takes its whole
// subordinate block with it, at any level, while siblings, a nil element and
// the header are kept, and that a record with nothing to drop is untouched.
func TestDowngrade70DropsTagBlocks(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version: gedcom.Version70,
			Tags:    []*gedcom.Tag{{Level: 1, Tag: "SNOTE", Value: "@N1@"}},
		},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "NAME", Value: "John /Doe/"},
					{Level: 2, Tag: "TRAN", Value: "Johann /Doe/"},
					{Level: 3, Tag: "LANG", Value: "de"},
					nil,
					{Level: 2, Tag: "SURN", Value: "Doe"},
					{Level: 1, Tag: "CREA"},
					{Level: 2, Tag: "DATE", Value: "1 JAN 2020"},
					{Level: 1, Tag: "SNOTE", Value: "@N1@"},
				},
			},
			{XRef: "@I2@", Type: gedcom.RecordTypeIndividual, Tags: []*gedcom.Tag{{Level: 1, Tag: "NAME", Value: "Jane /Doe/"}}},
			{XRef: "@N1@", Type: gedcom.RecordTypeSharedNote, Value: "Shared"},
		},
	}

	converted, report, err := converter.Convert(doc, gedcom.Version551)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}

	var got []string
	for _, tag := range converted.Records[0].Tags {
		if tag == nil {
			got = append(got, "<nil>")
			continue
		}
		got = append(got, tag.Tag)
	}
	if want := "NAME <nil> SURN NOTE"; strings.Join(got, " ") != want {
		t.Errorf("@I1@ tags = %s, want %s", strings.Join(got, " "), want)
	}
	if tags := converted.Records[1].Tags; len(tags) != 1 || tags[0].Tag != "NAME" {
		t.Errorf("record with nothing to drop changed: %+v", tags)
	}
	for _, tag := range converted.Header.Tags {
		if tag != nil && tag.Value == "@N1@" && tag.Tag != "NOTE" {
			t.Errorf("header pointer to @N1@ = %s, want NOTE", tag.Tag)
		}
	}
	if converted.Records[2].Type != gedcom.RecordTypeNote {
		t.Errorf("@N1@ type = %s, want NOTE", converted.Records[2].Type)
	}

	lost := make(map[string][]string)
	for _, loss := range report.DataLoss {
		lost[loss.Feature] = loss.AffectedRecords
	}
	for _, feature := range []string{"TRAN tags", "CREA tags"} {
		if xrefs := lost[feature]; len(xrefs) != 1 || xrefs[0] != "@I1@" {
			t.Errorf("%s affected = %v, want [@I1@]", feature, xrefs)
		}
	}
	if transformationCount(report, "SNOTE_POINTER_TO_NOTE") != 2 {
		t.Errorf("SNOTE_POINTER_TO_NOTE count = %d, want 2", transformationCount(report, "SNOTE_POINTER_TO_NOTE"))
	}
}
