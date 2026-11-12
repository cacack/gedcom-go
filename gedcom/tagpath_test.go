package gedcom

import (
	"slices"
	"testing"
)

// pathTestTags is an INDI body with a decoy NAME nested under ASSO, two BIRTs
// (the first without a PLAC), a skipped level and a nil entry.
func pathTestTags() []*Tag {
	return []*Tag{
		{Level: 1, Tag: "NAME", Value: "John /Doe/"},
		{Level: 2, Tag: "GIVN", Value: "John"},
		{Level: 1, Tag: "ASSO", Value: "@I2@"},
		{Level: 2, Tag: "NAME", Value: "decoy"},
		{Level: 1, Tag: "BIRT"},
		{Level: 2, Tag: "DATE", Value: "1900"},
		{Level: 1, Tag: "BIRT"},
		{Level: 2, Tag: "PLAC", Value: "Boston"},
		{Level: 3, Tag: "MAP"},
		{Level: 4, Tag: "LATI", Value: "N42.36"},
		nil,
		{Level: 2, Tag: "SOUR", Value: "@S1@"},
		{Level: 1, Tag: "DEAT"},
		{Level: 3, Tag: "PLAC", Value: "skipped level"},
		{Level: 1, Tag: "NAME", Value: "Johnny /Doe/"},
	}
}

func values(tags []*Tag) []string {
	var out []string
	for _, t := range tags {
		out = append(out, t.Value)
	}
	return out
}

func TestRecordFindAll(t *testing.T) {
	r := &Record{Tags: pathTestTags()}
	tests := []struct {
		name string
		path []string
		want []string
	}{
		{"level-1 names only", []string{"NAME"}, []string{"John /Doe/", "Johnny /Doe/"}},
		{"nested path", []string{"BIRT", "PLAC"}, []string{"Boston"}},
		{"deep path", []string{"BIRT", "PLAC", "MAP", "LATI"}, []string{"N42.36"}},
		{"after nil entry", []string{"BIRT", "SOUR"}, []string{"@S1@"}},
		{"wrong parent", []string{"ASSO", "GIVN"}, nil},
		{"skipped level not matched", []string{"DEAT", "PLAC"}, nil},
		{"no match", []string{"BURI"}, nil},
		{"case-sensitive", []string{"name"}, nil},
		{"empty path", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := values(r.FindAll(tt.path...)); !slices.Equal(got, tt.want) {
				t.Errorf("FindAll(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestRecordFind(t *testing.T) {
	tags := pathTestTags()
	r := &Record{Tags: tags}
	if got := r.Find("BIRT", "PLAC"); got != tags[7] {
		t.Errorf("Find(BIRT, PLAC) = %+v, want the PLAC under the second BIRT", got)
	}
	if got := r.Find("NAME"); got != tags[0] {
		t.Errorf("Find(NAME) = %+v, want the first level-1 NAME", got)
	}
	if got := r.Find("ASSO", "NAME"); got != tags[3] {
		t.Errorf("Find(ASSO, NAME) = %+v, want the NAME under ASSO", got)
	}
	if got := r.Find("BURI"); got != nil {
		t.Errorf("Find(BURI) = %+v, want nil", got)
	}
	if got := r.Find(); got != nil {
		t.Errorf("Find() = %+v, want nil", got)
	}
}

func TestFindAll_LevelZeroEndsBranch(t *testing.T) {
	h := &Header{Tags: []*Tag{
		{Level: 1, Tag: "SOUR", Value: "App"},
		{Level: 0, Tag: "INDI"},
		{Level: 2, Tag: "VERS", Value: "orphan"},
		{Level: 1, Tag: "SOUR", Value: "App2"},
		{Level: 2, Tag: "VERS", Value: "2.0"},
	}}
	if got := values(h.FindAll("SOUR", "VERS")); !slices.Equal(got, []string{"2.0"}) {
		t.Errorf("FindAll(SOUR, VERS) = %q, want [2.0]", got)
	}
}

func TestRecordSubtree(t *testing.T) {
	tags := pathTestTags()
	r := &Record{Tags: tags}

	got := r.Subtree(tags[6]) // second BIRT
	if !slices.Equal(got, tags[6:12]) {
		t.Fatalf("Subtree(BIRT) = %v, want tags[6:12]", got)
	}
	if cap(got) != len(got) {
		t.Errorf("Subtree capacity = %d, want %d", cap(got), len(got))
	}
	_ = append(got, &Tag{Level: 1, Tag: "X"})
	if tags[12].Tag != "DEAT" {
		t.Errorf("append to Subtree result overwrote r.Tags[12]: %+v", tags[12])
	}

	if got := r.Subtree(tags[14]); !slices.Equal(got, tags[14:]) {
		t.Errorf("Subtree(last NAME) = %v, want only that tag", got)
	}
	if got := r.Subtree(&Tag{Level: 1, Tag: "NAME", Value: "John /Doe/"}); got != nil {
		t.Errorf("Subtree(equal but foreign tag) = %v, want nil", got)
	}
	if got := r.Subtree(nil); got != nil {
		t.Errorf("Subtree(nil) = %v, want nil", got)
	}
}

func TestHeaderPathHelpers(t *testing.T) {
	h := &Header{Tags: []*Tag{
		{Level: 1, Tag: "GEDC"},
		{Level: 2, Tag: "VERS", Value: "5.5.1"},
		{Level: 1, Tag: "SOUR", Value: "App"},
		{Level: 2, Tag: "VERS", Value: "1.0"},
		{Level: 2, Tag: "NAME", Value: "My App"},
	}}
	if got := h.Find("SOUR", "VERS"); got == nil || got.Value != "1.0" {
		t.Errorf("Find(SOUR, VERS) = %+v, want 1.0", got)
	}
	if got := values(h.FindAll("VERS")); got != nil {
		t.Errorf("FindAll(VERS) = %q, want none at level 1", got)
	}
	if got := h.Subtree(h.Tags[2]); !slices.Equal(got, h.Tags[2:]) {
		t.Errorf("Subtree(SOUR) = %v, want SOUR and its two children", got)
	}
}

func TestPathHelpers_NilReceiver(t *testing.T) {
	var r *Record
	var h *Header
	tag := &Tag{Level: 1, Tag: "NAME"}
	if r.Find("NAME") != nil || r.FindAll("NAME") != nil || r.Subtree(tag) != nil {
		t.Error("nil *Record path helpers returned non-nil")
	}
	if h.Find("NAME") != nil || h.FindAll("NAME") != nil || h.Subtree(tag) != nil {
		t.Error("nil *Header path helpers returned non-nil")
	}
}
