package converter

import (
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// A nil header converted to the version it already defaults to took the
// no-conversion early return, which skipped the header materialisation and
// handed back a document whose Header was still nil. The guarantee is
// equivalence with the empty header a nil one is read as -- not a stamped
// version, which this path does not apply to an empty header either.
func TestConvertWithOptions_NilHeaderEveryTarget(t *testing.T) {
	for _, target := range []gedcom.Version{gedcom.Version55, gedcom.Version551, gedcom.Version70} {
		t.Run(string(target), func(t *testing.T) {
			nilHeader := &gedcom.Document{Records: []*gedcom.Record{}}
			emptyHeader := &gedcom.Document{Header: &gedcom.Header{}, Records: []*gedcom.Record{}}

			got, _, err := ConvertWithOptions(nilHeader, target, DefaultOptions())
			if err != nil {
				t.Fatalf("ConvertWithOptions() error = %v", err)
			}
			want, _, err := ConvertWithOptions(emptyHeader, target, DefaultOptions())
			if err != nil {
				t.Fatalf("ConvertWithOptions() with empty header error = %v", err)
			}

			if got.Header == nil {
				t.Fatal("converted document has a nil Header")
			}
			if got.Header.Version != want.Header.Version {
				t.Errorf("Header.Version = %q, want %q (must match the empty-header twin)",
					got.Header.Version, want.Header.Version)
			}
			if nilHeader.Header != nil {
				t.Error("ConvertWithOptions() mutated the caller's document")
			}
		})
	}
}

// consolidateCONCOnlyInTags and convertCONTOnlyInTags sit off the conversion
// routes -- consolidateCONCAndCONT and expandNewlinesToCONT are what the
// version paths call -- so their nil handling only gets exercised directly.
func TestCONCAndCONTOnlyRebuilders_PreserveNils(t *testing.T) {
	t.Run("consolidateCONCOnlyInTags", func(t *testing.T) {
		tags := []*gedcom.Tag{
			nil,
			{Level: 1, Tag: "NOTE", Value: "first "},
			nil,
			{Level: 2, Tag: "CONC", Value: "second"},
			nil,
		}
		result, concCount := consolidateCONCOnlyInTags(tags)

		if concCount != 1 {
			t.Errorf("concCount = %d, want 1 (a nil must not break the continuation)", concCount)
		}
		if got := countNils(result); got != 3 {
			t.Errorf("nils in result = %d, want 3 (every nil is preserved)", got)
		}
		note := firstNote(result)
		if note == nil || note.Value != "first second" {
			t.Errorf("NOTE value = %#v, want %q", note, "first second")
		}
	})

	t.Run("convertCONTOnlyInTags", func(t *testing.T) {
		tags := []*gedcom.Tag{
			nil,
			{Level: 1, Tag: "NOTE", Value: "line one"},
			nil,
			{Level: 2, Tag: "CONT", Value: "line two"},
			nil,
		}
		result, contCount := convertCONTOnlyInTags(tags)

		if contCount != 1 {
			t.Errorf("contCount = %d, want 1 (a nil must not break the continuation)", contCount)
		}
		if got := countNils(result); got != 3 {
			t.Errorf("nils in result = %d, want 3 (every nil is preserved)", got)
		}
		note := firstNote(result)
		if note == nil || note.Value != "line one\nline two" {
			t.Errorf("NOTE value = %#v, want %q", note, "line one\nline two")
		}
	})

	t.Run("nil-only input", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			run  func([]*gedcom.Tag) ([]*gedcom.Tag, int)
		}{
			{"CONC", consolidateCONCOnlyInTags},
			{"CONT", convertCONTOnlyInTags},
		} {
			result, count := tc.run([]*gedcom.Tag{nil, nil})
			if count != 0 {
				t.Errorf("%s: count = %d, want 0", tc.name, count)
			}
			if len(result) != 2 || countNils(result) != 2 {
				t.Errorf("%s: result = %#v, want two preserved nils", tc.name, result)
			}
		}
	})
}

// A nil record must not stop these two rebuilders either; they range
// Document.Records separately from the conversion routes.
func TestCONCAndCONTOnlyDocumentSweeps_SkipNilRecords(t *testing.T) {
	build := func() *gedcom.Document {
		return &gedcom.Document{
			Header: &gedcom.Header{Version: gedcom.Version551, Tags: []*gedcom.Tag{nil}},
			Records: []*gedcom.Record{
				nil,
				{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Tags: []*gedcom.Tag{
					{Level: 1, Tag: "NOTE", Value: "first "},
					{Level: 2, Tag: "CONC", Value: "second"},
					{Level: 2, Tag: "CONT", Value: "third"},
				}},
			},
		}
	}

	concDoc, contDoc := build(), build()
	consolidateCONC(concDoc, &gedcom.ConversionReport{})
	convertCONTToNewlines(contDoc, &gedcom.ConversionReport{})

	if note := firstNote(concDoc.Records[1].Tags); note == nil || note.Value != "first second" {
		t.Errorf("consolidateCONC NOTE = %#v, want %q", note, "first second")
	}
	if note := firstNote(contDoc.Records[1].Tags); note == nil || note.Value != "first \nthird" {
		t.Errorf("convertCONTToNewlines NOTE = %#v, want %q", note, "first \nthird")
	}
}

func countNils(tags []*gedcom.Tag) int {
	n := 0
	for _, tag := range tags {
		if tag == nil {
			n++
		}
	}
	return n
}

// firstNote returns the NOTE tag the continuation rebuilders merge into.
func firstNote(tags []*gedcom.Tag) *gedcom.Tag {
	for _, tag := range tags {
		if tag != nil && tag.Tag == "NOTE" {
			return tag
		}
	}
	return nil
}

// countTagsInRecord walks a tag slice directly rather than through a route, so
// its nil skip only gets exercised by a direct call.
func TestCountTagsInRecord_SkipsNilTags(t *testing.T) {
	found := map[string]int{}
	countTagsInRecord([]*gedcom.Tag{
		nil,
		{Level: 1, Tag: "EMAIL", Value: "a@b.c"},
		nil,
		{Level: 1, Tag: "EMAIL", Value: "d@e.f"},
	}, []string{"EMAIL", "FAX"}, found)

	if found["EMAIL"] != 2 {
		t.Errorf("EMAIL count = %d, want 2", found["EMAIL"])
	}
	if _, ok := found["FAX"]; ok {
		t.Errorf("FAX counted %d times, want absent", found["FAX"])
	}
}
