package validator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/decoder"
)

// TestNoIssueCarriesLineNumberDetailKey walks the whole testdata corpus and
// asserts that no Issue smuggles a source line through Details. The key was
// removed in v3 (#498) once Issue.LineNumber existed to hold it.
//
// Details["position"] is deliberately NOT included here: it is a byte offset
// within a field's value, not a source line, so it survives the removal. See
// the note on checkControlChars.
func TestNoIssueCarriesLineNumberDetailKey(t *testing.T) {
	// Validating the whole corpus at Strict takes ~15s, which is a poor trade
	// on every local iteration. CI runs the full suite.
	if testing.Short() {
		t.Skip("skipping full-corpus scan in -short mode")
	}

	var files []string
	err := filepath.Walk("../testdata", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".ged") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking testdata: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no .ged fixtures found; the test would prove nothing")
	}

	v := NewWithOptions(&ValidateOptions{
		TagRegistry:        NewTagRegistry(),
		ValidateCustomTags: true,
		Strictness:         StrictnessStrict,
	})

	var checked int
	for _, path := range files {
		f, err := os.Open(path) // #nosec G304 -- path comes from walking the repo's own testdata
		if err != nil {
			t.Fatalf("open %s: %v", path, err)
		}
		doc, err := decoder.Decode(f)
		_ = f.Close()
		if err != nil {
			continue // malformed fixtures are expected; they exercise the decoder, not this
		}

		for _, issue := range v.ValidateAll(doc) {
			checked++
			if _, ok := issue.Details["line_number"]; ok {
				t.Errorf("%s: %s still carries Details[line_number]", path, issue.Code)
			}
		}
	}

	if checked == 0 {
		t.Fatal("the corpus produced no issues at all; the assertion would be vacuous")
	}
	t.Logf("checked %d issues across %d fixtures", checked, len(files))
}
