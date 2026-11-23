package gedcom

import (
	"strconv"
	"testing"
	"time"
)

// livingEvent returns an occurred event of type typ dated date ("" for none).
func livingEvent(t *testing.T, typ EventType, date string) *Event {
	t.Helper()
	e := &Event{Type: typ, Date: date}
	if date != "" {
		e.ParsedDate = mustParse(t, date)
	}
	return e
}

func TestIndividual_PresumedLiving_OwnRecord(t *testing.T) {
	cfg := LivingConfig{ReferenceYear: 2026}
	tests := []struct {
		name   string
		ind    *Individual
		cfg    LivingConfig
		living bool
		reason LivingReason
	}{
		{"nil individual", nil, cfg, true, LivingReasonNoDates},
		{"no events", &Individual{}, cfg, true, LivingReasonNoDates},
		{"undated birth", &Individual{Events: []*Event{livingEvent(t, EventBirth, "")}},
			cfg, true, LivingReasonNoDates},
		{"phrase date only", &Individual{Events: []*Event{livingEvent(t, EventBirth, "(young)")}},
			cfg, true, LivingReasonNoDates},
		{"death flag", &Individual{Events: []*Event{{Type: EventDeath, Occurred: true}}},
			cfg, false, LivingReasonDeath},
		{"bare death", &Individual{Events: []*Event{{Type: EventDeath}}},
			cfg, false, LivingReasonDeath},
		{"recent dated death", &Individual{Events: []*Event{
			livingEvent(t, EventBirth, "2000"), livingEvent(t, EventDeath, "2020"),
		}}, cfg, false, LivingReasonDeath},
		{"burial", &Individual{Events: []*Event{livingEvent(t, EventBurial, "")}},
			cfg, false, LivingReasonDeath},
		{"cremation", &Individual{Events: []*Event{livingEvent(t, EventCremation, "2025")}},
			cfg, false, LivingReasonDeath},
		{"negative death assertion ignored", &Individual{Events: []*Event{
			{Type: EventDeath, IsNegative: true},
		}}, cfg, true, LivingReasonNoDates},
		{"birth one year past threshold", &Individual{Events: []*Event{livingEvent(t, EventBirth, "1915")}},
			cfg, false, LivingReasonOldEvent},
		{"birth at threshold", &Individual{Events: []*Event{livingEvent(t, EventBirth, "31 DEC 1916")}},
			cfg, true, LivingReasonWithinThreshold},
		{"christening past threshold", &Individual{Events: []*Event{livingEvent(t, EventChristening, "ABT 1850")}},
			cfg, false, LivingReasonOldEvent},
		{"BEF date past threshold", &Individual{Events: []*Event{livingEvent(t, EventBirth, "BEF 1900")}},
			cfg, false, LivingReasonOldEvent},
		{"AFT date never decides", &Individual{Events: []*Event{livingEvent(t, EventBirth, "AFT 1800")}},
			cfg, true, LivingReasonWithinThreshold},
		{"range judged by its end", &Individual{Events: []*Event{livingEvent(t, EventBirth, "BET 1900 AND 1920")}},
			cfg, true, LivingReasonWithinThreshold},
		{"negated old event ignored", &Individual{Events: []*Event{
			{Type: EventBirth, ParsedDate: mustParse(t, "1800"), IsNegative: true},
		}}, cfg, true, LivingReasonNoDates},
		{"dated attribute", &Individual{Attributes: []*Attribute{
			nil, {Type: AttributeOccupation, ParsedDate: mustParse(t, "1880")},
		}}, cfg, false, LivingReasonOldEvent},
		{"custom threshold", &Individual{Events: []*Event{livingEvent(t, EventBirth, "1950")}},
			LivingConfig{MaxAge: 70, ReferenceYear: 2026}, false, LivingReasonOldEvent},
		{"negative threshold uses default", &Individual{Events: []*Event{livingEvent(t, EventBirth, "1950")}},
			LivingConfig{MaxAge: -1, ReferenceYear: 2026}, true, LivingReasonWithinThreshold},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			living, reason := tt.ind.PresumedLiving(nil, tt.cfg)
			if living != tt.living || reason != tt.reason {
				t.Errorf("PresumedLiving = (%v, %q), want (%v, %q)", living, reason, tt.living, tt.reason)
			}
		})
	}
}

func TestIndividual_PresumedLiving_Relatives(t *testing.T) {
	// @I1@ and @I2@ are the spouses of @F1@, the parents of @I3@; @I4@
	// married in @F2@ and has no children.
	inds := []*Individual{
		{XRef: "@I1@", SpouseInFamilies: []FamilyLink{{FamilyXRef: "@F1@"}}},
		{XRef: "@I2@", SpouseInFamilies: []FamilyLink{{FamilyXRef: "@F1@"}},
			Events: []*Event{livingEvent(t, EventBirth, "1950")}},
		{XRef: "@I3@", ChildInFamilies: []FamilyLink{{FamilyXRef: "@F1@"}},
			Events: []*Event{livingEvent(t, EventBirth, "1900")}},
		{XRef: "@I4@", SpouseInFamilies: []FamilyLink{{FamilyXRef: "@F2@"}}},
	}
	fams := []*Family{
		{XRef: "@F1@", Husband: "@I1@", Wife: "@I2@", Children: []string{"@I3@"}},
		{XRef: "@F2@", Husband: "@I4@", Events: []*Event{livingEvent(t, EventMarriage, "1880")}},
	}
	doc := createRelationshipTestDocument(inds, fams)
	cfg := LivingConfig{ReferenceYear: 2026}

	tests := []struct {
		xref   string
		doc    *Document
		living bool
		reason LivingReason
	}{
		{"@I1@", doc, false, LivingReasonOldChildEvent},
		// Own recent birth places a date, but the child's still decides.
		{"@I2@", doc, false, LivingReasonOldChildEvent},
		{"@I4@", doc, false, LivingReasonOldFamilyEvent},
		// Parents' dates are not consulted.
		{"@I3@", doc, false, LivingReasonOldEvent},
		// Without the document only the record itself is read.
		{"@I1@", nil, true, LivingReasonNoDates},
		{"@I2@", nil, true, LivingReasonWithinThreshold},
	}
	for _, tt := range tests {
		living, reason := doc.GetIndividual(tt.xref).PresumedLiving(tt.doc, cfg)
		if living != tt.living || reason != tt.reason {
			t.Errorf("%s (doc %v): PresumedLiving = (%v, %q), want (%v, %q)",
				tt.xref, tt.doc != nil, living, reason, tt.living, tt.reason)
		}
	}
}

func TestIndividual_PresumedLiving_DefaultReferenceYear(t *testing.T) {
	recent := strconv.Itoa(time.Now().Year() - DefaultLivingMaxAge)
	old := strconv.Itoa(time.Now().Year() - DefaultLivingMaxAge - 1)

	ind := &Individual{Events: []*Event{livingEvent(t, EventBirth, recent)}}
	if living, reason := ind.PresumedLiving(nil, LivingConfig{}); !living || reason != LivingReasonWithinThreshold {
		t.Errorf("birth %s: PresumedLiving = (%v, %q), want living within threshold", recent, living, reason)
	}
	ind = &Individual{Events: []*Event{livingEvent(t, EventBirth, old)}}
	if living, reason := ind.PresumedLiving(nil, LivingConfig{}); living || reason != LivingReasonOldEvent {
		t.Errorf("birth %s: PresumedLiving = (%v, %q), want not living, old event", old, living, reason)
	}
}
