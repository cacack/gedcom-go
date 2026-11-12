package converter

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Issue #739: RESN payloads are lower case in 5.5/5.5.1 and upper case (and
// possibly a list) in 7.0, so conversion rewrites them.

const restriction551 = `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 RESN privacy
1 NAME John /Smith/
1 BIRT
2 DATE 1 JAN 1900
2 RESN confidential
0 @F1@ FAM
1 RESN locked
1 HUSB @I1@
0 TRLR
`

const restriction70 = `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 RESN CONFIDENTIAL, LOCKED, PRIVACY
1 NAME John /Smith/
1 BIRT
2 DATE 1 JAN 1900
2 RESN CONFIDENTIAL
0 @F1@ FAM
1 RESN LOCKED
1 HUSB @I1@
0 @I2@ INDI
1 RESN SECRET
0 TRLR
`

func TestConvertRestrictionUpgrade(t *testing.T) {
	converted, report, err := Convert(decodeText(t, restriction551), gedcom.Version70)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	indi := converted.XRefMap["@I1@"]
	for _, want := range []string{"1 RESN PRIVACY", "2 RESN CONFIDENTIAL"} {
		if !slices.Contains(tagLines(indi), want) {
			t.Errorf("@I1@ tags = %q, want %q", tagLines(indi), want)
		}
	}
	if got := tagLines(converted.XRefMap["@F1@"]); !slices.Contains(got, "1 RESN LOCKED") {
		t.Errorf("@F1@ tags = %q, want 1 RESN LOCKED", got)
	}
	// The typed fields follow the converted tags.
	if got := converted.GetIndividual("@I1@").Restriction; got != "PRIVACY" {
		t.Errorf("Individual.Restriction = %q, want PRIVACY", got)
	}
	if got := converted.GetFamily("@F1@").Restriction; got != "LOCKED" {
		t.Errorf("Family.Restriction = %q, want LOCKED", got)
	}
	if got := converted.GetIndividual("@I1@").Events[0].Restriction; got != "CONFIDENTIAL" {
		t.Errorf("Event.Restriction = %q, want CONFIDENTIAL", got)
	}
	assertRestrictionTransformation(t, report, 3)
	if !hasNormalized(report, "Individual @I1@ > BIRT > RESN", "RESN confidential", "RESN CONFIDENTIAL") {
		t.Errorf("no Normalized note for the BIRT RESN; got %+v", report.Normalized)
	}
	if report.HasDataLoss() {
		t.Errorf("upgrade reported data loss: %+v", report.DataLoss)
	}
}

func TestConvertRestrictionDowngrade(t *testing.T) {
	for _, target := range []gedcom.Version{gedcom.Version551, gedcom.Version55} {
		t.Run(target.String(), func(t *testing.T) {
			converted, report, err := Convert(decodeText(t, restriction70), target)
			if err != nil {
				t.Fatalf("Convert: %v", err)
			}
			indi := converted.XRefMap["@I1@"]
			for _, want := range []string{"1 RESN privacy", "2 RESN confidential"} {
				if !slices.Contains(tagLines(indi), want) {
					t.Errorf("@I1@ tags = %q, want %q", tagLines(indi), want)
				}
			}
			if got := converted.GetFamily("@F1@").Restriction; got != "locked" {
				t.Errorf("Family.Restriction = %q, want locked", got)
			}
			// Not a restriction keyword: left as written.
			if got := converted.GetIndividual("@I2@").Restriction; got != "SECRET" {
				t.Errorf("@I2@ Restriction = %q, want SECRET unchanged", got)
			}
			assertRestrictionTransformation(t, report, 3)
			if !hasNormalized(report, "Individual @I1@ > RESN", "RESN CONFIDENTIAL, LOCKED, PRIVACY", "RESN privacy") {
				t.Errorf("no Normalized note for the list; got %+v", report.Normalized)
			}
			var loss *gedcom.DataLossItem
			for i := range report.DataLoss {
				if report.DataLoss[i].Feature == "RESN restriction lists" {
					loss = &report.DataLoss[i]
				}
			}
			if loss == nil || !slices.Equal(loss.AffectedRecords, []string{"@I1@"}) {
				t.Errorf("DataLoss = %+v, want RESN restriction lists for @I1@", report.DataLoss)
			}
			if !hasNormalizedReason(report, "Individual @I1@ > RESN", "dropped confidential, locked") {
				t.Errorf("list note does not name the dropped keywords; got %+v", report.Normalized)
			}
		})
	}
}

// A record with no Tags is written from its entity, so the typed fields are
// what the pass must rewrite.
func restrictionEntityDoc(version gedcom.Version, record, event, attribute, family string) *gedcom.Document {
	ind := &gedcom.Individual{
		XRef:        "@I1@",
		Restriction: record,
		Events:      []*gedcom.Event{{Type: gedcom.EventBirth, Restriction: event}},
		Attributes:  []*gedcom.Attribute{{Type: gedcom.AttributeOccupation, Value: "Smith", Restriction: attribute}},
	}
	records := []*gedcom.Record{
		{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: ind},
		{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: &gedcom.Family{XRef: "@F1@", Restriction: family}},
	}
	doc := &gedcom.Document{Header: &gedcom.Header{Version: version}, Records: records, XRefMap: map[string]*gedcom.Record{}}
	for _, r := range records {
		doc.XRefMap[r.XRef] = r
	}
	return doc
}

func TestConvertRestrictionEntityOnly(t *testing.T) {
	t.Run("upgrade", func(t *testing.T) {
		doc := restrictionEntityDoc(gedcom.Version551, "privacy", "confidential", "locked", "locked")
		converted, report, err := Convert(doc, gedcom.Version70)
		if err != nil {
			t.Fatalf("Convert: %v", err)
		}
		ind := converted.GetIndividual("@I1@")
		got := []string{ind.Restriction, ind.Events[0].Restriction, ind.Attributes[0].Restriction, converted.GetFamily("@F1@").Restriction}
		if want := []string{"PRIVACY", "CONFIDENTIAL", "LOCKED", "LOCKED"}; !slices.Equal(got, want) {
			t.Errorf("restrictions = %q, want %q", got, want)
		}
		assertRestrictionTransformation(t, report, 4)
		if !hasNormalized(report, "Individual @I1@ > BIRT > RESN", "RESN confidential", "RESN CONFIDENTIAL") {
			t.Errorf("no Normalized note for the BIRT RESN; got %+v", report.Normalized)
		}
	})
	t.Run("downgrade", func(t *testing.T) {
		doc := restrictionEntityDoc(gedcom.Version70, "CONFIDENTIAL, LOCKED", "", "SECRET", "LOCKED")
		converted, report, err := Convert(doc, gedcom.Version551)
		if err != nil {
			t.Fatalf("Convert: %v", err)
		}
		ind := converted.GetIndividual("@I1@")
		got := []string{ind.Restriction, ind.Events[0].Restriction, ind.Attributes[0].Restriction, converted.GetFamily("@F1@").Restriction}
		if want := []string{"confidential", "", "SECRET", "locked"}; !slices.Equal(got, want) {
			t.Errorf("restrictions = %q, want %q", got, want)
		}
		assertRestrictionTransformation(t, report, 2)
		if !hasNormalizedReason(report, "Individual @I1@ > RESN", "dropped locked") {
			t.Errorf("list note does not name the dropped keyword; got %+v", report.Normalized)
		}
		var affected []string
		for _, loss := range report.DataLoss {
			if loss.Feature == "RESN restriction lists" {
				affected = loss.AffectedRecords
			}
		}
		if !slices.Equal(affected, []string{"@I1@"}) {
			t.Errorf("DataLoss = %+v, want RESN restriction lists for @I1@", report.DataLoss)
		}
	})
}

func TestConvertRestrictionListStrict(t *testing.T) {
	_, _, err := ConvertWithOptions(decodeText(t, restriction70), gedcom.Version551, &ConvertOptions{StrictDataLoss: true})
	if !errors.Is(err, ErrDataLoss) {
		t.Errorf("err = %v, want ErrDataLoss for a RESN list on a strict downgrade", err)
	}
}

func TestRestrictionFor(t *testing.T) {
	tests := []struct {
		value   string
		upgrade bool
		want    string
		dropped string
	}{
		{"privacy", true, "PRIVACY", ""},
		{"Locked,privacy", true, "LOCKED, PRIVACY", ""},
		{"PRIVACY", true, "PRIVACY", ""},
		{"LOCKED", false, "locked", ""},
		{"LOCKED, CONFIDENTIAL", false, "confidential", "locked"},
		{"LOCKED, LOCKED", false, "locked", ""},
		{"locked, secret", true, "locked, secret", ""},
		{"", false, "", ""},
	}
	for _, tt := range tests {
		got, dropped := restrictionFor(tt.value, tt.upgrade)
		if got != tt.want || dropped != tt.dropped {
			t.Errorf("restrictionFor(%q, %v) = %q, %q; want %q, %q", tt.value, tt.upgrade, got, dropped, tt.want, tt.dropped)
		}
	}
}

func assertRestrictionTransformation(t *testing.T, report *gedcom.ConversionReport, count int) {
	t.Helper()
	for _, tr := range report.Transformations {
		if tr.Type == "RESTRICTION_VALUES_MAPPED" {
			if tr.Count != count {
				t.Errorf("RESTRICTION_VALUES_MAPPED count = %d, want %d", tr.Count, count)
			}
			return
		}
	}
	t.Errorf("no RESTRICTION_VALUES_MAPPED transformation in %+v", report.Transformations)
}

func hasNormalized(report *gedcom.ConversionReport, path, original, result string) bool {
	for _, n := range report.Normalized {
		if strings.Contains(n.Path, path) && n.Original == original && n.Result == result {
			return true
		}
	}
	return false
}

func hasNormalizedReason(report *gedcom.ConversionReport, path, reason string) bool {
	for _, n := range report.Normalized {
		if strings.Contains(n.Path, path) && strings.Contains(n.Reason, reason) {
			return true
		}
	}
	return false
}
