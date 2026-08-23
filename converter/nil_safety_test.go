package converter

import (
	"reflect"
	"sort"
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// The converter must never panic on a nil (ADR 0007). A decoded document never
// holds one, so every case here builds the document in memory -- the path a
// programmatic consumer uses. The contract is skip, not error: a conversion of a
// document holding nils must produce the same result as the same document with
// the nils removed, except that a nil *Tag stays in the rebuilt tag slice at its
// original position (the converter transforms tags, it does not shorten the
// caller's slice).

// nilFixture builds a document holding every nil shape the converter must
// survive: a nil *Record among real ones, nil *Tags adjacent to CONC and CONT
// tags on both sides, a nil inside an EXID's subordinate block, a nil header
// tag, and typed-nil entities.
func nilFixture(version gedcom.Version) *gedcom.Document {
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version: version,
			Tags: []*gedcom.Tag{
				nil, // leading header tag
				{Level: 1, Tag: "SOUR", Value: "TestApp"},
			},
		},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "NAME", Value: "John /Doe/"},
					{Level: 1, Tag: "NOTE", Value: "first "},
					{Level: 2, Tag: "CONC", Value: "second"},
					nil, // between two continuations: must not break the merge
					{Level: 2, Tag: "CONT", Value: "third"},
					nil, // after the last continuation
					{Level: 1, Tag: "EMAIL", Value: "john@example.com"},
				},
			},
			nil, // a nil record among real ones
			{
				XRef: "@I2@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					nil, // first tag: the rebuilders start on a nil
					{Level: 1, Tag: "NOTE", Value: "line one\nline two"},
					{Level: 1, Tag: "EXID", Value: "KWCJ-QN7"},
					nil, // inside the EXID's subordinate block
					{Level: 2, Tag: "TYPE", Value: "https://www.familysearch.org/ark"},
					{Level: 1, Tag: "UID", Value: "d4f1a8e0"},
					{Level: 1, Tag: "_VENDOR", Value: "kept"},
				},
			},
			{
				XRef:   "@I3@",
				Type:   gedcom.RecordTypeIndividual,
				Entity: (*gedcom.Individual)(nil), // typed-nil entity
			},
			{
				XRef:   "@O1@",
				Type:   gedcom.RecordTypeMedia,
				Entity: (*gedcom.MediaObject)(nil), // typed-nil entity on the media path
			},
		},
	}
	doc.XRefMap = make(map[string]*gedcom.Record, len(doc.Records))
	for _, record := range doc.Records {
		if record != nil {
			doc.XRefMap[record.XRef] = record
		}
	}
	return doc
}

// stripNils returns a copy of doc with every nil *Record and nil *Tag dropped
// and every typed-nil Entity replaced by a plain nil. Applied to the input it
// yields the nil-free document a conversion must match; applied to both results
// it makes that match assertable.
func stripNils(doc *gedcom.Document) *gedcom.Document {
	if doc == nil {
		return nil
	}
	out := doc.Clone()

	records := make([]*gedcom.Record, 0, len(out.Records))
	for _, record := range out.Records {
		if record == nil {
			continue
		}
		if record.Entity != nil && reflect.ValueOf(record.Entity).IsNil() {
			record.Entity = nil
		}
		record.Tags = nonNilTags(record.Tags)
		records = append(records, record)
	}
	out.Records = records

	if out.Header != nil {
		out.Header.Tags = nonNilTags(out.Header.Tags)
	}

	out.XRefMap = make(map[string]*gedcom.Record, len(out.Records))
	for _, record := range out.Records {
		if record.XRef != "" {
			out.XRefMap[record.XRef] = record
		}
	}
	return out
}

// nonNilTags returns tags without its nil elements, preserving a nil slice.
func nonNilTags(tags []*gedcom.Tag) []*gedcom.Tag {
	if tags == nil {
		return nil
	}
	out := make([]*gedcom.Tag, 0, len(tags))
	for _, tag := range tags {
		if tag != nil {
			out = append(out, tag)
		}
	}
	return out
}

// sortedReport orders the map-derived slices of a report so two runs compare
// equal: record551Tags and record70DataLoss iterate a map, so their entries
// arrive in an arbitrary order.
func sortedReport(report *gedcom.ConversionReport) *gedcom.ConversionReport {
	if report == nil {
		return nil
	}
	out := *report

	out.DataLoss = append([]gedcom.DataLossItem(nil), report.DataLoss...)
	sort.Slice(out.DataLoss, func(i, j int) bool {
		return out.DataLoss[i].Feature < out.DataLoss[j].Feature
	})

	out.Dropped = append([]gedcom.ConversionNote(nil), report.Dropped...)
	sort.Slice(out.Dropped, func(i, j int) bool {
		if out.Dropped[i].Path != out.Dropped[j].Path {
			return out.Dropped[i].Path < out.Dropped[j].Path
		}
		return out.Dropped[i].Original < out.Dropped[j].Original
	})

	return &out
}

// skipValidation keeps the comparison about the converter: validation issues are
// the validator's contract, not this package's.
func skipValidation() *ConvertOptions {
	opts := DefaultOptions()
	opts.Validate = false
	return opts
}

// versionPairs covers both text paths: an upgrade runs CONC/CONT consolidation,
// a downgrade runs newline expansion, EXID mapping, and the data-loss sweeps.
var versionPairs = []struct {
	name   string
	source gedcom.Version
	target gedcom.Version
}{
	{"5.5 to 7.0", gedcom.Version55, gedcom.Version70},
	{"7.0 to 5.5", gedcom.Version70, gedcom.Version55},
	{"5.5.1 to 5.5", gedcom.Version551, gedcom.Version55},
	{"5.5 to 5.5.1", gedcom.Version55, gedcom.Version551},
}

func TestConvert_NilsMatchNilFreeDocument(t *testing.T) {
	for _, pair := range versionPairs {
		t.Run(pair.name, func(t *testing.T) {
			withNils := nilFixture(pair.source)
			withoutNils := stripNils(withNils)

			gotDoc, gotReport, err := ConvertWithOptions(withNils, pair.target, skipValidation())
			if err != nil {
				t.Fatalf("ConvertWithOptions() with nils error = %v", err)
			}
			wantDoc, wantReport, err := ConvertWithOptions(withoutNils, pair.target, skipValidation())
			if err != nil {
				t.Fatalf("ConvertWithOptions() without nils error = %v", err)
			}

			if !reflect.DeepEqual(stripNils(gotDoc), stripNils(wantDoc)) {
				t.Errorf("converted document differs from the nil-free conversion\n got = %s\nwant = %s",
					formatTags(gotDoc), formatTags(wantDoc))
			}
			if !reflect.DeepEqual(sortedReport(gotReport), sortedReport(wantReport)) {
				t.Errorf("report differs from the nil-free conversion\n got = %+v\nwant = %+v", gotReport, wantReport)
			}
		})
	}
}

func TestConvert_NilTagPreservedInPlace(t *testing.T) {
	for _, pair := range versionPairs {
		t.Run(pair.name, func(t *testing.T) {
			doc := &gedcom.Document{
				Header: &gedcom.Header{Version: pair.source},
				Records: []*gedcom.Record{
					{
						XRef: "@I1@",
						Type: gedcom.RecordTypeIndividual,
						Tags: []*gedcom.Tag{
							nil, // index 0
							{Level: 1, Tag: "NAME", Value: "John /Doe/"},
							nil, // index 2
						},
					},
				},
			}

			converted, _, err := ConvertWithOptions(doc, pair.target, skipValidation())
			if err != nil {
				t.Fatalf("ConvertWithOptions() error = %v", err)
			}

			tags := converted.Records[0].Tags
			if len(tags) != 3 {
				t.Fatalf("Tags length = %d, want 3 (the nils must not be dropped): %s", len(tags), formatTags(converted))
			}
			for _, idx := range []int{0, 2} {
				if tags[idx] != nil {
					t.Errorf("Tags[%d] = %+v, want nil at its original index", idx, tags[idx])
				}
			}
			if tags[1] == nil || tags[1].Tag != "NAME" {
				t.Errorf("Tags[1] = %+v, want the NAME tag", tags[1])
			}
		})
	}
}

func TestConvert_NilHeader(t *testing.T) {
	for _, target := range []gedcom.Version{gedcom.Version551, gedcom.Version70} {
		t.Run(target.String(), func(t *testing.T) {
			// An absent header carries no version, so both documents convert
			// from the Version55 default and must land on the same result.
			nilHeader := &gedcom.Document{
				Records: []*gedcom.Record{
					{XRef: "@I1@", Type: gedcom.RecordTypeIndividual,
						Tags: []*gedcom.Tag{{Level: 1, Tag: "NAME", Value: "John /Doe/"}}},
				},
			}
			emptyHeader := nilHeader.Clone()
			emptyHeader.Header = &gedcom.Header{}

			got, gotReport, err := ConvertWithOptions(nilHeader, target, skipValidation())
			if err != nil {
				t.Fatalf("ConvertWithOptions() error = %v", err)
			}
			if got.Header == nil {
				t.Fatal("converted document has no header; it must declare the target version")
			}
			if got.Header.Version != target {
				t.Errorf("Header.Version = %q, want %q", got.Header.Version, target)
			}
			if gotReport.SourceVersion != gedcom.Version55 {
				t.Errorf("report.SourceVersion = %q, want the %q default", gotReport.SourceVersion, gedcom.Version55)
			}

			want, wantReport, err := ConvertWithOptions(emptyHeader, target, skipValidation())
			if err != nil {
				t.Fatalf("ConvertWithOptions() with empty header error = %v", err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("nil header converted differently from an empty one\n got = %+v\nwant = %+v", got.Header, want.Header)
			}
			if !reflect.DeepEqual(gotReport, wantReport) {
				t.Errorf("report differs from the empty-header conversion\n got = %+v\nwant = %+v", gotReport, wantReport)
			}
		})
	}
}

// TestConvert_DefaultOptions covers the entry point most callers use, which
// also runs validation over the document still holding its nils.
func TestConvert_DefaultOptions(t *testing.T) {
	for _, pair := range versionPairs {
		t.Run(pair.name, func(t *testing.T) {
			converted, report, err := Convert(nilFixture(pair.source), pair.target)
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}
			if !report.Success {
				t.Error("report.Success = false, want true")
			}
			if converted.Header.Version != pair.target {
				t.Errorf("Header.Version = %q, want %q", converted.Header.Version, pair.target)
			}
		})
	}
}

// formatTags renders a document's tags for a failure message, since a diff of
// two []*gedcom.Tag is unreadable as pointers.
func formatTags(doc *gedcom.Document) string {
	out := "["
	for _, record := range doc.Records {
		if record == nil {
			out += "<nil record> "
			continue
		}
		out += record.XRef + ":{"
		for _, tag := range record.Tags {
			if tag == nil {
				out += "<nil> "
				continue
			}
			out += tag.Tag + "=" + tag.Value + " "
		}
		out += "} "
	}
	return out + "]"
}
