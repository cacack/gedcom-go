package converter

import (
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// findTag returns the first tag in the record with the given name, or nil.
func findTag(record *gedcom.Record, name string) *gedcom.Tag {
	for _, tag := range record.Tags {
		if tag.Tag == name {
			return tag
		}
	}
	return nil
}

// exidInDataLoss reports whether the report records EXID as lost.
func exidInDataLoss(report *gedcom.ConversionReport) bool {
	for _, loss := range report.DataLoss {
		if contains(loss.Feature, "EXID") {
			return true
		}
	}
	return false
}

// fsftidNormalized reports whether the report notes an EXID -> _FSFTID mapping.
func fsftidNormalized(report *gedcom.ConversionReport) bool {
	for _, note := range report.Normalized {
		if note.Result == "_FSFTID" {
			return true
		}
	}
	return false
}

func TestTransformEXIDToVendorTags_FamilySearchArkToFSFTID(t *testing.T) {
	tests := []struct {
		name          string
		targetVersion gedcom.Version
	}{
		{"7.0 to 5.5", gedcom.Version55},
		{"7.0 to 5.5.1", gedcom.Version551},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &gedcom.Document{
				Header: &gedcom.Header{Version: gedcom.Version70},
				Records: []*gedcom.Record{
					{
						XRef: "@I1@",
						Type: gedcom.RecordTypeIndividual,
						Tags: []*gedcom.Tag{
							{Level: 1, Tag: "EXID", Value: "KWCJ-QN7"},
							{Level: 2, Tag: "TYPE", Value: "https://www.familysearch.org/ark"},
						},
					},
				},
			}

			result, report, err := Convert(doc, tt.targetVersion)
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}

			rec := result.Records[0]
			fsftid := findTag(rec, "_FSFTID")
			if fsftid == nil {
				t.Fatalf("expected _FSFTID tag; got tags %+v", rec.Tags)
			}
			if fsftid.Value != "KWCJ-QN7" {
				t.Errorf("_FSFTID value = %q, want KWCJ-QN7", fsftid.Value)
			}
			if fsftid.Level != 1 {
				t.Errorf("_FSFTID level = %d, want 1", fsftid.Level)
			}
			if findTag(rec, "EXID") != nil {
				t.Error("EXID should have been replaced, not left in place")
			}
			if findTag(rec, "TYPE") != nil {
				t.Error("EXID's TYPE subordinate should have been removed")
			}
			if !fsftidNormalized(report) {
				t.Error("report should note the EXID -> _FSFTID mapping as normalized")
			}
			if exidInDataLoss(report) {
				t.Error("a converted EXID must not also be reported as data loss")
			}
		})
	}
}

func TestTransformEXIDToVendorTags_PointerShapedValueEscaped(t *testing.T) {
	// A FamilySearch ARK EXID whose value is itself pointer-shaped (e.g. "@I2@")
	// must be escaped to "@@I2@" in the synthesized _FSFTID so the XRef walk and
	// strict parsers do not mistake the literal identifier for a cross-reference
	// pointer (issue #346). A normal ARK id passes through unchanged.
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"pointer-shaped value escaped", "@I2@", "@@I2@"},
		{"normal ARK value unchanged", "KWCJ-QN7", "KWCJ-QN7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &gedcom.Document{
				Header: &gedcom.Header{Version: gedcom.Version70},
				Records: []*gedcom.Record{
					{
						XRef: "@I1@",
						Type: gedcom.RecordTypeIndividual,
						Tags: []*gedcom.Tag{
							{Level: 1, Tag: "EXID", Value: tt.value},
							{Level: 2, Tag: "TYPE", Value: "https://www.familysearch.org/ark"},
						},
					},
				},
			}

			result, _, err := Convert(doc, gedcom.Version551)
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}
			fsftid := findTag(result.Records[0], "_FSFTID")
			if fsftid == nil {
				t.Fatalf("expected _FSFTID tag; got tags %+v", result.Records[0].Tags)
			}
			if fsftid.Value != tt.want {
				t.Errorf("_FSFTID value = %q, want %q", fsftid.Value, tt.want)
			}
			if gedcom.IsPointerXRef(fsftid.Value) {
				t.Errorf("_FSFTID value %q must not be pointer-shaped after conversion", fsftid.Value)
			}
		})
	}
}

func TestTransformEXIDToVendorTags_NonFamilySearchStillDropped(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version70},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "EXID", Value: "12345"},
					{Level: 2, Tag: "TYPE", Value: "https://www.findagrave.com"},
				},
			},
		},
	}

	result, report, err := Convert(doc, gedcom.Version551)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	rec := result.Records[0]
	if findTag(rec, "_FSFTID") != nil {
		t.Error("non-FamilySearch EXID must not be mapped to _FSFTID")
	}
	if findTag(rec, "EXID") == nil {
		t.Error("non-FamilySearch EXID should be left in place for the data-loss sweep")
	}
	if !exidInDataLoss(report) {
		t.Error("non-FamilySearch EXID should still be reported as data loss")
	}
}

func TestTransformEXIDToVendorTags_ArkPathBoundary(t *testing.T) {
	// "familysearch.org/ark" must match only as a whole path segment, so an
	// unrelated URI containing it as a prefix (".../arkansas") is not converted.
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version70},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "EXID", Value: "42"},
					{Level: 2, Tag: "TYPE", Value: "https://www.familysearch.org/arkansas-records"},
				},
			},
		},
	}

	result, report, err := Convert(doc, gedcom.Version551)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	rec := result.Records[0]
	if findTag(rec, "_FSFTID") != nil {
		t.Error("a non-ARK URI that merely contains the ark prefix must not convert")
	}
	if findTag(rec, "EXID") == nil {
		t.Error("EXID should be left in place")
	}
	if !exidInDataLoss(report) {
		t.Error("non-ARK EXID should still be reported as data loss")
	}
}

func TestTransformEXIDToVendorTags_ExtraSubordinateNotConverted(t *testing.T) {
	// An EXID with a subordinate other than its matching TYPE would lose that
	// content if collapsed to a single _FSFTID, so it is not converted.
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version70},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "EXID", Value: "KWCJ-QN7"},
					{Level: 2, Tag: "TYPE", Value: "https://www.familysearch.org/ark"},
					{Level: 2, Tag: "NOTE", Value: "provenance detail"},
				},
			},
		},
	}

	result, report, err := Convert(doc, gedcom.Version551)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	rec := result.Records[0]
	if findTag(rec, "_FSFTID") != nil {
		t.Error("EXID with an extra subordinate must not be collapsed to _FSFTID")
	}
	if findTag(rec, "EXID") == nil || findTag(rec, "NOTE") == nil {
		t.Error("EXID and its NOTE subordinate should be left in place")
	}
	if !exidInDataLoss(report) {
		t.Error("un-converted EXID should still be reported as data loss")
	}
}

func TestTransformEXIDToVendorTags_ArkTypeWithDeepSubordinateNotConverted(t *testing.T) {
	// A matching TYPE that itself carries a subordinate would lose that content
	// if the block were collapsed to a single _FSFTID, so it is not converted.
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version70},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "EXID", Value: "KWCJ-QN7"},
					{Level: 2, Tag: "TYPE", Value: "https://www.familysearch.org/ark"},
					{Level: 3, Tag: "NOTE", Value: "provenance detail"},
				},
			},
		},
	}

	result, report, err := Convert(doc, gedcom.Version551)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	rec := result.Records[0]
	if findTag(rec, "_FSFTID") != nil {
		t.Error("EXID whose TYPE has a deeper subordinate must not be collapsed to _FSFTID")
	}
	if findTag(rec, "EXID") == nil || findTag(rec, "NOTE") == nil {
		t.Error("EXID and the NOTE under its TYPE should be left in place")
	}
	if !exidInDataLoss(report) {
		t.Error("un-converted EXID should still be reported as data loss")
	}
}

func TestTransformEXIDToVendorTags_SkippedWhenPreserveUnknownFalse(t *testing.T) {
	// _FSFTID is a vendor extension, so a caller who opts out of vendor tags
	// keeps the plain EXID-is-data-loss behavior.
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version70},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "EXID", Value: "KWCJ-QN7"},
					{Level: 2, Tag: "TYPE", Value: "https://www.familysearch.org/ark"},
				},
			},
		},
	}

	result, report, err := ConvertWithOptions(doc, gedcom.Version551, &ConvertOptions{Validate: true, PreserveUnknownTags: false})
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}
	rec := result.Records[0]
	if findTag(rec, "_FSFTID") != nil {
		t.Error("EXID must not be mapped to _FSFTID when PreserveUnknownTags is false")
	}
	if !exidInDataLoss(report) {
		t.Error("with vendor tags disabled, EXID should be reported as data loss")
	}
}

func TestTransformEXIDToVendorTags_RecordsTransformation(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version70},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "EXID", Value: "KWCJ-QN7"},
					{Level: 2, Tag: "TYPE", Value: "https://www.familysearch.org/ark"},
				},
			},
		},
	}

	_, report, err := Convert(doc, gedcom.Version551)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	found := false
	for _, tr := range report.Transformations {
		if tr.Type == "EXID_TO_VENDOR_TAG" {
			found = true
			if tr.Count != 1 {
				t.Errorf("transformation Count = %d, want 1", tr.Count)
			}
		}
	}
	if !found {
		t.Errorf("expected an EXID_TO_VENDOR_TAG transformation entry; got %+v", report.Transformations)
	}
}

func TestTransformEXIDToVendorTags_NonIndividualUntouched(t *testing.T) {
	// _FSFTID is an individual-only tag, so a FamilySearch ARK EXID on a
	// non-individual record has no faithful vendor mapping and stays data loss.
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version70},
		Records: []*gedcom.Record{
			{
				XRef: "@F1@",
				Type: gedcom.RecordTypeFamily,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "EXID", Value: "KWCJ-QN7"},
					{Level: 2, Tag: "TYPE", Value: "https://www.familysearch.org/ark"},
				},
			},
		},
	}

	result, report, err := Convert(doc, gedcom.Version551)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	rec := result.Records[0]
	if findTag(rec, "_FSFTID") != nil {
		t.Error("EXID on a family record must not be mapped to _FSFTID")
	}
	if findTag(rec, "EXID") == nil {
		t.Error("EXID on a family record should be left in place")
	}
	if !exidInDataLoss(report) {
		t.Error("EXID on a family record should still be reported as data loss")
	}
}
