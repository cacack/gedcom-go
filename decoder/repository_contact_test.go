package decoder

import (
	"reflect"
	"strings"
	"testing"
)

// TestRepositoryContactSlices checks that the {0:3} contact tags on a REPO
// record accumulate in document order rather than last-writer-wins, that FAX
// reaches the typed model, and that none of them allocates an Address (#494,
// #508).
func TestRepositoryContactSlices(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @R1@ REPO
1 NAME County Archive
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
	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	repo := result.Document.GetRepository("@R1@")
	if repo == nil {
		t.Fatal("GetRepository(@R1@) returned nil")
	}

	checks := []struct {
		field string
		got   []string
		want  []string
	}{
		{"Phone", repo.Phone, []string{"555-0001", "555-0002", "555-0003"}},
		{"Email", repo.Email, []string{"a@example.org", "b@example.org", "c@example.org"}},
		{"Fax", repo.Fax, []string{"555-0100"}},
		{"Website", repo.Website, []string{"https://archive.example.org"}},
	}
	for _, c := range checks {
		if !reflect.DeepEqual(c.got, c.want) {
			t.Errorf("repo.%s = %q, want %q", c.field, c.got, c.want)
		}
	}

	if repo.Address != nil {
		t.Errorf("repo.Address = %+v, want nil: contact tags are siblings of ADDR, not part of it", repo.Address)
	}

	for _, d := range result.Diagnostics {
		if d.Code == CodeUnknownTag {
			t.Errorf("unexpected UNKNOWN_TAG diagnostic: %s", d.Message)
		}
	}
}

// TestRepositoryContactWithAddress checks that contact tags and ADDR coexist:
// ADDR subordinates go to Address, and the siblings go to the Repository.
func TestRepositoryContactWithAddress(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @R1@ REPO
1 NAME County Archive
1 ADDR 1 Main St
2 CITY Springfield
1 PHON 555-0001
1 FAX 555-0100
0 TRLR
`
	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	repo := doc.GetRepository("@R1@")
	if repo == nil || repo.Address == nil {
		t.Fatalf("repository or address missing: %+v", repo)
	}
	if repo.Address.City != "Springfield" {
		t.Errorf("Address.City = %q, want Springfield", repo.Address.City)
	}
	if !reflect.DeepEqual(repo.Phone, []string{"555-0001"}) {
		t.Errorf("Phone = %q", repo.Phone)
	}
	if !reflect.DeepEqual(repo.Fax, []string{"555-0100"}) {
		t.Errorf("Fax = %q", repo.Fax)
	}
}
