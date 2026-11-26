package gedcom_test

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// continuedPlaceSource has a PLAC continued on a CONC line, with a MAP beneath
// it, in the layout a normalized encode would rewrite: CRLF terminators,
// indentation by level and blank lines between records.
const continuedPlaceSource = "0 HEAD\r\n" +
	"  1 GEDC\r\n" +
	"    2 VERS 5.5.1\r\n" +
	"  1 CHAR UTF-8\r\n" +
	"\r\n" +
	"0 @I1@ INDI\r\n" +
	"  1 NAME Ann /Example/\r\n" +
	"  1 BIRT\r\n" +
	"    2 PLAC Springfield, Sangamon Co\r\n" +
	"      3 CONC unty, Illinois\r\n" +
	"      3 MAP\r\n" +
	"        4 LATI N39.8\r\n" +
	"        4 LONG W89.6\r\n" +
	"    2 DATE 1 JAN 1900\r\n" +
	"  1 DEAT\r\n" +
	"    2 PLAC Chicago\r\n" +
	"\r\n" +
	"0 TRLR\r\n"

func decodeContinuedPlace(t *testing.T) (*gedcom.Document, *gedcom.Record) {
	t.Helper()
	doc, err := decoder.DecodeWithOptions(strings.NewReader(continuedPlaceSource),
		&decoder.DecodeOptions{PreserveLayout: true})
	if err != nil {
		t.Fatal(err)
	}
	return doc, doc.XRefMap["@I1@"]
}

// TestFullValue checks that a continued PLAC reads as its joined value and an
// uncontinued one as its Value.
func TestFullValue(t *testing.T) {
	_, rec := decodeContinuedPlace(t)

	if got, want := rec.FullValue(rec.Find("BIRT", "PLAC")), "Springfield, Sangamon County, Illinois"; got != want {
		t.Errorf("continued PLAC: FullValue = %q, want %q", got, want)
	}
	if got, want := rec.FullValue(rec.Find("DEAT", "PLAC")), "Chicago"; got != want {
		t.Errorf("uncontinued PLAC: FullValue = %q, want %q", got, want)
	}
}

// TestFullValueEdges covers CONT, continuation lines that are not t's own,
// and a tag the record does not hold.
func TestFullValueEdges(t *testing.T) {
	note := &gedcom.Tag{Level: 1, Tag: "NOTE", Value: "first"}
	rec := &gedcom.Record{Tags: []*gedcom.Tag{
		note,
		{Level: 2, Tag: "CONT", Value: "second"},
		nil,
		{Level: 2, Tag: "SOUR", Value: "@S1@"},
		{Level: 3, Tag: "CONC", Value: " (not the note's)"},
		{Level: 2, Tag: "CONC", Value: " line"},
		{Level: 1, Tag: "CONC", Value: "not beneath the note"},
	}}
	if got, want := rec.FullValue(note), "first\nsecond line"; got != want {
		t.Errorf("FullValue = %q, want %q", got, want)
	}

	stray := &gedcom.Tag{Level: 1, Tag: "PLAC", Value: "elsewhere"}
	var none *gedcom.Record
	tests := []struct {
		name string
		rec  *gedcom.Record
		tag  *gedcom.Tag
		want string
	}{
		{"tag not in record", rec, stray, "elsewhere"},
		{"nil record", none, stray, "elsewhere"},
		{"nil tag", rec, nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rec.FullValue(tt.tag); got != tt.want {
				t.Errorf("FullValue = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestSetFullValue checks that the setter replaces the value and its
// continuation lines, keeps the MAP beneath it and the tag's LineNumber, and
// leaves the record's other tags as they were.
func TestSetFullValue(t *testing.T) {
	_, rec := decodeContinuedPlace(t)
	plac := rec.Find("BIRT", "PLAC")
	line := plac.LineNumber
	mapTag := rec.Find("BIRT", "PLAC", "MAP")
	before := len(rec.Tags)

	if !rec.SetFullValue(plac, "Springfield, Illinois", gedcom.Version551) {
		t.Fatal("SetFullValue = false for a tag in the record")
	}
	if plac.Value != "Springfield, Illinois" || plac.LineNumber != line {
		t.Errorf("PLAC = %q at line %d, want %q at line %d", plac.Value, plac.LineNumber, "Springfield, Illinois", line)
	}
	if got := rec.FindAll("BIRT", "PLAC", "CONC"); len(got) != 0 {
		t.Errorf("old CONC lines kept: %d", len(got))
	}
	if rec.Find("BIRT", "PLAC", "MAP") != mapTag || len(rec.Subtree(mapTag)) != 3 {
		t.Error("MAP and its LATI and LONG were not kept")
	}
	if len(rec.Tags) != before-1 {
		t.Errorf("len(Tags) = %d, want %d", len(rec.Tags), before-1)
	}

	// Setting a value with a line break adds a CONT, placed before the MAP.
	if !rec.SetFullValue(plac, "Springfield\nIllinois", gedcom.Version551) {
		t.Fatal("SetFullValue = false")
	}
	sub := rec.Subtree(plac)
	if sub[1].Tag != "CONT" || sub[1].Value != "Illinois" || sub[1].Level != 3 || sub[1].LineNumber != 0 {
		t.Errorf("continuation = %+v, want a new level-3 CONT Illinois", *sub[1])
	}
	if sub[2] != mapTag {
		t.Errorf("tag after the CONT = %+v, want the MAP", *sub[2])
	}
	if got := rec.FullValue(plac); got != "Springfield\nIllinois" {
		t.Errorf("FullValue after set = %q", got)
	}
}

// TestSetFullValueSplitsAsEncoder checks that a long, multi-line value is
// split into the same CONT and CONC lines the encoder writes for a typed
// value in 5.5.1, and that 7.0 cuts no CONC.
func TestSetFullValueSplitsAsEncoder(t *testing.T) {
	value := strings.Repeat("Springfield ", 30) + "\n" + strings.Repeat("Sangamon ", 40)

	doc := gedcom.NewDocument(gedcom.Version551)
	indi := &gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual,
		Entity: &gedcom.Individual{XRef: "@I1@", InlineNotes: []string{value}}}
	if err := doc.AddRecord(indi); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := encoder.Encode(&out, doc); err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, line := range strings.Split(out.String(), "\n") {
		line = strings.TrimSuffix(line, "\r")
		if after, ok := strings.CutPrefix(line, "1 NOTE "); ok {
			want = append(want, "PLAC "+after)
		} else if after, ok := strings.CutPrefix(line, "2 CON"); ok {
			want = append(want, "CON"+after)
		}
	}
	if len(want) < 4 {
		t.Fatalf("encoder wrote %q, want a NOTE with CONT and CONC lines", want)
	}

	_, rec := decodeContinuedPlace(t)
	plac := rec.Find("BIRT", "PLAC")
	rec.SetFullValue(plac, value, gedcom.Version551)
	var got []string
	for _, tag := range rec.Subtree(plac) {
		if tag.Tag != "MAP" && tag.Level <= plac.Level+1 {
			got = append(got, tag.Tag+" "+tag.Value)
		}
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("SetFullValue lines:\n%q\nencoder lines:\n%q", got, want)
	}
	if rec.FullValue(plac) != value {
		t.Error("FullValue does not read back the value set")
	}

	rec.SetFullValue(plac, value, gedcom.Version70)
	if concs := rec.FindAll("BIRT", "PLAC", "CONC"); len(concs) != 0 {
		t.Errorf("7.0: %d CONC lines, want none", len(concs))
	}
	if rec.FullValue(plac) != value {
		t.Error("7.0: FullValue does not read back the value set")
	}
}

// TestSetFullValueNotInRecord checks that a tag the record does not hold, a
// nil tag and a nil record change nothing.
func TestSetFullValueNotInRecord(t *testing.T) {
	_, rec := decodeContinuedPlace(t)
	before := len(rec.Tags)
	stray := &gedcom.Tag{Level: 2, Tag: "PLAC", Value: "elsewhere"}
	var none *gedcom.Record

	if rec.SetFullValue(stray, "x", gedcom.Version551) || stray.Value != "elsewhere" {
		t.Error("SetFullValue changed a tag not in the record")
	}
	if rec.SetFullValue(nil, "x", gedcom.Version551) {
		t.Error("SetFullValue(nil) = true")
	}
	if none.SetFullValue(stray, "x", gedcom.Version551) {
		t.Error("nil record: SetFullValue = true")
	}
	if len(rec.Tags) != before {
		t.Errorf("len(Tags) = %d, want %d", len(rec.Tags), before)
	}
}

// TestSetFullValuePreservesLayout checks that, with a layout-preserving
// encode, only the rewritten PLAC and its continuation lines change: every
// other line is written byte for byte as read, MAP included. The PLAC keeps its
// source indentation; the new CONT, which has no source line, has none.
func TestSetFullValuePreservesLayout(t *testing.T) {
	doc, rec := decodeContinuedPlace(t)
	rec.SetFullValue(rec.Find("BIRT", "PLAC"), "Springfield,\nIllinois", doc.Header.Version)

	var out bytes.Buffer
	if err := encoder.EncodeWithOptions(&out, doc, &encoder.EncodeOptions{PreserveLayout: true}); err != nil {
		t.Fatal(err)
	}
	oldLines := "    2 PLAC Springfield, Sangamon Co\r\n      3 CONC unty, Illinois\r\n"
	newLines := "    2 PLAC Springfield,\r\n3 CONT Illinois\r\n"
	want := strings.Replace(continuedPlaceSource, oldLines, newLines, 1)
	if got := out.String(); got != want {
		t.Errorf("encoded:\n%q\nwant:\n%q", got, want)
	}
}

// TestSetFullValueBoundsWholeLine checks that SetFullValue cuts a value so a
// line with a long tag still fits 255 bytes, as the encoder does (#967).
func TestSetFullValueBoundsWholeLine(t *testing.T) {
	tag := &gedcom.Tag{Level: 1, Tag: "_" + strings.Repeat("T", 60)}
	rec := &gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Tags: []*gedcom.Tag{tag}}
	value := strings.Repeat("Springfield ", 60)
	rec.SetFullValue(tag, value, gedcom.Version551)
	if len(rec.Tags) < 3 {
		t.Fatalf("got %d tags, want the value split onto CONC lines", len(rec.Tags))
	}
	for _, c := range rec.Tags {
		if n := len(strconv.Itoa(c.Level)) + 1 + len(c.Tag) + 1 + len(c.Value); n > 255 {
			t.Errorf("%s line is %d bytes, want at most 255", c.Tag, n)
		}
	}
	if rec.FullValue(tag) != value {
		t.Error("FullValue does not read back the value set")
	}
}
