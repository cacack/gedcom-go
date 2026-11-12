package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWritesTheConverterTable(t *testing.T) {
	out := filepath.Join(t.TempDir(), "structures_gen.go")
	var stderr bytes.Buffer
	if code := run([]string{"-spec", "../../../testdata/spec", "-out", out}, &stderr); code != 0 {
		t.Fatalf("run = %d, stderr %q", code, stderr.String())
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("../../../converter/structures_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Error("output differs from converter/structures_gen.go")
	}
}

func TestRunReportsErrors(t *testing.T) {
	var stderr bytes.Buffer
	if code := run([]string{"-spec", t.TempDir()}, &stderr); code != 1 || !strings.Contains(stderr.String(), "genforeign:") {
		t.Errorf("missing spec: run = %d, stderr %q", code, stderr.String())
	}
	stderr.Reset()
	if code := run([]string{"-bogus"}, &stderr); code != 2 {
		t.Errorf("bad flag: run = %d", code)
	}
}
