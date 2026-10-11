package gedcom

import (
	"slices"
	"testing"
)

// spacedXRefDocument links @I1@ to the note @N 1@, a spaced identifier
// GEDCOM 5.5 and 5.5.1 allow, through a typed field and a raw tag, under the
// given declared version.
func spacedXRefDocument(declared Version) *Document {
	note := &Record{XRef: "@N 1@", Type: RecordTypeNote, Entity: &Note{XRef: "@N 1@", Text: "Shared"}}
	person := &Record{
		XRef:   "@I1@",
		Type:   RecordTypeIndividual,
		Entity: &Individual{XRef: "@I1@", NoteXRefs: []string{"@N 1@"}},
		Tags:   []*Tag{{Level: 1, Tag: "NOTE", Value: "@N 1@"}},
	}
	return &Document{
		Header:  &Header{Version: Version551, DeclaredVersion: declared},
		Records: []*Record{person, note},
		XRefMap: map[string]*Record{person.XRef: person, note.XRef: note},
	}
}

func TestDocumentIsPointerXRef(t *testing.T) {
	tests := []struct {
		name string
		doc  *Document
		want bool
	}{
		{"nil document", nil, false},
		{"nil header", &Document{}, false},
		{"undeclared", &Document{Header: &Header{Version: Version551}}, false},
		{"5.5", &Document{Header: &Header{DeclaredVersion: Version55}}, true},
		{"5.5.1", &Document{Header: &Header{DeclaredVersion: Version551}}, true},
		{"7.0", &Document{Header: &Header{DeclaredVersion: Version70}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.doc.IsPointerXRef("@N 1@"); got != tt.want {
				t.Errorf("IsPointerXRef(@N 1@) = %v, want %v", got, tt.want)
			}
			if !tt.doc.IsPointerXRef("@N1@") {
				t.Error("IsPointerXRef(@N1@) = false, want true")
			}
		})
	}
}

// TestVisitForVersionSpacedPointer pins issue #591: VisitForVersion follows a
// spaced pointer under 5.5/5.5.1 and not under 7.0, as Visit never does.
func TestVisitForVersionSpacedPointer(t *testing.T) {
	r := spacedXRefDocument("").GetRecord("@I1@")
	collect := func(walk func(func(string))) []string {
		var got []string
		walk(func(ref string) { got = append(got, ref) })
		return got
	}
	for _, v := range []Version{Version55, Version551} {
		got := collect(func(f func(string)) { VisitForVersion(r, v, f) })
		if want := []string{"@N 1@", "@N 1@"}; !slices.Equal(got, want) {
			t.Errorf("VisitForVersion(%s) = %q, want %q", v, got, want)
		}
	}
	if got := collect(func(f func(string)) { VisitForVersion(r, Version70, f) }); len(got) != 0 {
		t.Errorf("VisitForVersion(7.0) = %q, want none", got)
	}
	if got := collect(func(f func(string)) { Visit(r, f) }); len(got) != 0 {
		t.Errorf("Visit = %q, want none", got)
	}
}

// TestSubsetSpacedPointer pins issue #591: a 5.5/5.5.1 subset accepts a
// spaced seed and follows spaced pointers into the closure.
func TestSubsetSpacedPointer(t *testing.T) {
	src := spacedXRefDocument(Version551)
	sub, err := src.Subset([]string{"@I1@"})
	if err != nil {
		t.Fatalf("Subset: %v", err)
	}
	if sub.GetNote("@N 1@") == nil {
		t.Error("note @N 1@ not pulled into the subset")
	}
	if sub.Header.DeclaredVersion != Version551 {
		t.Errorf("DeclaredVersion = %q, want %q", sub.Header.DeclaredVersion, Version551)
	}
	if _, err := src.Subset([]string{"@N 1@"}); err != nil {
		t.Errorf("Subset with spaced seed: %v", err)
	}
}

// TestSubsetSpacedValue70 pins that under GEDCOM 7.0 a spaced value is not a
// pointer: it is neither followed nor accepted as a seed.
func TestSubsetSpacedValue70(t *testing.T) {
	src := spacedXRefDocument(Version70)
	sub, err := src.Subset([]string{"@I1@"})
	if err != nil {
		t.Fatalf("Subset: %v", err)
	}
	if sub.GetNote("@N 1@") != nil {
		t.Error("note @N 1@ pulled into a 7.0 subset")
	}
	if _, err := src.Subset([]string{"@N 1@"}); err == nil {
		t.Error("Subset with spaced seed succeeded, want error")
	}
}
