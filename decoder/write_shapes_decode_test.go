package decoder

import (
	"reflect"
	"strings"
	"testing"
)

// Issue #740: the structures the entity encoder needs typed to write a
// decoded record back for its version -- event FAMC, STAT.DATE, ASSO.TYPE.

func decodeWithUnknownLines(t *testing.T, in string) (res *DecodeResult, unknown []int) {
	t.Helper()
	res, err := DecodeWithDiagnostics(strings.NewReader(in), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range res.Diagnostics {
		if d.Code == CodeUnknownTag {
			unknown = append(unknown, d.Line)
		}
	}
	return res, unknown
}

// TestEventFamilyLinkDecoded checks that BIRT, CHR and ADOP FAMC are typed
// and not reported, and that a FAMC under any other event still is.
func TestEventFamilyLinkDecoded(t *testing.T) {
	const in = "0 HEAD\n1 GEDC\n2 VERS 7.0\n" +
		"0 @I1@ INDI\n1 BIRT\n2 FAMC @F1@\n1 CHR\n2 FAMC @F1@\n" +
		"1 ADOP\n2 FAMC @F1@\n3 ADOP HUSB\n4 PHRASE Father\n1 DEAT\n2 FAMC @F1@\n" +
		"0 @F1@ FAM\n0 TRLR\n"
	res, unknown := decodeWithUnknownLines(t, in)
	ev := res.Document.GetIndividual("@I1@").Events
	if ev[0].FamilyXRef != "@F1@" || ev[1].FamilyXRef != "@F1@" {
		t.Errorf("BIRT, CHR FamilyXRef = %q, %q", ev[0].FamilyXRef, ev[1].FamilyXRef)
	}
	if e := ev[2]; e.FamilyXRef != "@F1@" || e.AdoptedBy != "HUSB" || e.AdoptedByPhrase != "Father" {
		t.Errorf("ADOP FamilyXRef, AdoptedBy, AdoptedByPhrase = %q, %q, %q", e.FamilyXRef, e.AdoptedBy, e.AdoptedByPhrase)
	}
	if got := ev[3].FamilyXRef; got != "" {
		t.Errorf("DEAT FamilyXRef = %q, want empty: only BIRT, CHR and ADOP have one", got)
	}
	if !reflect.DeepEqual(unknown, []int{14}) {
		t.Errorf("UNKNOWN_TAG on lines %v, want [14] (DEAT.FAMC only)", unknown)
	}
}

func TestLDSStatusDateDecoded(t *testing.T) {
	const in = "0 HEAD\n1 GEDC\n2 VERS 7.0\n" +
		"0 @I1@ INDI\n1 BAPL\n2 STAT COMPLETED\n3 DATE 27 MAR 2022\n4 TIME 15:48\n0 TRLR\n"
	res, _ := decodeWithUnknownLines(t, in)
	ord := res.Document.GetIndividual("@I1@").LDSOrdinances[0]
	if ord.Status != "COMPLETED" || ord.StatusDate != "27 MAR 2022" || ord.StatusTime != "15:48" {
		t.Errorf("Status, StatusDate, StatusTime = %q, %q, %q", ord.Status, ord.StatusDate, ord.StatusTime)
	}
}

// TestAssociationRecordTypeDecoded: TYPE is kept in every version, and
// reported only where the declared version does not define it.
func TestAssociationRecordTypeDecoded(t *testing.T) {
	for version, wantUnknown := range map[string][]int{"5.5": nil, "5.5.1": {8}} {
		t.Run(version, func(t *testing.T) {
			in := "0 HEAD\n1 GEDC\n2 VERS " + version + "\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
				"0 @I1@ INDI\n1 ASSO @I2@\n2 TYPE INDI\n2 RELA Godfather\n0 @I2@ INDI\n0 TRLR\n"
			res, unknown := decodeWithUnknownLines(t, in)
			assoc := res.Document.GetIndividual("@I1@").Associations[0]
			if assoc.RecordType != "INDI" || assoc.Role != "Godfather" {
				t.Errorf("RecordType, Role = %q, %q", assoc.RecordType, assoc.Role)
			}
			if !reflect.DeepEqual(unknown, wantUnknown) {
				t.Errorf("UNKNOWN_TAG on lines %v, want %v", unknown, wantUnknown)
			}
		})
	}
}
