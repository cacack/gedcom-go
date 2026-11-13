package validator_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/cacack/gedcom-go/v3/converter"
	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/validator"
)

// checking returns a validator with the opt-in ENTITY_TAGS_DIVERGED check on.
func checking() *validator.Validator {
	return validator.NewWithOptions(&validator.ValidateOptions{CheckEntityTags: true})
}

// Issue #725: converting to 7.0 sets Header.Encoding and removes CHAR, which
// is not a header divergence; 7.0 has no CHAR to write.
func TestEntityTagsDivergedConverted70Header(t *testing.T) {
	f, err := os.Open("../testdata/gedcom-5.5/royal92.ged")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	doc, err := decoder.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	out, _, err := converter.Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatal(err)
	}
	for _, issue := range checking().ValidateAll(out) {
		if issue.Code == validator.CodeEntityTagsDiverged {
			t.Errorf("converted 7.0 document reported %v", issue)
		}
	}
}

// TestEntityTagsDivergedNotOnConverterOutput asserts that converter output,
// whose typed entities are re-derived from the converted Tags (#772), never
// reports ENTITY_TAGS_DIVERGED: for every fixture and every target version,
// including the 7.0 -> 7.0 copy. Before #772 this reported 228 issues.
func TestEntityTagsDivergedNotOnConverterOutput(t *testing.T) {
	t.Parallel() // #846: overlaps the other corpus-wide walks
	files, err := filepath.Glob("../testdata/*/*.ged")
	if err != nil || len(files) == 0 {
		t.Fatalf("no fixtures: %v", err)
	}
	// Each fixture is a parallel subtest (#846): run serially, the corpus
	// nears the 10-minute -race timeout on slow CI runners. The group waits
	// for every subtest, so checked is final once it returns.
	var checked atomic.Int64
	t.Run("fixtures", func(t *testing.T) {
		for _, path := range files {
			if filepath.Base(path) == "longsword.ged" {
				continue // the scale fixture, as other corpus walks skip it
			}
			t.Run(strings.TrimPrefix(filepath.ToSlash(path), "../testdata/"), func(t *testing.T) {
				t.Parallel()
				data, err := os.ReadFile(path) // #nosec G304 -- a fixture path from the repo's own testdata
				if err != nil {
					t.Fatal(err)
				}
				v := checking()
				for _, target := range []gedcom.Version{gedcom.Version55, gedcom.Version551, gedcom.Version70} {
					doc, err := decoder.Decode(bytes.NewReader(data))
					if err != nil || doc.Header == nil {
						continue // a malformed fixture, not a conversion input
					}
					if doc.Header.Version == target && target != gedcom.Version70 {
						continue // returned as is; only 7.0 -> 7.0 rewrites a copy
					}
					out, _, err := converter.Convert(doc, target)
					if err != nil {
						continue
					}
					checked.Add(1)
					for _, issue := range v.ValidateAll(out) {
						if issue.Code == validator.CodeEntityTagsDiverged {
							t.Errorf("-> %s: %s", target, issue.Message)
						}
					}
				}
			})
		}
	})
	if checked.Load() == 0 {
		t.Fatal("no conversion was checked")
	}
}
