package decoder

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Public conformance suites (issue #718): every file is decoded in strict and
// lenient mode and must decode without panic and without an error. A file that
// does not is named below with the reason, so "which official files do we
// fail?" has a checked answer.
//
// The lists are self-cleaning, in the style of byte_roundtrip_test.go: a file
// that starts passing fails the test asking for its entry to be removed, and a
// file that starts failing without an entry fails as a regression.

// conformanceSuiteDirs are the vendored suites, one directory of .ged files
// each: gedcom7code/test-files, the gedcom.io 7.0 test files held so far, and
// oxidged's hand-written every-structure files (#763).
// See testdata/README.md for provenance.
var conformanceSuiteDirs = []string{
	"../testdata/gedcom7code/5",
	"../testdata/gedcom7code/7",
	"../testdata/gedcom-7.0/familysearch-examples",
	"../testdata/oxidged",
}

// strictKnownFailures lists files whose strict decode returns an error.
var strictKnownFailures = map[string]string{}

// lenientKnownFailures lists files whose lenient decode returns an error or
// an error-severity diagnostic.
var lenientKnownFailures = map[string]string{}

func conformanceSuiteFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	for _, dir := range conformanceSuiteDirs {
		matches, err := filepath.Glob(filepath.Join(dir, "*.ged"))
		if err != nil {
			t.Fatalf("glob %s: %v", dir, err)
		}
		if len(matches) == 0 {
			t.Fatalf("no .ged files in %s; the suite would prove nothing", dir)
		}
		files = append(files, matches...)
	}
	sort.Strings(files)
	return files
}

// decodeOutcome decodes data in the given mode and describes the first
// failure, or returns "" when the decode is clean. A panic is a failure like
// any other, so it is reported rather than allowed to abort the whole run.
func decodeOutcome(data []byte, strict bool) (failure string) {
	defer func() {
		if r := recover(); r != nil {
			failure = fmt.Sprintf("panic: %v", r)
		}
	}()

	opts := DefaultOptions()
	opts.StrictMode = strict
	res, err := DecodeWithDiagnostics(bytes.NewReader(data), opts)
	switch {
	case err != nil:
		return err.Error()
	case res == nil || res.Document == nil:
		return "nil document"
	}
	for _, d := range res.Diagnostics {
		if d.Severity == SeverityError {
			return d.String()
		}
	}
	return ""
}

func TestConformanceSuitesDecode(t *testing.T) {
	for _, path := range conformanceSuiteFiles(t) {
		key := strings.TrimPrefix(filepath.ToSlash(path), "../")
		t.Run(key, func(t *testing.T) {
			data, err := os.ReadFile(path) // #nosec G304 -- path comes from globbing the repo's own testdata
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			for _, mode := range []struct {
				name   string
				strict bool
				known  map[string]string
			}{
				{"strict", true, strictKnownFailures},
				{"lenient", false, lenientKnownFailures},
			} {
				t.Run(mode.name, func(t *testing.T) {
					failure := decodeOutcome(data, mode.strict)
					reason, known := mode.known[key]
					switch {
					case failure == "" && known:
						t.Errorf("now decodes cleanly; remove %q from %sKnownFailures (was: %s)", key, mode.name, reason)
					case failure != "" && !known:
						t.Errorf("decode failed and the file is not listed in %sKnownFailures: %s", mode.name, failure)
					}
				})
			}
		})
	}
}
