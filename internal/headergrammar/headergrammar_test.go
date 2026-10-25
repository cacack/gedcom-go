package headergrammar

import (
	"slices"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

func TestRank(t *testing.T) {
	tests := []struct {
		version gedcom.Version
		tag     string
		want    int
	}{
		{gedcom.Version551, "SOUR", 0},
		{gedcom.Version551, "GEDC", 7},
		{gedcom.Version55, "CHAR", 8},
		{gedcom.Version70, "GEDC", 0},
		{gedcom.Version70, "SNOTE", 9}, // shares NOTE's position
		{gedcom.Version70, "CHAR", 11}, // not in the 7.0 grammar
		{gedcom.Version551, "_EXT", 12},
		{"5.5.5", "GEDC", 7}, // unmodelled versions take the 5.5 order
	}
	for _, tt := range tests {
		if got := Rank(tt.version, tt.tag); got != tt.want {
			t.Errorf("Rank(%s, %s) = %d, want %d", tt.version, tt.tag, got, tt.want)
		}
	}
}

func TestSort(t *testing.T) {
	orphan := &gedcom.Tag{Level: 2, Tag: "VERS", Value: "orphan"}
	ext := &gedcom.Tag{Level: 1, Tag: "_EXT"}
	char := &gedcom.Tag{Level: 1, Tag: "CHAR", Value: "UTF-8"}
	sour := &gedcom.Tag{Level: 1, Tag: "SOUR", Value: "X"}
	sourVers := &gedcom.Tag{Level: 2, Tag: "VERS", Value: "1"}
	gedc := &gedcom.Tag{Level: 1, Tag: "GEDC"}
	gedcVers := &gedcom.Tag{Level: 2, Tag: "VERS", Value: "5.5.1"}
	note := &gedcom.Tag{Level: 0, Tag: "NOTE"} // hand-built: Level unset

	input := []*gedcom.Tag{orphan, ext, char, nil, sour, sourVers, note, gedc, gedcVers}
	original := slices.Clone(input)

	tests := []struct {
		version gedcom.Version
		want    []*gedcom.Tag
	}{
		// The nil travels with CHAR, whose subtree it sits in; the orphan stays
		// first; the extension goes after every defined structure.
		{gedcom.Version551, []*gedcom.Tag{orphan, sour, sourVers, gedc, gedcVers, char, nil, note, ext}},
		// CHAR and _EXT are both undefined in 7.0 and keep their relative order.
		{gedcom.Version70, []*gedcom.Tag{orphan, gedc, gedcVers, sour, sourVers, note, ext, char, nil}},
	}
	for _, tt := range tests {
		got := Sort(input, tt.version)
		if !slices.Equal(got, tt.want) {
			t.Errorf("Sort(%s) = %v, want %v", tt.version, got, tt.want)
		}
		if !slices.Equal(input, original) {
			t.Fatalf("Sort(%s) modified its input", tt.version)
		}
	}

	if got := Sort(nil, gedcom.Version551); len(got) != 0 {
		t.Errorf("Sort(nil) = %v, want empty", got)
	}
}

func TestRequiresGEDCForm(t *testing.T) {
	cases := map[gedcom.Version]bool{
		gedcom.Version55:  true,
		gedcom.Version551: true,
		gedcom.Version70:  false,
		"":                false,
		"9.9":             false,
	}
	for v, want := range cases {
		if got := RequiresGEDCForm(v); got != want {
			t.Errorf("RequiresGEDCForm(%q) = %v, want %v", v, got, want)
		}
	}
}
