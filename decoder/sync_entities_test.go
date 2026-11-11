package decoder

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

func TestSyncEntitiesNil(t *testing.T) {
	SyncEntities(nil)                                                                          // must not panic
	SyncEntities(&gedcom.Document{Records: []*gedcom.Record{nil}})                             // nil header, nil record
	SyncEntities(&gedcom.Document{Records: []*gedcom.Record{{Type: gedcom.RecordTypeFamily}}}) // no Tags
}

// TestSyncEntitiesSkipsNilTags asserts a hand-built record holding nil tags is
// parsed without them rather than panicking, and its Tags are left as they are.
func TestSyncEntitiesSkipsNilTags(t *testing.T) {
	rec := &gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Tags: []*gedcom.Tag{nil, {Level: 1, Tag: "SEX", Value: "M"}, nil}}
	SyncEntities(&gedcom.Document{Records: []*gedcom.Record{rec}})
	indi, ok := rec.Entity.(*gedcom.Individual)
	if !ok || indi.Sex != "M" || indi.XRef != "@I1@" {
		t.Errorf("Entity = %+v, want an individual with Sex M", rec.Entity)
	}
	if len(rec.Tags) != 3 {
		t.Errorf("record Tags changed: %d, want 3", len(rec.Tags))
	}
}

// TestSyncEntitiesRebuildsFromTags asserts a tagged record's Entity is rebuilt
// from its Tags, discarding an edit made only to the Entity, while a tagless
// record keeps its Entity as is.
func TestSyncEntitiesRebuildsFromTags(t *testing.T) {
	doc, err := Decode(strings.NewReader("0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @I1@ INDI\n1 SEX M\n0 TRLR\n"))
	if err != nil {
		t.Fatal(err)
	}
	doc.GetIndividual("@I1@").Sex = "F" // an edit not written back to Tags
	handBuilt := &gedcom.Individual{XRef: "@I2@", Sex: "U"}
	doc.Records = append(doc.Records, &gedcom.Record{XRef: "@I2@", Type: gedcom.RecordTypeIndividual, Entity: handBuilt})

	SyncEntities(doc)

	if got := doc.Records[0].Entity.(*gedcom.Individual).Sex; got != "M" {
		t.Errorf("tagged record Sex = %q, want %q from its Tags", got, "M")
	}
	if doc.Records[1].Entity != handBuilt {
		t.Errorf("tagless record Entity replaced: %+v", doc.Records[1].Entity)
	}
}

// TestSyncEntitiesReadsByHeaderVersion asserts text is unescaped by the
// header's version, as Decode does (#708): "a@@b" is "a@b" in 5.5.1, and kept
// as is in 7.0, where only a leading "@@" is an escape.
func TestSyncEntitiesReadsByHeaderVersion(t *testing.T) {
	for _, tt := range []struct {
		v    gedcom.Version
		want string
	}{
		{gedcom.Version551, "a@b"},
		{gedcom.Version70, "a@@b"},
	} {
		doc := &gedcom.Document{
			Header: &gedcom.Header{Version: tt.v, DeclaredVersion: tt.v},
			Records: []*gedcom.Record{{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Tags: []*gedcom.Tag{
				{Level: 1, Tag: "OCCU", Value: "a@@b"},
			}}},
		}
		SyncEntities(doc)
		indi := doc.Records[0].Entity.(*gedcom.Individual)
		if len(indi.Attributes) != 1 || indi.Attributes[0].Value != tt.want {
			t.Errorf("%s: Attributes = %+v, want OCCU %q", tt.v, indi.Attributes, tt.want)
		}
	}
}

// TestAddressFoldedPayloadDropsLeadingEmptyLines asserts that an ADDR payload
// folded into one value reads as the same lines unfolded: an empty ADDR line
// followed by CONT lines drops the leading empty lines either way (#772).
func TestAddressFoldedPayloadDropsLeadingEmptyLines(t *testing.T) {
	for _, tt := range []struct {
		name string
		tags []*gedcom.Tag
		want string
	}{
		{"unfolded", []*gedcom.Tag{
			{Level: 1, Tag: "ADDR"}, {Level: 2, Tag: "CONT"}, {Level: 2, Tag: "CONT", Value: "1 Main St"},
		}, "1 Main St"},
		{"folded", []*gedcom.Tag{{Level: 1, Tag: "ADDR", Value: "\n\n1 Main St"}}, "1 Main St"},
		{"folded, empty", []*gedcom.Tag{{Level: 1, Tag: "ADDR", Value: "\n\n"}}, ""},
		{"inner newline kept", []*gedcom.Tag{{Level: 1, Tag: "ADDR", Value: "1 Main St\n\nTown"}}, "1 Main St\n\nTown"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseEventAddress(tt.tags, 0, 1, nil).Full; got != tt.want {
				t.Errorf("Full = %q, want %q", got, tt.want)
			}
		})
	}
}
