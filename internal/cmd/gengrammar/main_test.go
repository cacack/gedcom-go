package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWritesTables(t *testing.T) {
	out := filepath.Join(t.TempDir(), "grammar_tables.go")
	var stderr bytes.Buffer
	if code := run([]string{"-spec", "../../../testdata/spec", "-out", out}, &stderr); code != 0 {
		t.Fatalf("run = %d: %s", code, stderr.String())
	}
	got, err := os.ReadFile(out) // #nosec G304 -- test temp file
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("../../../validator/grammar_tables.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Error("output differs from validator/grammar_tables.go")
	}
}

func TestRunErrors(t *testing.T) {
	for name, tc := range map[string]struct {
		args []string
		code int
		want string
	}{
		"bad flag":   {[]string{"-nope"}, 2, "flag provided but not defined"},
		"bad spec":   {[]string{"-spec", filepath.Join(t.TempDir(), "absent")}, 1, "gengrammar:"},
		"bad output": {[]string{"-spec", "../../../testdata/spec", "-out", filepath.Join(t.TempDir(), "absent", "x.go")}, 1, "gengrammar:"},
	} {
		t.Run(name, func(t *testing.T) {
			var stderr bytes.Buffer
			if code := run(tc.args, &stderr); code != tc.code || !strings.Contains(stderr.String(), tc.want) {
				t.Errorf("run = %d, %q; want %d containing %q", code, stderr.String(), tc.code, tc.want)
			}
		})
	}
}
