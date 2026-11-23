package specgraph

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPlacementOf(t *testing.T) {
	g := Graph{
		"":         {"INDI": {"rec-INDI"}, "SOUR": {"rec-SOUR"}},
		"rec-INDI": {"BIRT": {"BIRT"}, "SOUR": {"cite"}},
		"rec-SOUR": {"TITL": {"TITL"}},
		"BIRT":     {"DATE": {"DATE"}, "SOUR": {"cite"}},
		"cite":     {"PAGE": {"PAGE"}},
	}
	got := PlacementOf(g)
	want := Placement{
		Tags:     []string{"BIRT", "DATE", "INDI", "PAGE", "SOUR", "TITL"},
		Records:  map[string][]string{"INDI": {"BIRT", "SOUR"}, "SOUR": {"TITL"}},
		Children: map[string][]string{"BIRT": {"DATE", "SOUR"}, "SOUR": {"PAGE"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("PlacementOf =\n%+v\nwant\n%+v", got, want)
	}
}

func TestGeneratePlacement(t *testing.T) {
	src, err := GeneratePlacement("../../testdata/spec")
	if err != nil {
		t.Fatal(err)
	}
	flat := strings.Join(strings.Fields(string(src)), " ")
	for _, want := range []string{
		"var placement55 = &tagPlacement{",
		"var placement551 = &tagPlacement{",
		"var placement70 = &tagPlacement{",
		// 5.5.1 adds RESI and RESN to FAM.
		`"FAM": {"ANUL", "CENS", "CHAN", "CHIL", "DIV", "DIVF", "ENGA", "EVEN", "HUSB", "MARB", "MARC", "MARL", "MARR", "MARS", "NCHI", "NOTE", "OBJE", "REFN", "RESI", "RESN", "RIN",`,
	} {
		if !strings.Contains(flat, want) {
			t.Errorf("generated source lacks %s", want)
		}
	}
	want, err := os.ReadFile("../../decoder/placement_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(src, want) {
		t.Error("decoder/placement_gen.go is stale; run go generate ./decoder")
	}
}

func TestGeneratePlacementMissingSpec(t *testing.T) {
	dir := t.TempDir()
	if _, err := GeneratePlacement(dir); err == nil {
		t.Error("GeneratePlacement of an empty directory succeeded")
	}
	// 5.5 present, 5.5.1 missing.
	if err := os.MkdirAll(filepath.Join(dir, "gedcom-5.5"), 0o750); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("../../testdata/spec/gedcom-5.5/substructures.tsv")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gedcom-5.5", "substructures.tsv"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := GeneratePlacement(dir); err == nil {
		t.Error("GeneratePlacement without 5.5.1 succeeded")
	}
}
