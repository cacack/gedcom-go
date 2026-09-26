package encoder

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestRepositoryToTagsContactOrder checks that every contact entry is written,
// in order, after ADDR and before NOTE (#494, #508).
func TestRepositoryToTagsContactOrder(t *testing.T) {
	repo := &gedcom.Repository{
		Name:        "Archive",
		Address:     &gedcom.Address{City: "Springfield"},
		Phone:       []string{"p1", "p2", "p3"},
		Email:       []string{"e1", "e2", "e3"},
		Fax:         []string{"f1"},
		Website:     []string{"w1"},
		InlineNotes: []string{"n1"},
	}
	var got []string
	for _, tag := range repositoryToTags(repo, nil) {
		if tag.Level == 1 {
			got = append(got, tag.Tag+" "+tag.Value)
		}
	}
	want := []string{
		"NAME Archive", "ADDR ",
		"PHON p1", "PHON p2", "PHON p3",
		"EMAIL e1", "EMAIL e2", "EMAIL e3",
		"FAX f1", "WWW w1", "NOTE n1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("level-1 tags =\n%q\nwant\n%q", got, want)
	}
}

// TestRepositoryContactEntityRoundTrip decodes a repository carrying
// 3x PHON, 3x EMAIL, FAX and WWW, drops the raw tags so the encoder must
// rebuild the record from the typed model, and requires byte-identical output.
func TestRepositoryContactEntityRoundTrip(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @R1@ REPO
1 NAME County Archive
1 ADDR 1 Main St
2 ADR1 1 Main St
2 CITY Springfield
1 PHON 555-0001
1 PHON 555-0002
1 PHON 555-0003
1 EMAIL a@example.org
1 EMAIL b@example.org
1 EMAIL c@example.org
1 FAX 555-0100
1 WWW https://archive.example.org
0 TRLR
`
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	for _, rec := range doc.Records {
		if rec.Type == gedcom.RecordTypeRepository {
			rec.Tags = nil
		}
	}

	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, doc, &EncodeOptions{LineEnding: "\n"}); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	if buf.String() != input {
		t.Errorf("entity round trip not byte-identical\ngot:\n%s\nwant:\n%s", buf.String(), input)
	}

	redoc, err := decoder.Decode(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatalf("re-Decode() error = %v", err)
	}
	orig, again := doc.GetRepository("@R1@"), redoc.GetRepository("@R1@")
	if again == nil {
		t.Fatal("repository missing after round trip")
	}
	if !reflect.DeepEqual(orig.Phone, again.Phone) || !reflect.DeepEqual(orig.Email, again.Email) ||
		!reflect.DeepEqual(orig.Fax, again.Fax) || !reflect.DeepEqual(orig.Website, again.Website) ||
		*orig.Address != *again.Address {
		t.Errorf("typed contact fields changed across round trip:\n%+v\n%+v", orig, again)
	}
}
