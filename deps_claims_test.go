package gedcomgo

// The module's dependencies are stated once, in FEATURES.md's API Design
// section; every other document links to it. Two things drift: that statement,
// when go.mod gains or loses a requirement, and the older "standard library
// only" wording, which survived in CONSTITUTION.md and USAGE.md long after
// go.mod required golang.org/x/text (#729). This checks both.

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// zeroDependencyClaim matches the wordings that say the library needs nothing
// beyond the standard library.
var zeroDependencyClaim = regexp.MustCompile(
	`(?i)standard library only|zero[- ]dependenc|only the (go )?standard library`)

// zeroDependencyExempt are documents that may use the wording because they are
// not about this library.
var zeroDependencyExempt = map[string]string{
	"scripts/spec55/README.md": "describes a Python helper script, not the Go module",
}

// docWalkSkip are directories that hold no current documentation of this
// library: history, fixtures, downloaded corpora, and other agents' worktrees.
var docWalkSkip = map[string]bool{
	".git": true, ".corpora": true, "docs/archive": true, "testdata": true, ".claude/worktrees": true,
}

func TestNoDocClaimsZeroDependencies(t *testing.T) {
	if len(goModRequirements(t)) == 0 {
		t.Skip("go.mod requires nothing, so a zero-dependency claim would be true")
	}
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		path = filepath.ToSlash(path)
		if d.IsDir() {
			if docWalkSkip[path] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".md") || path == "CHANGELOG.md" || zeroDependencyExempt[path] != "" {
			return nil
		}
		for i, line := range strings.Split(readFile(t, path), "\n") {
			if m := zeroDependencyClaim.FindString(line); m != "" {
				t.Errorf("%s:%d says %q, but go.mod has requirements; "+
					"link to FEATURES.md#api-design instead", path, i+1, m)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestFeaturesNamesEveryDependency(t *testing.T) {
	features := readFile(t, "FEATURES.md")
	statement := regexp.MustCompile(`(?m)^- (?:Single dependency|Dependencies): .*$`).FindString(features)
	if statement == "" {
		t.Fatal("FEATURES.md no longer states the module's dependencies under API Design; " +
			"if the wording changed, update this test with it")
	}
	for _, module := range goModRequirements(t) {
		if !strings.Contains(statement, "`"+module+"`") {
			t.Errorf("go.mod requires %s, which FEATURES.md's dependency statement does not name: %q",
				module, statement)
		}
	}
}

// goModRequirements returns the modules go.mod requires directly, in both the
// single-line and the block form, leaving out indirect ones.
func goModRequirements(t *testing.T) []string {
	t.Helper()

	var modules []string
	inBlock := false
	for _, line := range strings.Split(readFile(t, "go.mod"), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "require (":
			inBlock = true
			continue
		case inBlock && line == ")":
			inBlock = false
			continue
		case strings.HasPrefix(line, "require "):
			line = strings.TrimPrefix(line, "require ")
		case !inBlock:
			continue
		}
		if fields := strings.Fields(line); len(fields) >= 2 && !strings.Contains(line, "// indirect") {
			modules = append(modules, fields[0])
		}
	}
	return modules
}
