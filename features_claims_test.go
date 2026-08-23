package gedcomgo

// FEATURES.md's Multi-Version Support section states measured numbers rather
// than adjectives: how many GEDCOM 7.0 structures reach the typed model, and
// how many corpus fixtures survive a round-trip. Numbers in prose rot silently
// -- they are the one thing a reader trusts and the one thing nothing checks.
//
// This checks them against the two places they are actually derived: the
// generated coverage report, and the self-cleaning lists in
// byte_roundtrip_test.go. Fixing any of the issues those lists track will fail
// this test until the prose is updated, which is the point.

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The generated inventories, which are the authority for every coverage number
// stated anywhere else.
const (
	coverageReportPath   = "docs/reference/gedcom-7-coverage.md"
	coverage55ReportPath = "docs/reference/gedcom-5.5-coverage.md"
)

// featuresClaim is one number FEATURES.md states, with the pattern that finds
// it and the value it has to equal.
type featuresClaim struct {
	what    string
	pattern *regexp.Regexp
	want    int
}

func TestFeaturesNumbersMatchTheirSources(t *testing.T) {
	features := readFile(t, "FEATURES.md")
	typed, total := coverageReportTotals(t)
	typed55, total55, typed551, total551 := coverage55ReportTotals(t)

	fixtures := len(gedcomFixtures(t))
	decodable := fixtures - len(undecodable)

	claims := []featuresClaim{
		{"typed 7.0 structures", regexp.MustCompile(
			`\[(\d[\d,]*) of [\d,]+ structures\]\(docs/reference/gedcom-7-coverage\.md\)`), typed},
		{"total 7.0 structures", regexp.MustCompile(
			`\[[\d,]+ of (\d[\d,]*) structures\]\(docs/reference/gedcom-7-coverage\.md\)`), total},
		{"typed 5.5 structures", regexp.MustCompile(
			`\| GEDCOM 5\.5 \|[^|]*\| \[(\d[\d,]*) of `), typed55},
		{"total 5.5 structures", regexp.MustCompile(
			`\| GEDCOM 5\.5 \|[^|]*\| \[[\d,]+ of (\d[\d,]*) structures\]`), total55},
		{"typed 5.5.1 structures", regexp.MustCompile(
			`\| GEDCOM 5\.5\.1 \|[^|]*\| \[(\d[\d,]*) of `), typed551},
		{"total 5.5.1 structures", regexp.MustCompile(
			`\| GEDCOM 5\.5\.1 \|[^|]*\| \[[\d,]+ of (\d[\d,]*) structures\]`), total551},
		{"typed 7.0 structures, restated", regexp.MustCompile(
			`(\d[\d,]*) of 7\.0's structures reach the typed\s+model`), typed},
		{"typed 5.5 structures, restated", regexp.MustCompile(
			`reach the typed\s+model, (\d[\d,]*) of\s+5\.5's`), typed55},
		{"typed 5.5.1 structures, restated", regexp.MustCompile(
			`of\s+5\.5's, and (\d[\d,]*) of 5\.5\.1's`), typed551},
		{"corpus fixtures", regexp.MustCompile(`Of (\d[\d,]*) corpus\s+fixtures`), fixtures},
		{"undecodable fixtures", regexp.MustCompile(`fixtures, (\d[\d,]*) do not survive`), len(undecodable)},
		{"decodable fixtures", regexp.MustCompile(`of the (\d[\d,]*) that do`), decodable},
		{"headers that round-trip", regexp.MustCompile(`(\d[\d,]*)\s+reproduce their header`), decodable - len(headerKnownBad)},
		{"bodies that round-trip", regexp.MustCompile(`(\d[\d,]*) reproduce their record body`), decodable - len(bodyKnownBad)},
	}

	for _, claim := range claims {
		match := claim.pattern.FindStringSubmatch(features)
		if match == nil {
			t.Errorf("FEATURES.md no longer states the %s (looked for %s); "+
				"if the wording changed, update this test with it", claim.what, claim.pattern)
			continue
		}
		got, err := strconv.Atoi(strings.ReplaceAll(match[1], ",", ""))
		if err != nil {
			t.Errorf("%s: %q is not a number: %v", claim.what, match[1], err)
			continue
		}
		if got != claim.want {
			t.Errorf("FEATURES.md says %d %s; the source says %d. Update FEATURES.md.",
				got, claim.what, claim.want)
		}
	}

	// The percentages are derived from the counts, so they cannot be checked
	// against a source -- only against arithmetic.
	for _, share := range []struct{ typed, total int }{
		{typed, total}, {typed55, total55}, {typed551, total551},
	} {
		want := fmt.Sprintf("(%.1f%%)", 100*float64(share.typed)/float64(share.total))
		if !strings.Contains(features, want) {
			t.Errorf("FEATURES.md does not state the typed share as %s (%d of %d)",
				want, share.typed, share.total)
		}
	}
}

// coverage55ReportTotals reads both versions' counts out of the combined 5.5 and
// 5.5.1 report, whose summary table carries a count and a share per version.
func coverage55ReportTotals(t *testing.T) (typed55, total55, typed551, total551 int) {
	t.Helper()

	report := readFile(t, coverage55ReportPath)

	// The counts are read positionally, so the header has to name the versions
	// in the order the columns are in. A reordering of spec55Editions would
	// otherwise swap the two versions' numbers and check both claims against
	// each other's source.
	if !strings.Contains(report, "| Status | 5.5 | Share | 5.5.1 | Share | Meaning |") {
		t.Fatalf("%s: the summary header is not the 5.5-then-5.5.1 column order this "+
			"reads counts by; update both together", coverage55ReportPath)
	}

	// Rows are "| status | 5.5 count | share | 5.5.1 count | share | meaning |".
	rows := regexp.MustCompile(
		`(?m)^\| (typed|partial|raw \([a-z]+\)) \| (\d+) \| [\d.]+% \| (\d+) \|`).
		FindAllStringSubmatch(report, -1)
	if len(rows) == 0 {
		t.Fatalf("%s has no summary table; regenerate it with `make spec-coverage`",
			coverage55ReportPath)
	}
	for _, row := range rows {
		counts := make([]int, 2)
		for i, field := range row[2:4] {
			n, err := strconv.Atoi(field)
			if err != nil {
				t.Fatalf("%s: %q is not a count: %v", coverage55ReportPath, field, err)
			}
			counts[i] = n
		}
		total55 += counts[0]
		total551 += counts[1]
		if row[1] == "typed" {
			typed55, typed551 = counts[0], counts[1]
		}
	}
	return typed55, total55, typed551, total551
}

// coverageReportTotals reads the typed count and the total from the generated
// report's summary table, so FEATURES.md and the report cannot disagree.
func coverageReportTotals(t *testing.T) (typed, total int) {
	t.Helper()

	report := readFile(t, coverageReportPath)

	// Rows are "| status | count | share | meaning |"; the total is their sum.
	rows := regexp.MustCompile(`(?m)^\| (typed|partial|raw \([a-z]+\)) \| (\d+) \|`).FindAllStringSubmatch(report, -1)
	if len(rows) == 0 {
		t.Fatalf("%s has no summary table; regenerate it with `make spec-coverage`", coverageReportPath)
	}
	for _, row := range rows {
		n, err := strconv.Atoi(row[2])
		if err != nil {
			t.Fatalf("%s: %q is not a count: %v", coverageReportPath, row[2], err)
		}
		total += n
		if row[1] == "typed" {
			typed = n
		}
	}
	return typed, total
}

// readFile reads a repository file, failing the test if it cannot.
func readFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
