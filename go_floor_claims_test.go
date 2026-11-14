package gedcomgo

// The Go version this module needs is stated once, in go.mod's `go` line, and
// repeated in prose, in CI and in a Dependabot rule. Each copy drifted (#746):
// the `go` line was raised to a patch release, which raised every consumer's
// floor with it; CI named a Go release in eleven places and never reached the
// current one; and the rule holding golang.org/x/text back had no end. This
// checks the copies against go.mod.

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// goFloorDocs are the documents that tell a reader which Go version they need.
var goFloorDocs = []string{"README.md", "USAGE.md", "CONTRIBUTING.md", "CLAUDE.md", "examples/README.md"}

// goFloorClaim matches the wordings those documents use: "Go 1.25 or later"
// and "Go 1.25+".
var goFloorClaim = regexp.MustCompile(`Go (\d+\.\d+)(?: or later|\+)`)

// goFloor returns the minor version of go.mod's `go` line, such as 25 for
// "go 1.25.0", and the patch.
func goFloor(t *testing.T) (minor, patch int) {
	t.Helper()

	m := regexp.MustCompile(`(?m)^go 1\.(\d+)\.(\d+)$`).FindStringSubmatch(readFile(t, "go.mod"))
	if m == nil {
		t.Fatal("go.mod has no `go 1.N.P` line")
	}
	minor, _ = strconv.Atoi(m[1])
	patch, _ = strconv.Atoi(m[2])
	return minor, patch
}

// A library's `go` line is the floor of every module that requires it: `go mod
// tidy` in a consumer copies it up. A patch there patches nobody's standard
// library; the `toolchain` line is where this module pins its own.
func TestGoModFloorIsNotAPatchRelease(t *testing.T) {
	if minor, patch := goFloor(t); patch != 0 {
		t.Errorf("go.mod says go 1.%d.%d, which makes every consumer declare that patch release; "+
			"use go 1.%d.0 and put the patch on the toolchain line", minor, patch, minor)
	}
	if !regexp.MustCompile(`(?m)^toolchain go1\.\d+\.\d+$`).MatchString(readFile(t, "go.mod")) {
		t.Error("go.mod has no toolchain line; CI's 'min' job and the security scanners read the patched release from it")
	}
}

func TestDocsStateTheGoModFloor(t *testing.T) {
	minor, _ := goFloor(t)
	want := "1." + strconv.Itoa(minor)

	for _, path := range goFloorDocs {
		claims := goFloorClaim.FindAllStringSubmatch(readFile(t, path), -1)
		if len(claims) == 0 {
			t.Errorf("%s no longer states the Go version in a wording this test knows; update goFloorClaim or goFloorDocs", path)
		}
		for _, claim := range claims {
			if claim[1] != want {
				t.Errorf("%s says %q, but go.mod's floor is Go %s", path, claim[0], want)
			}
		}
	}
}

// The workflows take the floor from go.mod and the two supported releases from
// setup-go's aliases, so a written-out Go version is a copy that will go stale.
func TestWorkflowsNameNoGoVersion(t *testing.T) {
	ci := readFile(t, ".github/workflows/ci.yml")
	if !strings.Contains(ci, "go: ['min', 'oldstable', 'stable']") {
		t.Error("ci.yml's test matrix is no longer the floor plus the two most recent Go releases " +
			"(go: ['min', 'oldstable', 'stable'])")
	}

	pinned := regexp.MustCompile(`(?m)^\s*go-version: .*\d.*$`)
	for _, path := range []string{".github/workflows/ci.yml", ".github/workflows/nightly.yml"} {
		for _, line := range pinned.FindAllString(readFile(t, path), -1) {
			t.Errorf("%s pins a Go version (%s); use stable, oldstable or go-version-file",
				path, strings.TrimSpace(line))
		}
	}
}

// The Dependabot rule that holds golang.org/x/text below v0.42.0 exists only
// because that release needs Go 1.26. Raising the floor to 1.26 ends its
// reason, and this fails until the rule is removed with it.
func TestDependabotIgnoreIsLiftedWithTheGoFloor(t *testing.T) {
	const liftedAt = 26

	dependabot := readFile(t, ".github/dependabot.yml")
	held := strings.Contains(dependabot, `dependency-name: "golang.org/x/text"`)
	minor, _ := goFloor(t)

	switch {
	case held && minor >= liftedAt:
		t.Errorf("go.mod's floor is Go 1.%d, so nothing holds golang.org/x/text below v0.42.0 any more; "+
			"remove its ignore rule from .github/dependabot.yml and this test", minor)
	case held && !strings.Contains(dependabot, "go.mod's `go` line to 1."+strconv.Itoa(liftedAt)):
		t.Errorf("the golang.org/x/text ignore rule in .github/dependabot.yml no longer names Go 1.%d as the floor that lifts it", liftedAt)
	}
}
