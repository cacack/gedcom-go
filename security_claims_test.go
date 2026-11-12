package gedcomgo

// SECURITY.md states which scanners run and when, and which versions get
// security fixes. Both drifted from the workflows unnoticed (#745): it listed
// every scanner as running "on every pull request and nightly" when most run
// on one or the other, omitted osv-scanner, and called the module pre-1.0 with
// one supported line while release-2.x was wired into CI and release-please.
// This checks both against .github/workflows/.
//
// Limits: a scanner's cadence is its workflow file's triggers, not its job's
// `if:` gate, and the prose after the table (push branches, fuzz targets) is
// not checked.

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// securityScanner is one scanner SECURITY.md lists, with the text that shows
// a workflow runs it.
type securityScanner struct {
	name   string // as the SECURITY.md table names it
	marker string // an action or image a workflow uses to run it
}

var securityScanners = []securityScanner{
	{"CodeQL", "github/codeql-action/analyze"},
	{"govulncheck", "golang/govulncheck-action"},
	{"gosec", "securego/gosec"},
	{"Trivy", "aquasecurity/trivy-action"},
	{"osv-scanner", "google/osv-scanner-action"},
	{"Semgrep", "returntocorp/semgrep"},
	{"gitleaks", "gitleaks/gitleaks-action"},
	{"Dependency Review", "actions/dependency-review-action"},
	{"OSSF Scorecard", "ossf/scorecard-action"},
}

// sarifCategoryScanner names the scanner behind each SARIF upload category, so
// a scanner added to a workflow without being added here fails the test
// instead of going unlisted.
var sarifCategoryScanner = map[string]string{
	"/language:go": "CodeQL",
	"govulncheck":  "govulncheck",
	"gosec":        "gosec",
	"trivy":        "Trivy",
	"osv-scanner":  "osv-scanner",
	"semgrep":      "Semgrep",
	"scorecard":    "OSSF Scorecard",
}

// scanCadence is when a scanner runs: the SECURITY.md table's three columns.
type scanCadence struct{ pullRequest, nightly, weekly bool }

var (
	// pullRequestTrigger assumes block-style `on:` keys indented two spaces;
	// a mismatch after reformatting a workflow is this regex's, not SECURITY.md's.
	pullRequestTrigger = regexp.MustCompile(`(?m)^ {2}pull_request:`)
	cronSchedule       = regexp.MustCompile(`cron: '([^']+)'`)
	sarifCategory      = regexp.MustCompile(`(?m)^\s+category: "?([^"\s]+)"?\s*$`)
	scannerRow         = regexp.MustCompile(
		`(?m)^\| \[([^\]]+)\][^|]*\| *(yes)? *\| *(yes)? *\| *(yes)? *\|$`)
)

func TestSecurityScannerListMatchesWorkflows(t *testing.T) {
	workflows := workflowFiles(t)

	want := map[string]scanCadence{}
	for _, wf := range workflows {
		cadence := workflowCadence(wf)
		for _, s := range securityScanners {
			if !strings.Contains(wf, s.marker) {
				continue
			}
			c := want[s.name]
			c.pullRequest = c.pullRequest || cadence.pullRequest
			c.nightly = c.nightly || cadence.nightly
			c.weekly = c.weekly || cadence.weekly
			want[s.name] = c
		}
		for _, m := range sarifCategory.FindAllStringSubmatch(wf, -1) {
			if _, ok := sarifCategoryScanner[m[1]]; !ok {
				t.Errorf("a workflow uploads SARIF category %q, which this test does not know; "+
					"add the scanner to SECURITY.md and to securityScanners", m[1])
			}
		}
	}

	security := readFile(t, "SECURITY.md")
	_, tooling, found := strings.Cut(security, "## Security Tooling")
	if !found {
		t.Fatal("SECURITY.md has no Security Tooling section")
	}
	got := map[string]scanCadence{}
	for _, row := range scannerRow.FindAllStringSubmatch(tooling, -1) {
		got[row[1]] = scanCadence{row[2] != "", row[3] != "", row[4] != ""}
	}

	for _, name := range sortedKeys(want) {
		g, ok := got[name]
		switch {
		case !ok:
			t.Errorf("SECURITY.md does not list %s, which a workflow runs", name)
		case g != want[name]:
			t.Errorf("SECURITY.md says %s runs %+v; the workflows run it %+v", name, g, want[name])
		}
	}
	for _, name := range sortedKeys(got) {
		if _, ok := want[name]; !ok {
			t.Errorf("SECURITY.md lists %s, which no workflow runs", name)
		}
	}
}

func TestSecuritySupportedVersionsMatchReleaseLines(t *testing.T) {
	security := readFile(t, "SECURITY.md")
	if strings.Contains(security, "pre-1.0") {
		t.Error("SECURITY.md calls the module pre-1.0")
	}

	major := regexp.MustCompile(`(?m)^module .*/v(\d+)$`).FindStringSubmatch(readFile(t, "go.mod"))
	if major == nil {
		t.Fatal("go.mod's module path has no major version suffix")
	}
	lines := []string{major[1]}
	for _, m := range regexp.MustCompile(`release-(\d+)\.x`).FindAllStringSubmatch(
		readFile(t, ".github/workflows/release-please.yml"), -1) {
		lines = append(lines, m[1])
	}
	for _, line := range lines {
		row := regexp.MustCompile(`(?m)^\| ` + line + `\.[0-9x]+[^|]*\| :white_check_mark:`)
		if !row.MatchString(security) {
			t.Errorf("SECURITY.md has no supported row for the v%s release line", line)
		}
	}
}

// workflowFiles returns the contents of every workflow file.
func workflowFiles(t *testing.T) []string {
	t.Helper()
	paths, err := filepath.Glob(".github/workflows/*.yml")
	if err != nil || len(paths) == 0 {
		t.Fatalf("no workflows found: %v", err)
	}
	files := make([]string, 0, len(paths))
	for _, p := range paths {
		files = append(files, readFile(t, p))
	}
	return files
}

// workflowCadence reads when a workflow runs from its triggers: on pull
// requests, on a daily schedule, or on a schedule restricted to some weekdays.
func workflowCadence(wf string) scanCadence {
	c := scanCadence{pullRequest: pullRequestTrigger.MatchString(wf)}
	for _, m := range cronSchedule.FindAllStringSubmatch(wf, -1) {
		fields := strings.Fields(m[1])
		if len(fields) != 5 {
			continue
		}
		if fields[2] == "*" && fields[3] == "*" && fields[4] == "*" {
			c.nightly = true
		} else {
			c.weekly = true
		}
	}
	return c
}

func sortedKeys(m map[string]scanCadence) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
