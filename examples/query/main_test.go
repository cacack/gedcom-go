package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMain_NegativeAssertionsNotPrintedAsEvents runs the example against a
// GEDCOM 7.0 file whose first individual has a NO BIRT and a NO DEAT: neither
// may be printed as a birth or an event that happened, only as a negative
// assertion.
func TestMain_NegativeAssertionsNotPrintedAsEvents(t *testing.T) {
	const ged = `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME John /Doe/
1 NO BIRT
2 DATE FROM 1800 TO 1810
1 BIRT
2 DATE 1 JAN 1920
1 NO DEAT
2 DATE FROM 1900 TO 1910
0 TRLR
`
	path := filepath.Join(t.TempDir(), "neg.ged")
	if err := os.WriteFile(path, []byte(ged), 0o600); err != nil {
		t.Fatal(err)
	}

	out := runMain(t, path)

	if !strings.Contains(out, "@I1@: John /Doe/ - Born: 1 JAN 1920\n") {
		t.Errorf("birth line should show the real BIRT, got:\n%s", out)
	}
	if !strings.Contains(out, "  Events:\n    BIRT: 1 JAN 1920\n  Did not happen (negative assertions):\n    NO BIRT: FROM 1800 TO 1810\n    NO DEAT: FROM 1900 TO 1910\n") {
		t.Errorf("events should list only the BIRT, with the NO assertions labelled, got:\n%s", out)
	}
}

func runMain(t *testing.T, path string) string {
	t.Helper()
	oldArgs, oldStdout := os.Args, os.Stdout
	defer func() { os.Args, os.Stdout = oldArgs, oldStdout }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Args = []string{"query", path}
	os.Stdout = w

	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	main()
	_ = w.Close()
	return <-done
}
