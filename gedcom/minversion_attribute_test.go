package gedcom

import "testing"

// Attribute-borne 7.0 markers must reach RequiresGEDCOM7.
//
// individualRequiresGEDCOM7 used to skip Attributes on the stated
// precondition that "the Attribute struct carries no 7.0-only typed
// fields". Once SortDate and Associations landed on Attribute the
// precondition stopped holding, and MinimumVersion would have reported
// 5.5.1 for a document the encoder writes SDATE/ASSO-PHRASE into --
// which a 5.5.1 importer rejects or silently drops. UIDs are not a marker:
// the encoder writes them as _UID for 5.5.x output (#715), covered by
// TestRequiresGEDCOM7_UIDsAreNotAMarker.
func TestRequiresGEDCOM7_AttributeMarkers(t *testing.T) {
	tests := []struct {
		name string
		attr *Attribute
	}{
		{"SDATE on an attribute", &Attribute{Type: "OCCU", SortDate: "1900-01-01"}},
		{"ASSO with PHRASE on an attribute", &Attribute{
			Type:         "OCCU",
			Associations: []*Association{{IndividualXRef: "@I2@", Phrase: "godparent"}},
		}},
		// 5.5 and 5.5.1 define ASSOCIATION_STRUCTURE under INDI only, never
		// inside EVENT_DETAIL, so a bare ASSO here is 7.0-only by position.
		{"bare ASSO on an attribute", &Attribute{
			Type:         "OCCU",
			Associations: []*Association{{IndividualXRef: "@I2@"}},
		}},
	}

	for _, tt := range tests {
		t.Run("individual/"+tt.name, func(t *testing.T) {
			doc := &Document{Records: []*Record{{
				XRef: "@I1@", Type: RecordTypeIndividual,
				Entity: &Individual{XRef: "@I1@", Attributes: []*Attribute{tt.attr}},
			}}}
			if !doc.RequiresGEDCOM7() {
				t.Errorf("RequiresGEDCOM7() = false, want true; MinimumVersion() = %v", doc.MinimumVersion())
			}
		})

		t.Run("family/"+tt.name, func(t *testing.T) {
			doc := &Document{Records: []*Record{{
				XRef: "@F1@", Type: RecordTypeFamily,
				Entity: &Family{XRef: "@F1@", Attributes: []*Attribute{tt.attr}},
			}}}
			if !doc.RequiresGEDCOM7() {
				t.Errorf("RequiresGEDCOM7() = false, want true; MinimumVersion() = %v", doc.MinimumVersion())
			}
		})
	}
}

// An ASSO on an event is 7.0-only whether or not it carries a PHRASE:
// 5.5/5.5.1 have no EVENT_DETAIL.ASSO position at all. A witness on a
// baptism is the ordinary case and must not be labelled 5.5.1.
func TestRequiresGEDCOM7_EventAssociation(t *testing.T) {
	for _, assoc := range []*Association{
		{IndividualXRef: "@I2@", Phrase: "godparent"},
		{IndividualXRef: "@I2@", Role: "WITN"},
		{IndividualXRef: "@I2@"},
	} {
		doc := &Document{Records: []*Record{{
			XRef: "@I1@", Type: RecordTypeIndividual,
			Entity: &Individual{XRef: "@I1@", Events: []*Event{{
				Type: EventBaptism, Associations: []*Association{assoc},
			}}},
		}}}
		if !doc.RequiresGEDCOM7() {
			t.Errorf("RequiresGEDCOM7() = false for event ASSO %+v; MinimumVersion() = %v",
				assoc, doc.MinimumVersion())
		}
	}
}

// An INDI-level ASSO without PHRASE stays 5.5.1: 5.5.1 has INDI.ASSO, and
// PHRASE is the only 7.0 addition there. This is the asymmetry that makes
// associationPhraseRequiresGEDCOM7 the wrong test for events.
func TestRequiresGEDCOM7_IndividualAssociationWithoutPhrase(t *testing.T) {
	doc := &Document{Records: []*Record{{
		XRef: "@I1@", Type: RecordTypeIndividual,
		Entity: &Individual{XRef: "@I1@",
			Associations: []*Association{{IndividualXRef: "@I2@", Role: "GODP"}}},
	}}}
	if doc.RequiresGEDCOM7() {
		t.Error("RequiresGEDCOM7() = true for a bare INDI.ASSO, want false (5.5.1 has it)")
	}
}

// A 5.5.1-only attribute must not be dragged up to 7.0 by the new checks.
func TestRequiresGEDCOM7_PlainAttributeStays551(t *testing.T) {
	doc := &Document{Records: []*Record{{
		XRef: "@I1@", Type: RecordTypeIndividual,
		Entity: &Individual{XRef: "@I1@", Attributes: []*Attribute{{
			Type: "OCCU", Value: "Blacksmith", Date: "1900", Agency: "Guild",
		}}},
	}}}
	if doc.RequiresGEDCOM7() {
		t.Error("RequiresGEDCOM7() = true, want false for a 5.5.1-compatible attribute")
	}
}

// Each typed PHRASE field (issue #450) is 7.0-only and written unconditionally
// beneath its value, so it must push the document to 7.0.
func TestRequiresGEDCOM7_TypedPhrases(t *testing.T) {
	indi := func(i *Individual) *Record {
		i.XRef = "@I1@"
		return &Record{XRef: "@I1@", Type: RecordTypeIndividual, Entity: i}
	}
	tests := []struct {
		name string
		rec  *Record
	}{
		{"NAME.TYPE", indi(&Individual{Names: []*PersonalName{{Type: "OTHER", TypePhrase: "p"}}})},
		{"FAMC.PEDI", indi(&Individual{ChildInFamilies: []FamilyLink{{FamilyXRef: "@F1@", Pedigree: "OTHER", PedigreePhrase: "p"}}})},
		{"ASSO.ROLE", indi(&Individual{Associations: []*Association{{IndividualXRef: "@I2@", Role: "OTHER", RolePhrase: "p"}}})},
		{"event AGE", indi(&Individual{Events: []*Event{{Type: EventBirth, Age: "3y", AgePhrase: "p"}}})},
		{"event SDATE", indi(&Individual{Events: []*Event{{Type: EventBirth, SortDatePhrase: "p"}}})},
		{"attribute SDATE", indi(&Individual{Attributes: []*Attribute{{Type: "OCCU", SortDatePhrase: "p"}}})},
		{"FAM.HUSB", &Record{XRef: "@F1@", Type: RecordTypeFamily, Entity: &Family{Husband: "@I1@", HusbandPhrase: "p"}}},
		{"FAM.WIFE", &Record{XRef: "@F1@", Type: RecordTypeFamily, Entity: &Family{Wife: "@I1@", WifePhrase: "p"}}},
		{"FILE.FORM.MEDI", &Record{XRef: "@O1@", Type: RecordTypeMedia, Entity: &MediaObject{
			Files: []*MediaFile{{FileRef: "a.jpg", Form: "image/jpeg", MediaType: "OTHER", MediaPhrase: "p"}},
		}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &Document{Records: []*Record{tt.rec}}
			if !doc.RequiresGEDCOM7() {
				t.Errorf("RequiresGEDCOM7() = false, want true")
			}
		})
	}
}
