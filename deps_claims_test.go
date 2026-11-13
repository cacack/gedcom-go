package gedcomgo

// The module's dependencies are stated once, in FEATURES.md's API Design
// section; every other document links to it. Two things drift: that statement,
// when go.mod gains or loses a requirement, and the older "standard library
// only" wording, which survived in CONSTITUTION.md and USAGE.md long after
// go.mod required golang.org/x/text (#729). This checks both.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
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

// docSkip are directory prefixes whose tracked documents are not current
// documentation of this library: history and fixtures.
var docSkip = []string{"docs/archive/", "testdata/"}

func TestNoDocClaimsZeroDependencies(t *testing.T) {
	if len(goModRequirements(t)) == 0 {
		t.Skip("go.mod requires nothing, so a zero-dependency claim would be true")
	}
	for _, claim := range zeroDependencyClaims(t, ".") {
		t.Errorf("%s, but go.mod has requirements; link to FEATURES.md#api-design instead", claim)
	}
}

// TestZeroDependencyClaimsReadsTrackedDocsOnly guards #832: local notes that
// git does not track, such as a gitignored directory, are not documentation
// of this library and must not fail the check.
func TestZeroDependencyClaimsReadsTrackedDocsOnly(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(path, content string) {
		t.Helper()
		full := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "-q")
	write(".gitignore", ".notes/\n")
	write("TRACKED.md", "Has zero dependencies.\n")
	write("untracked.md", "Has zero dependencies.\n")
	write(".notes/research.md", "Has zero dependencies.\n")
	git("add", ".gitignore", "TRACKED.md")

	got := zeroDependencyClaims(t, dir)
	want := []string{`TRACKED.md:1 says "zero dependenc"`}
	if !slices.Equal(got, want) {
		t.Errorf("zeroDependencyClaims = %q, want %q", got, want)
	}
}

// zeroDependencyClaims returns each zero-dependency claim in the Markdown
// documents git tracks under dir, as "path:line says \"match\"". Untracked and
// gitignored files are local notes, not documentation, so they are not read.
func zeroDependencyClaims(t *testing.T, dir string) []string {
	t.Helper()

	var claims []string
	for _, path := range trackedDocs(t, dir) {
		if path == "CHANGELOG.md" || zeroDependencyExempt[path] != "" ||
			slices.ContainsFunc(docSkip, func(prefix string) bool { return strings.HasPrefix(path, prefix) }) {
			continue
		}
		for i, line := range strings.Split(readFile(t, filepath.Join(dir, filepath.FromSlash(path))), "\n") {
			if m := zeroDependencyClaim.FindString(line); m != "" {
				claims = append(claims, fmt.Sprintf("%s:%d says %q", path, i+1, m))
			}
		}
	}
	return claims
}

// trackedDocs returns the Markdown files git tracks under dir, as slash paths
// relative to it. Outside a git checkout, such as the module cache, there is
// no record of what is tracked, so the test is skipped.
func trackedDocs(t *testing.T, dir string) []string {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, so tracked documents cannot be listed")
	}
	if err := exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree").Run(); err != nil {
		t.Skipf("%s is not a git checkout, so tracked documents cannot be listed", dir)
	}
	out, err := exec.Command("git", "-C", dir, "ls-files", "-z", "--", "*.md").Output()
	if err != nil {
		t.Fatalf("git ls-files: %v", err)
	}
	var paths []string
	for _, path := range strings.Split(string(out), "\x00") {
		if path != "" {
			paths = append(paths, path)
		}
	}
	return paths
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
