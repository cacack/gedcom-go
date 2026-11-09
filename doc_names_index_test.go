package gedcomgo

// Quick starts used to read a person's name as x.Names[0].Full, which panics
// on a nameless individual -- and real exports have them (#714). Copied into
// an app, that one line breaks ADR 0007's never-panic promise on the first
// upload that holds one. Individual.Name() and FullName() are the nil-safe
// spelling; this keeps every Markdown document and package overview on them.

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// namesIndex matches indexing Names[0] off a value (ind.Names[0], f().Names[0]),
// not a mention of the field path in prose or a diagnostic ("Records[4].Names[0]").
var namesIndex = regexp.MustCompile(`[A-Za-z0-9_)]\.Names\[0\]`)

func TestDocsDoNotIndexNames(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if path != "." && (strings.HasPrefix(name, ".") || name == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		// CHANGELOG.md is generated from commit messages.
		isDoc := strings.HasSuffix(name, ".md") && path != "CHANGELOG.md"
		isOverview := name == "doc.go" || path == "gedcom_api.go"
		if !isDoc && !isOverview {
			return nil
		}
		for i, line := range strings.Split(readFile(t, path), "\n") {
			if namesIndex.MatchString(line) {
				t.Errorf("%s:%d indexes Names[0], which panics on a nameless individual; "+
					"use Name() or FullName(): %s", path, i+1, strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
