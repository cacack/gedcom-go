package converter

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestUpgradeLeavesExtensionSubtreesAlone: an inline OBJE or inline-text SOUR
// in an extension record or anywhere beneath an extension tag is not a GEDCOM
// structure, so a 5.x -> 7.0 upgrade leaves it unchanged and adds no record
// for it, while the same structures outside an extension are still promoted
// (issue #840).
func TestUpgradeLeavesExtensionSubtreesAlone(t *testing.T) {
	const extIndi = "1 _PHOTO\n2 OBJE\n3 FILE ext.jpg\n4 FORM jpg\n" +
		"1 _SRC note\n2 SOUR Vendor text\n3 TEXT kept\n"
	const extBirt = "2 _MEDIA\n3 _WRAP\n4 OBJE\n5 FILE deep.jpg\n4 SOUR Deep vendor text\n"
	const extRecord = "0 @X1@ _CUSTOM\n1 OBJE\n2 FILE rec.jpg\n1 SOUR Record vendor text\n"
	doc := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
		"0 @I1@ INDI\n1 NAME John /Doe/\n" + extIndi +
		"1 BIRT\n" + extBirt +
		"1 OBJE\n2 FILE photo.jpg\n3 FORM jpg\n" +
		"1 SOUR Family bible\n" +
		extRecord +
		"0 TRLR\n"

	converted, report, out := convertText(t, doc, gedcom.Version70)

	for _, want := range []string{extIndi, "1 BIRT\n" + extBirt, extRecord} {
		if !strings.Contains(out, want) {
			t.Errorf("extension subtree changed; output lacks %q:\n%s", want, out)
		}
	}

	// Outside the extensions, both are still promoted, with their notes.
	for _, want := range []string{
		"1 OBJE @O1@\n1 SOUR @S1@\n",
		"0 @O1@ OBJE\n1 FILE photo.jpg\n2 FORM image/jpeg\n", // FORM mapped to 7.0 by #777
		"0 @S1@ SOUR\n1 NOTE Family bible\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	var types []string
	for _, r := range converted.Records {
		types = append(types, r.XRef+" "+string(r.Type))
	}
	if got, want := strings.Join(types, ","), "@S1@ SOUR,@I1@ INDI,@X1@ _CUSTOM,@O1@ OBJE"; got != want {
		t.Errorf("records = %s, want %s", got, want)
	}
	var notes []string
	for _, n := range report.Normalized {
		if strings.HasPrefix(n.Original, "OBJE") || strings.HasPrefix(n.Original, "SOUR") {
			notes = append(notes, n.Path+" -> "+n.Result)
		}
	}
	if got, want := strings.Join(notes, "; "), "Individual @I1@ > OBJE -> OBJE @O1@; Individual @I1@ > SOUR -> SOUR @S1@"; got != want {
		t.Errorf("promotion notes = %q, want %q", got, want)
	}
}
