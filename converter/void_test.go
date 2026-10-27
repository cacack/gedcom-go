package converter_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/converter"
	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

const voidPointers70 = `0 HEAD
1 GEDC
2 VERS 7.0
1 SUBM @VOID@
0 @I1@ INDI
1 NAME John /Doe/
1 FAMC @VOID@
2 PHRASE adopted, family unknown
1 FAMS @F1@
1 BIRT
2 SOUR @VOID@
3 PAGE p. 4
1 SNOTE @VOID@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @VOID@
1 CHIL @VOID@
0 @S1@ SOUR
1 TITL Census
1 REPO @VOID@
0 TRLR
`

// TestDowngradeDropsVoidPointers pins issue #670: a 7.0 -> 5.x downgrade
// removes every @VOID@ null pointer, since 5.x has none and Subset would
// otherwise reject it as an unknown XRef.
func TestDowngradeDropsVoidPointers(t *testing.T) {
	for _, target := range []gedcom.Version{gedcom.Version551, gedcom.Version55} {
		t.Run(target.String(), func(t *testing.T) {
			doc, err := decoder.Decode(strings.NewReader(voidPointers70))
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			converted, report, err := converter.Convert(doc, target)
			if err != nil {
				t.Fatalf("Convert: %v", err)
			}

			if _, err := converted.Subset([]string{"@I1@"}); err != nil {
				t.Errorf("Subset after downgrade: %v", err)
			}
			if _, err := converted.Subset([]string{"@S1@"}); err != nil {
				t.Errorf("Subset(@S1@) after downgrade: %v", err)
			}

			var buf bytes.Buffer
			if err := encoder.Encode(&buf, converted); err != nil {
				t.Fatalf("encode: %v", err)
			}
			if out := buf.String(); strings.Contains(out, "@VOID@") || strings.Contains(out, "PHRASE") {
				t.Errorf("output keeps a @VOID@ pointer or its substructure:\n%s", out)
			}
			if fam := converted.GetFamily("@F1@"); fam.Wife != "" || len(fam.Children) != 0 || fam.Husband != "@I1@" {
				t.Errorf("family pointers = %q %q %v, want @I1@, \"\", none", fam.Husband, fam.Wife, fam.Children)
			}
			if converted.Header.Submitter != "" {
				t.Errorf("Header.Submitter = %q, want cleared", converted.Header.Submitter)
			}

			var loss *gedcom.DataLossItem
			for i := range report.DataLoss {
				if report.DataLoss[i].Feature == "@VOID@ null pointers" {
					loss = &report.DataLoss[i]
				}
			}
			if loss == nil {
				t.Fatalf("no @VOID@ data loss in report: %+v", report.DataLoss)
			}
			if got := strings.Join(loss.AffectedRecords, ","); got != "@I1@,@F1@,@S1@" {
				t.Errorf("AffectedRecords = %s, want @I1@,@F1@,@S1@", got)
			}

			var dropped []string
			for _, n := range report.Dropped {
				if strings.HasSuffix(n.Original, "@VOID@") {
					dropped = append(dropped, n.Path+"="+n.Original)
				}
			}
			want := []string{
				"Header > SUBM=SUBM @VOID@",
				"Individual @I1@ > FAMC=FAMC @VOID@",
				"Individual @I1@ > SNOTE=SNOTE @VOID@",
				"Individual @I1@ > SOUR=SOUR @VOID@",
				"Family @F1@ > CHIL=CHIL @VOID@",
				"Family @F1@ > WIFE=WIFE @VOID@",
				"Source @S1@ > REPO=REPO @VOID@",
			}
			if strings.Join(dropped, "\n") != strings.Join(want, "\n") {
				t.Errorf("dropped notes:\n%s\nwant:\n%s", strings.Join(dropped, "\n"), strings.Join(want, "\n"))
			}
			for _, n := range report.Dropped {
				if n.Original == "PHRASE" {
					t.Errorf("PHRASE under a removed @VOID@ pointer reported separately: %+v", n)
				}
			}
		})
	}
}

// TestDowngradeDropsVoidPointersFromEntity covers a record built in code with
// no Tags, which the encoder writes from its entity: every @VOID@ pointer in
// the entity is removed and the record is reported by its path.
func TestDowngradeDropsVoidPointersFromEntity(t *testing.T) {
	void := "@VOID@"
	cite := func() []*gedcom.SourceCitation {
		return []*gedcom.SourceCitation{{SourceXRef: void}, {SourceXRef: "@S1@", NoteXRefs: []string{void}}}
	}
	media := func() []*gedcom.MediaLink {
		return []*gedcom.MediaLink{{MediaXRef: void}, {MediaXRef: "@M1@", NoteXRefs: []string{void}}}
	}
	assoc := func() []*gedcom.Association {
		return []*gedcom.Association{{IndividualXRef: void}, {IndividualXRef: "@I2@", NoteXRefs: []string{void}, SourceCitations: cite()}}
	}
	event := func() *gedcom.Event {
		return &gedcom.Event{Type: "BIRT", NoteXRefs: []string{void}, PlaceDetail: &gedcom.PlaceDetail{Name: "X", NoteXRefs: []string{void}},
			SourceCitations: cite(), Media: media(), Associations: assoc()}
	}
	attr := func() *gedcom.Attribute {
		return &gedcom.Attribute{Type: "OCCU", Value: "Smith", NoteXRefs: []string{void}, PlaceDetail: &gedcom.PlaceDetail{NoteXRefs: []string{void}},
			SourceCitations: cite(), Media: media(), Associations: assoc()}
	}
	cd := func() *gedcom.ChangeDate { return &gedcom.ChangeDate{Date: "1 JAN 2000", NoteXRefs: []string{void}} }
	ords := func() []*gedcom.LDSOrdinance {
		return []*gedcom.LDSOrdinance{{Type: gedcom.LDSSealingChild, FamilyXRef: void, NoteXRefs: []string{void}}}
	}

	indi := &gedcom.Individual{
		XRef:             "@I1@",
		ChildInFamilies:  []gedcom.FamilyLink{{FamilyXRef: void}, {FamilyXRef: "@F1@", NoteXRefs: []string{void}}},
		SpouseInFamilies: []gedcom.FamilyLink{{FamilyXRef: void}},
		Names:            []*gedcom.PersonalName{{Full: "John /Doe/", NoteXRefs: []string{void}, SourceCitations: cite()}},
		NoteXRefs:        []string{void},
		Associations:     assoc(),
		SourceCitations:  cite(),
		Media:            media(),
		Events:           []*gedcom.Event{event()},
		Attributes:       []*gedcom.Attribute{attr()},
		LDSOrdinances:    ords(),
		ChangeDate:       cd(),
		Tags:             []*gedcom.Tag{{Level: 1, Tag: "_LINK", Value: void}, {Level: 2, Tag: "_X", Value: "y"}},
	}
	fam := &gedcom.Family{
		XRef: "@F1@", Husband: void, Wife: void, Children: []string{void, "@I1@"},
		NoteXRefs: []string{void}, SourceCitations: cite(), Media: media(),
		Events: []*gedcom.Event{event()}, Attributes: []*gedcom.Attribute{attr()},
		LDSOrdinances: ords(), ChangeDate: cd(), CreationDate: cd(),
	}
	src := &gedcom.Source{
		XRef: "@S1@", RepositoryLinks: []*gedcom.SourceRepositoryLink{{XRef: void}, {XRef: "@R1@", NoteXRefs: []string{void}}},
		NoteXRefs: []string{void}, Media: media(), ChangeDate: cd(), CreationDate: cd(),
	}
	obje := &gedcom.MediaObject{XRef: "@M1@", NoteXRefs: []string{void}, SharedNoteXRefs: []string{void},
		SourceCitations: cite(), ChangeDate: cd(), CreationDate: cd()}
	repo := &gedcom.Repository{XRef: "@R1@", Name: "R", NoteXRefs: []string{void}}
	subm := &gedcom.Submitter{XRef: "@U1@", Name: "U", NoteXRefs: []string{void}}
	note := &gedcom.Note{XRef: "@N1@", Text: "n", Tags: []*gedcom.Tag{{Level: 1, Tag: "SOUR", Value: void}}}
	snote := &gedcom.SharedNote{XRef: "@N2@", Text: "s", SourceCitations: cite(), ChangeDate: cd()}

	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version70},
		Records: []*gedcom.Record{
			{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: indi},
			{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: fam},
			{XRef: "@S1@", Type: gedcom.RecordTypeSource, Entity: src},
			{XRef: "@M1@", Type: gedcom.RecordTypeMedia, Entity: obje},
			{XRef: "@R1@", Type: gedcom.RecordTypeRepository, Entity: repo},
			{XRef: "@U1@", Type: gedcom.RecordTypeSubmitter, Entity: subm},
			{XRef: "@N1@", Type: gedcom.RecordTypeNote, Entity: note},
			{XRef: "@N2@", Type: gedcom.RecordTypeSharedNote, Entity: snote},
			{XRef: "@I2@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: "@I2@"}},
		},
	}
	doc.XRefMap = make(map[string]*gedcom.Record)
	for _, r := range doc.Records {
		doc.XRefMap[r.XRef] = r
	}

	converted, report, err := converter.Convert(doc, gedcom.Version551)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	for _, r := range converted.Records {
		gedcom.VisitForVersion(r, gedcom.Version551, func(ref string) {
			if ref == void {
				t.Errorf("record %s keeps a @VOID@ pointer", r.XRef)
			}
		})
	}
	if _, err := converted.Subset([]string{"@I1@", "@N1@", "@U1@"}); err != nil {
		t.Errorf("Subset after downgrade: %v", err)
	}
	var buf bytes.Buffer
	if err := encoder.Encode(&buf, converted); err != nil {
		t.Fatalf("encode: %v", err)
	}
	if strings.Contains(buf.String(), void) {
		t.Errorf("output keeps a @VOID@ pointer:\n%s", buf.String())
	}

	got := map[string]bool{}
	for _, n := range report.Dropped {
		if strings.HasSuffix(n.Original, void) {
			got[n.Path] = true
		}
	}
	for _, path := range []string{"Individual @I1@", "Family @F1@", "Source @S1@", "MediaObject @M1@",
		"Repository @R1@", "Submitter @U1@", "Note @N1@", "SNOTE @N2@"} {
		if !got[path] {
			t.Errorf("no @VOID@ drop reported for %s; got %v", path, got)
		}
	}
	if got["Individual @I2@"] {
		t.Error("record without @VOID@ reported")
	}
	if indi.ChildInFamilies[0].FamilyXRef != void || fam.Husband != void {
		t.Error("Convert mutated the input document")
	}
}

// TestDowngradeWithoutVoidReportsNothing checks a document with no @VOID@
// pointer gets no @VOID@ data loss entry.
func TestDowngradeWithoutVoidReportsNothing(t *testing.T) {
	doc, err := decoder.Decode(strings.NewReader("0 HEAD\n1 GEDC\n2 VERS 7.0\n0 @I1@ INDI\n1 NAME A /B/\n0 TRLR\n"))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	_, report, err := converter.Convert(doc, gedcom.Version551)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	for _, l := range report.DataLoss {
		if l.Feature == "@VOID@ null pointers" {
			t.Errorf("unexpected @VOID@ data loss: %+v", l)
		}
	}
}
