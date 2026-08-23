package validator

import (
	"reflect"
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// A document assembled in memory can hold a nil *Record, a nil *Tag, a typed-nil
// Record.Entity or a nil Header -- shapes decoding never produces. The validator
// must skip them silently rather than panic (ADR 0007). Reporting them is issue
// #457's job, so a skipped nil must leave the results unchanged.

// individualTags returns the INDI record's tags. Each call returns fresh
// pointers so a case can build both documents it compares without sharing.
func individualTags() []*gedcom.Tag {
	return []*gedcom.Tag{
		{Level: 1, Tag: "NAME", Value: "John /Smith/"},
		{Level: 1, Tag: "BIRT"},
		{Level: 2, Tag: "DATE", Value: "1 JAN 1900"},
		{Level: 2, Tag: "NOTE", Value: "banned \x01 character"},
		{Level: 1, Tag: "FAMS", Value: "@F999@"},
		{Level: 1, Tag: "_MILT", Value: "Army"},
		{Level: 1, Tag: "_APID", Value: "7602,2771226"},
		{Level: 1, Tag: "_ZZZ", Value: "unknown"},
	}
}

// nilSafetyDoc builds a GEDCOM 7.0 document that every validator has something
// to say about -- a broken FAMS reference, a banned control character, a custom
// tag under the wrong parent and an unregistered one -- then applies mutate.
func nilSafetyDoc(mutate func(*gedcom.Document)) *gedcom.Document {
	indi := &gedcom.Record{
		XRef: "@I1@",
		Type: gedcom.RecordTypeIndividual,
		Tags: individualTags(),
		Entity: &gedcom.Individual{
			XRef:             "@I1@",
			Names:            []*gedcom.PersonalName{{Full: "John /Smith/"}},
			SpouseInFamilies: []string{"@F1@", "@F999@"},
		},
	}
	fam := &gedcom.Record{
		XRef:   "@F1@",
		Type:   gedcom.RecordTypeFamily,
		Tags:   []*gedcom.Tag{{Level: 1, Tag: "HUSB", Value: "@I1@"}},
		Entity: &gedcom.Family{XRef: "@F1@", Husband: "@I1@"},
	}

	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version: gedcom.Version70,
			Tags:    []*gedcom.Tag{{Level: 1, Tag: "SOUR", Value: "TestApp"}},
		},
		Records: []*gedcom.Record{indi, fam},
		XRefMap: map[string]*gedcom.Record{"@I1@": indi, "@F1@": fam},
	}

	if mutate != nil {
		mutate(doc)
	}
	return doc
}

// insertNilRecords surrounds and separates the real records with nil ones.
func insertNilRecords(doc *gedcom.Document) {
	doc.Records = []*gedcom.Record{nil, doc.Records[0], nil, doc.Records[1], nil}
}

// insertNilTags splices nils into the individual's tag list, including one
// between BIRT and its level-2 children so the recursive scanners meet a nil
// nested below a real tag.
func insertNilTags(doc *gedcom.Document) {
	rec := doc.Records[0]
	tags := []*gedcom.Tag{nil, rec.Tags[0], rec.Tags[1], nil}
	tags = append(tags, rec.Tags[2:]...)
	tags = append(tags, nil)
	rec.Tags = tags
}

// insertNilHeaderTags puts nils around the header's tags, which the encoding
// validator scans separately from the records.
func insertNilHeaderTags(doc *gedcom.Document) {
	doc.Header.Tags = append([]*gedcom.Tag{nil}, append(doc.Header.Tags, nil)...)
}

// extraRecords appends one record of each entity-bearing type. With typed set,
// each carries a typed nil in Entity; without it, each carries no entity at all.
// The two documents must validate identically.
func extraRecords(typed bool) func(*gedcom.Document) {
	return func(doc *gedcom.Document) {
		var ind, fam, src interface{}
		if typed {
			ind = (*gedcom.Individual)(nil)
			fam = (*gedcom.Family)(nil)
			src = (*gedcom.Source)(nil)
		}

		add := func(xref string, recordType gedcom.RecordType, tags []*gedcom.Tag, entity interface{}) {
			rec := &gedcom.Record{XRef: xref, Type: recordType, Tags: tags, Entity: entity}
			doc.Records = append(doc.Records, rec)
			doc.XRefMap[xref] = rec
		}

		add("@I2@", gedcom.RecordTypeIndividual, []*gedcom.Tag{{Level: 1, Tag: "NAME", Value: "Jane /Doe/"}}, ind)
		add("@F2@", gedcom.RecordTypeFamily, []*gedcom.Tag{{Level: 1, Tag: "HUSB", Value: "@I1@"}}, fam)
		add("@S1@", gedcom.RecordTypeSource, []*gedcom.Tag{{Level: 1, Tag: "TITL", Value: "A Source"}}, src)
	}
}

// strictConfig enables every validator, so a comparison covers all of them.
func strictConfig() *ValidatorConfig {
	return &ValidatorConfig{
		Strictness:         StrictnessStrict,
		TagRegistry:        DefaultVendorRegistry(),
		ValidateCustomTags: true,
	}
}

// assertSameValidation checks that the nil-bearing document reports exactly what
// the nil-free one does -- not merely that neither panicked.
func assertSameValidation(t *testing.T, withNils, clean *gedcom.Document) {
	t.Helper()

	wantErrs := New().Validate(clean)
	if len(wantErrs) == 0 {
		t.Fatal("nil-free document reported no errors; the comparison would pass vacuously")
	}
	if gotErrs := New().Validate(withNils); !reflect.DeepEqual(gotErrs, wantErrs) {
		t.Errorf("Validate() with nils = %v, want %v", gotErrs, wantErrs)
	}

	wantIssues := NewWithConfig(strictConfig()).ValidateAll(clean)
	if len(wantIssues) == 0 {
		t.Fatal("nil-free document reported no issues; the comparison would pass vacuously")
	}
	if gotIssues := NewWithConfig(strictConfig()).ValidateAll(withNils); !reflect.DeepEqual(gotIssues, wantIssues) {
		t.Errorf("ValidateAll() with nils = %v, want %v", gotIssues, wantIssues)
	}
}

// nilCases are the nil shapes a document can hold. clean is nil where dropping
// the nil means the document is simply the unmutated one.
var nilCases = []struct {
	name     string
	withNils func(*gedcom.Document)
	clean    func(*gedcom.Document)
}{
	{name: "nil records among real ones", withNils: insertNilRecords},
	{name: "nil tags in a record", withNils: insertNilTags},
	{name: "nil tags in the header", withNils: insertNilHeaderTags},
	{name: "typed-nil entities", withNils: extraRecords(true), clean: extraRecords(false)},
}

func TestValidate_SkipsNils(t *testing.T) {
	for _, tt := range nilCases {
		t.Run(tt.name, func(t *testing.T) {
			assertSameValidation(t, nilSafetyDoc(tt.withNils), nilSafetyDoc(tt.clean))
		})
	}
}

// TagValidator.Validate and EncodingValidator.Validate are public API in their
// own right, so they get the same treatment directly.
func TestSubValidators_SkipNils(t *testing.T) {
	tv := NewTagValidator(DefaultVendorRegistry(), true)
	ev := NewEncodingValidator()

	for _, tt := range nilCases {
		t.Run(tt.name, func(t *testing.T) {
			withNils, clean := nilSafetyDoc(tt.withNils), nilSafetyDoc(tt.clean)

			want := tv.Validate(clean)
			if len(want) == 0 {
				t.Fatal("nil-free document reported no tag issues; the comparison would pass vacuously")
			}
			if got := tv.Validate(withNils); !reflect.DeepEqual(got, want) {
				t.Errorf("TagValidator.Validate() with nils = %v, want %v", got, want)
			}

			want = ev.Validate(clean)
			if len(want) == 0 {
				t.Fatal("nil-free document reported no encoding issues; the comparison would pass vacuously")
			}
			if got := ev.Validate(withNils); !reflect.DeepEqual(got, want) {
				t.Errorf("EncodingValidator.Validate() with nils = %v, want %v", got, want)
			}
		})
	}
}

func TestValidate_NilHeader(t *testing.T) {
	doc := nilSafetyDoc(func(d *gedcom.Document) { d.Header = nil })

	// Validate never reads the header, so it must report exactly as before.
	want := New().Validate(nilSafetyDoc(nil))
	if got := New().Validate(doc); !reflect.DeepEqual(got, want) {
		t.Errorf("Validate() with nil header = %v, want %v", got, want)
	}

	// ValidateAll's header-independent checks still run; the version-gated ones
	// (encoding, XRef length) have nothing to gate on and are skipped.
	issues := NewWithConfig(strictConfig()).ValidateAll(doc)
	found := false
	for _, issue := range issues {
		if issue.Code == CodeOrphanedFAMS {
			found = true
		}
	}
	if !found {
		t.Errorf("ValidateAll() with nil header = %v, want an %s issue", issues, CodeOrphanedFAMS)
	}
}

func TestValidate_NilDocument(t *testing.T) {
	v := New()

	if errs := v.Validate(nil); len(errs) != 0 {
		t.Errorf("Validate(nil) = %v, want no errors", errs)
	}
	if issues := v.ValidateAll(nil); len(issues) != 0 {
		t.Errorf("ValidateAll(nil) = %v, want no issues", issues)
	}
	if issues := NewTagValidator(DefaultVendorRegistry(), true).Validate(nil); len(issues) != 0 {
		t.Errorf("TagValidator.Validate(nil) = %v, want no issues", issues)
	}
	if issues := NewEncodingValidator().Validate(nil); len(issues) != 0 {
		t.Errorf("EncodingValidator.Validate(nil) = %v, want no issues", issues)
	}
}
