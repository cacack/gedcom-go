package converter_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/converter"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Issue #786: SUBM, REPO, NOTE and SNOTE type CHAN, CREA, REFN, UID and NOTE
// SOUR. A record with no Tags is converted from its entity, so these fields
// must follow the record across versions as Individual's do.

// TestDowngrade70ClearsRecordMetadataCREA checks that CREA, which 5.x does
// not define, is cleared from an entity-only SUBM, REPO and SNOTE on a
// downgrade and reported, while their CHAN and REFN are kept.
func TestDowngrade70ClearsRecordMetadataCREA(t *testing.T) {
	crea := func() *gedcom.ChangeDate { return &gedcom.ChangeDate{Date: "1 JAN 2020"} }
	chg := func() *gedcom.ChangeDate { return &gedcom.ChangeDate{Date: "2 FEB 2021"} }
	refs := []gedcom.RefNumber{{Value: "1", Type: "user"}}
	subm := &gedcom.Submitter{XRef: "@U1@", Name: "Submitter", CreationDate: crea(), ChangeDate: chg()}
	repo := &gedcom.Repository{XRef: "@R1@", Name: "Archive", CreationDate: crea(), ChangeDate: chg(), RefNumbers: refs}
	snote := &gedcom.SharedNote{XRef: "@N1@", Text: "Shared", CreationDate: crea(), ChangeDate: chg(), RefNumbers: refs}
	records := []*gedcom.Record{
		{XRef: "@U1@", Type: gedcom.RecordTypeSubmitter, Entity: subm},
		{XRef: "@R1@", Type: gedcom.RecordTypeRepository, Entity: repo},
		{XRef: "@N1@", Type: gedcom.RecordTypeSharedNote, Entity: snote},
	}
	doc := &gedcom.Document{Header: &gedcom.Header{Version: gedcom.Version70}, Records: records, XRefMap: map[string]*gedcom.Record{}}
	for _, r := range records {
		doc.XRefMap[r.XRef] = r
	}

	converted, report, err := converter.Convert(doc, gedcom.Version551)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	var buf bytes.Buffer
	if err := encoder.EncodeWithOptions(&buf, converted, &encoder.EncodeOptions{TargetVersion: gedcom.Version551}); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	out := buf.String()
	if recordLineTags(out)["CREA"] {
		t.Errorf("output writes dropped CREA:\n%s", out)
	}
	for _, xref := range []string{"@U1@", "@R1@", "@N1@"} {
		if !hasDataLoss(report, "CREA tags", xref) {
			t.Errorf("CREA not reported as data loss for %s", xref)
		}
	}
	for _, want := range []string{
		"0 @U1@ SUBM\n1 NAME Submitter\n1 CHAN\n2 DATE 2 FEB 2021\n",
		"0 @R1@ REPO\n1 NAME Archive\n1 REFN 1\n2 TYPE user\n1 CHAN\n2 DATE 2 FEB 2021\n",
		"0 @N1@ NOTE Shared\n1 REFN 1\n2 TYPE user\n1 CHAN\n2 DATE 2 FEB 2021\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing\n%s\ngot:\n%s", want, out)
		}
	}
	if converted.RequiresGEDCOM7() {
		t.Error("RequiresGEDCOM7() = true after downgrade, want false")
	}
	if subm.CreationDate == nil || repo.CreationDate == nil || snote.CreationDate == nil {
		t.Error("Convert changed the source document's entities")
	}
}

// TestUpgradeEntityOnlyNoteKeepsMetadata checks that an entity-only 5.5.1
// NOTE record upgraded to a 7.0 SNOTE keeps its citations, CHAN and REFN.
func TestUpgradeEntityOnlyNoteKeepsMetadata(t *testing.T) {
	note := &gedcom.Note{
		XRef:            "@N1@",
		Text:            "Note text",
		SourceCitations: []*gedcom.SourceCitation{{SourceXRef: "@S1@", Page: "p. 7"}},
		ChangeDate:      &gedcom.ChangeDate{Date: "3 MAR 2002"},
		RefNumbers:      []gedcom.RefNumber{{Value: "N-1", Type: "note number"}},
	}
	src := &gedcom.Source{XRef: "@S1@", Title: "Register"}
	records := []*gedcom.Record{
		{XRef: "@S1@", Type: gedcom.RecordTypeSource, Entity: src},
		{XRef: "@N1@", Type: gedcom.RecordTypeNote, Entity: note},
	}
	doc := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version551},
		Records: records,
		XRefMap: map[string]*gedcom.Record{"@S1@": records[0], "@N1@": records[1]},
	}

	converted, _, err := converter.Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	snote := converted.GetSharedNote("@N1@")
	if snote == nil {
		t.Fatal("@N1@ is not a shared note after the upgrade")
	}
	if !reflect.DeepEqual(snote.SourceCitations, note.SourceCitations) {
		t.Errorf("SourceCitations = %+v, want %+v", snote.SourceCitations, note.SourceCitations)
	}
	if !reflect.DeepEqual(snote.ChangeDate, note.ChangeDate) {
		t.Errorf("ChangeDate = %+v, want %+v", snote.ChangeDate, note.ChangeDate)
	}
	if !reflect.DeepEqual(snote.RefNumbers, note.RefNumbers) {
		t.Errorf("RefNumbers = %+v, want %+v", snote.RefNumbers, note.RefNumbers)
	}
}
