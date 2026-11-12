package decoder

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Issue #786: the record metadata Individual types -- CHAN, CREA, REFN (with
// TYPE), UID, and a NOTE record's SOUR citations -- is typed on the SUBM, REPO,
// NOTE and SNOTE records too, in the versions that define it.

const recordMetadata551 = `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @U1@ SUBM
1 NAME Submitter
1 _UID 0b9c8e1c-1111-4c4a-9d7e-000000000001
1 CHAN
2 DATE 1 JAN 2000
3 TIME 10:20:30
2 NOTE Changed by hand
0 @R1@ REPO
1 NAME Archive
1 REFN R-1
2 TYPE shelf
1 REFN R-2
1 CHAN
2 DATE 2 FEB 2001
0 @S1@ SOUR
1 TITL Register
0 @N1@ NOTE Note text
1 REFN N-1
2 TYPE note number
1 SOUR @S1@
2 PAGE p. 7
1 CHAN
2 DATE 3 MAR 2002
0 TRLR
`

const recordMetadata70 = `0 HEAD
1 GEDC
2 VERS 7.0
0 @U1@ SUBM
1 NAME Submitter
1 REFN U-1
2 TYPE submitter number
1 UID 0b9c8e1c-1111-4c4a-9d7e-000000000002
1 CHAN
2 DATE 4 APR 2003
3 TIME 01:02:03
1 CREA
2 DATE 5 MAY 2004
0 @R1@ REPO
1 NAME Archive
1 REFN R-1
1 UID 0b9c8e1c-1111-4c4a-9d7e-000000000003
1 UID 0b9c8e1c-1111-4c4a-9d7e-000000000004
1 CHAN
2 DATE 6 JUN 2005
1 CREA
2 DATE 7 JUL 2006
3 TIME 12:00:00
0 @N1@ SNOTE Shared text
1 REFN S-1
2 TYPE shared note number
1 UID 0b9c8e1c-1111-4c4a-9d7e-000000000005
1 CREA
2 DATE 8 AUG 2007
0 TRLR
`

func decodeRecordMetadata(t *testing.T, text string) *gedcom.Document {
	t.Helper()
	res, err := DecodeWithDiagnostics(strings.NewReader(text), nil)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, d := range res.Diagnostics {
		t.Errorf("unexpected diagnostic: %s", d)
	}
	return res.Document
}

func TestRecordMetadataTyped551(t *testing.T) {
	doc := decodeRecordMetadata(t, recordMetadata551)

	subm := doc.GetSubmitter("@U1@")
	if subm == nil {
		t.Fatal("no submitter @U1@")
	}
	wantChan := &gedcom.ChangeDate{Date: "1 JAN 2000", Time: "10:20:30", InlineNotes: []string{"Changed by hand"}}
	if !reflect.DeepEqual(subm.ChangeDate, wantChan) {
		t.Errorf("Submitter.ChangeDate = %+v, want %+v", subm.ChangeDate, wantChan)
	}
	if want := []string{"0b9c8e1c-1111-4c4a-9d7e-000000000001"}; !reflect.DeepEqual(subm.UIDs, want) {
		t.Errorf("Submitter.UIDs = %q, want %q (from _UID)", subm.UIDs, want)
	}

	repo := doc.GetRepository("@R1@")
	if repo == nil {
		t.Fatal("no repository @R1@")
	}
	if want := []gedcom.RefNumber{{Value: "R-1", Type: "shelf"}, {Value: "R-2"}}; !reflect.DeepEqual(repo.RefNumbers, want) {
		t.Errorf("Repository.RefNumbers = %+v, want %+v", repo.RefNumbers, want)
	}
	if repo.ChangeDate == nil || repo.ChangeDate.Date != "2 FEB 2001" {
		t.Errorf("Repository.ChangeDate = %+v, want DATE 2 FEB 2001", repo.ChangeDate)
	}

	note := doc.GetNote("@N1@")
	if note == nil {
		t.Fatal("no note @N1@")
	}
	if want := []gedcom.RefNumber{{Value: "N-1", Type: "note number"}}; !reflect.DeepEqual(note.RefNumbers, want) {
		t.Errorf("Note.RefNumbers = %+v, want %+v", note.RefNumbers, want)
	}
	if len(note.SourceCitations) != 1 || note.SourceCitations[0].SourceXRef != "@S1@" || note.SourceCitations[0].Page != "p. 7" {
		t.Errorf("Note.SourceCitations = %+v, want one citation of @S1@ with PAGE p. 7", note.SourceCitations)
	}
	if note.ChangeDate == nil || note.ChangeDate.Date != "3 MAR 2002" {
		t.Errorf("Note.ChangeDate = %+v, want DATE 3 MAR 2002", note.ChangeDate)
	}
}

func TestRecordMetadataTyped70(t *testing.T) {
	doc := decodeRecordMetadata(t, recordMetadata70)

	subm := doc.GetSubmitter("@U1@")
	if subm == nil {
		t.Fatal("no submitter @U1@")
	}
	if want := []gedcom.RefNumber{{Value: "U-1", Type: "submitter number"}}; !reflect.DeepEqual(subm.RefNumbers, want) {
		t.Errorf("Submitter.RefNumbers = %+v, want %+v", subm.RefNumbers, want)
	}
	if want := []string{"0b9c8e1c-1111-4c4a-9d7e-000000000002"}; !reflect.DeepEqual(subm.UIDs, want) {
		t.Errorf("Submitter.UIDs = %q, want %q", subm.UIDs, want)
	}
	if want := (&gedcom.ChangeDate{Date: "4 APR 2003", Time: "01:02:03"}); !reflect.DeepEqual(subm.ChangeDate, want) {
		t.Errorf("Submitter.ChangeDate = %+v, want %+v", subm.ChangeDate, want)
	}
	if want := (&gedcom.ChangeDate{Date: "5 MAY 2004"}); !reflect.DeepEqual(subm.CreationDate, want) {
		t.Errorf("Submitter.CreationDate = %+v, want %+v", subm.CreationDate, want)
	}

	repo := doc.GetRepository("@R1@")
	if repo == nil {
		t.Fatal("no repository @R1@")
	}
	if want := []gedcom.RefNumber{{Value: "R-1"}}; !reflect.DeepEqual(repo.RefNumbers, want) {
		t.Errorf("Repository.RefNumbers = %+v, want %+v", repo.RefNumbers, want)
	}
	wantUIDs := []string{"0b9c8e1c-1111-4c4a-9d7e-000000000003", "0b9c8e1c-1111-4c4a-9d7e-000000000004"}
	if !reflect.DeepEqual(repo.UIDs, wantUIDs) {
		t.Errorf("Repository.UIDs = %q, want %q", repo.UIDs, wantUIDs)
	}
	if want := (&gedcom.ChangeDate{Date: "6 JUN 2005"}); !reflect.DeepEqual(repo.ChangeDate, want) {
		t.Errorf("Repository.ChangeDate = %+v, want %+v", repo.ChangeDate, want)
	}
	if want := (&gedcom.ChangeDate{Date: "7 JUL 2006", Time: "12:00:00"}); !reflect.DeepEqual(repo.CreationDate, want) {
		t.Errorf("Repository.CreationDate = %+v, want %+v", repo.CreationDate, want)
	}

	snote := doc.GetSharedNote("@N1@")
	if snote == nil {
		t.Fatal("no shared note @N1@")
	}
	if want := []gedcom.RefNumber{{Value: "S-1", Type: "shared note number"}}; !reflect.DeepEqual(snote.RefNumbers, want) {
		t.Errorf("SharedNote.RefNumbers = %+v, want %+v", snote.RefNumbers, want)
	}
	if want := []string{"0b9c8e1c-1111-4c4a-9d7e-000000000005"}; !reflect.DeepEqual(snote.UIDs, want) {
		t.Errorf("SharedNote.UIDs = %q, want %q", snote.UIDs, want)
	}
	if want := (&gedcom.ChangeDate{Date: "8 AUG 2007"}); !reflect.DeepEqual(snote.CreationDate, want) {
		t.Errorf("SharedNote.CreationDate = %+v, want %+v", snote.CreationDate, want)
	}
}
